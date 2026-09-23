package workrepr

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/resourcevocab"

	"github.com/danielgtaylor/huma/v2"
)

type CompanyRef struct {
	Object string         `json:"object" enum:"company" maxLength:"7" doc:"Type discriminant. Always company."`
	ID     repr.DecimalID `json:"id" doc:"Company id: the catalog company id, which is also the id in the web's /galgame/official/{id}."`
	repr.CatalogName
}

type CatalogIntro struct {
	Locale     string  `json:"locale" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"BCP-47 tag of the text, as catalog records it."`
	Value      string  `json:"value" maxLength:"65535" doc:"The text. Free text; never use it as a decision input."`
	IsMachine  bool    `json:"is_machine" doc:"Whether the text is a machine translation."`
	DataSource *string `json:"data_source" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"Where catalog took the text from, such as vndb or erogamescape. null when unrecorded."`
}

type CatalogLink struct {
	Site string `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"What the link points at, such as official_site, twitter, vndb, bangumi or erogamescape. An open vocabulary; clients label the tokens themselves."`
	URL  string `json:"url" format:"uri" maxLength:"2048" doc:"The address."`
}

type WorkSummary struct {
	repr.WorkRef
	Banner               *repr.Image        `json:"banner" doc:"The landscape art at its original size, never the 16:9 crop. null when the work has none; clients fall back to cover."`
	ReleaseDate          *string            `json:"release_date" format:"date" maxLength:"10" doc:"Release date. A month- or year-precise date is the first day of that month or year. null when catalog has none."`
	ReleaseDatePrecision *string            `json:"release_date_precision" enum:"day,month,year" maxLength:"5" doc:"How much of release_date is known. null when release_date is null."`
	Maker                *CompanyRef        `json:"maker" doc:"The credited company a card names as the maker: developer, then circle, then brand, then publisher. null when no credited company has a name."`
	ViewCount            int                `json:"view_count" minimum:"0" doc:"Times the work's forum page was read. 0 for a work the forum has no page for."`
	LikeCount            int                `json:"like_count" minimum:"0" doc:"Likes on the forum page."`
	RatingScore          *float64           `json:"rating_score" minimum:"0" maximum:"10" doc:"Bayesian average of the forum's ratings, one decimal. null when rating_count is 0."`
	RatingCount          int                `json:"rating_count" minimum:"0" doc:"Forum ratings of the work."`
	ResourcePlatforms    []ResourcePlatform `json:"resource_platforms" doc:"Platforms the work's forum resources run on, each once, in vocabulary order. Empty array, never null."`
	ResourceLanguages    []ResourceLanguage `json:"resource_languages" doc:"Languages of the work's forum resources, each once, in vocabulary order. Empty array, never null."`
	ResourceUpdatedAt    *repr.DateTime     `json:"resource_updated_at" doc:"When a resource of the work last changed. null when it has none."`
	IsPublished          bool               `json:"is_published" doc:"Whether the forum has a listable page for the work. A catalog work the forum has no row for is false, with every forum count 0."`
}

type ResourcePlatform string

func (ResourcePlatform) Schema(huma.Registry) *huma.Schema {
	return vocabSchema(resourcevocab.PlatformKeys, "A resource platform key.")
}

type ResourceLanguage string

func (ResourceLanguage) Schema(huma.Registry) *huma.Schema {
	return vocabSchema(resourcevocab.LanguageKeys, "A resource language key.")
}

type ResourceType string

func (ResourceType) Schema(huma.Registry) *huma.Schema {
	return vocabSchema(resourcevocab.TypeKeys, "A resource type key.")
}

var GameTypes = []string{"ba_saku", "plot", "moe", "daily"}

func vocabSchema(keys []string, desc string) *huma.Schema {
	enum := make([]any, len(keys))
	maxLen := 0
	for i, k := range keys {
		enum[i] = k
		maxLen = max(maxLen, len(k))
	}
	return &huma.Schema{Type: huma.TypeString, Enum: enum, MaxLength: &maxLen, Description: desc}
}
