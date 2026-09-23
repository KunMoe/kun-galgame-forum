package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/storelink"
	"kun-galgame-api/internal/moemoepoint"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
)

type GalgameService struct {
	galgameRepo      *repository.GalgameRepository
	interactionRepo  *repository.GalgameInteractionRepository
	listRepo         *repository.GalgameListRepository
	resourceMetaRepo *repository.GalgameResourceMetaRepository
	detailRatingRepo *repository.GalgameDetailRatingRepository
	contributorRepo  *repository.GalgameContributorRepository
	stateRepo        *userRepo.StateRepository
	galgameClient    *client.GalgameClient
	userClient       *userclient.Client
	catalog          *catalogclient.Client
	helpers          InteractionHelpers
	storeLinks       *storelink.Resolver
}

func NewGalgameService(
	galgameRepo *repository.GalgameRepository,
	interactionRepo *repository.GalgameInteractionRepository,
	listRepo *repository.GalgameListRepository,
	resourceMetaRepo *repository.GalgameResourceMetaRepository,
	detailRatingRepo *repository.GalgameDetailRatingRepository,
	contributorRepo *repository.GalgameContributorRepository,
	stateRepo *userRepo.StateRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	catalog *catalogclient.Client,
	storeLinks *storelink.Resolver,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		interactionRepo:  interactionRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		storeLinks:       storeLinks,
		detailRatingRepo: detailRatingRepo,
		contributorRepo:  contributorRepo,
		stateRepo:        stateRepo,
		galgameClient:    galgameClient,
		userClient:       userClient,
		catalog:          catalog,
	}
}

func (s *GalgameService) ToggleLike(
	ctx context.Context,
	userID, workID int,
) *errors.AppError {
	ownerID, name := s.fetchOwnerAndName(ctx, workID)
	if ownerID == userID {
		return errors.ErrBadRequest("您不能给自己点赞")
	}

	txErr := s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
		liked, err := s.interactionRepo.ToggleLike(tx, userID, workID)
		if err != nil {
			return err
		}
		if !liked {
			s.helpers.AdjustMoemoepoint(tx, ownerID, -1,
				moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", workID))
			return nil
		}
		s.helpers.AdjustMoemoepoint(tx, ownerID, 1,
			moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", workID))
		return s.helpers.CreateGalgameMessageWithContent(tx, userID, ownerID, "liked", name, workID)
	})
	if txErr != nil {
		return errors.ErrInternal("点赞失败")
	}
	return nil
}

// Favorited is the subset of workIDs that sit in any of the reader's folders.
// Walking every membership to paint hearts is what spent user 90769's 10k/day
// quota on 2026-09-20 (3,560 items, 36 catalog pages per call). Holdings
// answers the same question for the ids on screen. An empty workIDs means
// likes only — there is no "all my favourites" list here any more.
// A session with no token, or one minted before the folder scopes, gets its
// likes and an empty favourite list rather than an error.
func (s *GalgameService) GetMyInteractions(ctx context.Context, userID int, token string, workIDs []int) dto.MyGalgameInteractions {
	out := dto.MyGalgameInteractions{
		Liked:     s.interactionRepo.UserLikedGalgames(userID),
		Favorited: []int{},
	}
	if token == "" || s.catalog == nil || len(workIDs) == 0 {
		return out
	}
	ids := make([]int64, 0, len(workIDs))
	for _, id := range workIDs {
		if id > 0 {
			ids = append(ids, int64(id))
		}
	}
	if len(ids) == 0 {
		return out
	}
	holdings, err := s.catalog.MyFolderHoldings(ctx, token, ids)
	if err != nil {
		if stderrors.Is(err, catalogclient.ErrInsufficientScope) {
			warnFoldersScope.warn("galgame: my folders unreadable, token lacks folder:read", "user_id", userID)
		} else {
			slog.Warn("galgame: my folders unreadable", "user_id", userID, "err", err)
		}
		return out
	}
	for _, h := range holdings {
		if h.WorkID == 0 {
			continue
		}
		out.Favorited = append(out.Favorited, int(h.WorkID))
	}
	return out
}

