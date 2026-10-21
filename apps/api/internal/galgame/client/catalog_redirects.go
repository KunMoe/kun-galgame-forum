package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

// CatalogRedirectsLimit is the feed's own page ceiling, shared by every v2
// collection. Asking for more answers 400 LIMIT_TOO_LARGE rather than clamping.
const CatalogRedirectsLimit = 100

type CatalogRedirect struct {
	OldID     int64
	CurrentID int64
	Revoked   bool
}

type CatalogRedirectPage struct {
	Items      []CatalogRedirect
	NextCursor string
}

// CatalogRedirects reads catalog's merge feed: every id that was merged away,
// with the id that replaced it, in the order the redirects last changed. A
// redirect that is flattened, re-pointed or revoked comes round again at the
// tail; a revoked one has its merge undone and the old id live again. An empty
// cursor starts at the beginning of the whole history, so the first drain
// replays every merge catalog has ever executed and clearing the stored cursor
// is how the forum's redirect ledger is rebuilt.
//
// No nsfw parameter, unlike the changes feed: this reads catalog_redirect, which
// is pure id history with no population or visibility notion attached.
//
// The last page carries no next_cursor, so the stored cursor stays on the page
// before it and that page is read again next tick. Re-reading is free — the
// fold is keyed on a local row that the previous pass already deleted.
func (c *GalgameClient) CatalogRedirects(ctx context.Context, cursor string, limit int) (*CatalogRedirectPage, *errors.AppError) {
	q := url.Values{
		"limit":  {strconv.Itoa(min(max(limit, 1), CatalogRedirectsLimit))},
		"object": {"work"},
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	data, appErr := c.CatalogGet(ctx, "/catalog/redirects", q)
	if appErr != nil {
		return nil, appErr
	}
	// old_id and current_id stay strings all the way here. rewriteV2Object
	// turns v2's string ids back into numbers, which is why the changes feed
	// decodes `id` as an int64, but its key list is id/work_id/canonical_id/
	// character_id/person_id/from/to — not these two. Declaring them int64
	// fails the whole page with "cannot unmarshal string", not just the field.
	var parsed struct {
		Items []struct {
			TargetObject string `json:"target_object"`
			OldID        string `json:"old_id"`
			CurrentID    string `json:"current_id"`
			Revoked      bool   `json:"revoked"`
		} `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 合并信道响应失败")
	}
	page := &CatalogRedirectPage{
		Items:      make([]CatalogRedirect, 0, len(parsed.Items)),
		NextCursor: parsed.NextCursor,
	}
	for _, it := range parsed.Items {
		oldID, errOld := strconv.ParseInt(it.OldID, 10, 64)
		currentID, errCurrent := strconv.ParseInt(it.CurrentID, 10, 64)
		if it.TargetObject != "work" || errOld != nil || errCurrent != nil || oldID <= 0 || currentID <= 0 {
			continue
		}
		page.Items = append(page.Items, CatalogRedirect{OldID: oldID, CurrentID: currentID, Revoked: it.Revoked})
	}
	return page, nil
}
