package trustclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CheckRequest struct {
	Text     string `json:"text"`
	AuthorID *int64 `json:"author_id,omitempty"`
}

type CheckResult struct {
	Decision string   `json:"decision"`
	Matched  []string `json:"matched"`
}

func (c *Client) Check(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var res CheckResult
	if err := c.postJSON(ctx, "/api/v1/trust/check", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

type ScanRequest struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	Text        string `json:"text"`
	AuthorID    *int64 `json:"author_id,omitempty"`
	// CheckSkipped is accepted only from infra #460 on; trust rejects unknown
	// fields with 422, so sending it to an older trust fails every scan.
	CheckSkipped bool `json:"check_skipped,omitempty"`
}

type ScanResult struct {
	ScanID    int64 `json:"scan_id"`
	Truncated bool  `json:"truncated"`
}

func (c *Client) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var res ScanResult
	if err := c.postJSON(ctx, "/api/v1/trust/scan", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) postJSON(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.basicAuth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("trustclient: decode %s response: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return fmt.Errorf("trustclient: %s failed (status %d, code %d): %s", path, resp.StatusCode, env.Code, env.Message)
	}
	return json.Unmarshal(env.Data, out)
}
