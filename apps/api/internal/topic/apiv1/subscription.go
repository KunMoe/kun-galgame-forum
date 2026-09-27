package apiv1

import (
	"context"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/access"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

const subscriptionSort = "last_reply_desc"

type TopicNotificationLevel string

func (TopicNotificationLevel) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum(model.SubscriptionWatching, model.SubscriptionNormal, model.SubscriptionMuted)
	s.Description = "How the caller hears about replies to this topic. watching: every new reply, folded into one notification per topic. " +
		"normal: only replies that mention the caller, and replied for the topic's own author only while watching. " +
		"muted: no reply, mention or comment notification from this topic."
	return s
}

type TopicSubscription struct {
	Object            string                 `json:"object" enum:"topic_subscription" maxLength:"18" doc:"Type discriminant. Always topic_subscription."`
	ID                repr.DecimalID         `json:"id" doc:"The topic id, the same value as topic_id."`
	TopicID           repr.DecimalID         `json:"topic_id" doc:"Topic id."`
	NotificationLevel TopicNotificationLevel `json:"notification_level"`
	LastReadFloor     int                    `json:"last_read_floor" minimum:"0" doc:"The highest floor recorded as read. Kept only while notification_level is watching; 0 otherwise."`
}

type SubscribedTopic struct {
	Object           string         `json:"object" enum:"subscribed_topic" maxLength:"16" doc:"Type discriminant. Always subscribed_topic."`
	ID               repr.DecimalID `json:"id" doc:"The topic id."`
	Topic            *TopicSummary  `json:"topic" doc:"The topic, shaped as listTopics shapes it. Never null in this list; the schema is nullable because every property named topic in this API shares the nullable TopicSummary."`
	LastReadFloor    int            `json:"last_read_floor" minimum:"0" doc:"The highest floor recorded as read."`
	UnreadReplyCount int            `json:"unread_reply_count" minimum:"0" doc:"Visible replies by other users above last_read_floor, counted now."`
	FirstUnreadFloor *int           `json:"first_unread_floor" minimum:"1" doc:"The lowest of those floors, for a link that opens at the first unread reply. null when unread_reply_count is 0."`
	LastReplyAt      repr.DateTime  `json:"last_reply_at" doc:"When a reply last landed in this topic since the caller subscribed; the subscription time when none has."`
}

type topicSubscriptionInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type TopicSubscriptionWrite struct {
	NotificationLevel TopicNotificationLevel `json:"notification_level"`
}

type setTopicSubscriptionInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    TopicSubscriptionWrite
}

type TopicReadMarkerWrite struct {
	Floor int `json:"floor" minimum:"0" maximum:"2147483647" doc:"The highest floor the caller has seen. Floors above the topic's last visible floor are stored as that floor."`
}

type markTopicReadInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    TopicReadMarkerWrite
}

type topicSubscriptionOutput struct {
	Body TopicSubscription
}

type listTopicSubscriptionsInput struct {
	collect.Page
	HasUnread   bool `query:"has_unread" default:"false" doc:"When true, only topics with a visible reply by another user above last_read_floor."`
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, NSFW topics are included. Default false."`
}

type listTopicSubscriptionsOutput struct {
	Body repr.List[SubscribedTopic]
}

func subscriptionBody(topicID int, row *model.TopicSubscription) TopicSubscription {
	out := TopicSubscription{
		Object: "topic_subscription", ID: repr.ID(topicID), TopicID: repr.ID(topicID),
		NotificationLevel: model.SubscriptionNormal,
	}
	if row != nil {
		out.NotificationLevel = TopicNotificationLevel(row.NotificationLevel)
		if row.NotificationLevel == model.SubscriptionWatching {
			out.LastReadFloor = row.LastReadFloor
		}
	}
	return out
}

