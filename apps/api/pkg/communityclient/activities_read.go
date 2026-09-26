package communityclient

import (
	"context"
	"net/http"
	"strings"
)

type ActivityItemView struct {
	ID             int64  `json:"id"`
	Site           string `json:"site"`
	Key            string `json:"key"`
	ActorID        int64  `json:"actor_id"`
	Verb           string `json:"verb"`
	ObjectKind     string `json:"object_kind"`
	ObjectLabel    string `json:"object_label"`
	Title          string `json:"title"`
	Excerpt        string `json:"excerpt"`
	URL            string `json:"url"`
	CoverImageHash string `json:"cover_image_hash"`
	WorkID         *int64 `json:"work_id"`
	ContentLimit   string `json:"content_limit"`
	OccurredAt     string `json:"occurred_at"`
}

type ActivityGroupView struct {
	ID          int64              `json:"id"`
	Site        string             `json:"site"`
	ActorID     int64              `json:"actor_id"`
	Verb        string             `json:"verb"`
	ObjectKind  string             `json:"object_kind"`
	ObjectLabel string             `json:"object_label"`
	Day         string             `json:"day"`
	ItemCount   int                `json:"item_count"`
	LatestAt    string             `json:"latest_at"`
	Items       []ActivityItemView `json:"items"`
}

type ActivityGroupListResponse struct {
	Groups     []ActivityGroupView `json:"groups"`
	NextCursor string              `json:"next_cursor"`
}

type ActivityItemListResponse struct {
	Items      []ActivityItemView `json:"items"`
	NextCursor string             `json:"next_cursor"`
}

type ActivityUnseenResponse struct {
	UnseenCount int     `json:"unseen_count"`
	SeenAt      *string `json:"seen_at"`
}

type ActivitySeenResponse struct {
	SeenAt string `json:"seen_at"`
}

func (c *Client) ListFollowingActivities(ctx context.Context, userID int64, cursor string, limit int, contentLimit string, sites, verbs []string) (*ActivityGroupListResponse, error) {
	var out ActivityGroupListResponse
	q := map[string]string{
		"cursor":        cursor,
		"content_limit": contentLimit,
		"sites":         strings.Join(sites, ","),
		"verbs":         strings.Join(verbs, ","),
	}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/users/"+itoa(userID)+"/following/activities"+query(q), nil, &out)
	return &out, err
}

func (c *Client) ListActivityGroupItems(ctx context.Context, groupID int64, cursor string, limit int, contentLimit string) (*ActivityItemListResponse, error) {
	var out ActivityItemListResponse
	q := map[string]string{"cursor": cursor, "content_limit": contentLimit}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/activity-groups/"+itoa(groupID)+"/items"+query(q), nil, &out)
	return &out, err
}

func (c *Client) GetFollowingActivitiesUnseen(ctx context.Context, userID int64, contentLimit string, sites, verbs []string) (*ActivityUnseenResponse, error) {
	var out ActivityUnseenResponse
	q := map[string]string{
		"content_limit": contentLimit,
		"sites":         strings.Join(sites, ","),
		"verbs":         strings.Join(verbs, ","),
	}
	err := c.do(ctx, http.MethodGet, "/users/"+itoa(userID)+"/following/activities/unseen"+query(q), nil, &out)
	return &out, err
}

func (c *Client) MarkFollowingActivitiesSeen(ctx context.Context, userID int64, at *string) (*ActivitySeenResponse, error) {
	var out ActivitySeenResponse
	req := struct {
		At *string `json:"at,omitempty"`
	}{At: at}
	err := c.do(ctx, http.MethodPost, "/users/"+itoa(userID)+"/following/activities/seen", req, &out)
	return &out, err
}
