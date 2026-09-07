package catalogclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Mirrors catalog's model.FoldersPerUserMax / FolderItemsMax. Both are refused
// upstream with 422; they are repeated here so the forum can say so in its own
// words before spending a request.
const (
	FoldersPerUserMax = 200
	FolderItemsMax    = 10_000
)

const (
	FolderVisibilityPrivate = "private"
	FolderVisibilityPublic  = "public"
)

// A whole folder list and a whole item list are read in one go rather than
// paged through to the caller: the forum's collection API is page-numbered and
// orders by "newest added" while catalog's keyset walks updated_at ascending,
// so the two cannot be zipped. Reading everything and sorting locally is
// affordable because the shapes are small — p50 is 2 items and 5 folders per
// user, p99 is 370 items, and the ceiling is the cap below.
const folderWalkPages = FolderItemsMax/v2PageMax + 1

type Folder struct {
	ID          int64  `json:"-"`
	OwnerUID    int64  `json:"-"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	IsDefault   bool   `json:"is_default"`
	ItemCount   int    `json:"item_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type FolderItem struct {
	FolderID  int64
	WorkID    int64
	CreatedAt string
	UpdatedAt string
}

type v2Folder struct {
	ID          json.RawMessage `json:"id"`
	OwnerUID    json.RawMessage `json:"owner_uid"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Visibility  string          `json:"visibility"`
	IsDefault   bool            `json:"is_default"`
	ItemCount   int             `json:"item_count"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type v2FolderItem struct {
	FolderID  json.RawMessage `json:"folder_id"`
	WorkID    json.RawMessage `json:"work_id"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

func (f v2Folder) view() Folder {
	return Folder{
		ID: parseFlexID(f.ID), OwnerUID: parseFlexID(f.OwnerUID), Name: f.Name,
		Description: f.Description, Visibility: f.Visibility, IsDefault: f.IsDefault,
		ItemCount: f.ItemCount, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

func (i v2FolderItem) view() FolderItem {
	return FolderItem{
		FolderID: parseFlexID(i.FolderID), WorkID: parseFlexID(i.WorkID),
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

// walkUserV2 pages a user-token collection to the end. collectV2List is the
// app-key twin and stops at 20 pages, which is under the 10,000-item folder cap.
func walkUserV2[T any](ctx context.Context, c *Client, token, path string, q url.Values) ([]T, error) {
	var out []T
	cursor := ""
	for page := 0; page < folderWalkPages; page++ {
		q.Set("limit", strconv.Itoa(v2PageMax))
		if cursor != "" {
			q.Set("cursor", cursor)
		} else {
			q.Del("cursor")
		}
		var got v2List[T]
		if err := c.userV2JSON(ctx, http.MethodGet, token, path+"?"+q.Encode(), nil, &got, nil); err != nil {
			return nil, err
		}
		rows := got.rows()
		out = append(out, rows...)
		if got.NextCursor == nil || *got.NextCursor == "" || len(rows) == 0 {
			return out, nil
		}
		cursor = *got.NextCursor
	}
	return out, nil
}

func walkAppV2[T any](ctx context.Context, c *Client, path string, q url.Values) ([]T, error) {
	var out []T
	cursor := ""
	for page := 0; page < folderWalkPages; page++ {
		q.Set("limit", strconv.Itoa(v2PageMax))
		if cursor != "" {
			q.Set("cursor", cursor)
		} else {
			q.Del("cursor")
		}
		var got v2List[T]
		if err := c.appV2JSON(ctx, path, q, &got); err != nil {
			return nil, err
		}
		rows := got.rows()
		out = append(out, rows...)
		if got.NextCursor == nil || *got.NextCursor == "" || len(rows) == 0 {
			return out, nil
		}
		cursor = *got.NextCursor
	}
	return out, nil
}

// MyFolders lists every folder the bearer owns, private ones included.
func (c *Client) MyFolders(ctx context.Context, token string) ([]Folder, error) {
	rows, err := walkUserV2[v2Folder](ctx, c, token, "/v2/me/folders", url.Values{})
	return foldersView(rows), err
}

// MyFoldersContaining answers "which of my folders already hold this work" in
// one request. Without it the picker read every folder and probed each one.
func (c *Client) MyFoldersContaining(ctx context.Context, token string, workID int64) ([]Folder, error) {
	q := url.Values{}
	q.Set("contains_work_id", strconv.FormatInt(workID, 10))
	rows, err := walkUserV2[v2Folder](ctx, c, token, "/v2/me/folders", q)
	return foldersView(rows), err
}

func (c *Client) MyFolder(ctx context.Context, token string, folderID int64) (*Folder, error) {
	var out v2Folder
	if err := c.userV2JSON(ctx, http.MethodGet, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10), nil, &out, nil); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

// FolderPreviewItems reads one page instead of the whole folder. The items
// lane is keyed updated_at ASCENDING, so these are the OLDEST memberships, not
// the newest: the cover mosaic used to show the four most recently added, and
// reproducing that would mean walking a 3,303-item folder to reach its tail on
// every profile view. Earliest-first is also steadier — the mosaic stops
// reshuffling every time its owner files something.
func (c *Client) FolderPreviewItems(ctx context.Context, token string, folderID int64, n int) ([]FolderItem, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(n))
	path := "/v2/me/folders/" + strconv.FormatInt(folderID, 10) + "/items"
	var got v2List[v2FolderItem]
	if token != "" {
		if err := c.userV2JSON(ctx, http.MethodGet, token, path+"?"+q.Encode(), nil, &got, nil); err != nil {
			return nil, err
		}
		return itemsView(got.rows()), nil
	}
	if err := c.appV2JSON(ctx, "/v2/folders/"+strconv.FormatInt(folderID, 10)+"/items", q, &got); err != nil {
		return nil, err
	}
	return itemsView(got.rows()), nil
}

func (c *Client) MyFolderItems(ctx context.Context, token string, folderID int64) ([]FolderItem, error) {
	rows, err := walkUserV2[v2FolderItem](ctx, c, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10)+"/items", url.Values{})
	return itemsView(rows), err
}

// PublicFolders and its siblings take the application key: a public folder is
// public to anonymous readers too, and folder:read is consent to read the
// bearer's OWN folders, which says nothing about anybody else's.
func (c *Client) PublicFolders(ctx context.Context, ownerUID int64) ([]Folder, error) {
	q := url.Values{}
	q.Set("owner_uid", strconv.FormatInt(ownerUID, 10))
	rows, err := walkAppV2[v2Folder](ctx, c, "/v2/folders", q)
	return foldersView(rows), err
}

func (c *Client) PublicFolder(ctx context.Context, folderID int64) (*Folder, error) {
	var out v2Folder
	if err := c.appV2JSON(ctx, "/v2/folders/"+strconv.FormatInt(folderID, 10), url.Values{}, &out); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

func (c *Client) PublicFolderItems(ctx context.Context, folderID int64) ([]FolderItem, error) {
	rows, err := walkAppV2[v2FolderItem](ctx, c,
		"/v2/folders/"+strconv.FormatInt(folderID, 10)+"/items", url.Values{})
	return itemsView(rows), err
}

type FolderWrite struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	IsDefault   *bool   `json:"is_default,omitempty"`
}

func (c *Client) CreateFolder(ctx context.Context, token string, in FolderWrite) (*Folder, error) {
	var out v2Folder
	if err := c.userV2JSON(ctx, http.MethodPost, token, "/v2/me/folders", in, &out, nil); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

func (c *Client) PatchFolder(ctx context.Context, token string, folderID int64, in FolderWrite) (*Folder, error) {
	var out v2Folder
	if err := c.userV2JSON(ctx, http.MethodPatch, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10), in, &out, nil); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

func (c *Client) DeleteFolder(ctx context.Context, token string, folderID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodDelete, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10), nil, nil)
	return err
}

func (c *Client) PutFolderItem(ctx context.Context, token string, folderID, workID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodPut, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10)+"/items/"+strconv.FormatInt(workID, 10), nil, nil)
	return err
}

func (c *Client) DeleteFolderItem(ctx context.Context, token string, folderID, workID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodDelete, token,
		"/v2/me/folders/"+strconv.FormatInt(folderID, 10)+"/items/"+strconv.FormatInt(workID, 10), nil, nil)
	return err
}

// The moderation trio carries the acting moderator's own token, so catalog
// judges the standing rather than trusting the forum's word for it.
func (c *Client) ModeratePatchFolder(ctx context.Context, token string, folderID int64, in FolderWrite) (*Folder, error) {
	var out v2Folder
	if err := c.userV2JSON(ctx, http.MethodPatch, token,
		"/v2/moderation/folders/"+strconv.FormatInt(folderID, 10), in, &out, nil); err != nil {
		return nil, err
	}
	f := out.view()
	return &f, nil
}

func (c *Client) ModerateDeleteFolder(ctx context.Context, token string, folderID int64) error {
	_, _, err := c.userV2Do(ctx, http.MethodDelete, token,
		"/v2/moderation/folders/"+strconv.FormatInt(folderID, 10), nil, nil)
	return err
}

type FolderPurgeReceipt struct {
	FoldersDeleted int64 `json:"folders_deleted"`
	ItemsDeleted   int64 `json:"items_deleted"`
}

func (c *Client) PurgeUserFolders(ctx context.Context, token string, uid int64) (*FolderPurgeReceipt, error) {
	var out FolderPurgeReceipt
	if err := c.userV2JSON(ctx, http.MethodDelete, token,
		"/v2/moderation/users/"+strconv.FormatInt(uid, 10)+"/folders", nil, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

func foldersView(rows []v2Folder) []Folder {
	out := make([]Folder, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.view())
	}
	return out
}

func itemsView(rows []v2FolderItem) []FolderItem {
	out := make([]FolderItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.view())
	}
	return out
}
