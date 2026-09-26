package apiv1

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const (
	siteKungal         = "kungal"
	followingSort      = "following"
	followingListPath  = "/me/following-activities"
	followingSummary   = followingListPath + "/summary"
	followingReadMark  = followingListPath + "/read-marker"
	activityGroupItems = "/activity-groups/{group_id}/items"
	followingDown      = "SERVICE_UNAVAILABLE when the community service or the account service is unreachable."
)

type ActivityVerb string

func (ActivityVerb) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum("publish", "reply", "comment", "rate", "like", "edit")
	s.Description = "What the performer did: publish, reply, comment, rate, like or edit."
	return s
}

type ActivitySite string

func (ActivitySite) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("activity_site", 64)
	s.Pattern = `^[a-z0-9][a-z0-9_-]*$`
	s.Description = "A NextMoe site token. kungal is this forum. An open vocabulary: show an unknown token as it is."
	return s
}

type FollowingActivityGroup struct {
	Object       string                  `json:"object" enum:"following_activity_group" maxLength:"24" doc:"Type discriminant. Always following_activity_group."`
	ID           repr.DecimalID          `json:"id" doc:"Community group id."`
	Site         string                  `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The NextMoe site the group happened on. kungal is this forum. An open vocabulary: show an unknown token as it is."`
	Actor        repr.UserRef            `json:"actor" doc:"Who did it. name is null when the account no longer exists; show a localized label."`
	Verb         ActivityVerb            `json:"verb" doc:"What the performer did."`
	ObjectKind   string                  `json:"object_kind" pattern:"^[a-z0-9_]{1,32}$" maxLength:"32" doc:"The site's type name for the object, such as topic or patch. An open vocabulary."`
	ObjectLabel  string                  `json:"object_label" maxLength:"16" doc:"Display name of object_kind. Show as it is. Free text; never use it as a decision input."`
	CalendarDate repr.CalendarDate       `json:"calendar_date" doc:"The Asia/Shanghai calendar day the group belongs to."`
	ItemCount    int                     `json:"item_count" minimum:"0" doc:"Live items in the group. At least 1 from community; 0 is reserved for the integer floor every *_count uses."`
	LatestAt     repr.DateTime           `json:"latest_at" doc:"When the newest item in the group occurred."`
	Items        []FollowingActivityItem `json:"items" maxItems:"3" doc:"Newest items in the group, at most 3. Empty array, never null."`
}

type FollowingActivityItem struct {
	Object        string          `json:"object" enum:"following_activity_item" maxLength:"23" doc:"Type discriminant. Always following_activity_item."`
	ID            repr.DecimalID  `json:"id" doc:"Community item id."`
	Site          string          `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The NextMoe site the item happened on. kungal is this forum. An open vocabulary: show an unknown token as it is."`
	Key           string          `json:"key" maxLength:"128" doc:"Community's idempotency key for the item. Free text; never use it as a decision input."`
	Verb          ActivityVerb    `json:"verb" doc:"What the performer did."`
	ObjectKind    string          `json:"object_kind" pattern:"^[a-z0-9_]{1,32}$" maxLength:"32" doc:"The site's type name for the object. An open vocabulary."`
	ObjectLabel   string          `json:"object_label" maxLength:"16" doc:"Display name of object_kind. Show as it is. Free text; never use it as a decision input."`
	Title         string          `json:"title" maxLength:"200" doc:"Plain-text title. Free text; never use it as a decision input."`
	Excerpt       string          `json:"excerpt" maxLength:"300" doc:"Plain-text excerpt, may be empty. Free text; never use it as a decision input."`
	URL           string          `json:"url" format:"uri" maxLength:"2048" doc:"Absolute https URL of the item."`
	InSitePath    *string         `json:"in_site_path" pattern:"^/" maxLength:"512" doc:"Path and query of url when site is kungal. null for other sites."`
	Cover         *repr.Image     `json:"cover" doc:"Cover from community's cover_image_hash. null when none."`
	RelatedWorkID *repr.DecimalID `json:"related_work_id" doc:"Catalog work id when the item names one. null when none."`
	IsNSFW        bool            `json:"is_nsfw" doc:"Whether the item is not sfw."`
	OccurredAt    repr.DateTime   `json:"occurred_at" doc:"When it happened."`
}

type FollowingActivitySummary struct {
	Object      string         `json:"object" enum:"following_activity_summary" maxLength:"26" doc:"Type discriminant. Always following_activity_summary."`
	LastSeenAt  *repr.DateTime `json:"last_seen_at" doc:"The caller's seen mark. null until the caller first sets it."`
	UnseenCount int            `json:"unseen_count" minimum:"0" maximum:"100" doc:"Followed groups whose latest_at is after the later of last_seen_at and when each follow began, counted up to 100: 100 means 100 or more. Uses the same include_nsfw, verbs and sites as listFollowingActivities."`
}

