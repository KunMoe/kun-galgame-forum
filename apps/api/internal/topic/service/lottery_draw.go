package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const drawSweepBatch = 50

// NewDrawSeed returns the secret and the commitment published with it. Anyone
// can recompute the winners from the seed once it is revealed, so the author
// cannot pick a favourable ordering after seeing who entered. This defends
// against the topic author, not against the site operator, which is the honest
// boundary of a commit-reveal scheme run on the operator's own server.
func NewDrawSeed() (seed, seedHash string) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand failing is not a condition to paper over with time.Now().
		panic("抽奖随机数生成失败: " + err.Error())
	}
	seed = hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(seed))
	return seed, hex.EncodeToString(sum[:])
}

func rankKey(seed string, lotteryID, userID int) string {
	mac := hmac.New(sha256.New, []byte(seed))
	// errcheck ships a default exclusion for (hash.Hash).Write but not for
	// fmt.Fprintf to a hash, so writing this the Fprintf way fails CI.
	mac.Write(fmt.Appendf(nil, "%d:%d", lotteryID, userID))
	return hex.EncodeToString(mac.Sum(nil))
}

// pointWeight is the random share a winner draws from a 拼手气 pool. It comes
// out of the revealed seed rather than crypto/rand so the commit-reveal proof
// covers the amounts too, not only who won. Four bytes keep the later
// pool*weight multiplication far away from overflowing int64; the +1 keeps the
// weight sum non-zero so it can be divided by.
func pointWeight(seed string, prizeID, userID int) uint64 {
	mac := hmac.New(sha256.New, []byte(seed))
	mac.Write(fmt.Appendf(nil, "point:%d:%d", prizeID, userID))
	return uint64(binary.BigEndian.Uint32(mac.Sum(nil)[:4])) + 1
}

// pointPayouts turns one point prize into one amount per winner, in the same
// order the winners were ranked.
//
// fixed hands every winner the configured amount. split and random read it as a
// pool and hand out all of it, so a prize with fewer winners than slots pays
// each of them more rather than leaving the remainder unspent — that is the
// point of offering a pool at all.
func pointPayouts(seed string, prize topicModel.TopicLotteryPrize, winners []topicModel.TopicLotteryEntry) []int {
	n := len(winners)
	if n == 0 || prize.PointAmount <= 0 {
		return make([]int, n)
	}

	out := make([]int, n)
	switch prize.PointMode {
	case topicModel.LotteryPointSplit:
		each, remainder := prize.PointAmount/n, prize.PointAmount%n
		for i := range out {
			out[i] = each
			if i < remainder {
				out[i]++
			}
		}

	case topicModel.LotteryPointRandom:
		floor := 1
		if prize.PointAmount < n {
			floor = 0
		}
		rest := prize.PointAmount - floor*n

		weights := make([]uint64, n)
		var totalWeight uint64
		for i, w := range winners {
			weights[i] = pointWeight(seed, prize.ID, w.UserID)
			totalWeight += weights[i]
		}
		handed := 0
		for i := range out {
			share := int(uint64(rest) * weights[i] / totalWeight)
			out[i] = floor + share
			handed += share
		}
		// Integer division always leaves a few points behind; they go to the
		// best-ranked winners so the pool is spent exactly.
		for i := 0; handed < rest; i, handed = i+1, handed+1 {
			out[i%n]++
		}

	default:
		for i := range out {
			out[i] = prize.PointAmount
		}
	}
	return out
}

// stampPointPayouts fills in PointAwarded on every winner of a point prize. It
// groups by prize first because a pooled prize is split across exactly the
// people who won that prize, which the flattened slot list does not say.
func stampPointPayouts(
	seed string,
	slots []topicModel.TopicLotteryPrize,
	winners []topicModel.TopicLotteryEntry,
) {
	byPrize := map[int][]int{}
	for i := range winners {
		if slots[i].Delivery == topicModel.LotteryDeliveryPoint {
			byPrize[slots[i].ID] = append(byPrize[slots[i].ID], i)
		}
	}
	for _, indices := range byPrize {
		group := make([]topicModel.TopicLotteryEntry, len(indices))
		for k, i := range indices {
			group[k] = winners[i]
		}
		for k, amount := range pointPayouts(seed, slots[indices[0]], group) {
			winners[indices[k]].PointAwarded = amount
		}
	}
}

