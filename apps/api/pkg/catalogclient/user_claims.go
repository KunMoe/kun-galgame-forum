package catalogclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type UserWorkSubmitRequest struct {
	ProductWorkID int64           `json:"product_work_id,omitempty"`
	Fields        map[string]any  `json:"fields"`
	Released      *WorkSubmitDate `json:"released,omitempty"`
}

func (c *Client) SubmitWorkUser(ctx context.Context, accessToken string, req UserWorkSubmitRequest) (*WorkSubmitResult, error) {
	display := ""
	if req.Fields != nil {
		if v, ok := req.Fields["catalog.work.display_name"].(string); ok {
			display = v
		}
	}
	body := map[string]any{"display_name": display}
	if req.ProductWorkID > 0 {
		body["site_work_id"] = strconv.FormatInt(req.ProductWorkID, 10)
	}
	if len(req.Fields) > 0 {
		body["field_values"] = req.Fields
	}
	if req.Released != nil {
		body["released"] = req.Released
	}
	var out v2Claim
	if err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/me/claims", body, &out, nil); err != nil {
		return nil, err
	}
	id := parseFlexID(out.ID)
	return &WorkSubmitResult{WorkID: id, ProductWorkID: id, ClaimState: out.State}, nil
}

type UserClaimActionRequest struct {
	ProductWorkID int64  `json:"product_work_id,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func (c *Client) ActOnClaimUser(ctx context.Context, accessToken string, workID int64, action string, req UserClaimActionRequest) (*ClaimActionResult, error) {
	id := strconv.FormatInt(workID, 10)
	switch action {
	case ClaimActionClaim:
		body := map[string]any{"work_id": id}
		if req.ProductWorkID > 0 {
			body["site_work_id"] = strconv.FormatInt(req.ProductWorkID, 10)
		}
		var out v2Claim
		if err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/me/claims", body, &out, nil); err != nil {
			return nil, err
		}
		return &ClaimActionResult{WorkID: parseFlexID(out.ID), To: out.State}, nil
	case ClaimActionWithdraw:
		return c.patchMyClaim(ctx, accessToken, id, "withdrawn")
	case ClaimActionPublish:
		return c.patchMyClaim(ctx, accessToken, id, "live")
	case ClaimActionSubmit:
		return c.patchMyClaim(ctx, accessToken, id, "pending")
	case ClaimActionApprove, ClaimActionDecline, ClaimActionBan, ClaimActionUnban:
		decision := action
		if action == ClaimActionApprove {
			decision = "approve"
		}
		if err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/moderation/claims/"+id+"/decisions",
			map[string]any{"decision": decision, "note": req.Reason}, nil, ifMatchStar()); err != nil {
			return nil, err
		}
		to := "live"
		switch action {
		case ClaimActionDecline:
			to = "declined"
		case ClaimActionBan:
			to = "hidden"
		}
		return &ClaimActionResult{WorkID: workID, To: to}, nil
	default:
		return nil, &UserAPIError{Status: http.StatusBadRequest, Message: "unknown claim action"}
	}
}

func (c *Client) patchMyClaim(ctx context.Context, accessToken, workID, state string) (*ClaimActionResult, error) {
	var out v2Claim
	if err := c.userV2JSON(ctx, http.MethodPatch, accessToken, "/v2/me/claims/"+workID,
		map[string]any{"state": state}, &out, ifMatchStar()); err != nil {
		return nil, err
	}
	return &ClaimActionResult{WorkID: parseFlexID(out.ID), To: out.State}, nil
}

// DeleteMyClaim removes a draft the caller owns; catalog refuses anything else,
// including a live or pending claim (those have to be withdrawn to draft first).
// It soft-deletes the catalog work and, unlike every other claim transition,
// writes NO claim event — nothing downstream ever hears about it, so whatever
// the caller keeps locally is the caller's to clean up.
//
// The spec does not list If-Match on this operation the way it does on PATCH,
// but it does list 412 and 428 among the responses, so send the precondition.
func (c *Client) DeleteMyClaim(ctx context.Context, accessToken string, workID int64) error {
	return c.userV2JSON(ctx, http.MethodDelete, accessToken,
		"/v2/me/claims/"+strconv.FormatInt(workID, 10), nil, nil, ifMatchStar())
}

func (c *Client) MyClaims(ctx context.Context, accessToken string, f UserClaimFilter) (*UserClaimPage, error) {
	q := url.Values{}
	if len(f.ClaimStates) > 0 {
		q.Set("claim_state", strings.Join(f.ClaimStates, ","))
	}
	if f.Before > 0 {
		q.Set("before", strconv.FormatInt(f.Before, 10))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.Kind != "" {
		q.Set("kind", f.Kind)
	}
	path := "/v2/me/claims"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var page v2List[v2Claim]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &page, nil); err != nil {
		return nil, err
	}
	rows := page.rows()
	out := &UserClaimPage{Items: make([]UserClaimItem, 0, len(rows))}
	if page.Total != nil {
		out.Total = *page.Total
	}
	for _, it := range rows {
		out.Items = append(out.Items, it.item())
	}
	return out, nil
}
