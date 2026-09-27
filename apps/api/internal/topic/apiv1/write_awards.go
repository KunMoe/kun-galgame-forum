package apiv1

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type pendingAward struct {
	userID int
	delta  int
	reason string
	ref    string
	key    string
}

func (w *Writes) db() *gorm.DB {
	if w == nil || w.reads == nil || w.reads.topics == nil {
		return nil
	}
	return w.reads.topics.DB()
}

func (w *Writes) caller(ctx context.Context) *middleware.UserInfo {
	return v1.User(ctx)
}

func (w *Writes) flushAwards(awards []pendingAward) {
	if w.award == nil {
		return
	}
	for _, a := range awards {
		w.award(a.userID, a.delta, a.reason, a.ref, a.key)
	}
}

func (w *Writes) notifyMentions(tx *gorm.DB, senderID, topicID, replyFloor int, content string) error {
	muted, err := mutedReaders(tx, topicID, service.MentionedIDs(content, mentionCap))
	if err != nil {
		return err
	}
	var h service.InteractionHelpers
	return h.NotifyMentionsLimited(tx, senderID, topicID, replyFloor, 0, content, mentionCap, muted)
}

func mutedReaders(tx *gorm.DB, topicID int, userIDs []int) (map[int]bool, error) {
	levels, err := repository.SubscriptionLevels(tx, topicID, userIDs)
	if err != nil {
		return nil, err
	}
	muted := make(map[int]bool, len(levels))
	for id, level := range levels {
		if level == model.SubscriptionMuted {
			muted[id] = true
		}
	}
	return muted, nil
}

func topicCreatedAward(userID, topicID int, consume bool) pendingAward {
	delta := topicSectionFootprint(consume)
	reason := moemoepoint.ReasonContentApproved
	if consume {
		reason = moemoepoint.ReasonContentRemoved
	}
	return pendingAward{
		userID: userID,
		delta:  delta,
		reason: reason,
		ref:    moemoepoint.Ref("topic", topicID),
		key:    moemoepoint.Key("topic_created", fmt.Sprintf("topic_%d", topicID)),
	}
}

func topicCostChangedAward(authorID, topicID, delta int) pendingAward {
	reason := moemoepoint.ReasonContentApproved
	if delta < 0 {
		reason = moemoepoint.ReasonContentRemoved
	}
	return pendingAward{
		userID: authorID,
		delta:  delta,
		reason: reason,
		ref:    moemoepoint.Ref("topic", topicID),
		key:    moemoepoint.KeyNonce("topic_cost_changed", fmt.Sprintf("topic_%d", topicID)),
	}
}

func repliedAward(authorID, replyID int) pendingAward {
	return pendingAward{
		userID: authorID,
		delta:  1,
		reason: moemoepoint.ReasonContentApproved,
		ref:    moemoepoint.Ref("topic_reply", replyID),
		key:    moemoepoint.Key("replied", fmt.Sprintf("topic_reply_%d", replyID)),
	}
}

func replyDeletedAward(authorID, replyID, penalty int) pendingAward {
	return pendingAward{
		userID: authorID,
		delta:  -penalty,
		reason: moemoepoint.ReasonContentRemoved,
		ref:    moemoepoint.Ref("topic_reply", replyID),
		key:    moemoepoint.Key("reply_deleted", fmt.Sprintf("topic_reply_%d", replyID)),
	}
}

func (w *Writes) rejectContent(ctx context.Context, text string, authorID int) (decision string, matched []string, p *problem.Problem) {
	id := int64(authorID)
	decision, matched = w.check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func (w *Writes) scanTopic(decision string, matched []string, topicID, authorID int, text string) {
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTopic, "subject_id", topicID, "author_id", authorID, "matched", matched)
	}
	w.scan.ScanBg(gate.SubjectKindTopic, strconv.Itoa(topicID), text, int64(authorID))
}

func (w *Writes) scanReply(decision string, matched []string, replyID, authorID int, text string) {
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindReply, "subject_id", replyID, "author_id", authorID, "matched", matched)
	}
	w.scan.ScanBg(gate.SubjectKindReply, strconv.Itoa(replyID), text, int64(authorID))
}

func (w *Writes) loadTopic(id int) (*model.Topic, *problem.Problem) {
	topic, err := w.reads.topics.FindByID(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return topic, nil
}

func (w *Writes) loadReply(id int) (*model.TopicReply, *problem.Problem) {
	row, err := w.reads.replies.FindByID(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return row, nil
}

type txFail struct {
	p *problem.Problem
}

func (e txFail) Error() string {
	if e.p == nil {
		return "tx fail"
	}
	return e.p.Error()
}

func txProblem(err error) *problem.Problem {
	var f txFail
	if asTxFail(err, &f) {
		return f.p
	}
	return nil
}

func asTxFail(err error, dest *txFail) bool {
	if err == nil {
		return false
	}
	f, ok := err.(txFail)
	if !ok {
		return false
	}
	*dest = f
	return true
}
