// Package newsclient reads the NextMoe news face — the partner index at
// /v2/news, republished from 月幕 Galgame and Galgame 批评.
//
// It holds its own credential rather than reusing the catalog one: the news
// face is gated on the scope news:read, and a key that carries only that scope
// cannot reach the catalogue's write endpoints if it ever leaks.
package newsclient

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	LaneNews   = "news"
	LaneColumn = "column"

	// The upstream rejects anything outside [1,50] rather than clamping.
	MaxLimit = 50
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
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: hc,
	}
}

func (c *Client) Configured() bool { return c.baseURL != "" }

var (
	ErrNotConfigured = errors.New("newsclient: not configured (empty base URL or API key)")
	ErrUnauthorized  = errors.New("newsclient: unauthorized (key missing the news:read scope?)")
	ErrBadRequest    = errors.New("newsclient: rejected by the news face")
	ErrUpstream      = errors.New("newsclient: news service error")
)

// Source is the partner block. Attribution is the condition Galgame 批评
// attached to republication and the click-through to SourceURL is 月幕's, so
// nothing downstream may render an item without also rendering both.
type Source struct {
	Key          string `json:"key"`
	DisplayName  string `json:"display_name"`
	HomepageURL  string `json:"homepage_url"`
	Attribution  string `json:"attribution"`
	ColumnURL    string `json:"column_url"`
	PublisherUID int64  `json:"publisher_uid"`
}

