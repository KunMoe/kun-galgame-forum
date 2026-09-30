package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type CompanyFollowRecord struct {
	CompanyID int64
	CreatedAt string
}

type v2CompanyFollow struct {
	CompanyID json.RawMessage `json:"company_id"`
	CreatedAt string          `json:"created_at"`
}

func (c *Client) MyCompanyFollow(ctx context.Context, accessToken string, companyID int64) (bool, error) {
	path := "/v2/me/followed-companies/" + strconv.FormatInt(companyID, 10)
	var out v2CompanyFollow
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) FollowCompany(ctx context.Context, accessToken string, companyID int64) error {
	path := "/v2/me/followed-companies/" + strconv.FormatInt(companyID, 10)
	_, _, err := c.userV2Do(ctx, http.MethodPut, accessToken, path, nil, nil)
	return err
}

func (c *Client) UnfollowCompany(ctx context.Context, accessToken string, companyID int64) error {
	path := "/v2/me/followed-companies/" + strconv.FormatInt(companyID, 10)
	_, _, err := c.userV2Do(ctx, http.MethodDelete, accessToken, path, nil, nil)
	if err != nil && errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) ListMyCompanyFollows(ctx context.Context, accessToken, cursor string, limit int) ([]CompanyFollowRecord, string, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/v2/me/followed-companies"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out v2List[v2CompanyFollow]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		return nil, "", err
	}
	rows := out.rows()
	items := make([]CompanyFollowRecord, 0, len(rows))
	for _, it := range rows {
		items = append(items, CompanyFollowRecord{
			CompanyID: parseFlexID(it.CompanyID),
			CreatedAt: it.CreatedAt,
		})
	}
	return items, out.cursor(), nil
}
