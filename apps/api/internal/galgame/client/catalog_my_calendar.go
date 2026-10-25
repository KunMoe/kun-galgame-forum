package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"kun-galgame-api/pkg/catalogclient"
)

func (c *GalgameClient) CatalogMyCalendar(ctx context.Context, accessToken, bucket string, q url.Values) (*CatalogWorksPage, error) {
	if c == nil || c.origin == "" {
		return nil, fmt.Errorf("%w", catalogclient.ErrNotConfigured)
	}
	if accessToken == "" {
		return nil, catalogclient.ErrUnauthorized
	}
	path := "/catalog/calendar" + bucket
	status, raw, err := c.doV2UserGET(ctx, accessToken, "/v2/me/calendar", v2CatalogQuery(path, q))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, userPlaneStatus(status, raw)
	}
	body := rewriteBanners(json.RawMessage(rewriteV2JSON(raw, c.imageCDNBase)), c.imageCDNBase)
	page, err := decodeCalendarPage(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", catalogclient.ErrUpstream, err)
	}
	return page, nil
}

func decodeCalendarPage(data json.RawMessage) (*CatalogWorksPage, error) {
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	count := parsed.Count
	if count == 0 {
		count = parsed.Total
	}
	page := &CatalogWorksPage{
		Items: parsed.Items, Count: count,
		Month: parsed.Month, Year: parsed.Year, Meta: parsed.Meta,
	}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}

func (c *GalgameClient) doV2UserGET(ctx context.Context, accessToken, v2Path string, q url.Values) (int, []byte, error) {
	reqURL := c.origin + v2Path
	if len(q) > 0 {
		reqURL += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", catalogclient.ErrUpstream, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Galgame 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return 0, nil, fmt.Errorf("%w: %v", catalogclient.ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("%w: %v", catalogclient.ErrUpstream, err)
	}
	return resp.StatusCode, raw, nil
}

func userPlaneStatus(status int, raw []byte) error {
	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	msg := p.Detail
	if msg == "" {
		msg = p.Title
	}
	if msg == "" {
		msg = strings.TrimSpace(string(raw))
	}
	blob := strings.ToLower(p.Code + " " + p.Detail + " " + p.Title + " " + string(raw))
	switch status {
	case http.StatusUnauthorized:
		return catalogclient.ErrUnauthorized
	case http.StatusNotFound:
		return catalogclient.ErrNotFound
	case http.StatusForbidden:
		if p.Code == "SCOPE_REQUIRED" || strings.Contains(blob, "scope") {
			return catalogclient.ErrInsufficientScope
		}
		return &catalogclient.UserAPIError{Status: status, Message: msg, ProblemCode: p.Code}
	default:
		if status >= 500 {
			return catalogclient.ErrUpstream
		}
		return &catalogclient.UserAPIError{Status: status, Message: msg, ProblemCode: p.Code}
	}
}
