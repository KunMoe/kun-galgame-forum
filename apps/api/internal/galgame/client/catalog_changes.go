package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

// CatalogChangesLimit is the feed's own page ceiling. A larger limit answers
// 400 LIMIT_TOO_LARGE rather than clamping.
const CatalogChangesLimit = 100

type CatalogChange struct {
	ID   int64
	Gone bool
}

type CatalogChangePage struct {
	Items      []CatalogChange
	NextCursor string
}

// CatalogChanges reads catalog's mirror channel. Every write that moves a work's
// claim state, its display axis (display_nsfw / content_rating) or its existence
// bumps updated_at and surfaces the id here, oldest-updated first. An empty
// cursor starts at the head of the inventory, so the first drain is a full
// re-sync and clearing the stored cursor is how the mirror is rebuilt.
//
// nsfw=true is sent even though the feed currently answers the same rows without
// it. The parameter is documented as "false or absent hides r18", and a 2000-row
// sample of this feed is 1993 r18 works, so on the day the gate starts being
// enforced here a poller without the flag walks past almost the whole population
// while still reporting success.
func (c *GalgameClient) CatalogChanges(ctx context.Context, cursor string, limit int) (*CatalogChangePage, *errors.AppError) {
	q := url.Values{
		"limit": {strconv.Itoa(min(max(limit, 1), CatalogChangesLimit))},
		"nsfw":  {"true"},
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	data, appErr := c.CatalogGet(ctx, "/catalog/changes", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		Items []struct {
			ID           int64  `json:"id"`
			TargetObject string `json:"target_object"`
			Gone         bool   `json:"gone"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 变更信道响应失败")
	}
	page := &CatalogChangePage{
		Items:      make([]CatalogChange, 0, len(parsed.Items)),
		NextCursor: parsed.NextCursor,
	}
	for _, it := range parsed.Items {
		if it.TargetObject != "work" || it.ID <= 0 {
			continue
		}
		page.Items = append(page.Items, CatalogChange{ID: it.ID, Gone: it.Gone})
	}
	return page, nil
}

// MirrorByCatalogIDs is the mirror channel's half of MirrorByGIDs: the feed
// names catalog ids, and the hydrated row carries the claim block, so the gid
// comes back with the fields and the gid cache is never involved.
//
// Only a kungal claim may name a local row. 10,289 of the forum's 11,562 gids
// are ALSO the catalog id of a different work, so the catalog-id fallback that
// gid() takes for an unclaimed row would write one game's fields onto another
// game's row on nearly every page.
func (c *GalgameClient) MirrorByCatalogIDs(ctx context.Context, ids []int64) (map[int]CatalogMirror, *errors.AppError) {
	out := make(map[int]CatalogMirror, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, "", "all")
	if appErr != nil {
		return nil, appErr
	}
	for i := range rows {
		row := &rows[i]
		if !row.isRenderable() || row.Claim == nil {
			continue
		}
		if !isKungalClaim(row.Claim.Site) || row.Claim.SiteWorkID <= 0 {
			continue
		}
		out[row.Claim.SiteWorkID] = mirrorOf(row)
	}
	return out, nil
}
