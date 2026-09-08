package catalogclient

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type EditProposal struct {
	ID              int64           `json:"id"`
	EntityType      string          `json:"entity_type"`
	EntityID        int64           `json:"entity_id"`
	BaseRevisionSeq int             `json:"base_revision_seq"`
	Patch           map[string]any  `json:"patch"`
	EffectivePatch  map[string]any  `json:"effective_patch,omitempty"`
	ProposerUID     int64           `json:"proposer_uid"`
	Note            string          `json:"note"`
	Site            string          `json:"site"`
	Status          string          `json:"status"`
	DecidedByUID    *int64          `json:"decided_by_uid,omitempty"`
	DecidedAt       *time.Time      `json:"decided_at,omitempty"`
	DecisionNote    string          `json:"decision_note,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Amendments      []EditAmendment `json:"amendments,omitempty"`
}

type EditAmendment struct {
	ID         int64          `json:"id"`
	Seq        int            `json:"seq"`
	Set        map[string]any `json:"set,omitempty"`
	Unset      []string       `json:"unset,omitempty"`
	AmenderUID int64          `json:"amender_uid"`
	Note       string         `json:"note"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (a *EditAmendment) UnmarshalJSON(b []byte) error {
	var aux struct {
		ID         json.RawMessage `json:"id"`
		Seq        int             `json:"seq"`
		Set        map[string]any  `json:"set"`
		Unset      []string        `json:"unset"`
		AmenderUID json.RawMessage `json:"amender_uid"`
		Note       string          `json:"note"`
		CreatedAt  time.Time       `json:"created_at"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	a.ID = parseFlexID(aux.ID)
	a.Seq = aux.Seq
	a.Set = aux.Set
	a.Unset = aux.Unset
	a.AmenderUID = parseFlexID(aux.AmenderUID)
	a.Note = aux.Note
	a.CreatedAt = aux.CreatedAt
	return nil
}

type EditRevision struct {
	ID            int64          `json:"id"`
	Seq           int            `json:"seq"`
	Action        string         `json:"action"`
	ChangedFields []string       `json:"changed_fields"`
	Snapshot      map[string]any `json:"snapshot"`
	ActorUID      int64          `json:"actor_uid"`
	AmenderUID    *int64         `json:"amender_uid,omitempty"`
	ProposalID    *int64         `json:"proposal_id,omitempty"`
	Site          string         `json:"site"`
	CreatedAt     time.Time      `json:"created_at"`
	LegacyAction  string         `json:"legacy_action,omitempty"`
	LegacyNote    string         `json:"legacy_note,omitempty"`
	LegacyMinor   bool           `json:"legacy_minor,omitempty"`
	LegacyID      *int64         `json:"legacy_id,omitempty"`
}

func (r *EditRevision) UnmarshalJSON(b []byte) error {
	var aux struct {
		ID            json.RawMessage `json:"id"`
		Seq           int             `json:"seq"`
		Action        string          `json:"action"`
		ChangedFields []string        `json:"changed_fields"`
		Snapshot      map[string]any  `json:"snapshot"`
		ActorUID      json.RawMessage `json:"actor_uid"`
		AmenderUID    json.RawMessage `json:"amender_uid"`
		ProposalID    json.RawMessage `json:"proposal_id"`
		Site          string          `json:"site"`
		CreatedAt     time.Time       `json:"created_at"`
		LegacyAction  string          `json:"legacy_action"`
		LegacyNote    string          `json:"legacy_note"`
		LegacyMinor   bool            `json:"legacy_minor"`
		LegacyID      json.RawMessage `json:"legacy_id"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	r.ID = parseFlexID(aux.ID)
	r.Seq = aux.Seq
	r.Action = aux.Action
	r.ChangedFields = aux.ChangedFields
	r.Snapshot = aux.Snapshot
	r.ActorUID = parseFlexID(aux.ActorUID)
	r.Site = aux.Site
	r.CreatedAt = aux.CreatedAt
	r.LegacyAction = aux.LegacyAction
	r.LegacyNote = aux.LegacyNote
	r.LegacyMinor = aux.LegacyMinor
	if n := parseFlexID(aux.AmenderUID); n != 0 {
		r.AmenderUID = &n
	}
	if n := parseFlexID(aux.ProposalID); n != 0 {
		r.ProposalID = &n
	}
	if n := parseFlexID(aux.LegacyID); n != 0 {
		r.LegacyID = &n
	}
	return nil
}

type EditRevertResult struct {
	Proposal EditProposal `json:"proposal"`
	Revision EditRevision `json:"revision"`
}

// One member of a list field's element object, as the schema declares it.
type EditSchemaElementMember struct {
	Key        string `json:"key"`
	Type       string `json:"type"`
	Vocabulary string `json:"vocabulary,omitempty"`
	Base       int    `json:"base,omitempty"`
	Nullable   bool   `json:"nullable,omitempty"`
}

type EditSchemaElement struct {
	Type    string                    `json:"type"`
	Members []EditSchemaElementMember `json:"members,omitempty"`
}

type EditSchemaField struct {
	Key            string `json:"key"`
	Kind           string `json:"kind"`
	DiffHint       string `json:"diff_hint"`
	Deprecated     bool   `json:"deprecated,omitempty"`
	Locked         bool   `json:"locked"`
	CanPropose     bool   `json:"can_propose"`
	CanReview      bool   `json:"can_review"`
	WouldAutomerge bool   `json:"would_automerge"`

	// The value face of the schema. Dropping these is not cosmetic: an enum
	// field whose vocabulary and encoding do not reach the browser has no
	// options to render and the form degrades it to read-only, which is what
	// "编辑页只看得到三个标签卡" was.
	MaxElements   int                `json:"max_elements,omitempty"`
	MaxSuppressed int                `json:"max_suppressed,omitempty"`
	Vocabulary    string             `json:"vocabulary,omitempty"`
	Encoding      string             `json:"encoding,omitempty"`
	Base          int                `json:"base,omitempty"`
	Nullable      bool               `json:"nullable,omitempty"`
	Element       *EditSchemaElement `json:"element,omitempty"`
}

type EditSchema struct {
	EntityType string            `json:"entity_type"`
	Fields     []EditSchemaField `json:"fields"`
}

type EditFieldDiff struct {
	Key      string `json:"key"`
	Kind     string `json:"kind,omitempty"`
	DiffHint string `json:"diff_hint,omitempty"`
	From     any    `json:"from"`
	To       any    `json:"to"`
}

type EditDiff struct {
	FromSeq int             `json:"from_seq"`
	ToSeq   int             `json:"to_seq"`
	Fields  []EditFieldDiff `json:"fields"`
}

type EditCreateResult struct {
	Proposal EditProposal  `json:"proposal"`
	Merged   bool          `json:"merged"`
	Revision *EditRevision `json:"revision,omitempty"`
}

type EditProposalFilter struct {
	EntityType  string
	EntityID    int64
	Site        string
	ProposerUID int64
	Status      string
	Limit       int
}

const EntityTypeWork = "catalog.work"

const FieldKeyPrefix = EntityTypeWork + "."

func (c *Client) ListEditProposals(ctx context.Context, f EditProposalFilter) ([]EditProposal, error) {
	page, err := c.listPublicProposals(ctx, f)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (c *Client) CountEditProposals(ctx context.Context, f EditProposalFilter) (int64, error) {
	f.Limit = 1
	page, err := c.listPublicProposals(ctx, f)
	if err != nil {
		return 0, err
	}
	return page.Total, nil
}

type proposalListPage struct {
	Items []EditProposal `json:"items"`
	Total int64          `json:"total"`
}

func (c *Client) listPublicProposals(ctx context.Context, f EditProposalFilter) (*proposalListPage, error) {
	q := url.Values{}
	q.Set("include_total", "true")
	if object := objectFamily(f.EntityType); object != "" {
		q.Set("object", object)
	}
	if f.EntityID > 0 {
		q.Set("entity_id", strconv.FormatInt(f.EntityID, 10))
	}
	if f.Site != "" {
		q.Set("site", f.Site)
	}
	if f.ProposerUID > 0 {
		q.Set("proposer_uid", strconv.FormatInt(f.ProposerUID, 10))
	}
	if f.Status != "" {
		q.Set("state", f.Status)
	}
	rows, total, err := collectV2List[v2Proposal](ctx, c, "/v2/catalog/proposals", q, f.Limit)
	if err != nil {
		return nil, err
	}
	out := &proposalListPage{Items: make([]EditProposal, 0, len(rows))}
	if total != nil {
		out.Total = *total
	}
	for _, it := range rows {
		out.Items = append(out.Items, it.proposal())
	}
	return out, nil
}

func (c *Client) ListEditRevisions(ctx context.Context, entityType string, entityID int64, limit int) ([]EditRevision, error) {
	q := url.Values{}
	if object := objectFamily(entityType); object != "" {
		q.Set("object", object)
	}
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	rows, _, err := collectV2List[v2Revision](ctx, c, "/v2/catalog/revisions", q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]EditRevision, 0, len(rows))
	for _, it := range rows {
		out = append(out, it.revision())
	}
	return out, nil
}

func (c *Client) RevisionIDBySeq(ctx context.Context, entityType string, entityID int64, seq int) (int64, error) {
	found, err := c.revisionIDsBySeq(ctx, entityType, entityID, seq)
	if err != nil {
		return 0, err
	}
	id := found[seq]
	if id == 0 {
		return 0, ErrNotFound
	}
	return id, nil
}

func (c *Client) revisionIDsBySeq(ctx context.Context, entityType string, entityID int64, seqs ...int) (map[int]int64, error) {
	want := make(map[int]struct{}, len(seqs))
	for _, seq := range seqs {
		if seq > 0 {
			want[seq] = struct{}{}
		}
	}
	found := make(map[int]int64, len(want))
	if len(want) == 0 {
		return found, nil
	}
	q := url.Values{}
	if object := objectFamily(entityType); object != "" {
		q.Set("object", object)
	}
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	cursor := ""
	for pages := 0; pages < v2PageWalkMax && len(found) < len(want); pages++ {
		q.Set("limit", strconv.Itoa(v2PageMax))
		if cursor != "" {
			q.Set("cursor", cursor)
		} else {
			q.Del("cursor")
		}
		var page v2List[v2Revision]
		if err := c.appV2JSON(ctx, "/v2/catalog/revisions", q, &page); err != nil {
			return nil, err
		}
		for _, it := range page.rows() {
			if _, ok := want[it.Seq]; ok {
				found[it.Seq] = parseFlexID(it.ID)
			}
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = *page.NextCursor
	}
	return found, nil
}

func (c *Client) DiffEditRevisions(ctx context.Context, entityType string, entityID int64, fromSeq, toSeq int) (*EditDiff, error) {
	ids, err := c.revisionIDsBySeq(ctx, entityType, entityID, fromSeq, toSeq)
	if err != nil {
		return nil, err
	}
	fromID, toID := ids[fromSeq], ids[toSeq]
	if fromID == 0 || toID == 0 {
		return nil, ErrNotFound
	}
	q := url.Values{"include": {"diff"}, "diff_base": {strconv.FormatInt(fromID, 10)}}
	var rec v2Revision
	if err := c.appV2JSON(ctx, "/v2/catalog/revisions/"+strconv.FormatInt(toID, 10), q, &rec); err != nil {
		return nil, err
	}
	return &EditDiff{FromSeq: fromSeq, ToSeq: toSeq, Fields: rec.Diff}, nil
}
