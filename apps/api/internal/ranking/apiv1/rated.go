package apiv1

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/utils"
)

type RatedWorks interface {
	CatalogRankings(ctx context.Context, q url.Values) (*client.CatalogRankingPage, *legacyErrors.AppError)
}

type WorkHydrator interface {
	FromRows(ctx context.Context, rows []client.CatalogWorkListItem) ([]workrepr.WorkSummary, *problem.Problem)
}

func (s *Service) WithRatedWorks(rated RatedWorks, hydrator WorkHydrator) *Service {
	s.rated = rated
	s.hydrator = hydrator
	return s
}

type ratedSort struct {
	metric, order   string
	consensus, site bool
}

var ratedSorts = map[string]ratedSort{
	"lower_desc":      {"lower", "desc", true, true},
	"mean_desc":       {"mean", "desc", true, true},
	"upper_asc":       {"upper", "asc", true, true},
	"votes_desc":      {"votes", "desc", true, true},
	"divergence_desc": {"divergence", "desc", true, false},
	"native_desc":     {"native", "desc", false, true},
	"median_desc":     {"median", "desc", false, true},
	"spread_desc":     {"spread", "desc", false, true},
}

func inconsistent(param, with, detail string) *problem.Problem {
	return problem.New(problem.CodeInvalidParameter, detail,
		problem.AtParameter(param, problem.ReasonInconsistentWith, with, nil))
}

func ratedQuery(in *ratedWorkRankingInput) (url.Values, *problem.Problem) {
	sort, ok := ratedSorts[in.Sort]
	if !ok {
		allowed := make([]string, 0, len(ratedSorts))
		for token := range ratedSorts {
			allowed = append(allowed, token)
		}
		return nil, problem.New(problem.CodeUnknownSort, "The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "not in this collection's closed vocabulary",
				&problem.FieldParams{Allowed: &allowed}))
	}
	bySite := in.Site != ""
	if bySite && !sort.site {
		return nil, inconsistent("sort", "site", in.Sort+" ranks the consensus and needs site absent.")
	}
	if !bySite && !sort.consensus {
		return nil, inconsistent("sort", "site", in.Sort+" ranks one site and needs site.")
	}
	if bySite && in.MinSiteCount > 0 {
		return nil, inconsistent("min_site_count", "site", "min_site_count shapes the consensus and needs site absent.")
	}
	if in.MinVoteCount > 0 && in.MaxVoteCount > 0 && in.MinVoteCount > in.MaxVoteCount {
		return nil, inconsistent("min_vote_count", "max_vote_count", "min_vote_count is above max_vote_count.")
	}
	if in.MinPlaytimeMinutes > 0 && in.MaxPlaytimeMinutes > 0 && in.MinPlaytimeMinutes > in.MaxPlaytimeMinutes {
		return nil, inconsistent("min_playtime_minutes", "max_playtime_minutes", "min_playtime_minutes is above max_playtime_minutes.")
	}
	from, err := utils.ParseReleaseLowerBound(in.ReleasedFrom)
	if err != nil {
		return nil, problem.New(problem.CodeInvalidParameter, "released_from is not a year or year-month.",
			problem.AtParameter("released_from", problem.ReasonInvalidFormat, "must be YYYY or YYYY-MM", nil))
	}
	to, err := utils.ParseReleaseUpperBound(in.ReleasedTo)
	if err != nil {
		return nil, problem.New(problem.CodeInvalidParameter, "released_to is not a year or year-month.",
			problem.AtParameter("released_to", problem.ReasonInvalidFormat, "must be YYYY or YYYY-MM", nil))
	}
	if from != "" && to != "" && from > to {
		return nil, inconsistent("released_from", "released_to", "released_from is after released_to.")
	}

	q := url.Values{
		"page":    {strconv.Itoa(in.Page)},
		"limit":   {strconv.Itoa(in.Limit)},
		"include": {workrepr.RowInclude},
		"metric":  {sort.metric},
		"order":   {sort.order},
	}
	if bySite {
		q.Set("by", string(in.Site))
	}
	setPositive := func(key string, n int) {
		if n > 0 {
			q.Set(key, strconv.Itoa(n))
		}
	}
	setPositive("min_votes", in.MinVoteCount)
	setPositive("max_votes", in.MaxVoteCount)
	setPositive("min_sources", in.MinSiteCount)
	setPositive("playtime_min", in.MinPlaytimeMinutes)
	setPositive("playtime_max", in.MaxPlaytimeMinutes)
	if from != "" {
		q.Set("released_after", from)
	}
	if to != "" {
		q.Set("released_before", to)
	}
	if in.Circulation != "" {
		q.Set("circulation", in.Circulation)
	}
	if len(in.Platforms) > 0 {
		platforms := make([]string, len(in.Platforms))
		for i, p := range in.Platforms {
			platforms[i] = string(p)
		}
		q.Set("platform", strings.Join(platforms, ","))
	}
	if in.IncludeAllGames {
		q.Set("scope", "all")
	}
	q.Set("facets", client.EntityFacets)
	in.EntityFilterInput.Apply(q)
	client.ApplyWorksGate(q, !in.IncludeNSFW)
	client.ApplyOriginalLanguageGate(q, !in.IncludeAllOriginalLanguages)
	in.Exclusion().Apply(q)
	return q, nil
}

