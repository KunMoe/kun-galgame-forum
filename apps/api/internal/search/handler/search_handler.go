package handler

import (
	"kun-galgame-api/internal/search/dto"
	"kun-galgame-api/internal/search/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type SearchHandler struct {
	searchService *service.SearchService
}

func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

func (h *SearchHandler) SearchEntities(c fiber.Ctx) error {
	var req dto.EntitySearchRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	groups, appErr := h.searchService.SearchEntities(
		c.Context(), req.Keywords, req.Family, req.Page, req.Limit, utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	res := dto.EntitySearchResult{Groups: groups}
	for _, group := range groups {
		res.Total += group.Total
	}
	return response.OK(c, res)
}

func (h *SearchHandler) ResolveEntities(c fiber.Ctx) error {
	var req dto.EntityResolveRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, appErr := h.searchService.ResolveEntities(
		c.Context(), req.Family, req.IDs, utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, dto.EntityResolveResult{Items: items})
}

func (h *SearchHandler) Search(c fiber.Ctx) error {
	var req dto.SearchRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	res, appErr := h.searchService.SearchResources(
		c.Context(), req.Keywords, req.Page, req.Limit, utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, res.Items, res.Total)
}
