package service

import (
	"strings"

	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/errors"
)

type SearchService struct {
	entityService *galgameService.EntitySearchService
	resource      *galgameService.ResourceService
}

func NewSearchService(
	entityService *galgameService.EntitySearchService,
	resource *galgameService.ResourceService,
) *SearchService {
	return &SearchService{entityService: entityService, resource: resource}
}

func tokenize(raw string) ([]string, *errors.AppError) {
	keywords := strings.Fields(strings.TrimSpace(raw))
	if len(keywords) == 0 {
		return nil, errors.ErrBadRequest("搜索关键词不能为空")
	}
	return keywords, nil
}
