package catalogclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateMyProposal(ctx context.Context, accessToken string, req UserEditCreateRequest, idempotencyKey string) (*EditProposal, string, error) {
	var out v2Proposal
	etag, err := c.userV2JSONMeta(ctx, http.MethodPost, accessToken, "/v2/me/proposals", map[string]any{
		"entity_type": req.EntityType,
		"entity_id":   strconv.FormatInt(req.EntityID, 10),
		"patch":       req.Patch,
		"note":        req.Note,
	}, &out, idempotencyHeader(nil, idempotencyKey))
	if err != nil {
		return nil, etag, err
	}
	prop := out.proposal()
	return &prop, etag, nil
}

func (c *Client) GetMyProposal(ctx context.Context, accessToken string, id int64) (*EditProposal, string, error) {
	return c.getProposal(ctx, accessToken, "/v2/me/proposals/"+strconv.FormatInt(id, 10))
}

func (c *Client) GetModerationProposal(ctx context.Context, accessToken string, id int64) (*EditProposal, string, error) {
	return c.getProposal(ctx, accessToken, "/v2/moderation/proposals/"+strconv.FormatInt(id, 10))
}

func (c *Client) getProposal(ctx context.Context, accessToken, path string) (*EditProposal, string, error) {
	q := url.Values{"include": {"patch,amendments"}}
	var out v2Proposal
	etag, err := c.userV2JSONMeta(ctx, http.MethodGet, accessToken, path+"?"+q.Encode(), nil, &out, nil)
	if err != nil {
		return nil, etag, err
	}
	prop := out.proposal()
	return &prop, etag, nil
}

func (c *Client) WithdrawMyProposal(ctx context.Context, accessToken string, id int64, ifMatch string) (*EditProposal, string, error) {
	var out v2Proposal
	etag, err := c.userV2JSONMeta(ctx, http.MethodPatch, accessToken, "/v2/me/proposals/"+strconv.FormatInt(id, 10),
		map[string]any{"state": "withdrawn"}, &out, ifMatchHeader(ifMatch))
	if err != nil {
		return nil, etag, err
	}
	prop := out.proposal()
	return &prop, etag, nil
}

// AmendMyProposal answers the whole proposal, amendment chain included: catalog
// replies to an amendment with the proposal record, not the amendment row.
func (c *Client) AmendMyProposal(ctx context.Context, accessToken string, id int64, set map[string]any, unset []string, note, ifMatch, idempotencyKey string) (*EditProposal, string, error) {
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
	var out v2Proposal
	etag, err := c.userV2JSONMeta(ctx, http.MethodPost, accessToken,
		"/v2/me/proposals/"+strconv.FormatInt(id, 10)+"/amendments", body, &out,
		idempotencyHeader(ifMatchHeader(ifMatch), idempotencyKey))
	if err != nil {
		return nil, etag, err
	}
	prop := out.proposal()
	return &prop, etag, nil
}

// DecideProposal returns no proposal: catalog answers a decision record whose
// id is the proposal id, so the caller re-reads.
func (c *Client) DecideProposal(ctx context.Context, accessToken string, id int64, decision, note, ifMatch string) error {
	return c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/moderation/proposals/"+strconv.FormatInt(id, 10)+"/decisions",
		map[string]any{"decision": decision, "note": note}, nil, ifMatchHeader(ifMatch))
}

func (c *Client) RevertToRevision(ctx context.Context, accessToken string, revisionID int64, note, idempotencyKey string) (*EditProposal, string, error) {
	var out v2Proposal
	etag, err := c.userV2JSONMeta(ctx, http.MethodPost, accessToken, "/v2/moderation/reverts",
		map[string]any{"revision_id": strconv.FormatInt(revisionID, 10), "reason": note},
		&out, idempotencyHeader(nil, idempotencyKey))
	if err != nil {
		return nil, etag, err
	}
	prop := out.proposal()
	return &prop, etag, nil
}
