package userclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/role"

	"golang.org/x/sync/singleflight"
)

type Config struct {
	BaseURL       string
	ClientID      string
	ClientSecret  string
	ImageCDNBase  string
	CacheTTL      time.Duration
	NegCacheTTL   time.Duration
	HTTPTimeout   time.Duration
	BatchPageSize int
}

type User struct {
	ID              int      `json:"id"`
	UUID            string   `json:"uuid"`
	Name            string   `json:"name"`
	Avatar          string   `json:"avatar"`
	AvatarImageHash string   `json:"avatar_image_hash"`
	Bio             string   `json:"bio"`
	Status          int      `json:"status"`
	Roles           []string `json:"roles"`
	SiteRoles       []string `json:"site_roles"`
	CreatedAt       string   `json:"created_at"`
}

type Client struct {
	cfg    Config
	http   *http.Client
	authHd string

	imageCDNBase string

	mu       sync.RWMutex
	hot      map[int]cacheEntry
	miss     map[int]time.Time
	sfGroup  singleflight.Group
	negTTL   time.Duration
	hotTTL   time.Duration
	pageSize int
}

type cacheEntry struct {
	user   User
	expire time.Time
}

func New(cfg Config) *Client {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Minute
	}
	if cfg.NegCacheTTL == 0 {
		cfg.NegCacheTTL = 1 * time.Minute
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 5 * time.Second
	}
	if cfg.BatchPageSize == 0 || cfg.BatchPageSize > 100 {
		cfg.BatchPageSize = 100
	}
	return &Client{
		cfg:          cfg,
		http:         &http.Client{Timeout: cfg.HTTPTimeout},
		authHd:       "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret)),
		imageCDNBase: strings.TrimRight(cfg.ImageCDNBase, "/"),
		hot:          map[int]cacheEntry{},
		miss:         map[int]time.Time{},
		hotTTL:       cfg.CacheTTL,
		negTTL:       cfg.NegCacheTTL,
		pageSize:     cfg.BatchPageSize,
	}
}

func (c *Client) resolveAvatarURL(u User) string {
	if c.imageCDNBase != "" && u.AvatarImageHash != "" {
		if url := imageclient.MainURL(c.imageCDNBase, u.AvatarImageHash, "webp"); url != "" {
			return url
		}
	}
	return u.Avatar
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type batchData struct {
	Users    []User `json:"users"`
	NotFound []int  `json:"not_found"`
}

type searchData struct {
	Users []User `json:"users"`
}

func (c *Client) Users(ctx context.Context, ids []int) (map[int]User, error) {
	out := map[int]User{}
	if len(ids) == 0 {
		return out, nil
	}

	now := time.Now()
	seen := map[int]struct{}{}
	missing := make([]int, 0, len(ids))

	c.mu.RLock()
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if e, ok := c.hot[id]; ok && now.Before(e.expire) {
			out[id] = e.user
			continue
		}
		if t, ok := c.miss[id]; ok && now.Before(t) {
			continue
		}
		missing = append(missing, id)
	}
	c.mu.RUnlock()

	if len(missing) == 0 {
		return out, nil
	}

	for start := 0; start < len(missing); start += c.pageSize {
		end := start + c.pageSize
		if end > len(missing) {
			end = len(missing)
		}
		shard := missing[start:end]

		key := joinIntsForKey(shard)
		raw, err, _ := c.sfGroup.Do(key, func() (any, error) {
			return c.fetchBatch(ctx, shard)
		})
		if err != nil {
			return out, err
		}
		bd := raw.(batchData)
		c.cacheStore(bd, now)
		for _, u := range bd.Users {
			out[u.ID] = u
		}
	}
	return out, nil
}

func (c *Client) User(ctx context.Context, id int) (User, bool, error) {
	m, err := c.Users(ctx, []int{id})
	if err != nil {
		return User{}, false, err
	}
	u, ok := m[id]
	return u, ok, nil
}

func (c *Client) SearchUsers(ctx context.Context, q string, limit int) ([]User, error) {
	if limit <= 0 {
		limit = 12
	}
	if limit > 50 {
		limit = 50
	}
	endpoint := c.cfg.BaseURL + "/users/search?" + url.Values{
		"q":     {q},
		"limit": {strconv.Itoa(limit)},
	}.Encode()
	var sd searchData
	if err := c.do(ctx, "GET", endpoint, &sd); err != nil {
		return nil, err
	}
	for i := range sd.Users {
		sd.Users[i].Avatar = c.resolveAvatarURL(sd.Users[i])
	}
	// Search results lack site_roles, so caching them stripped site roles from display for 10 minutes.
	return sd.Users, nil
}

func Placeholder(id int) User {
	return User{ID: id, Name: "已注销用户", Avatar: ""}
}

func (c *Client) fetchBatch(ctx context.Context, ids []int) (batchData, error) {
	endpoint := c.cfg.BaseURL + "/users/batch?ids=" + joinInts(ids, ",")
	var bd batchData
	if err := c.do(ctx, "GET", endpoint, &bd); err != nil {
		return bd, err
	}
	for i := range bd.Users {
		bd.Users[i].Avatar = c.resolveAvatarURL(bd.Users[i])
		bd.Users[i].Roles = role.Union(bd.Users[i].Roles, bd.Users[i].SiteRoles)
	}
	return bd, nil
}