// A work is favourited when it sits in any of the reader's folders. Asking
// upstream costs one request; the alternative is a second copy of the
// memberships in this database, which is what the cutover removed.
func (s *GalgameService) isFavorited(ctx context.Context, token string, workID int) bool {
	if token == "" || s.catalog == nil {
		return false
	}
	folders, err := s.catalog.MyFoldersContaining(ctx, token, int64(workID))
	if err != nil {
		if stderrors.Is(err, catalogclient.ErrInsufficientScope) {
			warnFavoriteScope.warn("galgame: favourite state unreadable, token lacks folder:read", "work_id", workID)
		} else {
			slog.Warn("galgame: favourite state unreadable", "work_id", workID, "err", err)
		}
		return false
	}
	return len(folders) > 0
}

func (s *GalgameService) fetchOwnerAndName(ctx context.Context, workID int) (int, string) {
	return s.ownerOf(workID), truncate(s.entryName(ctx, workID), constants.TextPreviewLength)
}

func (s *GalgameService) ownerOf(workID int) int {
	if s.galgameRepo == nil || workID <= 0 {
		return 0
	}
	row := s.galgameRepo.FindLocal(workID)
	if row.CreatorUserID == nil {
		return 0
	}
	return *row.CreatorUserID
}

func (s *GalgameService) entryName(ctx context.Context, workID int) string {
	if s.galgameClient == nil {
		return ""
	}
	rows, appErr := s.galgameClient.CatalogRowsByWorkIDs(ctx, []int{workID}, "names", "all")
	if appErr != nil {
		return ""
	}
	row, ok := rows[workID]
	if !ok {
		return ""
	}
	brief := client.CatalogItemToBrief(ctx, &row)
	return client.BriefName(&brief)
}

func (s *GalgameService) GetDetail(
	ctx context.Context,
	workID, currentUserID int,
	token string,
	isSFW bool,
) (*dto.GalgameDetail, *errors.AppError) {
	d, found, appErr := s.galgameClient.CatalogWorkDetail(ctx, workID)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该 Galgame")
	}
	g := client.CatalogDetailToFull(ctx, d, workID)
	s.galgameClient.HydrateOfficialLinks(ctx, &g)

	go s.galgameRepo.IncrementView(workID)

	local := s.galgameRepo.FindLocal(workID)
	isLiked := s.interactionRepo.UserLiked(currentUserID, workID)
	isFavorited := s.isFavorited(ctx, token, workID)

	platforms, languages, types := s.resourceMetaRepo.FindResourceMetaByWork(workID)

	ratings := s.buildDetailRatings(ctx, workID, currentUserID, g)

	if owner := s.ownerOf(workID); owner > 0 {
		g.UserID = owner
	}
	g.Contributor = s.contributorsOf(workID)
	users := s.hydrateDetailUsers(ctx, g)
	detail := galgameDetailFromNextMoe(g, users)
	store := s.storeLinks.Resolve(g.ID, g.Refs["dlsite"])
	detail.DlsitePurchaseURL = store.PurchaseURL
	detail.DlsiteCouponURL = store.CouponURL
	detail.DlsiteCampaignName = store.CampaignName
	detail.View = local.View
	detail.ResourceUpdateTime = utils.RFC3339OrEmpty(local.ResourceUpdateTime)
	detail.LikeCount = local.LikeCount
	detail.FavoriteCount = g.FavoriteCount
	detail.ResourcePublishBanned = local.ResourcePublishBanned
	detail.IsOnForum = local.ID != 0
	detail.Indexed = local.Published
	detail.IsLiked = isLiked
	detail.IsFavorited = isFavorited
	detail.Platform = platforms
	detail.Language = languages
	detail.Type = types
	detail.Ratings = ratings
	agg := s.listRepo.BayesianRatings([]int{workID})[workID]
	detail.Rating = agg.Score
	detail.RatingCount = agg.Count
	s.hydrateCoverVotes(ctx, workID, token, detail.Covers)
	detail.MyPlaytime = s.hydrateMyPlaytime(ctx, workID, token)
	if isSFW {
		detail.Tag = withoutSexualTags(detail.Tag)
	}
	return &detail, nil
}

