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

type EditProposalFilter struct {
	EntityType  string
	EntityID    int64
	Site        string
	ProposerUID int64
	Status      string
	Cursor      string
	Limit       int
}

const EntityTypeWork = "catalog.work"

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

func publicProposalQuery(f EditProposalFilter) url.Values {
	q := url.Values{}
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
	return q
}

func (c *Client) listPublicProposals(ctx context.Context, f EditProposalFilter) (*proposalListPage, error) {
	q := publicProposalQuery(f)
	q.Set("include_total", "true")
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