type LotteryDrawer struct {
	svc     *LotteryService
	running sync.Mutex
}

// How long a winner has to reveal a code prize before the sweep voids it.
const lotteryClaimGrace = 7 * 24 * time.Hour

func NewLotteryDrawer(svc *LotteryService) *LotteryDrawer {
	return &LotteryDrawer{svc: svc}
}

// Run is the every-minute sweep. topic_poll's notification_sent column sat dead
// from the day it was declared because the job that was supposed to write it
// was never built, so the lottery's own deadline handling lives here where it
// can be seen to run, and the poll half is finally wired up alongside it.
func (d *LotteryDrawer) Run() {
	if !d.running.TryLock() {
		return
	}
	defer d.running.Unlock()

	ctx := context.Background()
	drawn := d.svc.sweepDueLotteries(ctx)
	closed := d.svc.sweepClosedPolls()
	expired := d.svc.sweepExpiredClaims()
	if drawn > 0 || closed > 0 || expired > 0 {
		slog.Info("话题小程序到点处理完成",
			"lotteries_drawn", drawn, "polls_closed", closed, "claims_expired", expired)
	}
}

func (s *LotteryService) sweepDueLotteries(ctx context.Context) int {
	due, err := s.lotteryRepo.ClaimDue(time.Now(), drawSweepBatch)
	if err != nil {
		slog.Warn("抽奖到点扫描失败", "error", err)
		return 0
	}
	drawn := 0
	for i := range due {
		if appErr := s.draw(ctx, &due[i]); appErr != nil {
			slog.Warn("自动开奖失败, 已退回待开奖队列",
				"lottery_id", due[i].ID, "error", appErr.Message)
			if err := s.lotteryRepo.ReleaseDrawing(due[i].ID); err != nil {
				slog.Error("退回待开奖队列失败, 该抽奖会卡在 drawing",
					"lottery_id", due[i].ID, "error", err)
			}
			continue
		}
		drawn++
	}
	return drawn
}

// DrawOpen is the author's manual draw. It takes the same status flip the sweep
// takes, so the two cannot both draw the same lottery.
func (s *LotteryService) DrawOpen(ctx context.Context, lotteryID int) error {
	res := s.lotteryRepo.DB().Exec(
		`UPDATE topic_lottery SET status = 'drawing' WHERE id = ? AND status = 'open'`, lotteryID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrLotteryNotOpen
	}
	lottery, err := s.lotteryRepo.FindByID(lotteryID)
	if err != nil {
		return err
	}
	if appErr := s.draw(ctx, lottery); appErr != nil {
		if err := s.lotteryRepo.ReleaseDrawing(lotteryID); err != nil {
			slog.Error("退回待开奖队列失败", "lottery_id", lotteryID, "error", err)
		}
		return appErr
	}
	return nil
}

