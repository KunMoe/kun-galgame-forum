package content

import (
	"github.com/danielgtaylor/huma/v2"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
)

type WorkPreview struct {
	repr.WorkRef
	Maker                *workrepr.CompanyRef `json:"maker" doc:"The credited company a card names as the maker: developer, then circle, then brand, then publisher. null when no credited company has a name."`
	TopTags              []workrepr.TagRef    `json:"top_tags" maxItems:"4" doc:"Up to four of the work's tags, in the order its page lists them, leaving out spoiler and adult tags. Empty array, never null."`
	ReleaseDate          *string              `json:"release_date" format:"date" maxLength:"10" doc:"Release date. A month- or year-precise date is the first day of that month or year. null when catalog has none."`
	ReleaseDatePrecision *string              `json:"release_date_precision" enum:"day,month,year" maxLength:"5" doc:"How much of release_date is known. null when release_date is null."`
}

type CharacterPreview struct {
	Object string         `json:"object" enum:"character" maxLength:"9" doc:"Type discriminant. Always character."`
	ID     repr.DecimalID `json:"id" doc:"Character id: the catalog character id, which is also the id in the web's /galgame/character/{id}."`
	repr.CatalogName
	Image *repr.Image    `json:"image" doc:"The character's portrait, only when catalog grades it safe. null otherwise."`
	Works []repr.WorkRef `json:"works" maxItems:"3" doc:"Up to three works the character appears in, in catalog's order. Empty array, never null."`
}

type CompanyPreview struct {
	Object string         `json:"object" enum:"company" maxLength:"7" doc:"Type discriminant. Always company."`
	ID     repr.DecimalID `json:"id" doc:"Company id: the catalog company id, which is also the id in the web's /galgame/official/{id}."`
	repr.CatalogName
	CompanyKind      string      `json:"company_kind" enum:"game_brand,bunko,publisher,anime_studio,doujin_circle,group" maxLength:"13" doc:"What sort of company it is."`
	Logo             *repr.Image `json:"logo" doc:"The company's logo. null when catalog has none."`
	CatalogWorkCount int         `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included."`
}

type TagPreview struct {
	Object string         `json:"object" enum:"tag" maxLength:"3" doc:"Type discriminant. Always tag."`
	ID     repr.DecimalID `json:"id" doc:"Tag id: the catalog tag id, which is also the id in the web's /galgame/tag/{id}."`
	repr.CatalogName
	TagKind          string `json:"tag_kind" enum:"content,meta" maxLength:"7" doc:"What the tag describes: content is the story and characters, meta is the game as a product."`
	IsSexual         bool   `json:"is_sexual" doc:"Whether the tag is adult content."`
	CatalogWorkCount int    `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included."`
	Description      string `json:"description" maxLength:"200" doc:"Start of the tag's introduction, Chinese when catalog has it. Empty string when it has none. Free text; never use it as a decision input."`
}

type SeriesPreview struct {
	Object string         `json:"object" enum:"series" maxLength:"6" doc:"Type discriminant. Always series."`
	ID     repr.DecimalID `json:"id" doc:"Series id: the catalog series id, which is also the id in the web's /galgame/series/{id}."`
	repr.CatalogName
	Cover            *repr.Image    `json:"cover" doc:"Portrait cover of the series' earliest listed work. null when none is listed or it has no cover."`
	Works            []repr.WorkRef `json:"works" maxItems:"3" doc:"Up to three listed works of the series, earliest release first. Empty array, never null."`
	HasNSFWWorks     *bool          `json:"has_nsfw_works" doc:"Whether any work of the series is adult content. null when catalog does not know."`
	CatalogWorkCount int            `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included."`
}

type CreditNamePreview struct {
	Object string         `json:"object" enum:"credit_name" maxLength:"11" doc:"Type discriminant. Always credit_name."`
	ID     repr.DecimalID `json:"id" doc:"Credit name id: the catalog credit name id, which is also the id in the web's /galgame/staff/{id}."`
	repr.CatalogName
	Photo     *repr.Image      `json:"photo" doc:"A photo of the person. null when catalog has none."`
	RoleNames []CreditRoleName `json:"role_names" maxItems:"3" doc:"The person's most frequent credited roles as catalog names them, most frequent first. Empty array, never null."`
	Works     []repr.WorkRef   `json:"works" maxItems:"3" doc:"Up to three works the person is credited on, in catalog's order. Empty array, never null."`
}

type RatingPreview struct {
	Object       string         `json:"object" enum:"rating" maxLength:"6" doc:"Type discriminant. Always rating."`
	ID           repr.DecimalID `json:"id" doc:"Rating id, which is also the id in the web's /galgame-rating/{id}."`
	Work         *repr.WorkRef  `json:"work" doc:"The rated work. Never null here; nullable to match WorkRef elsewhere."`
	Author       repr.UserRef   `json:"author" doc:"Who wrote the rating."`
	Overall      int            `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
	ShortSummary string         `json:"short_summary" maxLength:"2000" doc:"The author's short review, plain text. Empty string if none. Free text; never use it as a decision input."`
	HasSpoilers  bool           `json:"has_spoilers" doc:"Whether the author marked the rating as containing spoilers. Hide short_summary until the reader asks."`
	CreatedAt    repr.DateTime  `json:"created_at" doc:"When the rating was written."`
}

func WorkTarget(p WorkPreview) EntityTarget             { return EntityTarget{p} }
func CharacterTarget(p CharacterPreview) EntityTarget   { return EntityTarget{p} }
func CompanyTarget(p CompanyPreview) EntityTarget       { return EntityTarget{p} }
func TagTarget(p TagPreview) EntityTarget               { return EntityTarget{p} }
func SeriesTarget(p SeriesPreview) EntityTarget         { return EntityTarget{p} }
func CreditNameTarget(p CreditNamePreview) EntityTarget { return EntityTarget{p} }
func RatingTarget(p RatingPreview) EntityTarget         { return EntityTarget{p} }

type CreditRoleName string

func (CreditRoleName) Schema(huma.Registry) *huma.Schema {
	n := 128
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n,
		Description: "A credited role as catalog names it, such as Scenario. " + repr.FreeTextSentence}
}
