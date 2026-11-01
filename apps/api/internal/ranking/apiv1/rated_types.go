package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

var ratingSites = []string{"vndb", "bangumi", "dlsite", "erogamescape", "nextmoe", "howlongtobeat"}

type RatingSite string

func (RatingSite) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum(ratingSites...)
	s.Description = "Rank by this rating site alone. Omitted ranks by the consensus of every site."
	return s
}

type ratedWorkRankingInput struct {
	Site  RatingSite `query:"site"`
	Sort  string     `query:"sort" enum:"lower_desc,mean_desc,upper_asc,native_desc,median_desc,divergence_desc,spread_desc,votes_desc" default:"lower_desc" maxLength:"15" doc:"What the list ranks by. lower: the bottom of the 90% interval, so only works that are good with confidence lead. upper_asc: the top of the interval, worst first, so only works that are bad with confidence lead. mean: the best estimate. native: the site's own published score. median: the shrunk median vote. divergence: how far apart the sites place the work. spread: how split one site's votes are. votes: how many votes. native, median and spread need site; divergence needs it absent."`
	Page  int        `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit int        `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`

	ReleasedFrom       string                      `query:"released_from" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or after this year (YYYY) or month (YYYY-MM)."`
	ReleasedTo         string                      `query:"released_to" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or before this year (YYYY) or month (YYYY-MM)."`
	Circulation        string                      `query:"circulation" enum:"commercial,doujin" maxLength:"10" doc:"commercial: a game brand made or sold the work. doujin: only a doujin circle did. Omitted means both, and works with neither credit."`
	Platforms          []workrepr.ResourcePlatform `query:"platforms" maxItems:"48" doc:"Only works released on any of these platforms, in the resource platform vocabulary. Comma-separated. Omitted means no filter."`
	MinPlaytimeMinutes int                         `query:"min_playtime_minutes" minimum:"0" maximum:"100000" doc:"Only works whose playtime, the median of the sites' estimates, is at least this many minutes. 0 or omitted means no floor."`
	MaxPlaytimeMinutes int                         `query:"max_playtime_minutes" minimum:"0" maximum:"100000" doc:"Only works whose playtime is at most this many minutes. 0 or omitted means no ceiling."`
	MinVoteCount       int                         `query:"min_vote_count" minimum:"0" maximum:"10000000" doc:"Only works with at least this many votes: on site, or over every site without it. 0 or omitted means no floor."`
	MaxVoteCount       int                         `query:"max_vote_count" minimum:"0" maximum:"10000000" doc:"Only works with at most this many votes, counted as min_vote_count counts them. 0 or omitted means no ceiling."`
	MinSiteCount       int                         `query:"min_site_count" minimum:"0" maximum:"6" doc:"Only works at least this many sites rated. Needs site absent. 0 or omitted means no floor."`
	IncludeAllGames    bool                        `query:"include_all_games" default:"false" doc:"When true, games that are not visual novels, such as console action and role-playing games catalog also holds, are ranked too. Default false."`
	IncludeNSFW        bool                        `query:"include_nsfw" default:"false" doc:"When true, works this forum displays as adult are ranked too. Default false."`

	IncludeAllOriginalLanguages bool `query:"include_all_original_languages" default:"false" doc:"When true, works whose original language is neither Japanese nor Chinese are ranked too. Default false."`

	workrepr.EntityFilterInput
	workrepr.TagExclusionInput
}

