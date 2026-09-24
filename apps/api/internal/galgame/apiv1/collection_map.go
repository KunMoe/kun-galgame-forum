package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const previewCoversPerCollection = 4

func parseCollectionID(raw string) (int64, bool) {
	n, ok := parseWorkID(raw)
	if !ok {
		return 0, false
	}
	return int64(n), true
}

func folderTime(raw string, folderID int64, field string) (repr.DateTime, bool) {
	if raw == "" {
		slog.Warn("collection: timestamp unparseable, row dropped", "folder_id", folderID, "field", field, "value", raw)
		return "", false
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return repr.Timestamp(t), true
		}
	}
	slog.Warn("collection: timestamp unparseable, row dropped", "folder_id", folderID, "field", field, "value", raw)
	return "", false
}

func folderVisibility(raw string, folderID int64) (string, bool) {
	switch raw {
	case catalogclient.FolderVisibilityPrivate, catalogclient.FolderVisibilityPublic:
		return raw, true
	case "":
		return catalogclient.FolderVisibilityPrivate, true
	default:
		slog.Warn("collection: unknown visibility, row dropped", "folder_id", folderID, "visibility", raw)
		return "", false
	}
}

func (s *Service) collectionViewer(user *middleware.UserInfo, folder catalogclient.Folder, hasWork bool) *CollectionViewer {
	if user == nil {
		return nil
	}
	ownerID := int(folder.OwnerUID)
	isOwner := user.ID == ownerID
	return &CollectionViewer{
		IsOwner:   isOwner,
		CanEdit:   isOwner || user.Can(perm.CollectionEditAny),
		CanDelete: (isOwner && !folder.IsDefault) || (!isOwner && user.Can(perm.CollectionDeleteAny)),
		HasWork:   hasWork,
	}
}

func (s *Service) lookupOwner(ctx context.Context, ownerUID int64) (userclient.User, bool, *problem.Problem) {
	users, p := s.lookupUsers(ctx, []int{int(ownerUID)})
	if p != nil {
		return userclient.User{}, false, p
	}
	u, ok := users[int(ownerUID)]
	if !ok || !userclient.IsRenderable(u) {
		return userclient.User{}, false, nil
	}
	return u, true, nil
}

func (s *Service) previewCovers(ctx context.Context, token string, folder catalogclient.Folder, includeNSFW bool) []repr.Image {
	out := []repr.Image{}
	if s.catalog == nil || folder.ItemCount == 0 || s.hydrator == nil {
		return out
	}
	items, err := s.catalog.FolderPreviewItems(ctx, token, folder.ID, previewCoversPerCollection)
	if err != nil {
		slog.Warn("collection: preview items unreadable", "folder_id", folder.ID, "upstream_status", upstreamStatus(err), "err", err)
		return out
	}
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, int(it.WorkID))
	}
	// Hydrating under sfw made catalog omit NSFW preview items and the hydrator
	// log each one as "catalog did not render work"; on 2026-09-24 that noise
	// was misread as a counting bug (G6.1). The IsNSFW check below is the gate.
	sums, p := s.hydrator.ByIDs(ctx, ids, true)
	if p != nil {
		slog.Warn("collection: preview hydrate failed", "folder_id", folder.ID, "err", p)
		return out
	}
	byID := map[int]previewArt{}
	for i := range sums {
		id, ok := repr.ParseID(sums[i].ID)
		if !ok {
			continue
		}
		// Catalog leaves banner null when no cover is landscape. v1 read the
		// banner alone, so such works vanished from every preview with a WARN
		// a minute (2026-09-24); the legacy preview fell back to the portrait.
		art := sums[i].Banner
		if art == nil {
			art = sums[i].Cover
		}
		byID[id] = previewArt{image: art, nsfw: sums[i].IsNSFW}
	}
	for _, it := range items {
		a, ok := byID[int(it.WorkID)]
		if !ok || a.image == nil {
			continue
		}
		if !includeNSFW && a.nsfw {
			continue
		}
		out = append(out, *a.image)
		if len(out) >= previewCoversPerCollection {
			break
		}
	}
	return out
}

type previewArt struct {
	image *repr.Image
	nsfw  bool
}

