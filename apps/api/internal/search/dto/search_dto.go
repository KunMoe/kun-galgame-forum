package dto

import (
	galgameDto "kun-galgame-api/internal/galgame/dto"
)

type SearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
	Type     string `query:"type" validate:"required,oneof=resource"`
	Page     int    `query:"page" validate:"min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=24"`
}

type EntitySearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
	Family   string `query:"family" validate:"omitempty,oneof=character company staff tag series engine"`
	Page     int    `query:"page" validate:"omitempty,min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=60"`
}

// A filter chip keys on a catalog id, so a shared link arrives holding ids and
// no names.
type EntityResolveRequest struct {
	Family string `query:"family" validate:"required,oneof=company tag"`
	IDs    string `query:"ids" validate:"required,max=107"`
}

type EntityResolveResult struct {
	Items []galgameDto.EntitySearchItem `json:"items"`
}

type PaginatedResult[T any] struct {
	Items []T
	Total int64
}

type EntitySearchResult struct {
	Groups []galgameDto.EntitySearchGroup `json:"groups"`
	Total  int64                          `json:"total"`
}