// draw fixes the winner list once and writes it down. It never recomputes:
// re-deriving winners at read time would change the answer every time someone
// new entered or a participant got banned.
func (s *LotteryService) draw(ctx context.Context, lottery *topicModel.TopicLottery) *errors.AppError {
	prizes, err := s.lotteryRepo.FindPrizes(lottery.ID)
	if err != nil {
		return errors.ErrInternal("获取奖项失败")
	}
	slots := make([]topicModel.TopicLotteryPrize, 0, len(prizes))
	for _, p := range prizes {
		for i := 0; i < p.Slots; i++ {
			slots = append(slots, p)
		}
	}

	var winners, pool []topicModel.TopicLotteryEntry
	var appErr *errors.AppError
	if lottery.EntryMode == topicModel.LotteryEntryFloor {
		winners, appErr = s.pickFloorWinners(ctx, lottery, len(slots))
	} else {
		pool, appErr = s.eligiblePool(ctx, lottery)
		if appErr == nil {
			winners = takeLowestRanked(pool, len(slots))
		}
	}
	if appErr != nil {
		return appErr
	}

	stampPointPayouts(lottery.Seed, slots, winners)
	refund := escrowRefund(lottery.PointEscrow, winners)

	now := time.Now()
	txErr := s.lotteryRepo.DB().Transaction(func(tx *gorm.DB) error {
		for i := range winners {
			prize := slots[i]
			fulfillment := topicModel.LotteryFulfillPending
			if prize.Delivery == topicModel.LotteryDeliveryPoint {
				fulfillment = topicModel.LotteryFulfillReceived
			}
			fields := map[string]any{
				"prize_id":      prize.ID,
				"rank_key":      winners[i].RankKey,
				"point_awarded": winners[i].PointAwarded,
				"won_at":        now,
				"fulfillment":   fulfillment,
				"updated":       now,
			}
			if prize.Delivery == topicModel.LotteryDeliveryCode {
				code, err := s.lotteryRepo.TakeCode(tx, prize.ID, winners[i].UserID)
				if err != nil {
					// A prize with fewer codes than slots is the author's mistake,
					// not a reason to abort the whole draw: the other winners keep
					// their prizes and this one falls back to manual delivery.
					slog.Warn("兑换码不足, 该名额改为人工发放",
						"lottery_id", lottery.ID, "prize_id", prize.ID, "user_id", winners[i].UserID)
				} else {
					fields["code_id"] = code.ID
					fields["claim_deadline"] = now.Add(lotteryClaimGrace)
				}
			}
			if winners[i].ID == 0 {
				entry := &topicModel.TopicLotteryEntry{
					LotteryID:    lottery.ID,
					UserID:       winners[i].UserID,
					ReplyFloor:   winners[i].ReplyFloor,
					PrizeID:      prize.ID,
					RankKey:      winners[i].RankKey,
					PointAwarded: winners[i].PointAwarded,
					WonAt:        &now,
					Fulfillment:  fulfillment,
				}
				if codeID, ok := fields["code_id"].(int); ok {
					entry.CodeID = codeID
				}
				if deadline, ok := fields["claim_deadline"].(time.Time); ok {
					entry.ClaimDeadline = &deadline
				}
				if err := s.lotteryRepo.CreateEntry(tx, entry); err != nil {
					return err
				}
				winners[i].ID = entry.ID
				continue
			}
			if err := s.lotteryRepo.UpdateEntryFields(tx, winners[i].ID, fields); err != nil {
				return err
			}
		}
		if err := s.lotteryRepo.SyncEntryCount(tx, lottery.ID); err != nil {
			return err
		}
		if refund > 0 {
			if err := s.lotteryRepo.AdjustCachedMoemoepoint(tx, lottery.UserID, refund); err != nil {
				return err
			}
		}
		return s.lotteryRepo.UpdateFields(tx, lottery.ID, map[string]any{
			"status": topicModel.LotteryStatusDrawn, "drawn_at": now, "updated": now,
			"point_escrow": 0,
		})
	})
	if txErr != nil {
		return errors.ErrInternal("开奖失败")
	}

	s.afterDraw(lottery, slots, winners, pool, refund)
	return nil
}

// eligiblePool ranks every eligible entry by HMAC(seed, lottery:user). Ban
// status is re-checked HERE rather than at entry time: an account banned between
// entering and the draw must not win, and the entry may be weeks old.
func (s *LotteryService) eligiblePool(
	ctx context.Context,
	lottery *topicModel.TopicLottery,
) ([]topicModel.TopicLotteryEntry, *errors.AppError) {
	entries, err := s.lotteryRepo.FindEntries(lottery.ID)
	if err != nil {
		return nil, errors.ErrInternal("获取参与记录失败")
	}
	uids := userclient.CollectIDs(entries, func(e topicModel.TopicLotteryEntry) int { return e.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	pool := make([]topicModel.TopicLotteryEntry, 0, len(entries))
	for _, e := range entries {
		if !userclient.IsRenderable(userMap[e.UserID]) {
			continue
		}
		e.RankKey = rankKey(lottery.Seed, lottery.ID, e.UserID)
		pool = append(pool, e)
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].RankKey < pool[j].RankKey })
	return pool, nil
}

