package trustclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type Client struct {
	basicAuth  string
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	var ba string
	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		ba = "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret))
	}
	return &Client{
		basicAuth:  ba,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: hc,
	}
}

func (c *Client) Configured() bool { return c.baseURL != "" && c.basicAuth != "" }

var (
	ErrNotConfigured = errors.New("trustclient: not configured (empty base URL or credentials)")
	ErrValidation    = errors.New("trustclient: report rejected (unregistered subject kind or unknown reason)")
	ErrRateLimited   = errors.New("trustclient: reporter rate limit exceeded")
	ErrForbidden     = errors.New("trustclient: client not bound to a site")
	ErrUnauthorized  = errors.New("trustclient: unauthorized (check client_id/secret)")
)

type ReportRequest struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	ReasonKey   string `json:"reason_key"`
	ReporterID  int64  `json:"reporter_id"`
	AuthorID    *int64 `json:"author_id,omitempty"`
	Note        string `json:"note,omitempty"`
	Snapshot    string `json:"snapshot,omitempty"`
	SubjectURL  string `json:"subject_url,omitempty"`
}

type ReportResult struct {
	ReportID     int64 `json:"report_id"`
	ReviewItemID int64 `json:"review_item_id,omitempty"`
}

type ReasonView struct {
	ID           int64  `json:"id"`
	Key          string `json:"key"`
	NameCN       string `json:"name_cn"`
	Severity     int    `json:"severity"`
	IsDeprecated bool   `json:"is_deprecated"`
	Site         string `json:"site,omitempty"`
}

func (c *Client) ListReportReasons(ctx context.Context) ([]ReasonView, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+"/api/v1/trust/report-reasons", nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.basicAuth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env struct {
		Code int `json:"code"`
		Data struct {
			Reasons []ReasonView `json:"reasons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return nil, fmt.Errorf("trustclient: list reasons failed (status %d, code %d)", resp.StatusCode, env.Code)
	}
	return env.Data.Reasons, nil
}

func (c *Client) SubmitReport(ctx context.Context, req ReportRequest) (*ReportResult, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/api/v1/trust/reports", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", c.basicAuth)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Data    *ReportResult `json:"data"`
	}
	_ = json.Unmarshal(raw, &env)

	if resp.StatusCode == http.StatusOK && env.Code == 0 && env.Data != nil {
		return env.Data, nil
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: %s", ErrRateLimited, env.Message)
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("%w: %s", ErrValidation, env.Message)
	case http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	}
	return nil, fmt.Errorf(
		"trustclient: report failed (status %d, code %d): %s", resp.StatusCode, env.Code, env.Message,
	)
}