func (s *GalgameService) contributorsOf(workID int) []dto.NextMoeContributor {
	rows := s.contributorRepo.FindContributors(workID, contributorMaxPerGalgame)
	out := make([]dto.NextMoeContributor, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.NextMoeContributor{UserID: int(row.UserID)})
	}
	return out
}

func (s *GalgameService) hydrateDetailUsers(ctx context.Context, g dto.NextMoeGalgameDetailFull) map[string]dto.NextMoeUser {
	uids := make([]int, 0, len(g.Contributor)+1)
	uids = append(uids, g.UserID)
	for _, c := range g.Contributor {
		uids = append(uids, c.UserID)
	}
	umap := s.userClient.Hydrate(ctx, uids)
	users := make(map[string]dto.NextMoeUser, len(umap))
	for id, u := range umap {
		users[strconv.Itoa(id)] = dto.NextMoeUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return users
}

func (s *GalgameService) buildDetailRatings(
	ctx context.Context,
	workID, currentUserID int,
	g dto.NextMoeGalgameDetailFull,
) []dto.GalgameDetailRating {
	rows := s.detailRatingRepo.FindRatingsByGalgame(workID)
	if len(rows) == 0 {
		return []dto.GalgameDetailRating{}
	}

	userIDs := make([]int, len(rows))
	ratingIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		ratingIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	likedSet := s.detailRatingRepo.FindLikedRatingIDs(currentUserID, ratingIDs)

	out := make([]dto.GalgameDetailRating, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		out = append(out, detailRatingFromRow(r, u, likedSet[r.ID], workID, g))
	}
	return out
}

func (s *GalgameService) GetList(
	ctx context.Context,
	req *dto.GalgameListRequest,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	releasedFrom, err := utils.ParseReleaseLowerBound(req.ReleasedFrom)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	releasedTo, err := utils.ParseReleaseUpperBound(req.ReleasedTo)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	releasedMonths, err := utils.ParseMonthSet(req.ReleasedMonths)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	collectedFrom, err := utils.ParseDateLowerBound(req.CollectedFrom, "收录日期")
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	collectedTo, err := utils.ParseDateUpperBound(req.CollectedTo, "收录日期")
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	collectedMonths, err := utils.ParseMonthSet(req.CollectedMonths)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}

	filter := model.GalgameListFilter{
		Type:                 req.Type,
		Language:             req.Language,
		Platform:             req.Platform,
		GameType:             req.GameType,
		SortField:            req.SortField,
		SortOrder:            sortOrder,
		IncludeProviders:     splitCSV(req.IncludeProviders),
		ExcludeOnlyProviders: splitCSV(req.ExcludeOnlyProviders),
		ReleasedFrom:         releasedFrom,
		ReleasedTo:           releasedTo,
		ReleasedMonths:       releasedMonths,
		CollectedFrom:        collectedFrom,
		CollectedTo:          collectedTo,
		CollectedMonths:      collectedMonths,
		MinRatingCount:       req.MinRatingCount,
		MinRating:            req.MinRating,
		ShowNoResource:       req.ShowNoResource,
		Indexed:              req.Indexed,
		Page:                 req.Page,
		Limit:                req.Limit,
	}
	if req.Library {
		return s.catalogLibrary(ctx, req, releasedFrom, releasedTo, isSFW)
	}

	return s.hydrateListCards(ctx, filter, isSFW)
}