func (x *Interactions) getTopicSubscription(ctx context.Context, in *topicSubscriptionInput) (*topicSubscriptionOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, user, p := x.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	row, err := repository.FindSubscription(x.db, user.ID, topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &topicSubscriptionOutput{Body: subscriptionBody(topic.ID, row)}, nil
}

func (x *Interactions) setTopicSubscription(ctx context.Context, in *setTopicSubscriptionInput) (*topicSubscriptionOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, user, p := x.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	var row *model.TopicSubscription
	err := x.db.Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = repository.SetSubscriptionLevel(tx, user.ID, topic.ID, string(in.Body.NotificationLevel))
		return err
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &topicSubscriptionOutput{Body: subscriptionBody(topic.ID, row)}, nil
}

func (x *Interactions) markTopicRead(ctx context.Context, in *markTopicReadInput) (*topicSubscriptionOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, user, p := x.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	var row *model.TopicSubscription
	err := x.db.Transaction(func(tx *gorm.DB) error {
		var err error
		row, err = repository.MarkTopicRead(tx, user.ID, topic.ID, in.Body.Floor)
		return err
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &topicSubscriptionOutput{Body: subscriptionBody(topic.ID, row)}, nil
}

func (x *Interactions) listTopicSubscriptions(ctx context.Context, in *listTopicSubscriptionsInput) (*listTopicSubscriptionsOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	user := v1.User(ctx)
	fp := collect.Fingerprint("topic_subscriptions", strconv.Itoa(user.ID),
		strconv.FormatBool(in.HasUnread), strconv.FormatBool(in.IncludeNSFW))
	keys, curErr := collect.DecodeCursor(in.Cursor, subscriptionSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	query := repository.SubscribedQuery{
		UserID: user.ID, HasUnread: in.HasUnread, IncludeNSFW: in.IncludeNSFW, Limit: in.Limit,
	}
	if keys != nil {
		pos, ok := parseSubscribedPos(keys)
		if !ok {
			return nil, invalidCursor()
		}
		query.Pos = pos
	}

	view := access.Snapshot(user)
	items := make([]SubscribedTopic, 0, in.Limit)
	hasMore := false
	var last repository.SubscribedTopicRow
windows:
	for range maxWindows {
		rows, err := x.reads.topics.ListSubscribed(query)
		if err != nil {
			return nil, problem.Internal(err)
		}
		more := len(rows) > in.Limit
		if more {
			rows = rows[:in.Limit]
		}
		readable, p := x.readableSubscriptions(rows, view)
		if p != nil {
			return nil, p
		}
		keyset := make([]repository.TopicKeysetRow, 0, len(rows))
		for i, row := range rows {
			if readable[i] {
				keyset = append(keyset, row.TopicKeysetRow)
			}
		}
		rendered, p := x.reads.summaries(ctx, keyset)
		if p != nil {
			return nil, p
		}
		next := 0
		for i, row := range rows {
			last = row
			if readable[i] {
				summary := rendered[next]
				next++
				if summary != nil {
					items = append(items, subscribedItem(row, *summary))
				}
			}
			if len(items) == in.Limit {
				hasMore = more || i < len(rows)-1
				break windows
			}
		}
		hasMore = more
		if !more {
			break
		}
		query.Pos = &repository.SubscribedPos{ActivityAt: last.ActivityAt, TopicID: last.ID}
	}

	var nextCursor *string
	if hasMore {
		cur := collect.EncodeCursor(subscriptionSort, fp,
			last.ActivityAt.UTC().Format(time.RFC3339Nano), strconv.Itoa(last.ID))
		nextCursor = &cur
	}
	return &listTopicSubscriptionsOutput{Body: repr.NewList(items, nextCursor)}, nil
}

// A subscription outlives the access that allowed it: the topic can be hidden
// or narrowed to a role later, so every row is judged again as getTopic would.
func (x *Interactions) readableSubscriptions(rows []repository.SubscribedTopicRow, view access.View) ([]bool, *problem.Problem) {
	var restricted []int
	for _, row := range rows {
		topic := subscribedTopicModel(row)
		if access.NeedsGrants(&topic) {
			restricted = append(restricted, row.ID)
		}
	}
	grants := map[int][]model.TopicAccessGrant{}
	if len(restricted) > 0 {
		var err error
		grants, err = x.reads.topics.FindAccessGrantsForTopics(restricted)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	out := make([]bool, len(rows))
	for i, row := range rows {
		topic := subscribedTopicModel(row)
		out[i] = access.Allowed(&topic, view, grants[row.ID])
	}
	return out, nil
}

func subscribedTopicModel(row repository.SubscribedTopicRow) model.Topic {
	return model.Topic{ID: row.ID, Status: row.Status, UserID: row.UserID, AccessScope: row.AccessScope, HiddenBy: row.HiddenBy}
}

func subscribedItem(row repository.SubscribedTopicRow, summary TopicSummary) SubscribedTopic {
	return SubscribedTopic{
		Object:           "subscribed_topic",
		ID:               repr.ID(row.ID),
		Topic:            &summary,
		LastReadFloor:    row.LastReadFloor,
		UnreadReplyCount: row.UnreadCount,
		FirstUnreadFloor: row.FirstUnreadFloor,
		LastReplyAt:      repr.Timestamp(row.ActivityAt),
	}
}

func parseSubscribedPos(keys []string) (*repository.SubscribedPos, bool) {
	if len(keys) != 2 {
		return nil, false
	}
	at, e1 := time.Parse(time.RFC3339Nano, keys[0])
	id, e2 := strconv.Atoi(keys[1])
	if e1 != nil || e2 != nil || id <= 0 {
		return nil, false
	}
	return &repository.SubscribedPos{ActivityAt: at, TopicID: id}, true
}
