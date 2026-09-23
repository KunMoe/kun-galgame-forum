package entityapiv1

import (
	"cmp"
	"context"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

const (
	seriesIndexTTL     = 10 * time.Minute
	seriesIndexPageCap = 100
	seriesCardFanout   = 8
	seriesMemberCap    = 100
	seriesSampleSize   = 5
)

type seriesRow struct {
	id      int
	name    repr.CatalogName
	count   int
	hasNSFW *bool
}

func (s *Service) seriesIndex(ctx context.Context) ([]seriesRow, error) {
	return s.series.get(ctx, seriesIndexTTL, s.buildSeriesIndex)
}

func (s *Service) buildSeriesIndex(ctx context.Context) ([]seriesRow, error) {
	rows := []seriesRow{}
	cursor := ""
	for page := 0; page < seriesIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.catalog.CatalogTaxonomyList(ctx, "series", q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			e := &res.Items[i]
			rows = append(rows, seriesRow{
				id:      int(e.ID),
				name:    workrepr.Name(cmp.Or(e.DisplayName, e.Name), e.Latin, client.LocalizedValues(e.Localized)),
				count:   max(e.WorkCount, 0),
				hasNSFW: e.HasNSFW,
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].count != rows[j].count {
			return rows[i].count > rows[j].count
		}
		return rows[i].id < rows[j].id
	})
	return rows, nil
}

func (s *Service) seriesCardIndex(ctx context.Context) ([]SeriesSummary, error) {
	return s.seriesCards.get(ctx, seriesIndexTTL, s.buildSeriesCards)
}

func (s *Service) buildSeriesCards(ctx context.Context) ([]SeriesSummary, error) {
	all, err := s.seriesIndex(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]seriesRow, 0, len(all))
	for _, r := range all {
		if r.count > 0 {
			candidates = append(candidates, r)
		}
	}
	cards := s.cardsFor(ctx, candidates)
	out := make([]SeriesSummary, 0, len(cards))
	for _, c := range cards {
		if c.ListedWorkCount > 0 {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ListedWorkCount != out[j].ListedWorkCount {
			return out[i].ListedWorkCount > out[j].ListedWorkCount
		}
		return idLess(out[i].ID, out[j].ID)
	})
	return out, nil
}

// cardsFor builds each series' listed count and samples from its first
// seriesMemberCap works. A series whose member read fails keeps a card with no
// samples rather than failing the whole list.
func (s *Service) cardsFor(ctx context.Context, rows []seriesRow) []SeriesSummary {
	out := make([]SeriesSummary, len(rows))
	sem := make(chan struct{}, seriesCardFanout)
	var wg sync.WaitGroup
	for i, r := range rows {
		wg.Add(1)
		go func(i int, r seriesRow) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = s.card(ctx, r)
		}(i, r)
	}
	wg.Wait()
	return out
}

func (s *Service) card(ctx context.Context, r seriesRow) SeriesSummary {
	card := SeriesSummary{
		Object:           "series",
		ID:               repr.ID(r.id),
		CatalogName:      r.name,
		HasNSFWWorks:     r.hasNSFW,
		CatalogWorkCount: r.count,
		SampleWorks:      []SeriesSampleWork{},
	}
	res, appErr := s.catalog.CatalogWorksSearch(ctx, client.OpenPopulation(url.Values{
		"series_id": {strconv.Itoa(r.id)},
		"page":      {"1"},
		"limit":     {strconv.Itoa(seriesMemberCap)},
		"include":   {workrepr.RowInclude},
		"sort":      {"released_asc"},
	}))
	if appErr != nil {
		return card
	}
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	ids := make([]int, 0, len(res.Items))
	for i := range res.Items {
		if client.CatalogItemRenderable(&res.Items[i]) && res.Items[i].ID > 0 {
			rows = append(rows, res.Items[i])
			ids = append(ids, int(res.Items[i].ID))
		}
	}
	if len(ids) == 0 {
		return card
	}
	listed, _ := s.lists.ListIDs(model.GalgameListFilter{RestrictIDs: ids, Page: 1, Limit: len(ids), SortOrder: "desc"})
	isListed := make(map[int]bool, len(listed))
	for _, id := range listed {
		isListed[id] = true
	}
	for i := range rows {
		if !isListed[int(rows[i].ID)] {
			continue
		}
		card.ListedWorkCount++
		if len(card.SampleWorks) < seriesSampleSize {
			card.SampleWorks = append(card.SampleWorks, SeriesSampleWork{
				WorkRef: workrepr.Ref(ctx, &rows[i], s.cdn),
				Banner:  workrepr.Banner(&rows[i], s.cdn),
			})
		}
	}
	return card
}