type FollowingActivityReadMarker struct {
	Object string        `json:"object" enum:"following_activity_read_marker" maxLength:"30" doc:"Type discriminant. Always following_activity_read_marker."`
	SeenAt repr.DateTime `json:"seen_at" doc:"The seen mark as stored by community."`
}

type FollowingActivityReadMarkerWrite struct {
	SeenAt *repr.DateTime `json:"seen_at,omitempty" required:"false" doc:"Mark everything at or before this instant as seen. Absent means now. The mark only moves forward and is never stored past now. Omit this field (send {}) when opening the list; do not send the time of the first group rendered."`
}

type FollowingFilters struct {
	IncludeNSFW bool           `query:"include_nsfw" default:"false" doc:"When true, NSFW groups are included. Default false."`
	Verbs       []ActivityVerb `query:"verbs" required:"false" maxItems:"6" doc:"Only these verbs, comma-separated. Absent means every verb."`
	Sites       []ActivitySite `query:"sites" required:"false" doc:"Only these NextMoe sites, comma-separated. Absent means every site."`
}

func (f FollowingFilters) contentLimit() string {
	if f.IncludeNSFW {
		return "all"
	}
	return "sfw"
}

func (f FollowingFilters) verbList() []string {
	out := make([]string, len(f.Verbs))
	for i, v := range f.Verbs {
		out[i] = string(v)
	}
	return out
}

func (f FollowingFilters) siteList() []string {
	out := make([]string, len(f.Sites))
	for i, s := range f.Sites {
		out[i] = string(s)
	}
	return out
}

func (f FollowingFilters) fingerprint(extra ...string) string {
	parts := append([]string{f.contentLimit(), strings.Join(f.verbList(), ","), strings.Join(f.siteList(), ",")}, extra...)
	return collect.Fingerprint(parts...)
}

type FollowingPage struct {
	Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit  int    `query:"limit" minimum:"1" maximum:"50" default:"20" doc:"Page size. 1–50, default 20. Values above 50 are rejected, not clamped."`
}

type followingListInput struct {
	FollowingPage
	FollowingFilters
}

type followingListOutput struct {
	Body repr.List[FollowingActivityGroup]
}

type followingSummaryInput struct {
	FollowingFilters
}

type followingSummaryOutput struct {
	Body FollowingActivitySummary
}

type followingReadMarkerInput struct {
	Body FollowingActivityReadMarkerWrite
}

type followingReadMarkerOutput struct {
	Body FollowingActivityReadMarker
}

type groupItemsInput struct {
	GroupID     string `path:"group_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Community group id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW items are included. Default false."`
	FollowingPage
}

type groupItemsOutput struct {
	Body repr.List[FollowingActivityItem]
}

func registerFollowing(api huma.API, s *Service) {
	tags := []string{"activities"}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listFollowingActivities",
		Method:      http.MethodGet,
		Path:        followingListPath,
		Summary:     "List grouped activity of the accounts the caller follows",
		Description: "Groups of activity by accounts the caller follows, across NextMoe sites, newest group first. " +
			"A group is one author's items of one verb and object_kind on one site on one Asia/Shanghai calendar day. " +
			"A group whose actor is not renderable is dropped, so a page can be shorter than limit; " +
			"next_cursor still comes from community, and only an absent next_cursor means the end. " +
			"The cursor is bound to include_nsfw, verbs and sites.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR, LIMIT_TOO_LARGE, UNKNOWN_ENUM_VALUE or INVALID_PARAMETER.",
			503: followingDown,
		}),
	}), s.listFollowingActivities)

	huma.Register(api, v1.Public(huma.Operation{
		OperationID: "listActivityGroupItems",
		Method:      http.MethodGet,
		Path:        activityGroupItems,
		Summary:     "List every live item of an activity group",
		Description: "Every live item of one activity group, newest first. " +
			"Unknown groups, and groups whose actor is not renderable, are NOT_FOUND. " +
			"A page can be shorter than limit. The cursor is bound to include_nsfw.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR, LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			503: followingDown,
		}),
	}), s.listActivityGroupItems)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getFollowingActivitySummary",
		Method:      http.MethodGet,
		Path:        followingSummary,
		Summary:     "Count the followed accounts' activity the caller has not seen",
		Description: "The caller's seen mark and a count, capped at 100, of followed groups whose latest_at is after the later of the mark and when each follow began. " +
			"100 means 100 or more. Following someone never lights up their history. " +
			"last_seen_at is null until the caller first sets the mark. The same include_nsfw, verbs and sites filters as listFollowingActivities apply.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE or INVALID_PARAMETER.",
			503: followingDown,
		}),
	}), s.getFollowingActivitySummary)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "markFollowingActivitiesSeen",
		Method:      http.MethodPut,
		Path:        followingReadMark,
		Summary:     "Move the caller's seen mark on the followed accounts' activity",
		Description: "The mark is account-wide across NextMoe sites, moves forward only, and is never stored past now. " +
			"Omit seen_at (send {}) to mark everything up to now, which is what opening the list should do; " +
			"never send the time of the first group the client rendered, or a hidden group above it would stay unseen.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			422: "VALIDATION_FAILED when seen_at is present but not a real instant.",
			503: followingDown,
		}),
	}), s.markFollowingActivitiesSeen)
}

