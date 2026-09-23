package service

import (
	"context"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/storelink"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"
)

type GalgameService struct {
	galgameRepo      *repository.GalgameRepository
	listRepo         *repository.GalgameListRepository
	resourceMetaRepo *repository.GalgameResourceMetaRepository
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
	listRepo *repository.GalgameListRepository,
	resourceMetaRepo *repository.GalgameResourceMetaRepository,
	contributorRepo *repository.GalgameContributorRepository,
	stateRepo *userRepo.StateRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	catalog *catalogclient.Client,
	storeLinks *storelink.Resolver,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		storeLinks:       storeLinks,
		contributorRepo:  contributorRepo,
		stateRepo:        stateRepo,
		galgameClient:    galgameClient,
		userClient:       userClient,
		catalog:          catalog,
	}
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