// CollectedCalendar lists the (year, month) pairs that have collected rows, so
// the filter UI only offers times the site actually has entries for. It takes
// the reader's SFW gate for the same reason: an option that can only ever
// answer with rows this reader may not see is an option that looks broken.
func (s *GalgameService) CollectedCalendar(isSFW bool) []repository.CollectedMonth {
	return s.listRepo.ListCollectedCalendar(isSFW)
}

func (s *GalgameService) hydrateListCards(
	ctx context.Context,
	filter model.GalgameListFilter,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	if len(filter.RestrictIDs) > 0 && !entityUsesLocalList(filter) {
		return s.hydrateIDPage(ctx, filter, isSFW)
	}
	filter.SFWOnly = isSFW
	ids, total := s.listRepo.ListIDs(filter)
	if len(ids) == 0 {
		return &dto.GalgameListPage{Galgames: []dto.GalgameListCard{}, Total: total}, nil
	}
	cards, appErr := s.HydrateCardsByIDs(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.GalgameListPage{Galgames: cards, Total: total}, nil
}

// Counts a taxonomy sub-set the way hydrateListCards lists it, or the chip ends
// up counting resource-carrying rows against a list of every catalog member.
func (s *GalgameService) countMembers(filter model.GalgameListFilter, isSFW bool) int64 {
	if len(filter.RestrictIDs) > 0 && !entityUsesLocalList(filter) {
		return int64(len(filter.RestrictIDs))
	}
	filter.SFWOnly = isSFW
	filter.Page, filter.Limit = 1, 1
	_, total := s.listRepo.ListIDs(filter)
	return total
}

func (s *GalgameService) HydrateCardsByIDs(
	ctx context.Context,
	ids []int,
	isSFW bool,
) ([]dto.GalgameListCard, *errors.AppError) {
	if len(ids) == 0 {
		return []dto.GalgameListCard{}, nil
	}

	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	localMap := s.galgameRepo.FindLocalBatch(ids)

	userMap := s.userClient.Hydrate(ctx, frozenCreatorIDs(ids, localMap))

	ratingMap := s.listRepo.BayesianRatings(ids)

	metaRows := s.resourceMetaRepo.FindResourceMetaBatch(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	cards := make([]dto.GalgameListCard, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		cards = append(cards, dto.GalgameListCard{
			ID:                         id,
			Name:                       b.Name,
			NameOriginal:               b.NameOriginal,
			User:                       frozenCreatorBrief(localMap[id], userMap),
			ContentLimit:               b.ContentLimit,
			View:                       localMap[id].View,
			LikeCount:                  localMap[id].LikeCount,
			Rating:                     ratingMap[id].Score,
			RatingCount:                ratingMap[id].Count,
			ResourceUpdateTime:         utils.RFC3339OrEmpty(localMap[id].ResourceUpdateTime),
			IsOnForum:                  localMap[id].Published,
			ReleaseDate:                b.ReleaseDate,
			ReleaseDateTBA:             b.ReleaseDateTBA,
			EffectiveBannerHash:        b.EffectiveBannerHash,
			EffectiveBannerURL:         b.EffectiveBannerURL,
			EffectiveBannerWidth:       b.EffectiveBannerWidth,
			EffectiveBannerHeight:      b.EffectiveBannerHeight,
			EffectiveBannerThumbhash:   b.EffectiveBannerThumbhash,
			EffectivePortraitHash:      b.EffectivePortraitHash,
			EffectivePortraitURL:       b.EffectivePortraitURL,
			EffectivePortraitWidth:     b.EffectivePortraitWidth,
			EffectivePortraitHeight:    b.EffectivePortraitHeight,
			EffectivePortraitThumbhash: b.EffectivePortraitThumbhash,
			Platform:                   emptyStrSliceIfNil(platformMap[id]),
			Language:                   emptyStrSliceIfNil(languageMap[id]),
			Company:                    b.Company,
		})
	}
	return cards, nil
}
