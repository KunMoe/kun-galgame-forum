package catalogclient

import (
	"context"
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
	EntityType   string
	EntityID     int64
	Status       string
	Cursor       string
	Limit        int
	Mine         bool
	IncludeTotal bool
}

type ProposalPage struct {
	Items      []EditProposal
	NextCursor string
	Total      int64
}

func (c *Client) ListEditProposalsUserPage(ctx context.Context, accessToken string, f UserEditProposalFilter) (*ProposalPage, error) {
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
	if f.Cursor != "" {
		q.Set("cursor", f.Cursor)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.IncludeTotal {
		q.Set("include_total", "true")
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
	return proposalPageOf(page), nil
}

func proposalPageOf(page v2List[v2Proposal]) *ProposalPage {
	rows := page.rows()
	out := &ProposalPage{Items: make([]EditProposal, 0, len(rows)), NextCursor: page.cursor()}
	if page.Total != nil {
		out.Total = *page.Total
	}
	for _, it := range rows {
		out.Items = append(out.Items, it.proposal())
	}
	return out
}
