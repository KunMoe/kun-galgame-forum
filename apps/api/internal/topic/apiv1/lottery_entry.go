package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func (l *Lotteries) enterLottery(ctx context.Context, in *lotteryInput) (*lotteryOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	if _, err := l.lots.FindEntry(lottery.ID, user.ID); err == nil {
		return l.lotteryOut(ctx, lottery.ID, user, false)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, problem.Internal(err)
	}
	reason, err := l.svc.EntryBlock(ctx, lottery, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	switch reason {
	case "":
	case service.EntryBlockNotOpen, service.EntryBlockPastClosesAt:
		return nil, lotteryClosed()
	default:
		return nil, lotteryIneligible(reason)
	}
	err = l.db().Transaction(func(tx *gorm.DB) error {
		inserted, err := l.lots.InsertEntryIfAbsent(tx, lottery.ID, user.ID)
		if err != nil || !inserted {
			return err
		}
		return l.lots.SyncEntryCount(tx, lottery.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return l.lotteryOut(ctx, lottery.ID, user, false)
}

func (l *Lotteries) withdrawLottery(ctx context.Context, in *lotteryInput) (*lotteryOutput, error) {
	lottery, _, user, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	entry, err := l.lots.FindEntry(lottery.ID, user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return l.lotteryOut(ctx, lottery.ID, user, false)
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	pastClose := lottery.Deadline != nil && time.Now().After(*lottery.Deadline)
	if entry.PrizeID > 0 || lottery.Status != model.LotteryStatusOpen || pastClose {
		return nil, lotteryClosed()
	}
	err = l.db().Transaction(func(tx *gorm.DB) error {
		deleted, err := l.lots.DeleteUnwonEntry(tx, lottery.ID, user.ID)
		if err != nil || !deleted {
			return err
		}
		return l.lots.SyncEntryCount(tx, lottery.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return l.lotteryOut(ctx, lottery.ID, user, false)
}

type lotteryWinnerInput struct {
	LotteryID string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
	WinnerID  string `path:"winner_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Winner id, as in LotteryWinner.id."`
}

func (l *Lotteries) getLotteryWinner(ctx context.Context, in *lotteryWinnerInput) (*lotteryWinnerOutput, error) {
	lottery, _, _, prob := l.visibleLottery(ctx, in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	winnerID, ok := parsePositiveID(in.WinnerID)
	if !ok {
		return nil, notFound()
	}
	entry, err := l.lots.FindWinner(lottery.ID, winnerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	return l.winnerOut(ctx, entry)
}

func (l *Lotteries) winnerOut(ctx context.Context, entry *model.TopicLotteryEntry) (*lotteryWinnerOutput, error) {
	users, prob := l.reads.lookupUsers(ctx, []int{entry.UserID})
	if prob != nil {
		return nil, prob
	}
	u, ok := users[entry.UserID]
	if ok && !userclient.IsRenderable(u) {
		return nil, notFound()
	}
	if !ok {
		u = userclient.Placeholder(entry.UserID)
	}
	return &lotteryWinnerOutput{Body: mapWinner(l.reads.cdn, *entry, u)}, nil
}

// offlineMoves is the fulfillment state machine for a prize the author hands
// over off the site. Code and point prizes are moved by the system alone.
var offlineMoves = map[string][]string{
	model.LotteryFulfillPending: {model.LotteryFulfillShipped, model.LotteryFulfillReceived, model.LotteryFulfillForfeited},
	model.LotteryFulfillShipped: {model.LotteryFulfillPending, model.LotteryFulfillReceived, model.LotteryFulfillForfeited},
}

var winnerMayMoveTo = []string{model.LotteryFulfillReceived, model.LotteryFulfillForfeited}

func (l *Lotteries) updateLotteryWinner(ctx context.Context, in *updateLotteryWinnerInput) (*lotteryWinnerOutput, error) {
	if prob := l.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	lottery, prob := l.findLottery(in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	winnerID, ok := parsePositiveID(in.WinnerID)
	if !ok {
		return nil, notFound()
	}
	entry, err := l.lots.FindWinner(lottery.ID, winnerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}

	// The winner acts on their own prize whether or not they can still read the
	// topic, as with revealLotteryCode; everyone else goes through the topic.
	isWinner := entry.UserID == user.ID
	if !isWinner {
		visible, _, _, prob := l.visibleLottery(ctx, in.LotteryID)
		if prob != nil {
			return nil, prob
		}
		lottery = visible
	}
	manage := capsForLottery(lottery, user).Manage
	target := in.Body.Fulfillment
	if !manage && !(isWinner && slices.Contains(winnerMayMoveTo, target)) {
		return nil, permissionRequired()
	}
	if target != entry.Fulfillment {
		prize, err := l.lots.FindPrizeByID(entry.PrizeID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		if prize.Delivery != model.LotteryDeliveryManual {
			return nil, invalidStateTransition("This prize is delivered by the site; its fulfillment is not moved by hand.")
		}
		if !slices.Contains(offlineMoves[entry.Fulfillment], target) {
			return nil, invalidStateTransition("The winner is " + entry.Fulfillment + ", which is final.")
		}
		moved, err := l.lots.MoveFulfillment(l.db(), entry.ID, []string{entry.Fulfillment}, target, time.Now())
		if err != nil {
			return nil, problem.Internal(err)
		}
		if !moved {
			return nil, invalidStateTransition("The winner's fulfillment changed while this request was made.")
		}
		entry.Fulfillment = target
	}
	return l.winnerOut(ctx, entry)
}

// revealLotteryCode is the only place a code leaves the database in plain
// text. It is a POST so that no page load can fetch it: Nuxt inlines every
// payload fetched during SSR into the page's __NUXT__ blob. It takes no
// Idempotency-Key either, because the idempotency store would keep the
// response, code and all, in Redis for a day (K22).
func (l *Lotteries) revealLotteryCode(ctx context.Context, in *lotteryInput) (*lotteryCodeRevealOutput, error) {
	if prob := l.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	lottery, prob := l.findLottery(in.LotteryID)
	if prob != nil {
		return nil, prob
	}
	entry, err := l.lots.FindEntry(lottery.ID, user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	if entry.PrizeID == 0 || entry.CodeID == 0 {
		return nil, notFound()
	}
	if entry.Fulfillment == model.LotteryFulfillForfeited {
		return nil, problem.New(problem.CodeRedemptionCodeForfeited, "The code was given up or not revealed in time.")
	}
	code, err := l.lots.FindCodeByID(entry.CodeID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if code.ClaimedBy != user.ID {
		return nil, notFound()
	}
	plain, err := l.svc.OpenCode(code.Secret)
	if err != nil {
		slog.Error("lottery redemption code failed to open", "lottery_id", lottery.ID, "code_id", code.ID, "error", err)
		return nil, problem.Internal(err)
	}
	if entry.Fulfillment != model.LotteryFulfillReceived {
		moved, err := l.lots.MoveFulfillment(l.db(), entry.ID,
			[]string{model.LotteryFulfillPending, model.LotteryFulfillShipped}, model.LotteryFulfillReceived, time.Now())
		if err != nil {
			return nil, problem.Internal(err)
		}
		if !moved {
			return nil, problem.New(problem.CodeRedemptionCodeForfeited, "The code was forfeited while this request was made.")
		}
	}
	return &lotteryCodeRevealOutput{Body: LotteryCodeReveal{
		Object:         "lottery_code_reveal",
		LotteryID:      repr.DecimalID(strconv.Itoa(lottery.ID)),
		RedemptionCode: plain,
	}}, nil
}
