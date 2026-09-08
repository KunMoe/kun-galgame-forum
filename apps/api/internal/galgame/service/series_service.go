package service

import (
	"context"
	"net/url"
	"strconv"
	"sync"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type SeriesService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
	galgameSvc    *GalgameService
	index         staleCache[indexedSeries]
}

func NewSeriesService(galgameClient *client.GalgameClient, enricher *GalgameEnricher, galgameSvc *GalgameService) *SeriesService {
	return &SeriesService{galgameClient: galgameClient, enricher: enricher, galgameSvc: galgameSvc}
}

const seriesIndexPageCap = 100

func (s *SeriesService) GetList(ctx context.Context) ([]dto.SeriesListItem, *errors.AppError) {
	rows, appErr := s.walkIndex(ctx)
	if appErr != nil {
		return nil, appErr
	}
	items := make([]dto.SeriesListItem, len(rows))
	for i, r := range rows {
		items[i] = r.SeriesListItem
	}
	return items, nil
}

type seriesIndexRow struct {
	dto.SeriesListItem
	hasNSFW *bool
}

func (s *SeriesService) walkIndex(ctx context.Context) ([]seriesIndexRow, *errors.AppError) {
	items := []seriesIndexRow{}
	cursor := ""
	for page := 0; page < seriesIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.galgameClient.CatalogTaxonomyList(ctx, "series", q)
		if appErr != nil {
			return nil, appErr
		}
		for _, e := range res.Items {
			items = append(items, seriesIndexRow{
				SeriesListItem: dto.SeriesListItem{
					ID:           int(e.ID),
					Name:         e.Label(ctx),
					GalgameCount: e.WorkCount,
				},
				hasNSFW: e.HasNSFW,
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	return items, nil
}

const seriesSampleSize = 5

func (s *SeriesService) GetCards(
	ctx context.Context,
	ids []int,
	page, limit int,
	isSFW bool,
) (*dto.SeriesCardPage, *errors.AppError) {
	if len(ids) > 0 {
		return s.cardsByID(ctx, ids, isSFW), nil
	}

	index, appErr := s.indexRows(ctx)
	if appErr != nil {
		return nil, appErr
	}

	rows := make([]dto.SeriesCard, 0, len(index))
	for _, it := range index {
		if isSFW && !sfwSafeSeries(it.hasNSFW) {
			continue
		}
		rows = append(rows, it.card)
	}
	total := int64(len(rows))

	if start := (page - 1) * limit; start >= len(rows) {
		rows = nil
	} else {
		rows = rows[start:min(start+limit, len(rows))]
	}
	return &dto.SeriesCardPage{Series: rows, Total: total}, nil
}

func sfwSafeSeries(hasNSFW *bool) bool {
	return hasNSFW != nil && !*hasNSFW
}

func seriesNSFW(hasNSFW *bool, sampled bool) bool {
	if hasNSFW != nil {
		return *hasNSFW
	}
	return sampled
}

func (s *SeriesService) cardsByID(ctx context.Context, ids []int, isSFW bool) *dto.SeriesCardPage {
	cards := make([]*dto.SeriesCard, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i, id int) {
			defer wg.Done()
			rec, found, appErr := s.galgameClient.CatalogSeries(ctx, strconv.Itoa(id))
			if appErr != nil || !found {
				return
			}
			if isSFW && !sfwSafeSeries(rec.HasNSFW) {
				return
			}
			built := s.buildCard(ctx, seriesIndexRow{
				SeriesListItem: dto.SeriesListItem{
					ID: int(rec.ID), Name: rec.Label(ctx), GalgameCount: rec.WorkCount,
				},
				hasNSFW: rec.HasNSFW,
			})
			if built.card.GalgameCount == 0 {
				return
			}
			cards[i] = &built.card
		}(i, id)
	}
	wg.Wait()

	out := make([]dto.SeriesCard, 0, len(cards))
	for _, c := range cards {
		if c != nil {
			out = append(out, *c)
		}
	}
	return &dto.SeriesCardPage{Series: out, Total: int64(len(out))}
}

func (s *SeriesService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.SeriesDetail, *errors.AppError) {
	rec, found, appErr := s.galgameClient.CatalogSeries(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该系列")
	}

	filter := buildEntityFilter(rawQuery)
	memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,
		entityMemberQuery("series_id", id, filter), isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	filter.RestrictIDs = memberIDs
	page, appErr := s.galgameSvc.hydrateListCards(ctx, filter, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.SeriesDetail{
		ID:           int(rec.ID),
		Name:         rec.Label(ctx),
		Description:  seriesIntro(rec),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

func seriesIntro(rec *client.CatalogSeriesDetail) string {
	for _, lang := range []string{"zh-Hans", "zh-Hant", "ja", "en"} {
		for _, in := range rec.Intros {
			if in.Lang == lang && in.Intro != "" {
				return in.Intro
			}
		}
	}
	for _, in := range rec.Intros {
		if in.Intro != "" {
			return in.Intro
		}
	}
	return ""
}