// safeForSFW keeps the legacy rule: a series catalog cannot vouch for is
// treated as adult.
func safeForSFW(hasNSFW *bool) bool {
	return hasNSFW != nil && !*hasNSFW
}

type listSeriesInput struct {
	Q           string `query:"q" maxLength:"100" doc:"Case-insensitive substring of any of the series' names. Set, the collection is every catalog series that matches, listed works or not, most works first. Free text; never use it as a decision input."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"12" doc:"Page size. 1–100, default 12. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, series with adult works are included. Default false: only series catalog says have none."`
}

type listSeriesOutput struct {
	Body repr.PageList[SeriesSummary]
}

func (s *Service) listSeries(ctx context.Context, in *listSeriesInput) (*listSeriesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := checkDepth(in.Page, in.Limit, false); prob != nil {
		return nil, prob
	}
	q := trimQuery(in.Q)
	if q == "" {
		cards, err := s.seriesCardIndex(ctx)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		rows := make([]SeriesSummary, 0, len(cards))
		for _, c := range cards {
			if in.IncludeNSFW || safeForSFW(c.HasNSFWWorks) {
				rows = append(rows, c)
			}
		}
		return &listSeriesOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
	}
	all, err := s.seriesIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	matched := make([]seriesRow, 0)
	for _, r := range all {
		if (in.IncludeNSFW || safeForSFW(r.hasNSFW)) && nameMatches(q, r.name) {
			matched = append(matched, r)
		}
	}
	page := pageOf(matched, in.Page, in.Limit)
	cards := s.cardsFor(ctx, page)
	body := pageList(matched, in.Page, in.Limit)
	return &listSeriesOutput{Body: repr.NewPageList(cards, body.Total, body.TotalRelation)}, nil
}

type seriesPathInput struct {
	SeriesID string `path:"series_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Series id."`
}

type getSeriesOutput struct {
	Body Series
}

func (s *Service) seriesHead(ctx context.Context, rawID string) (*client.CatalogSeriesDetail, *problem.Problem) {
	if _, ok := pathID(rawID); !ok {
		return nil, notFound()
	}
	rec, found, appErr := s.catalog.CatalogSeries(ctx, rawID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if !found {
		return nil, notFound()
	}
	return rec, nil
}

func (s *Service) getSeries(ctx context.Context, in *seriesPathInput) (*getSeriesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	rec, prob := s.seriesHead(ctx, in.SeriesID)
	if prob != nil {
		return nil, prob
	}
	card := s.card(ctx, seriesRow{
		id:      int(rec.ID),
		name:    workrepr.Name(cmp.Or(rec.DisplayName, rec.Name), "", client.LocalizedValues(rec.Localized)),
		count:   max(rec.WorkCount, 0),
		hasNSFW: rec.HasNSFW,
	})
	return &getSeriesOutput{Body: Series{
		Object:           "series",
		ID:               card.ID,
		CatalogName:      card.CatalogName,
		HasNSFWWorks:     card.HasNSFWWorks,
		CatalogWorkCount: card.CatalogWorkCount,
		ListedWorkCount:  card.ListedWorkCount,
		SampleWorks:      card.SampleWorks,
		Intros:           workrepr.Intros(rec.Intros),
	}}, nil
}

type listSeriesWorksInput struct {
	SeriesID string `path:"series_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Series id."`
	WorksQuery
}

func (s *Service) listSeriesWorks(ctx context.Context, in *listSeriesWorksInput) (*workPageOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if _, prob := s.seriesHead(ctx, in.SeriesID); prob != nil {
		return nil, prob
	}
	page, prob := s.worksPage(ctx, s.walkMembers("series_id", in.SeriesID), in.WorksQuery)
	if prob != nil {
		return nil, prob
	}
	return &workPageOutput{Body: page}, nil
}
