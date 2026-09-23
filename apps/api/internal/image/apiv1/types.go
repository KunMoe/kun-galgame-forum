package apiv1

import (
	"mime/multipart"

	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

const maxImageBytes = 10 * 1024 * 1024

type uploadInput struct {
	RawBody multipart.Form
}

type uploadOutput struct {
	Location string `header:"Location" format:"uri" maxLength:"512" doc:"The image's absolute URL, the same as url in the body."`
	Body     repr.Image
}

// huma.MultipartFormFiles publishes huma.FormFile's Go fields as a component
// the v1 gates reject, so the body is a plain multipart.Form described here.
func uploadRequestBody(field string, values []any, fieldDoc string) *huma.RequestBody {
	maxLen := maxImageBytes
	enumLen := 10
	return &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"multipart/form-data": {
				Schema: &huma.Schema{
					Type:     huma.TypeObject,
					Required: []string{"file", field},
					Properties: map[string]*huma.Schema{
						"file": {
							Type:        huma.TypeString,
							Format:      "binary",
							MaxLength:   &maxLen,
							Description: "The image. Its part must be image/* and at most 10 MiB.",
						},
						field: {
							Type:        huma.TypeString,
							Enum:        values,
							MaxLength:   &enumLen,
							Description: fieldDoc,
						},
					},
				},
				Encoding: map[string]*huma.Encoding{
					"file": {ContentType: "image/*"},
				},
			},
		},
	}
}
