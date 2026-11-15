package linkcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Status string

const (
	StatusAlive   Status = "alive"
	StatusDead    Status = "dead"
	StatusUnknown Status = "unknown"
)

type Config struct {
	BaseURL              string
	APIKey               string
	CFAccessClientID     string
	CFAccessClientSecret string
	Timeout              time.Duration
}

const defaultTimeout = 12 * time.Second

type Client struct {
	baseURL  string
	apiKey   string
	cfID     string
	cfSecret string
	http     *http.Client
}

func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:   cfg.APIKey,
		cfID:     cfg.CFAccessClientID,
		cfSecret: cfg.CFAccessClientSecret,
		http:     &http.Client{Timeout: timeout},
	}
}

type checkItem struct {
	URL      string `json:"url"`
	Passcode string `json:"passcode,omitempty"`
}

type batchRequest struct {
	Items []checkItem `json:"items"`
}

type Result struct {
	Provider string `json:"provider"`
	Status   Status `json:"status"`
	Reason   string `json:"reason"`
}

type batchResponse struct {
	Results []Result `json:"results"`
}

// CheckEach checks every URL on its own and returns one status per URL, in
// input order. Any failure of the call itself reports every URL unknown.
func (c *Client) CheckEach(ctx context.Context, urls []string, passcode string) []Status {
	out := make([]Status, len(urls))
	for i := range out {
		out[i] = StatusUnknown
	}
	if len(urls) == 0 {
		return out
	}
	items := make([]checkItem, len(urls))
	for i, u := range urls {
		items[i] = checkItem{URL: u, Passcode: passcode}
	}
	body, err := json.Marshal(batchRequest{Items: items})
	if err != nil {
		return out
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/check/batch", bytes.NewReader(body))
	if err != nil {
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.cfID != "" && c.cfSecret != "" {
		req.Header.Set("CF-Access-Client-Id", c.cfID)
		req.Header.Set("CF-Access-Client-Secret", c.cfSecret)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	var res batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return out
	}
	return statuses(res.Results, len(urls))
}

func statuses(results []Result, n int) []Status {
	out := make([]Status, n)
	for i := range out {
		out[i] = StatusUnknown
	}
	if len(results) != n {
		return out
	}
	for i, r := range results {
		if r.Status == StatusAlive || r.Status == StatusDead {
			out[i] = r.Status
		}
	}
	return out
}
