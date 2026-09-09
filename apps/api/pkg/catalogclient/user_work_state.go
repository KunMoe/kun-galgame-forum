package catalogclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	WorkStateWish    = "wish"
	WorkStateDoing   = "doing"
	WorkStateDone    = "done"
	WorkStateOnHold  = "on_hold"
	WorkStateDropped = "dropped"
)

type WorkStateRecord struct {
	WorkID     int64   `json:"work_id"`
	State      string  `json:"state"`
	Completion *string `json:"completion"`
}

func workStateRecord(out v2WorkState) WorkStateRecord {
	return WorkStateRecord{
		WorkID:     parseFlexID(out.WorkID),
		State:      out.State,
		Completion: out.Completion,
	}
}

func (c *Client) MyWorkState(ctx context.Context, accessToken string, workID int64) (*WorkStateRecord, error) {
	path := "/v2/me/work-states/" + strconv.FormatInt(workID, 10)
	var out v2WorkState
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if parseFlexID(out.WorkID) == 0 && out.State == "" {
		return nil, nil
	}
	rec := workStateRecord(out)
	return &rec, nil
}

func (c *Client) PutWorkState(ctx context.Context, accessToken string, workID int64, state string, completion *string) (*WorkStateRecord, error) {
	path := "/v2/me/work-states/" + strconv.FormatInt(workID, 10)
	body := struct {
		State      string  `json:"state"`
		Completion *string `json:"completion,omitempty"`
	}{
		State:      state,
		Completion: completion,
	}
	var out v2WorkState
	if err := c.userV2JSON(ctx, http.MethodPut, accessToken, path, body, &out, nil); err != nil {
		return nil, err
	}
	rec := workStateRecord(out)
	return &rec, nil
}

func (c *Client) MyWorkStates(ctx context.Context, accessToken string, workIDs []int64) (map[int64]WorkStateRecord, error) {
	out := make(map[int64]WorkStateRecord, len(workIDs))
	if len(workIDs) == 0 {
		return out, nil
	}
	ids := make([]string, len(workIDs))
	for i, id := range workIDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	q := url.Values{}
	q.Set("work_ids", strings.Join(ids, ","))
	var page v2List[v2WorkState]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, "/v2/me/work-states?"+q.Encode(), nil, &page, nil); err != nil {
		return nil, err
	}
	for _, it := range page.rows() {
		rec := workStateRecord(it)
		if rec.WorkID == 0 {
			continue
		}
		out[rec.WorkID] = rec
	}
	return out, nil
}

func (c *Client) DeleteWorkState(ctx context.Context, accessToken string, workID int64) error {
	path := "/v2/me/work-states/" + strconv.FormatInt(workID, 10)
	_, _, err := c.userV2Do(ctx, http.MethodDelete, accessToken, path, nil, nil)
	if err != nil && errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) ListMyWorkStates(ctx context.Context, accessToken, cursor string,
	limit int) ([]WorkStateRecord, string, error) {

	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/v2/me/work-states"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out v2List[v2WorkState]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		return nil, "", err
	}
	rows := out.rows()
	items := make([]WorkStateRecord, 0, len(rows))
	for _, it := range rows {
		items = append(items, workStateRecord(it))
	}
	return items, out.cursor(), nil
}
