package apiv1

import (
	"context"
	"slices"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

// The account service's /users/search has no offset and stops at 50.
const userSearchCap = 50

var badgeRoles = []string{"creator", "moderator", "admin", "ren"}

type usersOutput struct {
	Body repr.PageList[UserSearchHit]
}

func (s *Service) searchUsers(ctx context.Context, in *usersInput) (*usersOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	q := strings.TrimSpace(in.Q)
	if _, prob := keywordsOf(q); prob != nil {
		return nil, prob
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	found, err := s.users.SearchUsers(ctx, q, userSearchCap)
	if userclient.IsInvalidParam(err) {
		return nil, problem.New(problem.CodeInvalidParameter, "The account service refused the name query.",
			problem.AtParameter("q", problem.ReasonNotAllowedValue, "the account service refused this name query", nil))
	}
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	relation := "eq"
	if len(found) >= userSearchCap {
		relation = "gte"
	}
	active := found[:0:0]
	for _, u := range found {
		if u.Status == 0 {
			active = append(active, u)
		}
	}
	start := min(in.Offset(), len(active))
	page := active[start:min(start+in.Limit, len(active))]

	ids := make([]int, len(page))
	for i, u := range page {
		ids[i] = u.ID
	}
	topics, replies := s.repo.CountUserPosts(ids, authenticated(ctx))

	items := make([]UserSearchHit, 0, len(page))
	for _, u := range page {
		ref := repr.NewUserRef(s.cdn, u)
		bio := u.Bio
		roles := make([]userapiv1.UserRole, 0, len(badgeRoles))
		held := role.Union(u.Roles, u.SiteRoles)
		for _, r := range badgeRoles {
			if slices.Contains(held, r) {
				roles = append(roles, userapiv1.UserRole(r))
			}
		}
		items = append(items, UserSearchHit{
			Object: "user", ID: ref.ID, Name: ref.Name, Avatar: ref.Avatar, Bio: &bio, Roles: roles,
			RegisteredAt: registeredAt(u.CreatedAt),
			TopicCount:   topics[u.ID], ReplyCount: replies[u.ID],
		})
	}
	return &usersOutput{Body: repr.NewPageList(items, len(active), relation)}, nil
}

func registeredAt(raw string) *repr.DateTime {
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil || at.IsZero() {
		return nil
	}
	return repr.TimestampPtr(&at)
}
