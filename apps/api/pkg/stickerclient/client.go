// Package stickerclient reads the NextMoe sticker face at /v2/sticker: the
// packs sticker.kungal.com publishes.
package stickerclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("stickerclient: not configured (empty base URL or API key)")
	ErrUpstream      = errors.New("stickerclient: sticker face error")
)

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:     cfg.APIKey,
		httpClient: hc,
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.baseURL != "" && c.apiKey != ""
}

type Image struct {
	Hash   string `json:"hash"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Sticker struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Image    Image  `json:"image"`
}

type Pack struct {
	ID        string            `json:"id"`
	Title     map[string]string `json:"title"`
	CreatedAt time.Time         `json:"created_at"`
	Stickers  []Sticker         `json:"stickers"`
}

// OfficialPacks answers every published pack the site publishes itself, with
// its stickers in display order. The list carries no stickers, so each pack
// is read on its own.
func (c *Client) OfficialPacks(ctx context.Context) ([]Pack, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var packs []Pack
	q := url.Values{"official": {"true"}, "limit": {"100"}}
	for {
		var page struct {
			Items      []Pack `json:"items"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.get(ctx, "/v2/sticker/packs?"+q.Encode(), &page); err != nil {
			return nil, err
		}
		packs = append(packs, page.Items...)
		if page.NextCursor == "" {
			break
		}
		q.Set("cursor", page.NextCursor)
	}
	out := make([]Pack, len(packs))
	for i, p := range packs {
		if err := c.get(ctx, "/v2/sticker/packs/"+url.PathEscape(p.ID), &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w (status %d): %s", ErrUpstream, resp.StatusCode, problemDetail(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: malformed %s: %v", ErrUpstream, path, err)
	}
	return nil
}

func problemDetail(raw []byte) string {
	var p struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(raw, &p) != nil || p.Code == "" {
		return "non-problem body"
	}
	return p.Code + ": " + p.Detail
}
