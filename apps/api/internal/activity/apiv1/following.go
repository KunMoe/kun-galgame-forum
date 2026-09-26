package apiv1

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/activity/repository"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const (
	siteKungal        = "kungal"
	followingFeedMark = "following"
	unseenCountLimit  = 100
	unmarkedLookback  = 7 * 24 * time.Hour
	followingListPath = "/me/following-activities"
	followingSummary  = followingListPath + "/summary"
	followingReadMark = followingListPath + "/read-marker"
	followingDown     = "SERVICE_UNAVAILABLE when the community follow graph, the account service or catalog is unreachable."
)

type Followees interface {
	IDs(ctx context.Context, userID int) ([]int, error)
}

type FollowingActivity struct {
	Object   string    `json:"object" enum:"following_activity" maxLength:"18" doc:"Type discriminant. Always following_activity."`
	Site     string    `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The NextMoe site it happened on. kungal is this forum and the only value sent today; other sites will join. An open vocabulary: show an unknown token as it is."`
	Activity *Activity `json:"activity" doc:"The activity, when site is kungal: the same object listActivities returns. An activity on another site will leave this null and carry a block of its own; none are sent yet, so skip an entry with neither."`
}

type FollowingActivitySummary struct {
	Object      string         `json:"object" enum:"following_activity_summary" maxLength:"26" doc:"Type discriminant. Always following_activity_summary."`
	LastSeenAt  *repr.DateTime `json:"last_seen_at" doc:"The caller's seen mark: activities at or before it count as seen. null until the caller first sets it."`
	UnseenCount int            `json:"unseen_count" minimum:"0" maximum:"100" doc:"Matching rows after last_seen_at counted in SQL up to 100: 100 means 100 or more. With no seen mark, the last seven days are counted. Can include rows the list leaves out, such as a banned author's or a work catalog does not show."`
}

type FollowingActivityReadMarker struct {
	Object string        `json:"object" enum:"following_activity_read_marker" maxLength:"30" doc:"Type discriminant. Always following_activity_read_marker."`
	SeenAt repr.DateTime `json:"seen_at" doc:"The seen mark as stored."`
}

type FollowingActivityReadMarkerWrite struct {
	SeenAt *repr.DateTime `json:"seen_at,omitempty" required:"false" doc:"Everything that occurred at or before this instant counts as seen. Absent means everything up to now. When present, send the occurred_at of the newest activity shown. The mark only moves forward, and a time in the future is stored as the server's current time."`
}

type FollowingFilters struct {
	Types              []ActivityType `query:"activity_types" required:"false" maxItems:"22" doc:"Only these activity types, comma-separated. Absent means every type."`
	TopicSections      string         `query:"topic_sections" enum:"normal,help,all" default:"normal" maxLength:"6" doc:"Which topic_creation activities to include: help is the resource and help sections (g-seeking, g-other, t-help), normal is every other section, all is both. Other kinds are not affected."`
	IncludeNSFW        bool           `query:"include_nsfw" default:"false" doc:"When true, NSFW activities and works are included. Default false."`
	IncludeUnresourced bool           `query:"include_galgames_without_resources" default:"false" doc:"When true, galgame_creation includes works that have no download resource yet. Default false."`
}

func (f *FollowingFilters) query(actors []int) repository.FeedQuery {
	if actors == nil {
		actors = []int{}
	}
	return repository.FeedQuery{
		Types: feedTypesOf(f.Types), TopicSections: f.TopicSections, IncludeNSFW: f.IncludeNSFW,
		IncludeUnresourced: f.IncludeUnresourced, ActorIDs: actors,
	}
}

type followingListInput struct {
	collect.Page
	FollowingFilters
}

