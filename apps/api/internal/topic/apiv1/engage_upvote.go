package apiv1

import (
	"context"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func trimUpvoteNote(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func notePtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (x *Interactions) upvoteTopic(ctx context.Context, in *upvoteTopicInput) (*upvoteTopicOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if user.ID == topic.UserID {
		return nil, selfUpvoteForbidden()
	}
	note := trimUpvoteNote(in.Body.Note)
	var (
		jobs    []pendingAward
		rowID   int
		created time.Time
	)
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		balance, err := x.reads.topics.LockMoemoepoint(tx, user.ID)
		if err != nil {
			return err
		}
		if balance < constants.CostUpvoteSender {
			p := problem.New(problem.CodeMoemoepointInsufficient, "The caller's moemoepoint balance is below what the operation costs.")
			p.SetExtension("required", constants.CostUpvoteSender)
			return engageError{p: p}
		}
		id, at, err := x.reads.topics.InsertTopicUpvote(tx, topic.ID, user.ID, note)
		if err != nil {
			return err
		}
		rowID, created = id, at
		if err := x.reads.topics.ApplyUpvoteCountAndTime(tx, topic.ID, time.Now()); err != nil {
			return err
		}
		ref := moemoepoint.Ref("topic_upvote", id)
		jobs = append(jobs,
			pendingAward{
				userID: user.ID,
				delta:  -constants.CostUpvoteSender,
				reason: moemoepoint.ReasonContentRemoved,
				ref:    ref,
				key:    moemoepoint.Key("upvote_sent", "topic_upvote_"+strconv.Itoa(id)),
			},
			pendingAward{
				userID: topic.UserID,
				delta:  constants.RewardUpvoteOwner,
				reason: moemoepoint.ReasonContentApproved,
				ref:    ref,
				key:    moemoepoint.Key("upvote_received", "topic_upvote_"+strconv.Itoa(id)),
			},
		)
		return notifyTopicLink(tx, user.ID, topic.UserID, "upvoted", topicPreview(topic.Title), topic.ID, 0, msgModel.TopicRef(topic.ID))
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	users, p := x.reads.lookupUsers(ctx, []int{user.ID})
	if p != nil {
		return nil, p
	}
	upvoter := repr.DeletedUserRef(user.ID)
	if u, ok := users[user.ID]; ok {
		upvoter = repr.NewUserRef(x.reads.cdn, u)
	}
	return &upvoteTopicOutput{Body: TopicUpvote{
		Object:    "topic_upvote",
		ID:        repr.ID(rowID),
		TopicID:   repr.ID(topic.ID),
		Upvoter:   upvoter,
		Note:      notePtr(note),
		CreatedAt: repr.Timestamp(created),
	}}, nil
}