func (s *Service) toCollection(ctx context.Context, folder catalogclient.Folder, user *middleware.UserInfo, token string, hasWork, includeNSFW bool) (*Collection, bool, *problem.Problem) {
	vis, ok := folderVisibility(folder.Visibility, folder.ID)
	if !ok {
		return nil, false, nil
	}
	created, ok := folderTime(folder.CreatedAt, folder.ID, "created_at")
	if !ok {
		return nil, false, nil
	}
	updated, ok := folderTime(folder.UpdatedAt, folder.ID, "updated_at")
	if !ok {
		return nil, false, nil
	}
	owner, renderable, p := s.lookupOwner(ctx, folder.OwnerUID)
	if p != nil {
		return nil, false, p
	}
	isOwner := user != nil && user.ID == int(folder.OwnerUID)
	if !renderable && !isOwner {
		return nil, false, nil
	}
	var ownerRef repr.UserRef
	if renderable {
		ownerRef = repr.NewUserRef(s.cdn, owner)
	} else {
		ownerRef = repr.DeletedUserRef(int(folder.OwnerUID))
	}
	title := truncateFolderText(folder, "name", folder.Name, 100)
	desc := truncateFolderText(folder, "description", folder.Description, 500)
	covers := s.previewCovers(ctx, token, folder, includeNSFW)
	if covers == nil {
		covers = []repr.Image{}
	}
	col := Collection{
		Object: "collection", ID: repr.ID(int(folder.ID)), Title: title, Description: desc,
		Visibility: vis, IsDefault: folder.IsDefault, ItemCount: max(folder.ItemCount, 0),
		Owner: ownerRef, PreviewCovers: covers, CreatedAt: created, UpdatedAt: updated,
		Viewer: s.collectionViewer(user, folder, hasWork),
	}
	return &col, true, nil
}

func (s *Service) toSummary(ctx context.Context, folder catalogclient.Folder, user *middleware.UserInfo, token string, hasWork, includeNSFW bool) (*CollectionSummary, bool, *problem.Problem) {
	col, ok, p := s.toCollection(ctx, folder, user, token, hasWork, includeNSFW)
	if p != nil || !ok {
		return nil, ok, p
	}
	sum := CollectionSummary{
		Object: col.Object, ID: col.ID, Title: col.Title, Description: col.Description,
		Visibility: col.Visibility, IsDefault: col.IsDefault, ItemCount: col.ItemCount,
		Owner: col.Owner, PreviewCovers: col.PreviewCovers, CreatedAt: col.CreatedAt,
		UpdatedAt: col.UpdatedAt, Viewer: col.Viewer,
	}
	return &sum, true, nil
}

func sortFolders(folders []catalogclient.Folder) {
	sort.SliceStable(folders, func(i, j int) bool {
		if folders[i].IsDefault != folders[j].IsDefault {
			return folders[i].IsDefault
		}
		if folders[i].UpdatedAt != folders[j].UpdatedAt {
			return folders[i].UpdatedAt > folders[j].UpdatedAt
		}
		return folders[i].ID > folders[j].ID
	})
}

func pageOfFolders(folders []catalogclient.Folder, page, limit int) []catalogclient.Folder {
	start := (page - 1) * limit
	if start >= len(folders) {
		return nil
	}
	end := start + limit
	if end > len(folders) {
		end = len(folders)
	}
	return folders[start:end]
}

const folderItemsCacheTTL = 6 * time.Hour

type cachedFolderItem struct {
	WorkID    int64  `json:"w"`
	CreatedAt string `json:"c"`
}

// Catalog bumps a folder's updated_at on every item add and remove, so the key
// goes stale by itself.
func folderItemsCacheKey(folder *catalogclient.Folder) string {
	return "kungal:folder-items:v2:" + strconv.FormatInt(folder.ID, 10) + ":" + folder.UpdatedAt
}

func (s *Service) loadFolderItems(ctx context.Context, folder *catalogclient.Folder, token string) ([]catalogclient.FolderItem, error) {
	if folder == nil || folder.ItemCount == 0 {
		return []catalogclient.FolderItem{}, nil
	}
	if items, ok := s.cachedFolderItems(ctx, folder); ok {
		return items, nil
	}
	var (
		items []catalogclient.FolderItem
		err   error
	)
	if token != "" {
		items, err = s.catalog.MyFolderItems(ctx, token, folder.ID)
	} else {
		items, err = s.catalog.PublicFolderItems(ctx, folder.ID)
	}
	if err != nil {
		return nil, err
	}
	s.storeFolderItems(ctx, folder, items)
	return items, nil
}

