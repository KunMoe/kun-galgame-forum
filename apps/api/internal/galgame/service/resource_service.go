package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/storelink"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// catalog answers one page of a keyword search, so the game-name half of a
// resource match is capped at it.
const resourceGalgameMatchCap = 100

type ResourceService struct {
	resourceRepo  *repository.ResourceRepository
	galgameRepo   *repository.GalgameRepository
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
	userClient    *userclient.Client
	check         *gate.CheckService
	scan          *gate.ScanService
	helpers       InteractionHelpers
	linkChecker   *linkcheck.Client
	storeLinks    *storelink.Resolver
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	galgameRepo *repository.GalgameRepository,
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
	userClient *userclient.Client,
	linkChecker *linkcheck.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
	storeLinks *storelink.Resolver,
) *ResourceService {
	return &ResourceService{
		resourceRepo:  resourceRepo,
		galgameRepo:   galgameRepo,
		galgameClient: galgameClient,
		catalog:       catalog,
		userClient:    userClient,
		check:         check,
		scan:          scan,
		linkChecker:   linkChecker,
		storeLinks:    storeLinks,
	}
}

func (s *ResourceService) dlsiteLinks(b client.GalgameBrief) storelink.Links {
	return s.storeLinks.Resolve(b.ID, b.DlsiteWorkno())
}

func resourceModerationText(note string, links []string) string {
	parts := make([]string, 0, 1+len(links))
	parts = append(parts, note)
	parts = append(parts, links...)
	return gate.ComposeText(parts...)
}

func (s *ResourceService) GetResourceList(
	ctx context.Context,
	req *dto.ResourceListRequest,
	currentUserID int,
	isSFW bool,
) (*dto.ResourceListPage, *errors.AppError) {
	if keywords := strings.Fields(strings.TrimSpace(req.Keywords)); len(keywords) > 0 {
		ids := s.MatchedGalgameIDs(ctx, req.Keywords, isSFW)
		return s.Search(ctx, keywords, ids, req.Page, req.Limit, currentUserID, isSFW), nil
	}

	total := s.resourceRepo.CountAll(isSFW)
	rows := s.resourceRepo.ListPaginated(req.Page, req.Limit, isSFW)

	return &dto.ResourceListPage{
		Resources: s.hydrateCards(ctx, rows, currentUserID, isSFW),
		Total:     total,
	}, nil
}

// MatchedGalgameIDs is the half of a resource keyword match the forum cannot do
// itself: it keeps no local copy of a game's name, so the games a keyword hits
// have to come back from catalog as ids. A failed catalog search costs that
// half, not the search — the note half is local and answers on its own.
func (s *ResourceService) MatchedGalgameIDs(ctx context.Context, raw string, isSFW bool) []int {
	if s.galgameClient == nil {
		return nil
	}
	q := url.Values{
		"q":     {raw},
		"page":  {"1"},
		"limit": {strconv.Itoa(resourceGalgameMatchCap)},
		"sort":  {"relevance"},
	}
	client.ApplyWorksGate(q, isSFW)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil
	}
	ids := make([]int, 0, len(res.Items))
	for i := range res.Items {
		if gid := client.CatalogItemGID(&res.Items[i]); gid > 0 {
			ids = append(ids, gid)
		}
	}
	return ids
}

// Search answers the search page's resource lane. galgameIDs are the games
// catalog matched on the same keyword; the note match is the repository's.
func (s *ResourceService) Search(
	ctx context.Context,
	keywords []string,
	galgameIDs []int,
	page, limit, currentUserID int,
	isSFW bool,
) *dto.ResourceListPage {
	rows, total := s.resourceRepo.SearchPaginated(keywords, galgameIDs, page, limit, isSFW)
	return &dto.ResourceListPage{
		Resources: s.hydrateCards(ctx, rows, currentUserID, isSFW),
		Total:     total,
	}
}

func (s *ResourceService) hydrateCards(
	ctx context.Context,
	rows []model.GalgameResourceRow,
	currentUserID int,
	isSFW bool,
) []dto.ResourceCard {
	galgameIDs, userIDs := collectIDs(rows)
	briefMap := s.fetchGalgameBriefsPublic(ctx, galgameIDs, isSFW)
	userMap := s.userClient.Hydrate(ctx, userIDs)

	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		resourceIDs[i] = r.ID
	}
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		b, hasBrief := briefMap[r.GalgameID]
		if !hasBrief {
			continue
		}
		card := rowToCard(r, u, likedSet[r.ID])
		card.GalgameName = b.Name
		store := s.dlsiteLinks(b)
		card.DlsitePurchaseURL, card.DlsiteCouponURL, card.DlsiteCampaignName = store.PurchaseURL, store.CouponURL, store.CampaignName
		cards = append(cards, card)
	}
	return cards
}

