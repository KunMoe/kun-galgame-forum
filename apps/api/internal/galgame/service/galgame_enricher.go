package service

import (
	"context"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"
)

type GalgameEnricher struct {
	galgameRepo *repository.GalgameRepository
	metaRepo    *repository.GalgameResourceMetaRepository
	listRepo    *repository.GalgameListRepository
	userClient  *userclient.Client
}

func NewGalgameEnricher(
	galgameRepo *repository.GalgameRepository,
	metaRepo *repository.GalgameResourceMetaRepository,
	listRepo *repository.GalgameListRepository,
	userClient *userclient.Client,
) *GalgameEnricher {
	return &GalgameEnricher{
		galgameRepo: galgameRepo, metaRepo: metaRepo,
		listRepo: listRepo, userClient: userClient,
	}
}

func (e *GalgameEnricher) ToCards(ctx context.Context, items []dto.NextMoeGalgameItem) []dto.GalgameCard {
	if len(items) == 0 {
		return []dto.GalgameCard{}
	}

	workIDs := make([]int, len(items))
	for i, g := range items {
		workIDs[i] = g.ID
	}

	localMap := e.galgameRepo.FindLocalBatch(workIDs)
	userMap := e.userClient.Hydrate(ctx, frozenCreatorIDs(workIDs, localMap))
	platformMap, languageMap := groupResourceMeta(e.metaRepo.FindResourceMetaBatch(workIDs))
	ratingMap := e.listRepo.BayesianRatings(workIDs)

	cards := make([]dto.GalgameCard, len(items))
	for i, g := range items {
		cards[i] = dto.GalgameCard{
			ID:                       g.ID,
			Name:                     g.Name,
			NameOriginal:             g.NameOriginal,
			User:                     frozenCreatorBrief(localMap[g.ID], userMap),
			ContentLimit:             g.ContentLimit,
			View:                     localMap[g.ID].View,
			LikeCount:                localMap[g.ID].LikeCount,
			Rating:                   ratingMap[g.ID].Score,
			RatingCount:              ratingMap[g.ID].Count,
			ResourceUpdateTime:       utils.RFC3339OrEmpty(localMap[g.ID].ResourceUpdateTime),
			ReleaseDate:              g.ReleaseDate,
			ReleaseDateTBA:           g.ReleaseDateTBA,
			ReleasePrecision:         g.ReleasePrecision,
			Status:                   g.Status,
			EffectiveBannerHash:      g.EffectiveBannerHash,
			EffectiveBannerURL:       g.EffectiveBannerURL,
			EffectiveBannerWidth:     g.EffectiveBannerWidth,
			EffectiveBannerHeight:    g.EffectiveBannerHeight,
			EffectiveBannerThumbhash: g.EffectiveBannerThumbhash,

			EffectivePortraitHash:      g.EffectivePortraitHash,
			EffectivePortraitURL:       g.EffectivePortraitURL,
			EffectivePortraitWidth:     g.EffectivePortraitWidth,
			EffectivePortraitHeight:    g.EffectivePortraitHeight,
			EffectivePortraitThumbhash: g.EffectivePortraitThumbhash,

			Company:  g.Company,
			Platform: emptyStrSliceIfNil(platformMap[g.ID]),
			Language: emptyStrSliceIfNil(languageMap[g.ID]),
			// Not "a local row exists" — since 068 the row is only an interaction
			// container, minted by the first view of any catalog work, so that
			// test called 124 unpublished entries "on the forum" and the card
			// then drew a byline with a blank author and a blank time.
			IsOnForum: localMap[g.ID].Published,
		}
	}
	return cards
}
