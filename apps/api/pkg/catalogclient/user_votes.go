package catalogclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var ErrInsufficientScope = errors.New("catalogclient: access token lacks the scope the call needs")

// One entry of an RFC 7807 body's errors[]. The editing engine names the field
// that failed in `pointer` (/patch/<key> for a locked or conflicting field, a
// bare /<key> for a validation failure), which is the only way the edit form can
// put the message next to the control instead of in a toast.
type ProblemFieldError struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
	Header    string `json:"header,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type DuplicateSuspect struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type UserAPIError struct {
	Status  int
	Code    int
	Message string
	// The problem body's own fields, kept whole. ProblemCode is the string code
	// (IMMUTABLE, NOT_PERMITTED, …), which is not the numeric Code above.
	ProblemCode string
	FieldErrors []ProblemFieldError
	RetryAfter  string
	Suspects    []DuplicateSuspect
}

func (e *UserAPIError) Error() string {
	return fmt.Sprintf("catalog user plane: status=%d code=%d %s", e.Status, e.Code, e.Message)
}

func (e *UserAPIError) Throttled() bool { return e.Status == http.StatusTooManyRequests }

type CoverVoteResult struct {
	CoverID   int64 `json:"cover_id"`
	VoteCount int64 `json:"vote_count"`
	Voted     bool  `json:"voted"`
}

func (c *Client) VoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodPut, accessToken, workID, coverID)
}

func (c *Client) UnvoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodDelete, accessToken, workID, coverID)
}

func (c *Client) coverVote(ctx context.Context, method, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	path := "/v2/me/cover-votes/" + strconv.FormatInt(coverID, 10)
	var body any
	if method == http.MethodPut {
		body = map[string]any{"vote": "up"}
	}
	var out v2CoverVote
	if err := c.userV2JSON(ctx, method, accessToken, path, body, &out, nil); err != nil {
		if errors.Is(err, ErrNotFound) && method == http.MethodDelete {
			return &CoverVoteResult{CoverID: coverID, Voted: false}, nil
		}
		return nil, err
	}
	id := parseFlexID(out.CoverID)
	if id == 0 {
		id = coverID
	}
	voted := method == http.MethodPut || out.Voted
	if method == http.MethodDelete {
		voted = out.Voted
	}
	return &CoverVoteResult{CoverID: id, VoteCount: int64(out.VoteCount), Voted: voted}, nil
}

func isScopeDenial(message string) bool {
	return strings.Contains(strings.ToLower(message), "scope")
}

type MyCoverVote struct {
	WorkID  int64
	CoverID int64
}

// MyCoverVotes is every cover the caller voted up. Catalog answers the whole
// list in one page (one ballot per work, no pagination) and needs catalog:edit.
func (c *Client) MyCoverVotes(ctx context.Context, accessToken string) ([]MyCoverVote, error) {
	var page v2List[v2CoverVote]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, "/v2/me/cover-votes", nil, &page, nil); err != nil {
		return nil, err
	}
	rows := page.rows()
	out := make([]MyCoverVote, 0, len(rows))
	for _, r := range rows {
		work, cover := parseFlexID(r.WorkID), parseFlexID(r.CoverID)
		if work <= 0 || cover <= 0 {
			return nil, fmt.Errorf("%w: /v2/me/cover-votes row without ids", ErrUpstream)
		}
		out = append(out, MyCoverVote{WorkID: work, CoverID: cover})
	}
	return out, nil
}
