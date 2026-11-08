package catalogclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

const (
	FolderSortFollowersDesc = "followers_desc"
	FolderSortUpdatedDesc   = "updated_desc"
	FolderSortCreatedDesc   = "created_desc"

	FollowedFoldersMax = 1000
	FolderNoteMax      = 500
)

type FolderPage struct {
	Folders    []Folder
	NextCursor string
	Follows    map[int64]FollowState
}

type FollowState struct {
	NewItemCount int
	SeenAt       string
}

type DirectoryQuery struct {
	Tag            string
	Q              string
	Sort           string
	IncludeNSFW    bool
	ContainsWorkID int64
	Cursor         string
	Limit          int
}

// FolderDirectory reads the public directory lane of GET /v2/folders: no
// owner_uid. nsfw is a filter there only; the owner lane ignores it.
func (c *Client) FolderDirectory(ctx context.Context, in DirectoryQuery) (*FolderPage, error) {
	q := url.Values{}
	if in.Sort != "" {
		q.Set("sort", in.Sort)
	}
	q.Set("nsfw", strconv.FormatBool(in.IncludeNSFW))
	if in.ContainsWorkID > 0 {
		q.Set("contains_work_id", strconv.FormatInt(in.ContainsWorkID, 10))
	}
	if in.Tag != "" {
		q.Set("tag", in.Tag)
	}
	if in.Q != "" {
		q.Set("q", in.Q)
	}
	if in.Cursor != "" {
		q.Set("cursor", in.Cursor)
	}
	if in.Limit > 0 {
		q.Set("limit", strconv.Itoa(in.Limit))
	}
	var got v2List[v2Folder]
	if err := c.appV2JSON(ctx, "/v2/folders", q, &got); err != nil {
		return nil, err
	}
	return &FolderPage{Folders: foldersView(got.rows()), NextCursor: got.cursor()}, nil
}

type FolderItemPatch struct {
	Note         *string `json:"note,omitempty"`
	AfterWorkID  string  `json:"after_work_id,omitempty"`
	BeforeWorkID string  `json:"before_work_id,omitempty"`
}

func folderItemPath(prefix string, folderID, workID int64) string {
	return prefix + strconv.FormatInt(folderID, 10) + "/items/" + strconv.FormatInt(workID, 10)
}

func (c *Client) PatchFolderItem(ctx context.Context, token string, folderID, workID int64, in FolderItemPatch) (*FolderItem, error) {
	var out v2FolderItem
	if err := c.userV2JSON(ctx, http.MethodPatch, token, folderItemPath("/v2/me/folders/", folderID, workID), in, &out, nil); err != nil {
		return nil, err
	}
	it := out.view()
	return &it, nil
}

func (c *Client) MyFolderItem(ctx context.Context, token string, folderID, workID int64) (*FolderItem, error) {
	var out v2FolderItem
	if err := c.userV2JSON(ctx, http.MethodGet, token, folderItemPath("/v2/me/folders/", folderID, workID), nil, &out, nil); err != nil {
		return nil, err
	}
	it := out.view()
	return &it, nil
}

func (c *Client) ModeratePatchFolderItem(ctx context.Context, token string, folderID, workID int64, note string) (*FolderItem, error) {
	var out v2FolderItem
	body := map[string]string{"note": note}
	if err := c.userV2JSON(ctx, http.MethodPatch, token, folderItemPath("/v2/moderation/folders/", folderID, workID), body, &out, nil); err != nil {
		return nil, err
	}
	it := out.view()
	return &it, nil
}

type v2FolderFollow struct {
	FolderID     json.RawMessage `json:"folder_id"`
	Folder       v2Folder        `json:"folder"`
	CreatedAt    string          `json:"created_at"`
	NewItemCount int             `json:"new_item_count"`
	SeenAt       string          `json:"seen_at"`
}

func followPath(folderID int64) string {
	return "/v2/me/followed-folders/" + strconv.FormatInt(folderID, 10)
}

// FollowFolder answers the folder as it stands after the follow, so its
// follower_count already counts the bearer.
func (c *Client) FollowFolder(ctx context.Context, token string, folderID int64) (*Folder, error) {
	var out v2FolderFollow
	if err := c.userV2JSON(ctx, http.MethodPut, token, followPath(folderID), nil, &out, nil); err != nil {
		return nil, err
	}
	f := out.Folder.view()
	return &f, nil
}

func (c *Client) UnfollowFolder(ctx context.Context, token string, folderID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodDelete, token, followPath(folderID), nil, nil)
	return err
}

