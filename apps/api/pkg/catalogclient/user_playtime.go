package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

// Catalog refuses anything above this, and its aggregate job ignores anything
// below PlaytimeMinutesFloor — a report under the floor is stored but never
// reaches the public median, which is how a user withdraws one.
const (
	PlaytimeMinutesMax   = 60_000
	PlaytimeMinutesFloor = 10
)

type PlaytimeReport struct {
	Minutes int `json:"minutes"`
}

type PlaytimeRecord struct {
	WorkID  int64 `json:"work_id"`
	Minutes int   `json:"minutes"`
}

type PlaytimeSelf struct {
	WorkID  int64 `json:"work_id"`
	Minutes int   `json:"minutes"`
}

func (c *Client) ReportPlaytime(ctx context.Context, accessToken string, workID int64,
	report PlaytimeReport) (*PlaytimeRecord, error) {

	var out v2Playtime
	path := "/v2/me/playtimes/" + strconv.FormatInt(workID, 10)
	if err := c.userV2JSON(ctx, http.MethodPut, accessToken, path, map[string]any{"minutes": report.Minutes}, &out, nil); err != nil {
		return nil, err
	}
	return &PlaytimeRecord{WorkID: parseFlexID(out.WorkID), Minutes: out.Minutes}, nil
}

// MyPlaytime answers (nil, nil) when the user has never reported on this work.
// Upstream calls that a 200 with a null payload, not a 404.
func (c *Client) MyPlaytime(ctx context.Context, accessToken string, workID int64) (*PlaytimeSelf, error) {
	path := "/v2/me/playtimes/" + strconv.FormatInt(workID, 10)
	raw, _, err := c.userV2Do(ctx, http.MethodGet, accessToken, path, nil, nil)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 || string(trim) == "null" {
		return nil, nil
	}
	var env struct {
		Code   int             `json:"code"`
		Data   json.RawMessage `json:"data"`
		Object string          `json:"object"`
	}
	if json.Unmarshal(trim, &env) == nil && env.Object == "" {
		data := bytes.TrimSpace(env.Data)
		if len(data) == 0 || string(data) == "null" {
			return nil, nil
		}
		trim = data
	}
	var out v2Playtime
	if err := json.Unmarshal(trim, &out); err != nil {
		return nil, err
	}
	if parseFlexID(out.WorkID) == 0 && out.Minutes == 0 {
		return nil, nil
	}
	return &PlaytimeSelf{WorkID: parseFlexID(out.WorkID), Minutes: out.Minutes}, nil
}

// ListMyPlaytime pages the caller's own rows oldest-change-first, returning the
// next cursor. One row per (work, client): the same work appears once per
// application that reported it.
func (c *Client) ListMyPlaytime(ctx context.Context, accessToken, cursor string,
	limit int) ([]PlaytimeRecord, string, error) {

	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/v2/me/playtimes"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out v2List[v2Playtime]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		return nil, "", err
	}
	rows := out.rows()
	items := make([]PlaytimeRecord, 0, len(rows))
	for _, it := range rows {
		items = append(items, PlaytimeRecord{WorkID: parseFlexID(it.WorkID), Minutes: it.Minutes})
	}
	return items, out.cursor(), nil
}
