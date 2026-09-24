package apiv1

import (
	"context"
	"errors"
	"math"
	"net/http"

	adminService "kun-galgame-api/internal/admin/service"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

type UserContent struct {
	Object               string         `json:"object" enum:"user_content" maxLength:"12" doc:"Type discriminant. Always user_content."`
	ID                   repr.DecimalID `json:"id" doc:"The user's id. JSON string of a decimal integer."`
	IsProtected          bool           `json:"is_protected" doc:"Whether the user holds the moderation capability, by the account service's current record. purgeUserContent refuses such a user with USER_PROTECTED."`
	IsAccountActive      bool           `json:"is_account_active" doc:"Whether the account service reports the account as usable: it exists and is neither banned nor deregistered. An active account can keep posting after a purge, so it should be banned or deregistered first."`
	TopicCount           int            `json:"topic_count" minimum:"0" doc:"Topics the user wrote."`
	ReplyCount           int            `json:"reply_count" minimum:"0" doc:"Replies the user wrote."`
	TopicCommentCount    int            `json:"topic_comment_count" minimum:"0" doc:"Topic comments the user wrote."`
	RatingCount          int            `json:"rating_count" minimum:"0" doc:"Galgame ratings the user wrote."`
	ResourceCount        int            `json:"resource_count" minimum:"0" doc:"Galgame resources the user published."`
	WebsiteCount         int            `json:"website_count" minimum:"0" doc:"Websites the user listed. A purge hands them to the directory's default owner instead of deleting them."`
	ToolsetCount         int            `json:"toolset_count" minimum:"0" doc:"Toolsets the user created."`
	ToolsetResourceCount int            `json:"toolset_resource_count" minimum:"0" doc:"Toolset resources the user published."`
	PollCount            int            `json:"poll_count" minimum:"0" doc:"Polls the user created."`
	LotteryCount         int            `json:"lottery_count" minimum:"0" doc:"Lotteries the user created."`
	DraftCount           int            `json:"draft_count" minimum:"0" doc:"Topic drafts the user saved."`
	QuizCount            int            `json:"quiz_count" minimum:"0" doc:"Quizzes the user wrote."`
	CollectionCount      int            `json:"collection_count" minimum:"0" doc:"Collections the user owns."`
	TodoCount            int            `json:"todo_count" minimum:"0" doc:"Todo board entries the user filed."`
	ChatMessageCount     int            `json:"chat_message_count" minimum:"0" doc:"Private messages the user sent."`
	MessageCount         int            `json:"message_count" minimum:"0" doc:"Notifications the user sent or received."`
	InteractionCount     int            `json:"interaction_count" minimum:"0" doc:"The user's likes, favorites, votes, reactions, answers and other interaction rows."`
	CommunityPostCount   *int           `json:"community_post_count" minimum:"0" doc:"The user's visible posts on the community comment walls. null when the community service is unavailable."`
	TotalCount           int            `json:"total_count" minimum:"0" doc:"The sum of the local counts above, community_post_count excluded. Rows of other users that go with the user's topics are not counted."`
}

type userContentInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
}

type userContentOutput struct {
	Body UserContent
}

type purgeUserContentOutput struct{}

