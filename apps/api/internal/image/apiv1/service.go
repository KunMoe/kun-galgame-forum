package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/image/repository"
	"kun-galgame-api/internal/middleware"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"gorm.io/gorm"
)

const dailyImageLimit = 50

const (
	imageFileTooLarge = 80007
	imageMIMEDenied   = 80009
)

var (
	errUnconfigured    = errors.New("apiv1 images: service is not configured")
	errMalformedUpload = errors.New("apiv1 images: upstream returned no usable hash")
)

var purposePresets = map[string]string{
	"content": "topic",
	"message": "message",
}

var editPresets = map[string]bool{"cover": true, "screenshot": true}

type Service struct {
	images  *imageclient.Client
	catalog *catalogclient.Client
	quota   *repository.ImageRepository
	states  *userRepo.StateRepository
	cdn     string
}

func New(images *imageclient.Client, catalog *catalogclient.Client, db *gorm.DB, cdn string) *Service {
	return &Service{
		images:  images,
		catalog: catalog,
		quota:   repository.NewImageRepository(db),
		states:  userRepo.NewStateRepository(db),
		cdn:     cdn,
	}
}

func validationFailed(field problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", field)
}

func unsupportedMedia() *problem.Problem {
	return problem.New(problem.CodeUnsupportedMediaType, "The request body media type is not supported.")
}

func fileTooLong() *problem.Problem {
	maxLen := maxImageBytes
	return validationFailed(problem.AtPointer("/file", problem.ReasonTooLong,
		"file must be at most 10 MiB", &problem.FieldParams{MaxLength: &maxLen}))
}

func dailyLimitReached() *problem.Problem {
	p := problem.New(problem.CodeImageDailyLimitReached, "The caller has uploaded as many images today as one user may.")
	p.SetExtension("limit", dailyImageLimit)
	return p
}

func enumField(form *multipart.Form, name string, allowed func(string) bool) (string, *problem.Problem) {
	values := form.Value[name]
	if len(values) == 0 || values[0] == "" {
		return "", validationFailed(problem.AtPointer("/"+name, problem.ReasonRequired, name+" is required", nil))
	}
	if !allowed(values[0]) {
		return "", validationFailed(problem.AtPointer("/"+name, problem.ReasonUnknownValue, name+" is not one of the listed values", nil))
	}
	return values[0], nil
}

func imagePart(form *multipart.Form) (*multipart.FileHeader, *problem.Problem) {
	files := form.File["file"]
	if len(files) == 0 {
		return nil, validationFailed(problem.AtPointer("/file", problem.ReasonRequired, "file is required", nil))
	}
	fh := files[0]
	if !strings.HasPrefix(fh.Header.Get("Content-Type"), "image/") {
		return nil, unsupportedMedia()
	}
	if fh.Size > maxImageBytes {
		return nil, fileTooLong()
	}
	return fh, nil
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

type accessTokenKey struct{}

func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	next(huma.WithValue(ctx, accessTokenKey{}, middleware.GetAccessToken(humafiber.Unwrap(ctx))))
}

func (s *Service) createImage(ctx context.Context, in *uploadInput) (*uploadOutput, error) {
	purpose, prob := enumField(&in.RawBody, "purpose", func(v string) bool { return purposePresets[v] != "" })
	if prob != nil {
		return nil, prob
	}
	fh, prob := imagePart(&in.RawBody)
	if prob != nil {
		return nil, prob
	}
	if s == nil || s.images == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	user := v1.User(ctx)
	if err := s.states.Ensure(user.ID); err != nil {
		return nil, problem.Internal(err)
	}
	reserved, err := s.quota.ReserveDaily(ctx, user.ID, dailyImageLimit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !reserved {
		return nil, dailyLimitReached()
	}
	out, prob := s.uploadToImageService(ctx, fh, purposePresets[purpose])
	if prob != nil {
		if err := s.quota.RefundDaily(context.WithoutCancel(ctx), user.ID); err != nil {
			slog.Error("image quota refund failed", "user_id", user.ID, "error", err)
		}
		return nil, prob
	}
	return out, nil
}

func (s *Service) uploadToImageService(ctx context.Context, fh *multipart.FileHeader, preset string) (*uploadOutput, *problem.Problem) {
	f, err := fh.Open()
	if err != nil {
		return nil, problem.Internal(err)
	}
	defer f.Close()
	res, err := s.images.Upload(ctx, f, fh.Filename, preset)
	if err != nil {
		return nil, imageServiceProblem(err)
	}
	meta := imageclient.ImageMeta{Width: res.Width, Height: res.Height, Thumbhash: res.Thumbhash}
	if metas, err := s.images.MetaBatch(ctx, []string{res.Hash}); err == nil {
		meta.Sexual = metas[res.Hash].Sexual
	}
	img := repr.NewImage(s.cdn, res.Hash, &meta)
	if img == nil {
		return nil, problem.Unavailable(errMalformedUpload)
	}
	return &uploadOutput{Location: img.URL, Body: *img}, nil
}

func imageServiceProblem(err error) *problem.Problem {
	if errors.Is(err, imageclient.ErrModerationRejected) {
		return problem.New(problem.CodeImageRejected, "The image service's moderation refused the image. Nothing was stored.")
	}
	var ie *imageclient.Error
	if errors.As(err, &ie) {
		switch ie.Code {
		case imageFileTooLarge:
			return fileTooLong()
		case imageMIMEDenied:
			return unsupportedMedia()
		}
	}
	return problem.Unavailable(err)
}

func (s *Service) createWorkEditImage(ctx context.Context, in *uploadInput) (*uploadOutput, error) {
	preset, prob := enumField(&in.RawBody, "preset", func(v string) bool { return editPresets[v] })
	if prob != nil {
		return nil, prob
	}
	fh, prob := imagePart(&in.RawBody)
	if prob != nil {
		return nil, prob
	}
	if s == nil || s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	token, _ := ctx.Value(accessTokenKey{}).(string)
	if token == "" {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, problem.Internal(err)
	}
	defer f.Close()
	res, err := s.catalog.UploadEditImageUser(ctx, token, f, fh.Filename, preset)
	if err != nil {
		return nil, catalogProblem(err)
	}
	img := repr.NewImage(s.cdn, res.Hash, &imageclient.ImageMeta{Width: res.Width, Height: res.Height, Thumbhash: res.Thumbhash})
	if img == nil {
		return nil, problem.Unavailable(errMalformedUpload)
	}
	return &uploadOutput{Location: img.URL, Body: *img}, nil
}

func catalogProblem(err error) *problem.Problem {
	switch {
	case errors.Is(err, catalogclient.ErrUnauthorized):
		return problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	case errors.Is(err, catalogclient.ErrInsufficientScope):
		return problem.New(problem.CodeScopeRequired, "The credential is valid but lacks the scope this operation needs.")
	}
	var apiErr *catalogclient.UserAPIError
	if !errors.As(err, &apiErr) {
		return problem.Unavailable(err)
	}
	switch apiErr.Status {
	case http.StatusForbidden:
		return problem.New(problem.CodePermissionRequired, "The catalog refused this upload for the signed-in user.")
	case http.StatusRequestEntityTooLarge:
		return fileTooLong()
	case http.StatusUnsupportedMediaType:
		return unsupportedMedia()
	case http.StatusUnprocessableEntity:
		return validationFailed(problem.AtPointer("/file", problem.ReasonInvalidFormat, apiErr.Message, nil))
	case http.StatusTooManyRequests:
		return problem.New(problem.CodeRateLimited, "The catalog's upload rate limit was exceeded.")
	}
	return problem.Unavailable(err)
}
