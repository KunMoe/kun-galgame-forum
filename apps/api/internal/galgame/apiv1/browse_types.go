package apiv1

import (
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

const defaultBrowseLimit = 24

type MonthNumber int

func (MonthNumber) Schema(huma.Registry) *huma.Schema {
	min, max := 1.0, 12.0
	return &huma.Schema{
		Type:        huma.TypeInteger,
		Minimum:     &min,
		Maximum:     &max,
		Format:      "int64",
		Description: "A calendar month 1–12.",
	}
}

type LibraryWorkSortToken string

func (LibraryWorkSortToken) Schema(huma.Registry) *huma.Schema {
	enum := []any{"popularity_desc", "released_desc", "released_asc", "updated_desc", "relevance_desc"}
	n := 15
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &n,
		Default:     "popularity_desc",
		Description: "Order. popularity: catalog popularity. released: release date. updated: last catalog edit. relevance: the search index's ranking.",
	}
}

var libraryWorkSorts = map[string]string{
	"popularity_desc": "popularity",
	"released_desc":   "released_desc",
	"released_asc":    "released_asc",
	"updated_desc":    "updated",
	"relevance_desc":  "relevance",
}

type listWorksInput struct {
	Page                  int                         `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit                 int                         `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	Sort                  entityapiv1.WorkSortToken   `query:"sort" default:"resource_updated_desc"`
	ResourceType          workrepr.ResourceType       `query:"resource_type" doc:"Only works with at least one forum resource of this type. Omitted means no filter."`
	ResourcePlatforms     []workrepr.ResourcePlatform `query:"resource_platforms" doc:"Only works with at least one forum resource for any of these platforms. Comma-separated. Omitted means no filter."`
	ResourceLanguages     []workrepr.ResourceLanguage `query:"resource_languages" doc:"Only works with at least one forum resource in any of these languages. Comma-separated. Omitted means no filter."`
	GameType              entityapiv1.GameTypeFilter  `query:"game_type"`
	ResourceProviders     []workrepr.ResourceProvider `query:"resource_providers" doc:"Only works with a resource hosted on any of these download hosts. Comma-separated. Omitted means no filter."`
	ExcludedSoleProviders []workrepr.ResourceProvider `query:"excluded_sole_providers" doc:"Only works whose resources are not hosted solely on these download hosts. Comma-separated. Omitted means no filter."`
	ReleasedFrom          string                      `query:"released_from" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or after this year (YYYY) or month (YYYY-MM)."`
	ReleasedTo            string                      `query:"released_to" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or before this year (YYYY) or month (YYYY-MM)."`
	ReleasedMonths        []MonthNumber               `query:"released_months" doc:"Only works whose release date falls in any of these months (1–12), any year. Comma-separated."`
	CollectedFrom         string                      `query:"collected_from" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"First listed on the forum in or after this year (YYYY) or month (YYYY-MM)."`
	CollectedTo           string                      `query:"collected_to" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"First listed on the forum in or before this year (YYYY) or month (YYYY-MM)."`
	CollectedMonths       []MonthNumber               `query:"collected_months" doc:"Only works first listed on the forum in any of these months (1–12), any year. Comma-separated."`
	MinRating             float64                     `query:"min_rating" minimum:"0" maximum:"10" doc:"Minimum Bayesian forum rating. 0 or omitted means no filter."`
	MinRatingCount        int                         `query:"min_rating_count" minimum:"0" doc:"Minimum number of forum ratings. 0 or omitted means no filter."`
	IncludeNSFW           bool                        `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false. A work whose content_limit has not been synced yet is excluded until it is."`
	IncludeResourceless   bool                        `query:"include_resourceless" default:"false" doc:"When true, published works with no forum resource are included. Default false. A resource-axis or host filter still requires a resource."`
}

type listLibraryWorksInput struct {
	Page         int                  `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit        int                  `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	Q            string               `query:"q" maxLength:"107" doc:"Search keywords. Omitted means the catalog browse population. Free text; never use it as a decision input."`
	Sort         LibraryWorkSortToken `query:"sort" default:"popularity_desc"`
	ReleasedFrom string               `query:"released_from" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or after this year (YYYY) or month (YYYY-MM)."`
	ReleasedTo   string               `query:"released_to" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or before this year (YYYY) or month (YYYY-MM)."`
	IncludeNSFW  bool                 `query:"include_nsfw" default:"false" doc:"When true, works this forum displays as adult are included. Default false."`
}

type listWorkCollectedMonthsInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, months that only hold adult works are included. Default false."`
}

type WorkCollectedMonth struct {
	Year  int `json:"year" minimum:"1" maximum:"9999" format:"int64" doc:"Calendar year the works were first listed on the forum."`
	Month int `json:"month" minimum:"1" maximum:"12" format:"int64" doc:"Calendar month 1–12."`
}

type WorkCollectedMonths struct {
	Object string               `json:"object" enum:"work_collected_months" maxLength:"21" doc:"Type discriminant. Always work_collected_months."`
	Items  []WorkCollectedMonth `json:"items" doc:"Months that have at least one published work with a forum resource, newest year first then month ascending. Empty array, never null."`
}