type ResourceNotFound struct{}

func (s *ResourceService) GetResourceDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDetailPage, *ResourceNotFound, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, &ResourceNotFound{}, nil
	}

	ownerUser, _, _ := s.userClient.User(ctx, row.UserID)
	if !userclient.IsRenderable(ownerUser) {
		return nil, &ResourceNotFound{}, nil
	}

	s.resourceRepo.IncrementView(resourceID)
	row.View++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)

	resource := rowToMeta(row, links, isLiked, ownerUser)

	if b, ok := s.fetchGalgameBriefs(ctx, []int{row.GalgameID})[row.GalgameID]; ok {
		store := s.dlsiteLinks(b)
		resource.DlsitePurchaseURL, resource.DlsiteCouponURL, resource.DlsiteCampaignName = store.PurchaseURL, store.CouponURL, store.CampaignName
	}

	galgameSummary := s.buildGalgameSummary(ctx, row.GalgameID)

	recRows := s.resourceRepo.FindRecommendations(row.GalgameID, resourceID, 6)
	recommendations := s.buildRecommendations(ctx, recRows, row.GalgameID, currentUserID)

	return &dto.ResourceDetailPage{
		Galgame:         galgameSummary,
		Resource:        resource,
		Recommendations: recommendations,
	}, nil, nil
}

func (s *ResourceService) GetResourceDownloadDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDownloadDetail, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	owner, _, _ := s.userClient.User(ctx, row.UserID)
	if !userclient.IsRenderable(owner) {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	s.resourceRepo.IncrementDownload(resourceID)
	row.Download++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)

	detail := rowToDownloadDetail(row, links, isLiked, owner)
	return &detail, nil
}

func (s *ResourceService) GetGalgameResources(
	ctx context.Context,
	req *dto.GalgameResourcesRequest,
	currentUserID int,
) ([]dto.ResourceCard, *errors.AppError) {
	rows := s.resourceRepo.FindByGalgameID(req.GalgameID)

	userIDs := make([]int, len(rows))
	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		resourceIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	var store storelink.Links
	if len(rows) > 0 {
		if b, ok := s.fetchGalgameBriefs(ctx, []int{req.GalgameID})[req.GalgameID]; ok {
			store = s.dlsiteLinks(b)
		}
	}

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := rowToCard(r, u, likedSet[r.ID])
		card.DlsitePurchaseURL, card.DlsiteCouponURL, card.DlsiteCampaignName = store.PurchaseURL, store.CouponURL, store.CampaignName
		cards = append(cards, card)
	}
	return cards, nil
}

func (s *ResourceService) CreateResource(
	ctx context.Context,
	userID int,
	accessToken string,
	req *dto.CreateGalgameResourceRequest,
) *errors.AppError {
	if s.resourceRepo.IsResourcePublishBanned(req.GalgameID) {
		return errors.ErrForbidden("该游戏已被禁止发布下载资源")
	}
	moderationText := resourceModerationText(req.Note, req.Link)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return gate.ErrContentBlocked()
	}

	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	res := &model.GalgameResource{
		Type:      req.Type,
		Language:  req.Language,
		Platform:  req.Platform,
		Size:      req.Size,
		Code:      req.Code,
		Password:  req.Password,
		Note:      req.Note,
		GalgameID: req.GalgameID,
		UserID:    userID,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if s.galgameRepo != nil {
			if err := s.galgameRepo.PublishLocal(tx, req.GalgameID); err != nil {
				return err
			}
			if err := s.galgameRepo.SetCreatorIfUnset(tx, req.GalgameID, userID); err != nil {
				return err
			}
		} else {
			tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&model.GalgameLocal{ID: req.GalgameID})
		}
		if err := s.resourceRepo.Create(tx, res); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, res.ID, providers); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviderNames(tx, res.ID, providerNames); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, res.ID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.AdjustLocalResourceCount(tx, req.GalgameID, 1); err != nil {
			return err
		}
		if err := s.resourceRepo.TouchGalgameUpdated(tx, req.GalgameID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, constants.RewardCreateResource,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_resource", req.GalgameID))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("创建 Galgame 资源失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameResource, "subject_id", res.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameResource, strconv.Itoa(res.ID), moderationText, int64(userID))
	s.claimOnFirstResource(ctx, accessToken, req.GalgameID)
	return nil
}

