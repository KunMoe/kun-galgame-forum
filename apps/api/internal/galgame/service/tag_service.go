package service

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type TagService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
	galgameSvc    *GalgameService
	index         staleCache[indexedTag]
}

func NewTagService(galgameClient *client.GalgameClient, enricher *GalgameEnricher, galgameSvc *GalgameService) *TagService {
	return &TagService{galgameClient: galgameClient, enricher: enricher, galgameSvc: galgameSvc}
}

type TagMultiPage struct {
	Galgames []dto.GalgameCard `json:"galgames"`
	Total    int64             `json:"total"`
}

const taxonomyMemberPageCap = 200

const maxTagFilterIDs = 10

const CatalogCardInclude = "names,covers,labels"

func tagCategory(kind string, sexual bool) string {
	if sexual {
		return "sexual"
	}
	return kind
}

func (s *TagService) Search(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) ([]dto.TaxonomySearchItem, *errors.AppError) {
	hits, _, appErr := s.searchHits(ctx, rawQuery.Get("q"), 1,
		atoiOr(rawQuery.Get("limit"), 20), isSFW)
	if appErr != nil {
		return nil, appErr
	}
	items := make([]dto.TaxonomySearchItem, 0, len(hits))
	for _, h := range hits {
		items = append(items, dto.TaxonomySearchItem{ID: int(h.ID), Name: h.VocabularyName()})
	}
	return items, nil
}

// searchHits carries the hidden-tier and SFW filtering that every tag surface
// owes the reader; the unified entity search needs the hits themselves and the
// catalog total, neither of which survives the TaxonomySearchItem projection.
func (s *TagService) searchHits(
	ctx context.Context,
	keywords string,
	page, limit int,
	isSFW bool,
) ([]client.CatalogEntityHit, int64, *errors.AppError) {
	hits, total, appErr := s.galgameClient.CatalogEntitySearch(ctx, "tags", keywords, page, limit)
	if appErr != nil {
		return nil, 0, appErr
	}

	kept := make([]client.CatalogEntityHit, 0, len(hits))
	ids := make([]int, 0, len(hits))
	for _, h := range hits {
		if h.Tier == client.TagTierHidden {
			continue
		}
		kept = append(kept, h)
		ids = append(ids, int(h.ID))
	}
	sexual := map[int]bool{}
	if isSFW && len(ids) > 0 {
		indexed, missing := s.sexualByID(ctx, ids)
		sexual = indexed
		for id, isSexual := range s.galgameClient.CatalogSexualTagIDs(ctx, missing) {
			sexual[id] = isSexual
		}
	}

	out := make([]client.CatalogEntityHit, 0, len(kept))
	for _, h := range kept {
		if sexual[int(h.ID)] {
			continue
		}
		out = append(out, h)
	}
	// The total is what the paginator divides into pages, so it must not depend
	// on which page came back: subtracting the rows THIS page hid made the page
	// count change every time the reader moved. It over-counts for an SFW
	// reader instead, and the pages it hid rows from are simply short.
	return out, max(total, int64(len(out))), nil
}

func (s *TagService) GetByMultiTag(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*TagMultiPage, *errors.AppError) {
	ids := rawQuery.Get("tag_ids")

	q := url.Values{
		"page":    {strconv.Itoa(atoiOr(rawQuery.Get("page"), 1))},
		"limit":   {strconv.Itoa(atoiOr(rawQuery.Get("limit"), 24))},
		"include": {CatalogCardInclude},
		"sort":    {"released_desc"},
	}
	if selected := splitCSV(ids); len(selected) > 0 {
		if len(selected) > maxTagFilterIDs {
			selected = selected[:maxTagFilterIDs]
		}
		q.Set("tag_id", strings.Join(selected, ","))
	}
	client.ApplyWorksGate(q, isSFW)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, appErr
	}
	return &TagMultiPage{
		Galgames: s.enricher.ToCards(ctx, catalogItemsToNextMoe(ctx, res.Items)),
		Total:    res.Total,
	}, nil
}

func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagListPage, *errors.AppError) {
	index, appErr := s.indexRows(ctx)
	if appErr != nil {
		return nil, appErr
	}

	tags := make([]dto.TagListItem, 0, len(index))
	for _, t := range index {
		if isSFW && t.sexual {
			continue
		}
		tags = append(tags, t.item)
	}
	total := int64(len(tags))

	page, limit := atoiOr(rawQuery.Get("page"), 1), atoiOr(rawQuery.Get("limit"), 100)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 100
	}
	if start := (page - 1) * limit; start >= len(tags) {
		tags = nil
	} else {
		tags = tags[start:min(start+limit, len(tags))]
	}
	return &dto.TagListPage{Tags: tags, Total: total}, nil
}

func (s *TagService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	t, found, appErr := s.galgameClient.CatalogTag(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该标签")
	}

	filter := buildEntityFilter(rawQuery)
	memberIDs, appErr := s.galgameClient.CatalogMemberWorkIDs(ctx,
		entityMemberQuery("tag_id", id, filter), isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	filter.RestrictIDs = memberIDs
	page, appErr := s.galgameSvc.hydrateListCards(ctx, filter, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.TagDetail{
		ID:           int(t.ID),
		Name:         t.Label(),
		Category:     tagCategory(t.Kind, t.Sexual),
		Hidden:       t.Tier == client.TagTierHidden,
		Description:  preferredIntro(t.Intros).Intro,
		Alias:        []string{},
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

func catalogItemsToNextMoe(ctx context.Context, items []client.CatalogWorkListItem) []dto.NextMoeGalgameItem {
	out := make([]dto.NextMoeGalgameItem, 0, len(items))
	for i := range items {
		if !client.CatalogItemRenderable(&items[i]) {
			continue
		}
		out = append(out, client.CatalogItemToNextMoeItem(ctx, &items[i]))
	}
	return out
}
