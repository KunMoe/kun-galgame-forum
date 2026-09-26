package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const followSort = "followed"

type UserFollowState struct {
	Object       string         `json:"object" enum:"user_follow_state" maxLength:"17" doc:"Type discriminant. Always user_follow_state."`
	ID           repr.DecimalID `json:"id" doc:"The target user id, the same value as user_id."`
	UserID       repr.DecimalID `json:"user_id" doc:"The user this standing is about."`
	IsFollowing  bool           `json:"is_following" doc:"Whether the caller follows this user."`
	IsFollowedBy bool           `json:"is_followed_by" doc:"Whether this user follows the caller."`
}

type UserFollower struct {
	Object     string         `json:"object" enum:"user_follower" maxLength:"13" doc:"Type discriminant. Always user_follower."`
	Follower   repr.UserRef   `json:"follower" doc:"The following account. name is null when the account no longer exists; show a localized label."`
	FollowedAt *repr.DateTime `json:"followed_at" doc:"When the follow was created. null when the community service did not send a time."`
}

type UserFollowee struct {
	Object     string         `json:"object" enum:"user_followee" maxLength:"13" doc:"Type discriminant. Always user_followee."`
	Followee   repr.UserRef   `json:"followee" doc:"The followed account. name is null when the account no longer exists; show a localized label."`
	FollowedAt *repr.DateTime `json:"followed_at" doc:"When the follow was created. null when the community service did not send a time."`
}

type followUserInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id of the account to follow or unfollow."`
}

type followStateOutput struct {
	Body UserFollowState
}

type listUserFollowsInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	collect.Page
}

type listUserFollowersOutput struct {
	Body repr.List[UserFollower]
}

type listUserFollowingOutput struct {
	Body repr.List[UserFollowee]
}

func (s *Users) registerFollows(api huma.API) {
	tags := []string{"users"}
	userMissing := "NOT_FOUND when the account does not exist or is not renderable."
	upstreamDown := "SERVICE_UNAVAILABLE when the community service is unreachable or unconfigured."

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "followUser",
		Method:      http.MethodPut,
		Path:        "/me/following/{user_id}",
		Summary:     "Follow a user",
		Description: "The caller follows the named user. Following a user the caller already follows changes nothing. " +
			userMissing,
		Tags: tags,
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED or ACCOUNT_BANNED.",
			404: userMissing,
			422: "VALIDATION_FAILED when the caller follows themselves, or when the following limit is reached.",
			503: upstreamDown,
		}),
	}), s.followUser)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "unfollowUser",
		Method:      http.MethodDelete,
		Path:        "/me/following/{user_id}",
		Summary:     "Stop following a user",
		Description: "The caller stops following the named user. Unfollowing a user the caller does not follow changes nothing. " +
			userMissing,
		Tags: tags,
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED or ACCOUNT_BANNED.",
			404: userMissing,
			503: upstreamDown,
		}),
	}), s.unfollowUser)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getUserFollowState",
		Method:      http.MethodGet,
		Path:        "/me/following/{user_id}",
		Summary:     "Get the caller's standing toward a user",
		Description: "Whether the caller follows the named user, and whether that user follows the caller. " +
			userMissing,
		Tags: tags,
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED or ACCOUNT_BANNED.",
			404: userMissing,
			503: upstreamDown,
		}),
	}), s.getUserFollowState)

	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listUserFollowers",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/followers",
		Summary:     "List a user's followers",
		Description: "Accounts that follow this user, newest first, as a cursor page. " +
			"An account the forum cannot resolve is still listed, with a null name. " +
			"The last page omits next_cursor. " + userMissing,
		Tags: tags,
		Responses: problemResponses(map[int]string{
			503: upstreamDown,
		}),
	}), s.listUserFollowers)

	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listUserFollowing",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/following",
		Summary:     "List the accounts a user follows",
		Description: "Accounts this user follows, newest first, as a cursor page. " +
			"An account the forum cannot resolve is still listed, with a null name. " +
			"The last page omits next_cursor. " + userMissing,
		Tags: tags,
		Responses: problemResponses(map[int]string{
			503: upstreamDown,
		}),
	}), s.listUserFollowing)
}

