package catalogclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func entityTypeFromObject(object string) string {
	switch object {
	case "company":
		return "catalog.label"
	case "":
		return ""
	default:
		if strings.HasPrefix(object, "catalog.") {
			return object
		}
		return "catalog." + object
	}
}

func objectFamily(entityType string) string {
	switch entityType {
	case EntityTypeWork, "work":
		return "work"
	case "catalog.label", "company":
		return "company"
	default:
		return strings.TrimPrefix(entityType, "catalog.")
	}
}

const (
	v2PageMax     = 100
	v2PageWalkMax = 20
)

func clampV2Limit(n int) int {
	if n <= 0 {
		return 20
	}
	if n > v2PageMax {
		return v2PageMax
	}
	return n
}

func collectV2List[T any](ctx context.Context, c *Client, path string, q url.Values, want int) ([]T, *int64, error) {
	if want <= 0 {
		want = 20
	}
	var (
		out    []T
		total  *int64
		cursor = q.Get("cursor")
	)
	for pages := 0; pages < v2PageWalkMax && len(out) < want; pages++ {
		n := want - len(out)
		if n > v2PageMax {
			n = v2PageMax
		}
		q.Set("limit", strconv.Itoa(n))
		if cursor != "" {
			q.Set("cursor", cursor)
		} else {
			q.Del("cursor")
		}
		var page v2List[T]
		if err := c.appV2JSON(ctx, path, q, &page); err != nil {
			return nil, total, err
		}
		if page.Total != nil {
			total = page.Total
		}
		rows := page.rows()
		out = append(out, rows...)
		if page.NextCursor == nil || *page.NextCursor == "" || len(rows) == 0 {
			break
		}
		cursor = *page.NextCursor
	}
	if len(out) > want {
		out = out[:want]
	}
	return out, total, nil
}

func encodeWatermark(id int64) string {
	if id <= 0 {
		return ""
	}
	return "cur_" + base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func parseRevisionAction(raw string) int16 {
	switch raw {
	case "created", "0":
		return EditActionCreated
	case "merged", "1":
		return EditActionMerged
	case "direct", "2":
		return EditActionDirect
	case "reverted", "3":
		return EditActionReverted
	}
	n, _ := strconv.ParseInt(raw, 10, 16)
	return int16(n)
}

func (c *Client) appV2Do(ctx context.Context, path string, query url.Values) ([]byte, error) {
	if !c.AppConfigured() {
		return nil, ErrNotConfigured
	}
	reqURL := c.origin() + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.appKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	retryAfter := resp.Header.Get("Retry-After")

	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return raw, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		if resp.StatusCode >= 500 {
			return nil, ErrUpstream
		}
		return nil, &UserAPIError{Status: resp.StatusCode, Message: problemMsg(p, raw), RetryAfter: retryAfter}
	}
}

func (c *Client) appV2JSON(ctx context.Context, path string, query url.Values, out any) error {
	raw, err := c.appV2Do(ctx, path, query)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