func (c *Client) sendEnvelope(ctx context.Context, method, endpoint, auth string, body, v any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", auth)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("userclient: %w", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("userclient: decode envelope: %w", err)
	}
	if env.Code != 0 {
		return &OAuthError{Code: env.Code, Message: env.Message}
	}
	if v == nil || len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, v)
}

func (c *Client) do(ctx context.Context, method, endpoint string, v any) error {
	return c.sendEnvelope(ctx, method, endpoint, c.authHd, nil, v)
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, body, v any) error {
	return c.sendEnvelope(ctx, method, endpoint, c.authHd, body, v)
}

type OAuthError struct {
	Code    int
	Message string
}

func (e *OAuthError) Error() string {
	return fmt.Sprintf("oauth code=%d msg=%q", e.Code, e.Message)
}

// The account service answers a parameter it refuses, such as a /users/search q
// over 50 characters, with its generic invalid-parameter code 9.
const codeInvalidParam = 9

func IsInvalidParam(err error) bool {
	var oe *OAuthError
	return errors.As(err, &oe) && oe.Code == codeInvalidParam
}

type MoemoepointResult struct {
	UserID  int  `json:"user_id"`
	Balance int  `json:"balance"`
	Applied bool `json:"applied"`
}

func (c *Client) AdjustMoemoepoint(
	ctx context.Context,
	userID, delta int,
	reason, ref, idempotencyKey string,
) (MoemoepointResult, error) {
	var out MoemoepointResult
	endpoint := fmt.Sprintf("%s/users/%d/moemoepoint", c.cfg.BaseURL, userID)
	err := c.doJSON(ctx, "POST", endpoint, map[string]any{
		"delta":           delta,
		"reason":          reason,
		"ref":             ref,
		"idempotency_key": idempotencyKey,
	}, &out)
	return out, err
}

func (c *Client) GetMoemoepoint(ctx context.Context, userID int) (int, error) {
	var out struct {
		Balance int `json:"balance"`
	}
	endpoint := fmt.Sprintf("%s/users/%d/moemoepoint", c.cfg.BaseURL, userID)
	if err := c.do(ctx, "GET", endpoint, &out); err != nil {
		return 0, err
	}
	return out.Balance, nil
}

type MoemoepointLogEntry struct {
	ID        int64  `json:"id"`
	Delta     int    `json:"delta"`
	Reason    string `json:"reason"`
	SourceApp string `json:"source_app"`
	Ref       string `json:"ref"`
	CreatedAt string `json:"created_at"`
	IsLocal   bool   `json:"is_local"`
}

type MoemoepointLogPage struct {
	Items   []MoemoepointLogEntry `json:"items"`
	HasMore bool                  `json:"has_more"`
}

func (c *Client) MoemoepointLog(
	ctx context.Context,
	userID, limit, beforeID int,
	reason string,
) (MoemoepointLogPage, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if beforeID > 0 {
		q.Set("before_id", strconv.Itoa(beforeID))
	}
	if reason != "" {
		q.Set("reason", reason)
	}
	endpoint := fmt.Sprintf("%s/users/%d/moemoepoint/log?%s", c.cfg.BaseURL, userID, q.Encode())

	var page MoemoepointLogPage
	if err := c.do(ctx, "GET", endpoint, &page); err != nil {
		return MoemoepointLogPage{}, err
	}
	if page.Items == nil {
		page.Items = []MoemoepointLogEntry{}
	}
	for i := range page.Items {
		page.Items[i].IsLocal = page.Items[i].SourceApp == c.cfg.ClientID
	}
	return page, nil
}

func (c *Client) cacheStore(bd batchData, now time.Time) {
	hotExp := now.Add(c.hotTTL)
	missExp := now.Add(c.negTTL)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, u := range bd.Users {
		c.hot[u.ID] = cacheEntry{user: u, expire: hotExp}
		delete(c.miss, u.ID)
	}
	for _, id := range bd.NotFound {
		c.miss[id] = missExp
	}
}

func (c *Client) Invalidate(ids ...int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.hot, id)
		delete(c.miss, id)
	}
}

func joinInts(xs []int, sep string) string {
	var b strings.Builder
	for i, x := range xs {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(strconv.Itoa(x))
	}
	return b.String()
}

func joinIntsForKey(xs []int) string {
	return joinInts(xs, ",")
}

type CreatorApplication struct {
	ID            int             `json:"id"`
	UserID        int             `json:"user_id"`
	Source        string          `json:"source"`
	Status        string          `json:"status"`
	Evidence      json.RawMessage `json:"evidence,omitempty"`
	Message       string          `json:"message"`
	DeclineReason string          `json:"decline_reason"`
	ReviewedAt    *string         `json:"reviewed_at,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

func (c *Client) doJSONWithToken(ctx context.Context, method, endpoint, token string, body, v any) error {
	return c.sendEnvelope(ctx, method, endpoint, "Bearer "+token, body, v)
}

func (c *Client) CreateCreatorApplication(ctx context.Context, token, source string, evidence json.RawMessage, message string) (*CreatorApplication, error) {
	var out CreatorApplication
	endpoint := c.cfg.BaseURL + "/creator/applications"
	body := map[string]any{"source": source, "evidence": evidence, "message": message}
	if err := c.doJSONWithToken(ctx, "POST", endpoint, token, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetMyCreatorApplication(ctx context.Context, token string) (*CreatorApplication, error) {
	var out *CreatorApplication
	endpoint := c.cfg.BaseURL + "/creator/applications/me"
	if err := c.doJSONWithToken(ctx, "GET", endpoint, token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
