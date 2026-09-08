package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const previewCoversPerCollection = 4

// A collection is a catalog folder. galgame_collection is the alias table that
// keeps this site's own ids alive (migration 091) and nothing else; every read
// and write below goes to /v2/me/folders or /v2/folders with the signed-in
// user's own access token, which is why every entry point takes one.
//
// The token is the user's OAuth access token and must carry folder:read /
// folder:write. A session minted before those scopes were requested is 403
// SCOPE_REQUIRED upstream and surfaces here as ErrReauthRequired (code 235):
// a refresh mints from the grant recorded at authorization, so signing in again
// is the one cure the reader holds themselves, and the one message worth
// showing them. Infra can also widen the grant rows in place for everyone at
// once, which is faster and invisible — but that is a message to send, not
// something this site can do.
type CollectionService struct {
	collectionRepo *repository.GalgameCollectionRepository
	galgameService *GalgameService
	galgameClient  *client.GalgameClient
	userClient     *userclient.Client
	catalog        *catalogclient.Client
	check          *gate.CheckService
	scan           *gate.ScanService
	helpers        InteractionHelpers
}

func NewCollectionService(
	collectionRepo *repository.GalgameCollectionRepository,
	galgameService *GalgameService,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	catalog *catalogclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *CollectionService {
	return &CollectionService{
		collectionRepo: collectionRepo,
		galgameService: galgameService,
		galgameClient:  galgameClient,
		userClient:     userClient,
		catalog:        catalog,
		check:          check,
		scan:           scan,
	}
}

func collectionErr(err error, fallback string) *errors.AppError {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("收藏夹不存在")
	// A grant too narrow for folder:read and an expired session are different
	// faults with different cures, and folding them together cost an outage on
	// 2026-09-08: this returned code 205, whose client-side handler re-checks
	// /api/user/status — which answers from this site's own cookie, finds it
	// perfectly healthy, and returns without a word. Every collection read
	// failed and nobody was told. Code 235 is what the five other scope-starved
	// paths here already use (playtime, cover votes, edits, submissions, image
	// upload) and its handler says the one thing that actually fixes it.
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return errors.ErrReauthRequired("收藏夹需要新的授权，请退出登录后重新登录以授予该权限")
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return errors.ErrAuthExpired()
	}
	var apiErr *catalogclient.UserAPIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.Status {
		case 403:
			return errors.ErrForbidden("你没有权限操作这个收藏夹")
		case 422:
			return errors.ErrBadRequest(folderRefusalText(apiErr.Message))
		}
	}
	slog.Error("collection: catalog call failed", "err", err)
	return errors.ErrInternal(fallback)
}

// Upstream answers a 422 in English, and this face has always spoken Chinese.
// A hand-maintained map over another service's prose is a bad shape in general;
// it is here because the alternative is showing readers a sentence in a
// language the rest of the page is not in, and the fallback is that sentence,
// so a phrase that changes upstream degrades rather than breaks. The two caps
// are the only 422s a request that passed this site's own validation can hit.
func folderRefusalText(upstream string) string {
	switch {
	case strings.Contains(upstream, "may keep at most"):
		return "收藏夹数量已达上限"
	case strings.Contains(upstream, "may hold at most"):
		return "这个收藏夹已经装满了"
	case strings.Contains(upstream, "default folder cannot be deleted"):
		return "默认收藏夹不能删除"
	}
	return upstream
}

