package service

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/errors"
)

const (
	seriesIndexTTL    = 10 * time.Minute
	seriesIndexFanout = 8
	seriesMemberCap   = 100
)

type indexedSeries struct {
	card    dto.SeriesCard
	hasNSFW *bool
}

func (s *SeriesService) indexRows(ctx context.Context) ([]indexedSeries, *errors.AppError) {
	return s.index.get(ctx, seriesIndexTTL, s.buildIndex)
}

func (s *SeriesService) buildIndex(ctx context.Context) ([]indexedSeries, *errors.AppError) {
	lane, appErr := s.walkIndex(ctx)
	if appErr != nil {
		return nil, appErr
	}

	candidates := make([]seriesIndexRow, 0, len(lane))
	for _, row := range lane {
		if row.GalgameCount > 0 {
			candidates = append(candidates, row)
		}
	}

	rows := make([]indexedSeries, len(candidates))
	sem := make(chan struct{}, seriesIndexFanout)
	var wg sync.WaitGroup
	for i, row := range candidates {
		wg.Add(1)
		go func(i int, row seriesIndexRow) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rows[i] = s.buildCard(ctx, row)
		}(i, row)
	}
	wg.Wait()

	out := make([]indexedSeries, 0, len(rows))
	for _, r := range rows {
		if r.card.GalgameCount > 0 {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].card.GalgameCount != out[j].card.GalgameCount {
			return out[i].card.GalgameCount > out[j].card.GalgameCount
		}
		return out[i].card.Name < out[j].card.Name
	})
	return out, nil
}

func (s *SeriesService) buildCard(ctx context.Context, row seriesIndexRow) indexedSeries {
	card := dto.SeriesCard{
		ID:                  row.ID,
		Name:                row.Name,
		IsNSFW:              seriesNSFW(row.hasNSFW, false),
		CatalogGalgameCount: row.GalgameCount,
		SampleGalgame:       []dto.SeriesSample{},
	}

	members, appErr := s.galgameClient.CatalogWorksSearch(ctx, client.OpenPopulation(url.Values{
		"series_id": {strconv.Itoa(row.ID)},
		"page":      {"1"},
		"limit":     {strconv.Itoa(seriesMemberCap)},
		"include":   {CatalogCardInclude},
		"sort":      {"released_asc"},
	}))
	if appErr != nil {
		return indexedSeries{card: card, hasNSFW: row.hasNSFW}
	}

	items := make([]dto.NextMoeGalgameItem, 0, len(members.Items))
	gids := make([]int, 0, len(members.Items))
	for i := range members.Items {
		if !client.CatalogItemRenderable(&members.Items[i]) {
			continue
		}
		it := client.CatalogItemToNextMoeItem(ctx, &members.Items[i])
		if it.ID <= 0 {
			continue
		}
		items = append(items, it)
		gids = append(gids, it.ID)
	}

	listable := s.listableGIDs(gids)
	sampleNSFW := false
	for _, it := range items {
		if !listable[it.ID] {
			continue
		}
		card.GalgameCount++
		if len(card.SampleGalgame) >= seriesSampleSize {
			continue
		}
		if it.ContentLimit == "nsfw" {
			sampleNSFW = true
		}
		card.SampleGalgame = append(card.SampleGalgame, dto.SeriesSample{
			Name:                     it.Name,
			EffectiveBannerHash:      it.EffectiveBannerHash,
			EffectiveBannerURL:       it.EffectiveBannerURL,
			EffectiveBannerThumbhash: it.EffectiveBannerThumbhash,
		})
	}
	card.IsNSFW = seriesNSFW(row.hasNSFW, sampleNSFW)
	return indexedSeries{card: card, hasNSFW: row.hasNSFW}
}

func (s *SeriesService) listableGIDs(gids []int) map[int]bool {
	out := make(map[int]bool, len(gids))
	if len(gids) == 0 {
		return out
	}
	if s.galgameSvc == nil {
		for _, gid := range gids {
			out[gid] = true
		}
		return out
	}
	ids, _ := s.galgameSvc.listRepo.ListIDs(model.GalgameListFilter{
		RestrictIDs: gids,
		Page:        1,
		Limit:       len(gids),
		SortOrder:   "desc",
	})
	for _, id := range ids {
		out[id] = true
	}
	return out
}