func (s *Users) readyFollows() *problem.Problem {
	if s == nil || s.accounts == nil {
		return problem.Internal(errUnconfigured)
	}
	if s.community == nil || !s.community.Configured() {
		return problem.Unavailable(communityclient.ErrNotConfigured)
	}
	return nil
}

func (s *Users) followUser(ctx context.Context, in *followUserInput) (*followStateOutput, error) {
	return s.setFollowing(ctx, in, true)
}

func (s *Users) unfollowUser(ctx context.Context, in *followUserInput) (*followStateOutput, error) {
	return s.setFollowing(ctx, in, false)
}

func (s *Users) getUserFollowState(ctx context.Context, in *followUserInput) (*followStateOutput, error) {
	if p := s.readyFollows(); p != nil {
		return nil, p
	}
	viewer := v1.User(ctx)
	target, p := s.requireRenderableOwner(ctx, in.UserID)
	if p != nil {
		return nil, p
	}
	return s.followStateOf(ctx, viewer.ID, target)
}

func (s *Users) setFollowing(ctx context.Context, in *followUserInput, following bool) (*followStateOutput, error) {
	if p := s.readyFollows(); p != nil {
		return nil, p
	}
	viewer := v1.User(ctx)
	target, p := s.requireRenderableOwner(ctx, in.UserID)
	if p != nil {
		return nil, p
	}
	if following && viewer.ID == target {
		return nil, validationFailed(problem.AtParameter("user_id", problem.ReasonNotPermitted,
			"the caller cannot follow themselves", nil))
	}
	var err error
	if following {
		_, err = s.community.FollowUser(ctx, int64(viewer.ID), int64(target))
	} else {
		_, err = s.community.UnfollowUser(ctx, int64(viewer.ID), int64(target))
	}
	if err != nil {
		return nil, followUpstreamProblem(err)
	}
	return s.followStateOf(ctx, viewer.ID, target)
}

func (s *Users) followStateOf(ctx context.Context, viewerID, targetID int) (*followStateOutput, error) {
	page, err := s.community.FollowStates(ctx, int64(viewerID), []int64{int64(targetID)})
	if err != nil {
		return nil, followUpstreamProblem(err)
	}
	st := UserFollowState{
		Object: "user_follow_state",
		ID:     repr.ID(targetID),
		UserID: repr.ID(targetID),
	}
	if len(page.States) > 0 {
		st.IsFollowing = page.States[0].ViewerFollows
		st.IsFollowedBy = page.States[0].FollowsViewer
	}
	return &followStateOutput{Body: st}, nil
}

func (s *Users) listUserFollowers(ctx context.Context, in *listUserFollowsInput) (*listUserFollowersOutput, error) {
	rows, next, p := s.listFollowEdges(ctx, in, "followers")
	if p != nil {
		return nil, p
	}
	items := make([]UserFollower, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserFollower{Object: "user_follower", Follower: row.ref, FollowedAt: row.at})
	}
	return &listUserFollowersOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Users) listUserFollowing(ctx context.Context, in *listUserFollowsInput) (*listUserFollowingOutput, error) {
	rows, next, p := s.listFollowEdges(ctx, in, "following")
	if p != nil {
		return nil, p
	}
	items := make([]UserFollowee, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserFollowee{Object: "user_followee", Followee: row.ref, FollowedAt: row.at})
	}
	return &listUserFollowingOutput{Body: repr.NewList(items, next)}, nil
}

type followEdge struct {
	ref repr.UserRef
	at  *repr.DateTime
}