func (s *CollectionService) Create(ctx context.Context, userID int, token string, req *dto.CreateCollectionRequest) (int, *errors.AppError) {
	moderationText := gate.ComposeText(req.Name, req.Description)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return 0, gate.ErrContentBlocked()
	}

	folder, err := s.catalog.CreateFolder(ctx, token, catalogclient.FolderWrite{
		Name: &req.Name, Description: &req.Description, Visibility: &req.Visibility,
	})
	if err != nil {
		return 0, collectionErr(err, "创建收藏夹失败")
	}
	id, mintErr := s.collectionRepo.MintAlias(s.collectionRepo.DB(), userID, folder.ID)
	if mintErr != nil {
		// The folder exists upstream; the alias is what this site addresses it
		// by. Leaving the folder behind is better than deleting somebody's
		// collection because a local insert failed — the next list call mints
		// the alias lazily.
		slog.Error("collection: folder created but alias insert failed",
			"folder_id", folder.ID, "user_id", userID, "err", mintErr)
		return 0, errors.ErrInternal("创建收藏夹失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameCollection, "subject_id", id, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameCollection, strconv.Itoa(id), moderationText, int64(userID))
	return id, nil
}

func (s *CollectionService) Update(ctx context.Context, userID int, token string, canEditAny bool, cid int, req *dto.UpdateCollectionRequest) *errors.AppError {
	alias, appErr := s.aliasForMutation(cid, userID, canEditAny)
	if appErr != nil {
		return appErr
	}
	if req.Name == nil && req.Description == nil && req.Visibility == nil {
		return errors.ErrBadRequest("没有要修改的内容")
	}

	// Only what the request actually changes is scanned. The unchanged half
	// was scanned when it was set, and reading it back to recompose the pair
	// would need a folder read this path does not otherwise make — and cannot
	// make at all when a moderator is acting on somebody's private folder.
	moderationText := gate.ComposeText(derefOr(req.Name, ""), derefOr(req.Description, ""))
	authorID := int64(alias.UserID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return gate.ErrContentBlocked()
	}

	write := catalogclient.FolderWrite{Name: req.Name, Description: req.Description, Visibility: req.Visibility}
	var err error
	if alias.UserID == userID {
		_, err = s.catalog.PatchFolder(ctx, token, alias.CatalogFolderID, write)
	} else {
		_, err = s.catalog.ModeratePatchFolder(ctx, token, alias.CatalogFolderID, write)
	}
	if err != nil {
		return collectionErr(err, "更新收藏夹失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameCollection, "subject_id", cid, "author_id", alias.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameCollection, strconv.Itoa(cid), moderationText, int64(alias.UserID))
	return nil
}

func (s *CollectionService) Delete(ctx context.Context, userID int, token string, canDeleteAny bool, cid int) *errors.AppError {
	alias, appErr := s.aliasForMutation(cid, userID, canDeleteAny)
	if appErr != nil {
		return appErr
	}

	// The local favourite counter only backs the "most favourited" ranking; it
	// is computed from this site's own writes, so it is adjusted only on the
	// path where this site can see what is being removed. A moderator deleting
	// somebody else's folder cannot read its contents and does not try.
	var orphaned []int
	if alias.UserID == userID {
		mine, mErr := s.catalog.MyFolders(ctx, token)
		if mErr != nil {
			return collectionErr(mErr, "删除收藏夹失败")
		}
		// Upstream refuses this too, in English. Refusing here keeps the
		// sentence this face has always shown and saves a round trip.
		for _, f := range mine {
			if f.ID == alias.CatalogFolderID && f.IsDefault {
				return errors.ErrForbidden("默认收藏夹不能删除")
			}
		}
		orphaned = s.worksLeavingTheLibrary(ctx, token, alias.CatalogFolderID, mine)
	}

	var err error
	if alias.UserID == userID {
		err = s.catalog.DeleteFolder(ctx, token, alias.CatalogFolderID)
	} else {
		err = s.catalog.ModerateDeleteFolder(ctx, token, alias.CatalogFolderID)
	}
	if err != nil {
		return collectionErr(err, "删除收藏夹失败")
	}
	if delErr := s.collectionRepo.DeleteAlias(cid); delErr != nil {
		slog.Error("collection: folder deleted but alias remains", "cid", cid, "err", delErr)
	}
	if len(orphaned) > 0 {
		_ = s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
			return s.collectionRepo.DecrementFavoriteCounts(tx, orphaned)
		})
	}
	return nil
}

// worksLeavingTheLibrary answers which works this folder holds that no other
// folder of the same owner holds — the ones whose favourite count drops when
// the folder goes.
func (s *CollectionService) worksLeavingTheLibrary(ctx context.Context, token string, folderID int64, folders []catalogclient.Folder) []int {
	items, err := s.catalog.MyFolderItems(ctx, token, folderID)
	if err != nil || len(items) == 0 {
		return nil
	}
	elsewhere := map[int64]bool{}
	for _, f := range folders {
		if f.ID == folderID {
			continue
		}
		others, oErr := s.catalog.MyFolderItems(ctx, token, f.ID)
		if oErr != nil {
			return nil
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
	return out
}

func (s *CollectionService) SetMembership(ctx context.Context, userID int, token string, galgameID int, targetIDs []int) *errors.AppError {
	targetIDs = dedupInts(targetIDs)
	owned, err := s.collectionRepo.FolderIDsOwnedBy(userID, targetIDs)
	if err != nil {
		return errors.ErrInternal("读取收藏夹失败")
	}
	if len(owned) != len(targetIDs) {
		return errors.ErrForbidden("收藏夹不存在或不属于您")
	}
	target := map[int64]bool{}
	for _, fid := range owned {
		target[fid] = true
	}

	holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(galgameID))
	if err != nil {
		return collectionErr(err, "读取收藏状态失败")
	}
	current := map[int64]bool{}
	for _, f := range holding {
		current[f.ID] = true
	}

	for fid := range target {
		if current[fid] {
			continue
		}
		if err := s.catalog.PutFolderItem(ctx, token, fid, int64(galgameID)); err != nil {
			return collectionErr(err, "更新收藏失败")
		}
	}
	for fid := range current {
		if target[fid] {
			continue
		}
		if err := s.catalog.DeleteFolderItem(ctx, token, fid, int64(galgameID)); err != nil {
			return collectionErr(err, "更新收藏失败")
		}
	}

	firstAdd := len(current) == 0 && len(target) > 0
	lastRemove := len(current) > 0 && len(target) == 0
	if !firstAdd && !lastRemove {
		return nil
	}

	// Local side effects follow the upstream write, never precede it: points
	// and a message for a favourite that was never stored would be worse than
	// a favourite whose points were missed.
	ownerID, name := s.galgameService.fetchOwnerAndName(ctx, galgameID)
	delta := 1
	if lastRemove {
		delta = -1
	}
	txErr := s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.collectionRepo.EnsureGalgameLocal(tx, galgameID); err != nil {
			return err
		}
		if err := s.collectionRepo.AdjustGalgameFavoriteCount(tx, galgameID, delta); err != nil {
			return err
		}
		if ownerID == 0 || ownerID == userID {
			return nil
		}
		s.helpers.AdjustMoemoepoint(tx, ownerID, delta,
			moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))
		if firstAdd {
			return s.helpers.CreateGalgameMessageWithContent(tx, userID, ownerID, "favorite", name, galgameID)
		}
		return nil
	})
	if txErr != nil {
		slog.Error("collection: membership stored but local bookkeeping failed",
			"galgame_id", galgameID, "user_id", userID, "err", txErr)
	}
	return nil
}

