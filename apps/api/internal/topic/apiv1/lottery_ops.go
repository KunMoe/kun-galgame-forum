package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type Lotteries struct {
	reads  *Service
	svc    *service.LotteryService
	lots   *repository.LotteryRepository
	check  *gate.CheckService
	scan   *gate.ScanService
	award  AwardFunc
	images func(hashes []string) map[string]imageclient.ImageMeta
}

func NewLotteries(
	reads *Service,
	svc *service.LotteryService,
	check *gate.CheckService,
	scan *gate.ScanService,
	award AwardFunc,
	images func(hashes []string) map[string]imageclient.ImageMeta,
) *Lotteries {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	l := &Lotteries{reads: reads, svc: svc, check: check, scan: scan, award: award, images: images}
	if svc != nil {
		l.lots = svc.Repo()
	}
	return l
}

func (l *Lotteries) ready() *problem.Problem {
	if l == nil || l.reads == nil || l.svc == nil || l.lots == nil || l.lots.DB() == nil || l.award == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (l *Lotteries) db() *gorm.DB {
	return l.lots.DB()
}

func (l *Lotteries) findLottery(idStr string) (*model.TopicLottery, *problem.Problem) {
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, notFound()
	}
	lottery, err := l.lots.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	return lottery, nil
}

// visibleLottery is a lottery whose topic getTopic would return to the caller,
// written by someone who is not banned. The legacy face showed a banned
// author's lottery while hiding a banned author's poll on the same page.
func (l *Lotteries) visibleLottery(ctx context.Context, idStr string) (*model.TopicLottery, *model.Topic, *middleware.UserInfo, *problem.Problem) {
	if prob := l.ready(); prob != nil {
		return nil, nil, nil, prob
	}
	lottery, prob := l.findLottery(idStr)
	if prob != nil {
		return nil, nil, nil, prob
	}
	topic, user, prob := l.reads.visibleTopic(ctx, strconv.Itoa(lottery.TopicID))
	if prob != nil {
		return nil, nil, nil, prob
	}
	if prob := l.reads.rejectUnrenderableAuthor(ctx, lottery.UserID); prob != nil {
		return nil, nil, nil, prob
	}
	return lottery, topic, user, nil
}

func (l *Lotteries) rejectContent(ctx context.Context, text string, authorID int) (string, []string, *problem.Problem) {
	id := int64(authorID)
	decision, matched := l.check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func (l *Lotteries) scanLottery(decision string, matched []string, lotteryID, authorID int, text string) {
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTopicLottery, "subject_id", lotteryID, "author_id", authorID, "matched", matched)
	}
	l.scan.ScanBg(gate.SubjectKindTopicLottery, strconv.Itoa(lotteryID), text, int64(authorID))
}

func lotteryClosed() *problem.Problem {
	return problem.New(problem.CodeLotteryClosed, "The lottery is not open, or it is past closes_at.")
}

func lotteryIneligible(reason string) *problem.Problem {
	p := problem.New(problem.CodeLotteryIneligible, "The caller does not meet this lottery's entry requirements.")
	p.SetExtension("reason", reason)
	return p
}

func lotteryDrawn() *problem.Problem {
	return problem.New(problem.CodeLotteryDrawn, "The lottery is being drawn, or has been drawn and only staff may delete it.")
}

func invalidStateTransition(detail string) *problem.Problem {
	return problem.New(problem.CodeInvalidStateTransition, detail)
}

type listTopicLotteriesInput struct {
	TopicID     string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, prize images marked adult or graded explicit carry their image. Default false, which withholds it."`
}

type listTopicLotteriesOutput struct {
	Body repr.List[Lottery]
}

type lotteryInput struct {
	LotteryID string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
}

type getLotteryInput struct {
	LotteryID   string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, prize images marked adult or graded explicit carry their image. Default false, which withholds it."`
}

type lotteryOutput struct {
	Body Lottery
}

type createLotteryInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    LotteryCreate
}

type createLotteryOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new lottery, such as /api/v1/lotteries/412."`
	Body     Lottery
}

type updateLotteryInput struct {
	LotteryID string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
	Body      LotteryPatch
}

type deleteLotteryOutput struct{}

type listLotteryEntriesInput struct {
	LotteryID string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
	collect.Page
}

type listLotteryEntriesOutput struct {
	Body repr.List[LotteryEntry]
}

type updateLotteryWinnerInput struct {
	LotteryID string `path:"lottery_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Lottery id."`
	WinnerID  string `path:"winner_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Winner id, as in LotteryWinner.id."`
	Body      LotteryWinnerPatch
}

type lotteryWinnerOutput struct {
	Body LotteryWinner
}

type lotteryCodeRevealOutput struct {
	Body LotteryCodeReveal
}
