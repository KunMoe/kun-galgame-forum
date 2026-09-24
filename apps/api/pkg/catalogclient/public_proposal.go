package catalogclient

import (
	"context"
	"net/url"
	"strconv"
)

// GetPublicProposal reads catalog's transparency face with the application key:
// state, target, proposer and site, never the patch.
func (c *Client) GetPublicProposal(ctx context.Context, id int64) (*EditProposal, error) {
	var out v2Proposal
	if err := c.appV2JSON(ctx, "/v2/catalog/proposals/"+strconv.FormatInt(id, 10), url.Values{}, &out); err != nil {
		return nil, err
	}
	prop := out.proposal()
	return &prop, nil
}
