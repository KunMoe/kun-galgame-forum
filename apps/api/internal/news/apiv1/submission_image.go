package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"mime"
	"net/http"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

var errMalformedUpload = errors.New("apiv1 news: the news image upload returned no usable hash")

var newsImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

func unsupportedMedia() *problem.Problem {
	return problem.New(problem.CodeUnsupportedMediaType, "The request body media type is not supported.")
}

func newsImageTooLong() *problem.Problem {
	maxLen := maxNewsImageBytes
	return validationFailed(problem.AtPointer("/file", problem.ReasonTooLong,
		"file must be at most 4,000,000 bytes", &problem.FieldParams{MaxLength: &maxLen}))
}

func requireMultipart(ctx huma.Context, next func(huma.Context)) {
	if mt, _, err := mime.ParseMediaType(ctx.Header("Content-Type")); err != nil || mt != "multipart/form-data" {
		fc := humafiber.Unwrap(ctx)
		if err := problem.Write(fc, unsupportedMedia()); err != nil {
			slog.Error("apiv1 write problem", "request_id", problem.RequestID(fc), "err", err)
		}
		return
	}
	next(ctx)
}

// huma.MultipartFormFiles publishes huma.FormFile's Go fields as a component
// the v1 gates reject, so the body is a plain multipart.Form described here.
func newsImageRequestBody() *huma.RequestBody {
	maxLen := maxNewsImageBytes
	return &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{
			"multipart/form-data": {
				Schema: &huma.Schema{
					Type:     huma.TypeObject,
					Required: []string{"file"},
					Properties: map[string]*huma.Schema{
						"file": {
							Type:        huma.TypeString,
							Format:      "binary",
							MaxLength:   &maxLen,
							Description: "The banner. Its part must be image/jpeg, image/png or image/webp, and at most 4,000,000 bytes.",
						},
					},
				},
				Encoding: map[string]*huma.Encoding{
					"file": {ContentType: "image/jpeg, image/png, image/webp"},
				},
			},
		},
	}
}

func (s *Service) createNewsSubmissionImage(ctx context.Context, in *newsImageInput) (*newsImageOutput, error) {
	_, token, p := s.submissionCaller(ctx)
	if p != nil {
		return nil, p
	}
	files := in.RawBody.File["file"]
	if len(files) == 0 {
		return nil, validationFailed(problem.AtPointer("/file", problem.ReasonRequired, "file is required", nil))
	}
	fh := files[0]
	if mt, _, err := mime.ParseMediaType(fh.Header.Get("Content-Type")); err != nil || !newsImageTypes[mt] {
		return nil, unsupportedMedia()
	}
	if fh.Size > maxNewsImageBytes {
		return nil, newsImageTooLong()
	}
	f, err := fh.Open()
	if err != nil {
		return nil, problem.Internal(err)
	}
	defer f.Close()
	res, err := s.catalog.UploadNewsImageUser(ctx, token, f, fh.Filename)
	if err != nil {
		return nil, mapNewsImageUpstream(err)
	}
	img := repr.NewImage(s.cdn, res.Hash, &imageclient.ImageMeta{Width: res.Width, Height: res.Height, Thumbhash: res.Thumbhash})
	if img == nil {
		return nil, problem.Unavailable(errMalformedUpload)
	}
	return &newsImageOutput{Location: img.URL, Body: *img}, nil
}

func mapNewsImageUpstream(err error) error {
	var api *catalogclient.UserAPIError
	if !errors.As(err, &api) {
		return mapSubmissionUpstream(err)
	}
	switch api.Status {
	case http.StatusRequestEntityTooLarge:
		return newsImageTooLong()
	case http.StatusUnprocessableEntity:
		// Upstream answers NOT_ALLOWED_VALUE for a refused type and for a
		// moderation refusal alike. The type was checked here before forwarding.
		for _, e := range api.FieldErrors {
			if e.Pointer == "/file" && e.Reason == problem.ReasonNotAllowedValue {
				return problem.New(problem.CodeImageRejected, "The image service's moderation refused the image. Nothing was stored.")
			}
		}
	}
	return mapSubmissionUpstream(err)
}
