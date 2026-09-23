package trustclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type AdminError struct {
	Status  int
	Code    int
	Message string
}

func (e *AdminError) Error() string {
	return fmt.Sprintf("trustclient admin: status %d code %d: %s", e.Status, e.Code, e.Message)
}

type ReviewItem struct {
	ID              int64      `json:"id"`
	Site            string     `json:"site"`
	SubjectKind     string     `json:"subject_kind"`
	SubjectID       string     `json:"subject_id"`
	Source          int16      `json:"source"`
	Severity        *int16     `json:"severity"`
	ClassifierScore *float32   `json:"classifier_score"`
	ReportWeightSum *float32   `json:"report_weight_sum"`
	SubjectReach    *int64     `json:"subject_reach"`
	Priority        float32    `json:"priority"`
	ContextNote     *string    `json:"context_note"`
	Status          int16      `json:"status"`
	ClaimedBy       *int64     `json:"claimed_by"`
	ClaimedAt       *time.Time `json:"claimed_at"`
	DecidedBy       *int64     `json:"decided_by"`
	DecidedAt       *time.Time `json:"decided_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Report struct {
	ID              int64     `json:"id"`
	ReporterID      int64     `json:"reporter_id"`
	ReasonID        int64     `json:"reason_id"`
	Note            *string   `json:"note"`
	SubjectSnapshot *string   `json:"subject_snapshot"`
	SubjectURL      *string   `json:"subject_url"`
	Weight          float32   `json:"weight"`
	CreatedAt       time.Time `json:"created_at"`
}

type ReviewItemPage struct {
	Items []ReviewItem `json:"items"`
	Total int64        `json:"total"`
}

type ReviewItemDetail struct {
	Item    ReviewItem `json:"item"`
	Reports []Report   `json:"reports"`
}

type ReviewQuery struct {
	Site   string
	Status *int16
	Page   int
	Limit  int
}

type DecideRequest struct {
	Decision   string  `json:"decision"`
	Action     *int16  `json:"action,omitempty"`
	ReasonCode string  `json:"reason_code,omitempty"`
	Statement  *string `json:"statement,omitempty"`
}

func (c *Client) doAdmin(
	ctx context.Context, method, token, path string, query url.Values, body any, out any,
) error {
	if c.baseURL == "" {
		return ErrNotConfigured
	}
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(raw, &env)

	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return &AdminError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("trustclient admin: decode %s: %w", path, err)
	}
	return nil
}

const adminBase = "/api/v1/admin/trust"

func (c *Client) ListReviewItems(ctx context.Context, token string, q ReviewQuery) (*ReviewItemPage, error) {
	query := url.Values{}
	if q.Site != "" {
		query.Set("site", q.Site)
	}
	if q.Status != nil {
		query.Set("status", strconv.Itoa(int(*q.Status)))
	}
	query.Set("page", strconv.Itoa(q.Page))
	query.Set("limit", strconv.Itoa(q.Limit))
	var page ReviewItemPage
	if err := c.doAdmin(ctx, http.MethodGet, token, adminBase+"/review-items", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) GetReviewItem(ctx context.Context, token string, id int64) (*ReviewItemDetail, error) {
	var detail ReviewItemDetail
	if err := c.doAdmin(ctx, http.MethodGet, token, fmt.Sprintf("%s/review-items/%d", adminBase, id), nil, nil, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func (c *Client) ClaimReviewItem(ctx context.Context, token string, id int64) error {
	return c.doAdmin(ctx, http.MethodPost, token, fmt.Sprintf("%s/review-items/%d/claim", adminBase, id), nil, nil, nil)
}

func (c *Client) DecideReviewItem(ctx context.Context, token string, id int64, req DecideRequest) error {
	return c.doAdmin(ctx, http.MethodPost, token, fmt.Sprintf("%s/review-items/%d/decide", adminBase, id), nil, req, nil)
}