func (s *Service) readyFollowing() *problem.Problem {
	if s == nil || s.users == nil || s.convert == nil {
		return problem.Internal(errUnconfigured)
	}
	if s.community == nil || !s.community.Configured() {
		return problem.Unavailable(communityclient.ErrNotConfigured)
	}
	return nil
}

func (s *Service) listFollowingActivities(ctx context.Context, in *followingListInput) (*followingListOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	fp := in.fingerprint()
	upstream, prob := unwrapCursor(in.Cursor, fp)
	if prob != nil {
		return nil, prob
	}
	page, err := s.community.ListFollowingActivities(ctx, int64(v1.User(ctx).ID), upstream, in.Limit, in.contentLimit(), in.siteList(), in.verbList())
	if err != nil {
		return nil, communityReadProblem(err, true)
	}
	actorIDs := make([]int, 0, len(page.Groups))
	for _, g := range page.Groups {
		actorIDs = append(actorIDs, int(g.ActorID))
	}
	users, prob := s.lookupUsers(ctx, actorIDs)
	if prob != nil {
		return nil, prob
	}
	out := make([]FollowingActivityGroup, 0, len(page.Groups))
	for _, g := range page.Groups {
		ref, ok := s.performer(users, int(g.ActorID))
		if !ok {
			continue
		}
		items := make([]FollowingActivityItem, 0, min(len(g.Items), 3))
		for i, it := range g.Items {
			if i == 3 {
				break
			}
			items = append(items, s.followingItem(it))
		}
		out = append(out, FollowingActivityGroup{
			Object:       "following_activity_group",
			ID:           decimalID(g.ID),
			Site:         g.Site,
			Actor:        ref,
			Verb:         ActivityVerb(g.Verb),
			ObjectKind:   g.ObjectKind,
			ObjectLabel:  g.ObjectLabel,
			CalendarDate: repr.CalendarDate(g.Day),
			ItemCount:    g.ItemCount,
			LatestAt:     parseDateTime(g.LatestAt),
			Items:        items,
		})
	}
	return &followingListOutput{Body: repr.NewList(out, wrapCursor(fp, page.NextCursor))}, nil
}

func (s *Service) listActivityGroupItems(ctx context.Context, in *groupItemsInput) (*groupItemsOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	gid, _ := strconv.ParseInt(in.GroupID, 10, 64)
	contentLimit := "sfw"
	if in.IncludeNSFW {
		contentLimit = "all"
	}
	fp := collect.Fingerprint(in.GroupID, contentLimit)
	upstream, prob := unwrapCursor(in.Cursor, fp)
	if prob != nil {
		return nil, prob
	}
	page, err := s.community.ListActivityGroupItems(ctx, gid, upstream, in.Limit, contentLimit)
	if err != nil {
		return nil, communityReadProblem(err, true)
	}
	actorIDs := make([]int, 0, len(page.Items))
	for _, it := range page.Items {
		actorIDs = append(actorIDs, int(it.ActorID))
	}
	users, prob := s.lookupUsers(ctx, actorIDs)
	if prob != nil {
		return nil, prob
	}
	for _, it := range page.Items {
		if _, ok := s.performer(users, int(it.ActorID)); !ok {
			return nil, followingNotFound()
		}
	}
	out := make([]FollowingActivityItem, 0, len(page.Items))
	for _, it := range page.Items {
		out = append(out, s.followingItem(it))
	}
	return &groupItemsOutput{Body: repr.NewList(out, wrapCursor(fp, page.NextCursor))}, nil
}

