package artifactclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kun-galgame-api/pkg/artifactclient/gen"
)

type (
	InitUploadRequest     = gen.InitUploadRequest
	InitUploadResponse    = gen.InitUploadResponse
	CompleteUploadRequest = gen.CompleteUploadRequest
	ArtifactResponse      = gen.ArtifactResponse
	DownloadResponse      = gen.DownloadResponse
	CompletedPart         = gen.CompletedPart
)

type UploadedPart struct {
	PartNumber int32  `json:"part_number"`
	Etag       string `json:"etag"`
	Size       int64  `json:"size"`
}

type ResumeUploadResponse struct {
	Uuid          string          `json:"uuid"`
	Multipart     bool            `json:"multipart"`
	ExpiresAt     string          `json:"expires_at"`
	PartSize      *int64          `json:"part_size,omitempty"`
	PartUrls      *[]gen.PartURL  `json:"part_urls,omitempty"`
	UploadUrl     *string         `json:"upload_url,omitempty"`
	UploadedParts *[]UploadedPart `json:"uploaded_parts,omitempty"`
}

const (
	codeArtifactNotFound       = 50001
	codeArtifactTooBig         = 50004
	codeArtifactUnauthorized   = 50006
	codeArtifactQuotaExceeded  = 50012
	codeArtifactUploadDisabled = 50014
	codeArtifactSizeMismatch   = 50015
	codeArtifactMIMEDenied     = 50017
)

var (
	ErrNotConfigured  = errors.New("artifactclient: not configured (empty base URL or credentials)")
	ErrUnauthorized   = errors.New("artifactclient: unauthorized (check client_id/secret + artifact_enabled)")
	ErrTooBig         = errors.New("artifactclient: file exceeds the per-site max size")
	ErrQuotaExceeded  = errors.New("artifactclient: daily quota exceeded")
	ErrMIMEDenied     = errors.New("artifactclient: file type not allowed for this site")
	ErrNotFound       = errors.New("artifactclient: artifact not found")
	ErrUploadDisabled = errors.New("artifactclient: upload disabled")
	ErrSizeMismatch   = errors.New("artifactclient: uploaded size does not match declared size")
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type Client struct {
	inner      *gen.ClientWithResponses
	basicAuth  string
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	base := strings.TrimRight(cfg.BaseURL, "/")

	var ba string
	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		ba = "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret))
	}

	c := &Client{basicAuth: ba, baseURL: base, httpClient: hc}
	if base != "" && ba != "" {
		inner, err := gen.NewClientWithResponses(base,
			gen.WithHTTPClient(hc),
			gen.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
				req.Header.Set("Authorization", ba)
				return nil
			}),
		)
		if err == nil {
			c.inner = inner
		}
	}
	return c
}

func (c *Client) Configured() bool { return c.inner != nil && c.basicAuth != "" }

func (c *Client) InitUpload(ctx context.Context, req InitUploadRequest) (*InitUploadResponse, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	resp, err := c.inner.InitUploadWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Code == 0 && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, mapErr(resp.StatusCode(), resp.JSONDefault)
}

func (c *Client) CompleteUpload(ctx context.Context, uuid string, req CompleteUploadRequest) (*ArtifactResponse, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	resp, err := c.inner.CompleteUploadWithResponse(ctx, uuid, req)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Code == 0 && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, mapErr(resp.StatusCode(), resp.JSONDefault)
}

func (c *Client) Resume(ctx context.Context, uuid string) (*ResumeUploadResponse, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	endpoint := c.baseURL + "/api/v1/artifacts/" + url.PathEscape(uuid) + "/resume"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var env struct {
			Code int                   `json:"code"`
			Data *ResumeUploadResponse `json:"data"`
		}
		if json.Unmarshal(body, &env) == nil && env.Code == 0 && env.Data != nil {
			return env.Data, nil
		}
	}
	var he gen.HouseError
	_ = json.Unmarshal(body, &he)
	return nil, mapErr(resp.StatusCode, &he)
}

func (c *Client) Download(ctx context.Context, uuid string) (*DownloadResponse, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	resp, err := c.inner.DownloadArtifactWithResponse(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Code == 0 && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, mapErr(resp.StatusCode(), resp.JSONDefault)
}

func (c *Client) Delete(ctx context.Context, uuid string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	resp, err := c.inner.DeleteArtifactWithResponse(ctx, uuid)
	if err != nil {
		return err
	}
	if resp.JSON200 != nil && resp.JSON200.Code == 0 {
		return nil
	}
	return mapErr(resp.StatusCode(), resp.JSONDefault)
}

func mapErr(status int, he *gen.HouseError) error {
	code := 0
	msg := ""
	if he != nil {
		code = int(he.Code)
		msg = he.Message
	}
	switch code {
	case codeArtifactNotFound:
		return ErrNotFound
	case codeArtifactTooBig:
		return ErrTooBig
	case codeArtifactUnauthorized:
		return ErrUnauthorized
	case codeArtifactQuotaExceeded:
		return ErrQuotaExceeded
	case codeArtifactUploadDisabled:
		return ErrUploadDisabled
	case codeArtifactSizeMismatch:
		return ErrSizeMismatch
	case codeArtifactMIMEDenied:
		return ErrMIMEDenied
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ErrUnauthorized
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return fmt.Errorf("artifactclient: request failed (code %d, http %d): %s", code, status, msg)
}
