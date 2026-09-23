package apiv1

import (
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}

const uploadNote = " file's part must be image/* and at most 10 MiB. Location is the image URL. Persist the hash, never the URL. " +
	"An Idempotency-Key retry must resend the same bytes: the multipart boundary is part of the fingerprint, so a rebuilt form conflicts."

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createImage",
			Method:        http.MethodPost,
			Path:          "/images",
			Summary:       "Upload an image",
			DefaultStatus: http.StatusCreated,
			Description: "Stores an image on the image host for use on this site. " +
				"purpose is content for anything published here (topic bodies and covers, banners, icons, prizes) and message for private messages. " +
				"Each user may upload 50 images per day, Asia/Shanghai; a failed upload does not count. " +
				"sexual is null for a new image until the nightly grader has seen it." + uploadNote,
			Tags:        []string{"images"},
			Middlewares: huma.Middlewares{requireMultipart},
			RequestBody: uploadRequestBody("purpose", []any{"content", "message"}, "What the image is for: content or message."),
			Responses: problemResponses(map[int]string{
				413: "PAYLOAD_TOO_LARGE when the whole body is over the server's limit.",
				415: "UNSUPPORTED_MEDIA_TYPE when the body is not multipart/form-data, or file is not an image the image host accepts.",
				422: "VALIDATION_FAILED when file or purpose is missing or wrong, or file is over 10 MiB. IMAGE_REJECTED when the image host's moderation refuses the image.",
				429: "IMAGE_DAILY_LIMIT_REACHED when the caller has used today's 50 uploads.",
				503: "SERVICE_UNAVAILABLE when the image host is not configured or cannot be reached.",
			}),
		})), s.createImage)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createWorkEditImage",
			Method:        http.MethodPost,
			Path:          "/work-edit-images",
			Summary:       "Upload an image for a work edit",
			DefaultStatus: http.StatusCreated,
			Description: "Stores an image under the caller's own catalog identity, for a cover or screenshot row of a work edit proposal, which refers to it by hash. " +
				"preset is cover or screenshot. Not counted against the daily image limit; the catalog applies its own. sexual is always null." + uploadNote,
			Tags:        []string{"images"},
			Middlewares: huma.Middlewares{requireMultipart, withAccessToken},
			RequestBody: uploadRequestBody("preset", []any{"cover", "screenshot"}, "Which edit slot the image is for: cover or screenshot."),
			Responses: problemResponses(map[int]string{
				403: "SCOPE_REQUIRED when the caller's sign-in lacks the catalog edit scope. PERMISSION_REQUIRED when the catalog refuses the caller.",
				413: "PAYLOAD_TOO_LARGE when the whole body is over the server's limit.",
				415: "UNSUPPORTED_MEDIA_TYPE when the body is not multipart/form-data, or file is not an image the catalog accepts.",
				422: "VALIDATION_FAILED when file or preset is missing or wrong, file is over 10 MiB, or the catalog cannot use the image.",
				429: "RATE_LIMITED when the catalog's upload limit is exceeded.",
				503: "SERVICE_UNAVAILABLE when the catalog is not configured or cannot be reached.",
			}),
		})), s.createWorkEditImage)
	}
}
