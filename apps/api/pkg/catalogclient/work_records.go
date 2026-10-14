package catalogclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	CodePlayRecordsPrivate = "PLAY_RECORDS_PRIVATE"

	PlayRecordsPublic  = "public"
	PlayRecordsPrivate = "private"
)

// WorkStates is catalog's closed play-state vocabulary, in its facet order.
func WorkStates() []string {
	return []string{WorkStateWish, WorkStateDoing, WorkStateDone, WorkStateOnHold, WorkStateDropped}
}

type WorkRecordsQuery struct {
	UserID int64
	States []string
	// SFWOnly asks for content_limit=sfw, which catalog also applies to the
	// state counts.
	SFWOnly bool
	Cursor  string
	Limit   int
	Counts  bool
}

// AccountWorkRecord is one work on an account's library: a work-state always,
// playtime and rating when the account has them.
type AccountWorkRecord struct {
	WorkID          int64
	State           string
	Completion      *string
	StateUpdatedAt  time.Time
	PlaytimeMinutes *int
	Rating          *Rating
}

type WorkRecordsPage struct {
	Items      []AccountWorkRecord
	NextCursor string
	// StateCounts holds every state, 0 included, when Counts was asked for.
	StateCounts map[string]int
}

type v2FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type v2WorkRecordsList struct {
	Items      []v2WorkRecord            `json:"items"`
	NextCursor *string                   `json:"next_cursor"`
	Facets     map[string][]v2FacetValue `json:"facets"`
}

// WorkRecords lists an account's library. With a token it reads the token
// holder's own records from /v2/me/work-records, which ignores the privacy
// switch and in.UserID. With token empty the app key reads in.UserID, and a
// private account answers 403 PLAY_RECORDS_PRIVATE.
func (c *Client) WorkRecords(ctx context.Context, token string, in WorkRecordsQuery) (WorkRecordsPage, error) {
	q := url.Values{}
	if len(in.States) > 0 {
		q.Set("state", strings.Join(in.States, ","))
	}
	if in.SFWOnly {
		q.Set("content_limit", "sfw")
	}
	if in.Cursor != "" {
		q.Set("cursor", in.Cursor)
	}
	if in.Limit > 0 {
		q.Set("limit", strconv.Itoa(in.Limit))
	}
	if in.Counts {
		q.Set("facets", "state")
	}

	var got v2WorkRecordsList
	var err error
	if token != "" {
		err = c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/work-records?"+q.Encode(), nil, &got, nil)
	} else {
		q.Set("user_id", strconv.FormatInt(in.UserID, 10))
		err = c.appV2JSON(ctx, "/v2/work-records", q, &got)
	}
	if err != nil {
		return WorkRecordsPage{}, err
	}

	out := WorkRecordsPage{Items: make([]AccountWorkRecord, 0, len(got.Items))}
	if got.NextCursor != nil {
		out.NextCursor = *got.NextCursor
	}
	for _, r := range got.Items {
		if r.WorkState == nil || r.WorkState.State == "" {
			continue
		}
		rec := AccountWorkRecord{
			WorkID:     parseFlexID(r.WorkID),
			State:      r.WorkState.State,
			Completion: r.WorkState.Completion,
			Rating:     r.Rating.rating(),
		}
		rec.StateUpdatedAt, _ = time.Parse(time.RFC3339Nano, r.WorkState.UpdatedAt)
		if r.Playtime != nil {
			m := r.Playtime.Minutes
			rec.PlaytimeMinutes = &m
		}
		out.Items = append(out.Items, rec)
	}
	if in.Counts {
		out.StateCounts = make(map[string]int, len(WorkStates()))
		for _, s := range WorkStates() {
			out.StateCounts[s] = 0
		}
		for _, b := range got.Facets["state"] {
			if _, ok := out.StateCounts[b.Value]; ok {
				out.StateCounts[b.Value] = b.Count
			}
		}
	}
	return out, nil
}

type PrivacySettings struct {
	PlayRecords string
	UpdatedAt   *time.Time
}

type v2Privacy struct {
	PlayRecords string  `json:"play_records"`
	UpdatedAt   *string `json:"updated_at"`
}

func (p v2Privacy) settings() PrivacySettings {
	out := PrivacySettings{PlayRecords: p.PlayRecords}
	if p.UpdatedAt != nil {
		if t, err := time.Parse(time.RFC3339Nano, *p.UpdatedAt); err == nil {
			out.UpdatedAt = &t
		}
	}
	return out
}

func (c *Client) MyPrivacy(ctx context.Context, token string) (PrivacySettings, error) {
	var out v2Privacy
	if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/privacy", nil, &out, nil); err != nil {
		return PrivacySettings{}, err
	}
	return out.settings(), nil
}

func (c *Client) PatchMyPrivacy(ctx context.Context, token, playRecords string) (PrivacySettings, error) {
	var out v2Privacy
	body := map[string]string{"play_records": playRecords}
	if err := c.userV2JSON(ctx, http.MethodPatch, token, "/v2/me/privacy", body, &out, nil); err != nil {
		return PrivacySettings{}, err
	}
	return out.settings(), nil
}
