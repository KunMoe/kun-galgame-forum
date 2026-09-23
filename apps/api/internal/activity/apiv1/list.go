package apiv1

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

const maxRounds = 5

type listInput struct {
	collect.Page
	Types              []ActivityType `query:"activity_types" required:"false" maxItems:"22" doc:"Only these activity types, comma-separated. Absent means every type."`
	TopicSections      string         `query:"topic_sections" enum:"normal,help,all" default:"normal" maxLength:"6" doc:"Which topic_creation activities to include: help is the resource and help sections (g-seeking, g-other, t-help), normal is every other section, all is both. Other kinds are not affected."`
	Sort               string         `query:"sort" enum:"occurred_desc,bumped_desc" default:"occurred_desc" maxLength:"13" doc:"occurred_desc (default) or bumped_desc. bumped_desc sorts topics by bump time and needs activity_types to be exactly topic_creation."`
	IncludeNSFW        bool           `query:"include_nsfw" default:"false" doc:"When true, NSFW activities and works are included. Default false."`
	IncludeUnresourced bool           `query:"include_galgames_without_resources" default:"false" doc:"When true, galgame_creation includes works that have no download resource yet. Default false."`
}

type listOutput struct {
	Body repr.List[Activity]
}

func (in *listInput) feedTypes() []string {
	kinds := in.Types
	if len(kinds) == 0 {
		out := make([]string, len(feedTypes))
		for i, t := range feedTypes {
			out[i] = t.feed
		}
		return out
	}
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if f := feedByKind[string(k)]; f != "" && !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

func (s *Service) listActivities(ctx context.Context, in *listInput) (*listOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	types := in.feedTypes()
	topicOnly := len(types) == 1 && types[0] == "TOPIC_CREATION"
	if in.Sort == "bumped_desc" && !topicOnly {
		return nil, problem.New(problem.CodeInvalidParameter, "sort bumped_desc lists topics only.",
			problem.AtParameter("sort", problem.ReasonInconsistentWith, "activity_types", nil))
	}
	fp := collect.Fingerprint(strings.Join(types, ","), in.TopicSections,
		strconv.FormatBool(in.IncludeNSFW), strconv.FormatBool(in.IncludeUnresourced))
	keys, prob := collect.DecodeCursor(in.Cursor, in.Sort, fp)
	if prob != nil {
		return nil, prob
	}

	var (
		collected []Activity
		lastKeys  []string
		scanned   []string
		exhausted bool
	)
	feedPos, topicPos, prob := parsePos(in.Sort, keys)
	if prob != nil {
		return nil, prob
	}
	for round := 0; len(collected) < in.Limit && round < maxRounds; round++ {
		var rows []repository.FeedRow
		var err error
		if in.Sort == "bumped_desc" {
			rows, err = s.repo.TopicFeedPage(repository.TopicFeedQuery{
				TopicSections: in.TopicSections, IncludeNSFW: in.IncludeNSFW, Limit: in.Limit, After: topicPos,
			})
		} else {
			rows, err = s.repo.FeedPage(repository.FeedQuery{
				Types: types, TopicSections: in.TopicSections, IncludeNSFW: in.IncludeNSFW,
				IncludeUnresourced: in.IncludeUnresourced, Limit: in.Limit, After: feedPos,
			})
		}
		if err != nil {
			return nil, problem.Internal(err)
		}
		if len(rows) == 0 {
			exhausted = true
			break
		}
		items, prob := s.assemble(ctx, rows, in.IncludeNSFW)
		if prob != nil {
			return nil, prob
		}
		for i, it := range items {
			if it == nil {
				continue
			}
			collected = append(collected, *it)
			lastKeys = rowKeys(in.Sort, rows[i])
			if len(collected) == in.Limit {
				break
			}
		}
		last := rows[len(rows)-1]
		scanned = rowKeys(in.Sort, last)
		feedPos = &repository.FeedPos{Created: last.Created, TypeStr: last.TypeStr, SourceID: last.SourceID}
		topicPos = &repository.TopicFeedPos{Bumped: last.Bumped, ID: last.SourceID}
		if len(rows) < in.Limit {
			exhausted = true
			break
		}
	}

	var next *string
	switch {
	case len(collected) == in.Limit:
		c := collect.EncodeCursor(in.Sort, fp, lastKeys...)
		next = &c
	case !exhausted && scanned != nil:
		c := collect.EncodeCursor(in.Sort, fp, scanned...)
		next = &c
	}
	return &listOutput{Body: repr.NewList(collected, next)}, nil
}

func rowKeys(sort string, r repository.FeedRow) []string {
	if sort == "bumped_desc" {
		return []string{r.Bumped.UTC().Format(time.RFC3339Nano), strconv.Itoa(r.SourceID)}
	}
	return []string{r.Created.UTC().Format(time.RFC3339Nano), r.TypeStr, strconv.Itoa(r.SourceID)}
}

func parsePos(sort string, keys []string) (*repository.FeedPos, *repository.TopicFeedPos, *problem.Problem) {
	if keys == nil {
		return nil, nil, nil
	}
	bad := problem.New(problem.CodeInvalidCursor, "The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil))
	if sort == "bumped_desc" {
		if len(keys) != 2 {
			return nil, nil, bad
		}
		t, err1 := time.Parse(time.RFC3339Nano, keys[0])
		id, err2 := strconv.Atoi(keys[1])
		if err1 != nil || err2 != nil {
			return nil, nil, bad
		}
		return nil, &repository.TopicFeedPos{Bumped: t, ID: id}, nil
	}
	if len(keys) != 3 {
		return nil, nil, bad
	}
	t, err1 := time.Parse(time.RFC3339Nano, keys[0])
	id, err2 := strconv.Atoi(keys[2])
	if err1 != nil || err2 != nil || kindByFeed[keys[1]] == "" {
		return nil, nil, bad
	}
	return &repository.FeedPos{Created: t, TypeStr: keys[1], SourceID: id}, nil, nil
}
