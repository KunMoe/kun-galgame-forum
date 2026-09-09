package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type v2Problem struct {
	Code   string              `json:"code"`
	Title  string              `json:"title"`
	Detail string              `json:"detail"`
	Errors []ProblemFieldError `json:"errors"`
}

func (c *Client) origin() string {
	u := strings.TrimRight(c.baseURL, "/")
	for _, suf := range []string{"/api/v1", "/v2", "/v1"} {
		if strings.HasSuffix(u, suf) {
			return strings.TrimSuffix(u, suf)
		}
	}
	return u
}

func parseFlexID(raw json.RawMessage) int64 {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return 0
		}
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	var n int64
	_ = json.Unmarshal(raw, &n)
	return n
}

func ifMatchStar() map[string]string {
	return map[string]string{"If-Match": "*"}
}

func (c *Client) userV2Do(ctx context.Context, method, accessToken, path string, body any, headers map[string]string) ([]byte, string, error) {
	if c.baseURL == "" {
		return nil, "", ErrNotConfigured
	}
	if accessToken == "" {
		return nil, "", ErrUnauthorized
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin()+path, rdr)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	etag := resp.Header.Get("ETag")

	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return raw, etag, nil
	case http.StatusUnauthorized:
		return nil, etag, ErrUnauthorized
	case http.StatusNotFound:
		return nil, etag, ErrNotFound
	case http.StatusForbidden:
		blob := strings.ToLower(p.Code + " " + p.Detail + " " + p.Title + " " + string(raw))
		if p.Code == "SCOPE_REQUIRED" || strings.Contains(blob, "scope") {
			return nil, etag, ErrInsufficientScope
		}
		return nil, etag, &UserAPIError{Status: resp.StatusCode, Message: problemMsg(p, raw)}
	default:
		if resp.StatusCode >= 500 {
			return nil, etag, ErrUpstream
		}
		return nil, etag, &UserAPIError{
			Status: resp.StatusCode, Message: problemMsg(p, raw),
			ProblemCode: p.Code, FieldErrors: p.Errors,
		}
	}
}

func problemMsg(p v2Problem, raw []byte) string {
	if p.Detail != "" {
		return p.Detail
	}
	if p.Title != "" {
		return p.Title
	}
	return strings.TrimSpace(string(raw))
}

func (c *Client) userV2JSON(ctx context.Context, method, accessToken, path string, body, out any, headers map[string]string) error {
	raw, _, err := c.userV2Do(ctx, method, accessToken, path, body, headers)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil
	}
	var env struct {
		Code   int             `json:"code"`
		Data   json.RawMessage `json:"data"`
		Object string          `json:"object"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Object == "" && len(bytes.TrimSpace(env.Data)) > 0 {
		raw = env.Data
	}
	return json.Unmarshal(raw, out)
}

type v2Proposal struct {
	ID             json.RawMessage `json:"id"`
	State          string          `json:"state"`
	Status         string          `json:"status"`
	TargetObject   string          `json:"target_object"`
	EntityType     string          `json:"entity_type"`
	EntityID       json.RawMessage `json:"entity_id"`
	Note           string          `json:"note"`
	DecisionNote   string          `json:"decision_note"`
	Patch          map[string]any  `json:"patch"`
	EffectivePatch map[string]any  `json:"effective_patch"`
	Amendments     []EditAmendment `json:"amendments"`
	ProposerUID    json.RawMessage `json:"proposer_uid"`
	Site           string          `json:"site"`
	Merged         bool            `json:"merged"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	DecidedAt      *string         `json:"decided_at"`
}

func (p v2Proposal) proposal() EditProposal {
	status := p.State
	if status == "" {
		status = p.Status
	}
	if status == "" {
		status = "open"
	}
	entityType := p.EntityType
	if entityType == "" {
		entityType = entityTypeFromObject(p.TargetObject)
	}
	patch := p.Patch
	if patch == nil {
		patch = map[string]any{}
	}
	effective := p.EffectivePatch
	if effective == nil {
		effective = map[string]any{}
	}
	amendments := p.Amendments
	if amendments == nil {
		amendments = []EditAmendment{}
	}
	out := EditProposal{
		ID:             parseFlexID(p.ID),
		EntityType:     entityType,
		EntityID:       parseFlexID(p.EntityID),
		Note:           p.Note,
		DecisionNote:   p.DecisionNote,
		Patch:          patch,
		EffectivePatch: effective,
		Amendments:     amendments,
		ProposerUID:    parseFlexID(p.ProposerUID),
		Site:           p.Site,
		Status:         status,
		CreatedAt:      parseRFC3339(p.CreatedAt),
		UpdatedAt:      parseRFC3339(p.UpdatedAt),
	}
	if p.DecidedAt != nil {
		t := parseRFC3339(*p.DecidedAt)
		if !t.IsZero() {
			out.DecidedAt = &t
		}
	}
	return out
}

