package service

import (
	"context"

	galgameDto "kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/search/dto"
	"kun-galgame-api/pkg/errors"
)

// A keyword reaches a download resource two ways: through the note its uploader
// wrote, and through the name of the game it hangs off. Only the first is local
// — the forum stores no game names — so the second one is a catalog search,
// which the resource service owns because the browse page needs it too.
func (s *SearchService) SearchResources(
	ctx context.Context,
	raw string,
	page, limit int,
	isSFW bool,
) (*dto.PaginatedResult[galgameDto.ResourceCard], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	if s.resource == nil {
		return nil, errors.ErrInternal("Galgame 资源搜索未启用")
	}

	res := s.resource.Search(
		ctx, keywords, s.resource.MatchedWorkIDs(ctx, raw, isSFW), page, limit, 0, isSFW)
	return &dto.PaginatedResult[galgameDto.ResourceCard]{
		Items: res.Resources,
		Total: res.Total,
	}, nil
}