type followingListOutput struct {
	Body repr.List[FollowingActivity]
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

func registerFollowing(api huma.API, s *Service) {
	tags := []string{"activities"}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listFollowingActivities",
		Method:      http.MethodGet,
		Path:        followingListPath,
		Summary:     "List the activity of the accounts the caller follows",
		Description: "The activities of every account the caller follows, newest first, with the filters of listActivities and its occurred_desc order. " +
			"Follows are NextMoe's, shared with the other NextMoe sites; a follow or unfollow made on this forum shows at once, one made elsewhere within a minute. " +
			"An activity whose actor is banned, or whose work catalog does not show under include_nsfw, is left out, so a page can be short; " +
			"only an absent next_cursor means the end. The cursor is bound to every filter, but not to who the caller follows.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR, LIMIT_TOO_LARGE, UNKNOWN_ENUM_VALUE or INVALID_PARAMETER.",
			503: followingDown,
		}),
	}), s.listFollowingActivities)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getFollowingActivitySummary",
		Method:      http.MethodGet,
		Path:        followingSummary,
		Summary:     "Count the followed accounts' activity the caller has not seen",
		Description: "The caller's seen mark and a SQL count, capped at 100, of matching rows by followed accounts after the mark (or in the last seven days when there is no mark), which can include rows listFollowingActivities leaves out.",
		Tags:        tags,
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE or INVALID_PARAMETER.",
			503: "SERVICE_UNAVAILABLE when the community follow graph is unreachable.",
		}),
	}), s.getFollowingActivitySummary)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "markFollowingActivitiesSeen",
		Method:      http.MethodPut,
		Path:        followingReadMark,
		Summary:     "Move the caller's seen mark on the followed accounts' activity",
		Description: "Stores the later of the current mark and seen_at, with a future seen_at taken as now. Absent seen_at means now. Replaying it, or sending an earlier time, changes nothing and is still 200.",
		Tags:        tags,
		Responses: problemResponses(map[int]string{
			422: "VALIDATION_FAILED when seen_at is present but not a real instant.",
		}),
	}), s.markFollowingActivitiesSeen)
}

func (s *Service) followeeIDs(ctx context.Context, userID int) ([]int, *problem.Problem) {
	if s.followees == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	ids, err := s.followees.IDs(ctx, userID)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return ids, nil
}

func (s *Service) listFollowingActivities(ctx context.Context, in *followingListInput) (*followingListOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	ids, prob := s.followeeIDs(ctx, v1.User(ctx).ID)
	if prob != nil {
		return nil, prob
	}
	q := in.query(ids)
	q.Limit = in.Limit
	fp := collect.Fingerprint(followingFeedMark, strings.Join(q.Types, ","), in.TopicSections,
		strconv.FormatBool(in.IncludeNSFW), strconv.FormatBool(in.IncludeUnresourced))
	items, next, prob := s.collectPage(ctx, feedPage{
		sort: "occurred_desc", fingerprint: fp, cursor: in.Cursor, limit: in.Limit, includeNSFW: in.IncludeNSFW,
		fetch: func(after *repository.FeedPos, _ *repository.TopicFeedPos) ([]repository.FeedRow, error) {
			if len(ids) == 0 {
				return nil, nil
			}
			q.After = after
			return s.repo.FeedPage(q)
		},
	})
	if prob != nil {
		return nil, prob
	}
	out := make([]FollowingActivity, len(items))
	for i := range items {
		out[i] = FollowingActivity{Object: "following_activity", Site: siteKungal, Activity: &items[i]}
	}
	return &followingListOutput{Body: repr.NewList(out, next)}, nil
}

func (s *Service) getFollowingActivitySummary(ctx context.Context, in *followingSummaryInput) (*followingSummaryOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	uid := v1.User(ctx).ID
	ids, prob := s.followeeIDs(ctx, uid)
	if prob != nil {
		return nil, prob
	}
	seen, err := s.repo.FollowingSeenAt(uid)
	if err != nil {
		return nil, problem.Internal(err)
	}
	count := 0
	if len(ids) > 0 {
		from := time.Now().Add(-unmarkedLookback)
		if seen != nil {
			// occurred_at goes out at second precision, so a client marking the
			// newest item it showed passes that item's truncated time.
			from = seen.Truncate(time.Second).Add(time.Second)
		}
		if count, err = s.repo.CountFeedSince(in.query(ids), from, unseenCountLimit); err != nil {
			return nil, problem.Internal(err)
		}
	}
	return &followingSummaryOutput{Body: FollowingActivitySummary{
		Object: "following_activity_summary", LastSeenAt: repr.TimestampPtr(seen), UnseenCount: count,
	}}, nil
}

func (s *Service) markFollowingActivitiesSeen(ctx context.Context, in *followingReadMarkerInput) (*followingReadMarkerOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	at := time.Now()
	if in.Body.SeenAt != nil {
		parsed, err := time.Parse(time.RFC3339, string(*in.Body.SeenAt))
		if err != nil {
			return nil, problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.",
				problem.AtPointer("/seen_at", problem.ReasonInvalidFormat, "must be a real instant in RFC 3339 UTC with second precision", nil))
		}
		at = parsed
	}
	seen, err := s.repo.MarkFollowingSeen(v1.User(ctx).ID, at)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &followingReadMarkerOutput{Body: FollowingActivityReadMarker{
		Object: "following_activity_read_marker", SeenAt: repr.Timestamp(seen),
	}}, nil
}