func claimRefusedByState(appErr *errors.AppError) bool {
	return appErr != nil && appErr.StatusCode == http.StatusConflict
}

func (s *ResourceService) claimOnFirstResource(ctx context.Context, accessToken string, gid int) {
	if accessToken == "" || s.catalog == nil || !s.catalog.Configured() {
		return
	}
	if _, appErr := adoptAndPublish(ctx, s.catalog, accessToken, int64(gid)); appErr != nil {
		if claimRefusedByState(appErr) {
			slog.Info("resource: catalog 作品已有归属, 跳过静默认领", "gid", gid)
			return
		}
		slog.Warn("resource: 静默认领 catalog 作品失败", "gid", gid, "error", appErr)
	}
}

func (s *ResourceService) SetResourcePublishBan(galgameID int, banned bool) *errors.AppError {
	if err := s.resourceRepo.SetResourcePublishBanned(galgameID, banned); err != nil {
		return errors.ErrInternal("更新资源发布禁止状态失败")
	}
	return nil
}

func (s *ResourceService) UpdateResource(
	ctx context.Context,
	userID int, canModerate bool,
	req *dto.UpdateGalgameResourceRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if row.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限更新这个 Galgame 资源")
	}
	if s.resourceRepo.IsResourcePublishBanned(row.GalgameID) {
		return errors.ErrForbidden("该游戏已被禁止发布下载资源")
	}

	moderationText := resourceModerationText(req.Note, req.Link)
	authorID := int64(row.UserID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return gate.ErrContentBlocked()
	}

	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	fields := map[string]any{
		"type":     req.Type,
		"language": req.Language,
		"platform": req.Platform,
		"size":     req.Size,
		"code":     req.Code,
		"password": req.Password,
		"note":     req.Note,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateFields(tx, req.GalgameResourceID, fields); err != nil {
			return err
		}
		if err := s.resourceRepo.DeleteLinks(tx, req.GalgameResourceID); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, req.GalgameResourceID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, req.GalgameResourceID, providers); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviderNames(tx, req.GalgameResourceID, providerNames); err != nil {
			return err
		}
		return s.resourceRepo.TouchGalgameUpdated(tx, row.GalgameID)
	})
	if txErr != nil {
		return errors.ErrInternal("更新 Galgame 资源失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameResource, "subject_id", req.GalgameResourceID, "author_id", row.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameResource, strconv.Itoa(req.GalgameResourceID), moderationText, int64(row.UserID))
	return nil
}

func (s *ResourceService) DeleteResource(
	userID int, canModerate bool, resourceID int,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除这个 Galgame 资源")
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.helpers.AdjustMoemoepoint(tx, row.UserID, -(row.LikeCount + 5),
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("galgame_resource", resourceID))
		if err := s.resourceRepo.DeleteByID(tx, resourceID); err != nil {
			return err
		}
		return s.resourceRepo.AdjustLocalResourceCount(tx, row.GalgameID, -1)
	})
	if txErr != nil {
		return errors.ErrInternal("删除 Galgame 资源失败")
	}
	return nil
}

func (s *ResourceService) ToggleLike(
	userID int,
	req *dto.ToggleResourceLikeRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到该资源")
	}
	if row.UserID == userID {
		return errors.ErrBadRequest("您不能给自己的资源点赞")
	}

	links := s.resourceRepo.FindLinks(req.GalgameResourceID)
	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, has := s.resourceRepo.FindLike(tx, req.GalgameResourceID, userID)
		var delta int
		if has {
			if err := s.resourceRepo.DeleteLike(tx, existing); err != nil {
				return err
			}
			delta = -1
		} else {
			if err := s.resourceRepo.CreateLike(tx, req.GalgameResourceID, userID); err != nil {
				return err
			}
			delta = 1
		}
		if err := s.resourceRepo.AdjustLikeCount(tx, req.GalgameResourceID, delta); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, row.UserID, delta,
			moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_resource", req.GalgameResourceID))
		return s.helpers.CreateGalgameMessageWithContent(
			tx, userID, row.UserID, "liked", preview, row.GalgameID,
		)
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// Anyone signed in may mark a resource expired, so only the owner being able to
// undo it left a wrongly-expired resource stuck for good once its publisher went
// quiet — 「管理核实过资源仍然可以输入提取码, 仍可正常下载, 但无权限更改为正常资源」.
func (s *ResourceService) MarkValid(userID int, canEditAny bool, resourceID int) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if row.UserID != userID && !canEditAny {
		return errors.ErrForbidden("您没有权限修改这个 Galgame 资源")
	}
	if err := s.resourceRepo.UpdateStatus(s.resourceRepo.DB(), resourceID, 0); err != nil {
		return errors.ErrInternal("更新失败")
	}
	return nil
}