// /v2 calls the source key `name`; the retired /v1 called it `key`. Reading only
// `key` left it empty on every item, and the handler maps sources into a map
// keyed by it — so both partners collapsed onto "" and the feed could name only
// one of them, while ?source= filtered on an empty string.
func (s *Source) UnmarshalJSON(b []byte) error {
	type wire Source
	var aux struct {
		wire
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*s = Source(aux.wire)
	if s.Key == "" {
		s.Key = aux.Name
	}
	return nil
}

// Item carries preview and banner only. The article body is never served and
// is not stored anywhere upstream — following SourceURL is the only way to the
// full text.
type Item struct {
	ID          int64     `json:"id"`
	Source      Source    `json:"source"`
	Lane        string    `json:"lane"`
	SourceURL   string    `json:"source_url"`
	Title       string    `json:"title"`
	Preview     string    `json:"preview"`
	BannerURL   string    `json:"banner_url"`
	PublishedAt time.Time `json:"published_at"`
	WorkIDs     []int64   `json:"work_ids"`
}

func (it *Item) UnmarshalJSON(b []byte) error {
	var aux struct {
		ID          json.RawMessage `json:"id"`
		Source      Source          `json:"source"`
		Lane        string          `json:"lane"`
		SourceURL   string          `json:"source_url"`
		Title       string          `json:"title"`
		Preview     string          `json:"preview"`
		Summary     string          `json:"summary"`
		BannerURL   string          `json:"banner_url"`
		PublishedAt time.Time       `json:"published_at"`
		WorkIDs     json.RawMessage `json:"work_ids"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	it.Source, it.Lane, it.SourceURL = aux.Source, aux.Lane, aux.SourceURL
	it.Title, it.BannerURL, it.PublishedAt = aux.Title, aux.BannerURL, aux.PublishedAt
	// Same rename as the source key: /v2 calls the lede `summary`, /v1 called it
	// `preview`. Every card on the 情报 tab rendered a title and nothing else.
	it.Preview = cmp.Or(aux.Preview, aux.Summary)
	raw := bytes.TrimSpace(aux.ID)
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		n, _ := strconv.ParseInt(s, 10, 64)
		it.ID = n
	} else if len(raw) > 0 {
		_ = json.Unmarshal(raw, &it.ID)
	}
	return nil
}

type Feed struct {
	Items      []Item `json:"items"`
	Count      int64  `json:"count"`
	NextCursor string `json:"next_cursor"`
}

type FeedQuery struct {
	Lane   string
	Source string
	Cursor string
	Limit  int
	// Both bounds are INCLUSIVE upstream (published_at >= / <=), so an
	// exclusive upper bound has to be expressed as T-1µs — the resolution of a
	// Postgres timestamptz. Formatting drops what RFC3339 cannot carry, hence
	// RFC3339Nano below: plain RFC3339 truncates 23:59:59.999999 down to
	// 23:59:59 and loses whatever was published in that last second.
	PublishedAfter  time.Time
	PublishedBefore time.Time
}

func (q FeedQuery) values() url.Values {
	v := url.Values{"include_total": {"true"}}
	if q.Lane != "" {
		v.Set("lane", q.Lane)
	}
	if q.Source != "" {
		v.Set("source", q.Source)
	}
	if q.Cursor != "" {
		v.Set("cursor", q.Cursor)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(min(q.Limit, MaxLimit)))
	}
	if !q.PublishedAfter.IsZero() {
		v.Set("published_after", q.PublishedAfter.UTC().Format(time.RFC3339Nano))
	}
	if !q.PublishedBefore.IsZero() {
		v.Set("published_before", q.PublishedBefore.UTC().Format(time.RFC3339Nano))
	}
	return v
}

func (c *Client) Feed(ctx context.Context, q FeedQuery) (*Feed, error) {
	data, err := c.getData(ctx, "/v2/news", q.values())
	if err != nil {
		return nil, err
	}
	var page struct {
		Items      []Item  `json:"items"`
		Total      *int64  `json:"total"`
		NextCursor *string `json:"next_cursor"`
		Count      int64   `json:"count"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("%w: malformed feed payload", ErrUpstream)
	}
	feed := &Feed{Items: page.Items, Count: page.Count}
	if page.Total != nil {
		feed.Count = *page.Total
	}
	if page.NextCursor != nil {
		feed.NextCursor = *page.NextCursor
	}
	if feed.Items == nil {
		feed.Items = []Item{}
	}
	return feed, nil
}

// Count is how many items match a filter. It is the only aggregate the news
// face exposes — there is no group-by endpoint — so an archive of years and
// months has to be probed one boundary at a time.
func (c *Client) Count(ctx context.Context, q FeedQuery) (int64, error) {
	q.Cursor = ""
	q.Limit = 1
	feed, err := c.Feed(ctx, q)
	if err != nil {
		return 0, err
	}
	return feed.Count, nil
}

// Sources is the whole partner directory, not just the partners that happen to
// appear on the page you fetched. The feed inlines a source block per item, so
// a consumer that only reads the feed can never list a partner whose newest
// item fell off the first page — which is exactly what a source filter has to
// do before the first item is loaded.
func (c *Client) Sources(ctx context.Context) ([]Source, error) {
	data, err := c.getData(ctx, "/v2/news/sources", nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Items   []Source `json:"items"`
		Sources []Source `json:"sources"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("%w: malformed sources payload", ErrUpstream)
	}
	if len(out.Items) > 0 {
		return out.Items, nil
	}
	return out.Sources, nil
}

func (c *Client) getData(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	base := strings.TrimRight(c.baseURL, "/")
	for _, suf := range []string{"/api/v1", "/v2", "/v1"} {
		if strings.HasSuffix(base, suf) {
			base = strings.TrimSuffix(base, suf)
			break
		}
	}
	reqURL := base + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var env struct {
			Code   int             `json:"code"`
			Data   json.RawMessage `json:"data"`
			Object string          `json:"object"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Object == "" && len(bytes.TrimSpace(env.Data)) > 0 {
			return env.Data, nil
		}
		return raw, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, envelopeMessage(raw))
	default:
		return nil, fmt.Errorf("%w (status %d)", ErrUpstream, resp.StatusCode)
	}
}

func envelopeMessage(raw []byte) string {
	var env struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || env.Message == "" {
		return "bad request"
	}
	return env.Message
}
