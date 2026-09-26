package communityclient

import (
	"context"
	"net/http"
)

type ActivityWriteRequest struct {
	Items []ActivityWriteItem `json:"items"`
}

type ActivityWriteItem struct {
	Key            string `json:"key"`
	ActorID        int64  `json:"actor_id"`
	Revision       int64  `json:"revision"`
	Verb           string `json:"verb,omitempty"`
	ObjectKind     string `json:"object_kind,omitempty"`
	ObjectLabel    string `json:"object_label,omitempty"`
	Title          string `json:"title,omitempty"`
	Excerpt        string `json:"excerpt,omitempty"`
	URL            string `json:"url,omitempty"`
	CoverImageHash string `json:"cover_image_hash,omitempty"`
	WorkID         int64  `json:"work_id,omitempty"`
	ContentLimit   string `json:"content_limit,omitempty"`
	Notify         bool   `json:"notify,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
	Removed        bool   `json:"removed,omitempty"`
}

type ActivityWriteResponse struct {
	Results []ActivityWriteOutcome `json:"results"`
}

type ActivityWriteOutcome struct {
	Key     string `json:"key"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
}

type SiteActivityListResponse struct {
	Activities []SiteActivityView `json:"activities"`
	NextCursor string             `json:"next_cursor"`
}

type SiteActivityView struct {
	ID             int64   `json:"id"`
	Key            string  `json:"key"`
	ActorID        int64   `json:"actor_id"`
	Revision       int64   `json:"revision"`
	Verb           string  `json:"verb"`
	ObjectKind     string  `json:"object_kind"`
	ObjectLabel    string  `json:"object_label"`
	Title          string  `json:"title"`
	Excerpt        string  `json:"excerpt"`
	URL            string  `json:"url"`
	CoverImageHash *string `json:"cover_image_hash"`
	WorkID         *int64  `json:"work_id"`
	ContentLimit   string  `json:"content_limit"`
	Notify         bool    `json:"notify"`
	OccurredAt     string  `json:"occurred_at"`
	Removed        bool    `json:"removed"`
	UpdatedAt      string  `json:"updated_at"`
}

func (c *Client) WriteActivities(ctx context.Context, items []ActivityWriteItem) (*ActivityWriteResponse, error) {
	var out ActivityWriteResponse
	err := c.do(ctx, http.MethodPost, "/activities", ActivityWriteRequest{Items: items}, &out)
	return &out, err
}

func (c *Client) ListSiteActivities(ctx context.Context, cursor string, limit int) (*SiteActivityListResponse, error) {
	var out SiteActivityListResponse
	q := map[string]string{"cursor": cursor}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/activities"+query(q), nil, &out)
	return &out, err
}
