package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ForumAPI reads the forum's own public /api/v1, so every gate the web applies
// (renderable works, original-language hosting, adult content) applies to the
// assistant too.
type ForumAPI struct {
	base string
	http *http.Client
}

func NewForumAPI(base string) *ForumAPI {
	return &ForumAPI{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 15 * time.Second}}
}

var errNotFound = errors.New("not found")

func (a *ForumAPI) get(ctx context.Context, path string, q url.Values, out any) error {
	target := a.base + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("GET %s: %d %s", path, res.StatusCode, body)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

type page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type localized map[string]struct {
	Value string `json:"value"`
}

type image struct {
	URL    string  `json:"url"`
	Sexual *string `json:"sexual"`
}

type named struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Localized   localized `json:"localized"`
}

// zhName is what the forum shows by default: the Simplified Chinese title when
// catalog has one.
func (n named) zhName() string {
	if v := strings.TrimSpace(n.Localized["zh-Hans"].Value); v != "" {
		return v
	}
	return n.DisplayName
}

type workRef struct {
	named
	Cover    *image `json:"cover"`
	CoverSFW *image `json:"cover_sfw"`
	IsNSFW   bool   `json:"is_nsfw"`
}

type workSummary struct {
	workRef
	ReleaseDate       *string  `json:"release_date"`
	Maker             *named   `json:"maker"`
	ViewCount         int      `json:"view_count"`
	LikeCount         int      `json:"like_count"`
	RatingScore       *float64 `json:"rating_score"`
	RatingCount       int      `json:"rating_count"`
	ResourcePlatforms []string `json:"resource_platforms"`
	ResourceLanguages []string `json:"resource_languages"`
	ResourceUpdatedAt *string  `json:"resource_updated_at"`
}

type tagRef struct {
	named
	IsSexual bool `json:"is_sexual"`
}

type engine struct {
	named
	Aliases   []string `json:"aliases"`
	WorkCount int      `json:"catalog_work_count"`
}

type workDetail struct {
	workSummary
	Aliases       []string  `json:"aliases"`
	Companies     []named   `json:"companies"`
	Engines       []engine  `json:"engines"`
	Tags          []tagRef  `json:"tags"`
	FavoriteCount int       `json:"favorite_count"`
	ResourceTypes []string  `json:"resource_types"`
	HostedOn      bool      `json:"is_resource_hosted_elsewhere"`
	Series        []named   `json:"series"`
	Intros        []intro   `json:"intros"`
	OriginalLang  *string   `json:"original_language"`
	Related       []workRef `json:"related_works"`
}

type intro struct {
	Locale string `json:"locale"`
	Value  string `json:"value"`
}

type userRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar *image `json:"avatar"`
}

type resource struct {
	ID            string   `json:"id"`
	Work          *workRef `json:"work"`
	Author        *userRef `json:"author"`
	Type          string   `json:"resource_type"`
	Languages     []string `json:"resource_languages"`
	Platforms     []string `json:"resource_platforms"`
	Runtimes      []string `json:"resource_runtimes"`
	Title         string   `json:"title"`
	VersionLabel  *string  `json:"version_label"`
	Size          string   `json:"size"`
	Providers     []string `json:"provider_names"`
	State         string   `json:"state"`
	DownloadCount int      `json:"download_count"`
	LikeCount     int      `json:"like_count"`
	UpdatedAt     string   `json:"updated_at"`
}

type rating struct {
	ID            string   `json:"id"`
	Work          *workRef `json:"work"`
	Author        *userRef `json:"author"`
	Recommend     string   `json:"recommend"`
	Overall       int      `json:"overall"`
	PlayStatus    *string  `json:"play_status"`
	SpoilerLevel  string   `json:"spoiler_level"`
	Summary       string   `json:"short_summary_text"`
	SummarySpoils bool     `json:"has_short_summary_spoiler"`
	ViewCount     int      `json:"view_count"`
	LikeCount     int      `json:"like_count"`
	CreatedAt     string   `json:"created_at"`
}

type toolset struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Aliases       []string `json:"aliases"`
	Type          string   `json:"toolset_type"`
	Language      string   `json:"interface_language"`
	Platform      string   `json:"platform"`
	Homepages     []string `json:"homepage_urls"`
	ViewCount     int      `json:"view_count"`
	DownloadCount int      `json:"download_count"`
	Practicality  *float64 `json:"practicality_average"`
	UpdatedAt     string   `json:"resource_updated_at"`
}

type userHit struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Avatar     *image   `json:"avatar"`
	Bio        string   `json:"bio"`
	Roles      []string `json:"roles"`
	TopicCount int      `json:"topic_count"`
	ReplyCount int      `json:"reply_count"`
}

func nsfwParam(q url.Values, include bool) url.Values {
	q.Set("include_nsfw", strconv.FormatBool(include))
	return q
}

func apiFail(op string, err error) error {
	slog.Error("agent MCP: "+op, "error", err)
	return errUnavailable
}
