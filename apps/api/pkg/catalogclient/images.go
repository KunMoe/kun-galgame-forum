package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type EditImageResult struct {
	Hash         string            `json:"hash"`
	URL          string            `json:"url"`
	VariantURLs  map[string]string `json:"variant_urls"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	Thumbhash    string            `json:"thumbhash"`
	SizeBytes    int64             `json:"size_bytes"`
	Deduplicated bool              `json:"deduplicated"`
}

func (c *Client) UploadEditImageUser(ctx context.Context, accessToken string, r io.Reader, filename, preset string) (*EditImageResult, error) {
	if c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	if accessToken == "" {
		return nil, ErrUnauthorized
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	if _, err := io.Copy(fw, r); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	if err := mw.WriteField("preset", preset); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.origin()+"/v2/me/edit-images", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusForbidden:
		blob := strings.ToLower(p.Code + " " + p.Detail + " " + p.Title + " " + string(raw))
		if p.Code == "SCOPE_REQUIRED" || strings.Contains(blob, "scope") {
			return nil, ErrInsufficientScope
		}
		return nil, &UserAPIError{Status: resp.StatusCode, Message: problemMsg(p, raw)}
	default:
		if resp.StatusCode >= 500 {
			return nil, ErrUpstream
		}
		return nil, &UserAPIError{Status: resp.StatusCode, Message: problemMsg(p, raw)}
	}

	var rec struct {
		Hash           string  `json:"hash"`
		URL            string  `json:"url"`
		Width          *int    `json:"width"`
		Height         *int    `json:"height"`
		Thumbhash      *string `json:"thumbhash"`
		SizeBytes      int64   `json:"size_bytes"`
		IsDeduplicated bool    `json:"is_deduplicated"`
		Deduplicated   bool    `json:"deduplicated"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("%w: malformed upload result", ErrUpstream)
	}
	out := EditImageResult{
		Hash: rec.Hash, URL: rec.URL, SizeBytes: rec.SizeBytes,
		Deduplicated: rec.IsDeduplicated || rec.Deduplicated,
	}
	if rec.Width != nil {
		out.Width = *rec.Width
	}
	if rec.Height != nil {
		out.Height = *rec.Height
	}
	if rec.Thumbhash != nil {
		out.Thumbhash = *rec.Thumbhash
	}
	return &out, nil
}
