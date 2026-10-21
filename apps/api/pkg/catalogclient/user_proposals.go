package catalogclient

import (
	"context"
	"net/http"
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

// DecideProposal answers the state catalog read back after the decision, which
// can differ from the verb: a suppression rule can close a merge as declined.
func (c *Client) DecideProposal(ctx context.Context, accessToken string, id int64, decision, note, ifMatch string) (string, error) {
	var out struct {
		ToState string `json:"to_state"`
	}
	err := c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/moderation/proposals/"+strconv.FormatInt(id, 10)+"/decisions",
		map[string]any{"decision": decision, "note": note}, &out, ifMatchHeader(ifMatch))
	return out.ToState, err
}