type RatedWorkRanking struct {
	Object        string                  `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items         []RatedWorkRankingEntry `json:"items" doc:"Members of this page. Empty array, never null."`
	Total         int                     `json:"total" minimum:"0" doc:"Works the ranking holds under these filters. Counted up to the depth limit when total_relation is gte."`
	TotalRelation string                  `json:"total_relation" enum:"eq,gte" maxLength:"3" doc:"eq when total is exact, gte when it stopped at the depth limit and there are at least that many."`
	Facets        workrepr.WorkFacets     `json:"facets" doc:"The tags and companies most common among the ranked works under these filters, not only this page."`
	ModelFittedAt *repr.DateTime          `json:"model_fitted_at" doc:"When catalog last fitted the rating model behind every derived number here. It is refitted nightly. null when catalog did not say."`
}

type ratedWorkRankingOutput struct {
	Body RatedWorkRanking
}

type RatedWorkRankingEntry struct {
	Object      string               `json:"object" enum:"rated_work_ranking_entry" maxLength:"24" doc:"Type discriminant. Always rated_work_ranking_entry."`
	Rank        int                  `json:"rank" minimum:"1" doc:"1-based place in the whole ranking, not in the page. Equal values keep distinct places. A work this forum cannot show is left out and keeps its place, so a page can skip a number."`
	MetricValue float64              `json:"metric_value" minimum:"0" doc:"The value the list is sorted by. A deviation score for lower, mean and upper without site; the site's own scale for lower, mean, upper, native, median and spread with it; percentile points for divergence; a vote count for votes."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"The ranked work."`
	Consensus   *RatingConsensus     `json:"consensus" doc:"The work's standing across every rating site. null when no site that feeds the consensus rated it."`
	SiteRatings []SiteRating         `json:"site_ratings" doc:"Every site that rated the work, in a fixed site order. Empty array, never null."`
}

type RatingConsensus struct {
	Deviation  float64  `json:"deviation" minimum:"0" maximum:"100" doc:"The consensus as a deviation score: 50 is the typical work two or more sites rated, and every 10 points is one standard deviation."`
	Lower      float64  `json:"lower" minimum:"0" maximum:"100" doc:"Bottom of the 90% interval of deviation. Few votes or a single site widen the interval."`
	Upper      float64  `json:"upper" minimum:"0" maximum:"100" doc:"Top of the 90% interval of deviation."`
	Percentile float64  `json:"percentile" minimum:"0" maximum:"100" doc:"Share of the works with a consensus that score lower, in percent."`
	SiteCount  int      `json:"site_count" minimum:"0" maximum:"6" doc:"Sites that contributed."`
	VoteCount  int      `json:"vote_count" minimum:"0" doc:"Votes across the contributing sites."`
	Divergence *float64 `json:"divergence" minimum:"0" maximum:"100" doc:"How far apart the sites place the work, in percentile points between the highest and the lowest. null when fewer than two sites have enough votes to say."`
}

type SiteRating struct {
	Site           string   `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The rating site, such as vndb or erogamescape. An open vocabulary."`
	PublishedScore float64  `json:"published_score" minimum:"0" doc:"The site's own published score, unchanged: vndb's Bayesian rating, bangumi's and dlsite's mean, erogamescape's median, nextmoe's smoothed mean, howlongtobeat's review score."`
	ScaleMin       float64  `json:"scale_min" minimum:"0" doc:"Lowest score the site's scale allows."`
	ScaleMax       float64  `json:"scale_max" minimum:"0" doc:"Highest score the site's scale allows."`
	VoteCount      int      `json:"vote_count" minimum:"0" doc:"Votes behind published_score."`
	Estimate       float64  `json:"estimate" minimum:"0" doc:"Catalog's estimate of the work's true mean on the site's scale: its own mean pulled toward what works with as many votes usually score. Few votes pull hard."`
	Lower          float64  `json:"lower" minimum:"0" doc:"Bottom of the 90% interval of estimate."`
	Upper          float64  `json:"upper" minimum:"0" doc:"Top of the 90% interval of estimate."`
	Median         *float64 `json:"median" minimum:"0" doc:"The median vote with the same pull applied. null when the site publishes no histogram."`
	Spread         *float64 `json:"spread" minimum:"0" doc:"Standard deviation of the votes on the site's scale; the larger, the more split. null below two votes or when the site publishes neither."`
	Percentile     float64  `json:"percentile" minimum:"0" maximum:"100" doc:"Share of the works this site rated whose estimate is lower, in percent."`
}
