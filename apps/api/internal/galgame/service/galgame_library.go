package service

import (
	"context"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/errors"
)

// EntityUsesLocalList reports whether an entity page's work list leaves the
// catalog membership for local SQL, which only knows resource-carrying rows.
func EntityUsesLocalList(f model.GalgameListFilter) bool { return entityUsesLocalList(f) }

func entityUsesLocalList(f model.GalgameListFilter) bool {
	if f.HasResourcePredicate() {
		return true
	}
	if f.GameType != "" && f.GameType != "all" {
		return true
	}
	if f.MinRatingCount > 0 || f.MinRating > 0 {
		return true
	}
	return false
}

func (s *GalgameService) catalogLibrary(
	ctx context.Context,
	req *dto.GalgameListRequest,
	releasedFrom, releasedTo string,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	if s.galgameClient == nil {
		return nil, errors.ErrInternal("Galgame 目录未启用")
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 24
	}
	q := url.Values{
		"page":    {strconv.Itoa(page)},
		"limit":   {strconv.Itoa(limit)},
		"include": {CatalogCardInclude},
		"sort":    {catalogLibrarySort(req.SortField, req.SortOrder)},
	}
	if releasedFrom != "" {
		q.Set("released_after", releasedFrom)
	}
	if releasedTo != "" {
		q.Set("released_before", releasedTo)
	}
	client.ApplyWorksGate(q, isSFW)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, appErr
	}
	ids := make([]int, 0, len(res.Items))
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			continue
		}
		if id := int(res.Items[i].ID); id > 0 {
			ids = append(ids, id)
		}
	}
	cards, appErr := s.HydrateCardsByIDs(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.GalgameListPage{Galgames: cards, Total: res.Total}, nil
}

func catalogLibrarySort(field, order string) string {
	switch field {
	case "release_date":
		if order == "asc" {
			return "released_asc"
		}
		return "released_desc"
	case "time", "updated":
		return "updated"
	case "relevance":
		return "relevance"
	default:
		return "popularity"
	}
}

func (s *GalgameService) hydrateIDPage(
	ctx context.Context,
	filter model.GalgameListFilter,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 24
	}
	ids := filter.RestrictIDs
	if catalogMemberSort(filter) == "" {
		ids = s.listRepo.OrderRestrictIDs(ids, filter)
	}
	total := int64(len(ids))
	start := (page - 1) * limit
	if start >= len(ids) {
		return &dto.GalgameListPage{Galgames: []dto.GalgameListCard{}, Total: total}, nil
	}
	end := min(start+limit, len(ids))
	cards, appErr := s.HydrateCardsByIDs(ctx, ids[start:end], isSFW)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.GalgameListPage{Galgames: cards, Total: total}, nil
}
