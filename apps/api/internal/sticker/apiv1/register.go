package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listStickerPacks",
			Method:      http.MethodGet,
			Path:        "/sticker-packs",
			Summary:     "List the editor's sticker packs",
			Description: "The official packs of sticker.kungal.com that the editor's sticker picker offers, all-ages only, in picker order. Not paged. " +
				"Refreshed from the sticker site at most hourly; while it cannot be reached the last good list is served.",
			Tags: []string{"stickers"},
			Responses: map[string]*huma.Response{
				"304": {Description: "The If-None-Match ETag still matches. No body."},
				"503": {
					Description: "SERVICE_UNAVAILABLE when the sticker site is not configured or cannot be reached and no list has been fetched yet.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), s.listStickerPacks)
	}
}
