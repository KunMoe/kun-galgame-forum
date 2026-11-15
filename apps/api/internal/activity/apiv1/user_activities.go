package apiv1

import (
	"context"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

type UserActivityGroup struct {
	Object       string                  `json:"object" enum:"user_activity_group" maxLength:"19" doc:"Type discriminant. Always user_activity_group."`
	ID           repr.DecimalID          `json:"id" doc:"Community group id. Pass it to listActivityGroupItems for every item."`
	Site         string                  `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The NextMoe site the group happened on. kungal is this forum. An open vocabulary: show an unknown token as it is."`
	Verb         ActivityVerb            `json:"verb" doc:"What the user did."`
	ObjectKind   string                  `json:"object_kind" pattern:"^[a-z0-9_]{1,32}$" maxLength:"32" doc:"The site's type name for the object, such as topic or patch. An open vocabulary."`
	ObjectLabel  string                  `json:"object_label" maxLength:"16" doc:"Display name of object_kind. Show as it is. Free text; never use it as a decision input."`
	CalendarDate repr.CalendarDate       `json:"calendar_date" doc:"The Asia/Shanghai calendar day the group belongs to."`
	ItemCount    int                     `json:"item_count" minimum:"0" doc:"Live items in the group. At least 1 from community; 0 is reserved for the integer floor every *_count uses."`
	LatestAt     repr.DateTime           `json:"latest_at" doc:"When the newest item in the group occurred."`
	Items        []FollowingActivityItem `json:"items" maxItems:"3" doc:"Newest items in the group, at most 3. Empty array, never null."`
}

type userActivitiesInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	FollowingPage
	FollowingFilters
}

type userActivitiesOutput struct {
	Body repr.List[UserActivityGroup]
}

func registerUserActivities(api huma.API, s *Service) {
	huma.Register(api, v1.ProfileGated(v1.Optional(huma.Operation{
		OperationID: "listUserActivities",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/activities",
		Summary:     "List a user's own activity across NextMoe sites",
		Description: "Groups of one user's activity across NextMoe sites, newest group first. " +
			"A group is the user's items of one verb and object_kind on one site on one Asia/Shanghai calendar day. " +
			"A user who hides their activity has none listed, except to themselves. " +
			"A page can be shorter than limit; only an absent next_cursor means the end. " +
			"The cursor is bound to include_nsfw, verbs and sites. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"activities"},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR, LIMIT_TOO_LARGE, UNKNOWN_ENUM_VALUE or INVALID_PARAMETER.",
			404: "NOT_FOUND when the account does not exist or is not renderable.",
			503: followingDown,
		}),
	}), "user_id"), s.listUserActivities)
}

func (s *Service) listUserActivities(ctx context.Context, in *userActivitiesInput) (*userActivitiesOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	ownerID, _ := strconv.Atoi(in.UserID)
	users, prob := s.lookupUsers(ctx, []int{ownerID})
	if prob != nil {
		return nil, prob
	}
	if u, ok := users[ownerID]; !ok || !userclient.IsRenderable(u) {
		return nil, followingNotFound()
	}
	fp := in.fingerprint(in.UserID)
	upstream, prob := unwrapCursor(in.Cursor, fp)
	if prob != nil {
		return nil, prob
	}
	var viewer int64
	if u := v1.User(ctx); u != nil {
		viewer = int64(u.ID)
	}
	page, err := s.community.ListUserActivities(ctx, int64(ownerID), viewer, upstream, in.Limit, in.contentLimit(), in.siteList(), in.upstreamVerbs())
	if err != nil {
		return nil, communityReadProblem(err, true)
	}
	page.Groups = knownGroups(page.Groups)
	var shown []communityclient.ActivityItemView
	for _, g := range page.Groups {
		shown = append(shown, g.Items[:min(len(g.Items), 3)]...)
	}
	extras, prob := s.itemExtras(ctx, shown, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	out := make([]UserActivityGroup, 0, len(page.Groups))
	for _, g := range page.Groups {
		items := make([]FollowingActivityItem, 0, min(len(g.Items), 3))
		for _, it := range g.Items[:min(len(g.Items), 3)] {
			items = append(items, s.followingItem(it, extras))
		}
		out = append(out, UserActivityGroup{
			Object:       "user_activity_group",
			ID:           decimalID(g.ID),
			Site:         g.Site,
			Verb:         ActivityVerb(g.Verb),
			ObjectKind:   g.ObjectKind,
			ObjectLabel:  g.ObjectLabel,
			CalendarDate: repr.CalendarDate(g.Day),
			ItemCount:    g.ItemCount,
			LatestAt:     parseDateTime(g.LatestAt),
			Items:        items,
		})
	}
	return &userActivitiesOutput{Body: repr.NewList(out, wrapCursor(fp, page.NextCursor))}, nil
}
