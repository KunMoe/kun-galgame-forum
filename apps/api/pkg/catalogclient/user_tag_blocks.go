package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

const (
	TagBlockTree  = "tree"
	TagBlockExact = "exact"

	TagBlocksMax = 100
)

type TagBlock struct {
	TagID     int64
	Mode      string
	CreatedAt string
}

type v2TagBlock struct {
	TagID     json.RawMessage `json:"tag_id"`
	Mode      string          `json:"mode"`
	CreatedAt string          `json:"created_at"`
}

func (r v2TagBlock) view() TagBlock {
	return TagBlock{TagID: parseFlexID(r.TagID), Mode: r.Mode, CreatedAt: r.CreatedAt}
}

func (c *Client) MyTagBlocks(ctx context.Context, accessToken string) ([]TagBlock, error) {
	var got v2List[v2TagBlock]
	path := "/v2/me/blocked-tags?limit=" + strconv.Itoa(TagBlocksMax)
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &got, nil); err != nil {
		return nil, err
	}
	out := make([]TagBlock, 0, len(got.Items))
	for _, row := range got.Items {
		if b := row.view(); b.TagID > 0 {
			out = append(out, b)
		}
	}
	return out, nil
}

func (c *Client) PutTagBlock(ctx context.Context, accessToken string, tagID int64, mode string) (*TagBlock, error) {
	path := "/v2/me/blocked-tags/" + strconv.FormatInt(tagID, 10)
	body := struct {
		Mode string `json:"mode"`
	}{Mode: mode}
	var out v2TagBlock
	if err := c.userV2JSON(ctx, http.MethodPut, accessToken, path, body, &out, nil); err != nil {
		return nil, err
	}
	b := out.view()
	return &b, nil
}

func (c *Client) DeleteTagBlock(ctx context.Context, accessToken string, tagID int64) error {
	path := "/v2/me/blocked-tags/" + strconv.FormatInt(tagID, 10)
	_, _, err := c.userV2Do(ctx, http.MethodDelete, accessToken, path, nil, nil)
	return err
}

func (c *Client) MyTagBlock(ctx context.Context, accessToken string, tagID int64) (*TagBlock, error) {
	path := "/v2/me/blocked-tags/" + strconv.FormatInt(tagID, 10)
	var out v2TagBlock
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	b := out.view()
	return &b, nil
}