func (s *Users) registerAdminPurge(api huma.API) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getUserContent",
		Method:      http.MethodGet,
		Path:        "/admin/user-contents/{user_id}",
		Summary:     "Get what a purge would delete for a user",
		Description: "Counts the user's content on this site and reports whether the user may be purged and whether the account is still usable. " +
			"Any user id that fits the path answers, with zero counts when the user has nothing here. " +
			"It needs the user.purge_content permission, which a Bearer request never carries.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED when the caller lacks user.purge_content.",
			404: "NOT_FOUND when the id is larger than any user id can be.",
			503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
		}),
	}), s.getUserContent)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "purgeUserContent",
		Method:        http.MethodDelete,
		Path:          "/admin/user-contents/{user_id}",
		Summary:       "Purge a user's content",
		DefaultStatus: http.StatusNoContent,
		Description: "Deletes everything the user has on this site, with what cascades from it, hands their listed websites to the directory's default owner, " +
			"then purges their catalog collections and community posts. The account itself is untouched: ban or deregister it at the account service first, " +
			"or it can keep posting. Every local row the purge deletes or changes is archived for 30 days so a developer can undo it; the catalog and community parts cannot be undone. " +
			"Purging again is safe and is how a purge that failed after its local part is finished. " +
			"It needs the user.purge_content permission, which a Bearer request never carries.",
		Tags:        []string{"users"},
		Middlewares: huma.Middlewares{withUpstream},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED when the caller lacks user.purge_content. USER_PROTECTED when the user holds the moderation capability.",
			404: "NOT_FOUND when the id is larger than any user id can be.",
			409: "LOTTERY_DRAWN when one of the user's lotteries is being drawn; nothing was deleted, retry once the draw ends.",
			503: "SERVICE_UNAVAILABLE when the account service, the catalog or the community service cannot be reached. If it happens after the local part, the local rows are already purged and retrying finishes the rest.",
		}),
	}), s.purgeUserContent)
}

func (s *Users) purgeTarget(ctx context.Context, raw string) (int, *problem.Problem) {
	if s == nil || s.purge == nil {
		return 0, problem.Internal(errUnconfigured)
	}
	if !v1.User(ctx).Can(perm.UserPurgeContent) {
		return 0, permissionRequired()
	}
	id, ok := repr.ParseID(repr.DecimalID(raw))
	if !ok || id > math.MaxInt32 {
		return 0, notFound()
	}
	return id, nil
}

func (s *Users) getUserContent(ctx context.Context, in *userContentInput) (*userContentOutput, error) {
	id, prob := s.purgeTarget(ctx, in.UserID)
	if prob != nil {
		return nil, prob
	}
	p, err := s.purge.Preview(ctx, id)
	if err != nil {
		return nil, mapPurgeError(err)
	}
	c := p.Counts
	out := UserContent{
		Object:               "user_content",
		ID:                   repr.ID(id),
		IsProtected:          p.Protected,
		IsAccountActive:      p.AccountActive,
		TopicCount:           int(c.Topics),
		ReplyCount:           int(c.Replies),
		TopicCommentCount:    int(c.TopicComments),
		RatingCount:          int(c.Ratings),
		ResourceCount:        int(c.Resources),
		WebsiteCount:         int(c.Websites),
		ToolsetCount:         int(c.Toolsets),
		ToolsetResourceCount: int(c.ToolsetResources),
		PollCount:            int(c.Polls),
		LotteryCount:         int(c.Lotteries),
		DraftCount:           int(c.Drafts),
		QuizCount:            int(c.Quizzes),
		CollectionCount:      int(c.Collections),
		TodoCount:            int(c.Todos),
		ChatMessageCount:     int(c.ChatMessages),
		MessageCount:         int(c.Messages),
		InteractionCount:     int(c.Interactions),
		TotalCount:           int(c.Total),
	}
	if p.CommunityPosts != nil {
		n := int(*p.CommunityPosts)
		out.CommunityPostCount = &n
	}
	return &userContentOutput{Body: out}, nil
}

func (s *Users) purgeUserContent(ctx context.Context, in *userContentInput) (*purgeUserContentOutput, error) {
	id, prob := s.purgeTarget(ctx, in.UserID)
	if prob != nil {
		return nil, prob
	}
	if err := s.purge.Purge(ctx, v1.User(ctx).ID, id, accessToken(ctx)); err != nil {
		return nil, mapPurgeError(err)
	}
	return &purgeUserContentOutput{}, nil
}

func mapPurgeError(err error) *problem.Problem {
	var unavailable *adminService.UnavailableError
	switch {
	case errors.Is(err, adminService.ErrPurgeTargetProtected):
		return problem.New(problem.CodeUserProtected, "The user holds the moderation capability; their content is never purged.")
	case errors.Is(err, adminService.ErrPurgeLotteryDrawing):
		return problem.New(problem.CodeLotteryDrawn, "One of the user's lotteries is being drawn; nothing was deleted.")
	case errors.As(err, &unavailable):
		return problem.Unavailable(err)
	}
	return problem.Internal(err)
}
