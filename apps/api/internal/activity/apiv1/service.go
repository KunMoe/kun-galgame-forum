package apiv1

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"kun-galgame-api/internal/activity/repository"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/galgame/client"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/sync/singleflight"
)

var errUnconfigured = errors.New("activity v1 is not configured")

type Catalog interface {
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError)
}

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type Service struct {
	repo      *repository.ActivityRepository
	catalog   Catalog
	users     Users
	followees Followees
	convert   *content.Converter
	cdn       string

	mu     sync.Mutex
	works  map[workKey]workEntry
	flight singleflight.Group
}

type workKey struct {
	id  int
	sfw bool
}

type workEntry struct {
	row     client.CatalogWorkListItem
	found   bool
	expires time.Time
}

func New(repo *repository.ActivityRepository, catalog Catalog, users Users, followees Followees, convert *content.Converter, cdn string) *Service {
	return &Service{repo: repo, catalog: catalog, users: users, followees: followees, convert: convert, cdn: cdn, works: map[workKey]workEntry{}}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.repo == nil || s.catalog == nil || s.users == nil || s.convert == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

const (
	workCacheTTL        = 2 * time.Minute
	workCacheMaxEntries = 20000
	catalogInclude      = "names,intros,labels,covers,refs"
)

// The home page renders this collection for every visitor, and catalog's /v2
// limiter counts per client IP: this forum has one. Rows are cached like the
// galgame client caches its briefs.
func (s *Service) catalogRows(ctx context.Context, ids []int, sfw bool) (map[int]client.CatalogWorkListItem, *problem.Problem) {
	out := make(map[int]client.CatalogWorkListItem, len(ids))
	var missing []int
	now := time.Now()
	s.mu.Lock()
	for _, id := range ids {
		if e, ok := s.works[workKey{id, sfw}]; ok && now.Before(e.expires) {
			if e.found {
				out[id] = e.row
			}
			continue
		}
		missing = append(missing, id)
	}
	s.mu.Unlock()
	if len(missing) == 0 {
		return out, nil
	}
	limit := "all"
	if sfw {
		limit = "sfw"
	}
	slices.Sort(missing)
	key := make([]string, 0, len(missing)+1)
	key = append(key, limit)
	for _, id := range missing {
		key = append(key, strconv.Itoa(id))
	}
	shared, err, _ := s.flight.Do(strings.Join(key, ","), func() (any, error) {
		rows, appErr := s.catalog.CatalogRowsByWorkIDs(context.WithoutCancel(ctx), missing, catalogInclude, limit)
		if appErr != nil {
			return nil, appErr
		}
		s.mu.Lock()
		if len(s.works) > workCacheMaxEntries {
			clear(s.works)
		}
		for _, id := range missing {
			row, ok := rows[id]
			s.works[workKey{id, sfw}] = workEntry{row: row, found: ok, expires: now.Add(workCacheTTL)}
		}
		s.mu.Unlock()
		return rows, nil
	})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	for id, row := range shared.(map[int]client.CatalogWorkListItem) {
		out[id] = row
	}
	return out, nil
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listActivities",
			Method:      "GET",
			Path:        "/activities",
			Summary:     "List activities",
			Description: "The site's activity stream, newest first: topics, replies, comments, upvotes, best answers, works and their resources, ratings, " +
				"edits and quizzes, toolsets, websites, todos and update logs. occurred_desc sorts by occurred_at with ties broken on kind and subject; " +
				"bumped_desc lists topics by bump time and is allowed only when activity_types is exactly topic_creation. " +
				"An activity whose actor is banned, or whose work catalog does not show under include_nsfw, is left out, so a page can be short; " +
				"only an absent next_cursor means the end. The cursor is bound to the sort and to every filter.",
			Tags: []string{"activities"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when sort bumped_desc comes with any activity_types but exactly topic_creation, or a parameter is malformed; " +
					"INVALID_CURSOR, LIMIT_TOO_LARGE, UNKNOWN_SORT or UNKNOWN_ENUM_VALUE.",
				503: "SERVICE_UNAVAILABLE when the account service or catalog is unreachable.",
			}),
		}), s.listActivities)
		registerFollowing(api, s)
	}
}

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}
