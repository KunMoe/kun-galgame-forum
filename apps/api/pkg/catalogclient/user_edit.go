package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type UserEditCreateRequest struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	Patch      map[string]any `json:"patch"`
	Note       string         `json:"note,omitempty"`
}

func (c *Client) CreateEditProposalUser(ctx context.Context, accessToken string, req UserEditCreateRequest) (*EditCreateResult, error) {
	raw, _, err := c.userV2Do(ctx, http.MethodPost, accessToken, "/v2/me/proposals", map[string]any{
		"entity_type": req.EntityType,
		"entity_id":   strconv.FormatInt(req.EntityID, 10),
		"patch":       req.Patch,
		"note":        req.Note,
	}, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Code     int             `json:"code"`
		Data     json.RawMessage `json:"data"`
		Object   string          `json:"object"`
		Merged   bool            `json:"merged"`
		Proposal json.RawMessage `json:"proposal"`
		Revision *EditRevision   `json:"revision"`
	}
	_ = json.Unmarshal(raw, &env)
	payload := raw
	if env.Object == "" && len(bytes.TrimSpace(env.Data)) > 0 {
		payload = env.Data
		_ = json.Unmarshal(payload, &env)
	}
	if len(bytes.TrimSpace(env.Proposal)) > 0 {
		var prop EditProposal
		if err := json.Unmarshal(env.Proposal, &prop); err != nil {
			return nil, err
		}
		return &EditCreateResult{Proposal: prop, Merged: env.Merged, Revision: env.Revision}, nil
	}
	var out v2Proposal
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	prop := out.proposal()
	return &EditCreateResult{Proposal: prop, Merged: prop.Status == "merged" || env.Merged, Revision: env.Revision}, nil
}

func (c *Client) WithdrawEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	var out v2Proposal
	err := c.userV2JSON(ctx, http.MethodPatch, accessToken, "/v2/me/proposals/"+strconv.FormatInt(id, 10),
		map[string]any{"state": "withdrawn"}, &out, ifMatchStar())
	if err != nil {
		return nil, err
	}
	prop := out.proposal()
	return &prop, nil
}

func (c *Client) GetEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	var out v2Proposal
	q := url.Values{"include": {"patch,amendments"}}
	err := c.userV2JSON(ctx, http.MethodGet, accessToken,
		"/v2/moderation/proposals/"+strconv.FormatInt(id, 10)+"?"+q.Encode(),
		nil, &out, nil)
	if err != nil {
		return nil, err
	}
	prop := out.proposal()
	return &prop, nil
}

func (c *Client) AmendEditProposalUser(ctx context.Context, accessToken string, id int64, set map[string]any, unset []string, note string) (*EditAmendment, error) {
	body := map[string]any{}
	if len(set) > 0 {
		body["set"] = set
	}
	if len(unset) > 0 {
		body["unset"] = unset
	}
	if note != "" {
		body["note"] = note
	}
	var out EditAmendment
	err := c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/me/proposals/"+strconv.FormatInt(id, 10)+"/amendments", body, &out,
		ifMatchStar())
	if err != nil {
		return nil, err
	}
	if out.Set == nil {
		out.Set = set
	}
	if out.Unset == nil {
		out.Unset = unset
	}
	if out.Note == "" {
		out.Note = note
	}
	return &out, nil
}

func (c *Client) MergeEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditRevision, error) {
	var out EditRevision
	err := c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/moderation/proposals/"+strconv.FormatInt(id, 10)+"/decisions",
		map[string]any{"decision": "merge", "note": note}, &out,
		ifMatchStar())
	if err != nil {
		return nil, err
	}
	if out.Action == "" {
		out.Action = "merged"
	}
	return &out, nil
}

func (c *Client) DeclineEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditProposal, error) {
	var out v2Proposal
	err := c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/moderation/proposals/"+strconv.FormatInt(id, 10)+"/decisions",
		map[string]any{"decision": "decline", "note": note}, &out,
		ifMatchStar())
	if err != nil {
		return nil, err
	}
	prop := out.proposal()
	if prop.ID == 0 {
		prop.ID = id
	}
	if prop.Status == "" || prop.Status == "open" {
		prop.Status = "declined"
	}
	if prop.DecisionNote == "" {
		prop.DecisionNote = note
	}
	return &prop, nil
}

