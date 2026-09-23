package service

import (
	"context"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
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
