package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type AdminTopics struct {
	reads *Service
	lots  *repository.LotteryRepository
	award AwardFunc
}

func NewAdminTopics(reads *Service, lots *repository.LotteryRepository, award AwardFunc) *AdminTopics {
	return &AdminTopics{reads: reads, lots: lots, award: award}
}

func (a *AdminTopics) ready() *problem.Problem {
	if a == nil || a.reads == nil || a.reads.topics == nil || a.lots == nil || a.award == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

type HiddenTopicSummary struct {
	Object     string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID         repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title      string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	State      string         `json:"state" enum:"published,hidden" maxLength:"9" doc:"Lifecycle state. Always hidden in this collection."`
	HiddenBy   *string        `json:"hidden_by" enum:"author,moderator,trust" maxLength:"9" doc:"Who hid the topic: its author, a moderator, or the trust-and-safety service."`
	Author     repr.UserRef   `json:"author" doc:"The topic's author. Banned authors are listed too: this is a staff table."`
	ReplyCount int            `json:"reply_count" minimum:"0" doc:"Number of replies."`
	BumpedAt   repr.DateTime  `json:"bumped_at" doc:"Bump time, which is also this collection's sort key."`
	CreatedAt  repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type AdminTopic struct {
	Object            string         `json:"object" enum:"admin_topic" maxLength:"11" doc:"Type discriminant. Always admin_topic."`
	ID                repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title             string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	State             string         `json:"state" enum:"published,hidden" maxLength:"9" doc:"Lifecycle state."`
	HiddenBy          *string        `json:"hidden_by" enum:"author,moderator,trust" maxLength:"9" doc:"Who hid the topic. null when state is published."`
	Author            repr.UserRef   `json:"author" doc:"The topic's author."`
	ReplyCount        int            `json:"reply_count" minimum:"0" doc:"Replies the purge deletes."`
	CommentCount      int            `json:"comment_count" minimum:"0" doc:"Comments the purge deletes."`
	PollCount         int            `json:"poll_count" minimum:"0" doc:"Polls the purge deletes."`
	LotteryCount      int            `json:"lottery_count" minimum:"0" doc:"Lotteries the purge deletes, drawn or not."`
	DrawnLotteryCount int            `json:"drawn_lottery_count" minimum:"0" doc:"Of lottery_count, how many are drawn: their winners lose the page they collect prizes from."`
	FavoriteCount     int            `json:"favorite_count" minimum:"0" doc:"Favorites the purge deletes."`
	OpenLotteryEscrow int            `json:"open_lottery_escrow" minimum:"0" doc:"Moemoepoint held by the topic's open lotteries, which purgeTopic hands back to their authors, who paid for those point prizes."`
}

type listHiddenTopicsInput struct {
	collect.PageNumber
	HiddenBy string `query:"hidden_by" enum:"author,moderator,trust" maxLength:"9" doc:"Only topics hidden this way. Absent means every hidden topic."`
	Q        string `query:"q" maxLength:"100" doc:"Case-insensitive substring of the title. Free text; never use it as a decision input."`
}

type listHiddenTopicsOutput struct {
	Body repr.PageList[HiddenTopicSummary]
}

type adminTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type adminTopicOutput struct {
	Body AdminTopic
}

type purgeTopicOutput struct{}

func (a *AdminTopics) require(ctx context.Context, p perm.Permission) *problem.Problem {
	if prob := a.ready(); prob != nil {
		return prob
	}
	if !v1.User(ctx).Can(p) {
		return permissionRequired()
	}
	return nil
}

func (a *AdminTopics) listHiddenTopics(ctx context.Context, in *listHiddenTopicsInput) (*listHiddenTopicsOutput, error) {
	if prob := a.require(ctx, perm.TopicViewHidden); prob != nil {
		return nil, prob
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	rows, count, err := a.reads.topics.ListHiddenTopics(in.HiddenBy, strings.TrimSpace(in.Q), in.Offset(), in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	uids := make([]int, len(rows))
	for i, row := range rows {
		uids[i] = row.UserID
	}
	users, prob := a.reads.lookupUsers(ctx, uids)
	if prob != nil {
		return nil, prob
	}
	items := make([]HiddenTopicSummary, 0, len(rows))
	for _, row := range rows {
		state, hiddenBy, err := topicLifecycle(row.Status, row.HiddenBy)
		if err != nil {
			return nil, problem.Internal(err)
		}
		author := repr.DeletedUserRef(row.UserID)
		if u, ok := users[row.UserID]; ok {
			author = repr.NewUserRef(a.reads.cdn, u)
		}
		items = append(items, HiddenTopicSummary{
			Object:     "topic",
			ID:         repr.ID(row.ID),
			Title:      row.Title,
			State:      state,
			HiddenBy:   hiddenBy,
			Author:     author,
			ReplyCount: row.ReplyCount,
			BumpedAt:   repr.Timestamp(row.StatusUpdateTime),
			CreatedAt:  repr.Timestamp(row.Created),
		})
	}
	total, relation := collect.ClampTotal(count)
	return &listHiddenTopicsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (a *AdminTopics) findTopic(tx *gorm.DB, idStr string, lock bool) (*repository.HiddenTopicRow, *problem.Problem) {
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, notFound()
	}
	row, err := a.reads.topics.FindHiddenTopicRow(tx, id, lock)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	return row, nil
}

func (a *AdminTopics) getAdminTopic(ctx context.Context, in *adminTopicInput) (*adminTopicOutput, error) {
	if prob := a.require(ctx, perm.TopicDeleteAny); prob != nil {
		return nil, prob
	}
	db := a.reads.topics.DB()
	row, prob := a.findTopic(db, in.TopicID, false)
	if prob != nil {
		return nil, prob
	}
	counts, err := a.reads.topics.CountTopicDependents(db, row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	escrow, err := a.reads.topics.SumOpenEscrow(db, row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	state, hiddenBy, err := topicLifecycle(row.Status, row.HiddenBy)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, prob := a.reads.lookupUsers(ctx, []int{row.UserID})
	if prob != nil {
		return nil, prob
	}
	author := repr.DeletedUserRef(row.UserID)
	if u, ok := users[row.UserID]; ok {
		author = repr.NewUserRef(a.reads.cdn, u)
	}
	return &adminTopicOutput{Body: AdminTopic{
		Object:            "admin_topic",
		ID:                repr.ID(row.ID),
		Title:             row.Title,
		State:             state,
		HiddenBy:          hiddenBy,
		Author:            author,
		ReplyCount:        int(counts.Replies),
		CommentCount:      int(counts.Comments),
		PollCount:         int(counts.Polls),
		LotteryCount:      int(counts.Lotteries),
		DrawnLotteryCount: int(counts.DrawnLotteries),
		FavoriteCount:     int(counts.Favorites),
		OpenLotteryEscrow: escrow,
	}}, nil
}

type escrowRefund struct {
	lotteryID int
	authorID  int
	amount    int
}

// purgeTopic hard-deletes a topic with everything that cascades from it. The
// cascade reaches topic_lottery, so an open lottery's escrow would vanish with
// the row; it is handed back to its author in the same transaction instead.
func (a *AdminTopics) purgeTopic(ctx context.Context, in *adminTopicInput) (*purgeTopicOutput, error) {
	if prob := a.require(ctx, perm.TopicDeleteAny); prob != nil {
		return nil, prob
	}
	operator := v1.User(ctx)
	var (
		row     *repository.HiddenTopicRow
		counts  repository.TopicPurgeCounts
		refunds []escrowRefund
	)
	err := a.reads.topics.DB().Transaction(func(tx *gorm.DB) error {
		found, prob := a.findTopic(tx, in.TopicID, true)
		if prob != nil {
			return engageError{p: prob}
		}
		row = found
		lotteries, err := a.reads.topics.LockTopicLotteries(tx, row.ID)
		if err != nil {
			return err
		}
		for _, l := range lotteries {
			if l.Status == model.LotteryStatusDrawing {
				return engageError{p: lotteryDrawn()}
			}
			if l.Status == model.LotteryStatusOpen && l.PointEscrow > 0 {
				if err := a.lots.AdjustCachedMoemoepoint(tx, l.UserID, l.PointEscrow); err != nil {
					return err
				}
				refunds = append(refunds, escrowRefund{lotteryID: l.ID, authorID: l.UserID, amount: l.PointEscrow})
			}
		}
		counts, err = a.reads.topics.CountTopicDependents(tx, row.ID)
		if err != nil {
			return err
		}
		return a.reads.topics.PurgeTopic(tx, row.ID)
	})
	if err != nil {
		return nil, mapEngageErr(err)
	}
	for _, r := range refunds {
		a.award(r.authorID, r.amount, moemoepoint.ReasonContentApproved, escrowRef(r.lotteryID), service.EscrowRefundKey(r.lotteryID))
	}
	slog.Info("admin topic purged", "operator_id", operator.ID, "topic_id", row.ID, "title", row.Title,
		"author_id", row.UserID, "replies", counts.Replies, "comments", counts.Comments, "polls", counts.Polls,
		"lotteries", counts.Lotteries, "drawn_lotteries", counts.DrawnLotteries, "favorites", counts.Favorites,
		"refunded_lotteries", len(refunds))
	return &purgeTopicOutput{}, nil
}
