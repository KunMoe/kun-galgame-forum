package relocation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BatchSize is LetMoe's own limit; a larger batch is refused whole.
const BatchSize = 100

type Receipt struct {
	ForumID    int    `json:"forum_id"`
	ResourceID int64  `json:"resource_id"`
	Result     string `json:"result"`
	Public     bool   `json:"public"`
	Error      string `json:"error"`
}

func (r Receipt) Landed() bool {
	return (r.Result == "created" || r.Result == "exists") && r.ResourceID > 0
}

type LetMoe struct {
	url  string
	key  string
	http *http.Client
}

func NewLetMoe(url, key string) *LetMoe {
	return &LetMoe{url: url, key: key, http: &http.Client{Timeout: 3 * time.Minute}}
}

func (c *LetMoe) Import(ctx context.Context, items []json.RawMessage, dryRun bool) ([]Receipt, error) {
	body, err := json.Marshal(struct {
		DryRun bool              `json:"dry_run"`
		Items  []json.RawMessage `json:"items"`
	}{dryRun, items})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Results []Receipt `json:"results"`
		} `json:"data"`
	}
	if resp.StatusCode != http.StatusOK {
		_ = json.Unmarshal(raw, &env)
		return nil, fmt.Errorf("letmoe import: HTTP %d, code %d: %s", resp.StatusCode, env.Code, env.Message)
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("letmoe import: unreadable answer: %w", err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("letmoe import: code %d: %s", env.Code, env.Message)
	}
	if len(env.Data.Results) != len(items) {
		return nil, fmt.Errorf("letmoe import: %d receipts for %d items", len(env.Data.Results), len(items))
	}
	return env.Data.Results, nil
}