func parseRFC3339(raw string) time.Time {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, _ = time.Parse("2006-01-02T15:04:05Z", raw)
	}
	return t
}

type v2Revision struct {
	ID            json.RawMessage `json:"id"`
	TargetObject  string          `json:"target_object"`
	EntityID      json.RawMessage `json:"entity_id"`
	SiteWorkID    json.RawMessage `json:"site_work_id"`
	Seq           int             `json:"seq"`
	Action        string          `json:"action"`
	ChangedFields []string        `json:"changed_fields"`
	ActorUID      json.RawMessage `json:"actor_uid"`
	AmenderUID    json.RawMessage `json:"amender_uid"`
	ProposalID    json.RawMessage `json:"proposal_id"`
	Site          string          `json:"site"`
	CreatedAt     string          `json:"created_at"`
	Diff          []EditFieldDiff `json:"diff"`
}

func (r v2Revision) time() time.Time {
	return parseRFC3339(r.CreatedAt)
}

func (r v2Revision) revision() EditRevision {
	out := EditRevision{
		ID:            parseFlexID(r.ID),
		Seq:           r.Seq,
		Action:        r.Action,
		ChangedFields: r.ChangedFields,
		ActorUID:      parseFlexID(r.ActorUID),
		Site:          r.Site,
		CreatedAt:     r.time(),
	}
	if n := parseFlexID(r.AmenderUID); n != 0 {
		out.AmenderUID = &n
	}
	if n := parseFlexID(r.ProposalID); n != 0 {
		out.ProposalID = &n
	}
	return out
}

type v2Claim struct {
	ID            json.RawMessage `json:"id"`
	WorkID        json.RawMessage `json:"work_id"`
	State         string          `json:"state"`
	ClaimState    string          `json:"claim_state"`
	DisplayName   string          `json:"display_name"`
	LastToState   string          `json:"last_to_state"`
	LastFromState *string         `json:"last_from_state"`
	LastEventID   int64           `json:"last_event_id"`
	LastActorUID  int64           `json:"last_actor_uid"`
	ActedCount    int             `json:"acted_count"`
}

func (c v2Claim) item() UserClaimItem {
	id := parseFlexID(c.ID)
	if id == 0 {
		id = parseFlexID(c.WorkID)
	}
	state := c.State
	if state == "" {
		state = c.ClaimState
	}
	return UserClaimItem{
		WorkID: id, DisplayName: c.DisplayName, ClaimState: state,
		LastToState: c.LastToState, LastFromState: c.LastFromState,
		LastEventID: c.LastEventID, LastActorUID: c.LastActorUID, ActedCount: c.ActedCount,
	}
}

type v2List[T any] struct {
	Items      []T     `json:"items"`
	Covers     []T     `json:"covers"`
	Total      *int64  `json:"total"`
	NextCursor *string `json:"next_cursor"`
	Cursor     string  `json:"cursor"`
}

func (p v2List[T]) rows() []T {
	if len(p.Items) > 0 {
		return p.Items
	}
	return p.Covers
}

func (p v2List[T]) cursor() string {
	if p.NextCursor != nil {
		return *p.NextCursor
	}
	return p.Cursor
}

type v2Schema struct {
	EntityType string `json:"entity_type"`
	Fields     []struct {
		Key           string             `json:"key"`
		FieldType     string             `json:"field_type"`
		Kind          string             `json:"kind"`
		DiffHint      string             `json:"diff_hint"`
		Deprecated    bool               `json:"deprecated"`
		CanPropose    *bool              `json:"can_propose"`
		CanReview     *bool              `json:"can_review"`
		MaxElements   int                `json:"max_elements"`
		MaxSuppressed int                `json:"max_suppressed"`
		Vocabulary    string             `json:"vocabulary"`
		Encoding      string             `json:"encoding"`
		Base          int                `json:"base"`
		Nullable      bool               `json:"nullable"`
		Element       *EditSchemaElement `json:"element"`
	} `json:"fields"`
}

type v2Snapshot struct {
	FieldValues map[string]any `json:"field_values"`
	Values      map[string]any `json:"values"`
}

func (s v2Snapshot) values() map[string]any {
	if s.FieldValues != nil {
		return s.FieldValues
	}
	if s.Values != nil {
		return s.Values
	}
	return map[string]any{}
}

type v2Playtime struct {
	WorkID  json.RawMessage `json:"work_id"`
	Minutes int             `json:"minutes"`
}

type v2WorkState struct {
	WorkID     json.RawMessage `json:"work_id"`
	State      string          `json:"state"`
	Completion *string         `json:"completion"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type v2CoverVote struct {
	CoverID   json.RawMessage `json:"cover_id"`
	WorkID    json.RawMessage `json:"work_id"`
	Vote      string          `json:"vote"`
	VoteCount int             `json:"vote_count"`
	Voted     bool            `json:"voted"`
	Hash      string          `json:"hash"`
	ImageHash string          `json:"image_hash"`
}