func (s *Service) cachedFolderItems(ctx context.Context, folder *catalogclient.Folder) ([]catalogclient.FolderItem, bool) {
	if s.rdb == nil {
		return nil, false
	}
	raw, err := s.rdb.Get(ctx, folderItemsCacheKey(folder)).Bytes()
	if err != nil {
		return nil, false
	}
	var rows []cachedFolderItem
	if json.Unmarshal(raw, &rows) != nil {
		return nil, false
	}
	items := make([]catalogclient.FolderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, catalogclient.FolderItem{FolderID: folder.ID, WorkID: row.WorkID, CreatedAt: row.CreatedAt})
	}
	return items, true
}

func (s *Service) storeFolderItems(ctx context.Context, folder *catalogclient.Folder, items []catalogclient.FolderItem) {
	if s.rdb == nil {
		return
	}
	rows := make([]cachedFolderItem, 0, len(items))
	for _, it := range items {
		rows = append(rows, cachedFolderItem{WorkID: it.WorkID, CreatedAt: it.CreatedAt})
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return
	}
	if err := s.rdb.Set(ctx, folderItemsCacheKey(folder), raw, folderItemsCacheTTL).Err(); err != nil {
		slog.Warn("collection: folder items cache write failed", "folder_id", folder.ID, "err", err)
	}
}

func (s *Service) firstAddSideEffects(ctx context.Context, userID, workID int) {
	if s.store == nil || !s.store.Ready() {
		return
	}
	ownerID, p := s.localCreator(workID)
	if p != nil {
		slog.Error("collection: membership stored but local bookkeeping failed", "work_id", workID, "user_id", userID, "err", p)
		return
	}
	preview := ""
	if s.works != nil {
		rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, []int{workID}, "names", "all")
		if appErr == nil {
			if row, ok := rows[workID]; ok {
				preview = workNamePreview(ctx, &row)
			}
		}
	}
	var jobs []pendingAward
	txErr := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		if err := s.store.EnsureLocal(tx, workID); err != nil {
			return err
		}
		if err := s.store.AdjustFavoriteCount(tx, workID, 1); err != nil {
			return err
		}
		if ownerID <= 0 || ownerID == userID {
			return nil
		}
		if err := s.store.CreateFavoriteMessage(tx, userID, ownerID, preview, workID); err != nil {
			return err
		}
		jobs = append(jobs, pendingAward{
			userID: ownerID, delta: 1, reason: moemoepoint.ReasonLiked,
			ref: moemoepoint.Ref("galgame", workID),
			key: moemoepoint.KeyNonce(moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", workID)),
		})
		return nil
	})
	if txErr != nil {
		slog.Error("collection: membership stored but local bookkeeping failed",
			"work_id", workID, "user_id", userID, "err", txErr)
		return
	}
	s.flushAwards(jobs)
}

func (s *Service) lastRemoveSideEffects(userID, workID int) {
	if s.store == nil || !s.store.Ready() {
		return
	}
	ownerID, p := s.localCreator(workID)
	if p != nil {
		slog.Error("collection: membership stored but local bookkeeping failed", "work_id", workID, "user_id", userID, "err", p)
		return
	}
	var jobs []pendingAward
	txErr := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		if err := s.store.AdjustFavoriteCount(tx, workID, -1); err != nil {
			return err
		}
		if ownerID <= 0 || ownerID == userID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: ownerID, delta: -1, reason: moemoepoint.ReasonLiked,
			ref: moemoepoint.Ref("galgame", workID),
			key: moemoepoint.KeyNonce(moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", workID)),
		})
		return nil
	})
	if txErr != nil {
		slog.Error("collection: membership stored but local bookkeeping failed",
			"work_id", workID, "user_id", userID, "err", txErr)
		return
	}
	s.flushAwards(jobs)
}

