package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const maxAvatarBytes = 4 * 1024 * 1024

type putMyAvatarInput struct {
	RawBody multipart.Form
}

// huma.MultipartFormFiles publishes huma.FormFile's Go fields (IsSet, Size,
// Filename) as a component that the v1 naming and maxLength gates reject, so
// the body is a plain multipart.Form with its schema declared here.
func avatarRequestBody() *huma.RequestBody {
	maxLen := maxAvatarBytes
	return &huma.RequestBody{
		Content: map[string]*huma.MediaType{
			"multipart/form-data": {
				Schema: &huma.Schema{
					Type: huma.TypeObject,
					Properties: map[string]*huma.Schema{
						"file": {
							Type:        huma.TypeString,
							Format:      "binary",
							MaxLength:   &maxLen,
							Description: "Avatar image. Must be image/* and at most 4 MiB.",
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

type putMyAvatarOutput struct {
	Body repr.Image
}

type avatarUpload struct {
	Hash   string `json:"hash"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (s *Users) putMyAvatar(ctx context.Context, in *putMyAvatarInput) (*putMyAvatarOutput, error) {
	if prob := s.readyOAuth(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	files := in.RawBody.File["file"]
	if len(files) == 0 {
		return nil, validationFailed(problem.AtPointer("/file", problem.ReasonRequired,
			"file is required", nil))
	}
	fh := files[0]
	ct := fh.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		return nil, problem.New(problem.CodeUnsupportedMediaType,
			"The request body media type is not supported.")
	}
	if fh.Size > maxAvatarBytes {
		maxLen := maxAvatarBytes
		return nil, validationFailed(problem.AtPointer("/file", problem.ReasonTooLong,
			"file must be at most 4 MiB",
			&problem.FieldParams{MaxLength: &maxLen}))
	}
	f, err := fh.Open()
	if err != nil {
		return nil, problem.Internal(err)
	}
	raw, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		return nil, problem.Internal(err)
	}
	payload, formType, err := rebuildAvatarMultipart(raw, ct)
	if err != nil {
		return nil, problem.Internal(err)
	}
	resp, err := s.oauth.UploadAvatar(accessToken(ctx), payload, formType)
	if err != nil {
		return nil, unavailable(err)
	}
	if s.accounts != nil {
		s.accounts.Invalidate(user.ID)
	}
	img, err := mapAvatarImage(resp)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &putMyAvatarOutput{Body: img}, nil
}

func rebuildAvatarMultipart(file []byte, contentType string) ([]byte, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="avatar"`)
	h.Set("Content-Type", contentType)
	part, err := mw.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(file); err != nil {
		return nil, "", err
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), mw.FormDataContentType(), nil
}

func mapAvatarImage(data json.RawMessage) (repr.Image, error) {
	var src avatarUpload
	if err := json.Unmarshal(data, &src); err != nil {
		return repr.Image{}, err
	}
	img := repr.Image{URL: src.URL, Hash: src.Hash}
	if src.Width > 0 {
		w := src.Width
		img.Width = &w
	}
	if src.Height > 0 {
		h := src.Height
		img.Height = &h
	}
	return img, nil
}
