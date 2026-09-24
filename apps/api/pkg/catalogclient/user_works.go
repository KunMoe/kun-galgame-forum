package catalogclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// MyWorksMax is catalog's cap on work_ids for GET /v2/me/works (400
// TOO_MANY_IDS above it).
const MyWorksMax = 100

// MyWork is the caller's own relation to one work, as GET /v2/me/works
// answers it: the folders holding it, their playtime (the max across apps) and
// their work state. Playtime and WorkState are nil when there is none.
type MyWork struct {
	WorkID    int64
	FolderIDs []int64
	Playtime  *PlaytimeSelf
	WorkState *WorkStateRecord
}

type v2MyWork struct {
	WorkID    json.RawMessage   `json:"work_id"`
	FolderIDs []json.RawMessage `json:"folder_ids"`
	Playtime  *struct {
		Minutes int `json:"minutes"`
	} `json:"playtime"`
	WorkState *v2WorkState `json:"work_state"`
}

// MyWorks answers every id once, in request order; catalog answers unknown
// works too, with no folders and no playtime or state. It needs folder:read.
func (c *Client) MyWorks(ctx context.Context, token string, workIDs []int64) ([]MyWork, error) {
	want := make([]int64, 0, len(workIDs))
	seen := map[int64]bool{}
	for _, id := range workIDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			want = append(want, id)
		}
	}
	out := make([]MyWork, 0, len(want))
	for _, chunk := range chunkInt64s(want, MyWorksMax) {
		if len(chunk) == 0 {
			continue
		}
		q := url.Values{}
		q.Set("work_ids", joinInt64s(chunk))
		var got v2List[v2MyWork]
		if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/works?"+q.Encode(), nil, &got, nil); err != nil {
			return nil, err
		}
		for _, row := range got.rows() {
			w, err := row.view()
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		}
	}
	return out, nil
}

func (r v2MyWork) view() (MyWork, error) {
	w := MyWork{WorkID: parseFlexID(r.WorkID), FolderIDs: make([]int64, 0, len(r.FolderIDs))}
	if w.WorkID <= 0 {
		return MyWork{}, fmt.Errorf("%w: /v2/me/works item without a work_id", ErrUpstream)
	}
	for _, raw := range r.FolderIDs {
		id := parseFlexID(raw)
		if id <= 0 {
			return MyWork{}, fmt.Errorf("%w: /v2/me/works folder id %s", ErrUpstream, raw)
		}
		w.FolderIDs = append(w.FolderIDs, id)
	}
	if r.Playtime != nil {
		w.Playtime = &PlaytimeSelf{WorkID: w.WorkID, Minutes: r.Playtime.Minutes}
	}
	if r.WorkState != nil {
		rec := workStateRecord(*r.WorkState)
		rec.WorkID = w.WorkID
		w.WorkState = &rec
	}
	return w, nil
}
