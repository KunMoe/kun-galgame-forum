package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// RatingChangesLimit is the feed's page ceiling; a full page is the only one
// that carries next_cursor.
const RatingChangesLimit = 100

type RatingAspects struct {
	Art         *int `json:"art"`
	Story       *int `json:"story"`
	Music       *int `json:"music"`
	Character   *int `json:"character"`
	Route       *int `json:"route"`
	System      *int `json:"system"`
	Voice       *int `json:"voice"`
	ReplayValue *int `json:"replay_value"`
}

// Rating is one person's rating of one work, the same on every NextMoe site.
// WorkState is the author's play state as it stands now, nil when they have
// none.
type Rating struct {
	ID           int64
	WorkID       int64
	UserID       int64
	Overall      int
	Aspects      RatingAspects
	Recommend    string
	SpoilerLevel string
	ShortSummary string
	ClientID     string
	Site         string
	WorkState    *WorkStateRecord
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type v2Rating struct {
	ID           json.RawMessage `json:"id"`
	WorkID       json.RawMessage `json:"work_id"`
	UserID       json.RawMessage `json:"user_id"`
	Overall      int             `json:"overall"`
	AspectScores RatingAspects   `json:"aspect_scores"`
	Recommend    string          `json:"recommend"`
	SpoilerLevel string          `json:"spoiler_level"`
	ShortSummary string          `json:"short_summary"`
	ClientID     string          `json:"client_id"`
	Site         string          `json:"site"`
	WorkState    *struct {
		State      string  `json:"state"`
		Completion *string `json:"completion"`
	} `json:"work_state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *v2Rating) rating() *Rating {
	if r == nil {
		return nil
	}
	out := &Rating{
		ID: parseFlexID(r.ID), WorkID: parseFlexID(r.WorkID), UserID: parseFlexID(r.UserID),
		Overall: r.Overall, Aspects: r.AspectScores, Recommend: r.Recommend,
		SpoilerLevel: r.SpoilerLevel, ShortSummary: r.ShortSummary,
		ClientID: r.ClientID, Site: r.Site, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if s := r.WorkState; s != nil && s.State != "" {
		out.WorkState = &WorkStateRecord{WorkID: out.WorkID, State: s.State, Completion: s.Completion}
	}
	return out
}

// RatingInput is a whole rating: a PUT replaces every field, so an aspect left
// nil is unrated afterwards.
type RatingInput struct {
	Overall      int            `json:"overall"`
	AspectScores *RatingAspects `json:"aspect_scores,omitempty"`
	Recommend    string         `json:"recommend"`
	SpoilerLevel string         `json:"spoiler_level,omitempty"`
	ShortSummary string         `json:"short_summary,omitempty"`
}

type WorkRecordState struct {
	State      string `json:"state"`
	Completion string `json:"completion,omitempty"`
}

type WorkRecordPlaytime struct {
	Minutes int `json:"minutes"`
}

// WorkRecordInput replaces the caller's whole record on a work: a nil part is
// deleted upstream, and Playtime is this application's own report only.
type WorkRecordInput struct {
	WorkState *WorkRecordState    `json:"work_state,omitempty"`
	Playtime  *WorkRecordPlaytime `json:"playtime,omitempty"`
	Rating    *RatingInput        `json:"rating,omitempty"`
}

// WorkRecord is what the caller has recorded on a work. Playtime is the most
// minutes any application reported, so it can outlive this one's report.
type WorkRecord struct {
	WorkState *WorkStateRecord
	Playtime  *PlaytimeSelf
	Rating    *Rating
}

type v2WorkRecord struct {
	WorkID    json.RawMessage `json:"work_id"`
	WorkState *v2WorkState    `json:"work_state"`
	Playtime  *v2Playtime     `json:"playtime"`
	Rating    *v2Rating       `json:"rating"`
}

func (r v2WorkRecord) record() *WorkRecord {
	out := &WorkRecord{Rating: r.Rating.rating()}
	if s := r.WorkState; s != nil && s.State != "" {
		rec := workStateRecord(*s)
		out.WorkState = &rec
	}
	if p := r.Playtime; p != nil {
		out.Playtime = &PlaytimeSelf{WorkID: parseFlexID(p.WorkID), Minutes: p.Minutes}
	}
	return out
}

func workRecordPath(workID int64) string {
	return "/v2/me/work-records/" + strconv.FormatInt(workID, 10)
}

func myRatingPath(workID int64) string {
	return "/v2/me/ratings/" + strconv.FormatInt(workID, 10)
}

func (c *Client) WorkRecord(ctx context.Context, accessToken string, workID int64) (*WorkRecord, error) {
	var out v2WorkRecord
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, workRecordPath(workID), nil, &out, nil); err != nil {
		return nil, err
	}
	return out.record(), nil
}

func (c *Client) PutWorkRecord(ctx context.Context, accessToken string, workID int64, in WorkRecordInput) (*WorkRecord, error) {
	var out v2WorkRecord
	if err := c.userV2JSON(ctx, http.MethodPut, accessToken, workRecordPath(workID), in, &out, nil); err != nil {
		return nil, err
	}
	return out.record(), nil
}

// MyRating answers (nil, nil) when the caller has not rated the work.
func (c *Client) MyRating(ctx context.Context, accessToken string, workID int64) (*Rating, error) {
	var out v2Rating
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, myRatingPath(workID), nil, &out, nil); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return out.rating(), nil
}

func (c *Client) PutMyRating(ctx context.Context, accessToken string, workID int64, in RatingInput) (*Rating, error) {
	var out v2Rating
	if err := c.userV2JSON(ctx, http.MethodPut, accessToken, myRatingPath(workID), in, &out, nil); err != nil {
		return nil, err
	}
	return out.rating(), nil
}

// DeleteMyRating succeeds also when there was no rating.
func (c *Client) DeleteMyRating(ctx context.Context, accessToken string, workID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodDelete, accessToken, myRatingPath(workID), nil, nil)
	return err
}

// ModerationDeleteRating removes a rating from every site. The token's holder
// must moderate the catalog; the forum's own rating permission does not count
// there, and a holder without that standing is answered 403.
func (c *Client) ModerationDeleteRating(ctx context.Context, accessToken string, ratingID int64) error {
	path := "/v2/moderation/ratings/" + strconv.FormatInt(ratingID, 10)
	_, _, err := c.userV2Do(ctx, http.MethodDelete, accessToken, path, nil, nil)
	return err
}

type RatingChange struct {
	ID     int64
	WorkID int64
	Gone   bool
	Rating *Rating
}

type RatingChangePage struct {
	Items      []RatingChange
	NextCursor string
}

// RatingChanges reads the mirror feed with the application key, oldest change
// first. Only a full page carries a next cursor; the caller keeps the last one
// it got and reads the partial tail after it again.
func (c *Client) RatingChanges(ctx context.Context, cursor string) (*RatingChangePage, error) {
	q := url.Values{"limit": {strconv.Itoa(RatingChangesLimit)}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	var out v2List[struct {
		ID     json.RawMessage `json:"id"`
		WorkID json.RawMessage `json:"work_id"`
		Gone   bool            `json:"gone"`
		Rating *v2Rating       `json:"rating"`
	}]
	if err := c.appV2JSON(ctx, "/v2/ratings/changes", q, &out); err != nil {
		return nil, err
	}
	page := &RatingChangePage{Items: make([]RatingChange, 0, len(out.Items))}
	if out.NextCursor != nil {
		page.NextCursor = *out.NextCursor
	}
	for _, it := range out.Items {
		page.Items = append(page.Items, RatingChange{
			ID: parseFlexID(it.ID), WorkID: parseFlexID(it.WorkID), Gone: it.Gone, Rating: it.Rating.rating(),
		})
	}
	return page, nil
}

// CurrentRating reads the person's rating of the work off the public ratings
// list with the application key, adult works included.
func (c *Client) CurrentRating(ctx context.Context, userID, workID int64) (int64, bool, error) {
	q := url.Values{
		"user_id": {strconv.FormatInt(userID, 10)},
		"work_id": {strconv.FormatInt(workID, 10)},
		"nsfw":    {"true"},
		"limit":   {"1"},
	}
	var out v2List[v2Rating]
	if err := c.appV2JSON(ctx, "/v2/ratings", q, &out); err != nil {
		return 0, false, err
	}
	if rows := out.rows(); len(rows) > 0 {
		if id := parseFlexID(rows[0].ID); id > 0 {
			return id, true, nil
		}
	}
	return 0, false, nil
}
