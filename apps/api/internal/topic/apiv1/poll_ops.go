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
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type Polls struct {
	reads *Service
	polls *repository.PollRepository
	check *gate.CheckService
	scan  *gate.ScanService
}

func NewPolls(reads *Service, polls *repository.PollRepository, check *gate.CheckService, scan *gate.ScanService) *Polls {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	if polls == nil && reads != nil && reads.topics != nil {
		polls = repository.NewPollRepository(reads.topics.DB())
	}
	return &Polls{reads: reads, polls: polls, check: check, scan: scan}
}

func (p *Polls) ready() *problem.Problem {
	if p == nil || p.reads == nil || p.polls == nil || p.polls.DB() == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (p *Polls) db() *gorm.DB {
	return p.polls.DB()
}

func (p *Polls) rejectContent(ctx context.Context, text string, authorID int) (gate.Verdict, *problem.Problem) {
	id := int64(authorID)
	verdict := p.check.Verdict(ctx, text, &id)
	if verdict.Decision == gate.DecisionDeny {
		return verdict, contentRejected()
	}
	return verdict, nil
}

func (p *Polls) scanPoll(verdict gate.Verdict, pollID, authorID int, text string) {
	if verdict.Decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTopicPoll, "subject_id", pollID, "author_id", authorID, "matched", verdict.Matched)
	}
	p.scan.ScanBg(gate.SubjectKindTopicPoll, strconv.Itoa(pollID), text, int64(authorID), verdict.Skipped)
}

func (p *Polls) visiblePoll(ctx context.Context, idStr string) (*model.TopicPoll, *model.Topic, *middleware.UserInfo, *problem.Problem) {
	if prob := p.ready(); prob != nil {
		return nil, nil, nil, prob
	}
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, nil, nil, notFound()
	}
	poll, err := p.polls.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, notFound()
		}
		return nil, nil, nil, problem.Internal(err)
	}
	topic, user, prob := p.reads.visibleTopic(ctx, strconv.Itoa(poll.TopicID))
	if prob != nil {
		return nil, nil, nil, prob
	}
	if prob := p.reads.rejectUnrenderableAuthor(ctx, poll.UserID); prob != nil {
		return nil, nil, nil, prob
	}
	return poll, topic, user, nil
}

type listTopicPollsInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type listTopicPollsOutput struct {
	Body repr.List[Poll]
}

type pollInput struct {
	PollID string `path:"poll_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Poll id."`
}

type pollOutput struct {
	Body Poll
}

type createPollInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    PollCreate
}

type createPollOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new poll, such as /api/v1/polls/412."`
	Body     Poll
}

type updatePollInput struct {
	PollID string `path:"poll_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Poll id."`
	Body   PollPatch
}

type setPollVoteInput struct {
	PollID string `path:"poll_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Poll id."`
	Body   PollVoteSet
}

type listPollVotesInput struct {
	PollID string `path:"poll_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Poll id."`
	collect.Page
}

type listPollVotesOutput struct {
	Body repr.List[PollVote]
}