func (s *Users) listFollowEdges(ctx context.Context, in *listUserFollowsInput, relation string) ([]followEdge, *string, *problem.Problem) {
	if p := s.readyFollows(); p != nil {
		return nil, nil, p
	}
	owner, p := s.requireRenderableOwner(ctx, in.UserID)
	if p != nil {
		return nil, nil, p
	}
	fp := collect.Fingerprint(strconv.Itoa(owner), relation)
	keys, curErr := collect.DecodeCursor(in.Cursor, followSort, fp)
	if curErr != nil {
		return nil, nil, curErr
	}
	upstream := ""
	if keys != nil {
		if len(keys) != 1 || keys[0] == "" {
			return nil, nil, invalidCursor()
		}
		upstream = keys[0]
	}
	limit := in.Limit
	if limit < 1 {
		limit = collect.DefaultLimit
	}
	var page *communityclient.FollowListResponse
	var err error
	if relation == "followers" {
		page, err = s.community.ListFollowers(ctx, int64(owner), upstream, limit)
	} else {
		page, err = s.community.ListFollowing(ctx, int64(owner), upstream, limit)
	}
	if err != nil {
		return nil, nil, followUpstreamProblem(err)
	}
	ids := make([]int, 0, len(page.Users))
	for _, row := range page.Users {
		ids = append(ids, int(row.UserID))
	}
	users, err := s.accounts.Users(ctx, ids)
	if err != nil {
		return nil, nil, unavailable(err)
	}
	edges := make([]followEdge, 0, len(page.Users))
	for _, row := range page.Users {
		edges = append(edges, followEdge{ref: renderableUserRef(s.cdn, users, int(row.UserID)), at: parseFollowedAt(row.FollowedAt)})
	}
	var next *string
	if page.NextCursor != "" {
		cur := collect.EncodeCursor(followSort, fp, page.NextCursor)
		next = &cur
	}
	return edges, next, nil
}

func renderableUserRef(cdn string, users map[int]userclient.User, id int) repr.UserRef {
	if u, ok := users[id]; ok && userclient.IsRenderable(u) {
		return repr.NewUserRef(cdn, u)
	}
	return repr.DeletedUserRef(id)
}

func parseFollowedAt(raw *string) *repr.DateTime {
	if raw == nil || *raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, *raw)
	}
	if err != nil {
		return nil
	}
	ts := repr.Timestamp(t)
	return &ts
}

func (s *Users) followCounts(ctx context.Context, userID int) (followers, following *int) {
	if s.community == nil || !s.community.Configured() {
		return nil, nil
	}
	page, err := s.community.FollowStates(ctx, 0, []int64{int64(userID)})
	if err != nil {
		slog.Warn("user follow counts unavailable", "user_id", userID, "err", err)
		return nil, nil
	}
	if len(page.States) == 0 {
		return nil, nil
	}
	f := int(page.States[0].FollowersCount)
	g := int(page.States[0].FollowingCount)
	return &f, &g
}

func followUpstreamProblem(err error) *problem.Problem {
	switch {
	case errors.Is(err, communityclient.ErrRateLimited):
		return problem.New(problem.CodeRateLimited, "The community service refused the write under its new-account rate limit.")
	case errors.Is(err, communityclient.ErrNotConfigured), errors.Is(err, communityclient.ErrForbidden):
		return problem.Unavailable(err)
	}
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) {
		return problem.Unavailable(err)
	}
	switch {
	case apiErr.Status == 404:
		return notFound()
	case apiErr.Status == 422:
		maxFollows := float64(5000)
		return problem.New(problem.CodeValidationFailed, "The following limit is reached.",
			problem.AtParameter("user_id", problem.ReasonOutOfRange, "at most 5000 users may be followed",
				&problem.FieldParams{Maximum: &maxFollows}))
	case apiErr.Status >= 500:
		return problem.Unavailable(err)
	}
	slog.Error("community upstream refused a follow request this service should not have sent",
		"status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Msg)
	return problem.Internal(err)
}
