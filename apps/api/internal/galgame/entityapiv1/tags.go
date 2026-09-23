package entityapiv1

import (
	"cmp"
	"context"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

const (
	tagIndexTTL     = 10 * time.Minute
	tagIndexPageCap = 40
)

var tagKinds = map[string]bool{"content": true, "meta": true}

func (s *Service) tagIndex(ctx context.Context) ([]TagSummary, error) {
	return s.tags.get(ctx, tagIndexTTL, s.buildTagIndex)
}

func (s *Service) buildTagIndex(ctx context.Context) ([]TagSummary, error) {
	rows := make([]TagSummary, 0, 4096)
	cursor := ""
	for page := 0; page < tagIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"has_works": {"1"}, "limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.catalog.CatalogTaxonomyList(ctx, "tags", q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			t := &res.Items[i]
			if t.Tier == client.TagTierHidden {
				continue
			}
			if !tagKinds[t.Kind] {
				slog.Warn("tag index: unknown tag_kind, row dropped", "tag_id", t.ID, "tag_kind", t.Kind)
				continue
			}
			rows = append(rows, TagSummary{
				Object:           "tag",
				ID:               repr.ID(int(t.ID)),
				CatalogName:      workrepr.Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized)),
				TagKind:          t.Kind,
				IsSexual:         t.Sexual,
				CatalogWorkCount: max(t.WorkCount, 0),
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.CatalogWorkCount != b.CatalogWorkCount {
			return a.CatalogWorkCount > b.CatalogWorkCount
		}
		return idLess(a.ID, b.ID)
	})
	return rows, nil
}

func idLess(a, b repr.DecimalID) bool {
	x, _ := strconv.Atoi(string(a))
	y, _ := strconv.Atoi(string(b))
	return x < y
}

type listTagsInput struct {
	Q           string `query:"q" maxLength:"100" doc:"Name search. Set, the collection is catalog's 100 best name matches in relevance order, hidden and gated tags removed. Free text; never use it as a decision input."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000, or 100 when q is set."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"100" doc:"Page size. 1–100, default 100. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult tags are included. Default false."`
}

type listTagsOutput struct {
	Body repr.PageList[TagSummary]
}

func (s *Service) listTags(ctx context.Context, in *listTagsInput) (*listTagsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q := trimQuery(in.Q)
	if prob := checkDepth(in.Page, in.Limit, q != ""); prob != nil {
		return nil, prob
	}
	index, err := s.tagIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	var rows []TagSummary
	if q == "" {
		rows = make([]TagSummary, 0, len(index))
		for _, r := range index {
			if in.IncludeNSFW || !r.IsSexual {
				rows = append(rows, r)
			}
		}
	} else {
		found, prob := s.searchTags(ctx, q, index, in.IncludeNSFW)
		if prob != nil {
			return nil, prob
		}
		rows = found
	}
	return &listTagsOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

func (s *Service) searchTags(ctx context.Context, q string, index []TagSummary, includeNSFW bool) ([]TagSummary, *problem.Problem) {
	hits, _, appErr := s.catalog.CatalogEntitySearch(ctx, "tags", q, 1, searchDepth)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	byID := make(map[repr.DecimalID]TagSummary, len(index))
	for _, r := range index {
		byID[r.ID] = r
	}
	var missing []int
	for _, h := range hits {
		if h.Tier != client.TagTierHidden {
			if _, ok := byID[repr.ID(int(h.ID))]; !ok {
				missing = append(missing, int(h.ID))
			}
		}
	}
	sexual := map[int]bool{}
	if len(missing) > 0 {
		sexual = s.catalog.CatalogSexualTagIDs(ctx, missing)
	}
	out := make([]TagSummary, 0, len(hits))
	for i := range hits {
		h := &hits[i]
		if h.Tier == client.TagTierHidden {
			continue
		}
		row, ok := byID[repr.ID(int(h.ID))]
		if !ok {
			if !tagKinds[h.Kind] {
				continue
			}
			row = TagSummary{
				Object:      "tag",
				ID:          repr.ID(int(h.ID)),
				CatalogName: workrepr.Name(h.DisplayName, h.Latin, client.LocalizedValues(h.Localized)),
				TagKind:     h.Kind,
				IsSexual:    sexual[int(h.ID)],
			}
		}
		if row.IsSexual && !includeNSFW {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type tagPathInput struct {
	TagID       string `path:"tag_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Tag id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, an adult tag answers. Default false: it is NOT_FOUND."`
}

type getTagOutput struct {
	Body Tag
}

func (s *Service) tagHead(ctx context.Context, rawID string, includeNSFW bool) (*client.CatalogTagDetail, *problem.Problem) {
	if _, ok := pathID(rawID); !ok {
		return nil, notFound()
	}
	t, found, appErr := s.catalog.CatalogTag(ctx, rawID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if !found || (t.Sexual && !includeNSFW) {
		return nil, notFound()
	}
	return t, nil
}

func (s *Service) getTag(ctx context.Context, in *tagPathInput) (*getTagOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	t, prob := s.tagHead(ctx, in.TagID, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	kind := t.Kind
	if !tagKinds[kind] {
		return nil, problem.Internal(errUnknownVocab("tag_kind", kind))
	}
	return &getTagOutput{Body: Tag{
		Object:           "tag",
		ID:               repr.ID(int(t.ID)),
		CatalogName:      workrepr.Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized)),
		TagKind:          kind,
		IsSexual:         t.Sexual,
		CatalogWorkCount: max(t.WorkCount, 0),
		IsHidden:         t.Tier == client.TagTierHidden,
		Intros:           workrepr.Intros(t.Intros),
	}}, nil
}

type listTagWorksInput struct {
	TagID string `path:"tag_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Tag id."`
	WorksQuery
}

func (s *Service) listTagWorks(ctx context.Context, in *listTagWorksInput) (*workPageOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if _, prob := s.tagHead(ctx, in.TagID, in.IncludeNSFW); prob != nil {
		return nil, prob
	}
	page, prob := s.worksPage(ctx, s.walkMembers("tag_id", in.TagID), in.WorksQuery)
	if prob != nil {
		return nil, prob
	}
	return &workPageOutput{Body: page}, nil
}