func (s *Service) getFollowingActivitySummary(ctx context.Context, in *followingSummaryInput) (*followingSummaryOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	page, err := s.community.GetFollowingActivitiesUnseen(ctx, int64(v1.User(ctx).ID), in.contentLimit(), in.siteList(), in.verbList())
	if err != nil {
		return nil, communityReadProblem(err, false)
	}
	return &followingSummaryOutput{Body: FollowingActivitySummary{
		Object: "following_activity_summary", LastSeenAt: parseDateTimePtr(page.SeenAt), UnseenCount: page.UnseenCount,
	}}, nil
}

func (s *Service) markFollowingActivitiesSeen(ctx context.Context, in *followingReadMarkerInput) (*followingReadMarkerOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	var at *string
	if in.Body.SeenAt != nil {
		s := string(*in.Body.SeenAt)
		at = &s
	}
	page, err := s.community.MarkFollowingActivitiesSeen(ctx, int64(v1.User(ctx).ID), at)
	if err != nil {
		return nil, communityReadProblem(err, false)
	}
	return &followingReadMarkerOutput{Body: FollowingActivityReadMarker{
		Object: "following_activity_read_marker", SeenAt: parseDateTime(page.SeenAt),
	}}, nil
}

func (s *Service) followingItem(it communityclient.ActivityItemView) FollowingActivityItem {
	return FollowingActivityItem{
		Object:        "following_activity_item",
		ID:            decimalID(it.ID),
		Site:          it.Site,
		Key:           it.Key,
		Verb:          ActivityVerb(it.Verb),
		ObjectKind:    it.ObjectKind,
		ObjectLabel:   it.ObjectLabel,
		Title:         it.Title,
		Excerpt:       it.Excerpt,
		URL:           it.URL,
		InSitePath:    inSitePath(it.Site, it.URL),
		Cover:         s.coverImage(it.CoverImageHash),
		RelatedWorkID: workIDPtr(it.WorkID),
		IsNSFW:        it.ContentLimit != "sfw",
		OccurredAt:    parseDateTime(it.OccurredAt),
	}
}

func (s *Service) coverImage(hash string) *repr.Image {
	if hash == "" {
		return nil
	}
	var meta *imageclient.ImageMeta
	if s.convert.Images != nil {
		if m, ok := s.convert.Images([]string{hash})[hash]; ok {
			cp := m
			meta = &cp
		}
	}
	return repr.NewImage(s.cdn, hash, meta)
}

func (s *Service) performer(users map[int]userclient.User, id int) (repr.UserRef, bool) {
	u, ok := users[id]
	if !ok {
		return repr.DeletedUserRef(id), true
	}
	if !userclient.IsRenderable(u) {
		return repr.UserRef{}, false
	}
	return repr.NewUserRef(s.cdn, u), true
}

func unwrapCursor(cur, fp string) (string, *problem.Problem) {
	keys, prob := collect.DecodeCursor(cur, followingSort, fp)
	if prob != nil {
		return "", prob
	}
	if keys == nil {
		return "", nil
	}
	if len(keys) != 1 {
		return "", followingInvalidCursor()
	}
	return keys[0], nil
}

func wrapCursor(fp, upstream string) *string {
	if upstream == "" {
		return nil
	}
	c := collect.EncodeCursor(followingSort, fp, upstream)
	return &c
}

func communityReadProblem(err error, cursor400 bool) *problem.Problem {
	switch {
	case errors.Is(err, communityclient.ErrNotConfigured), errors.Is(err, communityclient.ErrForbidden):
		return problem.Unavailable(err)
	}
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) {
		return problem.Unavailable(err)
	}
	switch {
	case apiErr.Status == http.StatusBadRequest && cursor400:
		return followingInvalidCursor()
	case apiErr.Status == http.StatusNotFound:
		return followingNotFound()
	case apiErr.Status >= 500:
		return problem.Unavailable(err)
	}
	return problem.Unavailable(err)
}

func followingInvalidCursor() *problem.Problem {
	return problem.New(problem.CodeInvalidCursor, "The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil))
}

func followingNotFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func decimalID(id int64) repr.DecimalID {
	return repr.DecimalID(strconv.FormatInt(id, 10))
}

func workIDPtr(id *int64) *repr.DecimalID {
	if id == nil || *id <= 0 {
		return nil
	}
	d := decimalID(*id)
	return &d
}

func inSitePath(site, rawURL string) *string {
	if site != siteKungal {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	p := u.RequestURI()
	return &p
}

func parseDateTime(raw string) repr.DateTime {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return repr.DateTime(raw)
	}
	return repr.Timestamp(t)
}

func parseDateTimePtr(raw *string) *repr.DateTime {
	if raw == nil || *raw == "" {
		return nil
	}
	t := parseDateTime(*raw)
	return &t
}
