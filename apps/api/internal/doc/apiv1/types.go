package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

const bannerHashRegex = `^([0-9a-f]{64})?$`

var docCategories = []string{"galgame", "notice", "kun", "other"}

type DocSummary struct {
	Object      string         `json:"object" enum:"doc" maxLength:"3" doc:"Type discriminant. Always doc."`
	ID          repr.DecimalID `json:"id" doc:"Doc id. JSON string of a decimal integer."`
	Slug        string         `json:"slug" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" maxLength:"128" doc:"URL segment. The doc's page is /doc/{slug}."`
	Title       string         `json:"title" maxLength:"233" doc:"Doc title. Free text; never use it as a decision input."`
	Description string         `json:"description" maxLength:"777" doc:"Short summary shown on cards. Empty string if none. Free text; never use it as a decision input."`
	DocCategory string         `json:"doc_category" enum:"galgame,notice,kun,other" maxLength:"7" doc:"Which shelf of the help center the doc sits on. Clients label the tokens themselves."`
	Banner      *repr.Image    `json:"banner" doc:"Banner image. null when the doc has none."`
	IsPinned    bool           `json:"is_pinned" doc:"Whether the doc is pinned to the home carousel."`
	ViewCount   int            `json:"view_count" minimum:"0" doc:"Times the doc page was read."`
	PublishedAt repr.DateTime  `json:"published_at" doc:"When the doc was published. Never changes."`
	EditedAt    *repr.DateTime `json:"edited_at" doc:"When the doc's text or metadata last changed. Pinning does not count. null if never edited."`
}

type Doc struct {
	Object      string                  `json:"object" enum:"doc" maxLength:"3" doc:"Type discriminant. Always doc."`
	ID          repr.DecimalID          `json:"id" doc:"Doc id. JSON string of a decimal integer."`
	Slug        string                  `json:"slug" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" maxLength:"128" doc:"URL segment. The doc's page is /doc/{slug}."`
	Title       string                  `json:"title" maxLength:"233" doc:"Doc title. Free text; never use it as a decision input."`
	Description string                  `json:"description" maxLength:"777" doc:"Short summary shown on cards. Empty string if none. Free text; never use it as a decision input."`
	DocCategory string                  `json:"doc_category" enum:"galgame,notice,kun,other" maxLength:"7" doc:"Which shelf of the help center the doc sits on. Clients label the tokens themselves."`
	Banner      *repr.Image             `json:"banner" doc:"Banner image. null when the doc has none."`
	IsPinned    bool                    `json:"is_pinned" doc:"Whether the doc is pinned to the home carousel."`
	ViewCount   int                     `json:"view_count" minimum:"0" doc:"Times the doc page was read, this read included."`
	PublishedAt repr.DateTime           `json:"published_at" doc:"When the doc was published. Never changes."`
	EditedAt    *repr.DateTime          `json:"edited_at" doc:"When the doc's text or metadata last changed. Pinning does not count. null if never edited."`
	Author      repr.UserRef            `json:"author" doc:"Who wrote the doc."`
	Content     content.ContentDocument `json:"content" doc:"Doc body as a node tree. A table of contents comes from its heading nodes."`
}

type AdminDoc struct {
	Object          string         `json:"object" enum:"admin_doc" maxLength:"9" doc:"Type discriminant. Always admin_doc."`
	ID              repr.DecimalID `json:"id" doc:"Doc id. JSON string of a decimal integer."`
	Slug            string         `json:"slug" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" maxLength:"128" doc:"URL segment. The doc's page is /doc/{slug}."`
	Title           string         `json:"title" maxLength:"233" doc:"Doc title. Free text; never use it as a decision input."`
	Description     string         `json:"description" maxLength:"777" doc:"Short summary shown on cards. Empty string if none. Free text; never use it as a decision input."`
	DocCategory     string         `json:"doc_category" enum:"galgame,notice,kun,other" maxLength:"7" doc:"Which shelf of the help center the doc sits on."`
	Banner          *repr.Image    `json:"banner" doc:"Banner image. null when the doc has none."`
	IsPinned        bool           `json:"is_pinned" doc:"Whether the doc is pinned to the home carousel."`
	ViewCount       int            `json:"view_count" minimum:"0" doc:"Times the doc page was read."`
	PublishedAt     repr.DateTime  `json:"published_at" doc:"When the doc was published. Never changes."`
	EditedAt        *repr.DateTime `json:"edited_at" doc:"When the doc's text or metadata last changed. Pinning does not count. null if never edited."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"100007" doc:"Doc body as the stored Markdown source. Free text; never use it as a decision input."`
}

type BannerImageHash string

func (BannerImageHash) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     bannerHashRegex,
		MaxLength:   &n,
		Description: "Image-service content hash of the banner, or an empty string for no banner.",
	}
}

type DocCreate struct {
	Slug            string           `json:"slug" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" minLength:"1" maxLength:"128" doc:"URL segment: lowercase letters and digits in hyphen-separated runs. Must be unused; a taken slug is ALREADY_EXISTS."`
	Title           string           `json:"title" minLength:"1" maxLength:"233" doc:"Doc title. Leading and trailing whitespace is removed before it is stored, and a title of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	Description     *string          `json:"description,omitempty" maxLength:"777" doc:"Short summary. Trimmed before it is stored. Absent means an empty summary. Free text; never use it as a decision input."`
	DocCategory     string           `json:"doc_category" enum:"galgame,notice,kun,other" maxLength:"7" doc:"Shelf of the help center."`
	BannerImageHash *BannerImageHash `json:"banner_image_hash,omitempty" doc:"Banner by image-service hash. Absent or an empty string means no banner."`
	IsPinned        *bool            `json:"is_pinned,omitempty" doc:"Pin the doc to the home carousel. Absent means false."`
	ContentMarkdown string           `json:"content_markdown" minLength:"1" maxLength:"100000" doc:"Doc body as Markdown source. A body of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
}

type DocPatch struct {
	Slug            *string          `json:"slug,omitempty" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" minLength:"1" maxLength:"128" doc:"New URL segment. The old /doc/{slug} stops resolving. A taken slug is ALREADY_EXISTS."`
	Title           *string          `json:"title,omitempty" minLength:"1" maxLength:"233" doc:"New title. Trimmed and checked as in createDoc. Free text; never use it as a decision input."`
	Description     *string          `json:"description,omitempty" maxLength:"777" doc:"New summary. Trimmed before it is stored. Free text; never use it as a decision input."`
	DocCategory     *string          `json:"doc_category,omitempty" enum:"galgame,notice,kun,other" maxLength:"7" doc:"New shelf."`
	BannerImageHash *BannerImageHash `json:"banner_image_hash,omitempty" doc:"New banner by image-service hash. An empty string removes the banner."`
	IsPinned        *bool            `json:"is_pinned,omitempty" doc:"New pin flag. Changing only this leaves edited_at alone."`
	ContentMarkdown *string          `json:"content_markdown,omitempty" minLength:"1" maxLength:"100000" doc:"New body as Markdown source. Checked as in createDoc. Free text; never use it as a decision input."`
}

type DocOrder struct {
	DocIDs []repr.DecimalID `json:"doc_ids" minItems:"1" maxItems:"1000" uniqueItems:"true" doc:"Every doc's id, once each, in the new display order. An id that is not a doc is UNKNOWN_REFERENCE; leaving a doc out is TOO_FEW_ITEMS with min_items set to the number of docs."`
}
