package service

import (
	"context"

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

const CatalogCardInclude = "names,covers,labels"

func tagCategory(kind string, sexual bool) string {
	if sexual {
		return "sexual"
	}
	return kind
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