func (s *CollectionService) GetMyCollectionsForGalgame(ctx context.Context, userID int, token string, galgameID int) ([]dto.MyCollectionForGalgame, *errors.AppError) {
	folders, err := s.catalog.MyFolders(ctx, token)
	if err != nil {
		return nil, collectionErr(err, "读取收藏夹失败")
	}
	if len(folders) == 0 {
		created, cErr := s.ensureDefault(ctx, token)
		if cErr != nil {
			return nil, cErr
		}
		folders = created
	}
	holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(galgameID))
	if err != nil {
		return nil, collectionErr(err, "读取收藏状态失败")
	}
	contains := map[int64]bool{}
	for _, f := range holding {
		contains[f.ID] = true
	}

	sortFolders(folders)
	aliases, aErr := s.aliasesFor(userID, folders)
	if aErr != nil {
		return nil, aErr
	}
	out := make([]dto.MyCollectionForGalgame, 0, len(folders))
	for _, f := range folders {
		out = append(out, dto.MyCollectionForGalgame{
			ID:         aliases[f.ID],
			Name:       f.Name,
			Visibility: f.Visibility,
			IsDefault:  f.IsDefault,
			ItemCount:  f.ItemCount,
			Contains:   contains[f.ID],
		})
	}
	return out, nil
}

