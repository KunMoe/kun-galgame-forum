package catalogclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type NewsSubmissionWrite struct {
	Title     string
	Summary   string
	Body      string
	SourceURL string
}

type NewsSubmissionPatch struct {
	Withdraw  bool
	Title     *string
	Summary   *string
	Body      *string
	SourceURL *string
}

type NewsSubmission struct {
	ID           int64
	SourceKey    string
	Lane         string
	Status       string
	Title        string
	Summary      string
	Body         string
	SourceURL    string
	PublishedAt  time.Time
	SubmitterUID int64
}

type v2NewsSource struct {
	Name string `json:"name"`
}

type v2NewsSubmission struct {
	ID           json.RawMessage `json:"id"`
	Source       v2NewsSource    `json:"source"`
	Lane         string          `json:"lane"`
	Status       string          `json:"status"`
	Title        string          `json:"title"`
	Summary      string          `json:"summary"`
	Body         string          `json:"body"`
	SourceURL    string          `json:"source_url"`
	PublishedAt  string          `json:"published_at"`
	SubmitterUID json.RawMessage `json:"submitter_uid"`
}

func (v v2NewsSubmission) submission() NewsSubmission {
	return NewsSubmission{
		ID:           parseFlexID(v.ID),
		SourceKey:    v.Source.Name,
		Lane:         v.Lane,
		Status:       v.Status,
		Title:        v.Title,
		Summary:      v.Summary,
		Body:         v.Body,
		SourceURL:    v.SourceURL,
		PublishedAt:  parseRFC3339(v.PublishedAt),
		SubmitterUID: parseFlexID(v.SubmitterUID),
	}
}

func newsWriteBody(in NewsSubmissionWrite) map[string]any {
	body := map[string]any{
		"title":   in.Title,
		"summary": in.Summary,
	}
	if in.Body != "" {
		body["body"] = in.Body
	}
	if in.SourceURL != "" {
		body["source_url"] = in.SourceURL
	}
	return body
}

func newsPatchBody(in NewsSubmissionPatch) any {
	if in.Withdraw {
		return map[string]string{"status": "withdrawn"}
	}
	body := map[string]any{}
	if in.Title != nil {
		body["title"] = *in.Title
	}
	if in.Summary != nil {
		body["summary"] = *in.Summary
	}
	if in.Body != nil {
		body["body"] = *in.Body
	}
	if in.SourceURL != nil {
		body["source_url"] = *in.SourceURL
	}
	return body
}

func myNewsItemPath(id int64) string {
	return "/v2/me/news/" + strconv.FormatInt(id, 10)
}

func (c *Client) CreateMyNews(ctx context.Context, accessToken string, in NewsSubmissionWrite, idempotencyKey string) (*NewsSubmission, error) {
	var out v2NewsSubmission
	if err := c.userV2JSON(ctx, http.MethodPost, accessToken, "/v2/me/news", newsWriteBody(in), &out, idempotencyHeader(nil, idempotencyKey)); err != nil {
		return nil, err
	}
	sub := out.submission()
	return &sub, nil
}

func (c *Client) ListMyNews(ctx context.Context, accessToken, cursor string, limit int) ([]NewsSubmission, string, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/v2/me/news"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out v2List[v2NewsSubmission]
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, path, nil, &out, nil); err != nil {
		return nil, "", err
	}
	rows := out.rows()
	items := make([]NewsSubmission, 0, len(rows))
	for _, it := range rows {
		items = append(items, it.submission())
	}
	return items, out.cursor(), nil
}

func (c *Client) GetMyNews(ctx context.Context, accessToken string, id int64) (*NewsSubmission, error) {
	var out v2NewsSubmission
	if err := c.userV2JSON(ctx, http.MethodGet, accessToken, myNewsItemPath(id), nil, &out, nil); err != nil {
		return nil, err
	}
	sub := out.submission()
	return &sub, nil
}

func (c *Client) PatchMyNews(ctx context.Context, accessToken string, id int64, in NewsSubmissionPatch) (*NewsSubmission, error) {
	var headers map[string]string
	if in.Withdraw {
		headers = ifMatchStar()
	}
	var out v2NewsSubmission
	if err := c.userV2JSON(ctx, http.MethodPatch, accessToken, myNewsItemPath(id), newsPatchBody(in), &out, headers); err != nil {
		return nil, err
	}
	sub := out.submission()
	return &sub, nil
}
