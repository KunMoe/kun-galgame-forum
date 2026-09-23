package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type Service struct {
	list     *repository.TopicListRepository
	topics   *repository.TopicRepository
	taxonomy *repository.TopicTaxonomyRepository
	replies  *repository.ReplyRepository
	comments *repository.CommentRepository
	users    *userclient.Client
	convert  *content.Converter
	cdn      string
}

func New(
	list *repository.TopicListRepository,
	topics *repository.TopicRepository,
	taxonomy *repository.TopicTaxonomyRepository,
	replies *repository.ReplyRepository,
	comments *repository.CommentRepository,
	users *userclient.Client,
	convert *content.Converter,
	cdn string,
) *Service {
	return &Service{
		list: list, topics: topics, taxonomy: taxonomy,
		replies: replies, comments: comments, users: users,
		convert: convert, cdn: cdn,
	}
}

type listTopicsInput struct {
	collect.Page
	Sort        SortToken   `query:"sort" default:"bumped_desc"`
	Category    string      `query:"category" enum:"galgame,technique,others" maxLength:"9" doc:"When set, only this category. Omitted means every category. There is no all token."`
	IncludeNSFW bool        `query:"include_nsfw" default:"false" doc:"When true, NSFW topics are included. Default false."`
	Section     SectionSlug `query:"section" doc:"When set, only topics filed under this section. Omitted means every section."`
}

type listTopicsOutput struct {
	Body repr.List[TopicSummary]
}

func (s *Service) listTopics(ctx context.Context, in *listTopicsInput) (*listTopicsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	token := string(in.Sort)
	spec, ok := lookupSort(token)
	if !ok {
		return nil, problem.New(
			problem.CodeUnknownSort,
			"The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "use a sort token declared by this operation", nil),
		)
	}
	authenticated := v1.User(ctx) != nil
	fp := listFingerprint(spec.Token, in.Category, string(in.Section), in.IncludeNSFW, authenticated)
	keys, curErr := collect.DecodeCursor(in.Cursor, spec.Token, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseKeysetPos(keys, spec)
	if posErr != nil {
		return nil, posErr
	}

	query := repository.KeysetQuery{
		SortKey:       spec.Key,
		Direction:     spec.Direction,
		Category:      in.Category,
		Section:       string(in.Section),
		IncludeNSFW:   in.IncludeNSFW,
		Authenticated: authenticated,
		Pos:           pos,
	}
	items := make([]TopicSummary, 0, in.Limit)
	hasMore := false
	var last repository.TopicKeysetRow
	// Banned authors are dropped after the query. Reading one window per page
	// returned an empty page with a cursor on dev (views_30d_desc), which stalls
	// an infinite-scroll client, so read on until the page is full.
windows:
	for range maxWindows {
		query.Limit = in.Limit
		rows, err := s.list.FindKeyset(query)
		if err != nil {
			return nil, problem.Internal(err)
		}
		more := len(rows) > in.Limit
		if more {
			rows = rows[:in.Limit]
		}
		rendered, p := s.summaries(ctx, rows)
		if p != nil {
			return nil, p
		}
		for i, item := range rendered {
			last = rows[i]
			if item != nil {
				items = append(items, *item)
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
		query.Pos = positionAfter(last, spec)
	}

	var next *string
	if hasMore {
		cur := collect.EncodeCursor(spec.Token, fp, encodeKeys(last, spec)...)
		next = &cur
	}
	return &listTopicsOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) summaries(ctx context.Context, rows []repository.TopicKeysetRow) ([]*TopicSummary, *problem.Problem) {
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	sectionMap := map[int][]string{}
	if len(ids) > 0 {
		var err error
		if sectionMap, err = s.taxonomy.FindSectionNamesByTopicIDs(ids); err != nil {
			return nil, problem.Internal(err)
		}
	}
	miniApps, err := s.topics.LookupMiniApps(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, err := s.users.Users(ctx, userclient.CollectIDs(rows, func(r repository.TopicKeysetRow) int { return r.UserID }))
	if err != nil {
		return nil, problem.Unavailable(err)
	}

	out := make([]*TopicSummary, len(rows))
	for i, row := range rows {
		author := repr.DeletedUserRef(row.UserID)
		if u, ok := users[row.UserID]; ok {
			if !userclient.IsRenderable(u) {
				continue
			}
			author = repr.NewUserRef(s.cdn, u)
		}
		item, err := mapSummary(s.cdn, row, author, sectionMap[row.ID], miniApps[row.ID])
		if err != nil {
			return nil, problem.Internal(err)
		}
		out[i] = &item
	}
	return out, nil
}

func positionAfter(row repository.TopicKeysetRow, spec sortSpec) *repository.KeysetPos {
	return &repository.KeysetPos{ID: row.ID, SortInt: row.SortInt, SortTime: row.SortTime, TimeSort: spec.Kind == sortKindTime}
}

const maxWindows = 5

var errUnconfigured = errors.New("apiv1 topics: service is not configured")

func parseKeysetPos(keys []string, spec sortSpec) (*repository.KeysetPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	pos := &repository.KeysetPos{ID: id, TimeSort: spec.Kind == sortKindTime}
	if spec.Kind == sortKindTime {
		t, err := time.Parse(time.RFC3339Nano, keys[0])
		if err != nil {
			return nil, invalidCursor()
		}
		pos.SortTime = t
		return pos, nil
	}
	n, err := strconv.ParseInt(keys[0], 10, 64)
	if err != nil {
		return nil, invalidCursor()
	}
	pos.SortInt = n
	return pos, nil
}

func encodeKeys(row repository.TopicKeysetRow, spec sortSpec) []string {
	id := strconv.Itoa(row.ID)
	if spec.Kind == sortKindTime {
		return []string{row.SortTime.UTC().Format(time.RFC3339Nano), id}
	}
	return []string{strconv.FormatInt(row.SortInt, 10), id}
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}