func (c *Client) FollowedFolders(ctx context.Context, token, cursor string, limit int) (*FolderPage, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var got v2List[v2FolderFollow]
	if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/followed-folders?"+q.Encode(), nil, &got, nil); err != nil {
		return nil, err
	}
	rows := got.rows()
	folders := make([]Folder, 0, len(rows))
	follows := make(map[int64]FollowState, len(rows))
	for _, r := range rows {
		f := r.Folder.view()
		folders = append(folders, f)
		follows[f.ID] = FollowState{NewItemCount: r.NewItemCount, SeenAt: r.SeenAt}
	}
	return &FolderPage{Folders: folders, NextCursor: got.cursor(), Follows: follows}, nil
}

// FollowedAmong answers which of folderIDs the bearer follows. Catalog takes at
// most 100 ids per call; this chunks.
func (c *Client) FollowedAmong(ctx context.Context, token string, folderIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	for _, chunk := range chunkInt64s(folderIDs, v2PageMax) {
		if len(chunk) == 0 {
			continue
		}
		q := url.Values{}
		q.Set("folder_ids", joinInt64s(chunk))
		var got v2List[v2FolderFollow]
		if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/followed-folders?"+q.Encode(), nil, &got, nil); err != nil {
			return nil, err
		}
		for _, r := range got.rows() {
			if id := parseFlexID(r.FolderID); id > 0 {
				out[id] = true
			}
		}
	}
	return out, nil
}

type FolderReport struct {
	FolderID string `json:"folder_id"`
	Reason   string `json:"reason"`
	Note     string `json:"note,omitempty"`
	WorkID   string `json:"work_id,omitempty"`
}

func (c *Client) ReportFolder(ctx context.Context, token string, in FolderReport, idempotencyKey string) error {
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"Idempotency-Key": idempotencyKey}
	}
	_, _, err := c.userV2Do(ctx, http.MethodPost, token, "/v2/me/folder-reports", in, headers)
	return err
}

type FolderTagCount struct {
	Tag         string `json:"tag"`
	FolderCount int    `json:"folder_count"`
}

func (c *Client) FolderTags(ctx context.Context, limit int, includeNSFW bool) ([]FolderTagCount, error) {
	q := url.Values{}
	q.Set("nsfw", strconv.FormatBool(includeNSFW))
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var got v2List[FolderTagCount]
	if err := c.appV2JSON(ctx, "/v2/folders/tags", q, &got); err != nil {
		return nil, err
	}
	return got.rows(), nil
}

func (c *Client) CopyFolder(ctx context.Context, token string, folderID int64, name, idempotencyKey string) (*Folder, error) {
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"Idempotency-Key": idempotencyKey}
	}
	body := map[string]string{"folder_id": strconv.FormatInt(folderID, 10)}
	if name != "" {
		body["name"] = name
	}
	var out v2Folder
	if err := c.userV2JSON(ctx, http.MethodPost, token, "/v2/me/folders/copies", body, &out, headers); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

func (c *Client) FollowedUpdates(ctx context.Context, token string) (int, error) {
	var out struct {
		UpdatedCount int `json:"updated_count"`
	}
	if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/followed-folders/updates", nil, &out, nil); err != nil {
		return 0, err
	}
	return out.UpdatedCount, nil
}

func (c *Client) MarkFollowedSeen(ctx context.Context, token string, folderID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodPost, token, followPath(folderID)+"/seen", map[string]string{}, nil)
	return err
}

type FolderProgress struct {
	FolderID  int64
	ItemCount int
	Collected int
	Wish      int
	Doing     int
	Done      int
	OnHold    int
	Dropped   int
}

type v2FolderProgress struct {
	FolderID  json.RawMessage `json:"folder_id"`
	ItemCount int             `json:"item_count"`
	Collected int             `json:"collected"`
	Wish      int             `json:"wish"`
	Doing     int             `json:"doing"`
	Done      int             `json:"done"`
	OnHold    int             `json:"on_hold"`
	Dropped   int             `json:"dropped"`
}

func (c *Client) FolderProgress(ctx context.Context, token string, folderIDs []int64) ([]FolderProgress, error) {
	q := url.Values{}
	q.Set("folder_ids", joinInt64s(folderIDs))
	var got v2List[v2FolderProgress]
	if err := c.userV2JSON(ctx, http.MethodGet, token, "/v2/me/folders/progress?"+q.Encode(), nil, &got, nil); err != nil {
		return nil, err
	}
	out := make([]FolderProgress, 0, len(got.Items))
	for _, r := range got.rows() {
		out = append(out, FolderProgress{
			FolderID: parseFlexID(r.FolderID), ItemCount: r.ItemCount, Collected: r.Collected,
			Wish: r.Wish, Doing: r.Doing, Done: r.Done, OnHold: r.OnHold, Dropped: r.Dropped,
		})
	}
	return out, nil
}
