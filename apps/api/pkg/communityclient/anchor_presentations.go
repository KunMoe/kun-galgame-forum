package communityclient

import (
	"context"
	"net/http"
)

type AnchorPresentationWriteRequest struct {
	Items []AnchorPresentationItem `json:"items"`
}

type AnchorPresentationItem struct {
	AnchorKind     int32  `json:"anchor_kind"`
	AnchorID       string `json:"anchor_id"`
	Revision       int64  `json:"revision"`
	Title          string `json:"title,omitempty"`
	URL            string `json:"url,omitempty"`
	WorkID         int64  `json:"work_id,omitempty"`
	CoverImageHash string `json:"cover_image_hash,omitempty"`
	ContentLimit   string `json:"content_limit,omitempty"`
	Removed        bool   `json:"removed,omitempty"`
}

type AnchorPresentationWriteResponse struct {
	Results []AnchorPresentationOutcome `json:"results"`
}

type AnchorPresentationOutcome struct {
	AnchorKind int32  `json:"anchor_kind"`
	AnchorID   string `json:"anchor_id"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason,omitempty"`
}

type AnchorPresentationListResponse struct {
	Presentations []AnchorPresentationView `json:"presentations"`
	NextCursor    string                   `json:"next_cursor"`
}

type AnchorPresentationView struct {
	AnchorKind     int32   `json:"anchor_kind"`
	AnchorID       string  `json:"anchor_id"`
	Title          string  `json:"title"`
	URL            string  `json:"url"`
	WorkID         *int64  `json:"work_id"`
	CoverImageHash *string `json:"cover_image_hash"`
	ContentLimit   string  `json:"content_limit"`
	Revision       int64   `json:"revision"`
	Removed        bool    `json:"removed"`
	UpdatedAt      string  `json:"updated_at"`
	RemovedAt      *string `json:"removed_at"`
}

func (c *Client) WriteAnchorPresentations(ctx context.Context, items []AnchorPresentationItem) (*AnchorPresentationWriteResponse, error) {
	var out AnchorPresentationWriteResponse
	err := c.do(ctx, http.MethodPut, "/anchor-presentations", AnchorPresentationWriteRequest{Items: items}, &out)
	return &out, err
}

func (c *Client) ListAnchorPresentations(ctx context.Context, cursor string, limit int) (*AnchorPresentationListResponse, error) {
	var out AnchorPresentationListResponse
	q := map[string]string{"cursor": cursor}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/anchor-presentations"+query(q), nil, &out)
	return &out, err
}
