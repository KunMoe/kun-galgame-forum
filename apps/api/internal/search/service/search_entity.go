package service

import (
	"context"

	galgameDto "kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

const entitySearchDefaultLimit = 24

func (s *SearchService) SearchEntities(
	ctx context.Context,
	raw string,
	family string,
	page, limit int,
	isSFW bool,
) ([]galgameDto.EntitySearchGroup, *errors.AppError) {
	if _, appErr := tokenize(raw); appErr != nil {
		return nil, appErr
	}
	if s.entityService == nil {
		return nil, errors.ErrInternal("Galgame 资料库搜索未启用")
	}
	if limit <= 0 {
		limit = entitySearchDefaultLimit
	}
	if page <= 0 {
		page = 1
	}
	return s.entityService.Search(ctx, raw, family, page, limit, isSFW)
}
