// Package letmoeclient reads the NextMoe letmoe face at /v2/letmoe: the download
// resources 一起萌 (www.letmoe.com) holds for a game.
package letmoeclient

import (
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

var (
	ErrNotConfigured = errors.New("letmoeclient: not configured (empty base URL or API key)")
	ErrNotFound      = errors.New("letmoeclient: not found")
	ErrUpstream      = errors.New("letmoeclient: letmoe face error")
)

// A game with more resources than this is cut at this many; the reader still
// has the game's page on letmoe.
const maxResourcePages = 5

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

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// Resource carries no download link, code or password by design; Files is the
// way to them.
type Resource struct {
	ID             string    `json:"id"`
	WorkID         string    `json:"work_id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	Version        string    `json:"version"`
	Language       []string  `json:"language"`
	Platform       []string  `json:"platform"`
	Runtime        []string  `json:"runtime"`
	Size           string    `json:"size"`
	Description    string    `json:"description"`
	InstallNote    string    `json:"install_note"`
	ContentWarning string    `json:"content_warning"`
	Providers      []string  `json:"providers"`
	DirectDownload bool      `json:"direct_download"`
	CompositeCount int       `json:"composite_count"`
	DownloadCount  int       `json:"download_count"`
	WebURL         string    `json:"web_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Publisher      *User     `json:"publisher"`
}

type Mirror struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Provider        string `json:"provider"`
	URL             string `json:"url"`
	ExtractCode     string `json:"extract_code"`
	ArchivePassword string `json:"archive_password"`
	Note            string `json:"note"`
	IsPrimary       bool   `json:"is_primary"`
}

type CompositePatch struct {
	MoyuResourceID string   `json:"moyu_resource_id"`
	Name           string   `json:"name"`
	Type           []string `json:"type"`
	Language       []string `json:"language"`
	Publisher      string   `json:"publisher"`
}

type Composite struct {
	ID       string         `json:"id"`
	URL      string         `json:"url"`
	Filename string         `json:"filename"`
	Bytes    int64          `json:"bytes"`
	Patch    CompositePatch `json:"patch"`
}

type ResourceFiles struct {
	ResourceID string      `json:"resource_id"`
	Mirrors    []Mirror    `json:"mirrors"`
	Composites []Composite `json:"composites"`
}

// ResourcesForWork answers every published resource letmoe holds for one
// catalog work, publishers attached. A work letmoe never held a resource for is
// an empty list, not an error.
func (c *Client) ResourcesForWork(ctx context.Context, catalogWorkID int64) ([]Resource, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	path := "/v2/letmoe/works/" + strconv.FormatInt(catalogWorkID, 10) + "/resources"
	q := url.Values{
		"limit":   {"100"},
		"include": {"publisher"},
		// The face defaults to nsfw=false, which drops every game catalog rates
		// as adult — most of the galgames this forum lists.
		"nsfw": {"true"},
	}
	var out []Resource
	for range maxResourcePages {
		var page struct {
			Items      []Resource `json:"items"`
			NextCursor *string    `json:"next_cursor"`
		}
		err := c.get(ctx, path, q, &page)
		if errors.Is(err, ErrNotFound) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		q.Set("cursor", *page.NextCursor)
	}
	return out, nil
}

func (c *Client) Resource(ctx context.Context, resourceID string) (*Resource, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var r Resource
	q := url.Values{"nsfw": {"true"}, "include": {"publisher"}}
	if err := c.get(ctx, "/v2/letmoe/resources/"+url.PathEscape(resourceID), q, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Files answers a resource's download options. Each successful call counts one
// download on letmoe, so call it only when a reader asks to download.
func (c *Client) Files(ctx context.Context, resourceID string) (*ResourceFiles, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var files ResourceFiles
	err := c.get(ctx, "/v2/letmoe/resources/"+url.PathEscape(resourceID)+"/file", url.Values{"nsfw": {"true"}}, &files)
	if err != nil {
		return nil, err
	}
	return &files, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
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

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w (status %d): %s", ErrUpstream, resp.StatusCode, problemDetail(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: malformed body: %v", ErrUpstream, err)
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
