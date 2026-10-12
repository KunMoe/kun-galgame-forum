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
				DescendantCount:  t.DescendantCount,
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
	Q           string           `query:"q" maxLength:"100" doc:"Name search. Set, the collection is catalog's 100 best name matches in relevance order, hidden and gated tags removed. Free text; never use it as a decision input. Mutually exclusive with ids."`
	IDs         []repr.DecimalID `query:"ids" maxItems:"100" doc:"Tag ids to resolve, comma-separated. 1 to 100 of them. Mutually exclusive with q. Absent ids are omitted."`
	Page        int              `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000, or 100 when q is set."`
	Limit       int              `query:"limit" minimum:"1" maximum:"100" default:"100" doc:"Page size. 1–100, default 100. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool             `query:"include_nsfw" default:"false" doc:"When true, adult tags are included. Default false."`
}

type listTagsOutput struct {
	Body repr.PageList[TagSummary]
}

func (s *Service) listTags(ctx context.Context, in *listTagsInput) (*listTagsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q := trimQuery(in.Q)
	if q != "" && len(in.IDs) > 0 {
		return nil, problem.New(problem.CodeInvalidParameter, "q and ids cannot both be set.",
			problem.AtParameter("ids", problem.ReasonInconsistentWith, "q", nil))
	}
	if prob := checkDepth(in.Page, in.Limit, q != ""); prob != nil {
		return nil, prob
	}
	index, err := s.tagIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	var rows []TagSummary
	switch {
	case len(in.IDs) > 0:
		found, prob := s.tagsByIDs(ctx, in.IDs, index, in.IncludeNSFW)
		if prob != nil {
			return nil, prob
		}
		rows = found
	case q == "":
		rows = make([]TagSummary, 0, len(index))
		for _, r := range index {
			if in.IncludeNSFW || !r.IsSexual {
				rows = append(rows, r)
			}
		}
	default:
		found, prob := s.searchTags(ctx, q, index, in.IncludeNSFW)
		if prob != nil {
			return nil, prob
		}
		rows = found
	}
	return &listTagsOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

func (s *Service) tagsByIDs(ctx context.Context, parts []repr.DecimalID, index []TagSummary, includeNSFW bool) ([]TagSummary, *problem.Problem) {
	ids, prob := parseEntityIDs(parts)
	if prob != nil {
		return nil, prob
	}
	ids = uniqueIDs(ids)
	byID := make(map[repr.DecimalID]TagSummary, len(index))
	for _, r := range index {
		byID[r.ID] = r
	}
	out := make([]TagSummary, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[repr.ID(id)]; ok {
			if row.IsSexual && !includeNSFW {
				continue
			}
			out = append(out, row)
			continue
		}
		t, found, appErr := s.catalog.CatalogTag(ctx, strconv.Itoa(id))
		if appErr != nil {
			return nil, unavailable(appErr)
		}
		if !found || t.Tier == client.TagTierHidden || (t.Sexual && !includeNSFW) {
			continue
		}
		if !tagKinds[t.Kind] {
			slog.Warn("list tags ids: unknown tag_kind, row dropped", "tag_id", t.ID, "tag_kind", t.Kind)
			continue
		}
		out = append(out, TagSummary{
			Object:           "tag",
			ID:               repr.ID(int(t.ID)),
			CatalogName:      workrepr.Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized)),
			TagKind:          t.Kind,
			IsSexual:         t.Sexual,
			CatalogWorkCount: max(t.WorkCount, 0),
			DescendantCount:  t.DescendantCount,
		})
	}
	return out, nil
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
	facts := map[int]client.CatalogTagFacts{}
	if len(missing) > 0 {
		facts = s.catalog.CatalogTagFacts(ctx, missing)
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
			f := facts[int(h.ID)]
			row = TagSummary{
				Object:          "tag",
				ID:              repr.ID(int(h.ID)),
				CatalogName:     workrepr.Name(h.DisplayName, h.Latin, client.LocalizedValues(h.Localized)),
				TagKind:         h.Kind,
				IsSexual:        f.Sexual,
				DescendantCount: f.DescendantCount,
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
	subtags, prob := s.subtags(ctx, t.Children, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
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
		ParentTags:       tagRefs(t.Parents, in.IncludeNSFW),
		Subtags:          subtags,
		DescendantCount:  t.DescendantCount,
		Ancestors:        tagNodes(t.Ancestors, nil),
	}}, nil
}

func (s *Service) subtags(ctx context.Context, children []client.CatalogTagNode, includeNSFW bool) ([]TagSummary, *problem.Problem) {
	if len(children) == 0 {
		return []TagSummary{}, nil
	}
	index, err := s.tagIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	ids := make([]repr.DecimalID, len(children))
	for i, c := range children {
		ids[i] = repr.ID(int(c.ID))
	}
	rows, prob := s.tagsByIDs(ctx, ids, index, includeNSFW)
	if prob != nil {
		return nil, prob
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].CatalogWorkCount != rows[j].CatalogWorkCount {
			return rows[i].CatalogWorkCount > rows[j].CatalogWorkCount
		}
		return idLess(rows[i].ID, rows[j].ID)
	})
	return rows, nil
}

func tagRefs(in []client.CatalogTagNode, includeNSFW bool) []TagRef {
	out := make([]TagRef, 0, len(in))
	for _, t := range in {
		if t.Sexual && !includeNSFW {
			continue
		}
		out = append(out, TagRef{Object: "tag", ID: repr.ID(int(t.ID)), CatalogName: workrepr.Name(t.DisplayName, "", nil), IsSexual: t.Sexual})
	}
	return out
}

func tagNodes(in []client.CatalogTagNode, keep map[int64]bool) []TagNode {
	out := make([]TagNode, 0, len(in))
	for _, t := range in {
		if keep != nil && !keep[t.ID] {
			continue
		}
		parents := make([]repr.DecimalID, 0, len(t.ParentIDs))
		for _, p := range t.ParentIDs {
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil || (keep != nil && !keep[id]) {
				continue
			}
			parents = append(parents, repr.DecimalID(p))
		}
		out = append(out, TagNode{Object: "tag", ID: repr.ID(int(t.ID)), CatalogName: workrepr.Name(t.DisplayName, "", nil), IsSexual: t.Sexual, ParentIDs: parents})
	}
	return out
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

type listTagDescendantsOutput struct {
	Body repr.List[TagNode]
}

func (s *Service) listTagDescendants(ctx context.Context, in *tagPathInput) (*listTagDescendantsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	t, prob := s.tagHead(ctx, in.TagID, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	nodes, appErr := s.catalog.CatalogTagDescendants(ctx, in.TagID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	var keep map[int64]bool
	if !in.IncludeNSFW {
		keep = sfwSubtree(t.ID, nodes)
	}
	return &listTagDescendantsOutput{Body: repr.NewList(tagNodes(nodes, keep), nil)}, nil
}

func sfwSubtree(root int64, nodes []client.CatalogTagNode) map[int64]bool {
	children := make(map[int64][]client.CatalogTagNode, len(nodes))
	for _, n := range nodes {
		for _, p := range n.ParentIDs {
			if id, err := strconv.ParseInt(p, 10, 64); err == nil {
				children[id] = append(children[id], n)
			}
		}
	}
	keep := map[int64]bool{root: true}
	queue := []int64{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			if !c.Sexual && !keep[c.ID] {
				keep[c.ID] = true
				queue = append(queue, c.ID)
			}
		}
	}
	return keep
}