func (s *Service) folderDeleteSideEffects(orphaned []int) {
	if s.store == nil || !s.store.Ready() || len(orphaned) == 0 {
		return
	}
	if err := s.store.InTx(func(tx *gorm.DB) error {
		return s.store.DecrementFavoriteCounts(tx, orphaned)
	}); err != nil {
		slog.Error("collection: folder deleted but local favourite counts failed", "err", err)
	}
}

func (s *Service) worksLeavingTheLibrary(ctx context.Context, token string, folderID int64) ([]int, error) {
	items, err := s.catalog.MyFolderItems(ctx, token, folderID)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	folders, err := s.catalog.MyFolders(ctx, token)
	if err != nil {
		return nil, err
	}
	elsewhere := map[int64]bool{}
	for _, f := range folders {
		if f.ID == folderID {
			continue
		}
		others, oErr := s.catalog.MyFolderItems(ctx, token, f.ID)
		if oErr != nil {
			return nil, oErr
		}
		for _, it := range others {
			elsewhere[it.WorkID] = true
		}
	}
	out := []int{}
	for _, it := range items {
		if !elsewhere[it.WorkID] {
			out = append(out, int(it.WorkID))
		}
	}
	return out, nil
}

func workNSFW(it *client.CatalogWorkListItem) bool {
	if it == nil {
		return false
	}
	return client.CatalogItemToBrief(context.Background(), it).ContentLimit == "nsfw"
}

func callerUser(ctx context.Context) *middleware.UserInfo {
	return v1.User(ctx)
}

func truncateFolderText(folder catalogclient.Folder, field, v string, max int) string {
	if utf8.RuneCountInString(v) <= max {
		return v
	}
	slog.Warn("catalog folder text longer than the v1 schema; truncated",
		"folder_id", folder.ID, "field", field, "runes", utf8.RuneCountInString(v), "max", max)
	return string([]rune(v)[:max])
}

const folderPopulationTTL = 10 * time.Minute

func folderPopulationKey(folder *catalogclient.Folder, includeNSFW bool) string {
	return "kungal:folder-population:v1:" + strconv.FormatInt(folder.ID, 10) + ":" + folder.UpdatedAt + ":" + workrepr.ContentLimit(includeNSFW)
}

// Catalog's folder item list carries no content gate, so counting what a reader
// may page through means reading every work in the folder.
func (s *Service) folderPopulation(ctx context.Context, folder *catalogclient.Folder, items []catalogclient.FolderItem, includeNSFW bool) ([]int, *problem.Problem) {
	key := folderPopulationKey(folder, includeNSFW)
	if s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var cached []int
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}
	shared, err, _ := s.popFlight.Do(key, func() (any, error) {
		bctx := context.WithoutCancel(ctx)
		population, p := s.buildFolderPopulation(bctx, folder, items, includeNSFW)
		if p != nil {
			return nil, p
		}
		if s.rdb != nil {
			if raw, mErr := json.Marshal(population); mErr == nil {
				if sErr := s.rdb.Set(bctx, key, raw, folderPopulationTTL).Err(); sErr != nil {
					slog.Warn("collection: folder population cache write failed", "folder_id", folder.ID, "err", sErr)
				}
			}
		}
		return population, nil
	})
	if err != nil {
		var p *problem.Problem
		if errors.As(err, &p) {
			return nil, p
		}
		return nil, problem.Unavailable(err)
	}
	return shared.([]int), nil
}

func (s *Service) buildFolderPopulation(ctx context.Context, folder *catalogclient.Folder, items []catalogclient.FolderItem, includeNSFW bool) ([]int, *problem.Problem) {
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, int(it.WorkID))
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, ids, workrepr.RowInclude, workrepr.ContentLimit(includeNSFW))
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	population := make([]int, 0, len(ids))
	for _, id := range ids {
		row, ok := rows[id]
		if !ok {
			if includeNSFW {
				slog.Warn("collection: catalog did not render work, dropped", "work_id", id, "folder_id", folder.ID)
			}
			continue
		}
		if !client.CatalogItemRenderable(&row) {
			slog.Warn("collection: catalog row not renderable, dropped", "work_id", id, "folder_id", folder.ID)
			continue
		}
		if !includeNSFW && workNSFW(&row) {
			continue
		}
		population = append(population, id)
	}
	return population, nil
}
