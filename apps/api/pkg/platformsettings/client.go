// Package platformsettings reads the public keys NextMoe's config center
// publishes for this site (infra docs/integration/oauth/14-settings.md):
// GET {oauth}/settings with the forum's client credentials, polled with ETag.
package platformsettings

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	AgentEnabled = "agent.enabled"

	pollEvery = time.Minute
)

type Client struct {
	url  string
	auth string
	http *http.Client

	mu       sync.RWMutex
	etag     string
	settings map[string]json.RawMessage
}

// New takes the OAuth API base (OAUTH_SERVER_URL, ending in /api/v1).
func New(oauthBase, clientID, clientSecret string) *Client {
	return &Client{
		url:      strings.TrimRight(oauthBase, "/") + "/settings",
		auth:     "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)),
		http:     &http.Client{Timeout: 10 * time.Second},
		settings: map[string]json.RawMessage{},
	}
}

// Bool is the key's last value read, false until one is read. A failed poll
// keeps the previous snapshot.
func (c *Client) Bool(key string) bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	raw, ok := c.settings[key]
	c.mu.RUnlock()
	var v bool
	if ok {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

type envelope struct {
	Code int `json:"code"`
	Data struct {
		Settings map[string]json.RawMessage `json:"settings"`
	} `json:"data"`
}

func (c *Client) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.auth)
	c.mu.RLock()
	if c.etag != "" {
		req.Header.Set("If-None-Match", c.etag)
	}
	c.mu.RUnlock()
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("platformsettings: GET /settings answered %d", resp.StatusCode)
	}
	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("platformsettings: %w", err)
	}
	if body.Code != 0 || body.Data.Settings == nil {
		return fmt.Errorf("platformsettings: code %d", body.Code)
	}
	c.mu.Lock()
	c.etag, c.settings = resp.Header.Get("ETag"), body.Data.Settings
	c.mu.Unlock()
	return nil
}

// Start reads once before returning, then polls until the returned stop runs.
func (c *Client) Start() func() {
	ctx, cancel := context.WithCancel(context.Background())
	poll := func() {
		if err := c.refresh(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("platform settings poll failed; keeping the last snapshot", "error", err)
		}
	}
	poll()
	go func() {
		t := time.NewTicker(pollEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				poll()
			}
		}
	}()
	return cancel
}