// The default folder is unnamed on purpose: the forum stored its default
// collections with name = ” and renders the label from the owner's name, and
// the catalog backfill carried that through (upstream deviation 112).
func (s *CollectionService) ensureDefault(ctx context.Context, token string) ([]catalogclient.Folder, *errors.AppError) {
	empty, isDefault := "", true
	pub := model.CollectionPublic
	folder, err := s.catalog.CreateFolder(ctx, token, catalogclient.FolderWrite{
		Name: &empty, Description: &empty, Visibility: &pub, IsDefault: &isDefault,
	})
	if err != nil {
		return nil, collectionErr(err, "创建默认收藏夹失败")
	}
	return []catalogclient.Folder{*folder}, nil
}

func (s *CollectionService) GetDetail(ctx context.Context, viewerID int, token string, cid, page, limit int, isSFW bool) (*dto.CollectionDetail, *errors.AppError) {
	alias, err := s.collectionRepo.AliasByID(cid)
	if err != nil {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}
	isOwner := viewerID == alias.UserID

	owner, _, _ := s.userClient.User(ctx, alias.UserID)
	if !userclient.IsRenderable(owner) && !isOwner {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}

	var folder *catalogclient.Folder
	var items []catalogclient.FolderItem
	var cErr error
	if isOwner && token != "" {
		if folder, cErr = s.catalog.MyFolder(ctx, token, alias.CatalogFolderID); cErr == nil {
			items, cErr = s.catalog.MyFolderItems(ctx, token, alias.CatalogFolderID)
		}
	} else {
		// The public lane 404s a private folder for everyone including its
		// owner, which is exactly the answer this face already gave.
		if folder, cErr = s.catalog.PublicFolder(ctx, alias.CatalogFolderID); cErr == nil {
			items, cErr = s.catalog.PublicFolderItems(ctx, alias.CatalogFolderID)
		}
	}
	if cErr != nil {
		return nil, collectionErr(cErr, "读取收藏夹失败")
	}

	// Catalog pages items by updated_at ascending because that keyset is a
	// sync watermark; this face has always shown newest-added first and is
	// page-numbered, so the ordering and the slicing happen here.
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	total := int64(len(items))
	ids := pageOfWorkIDs(items, page, limit)
	cards, appErr := s.galgameService.HydrateCardsByIDs(ctx, ids, isSFW)
	if appErr != nil {
		cards = []dto.GalgameListCard{}
	}

	return &dto.CollectionDetail{
		ID:          cid,
		Name:        folder.Name,
		Description: folder.Description,
		Visibility:  folder.Visibility,
		IsDefault:   folder.IsDefault,
		ItemCount:   folder.ItemCount,
		IsOwner:     isOwner,
		Owner:       dto.UserBrief{ID: owner.ID, Name: owner.Name, Avatar: owner.Avatar},
		Galgames:    cards,
		Total:       total,
		Created:     parseCatalogTime(folder.CreatedAt),
		Updated:     parseCatalogTime(folder.UpdatedAt),
	}, nil
}