func (c *Client) RevertEditEntityUser(ctx context.Context, accessToken string, revisionID int64, note string) (*EditRevertResult, error) {
	var out v2Proposal
	err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/moderation/reverts",
		map[string]any{"revision_id": strconv.FormatInt(revisionID, 10), "reason": note},
		&out, nil)
	if err != nil {
		return nil, err
	}
	return &EditRevertResult{Proposal: out.proposal()}, nil
}

func (c *Client) GetEditSchemaUser(ctx context.Context, accessToken, entityType string, entityID int64) (*EditSchema, error) {
	object := "work"
	if strings.HasPrefix(entityType, "catalog.") {
		object = strings.TrimPrefix(entityType, "catalog.")
		if object == "label" {
			object = "company"
		}
	}
	var schema v2Schema
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, "/v2/catalog/schemas/"+object, nil, &schema, nil); err != nil {
		return nil, err
	}
	out := &EditSchema{EntityType: schema.EntityType}
	if out.EntityType == "" {
		out.EntityType = entityType
	}
	for _, f := range schema.Fields {
		kind := f.FieldType
		if kind == "" {
			kind = f.Kind
		}
		canPropose, canReview := !f.Deprecated, true
		if f.CanPropose != nil {
			canPropose = *f.CanPropose
		}
		if f.CanReview != nil {
			canReview = *f.CanReview
		}
		locked := !canPropose && !canReview && !f.Deprecated
		out.Fields = append(out.Fields, EditSchemaField{
			Key:           f.Key,
			Kind:          kind,
			DiffHint:      f.DiffHint,
			Deprecated:    f.Deprecated,
			Locked:        locked,
			CanPropose:    canPropose,
			CanReview:     canReview,
			MaxElements:   f.MaxElements,
			MaxSuppressed: f.MaxSuppressed,
			Vocabulary:    f.Vocabulary,
			Encoding:      f.Encoding,
			Base:          f.Base,
			Nullable:      f.Nullable,
			Element:       f.Element,
		})
	}
	return out, nil
}

func (c *Client) EditSnapshotUser(ctx context.Context, accessToken, entityType string, entityID int64) (map[string]any, error) {
	object := "work"
	if strings.HasPrefix(entityType, "catalog.") {
		object = strings.TrimPrefix(entityType, "catalog.")
		if object == "label" {
			object = "company"
		}
	}
	var snap v2Snapshot
	err := c.userV2JSON(ctx, http.MethodGet, accessToken,
		"/v2/moderation/snapshots/"+object+"/"+strconv.FormatInt(entityID, 10), nil, &snap, nil)
	if err != nil {
		return nil, err
	}
	return snap.values(), nil
}

type UserEditProposalFilter struct {
	EntityType string
	EntityID   int64
	Status     string
	Limit      int
	Mine       bool
}

func (c *Client) ListEditProposalsUser(ctx context.Context, accessToken string, f UserEditProposalFilter) ([]EditProposal, error) {
	q := url.Values{}
	if f.EntityType != "" {
		q.Set("entity_type", f.EntityType)
	}
	if f.EntityID > 0 {
		q.Set("entity_id", strconv.FormatInt(f.EntityID, 10))
	}
	if f.Status != "" {
		q.Set("state", f.Status)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	path := "/v2/me/proposals"
	if !f.Mine {
		path = "/v2/moderation/proposals"
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var page v2List[v2Proposal]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &page, nil); err != nil {
		return nil, err
	}
	rows := page.rows()
	out := make([]EditProposal, 0, len(rows))
	for _, it := range rows {
		out = append(out, it.proposal())
	}
	return out, nil
}

func (c *Client) WorkCoversUser(ctx context.Context, accessToken string, workID int64) ([]CoverTally, error) {
	var page v2List[struct {
		ID        json.RawMessage `json:"id"`
		VoteCount int             `json:"vote_count"`
		Hash      string          `json:"hash"`
		ImageHash string          `json:"image_hash"`
		Voted     bool            `json:"voted"`
	}]
	err := c.userV2JSON(ctx, http.MethodGet, accessToken,
		"/v2/catalog/works/"+strconv.FormatInt(workID, 10)+"/covers?nsfw=true", nil, &page, nil)
	if err != nil {
		return nil, err
	}
	rows := page.rows()
	out := make([]CoverTally, 0, len(rows))
	for _, it := range rows {
		hash := it.Hash
		if hash == "" {
			hash = it.ImageHash
		}
		out = append(out, CoverTally{ID: parseFlexID(it.ID), ImageHash: hash, VoteCount: it.VoteCount, Voted: it.Voted})
	}
	return out, nil
}