func (s *Service) listRatedWorkRanking(ctx context.Context, in *ratedWorkRankingInput) (*ratedWorkRankingOutput, error) {
	if s == nil || s.rated == nil || s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	pg := collect.PageNumber{Page: in.Page, Limit: in.Limit}
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	q, prob := ratedQuery(in)
	if prob != nil {
		return nil, prob
	}
	res, appErr := s.rated.CatalogRankings(ctx, q)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}

	entries := make([]client.CatalogRankingEntry, 0, len(res.Items))
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	for _, e := range res.Items {
		if !client.CatalogItemRenderable(&e.Work) {
			slog.Warn("rated-work-ranking: catalog row not renderable, dropped", "work_id", e.Work.ID, "rank", e.Rank)
			continue
		}
		entries = append(entries, e)
		rows = append(rows, e.Work)
	}
	works, prob := s.hydrator.FromRows(ctx, rows)
	if prob != nil {
		return nil, prob
	}

	scales := make(map[string]client.CatalogRatingScale, len(res.Model.Sources))
	for _, sc := range res.Model.Sources {
		scales[sc.Source] = sc
	}
	items := make([]RatedWorkRankingEntry, len(entries))
	for i, e := range entries {
		items[i] = RatedWorkRankingEntry{
			Object:      "rated_work_ranking_entry",
			Rank:        e.Rank,
			MetricValue: e.Value,
			WorkSummary: works[i],
			Consensus:   ratingConsensus(e.Consensus),
			SiteRatings: siteRatings(e.PerSource, scales),
		}
	}

	n, rel := collect.ClampTotal(int(res.Total))
	body := RatedWorkRanking{
		Object: "list", Items: items, Total: n, TotalRelation: rel,
		Facets: workrepr.NewWorkFacets(res.Facets.Tags(), res.Facets.Companies()),
	}
	if t, err := time.Parse(time.RFC3339Nano, res.Model.FittedAt); err == nil {
		at := repr.Timestamp(t)
		body.ModelFittedAt = &at
	}
	return &ratedWorkRankingOutput{Body: body}, nil
}

func ratingConsensus(c *client.CatalogRankingConsensus) *RatingConsensus {
	if c == nil {
		return nil
	}
	return &RatingConsensus{
		Deviation:  c.Score,
		Lower:      c.Lower,
		Upper:      c.Upper,
		Percentile: c.Percentile,
		SiteCount:  c.SourceCount,
		VoteCount:  c.VoteCount,
		Divergence: c.Divergence,
	}
}

func siteRatings(in []client.CatalogRankingSource, scales map[string]client.CatalogRatingScale) []SiteRating {
	out := make([]SiteRating, len(in))
	for i, r := range in {
		sc := scales[r.Source]
		out[i] = SiteRating{
			Site:           r.Source,
			PublishedScore: r.Score,
			ScaleMin:       sc.ScaleMin,
			ScaleMax:       sc.ScaleMax,
			VoteCount:      r.VoteCount,
			Estimate:       r.Mean,
			Lower:          r.Lower,
			Upper:          r.Upper,
			Median:         r.Median,
			Spread:         r.Spread,
			Percentile:     r.Percentile,
		}
	}
	return out
}
