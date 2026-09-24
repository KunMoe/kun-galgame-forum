package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

const (
	ClaimStateNone     = "none"
	ClaimStateLive     = "live"
	ClaimStateDraft    = "draft"
	ClaimStatePending  = "pending"
	ClaimStateDeclined = "declined"
	ClaimStateHidden   = "hidden"
)

const (
	ClaimKindSubmitted = "submitted"
	ClaimKindAudited   = "audited"
)

const (
	ClaimActionClaim    = "claim"
	ClaimActionSubmit   = "submit"
	ClaimActionPublish  = "publish"
	ClaimActionWithdraw = "withdraw"
	ClaimActionApprove  = "approve"
	ClaimActionDecline  = "decline"
	ClaimActionBan      = "ban"
	ClaimActionUnban    = "unban"
)

type ClaimActionResult struct {
	WorkID  int64   `json:"work_id"`
	From    *string `json:"from_state"`
	To      string  `json:"to_state"`
	EventID int64   `json:"event_id"`
}

type WorkSubmitDate struct {
	Y int16 `json:"y"`
	M int16 `json:"m,omitempty"`
	D int16 `json:"d,omitempty"`
}

type WorkSubmitResult struct {
	WorkID        int64  `json:"work_id"`
	ProductWorkID int64  `json:"product_work_id"`
	ClaimState    string `json:"claim_state"`
	EventID       int64  `json:"event_id"`
	ReleaseID     int64  `json:"release_id,omitempty"`
}

type ClaimEventFeedItem struct {
	ID            int64
	WorkID        int64
	FromState     *string
	ToState       string
	ActorUID      int64
	Reason        *string
	Site          string
	ProductWorkID *int64
	CreatedAt     time.Time
}

type ClaimEventFeedPage struct {
	Items   []ClaimEventFeedItem
	HasMore bool
}

// Every id on this face is a decimal STRING on the wire.
type v2ClaimEvent struct {
	ID            json.RawMessage `json:"id"`
	WorkID        json.RawMessage `json:"work_id"`
	FromState     *string         `json:"from_state"`
	ToState       string          `json:"to_state"`
	ActorUID      json.RawMessage `json:"actor_uid"`
	Reason        *string         `json:"reason"`
	Site          string          `json:"site"`
	ProductWorkID json.RawMessage `json:"product_work_id"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ClaimEventsSince walks the claim-transition feed oldest-first from a
// watermark. The watermark is the caller's own durable event id rather than the
// page's opaque next_cursor: an event's effects have to land before the cursor
// may pass it, so a page that fails halfway has to resume in the middle of that
// page, which next_cursor cannot address.
//
// The scope is claim_events:read ON TOP OF catalog:read, and it is granted by an
// operator rather than self-service, so the read key does not carry it by
// default.
func (c *Client) ClaimEventsSince(ctx context.Context, since int64, limit int, site string) (*ClaimEventFeedPage, error) {
	q := url.Values{
		"sort":  {"recorded_asc"},
		"limit": {strconv.Itoa(clampV2Limit(limit))},
	}
	if site != "" {
		q.Set("site", site)
	}
	if cur := encodeWatermark(since); cur != "" {
		q.Set("cursor", cur)
	}
	return c.claimEventPage(ctx, q)
}

// ClaimEventHead is the newest event id, which seeds the watermark on a first
// run so that history is not replayed. It asks the feed for its last row rather
// than walking to the end: at limit<=100 a walk over the whole collection is
// hundreds of pages, and a walk that stops at a page cap seeds the watermark in
// the middle of history and replays everything after it.
func (c *Client) ClaimEventHead(ctx context.Context, site string) (int64, error) {
	q := url.Values{"sort": {"recorded_desc"}, "limit": {"1"}}
	if site != "" {
		q.Set("site", site)
	}
	page, err := c.claimEventPage(ctx, q)
	if err != nil || len(page.Items) == 0 {
		return 0, err
	}
	return page.Items[0].ID, nil
}

func (c *Client) claimEventPage(ctx context.Context, q url.Values) (*ClaimEventFeedPage, error) {
	var raw v2List[v2ClaimEvent]
	if err := c.appV2JSON(ctx, "/v2/catalog/claim-events", q, &raw); err != nil {
		return nil, err
	}
	rows := raw.rows()
	page := &ClaimEventFeedPage{
		Items:   make([]ClaimEventFeedItem, 0, len(rows)),
		HasMore: raw.cursor() != "",
	}
	for i := range rows {
		it := ClaimEventFeedItem{
			ID:        parseFlexID(rows[i].ID),
			WorkID:    parseFlexID(rows[i].WorkID),
			FromState: rows[i].FromState,
			ToState:   rows[i].ToState,
			ActorUID:  parseFlexID(rows[i].ActorUID),
			Reason:    rows[i].Reason,
			Site:      rows[i].Site,
			CreatedAt: rows[i].CreatedAt,
		}
		if n := parseFlexID(rows[i].ProductWorkID); n > 0 {
			it.ProductWorkID = &n
		}
		page.Items = append(page.Items, it)
	}
	return page, nil
}

type UserClaimItem struct {
	WorkID        int64  `json:"work_id"`
	DisplayName   string `json:"display_name"`
	Site          string `json:"site"`
	ProductWorkID *int64 `json:"product_work_id"`
	ClaimState    string `json:"claim_state"`

	LastEventID   int64     `json:"last_event_id"`
	LastFromState *string   `json:"last_from_state"`
	LastToState   string    `json:"last_to_state"`
	LastReason    *string   `json:"last_reason"`
	LastActorUID  int64     `json:"last_actor_uid"`
	LastEventAt   time.Time `json:"last_event_at"`

	FirstActedAt time.Time `json:"first_acted_at"`
	ActedCount   int       `json:"acted_count"`
}

type UserClaimPage struct {
	Items      []UserClaimItem `json:"items"`
	NextBefore int64           `json:"next_before"`
	NextCursor string          `json:"next_cursor"`
	Total      int64           `json:"total"`
}

// Kind ("submitted"/"audited") only exists on the bearer /claims/mine face.
// The by-uid face /catalog/users/{uid}/claims answers ClaimsByActor — every work
// the user TOUCHED — and huma drops unknown query params without erroring, so
// asking it to filter by kind returns someone else's works silently. That face
// has no client here on purpose; a per-user owned list comes from
// GalgameRepository.PublishedIDsByCreator.
type UserClaimFilter struct {
	ClaimStates  []string
	Before       int64
	Cursor       string
	Limit        int
	Kind         string
	IncludeTotal bool
}