func (s *CollectionService) ListForUser(ctx context.Context, ownerID, viewerID int, token string, page, limit int, isSFW bool) ([]dto.CollectionSummary, int64, *errors.AppError) {
	owner, _, _ := s.userClient.User(ctx, ownerID)
	if !userclient.IsRenderable(owner) && viewerID != ownerID {
		return []dto.CollectionSummary{}, 0, nil
	}

	var folders []catalogclient.Folder
	var err error
	if viewerID == ownerID && token != "" {
		folders, err = s.catalog.MyFolders(ctx, token)
	} else {
		folders, err = s.catalog.PublicFolders(ctx, int64(ownerID))
	}
	if err != nil {
		return nil, 0, collectionErr(err, "读取收藏夹列表失败")
	}

	sortFolders(folders)
	total := int64(len(folders))
	pageFolders := pageOfFolders(folders, page, limit)
	if len(pageFolders) == 0 {
		return []dto.CollectionSummary{}, total, nil
	}
	aliases, aErr := s.aliasesFor(ownerID, pageFolders)
	if aErr != nil {
		return nil, 0, aErr
	}
	previewToken := ""
	if viewerID == ownerID {
		previewToken = token
	}
	coverByFolder := s.resolvePreviewCovers(ctx, previewToken, pageFolders, isSFW)

	out := make([]dto.CollectionSummary, 0, len(pageFolders))
	for _, f := range pageFolders {
		covers := coverByFolder[f.ID]
		if covers == nil {
			covers = []string{}
		}
		out = append(out, dto.CollectionSummary{
			ID:            aliases[f.ID],
			Name:          f.Name,
			Description:   f.Description,
			Visibility:    f.Visibility,
			IsDefault:     f.IsDefault,
			ItemCount:     f.ItemCount,
			PreviewCovers: covers,
			Created:       parseCatalogTime(f.CreatedAt),
			Updated:       parseCatalogTime(f.UpdatedAt),
		})
	}
	return out, total, nil
}

func (s *CollectionService) resolvePreviewCovers(ctx context.Context, token string, folders []catalogclient.Folder, isSFW bool) map[int64][]string {
	result := make(map[int64][]string, len(folders))
	byFolder := make(map[int64][]int, len(folders))
	allGids := []int{}
	for _, f := range folders {
		result[f.ID] = []string{}
		if f.ItemCount == 0 {
			continue
		}
		items, err := s.catalog.FolderPreviewItems(ctx, token, f.ID, previewCoversPerCollection)
		if err != nil {
			continue
		}
		for _, it := range items {
			byFolder[f.ID] = append(byFolder[f.ID], int(it.WorkID))
			allGids = append(allGids, int(it.WorkID))
		}
	}
	if len(allGids) == 0 {
		return result
	}
	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, dedupInts(allGids), isSFW)
	if appErr != nil {
		return result
	}
	for fid, gids := range byFolder {
		covers := make([]string, 0, len(gids))
		for _, gid := range gids {
			if b, ok := briefMap[gid]; ok && b.EffectiveBannerURL != "" {
				covers = append(covers, b.EffectiveBannerURL)
			}
		}
		result[fid] = covers
	}
	return result
}

func (s *CollectionService) aliasForMutation(cid, userID int, canMutateAny bool) (*repository.CollectionAlias, *errors.AppError) {
	alias, err := s.collectionRepo.AliasByID(cid)
	if err != nil {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}
	if alias.UserID != userID && !canMutateAny {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}
	return alias, nil
}

func (s *CollectionService) aliasesFor(ownerID int, folders []catalogclient.Folder) (map[int64]int, *errors.AppError) {
	ids := make([]int64, 0, len(folders))
	for _, f := range folders {
		ids = append(ids, f.ID)
	}
	aliases, err := s.collectionRepo.AliasesForFolders(ownerID, ids)
	if err != nil {
		return nil, errors.ErrInternal("读取收藏夹失败")
	}
	return aliases, nil
}

func sortFolders(folders []catalogclient.Folder) {
	sort.SliceStable(folders, func(i, j int) bool {
		if folders[i].IsDefault != folders[j].IsDefault {
			return folders[i].IsDefault
		}
		return folders[i].UpdatedAt > folders[j].UpdatedAt
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

func pageOfWorkIDs(items []catalogclient.FolderItem, page, limit int) []int {
	start := (page - 1) * limit
	if start >= len(items) {
		return []int{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]int, 0, end-start)
	for _, it := range items[start:end] {
		out = append(out, int(it.WorkID))
	}
	return out
}

func parseCatalogTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

func dedupInts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