func takeLowestRanked(pool []topicModel.TopicLotteryEntry, slots int) []topicModel.TopicLotteryEntry {
	if len(pool) > slots {
		return pool[:slots]
	}
	return pool
}

// pickFloorWinners resolves the floor rule against the replies that exist right
// now. A floor nobody reached, or one whose reply was deleted, simply has no
// winner — inventing a substitute would break the one property a floor lottery
// has, that a reader can verify it by counting.
func (s *LotteryService) pickFloorWinners(
	ctx context.Context,
	lottery *topicModel.TopicLottery,
	slots int,
) ([]topicModel.TopicLotteryEntry, *errors.AppError) {
	floors, err := ParseFloorRule(lottery.FloorRule, slots)
	if err != nil {
		return nil, errors.ErrInternal("楼层规则无法解析")
	}
	replies, err := s.lotteryRepo.FindRepliesByFloors(lottery.TopicID, floors)
	if err != nil {
		return nil, errors.ErrInternal("获取楼层失败")
	}
	byFloor := make(map[int]int, len(replies))
	for _, r := range replies {
		byFloor[r.Floor] = r.UserID
	}

	uids := make([]int, 0, len(replies))
	for _, r := range replies {
		uids = append(uids, r.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	seen := map[int]bool{}
	out := make([]topicModel.TopicLotteryEntry, 0, len(floors))
	for _, floor := range floors {
		uid, ok := byFloor[floor]
		if !ok || uid == lottery.UserID || seen[uid] {
			continue
		}
		if !userclient.IsRenderable(userMap[uid]) {
			continue
		}
		seen[uid] = true
		out = append(out, topicModel.TopicLotteryEntry{
			LotteryID: lottery.ID, UserID: uid, ReplyFloor: floor,
		})
	}
	return out, nil
}

// escrowRefund is what the author gets back at the draw: the pool they paid
// for less what the winners were actually paid. A lottery created before the
// author paid for it has no escrow and refunds nothing.
func escrowRefund(escrow int, winners []topicModel.TopicLotteryEntry) int {
	paid := 0
	for _, w := range winners {
		paid += w.PointAwarded
	}
	if escrow <= paid {
		return 0
	}
	return escrow - paid
}

func (s *LotteryService) afterDraw(
	lottery *topicModel.TopicLottery,
	slots []topicModel.TopicLotteryPrize,
	winners, pool []topicModel.TopicLotteryEntry,
	refund int,
) {
	won := make(map[int]bool, len(winners))
	specs := make([]msgService.Spec, 0, len(pool))
	for i := range winners {
		prize := slots[i]
		won[winners[i].UserID] = true
		content := fmt.Sprintf("恭喜您在抽奖「%s」中获得 %s", lottery.Title, prize.Name)
		if prize.Delivery == topicModel.LotteryDeliveryPoint && winners[i].PointAwarded > 0 {
			content += fmt.Sprintf(", 已发放 %d 萌萌点", winners[i].PointAwarded)
		}
		if prize.Delivery == topicModel.LotteryDeliveryCode && winners[i].ClaimDeadline != nil {
			content += fmt.Sprintf(", 请在 %s 前领取兑换码",
				winners[i].ClaimDeadline.Format("2006-01-02 15:04"))
		}
		specs = append(specs, msgService.Spec{
			SenderID:   lottery.UserID,
			ReceiverID: winners[i].UserID,
			Kind:       msgService.NotifyLotteryWon,
			Content:    content,
			TopicID:    lottery.TopicID,
		})
		if prize.Delivery == topicModel.LotteryDeliveryPoint && winners[i].PointAwarded > 0 {
			// content_approved is the only credit reason infra exposes to a
			// downstream, so a lottery payout shows up in the user's moemoepoint
			// log labelled as approved content. Relabelling needs a new reason on
			// the OAuth side, not a local workaround.
			s.award(winners[i].UserID, winners[i].PointAwarded,
				moemoepoint.ReasonContentApproved,
				moemoepoint.Ref("topic_lottery", lottery.ID),
				fmt.Sprintf("kungal:lottery_won:%d_%d", lottery.ID, winners[i].UserID))
		}
	}
	// Someone who entered and lost is still waiting on an answer, and the topic
	// page will not tell them unless they go back and look.
	for _, entry := range pool {
		if won[entry.UserID] {
			continue
		}
		specs = append(specs, msgService.Spec{
			SenderID:   lottery.UserID,
			ReceiverID: entry.UserID,
			Kind:       msgService.NotifyLotteryClosed,
			Content:    fmt.Sprintf("您参与的抽奖「%s」已经开奖, 很遗憾这次没有中奖", lottery.Title),
			TopicID:    lottery.TopicID,
		})
	}
	if refund > 0 {
		s.award(lottery.UserID, refund, moemoepoint.ReasonContentApproved,
			moemoepoint.Ref("topic_lottery_escrow", lottery.ID), EscrowRefundKey(lottery.ID))
	}
	if err := s.notifier.EmitMany(nil, specs); err != nil {
		slog.Warn("开奖通知发送失败", "lottery_id", lottery.ID, "error", err)
	}
}

// A lottery is refunded at most once in its life: the draw, a cancel and a
// delete each zero point_escrow in the transaction that decides the refund.
func EscrowRefundKey(lotteryID int) string {
	return moemoepoint.Key("lottery_escrow_refund", "topic_lottery_"+strconv.Itoa(lotteryID))
}

// sweepClosedPolls finally writes topic_poll.notification_sent. The column has
// existed since the poll mini-app shipped with no reader and no writer anywhere
// in the tree: "tell the voters the poll closed" was declared in the schema and
// never built, so a deadline passed in silence.
func (s *LotteryService) sweepClosedPolls() int {
	polls, err := s.lotteryRepo.ClaimPollsPastDeadline(time.Now(), drawSweepBatch)
	if err != nil {
		slog.Warn("投票截止扫描失败", "error", err)
		return 0
	}
	for _, poll := range polls {
		voterIDs, err := s.lotteryRepo.FindPollVoterIDs(poll.ID)
		if err != nil {
			slog.Warn("获取投票参与者失败", "poll_id", poll.ID, "error", err)
			continue
		}
		specs := make([]msgService.Spec, 0, len(voterIDs))
		for _, uid := range voterIDs {
			specs = append(specs, msgService.Spec{
				SenderID:   poll.UserID,
				ReceiverID: uid,
				Kind:       msgService.NotifyPollClosed,
				Content:    fmt.Sprintf("您参与的投票「%s」已经截止, 可以查看结果了", poll.Title),
				TopicID:    poll.TopicID,
			})
		}
		if err := s.notifier.EmitMany(nil, specs); err != nil {
			slog.Warn("投票截止通知发送失败", "poll_id", poll.ID, "error", err)
		}
	}
	return len(polls)
}

// sweepExpiredClaims voids a code prize the winner never revealed. The code
// stays sealed and stays attached to the entry: nothing on the site can read it
// back out, so handing it to someone else is not something this can offer.
func (s *LotteryService) sweepExpiredClaims() int {
	expired, err := s.lotteryRepo.ClaimExpiredCodeWins(time.Now(), drawSweepBatch)
	if err != nil {
		slog.Warn("兑换码领取期限扫描失败", "error", err)
		return 0
	}
	if len(expired) == 0 {
		return 0
	}
	specs := make([]msgService.Spec, 0, len(expired))
	for _, entry := range expired {
		lottery, err := s.lotteryRepo.FindByID(entry.LotteryID)
		if err != nil {
			continue
		}
		specs = append(specs, msgService.Spec{
			SenderID:   lottery.UserID,
			ReceiverID: entry.UserID,
			Kind:       msgService.NotifyLotteryExpired,
			Content: fmt.Sprintf("您在抽奖「%s」中的兑换码超过领取期限未领取, 已作废",
				lottery.Title),
			TopicID: lottery.TopicID,
		})
	}
	if err := s.notifier.EmitMany(nil, specs); err != nil {
		slog.Warn("兑换码过期通知发送失败", "error", err)
	}
	return len(expired)
}
