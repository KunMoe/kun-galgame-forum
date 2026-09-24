package catalogclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type UserWorkSubmitRequest struct {
	ProductWorkID     int64           `json:"product_work_id,omitempty"`
	Fields            map[string]any  `json:"fields"`
	Released          *WorkSubmitDate `json:"released,omitempty"`
	ConfirmDuplicates bool            `json:"confirm_duplicates,omitempty"`
	IdempotencyKey    string          `json:"-"`
}

// The mint needs field_values. display_name alone is not a mint request at all:
// catalog answers `work_id, refs, site_work_id, or field_values is required.`
// and nothing is written. The v1 lane posted the whole request, so the v2
// rewrite dropped the field map silently and every "publish a galgame" the
// wizard sent between 2026-08-25 and 2026-09-09 died on that 422 — the tests
// stub this call and assert the path, never the body.
func (c *Client) SubmitWorkUser(ctx context.Context, accessToken string, req UserWorkSubmitRequest) (*WorkSubmitResult, error) {
	display, _ := req.Fields["catalog.work.display_name"].(string)
	body := map[string]any{"display_name": display}
	if len(req.Fields) > 0 {
		body["field_values"] = req.Fields
	}
	if req.ProductWorkID > 0 {
		body["site_work_id"] = strconv.FormatInt(req.ProductWorkID, 10)
	}
	if req.Released != nil {
		body["released"] = req.Released
	}
	if req.ConfirmDuplicates {
		body["confirm_duplicates"] = true
	}
	var out v2Claim
	if err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/me/claims", body, &out,
		idempotencyHeader(nil, req.IdempotencyKey)); err != nil {
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
		return c.patchMyClaim(ctx, accessToken, workID, "withdrawn")
	case ClaimActionPublish:
		return c.patchMyClaim(ctx, accessToken, workID, "live")
	case ClaimActionSubmit:
		return c.patchMyClaim(ctx, accessToken, workID, "pending")
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

func (c *Client) patchMyClaim(ctx context.Context, accessToken string, workID int64, state string) (*ClaimActionResult, error) {
	item, _, err := c.PatchMyClaim(ctx, accessToken, workID, state, "")
	if err != nil {
		return nil, err
	}
	return &ClaimActionResult{WorkID: item.WorkID, To: item.ClaimState}, nil
}

func (c *Client) PatchMyClaim(ctx context.Context, accessToken string, workID int64, state, ifMatch string) (*UserClaimItem, string, error) {
	var out v2Claim
	etag, err := c.userV2JSONMeta(ctx, http.MethodPatch, accessToken,
		"/v2/me/claims/"+strconv.FormatInt(workID, 10),
		map[string]any{"state": state}, &out, ifMatchHeader(ifMatch))
	if err != nil {
		return nil, etag, err
	}
	item := out.item()
	return &item, etag, nil
}

func (c *Client) DecideClaim(ctx context.Context, accessToken string, workID int64, decision, note, ifMatch string) error {
	return c.userV2JSON(ctx, http.MethodPost, accessToken,
		"/v2/moderation/claims/"+strconv.FormatInt(workID, 10)+"/decisions",
		map[string]any{"decision": decision, "note": note}, nil, ifMatchHeader(ifMatch))
}

func (c *Client) GetMyClaim(ctx context.Context, accessToken string, workID int64) (*UserClaimItem, string, error) {
	return c.getClaim(ctx, accessToken, "/v2/me/claims/"+strconv.FormatInt(workID, 10))
}

func (c *Client) GetModerationClaim(ctx context.Context, accessToken string, workID int64) (*UserClaimItem, string, error) {
	return c.getClaim(ctx, accessToken, "/v2/moderation/claims/"+strconv.FormatInt(workID, 10))
}

func (c *Client) getClaim(ctx context.Context, accessToken, path string) (*UserClaimItem, string, error) {
	var out v2Claim
	etag, err := c.userV2JSONMeta(ctx, http.MethodGet, accessToken, path, nil, &out, nil)
	if err != nil {
		return nil, etag, err
	}
	item := out.item()
	return &item, etag, nil
}

// DeleteMyClaim removes a draft the caller owns; catalog refuses anything else,
// including a live or pending claim (those have to be withdrawn to draft first).
// It soft-deletes the catalog work and, unlike every other claim transition,
// writes NO claim event — nothing downstream ever hears about it, so whatever
// the caller keeps locally is the caller's to clean up.
//
// The spec does not list If-Match on this operation the way it does on PATCH,
// but it does list 412 and 428 among the responses, so send the precondition.
func (c *Client) DeleteMyClaim(ctx context.Context, accessToken string, workID int64, ifMatch string) error {
	return c.userV2JSON(ctx, http.MethodDelete, accessToken,
		"/v2/me/claims/"+strconv.FormatInt(workID, 10), nil, nil, ifMatchHeader(ifMatch))
}

func (c *Client) MyClaims(ctx context.Context, accessToken string, f UserClaimFilter) (*UserClaimPage, error) {
	return c.listClaims(ctx, accessToken, "/v2/me/claims", f)
}

func (c *Client) ListModerationClaims(ctx context.Context, accessToken string, f UserClaimFilter) (*UserClaimPage, error) {
	return c.listClaims(ctx, accessToken, "/v2/moderation/claims", f)
}

func (c *Client) listClaims(ctx context.Context, accessToken, path string, f UserClaimFilter) (*UserClaimPage, error) {
	q := url.Values{}
	if len(f.ClaimStates) > 0 {
		q.Set("claim_state", strings.Join(f.ClaimStates, ","))
	}
	if f.Before > 0 {
		q.Set("before", strconv.FormatInt(f.Before, 10))
	}
	if f.Cursor != "" {
		q.Set("cursor", f.Cursor)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.Kind != "" {
		q.Set("kind", f.Kind)
	}
	if f.IncludeTotal {
		q.Set("include_total", "true")
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var page v2List[v2Claim]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &page, nil); err != nil {
		return nil, err
	}
	rows := page.rows()
	out := &UserClaimPage{Items: make([]UserClaimItem, 0, len(rows)), NextCursor: page.cursor()}
	if page.Total != nil {
		out.Total = *page.Total
	}
	for _, it := range rows {
		out.Items = append(out.Items, it.item())
	}
	return out, nil
}