func (s *ResourceService) MarkExpired(ctx context.Context, userID int, resourceID int) (*dto.ReportExpireResult, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.Status == 1 {
		return nil, errors.ErrBadRequest("该资源已经被标记为失效")
	}

	links := s.resourceRepo.FindLinks(resourceID)

	verdict := ""
	if s.linkChecker != nil && len(links) > 0 {
		verdict = string(s.linkChecker.CheckShare(ctx, links, row.Code))
		if verdict == string(linkcheck.StatusAlive) {
			return &dto.ReportExpireResult{Verdict: verdict, Marked: false}, nil
		}
	}

	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateStatus(tx, resourceID, 1); err != nil {
			return err
		}
		return s.helpers.CreateGalgameMessageWithContent(
			tx, userID, row.UserID, "expired", preview, row.GalgameID,
		)
	})
	if txErr != nil {
		return nil, errors.ErrInternal("更新失败")
	}
	return &dto.ReportExpireResult{Verdict: verdict, Marked: true}, nil
}

func (s *ResourceService) fetchGalgameBriefs(
	ctx context.Context,
	galgameIDs []int,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	briefMap, _ := s.galgameClient.GetBatch(ctx, galgameIDs)
	if briefMap == nil {
		return map[int]client.GalgameBrief{}
	}
	return briefMap
}

func (s *ResourceService) fetchGalgameBriefsPublic(
	ctx context.Context,
	galgameIDs []int,
	isSFW bool,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	briefMap, _ := s.galgameClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	if briefMap == nil {
		return map[int]client.GalgameBrief{}
	}
	return briefMap
}

func (s *ResourceService) buildGalgameSummary(
	ctx context.Context,
	galgameID int,
) dto.ResourceGalgameSummary {
	summary := dto.ResourceGalgameSummary{
		ID:       galgameID,
		Platform: []string{}, Language: []string{}, Type: []string{},
	}

	briefMap := s.fetchGalgameBriefs(ctx, []int{galgameID})
	b, ok := briefMap[galgameID]
	if !ok {
		return summary
	}

	aggs := s.resourceRepo.AggregateByGalgame(galgameID)
	platforms, languages, types := collectAggregate(aggs)
	local := s.resourceRepo.FindGalgameLocal(galgameID)

	return dto.ResourceGalgameSummary{
		ID:                       b.ID,
		Name:                     b.Name,
		EffectiveBannerHash:      b.EffectiveBannerHash,
		EffectiveBannerURL:       b.EffectiveBannerURL,
		EffectiveBannerWidth:     b.EffectiveBannerWidth,
		EffectiveBannerHeight:    b.EffectiveBannerHeight,
		EffectiveBannerThumbhash: b.EffectiveBannerThumbhash,
		ContentLimit:             b.ContentLimit,
		View:                     local.View,
		ResourceUpdateTime:       utils.RFC3339OrEmpty(local.ResourceUpdateTime),
		OriginalLanguage:         b.OriginalLanguage,
		AgeLimit:                 b.AgeLimit,
		Platform:                 platforms,
		Language:                 languages,
		Type:                     types,
	}
}

func (s *ResourceService) buildRecommendations(
	ctx context.Context,
	rows []model.GalgameResourceRow,
	galgameID int,
	currentUserID int,
) []dto.ResourceCard {
	userIDs := make([]int, len(rows))
	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		resourceIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchGalgameBriefs(ctx, []int{galgameID})
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := rowToCard(r, u, likedSet[r.ID])
		if b, ok := briefMap[galgameID]; ok {
			card.GalgameName = b.Name
		}
		cards = append(cards, card)
	}
	return cards
}
