package apiv1

import (
	"encoding/json"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

const (
	hostPattern      = `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`
	hostInputPattern = `^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`
	slugPattern      = `^[a-z0-9_-]+$`
	languagePattern  = `^[a-z]{2,3}(-[a-z0-9]{2,8})*$`
	hashOrEmpty      = `^([0-9a-f]{64})?$`
)

// SiteURL is one address of a website.
type SiteURL string

func (SiteURL) Schema(huma.Registry) *huma.Schema {
	n := 100
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "uri",
		MaxLength:   &n,
		Description: "An http or https URL of at most 100 characters.",
	}
}

// Language is a BCP 47 tag in lower case, as stored. It is an open vocabulary:
// the tag set is the world's, not the forum's.
type Language string

func (Language) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("bcp47_language_tag", 10)
	s.Pattern = languagePattern
	s.Description = "BCP 47 language tag of the site, lower case, such as zh-cn or ja-jp. An open vocabulary: show an unknown tag as it is."
	return s
}

type WebsiteCategoryRef struct {
	Object string         `json:"object" enum:"website_category" maxLength:"16" doc:"Type discriminant. Always website_category."`
	ID     repr.DecimalID `json:"id" doc:"Category id."`
	Slug   string         `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key of the category page /website-category/{slug}."`
	Label  string         `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
}

type WebsiteCategory struct {
	Object       string         `json:"object" enum:"website_category" maxLength:"16" doc:"Type discriminant. Always website_category."`
	ID           repr.DecimalID `json:"id" doc:"Category id."`
	Slug         string         `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key of the category page /website-category/{slug}."`
	Label        string         `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description  string         `json:"description" maxLength:"300" doc:"Plain-text description. Empty string when none. Free text; never use it as a decision input."`
	SortOrder    int            `json:"sort_order" minimum:"0" maximum:"9999" doc:"Position among categories, ascending."`
	WebsiteCount int            `json:"website_count" minimum:"0" doc:"Every website in the category, NSFW ones included. It is the staff guard for deleting a category; a page that lists websites shows that list's total instead."`
}

type WebsiteTag struct {
	Object            string          `json:"object" enum:"website_tag" maxLength:"11" doc:"Type discriminant. Always website_tag."`
	ID                repr.DecimalID  `json:"id" doc:"Tag id."`
	Slug              string          `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key of the tag page /website-tag/{slug}."`
	Label             string          `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description       string          `json:"description" maxLength:"300" doc:"Plain-text description. Empty string when none. Free text; never use it as a decision input."`
	Level             int             `json:"level" minimum:"-100" maximum:"20" doc:"Weight of the tag. A website's score is the sum of its tags' levels."`
	WebsiteTagGroupID *repr.DecimalID `json:"website_tag_group_id" doc:"Group the tag belongs to. null when it is in none."`
}

type WebsiteTagGroup struct {
	Object        string         `json:"object" enum:"website_tag_group" maxLength:"17" doc:"Type discriminant. Always website_tag_group."`
	ID            repr.DecimalID `json:"id" doc:"Group id."`
	Slug          string         `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"Stable key of the group."`
	Label         string         `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description   string         `json:"description" maxLength:"300" doc:"Plain-text description. Empty string when none. Free text; never use it as a decision input."`
	SortOrder     int            `json:"sort_order" minimum:"0" maximum:"9999" doc:"Position among groups, ascending."`
	IsMultiSelect bool           `json:"is_multi_select" doc:"Whether a website may carry more than one tag of this group."`
}

type WebsiteSummary struct {
	Object          string             `json:"object" enum:"website" maxLength:"7" doc:"Type discriminant. Always website."`
	ID              repr.DecimalID     `json:"id" doc:"Website id."`
	Host            string             `json:"host" pattern:"^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$" maxLength:"253" doc:"Main host name, lower case. It is the page address /website/{host} and can be renamed; refer to a website by id."`
	Title           string             `json:"title" maxLength:"233" doc:"Name of the site. Free text; never use it as a decision input."`
	Description     string             `json:"description" maxLength:"1000" doc:"Plain-text description; keep its line breaks. Free text; never use it as a decision input."`
	Icon            *repr.Image        `json:"icon" doc:"Site icon from the image service. null when the site has none there."`
	ExternalIconURL *string            `json:"external_icon_url" format:"uri" maxLength:"500" doc:"Third-party favicon URL for a site whose icon was never uploaded. null whenever icon is set. Transitional."`
	WebsiteCategory WebsiteCategoryRef `json:"website_category" doc:"Category the site is listed under."`
	IsNSFW          bool               `json:"is_nsfw" doc:"Whether the site is adult-oriented."`
	State           string             `json:"state" enum:"normal,unreachable,closed" maxLength:"11" doc:"normal: the site is up. unreachable: temporarily down. closed: shut down for good."`
	Score           int                `json:"score" minimum:"-2000" maximum:"400" doc:"Sum of the levels of the site's tags. May be negative."`
}

type Website struct {
	Object          string             `json:"object" enum:"website" maxLength:"7" doc:"Type discriminant. Always website."`
	ID              repr.DecimalID     `json:"id" doc:"Website id."`
	Host            string             `json:"host" pattern:"^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$" maxLength:"253" doc:"Main host name, lower case. It is the page address /website/{host} and can be renamed; refer to a website by id."`
	Title           string             `json:"title" maxLength:"233" doc:"Name of the site. Free text; never use it as a decision input."`
	Description     string             `json:"description" maxLength:"1000" doc:"Plain-text description; keep its line breaks. Free text; never use it as a decision input."`
	Icon            *repr.Image        `json:"icon" doc:"Site icon from the image service. null when the site has none there."`
	ExternalIconURL *string            `json:"external_icon_url" format:"uri" maxLength:"500" doc:"Third-party favicon URL for a site whose icon was never uploaded. null whenever icon is set. Transitional."`
	WebsiteCategory WebsiteCategoryRef `json:"website_category" doc:"Category the site is listed under."`
	IsNSFW          bool               `json:"is_nsfw" doc:"Whether the site is adult-oriented. The detail is returned either way; a client that hides NSFW content gates it."`
	State           string             `json:"state" enum:"normal,unreachable,closed" maxLength:"11" doc:"normal: the site is up. unreachable: temporarily down. closed: shut down for good."`
	Score           int                `json:"score" minimum:"-2000" maximum:"400" doc:"Sum of the levels of the site's tags. May be negative."`
	Language        Language           `json:"language" doc:"Main language of the site."`
	URLs            []SiteURL          `json:"urls" maxItems:"10" doc:"Every known address of the site, the main one usually included. Empty array, never null."`
	Founded         string             `json:"founded" maxLength:"20" doc:"When the site was founded, as its operators state it, such as 2014-05-01 or a phrase for about 2014. Empty string when unknown. Free text; never use it as a decision input."`
	WebsiteTags     []WebsiteTag       `json:"website_tags" maxItems:"20" doc:"Tags of the site, ordered by group then by descending level; ungrouped tags last. Empty array, never null."`
	ViewCount       int                `json:"view_count" minimum:"0" doc:"Lifetime view count. Each read of this operation adds one."`
	LikeCount       int                `json:"like_count" minimum:"0" doc:"Like count."`
	FavoriteCount   int                `json:"favorite_count" minimum:"0" doc:"Favorite count."`
	CommentCount    int                `json:"comment_count" minimum:"0" doc:"Comments on the site's wall."`
	CreatedAt       repr.DateTime      `json:"created_at" doc:"Time the site was listed."`
	UpdatedAt       repr.DateTime      `json:"updated_at" doc:"Time of the latest edit. Values before 2026-09-23 may be the time of a view instead."`
	Viewer          *WebsiteViewer     `json:"viewer" doc:"The caller's own state on the site. null for an anonymous caller."`
}

type WebsiteViewer struct {
	HasLiked     bool `json:"has_liked" doc:"Whether the caller liked the site."`
	HasFavorited bool `json:"has_favorited" doc:"Whether the caller favorited the site."`
	CanEdit      bool `json:"can_edit" doc:"Whether the caller holds website.edit. Requests authenticated with a Bearer token never carry it."`
	CanDelete    bool `json:"can_delete" doc:"Whether the caller holds website.delete. Requests authenticated with a Bearer token never carry it."`
}

type WebsiteEngagement struct {
	Object        string                   `json:"object" enum:"website_engagement" maxLength:"18" doc:"Type discriminant. Always website_engagement."`
	WebsiteID     repr.DecimalID           `json:"website_id" doc:"Id of the website."`
	LikeCount     int                      `json:"like_count" minimum:"0" doc:"Like count after this request."`
	FavoriteCount int                      `json:"favorite_count" minimum:"0" doc:"Favorite count after this request."`
	Viewer        *WebsiteEngagementViewer `json:"viewer" doc:"The caller's state after this request."`
}

type WebsiteEngagementViewer struct {
	HasLiked     bool `json:"has_liked" doc:"Whether the caller liked the site."`
	HasFavorited bool `json:"has_favorited" doc:"Whether the caller favorited the site."`
}

type AdminWebsite struct {
	Object            string           `json:"object" enum:"admin_website" maxLength:"13" doc:"Type discriminant. Always admin_website."`
	ID                repr.DecimalID   `json:"id" doc:"Website id."`
	Host              string           `json:"host" pattern:"^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$" maxLength:"253" doc:"Main host name, lower case."`
	Title             string           `json:"title" maxLength:"233" doc:"Name of the site. Free text; never use it as a decision input."`
	Description       string           `json:"description" maxLength:"1000" doc:"Plain-text description. Free text; never use it as a decision input."`
	Icon              *repr.Image      `json:"icon" doc:"Current icon, for a preview. null when the site has none in the image service."`
	ExternalIconURL   *string          `json:"external_icon_url" format:"uri" maxLength:"500" doc:"Third-party favicon URL; read-only. null whenever icon is set."`
	WebsiteCategoryID repr.DecimalID   `json:"website_category_id" doc:"Category id."`
	WebsiteTagIDs     []repr.DecimalID `json:"website_tag_ids" maxItems:"20" doc:"Tag ids, ascending. Empty array, never null."`
	IsNSFW            bool             `json:"is_nsfw" doc:"Whether the site is adult-oriented."`
	State             string           `json:"state" enum:"normal,unreachable,closed" maxLength:"11" doc:"Lifecycle state."`
	Language          Language         `json:"language" doc:"Main language of the site."`
	URLs              []SiteURL        `json:"urls" maxItems:"10" doc:"Every known address of the site. Empty array, never null."`
	Founded           string           `json:"founded" maxLength:"20" doc:"When the site was founded, as its operators state it. Free text; never use it as a decision input."`
	CreatedAt         repr.DateTime    `json:"created_at" doc:"Time the site was listed."`
	UpdatedAt         repr.DateTime    `json:"updated_at" doc:"Time of the latest write to the row."`
}

type WebsiteCreate struct {
	Host              string           `json:"host" minLength:"4" maxLength:"253" pattern:"^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$" doc:"Main host name without scheme or path. Stored lower case."`
	Title             string           `json:"title" minLength:"1" maxLength:"233" doc:"Name of the site. Only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
	Description       string           `json:"description" minLength:"10" maxLength:"1000" doc:"Plain-text description, at least 10 characters once trimmed. Free text; never use it as a decision input."`
	IconImageHash     *string          `json:"icon_image_hash,omitempty" pattern:"^([0-9a-f]{64})?$" maxLength:"64" doc:"Image-service hash of the icon. Absent or empty for none."`
	WebsiteCategoryID repr.DecimalID   `json:"website_category_id" doc:"Category id. An unknown id is UNKNOWN_REFERENCE."`
	WebsiteTagIDs     []repr.DecimalID `json:"website_tag_ids" maxItems:"20" doc:"Tag ids: no duplicates, each existing, at most one per single-select group."`
	IsNSFW            bool             `json:"is_nsfw" doc:"Whether the site is adult-oriented."`
	State             *string          `json:"state,omitempty" enum:"normal,unreachable,closed" maxLength:"11" doc:"Lifecycle state. Absent means normal."`
	Language          string           `json:"language" maxLength:"10" pattern:"^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$" doc:"BCP 47 language tag. Stored lower case."`
	URLs              []SiteURL        `json:"urls" required:"false" maxItems:"10" doc:"Every known address of the site, each an http or https URL of at most 100 characters. Absent means none."`
	Founded           *string          `json:"founded,omitempty" maxLength:"20" doc:"When the site was founded, trimmed. Absent means unknown. Free text; never use it as a decision input."`
}

type WebsitePatch struct {
	Host              *string          `json:"host,omitempty" minLength:"4" maxLength:"253" pattern:"^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$" doc:"New main host name. Stored lower case."`
	Title             *string          `json:"title,omitempty" minLength:"1" maxLength:"233" doc:"New name. Free text; never use it as a decision input."`
	Description       *string          `json:"description,omitempty" minLength:"10" maxLength:"1000" doc:"New description. Free text; never use it as a decision input."`
	IconImageHash     *string          `json:"icon_image_hash,omitempty" pattern:"^([0-9a-f]{64})?$" maxLength:"64" doc:"New icon hash; empty string removes the icon."`
	WebsiteCategoryID *repr.DecimalID  `json:"website_category_id,omitempty" doc:"New category id."`
	WebsiteTagIDs     []repr.DecimalID `json:"website_tag_ids" required:"false" maxItems:"20" doc:"When present, replaces every tag of the site; absent keeps them. Same rules as createAdminWebsite."`
	IsNSFW            *bool            `json:"is_nsfw,omitempty" doc:"New NSFW flag."`
	State             *string          `json:"state,omitempty" enum:"normal,unreachable,closed" maxLength:"11" doc:"New lifecycle state."`
	Language          *string          `json:"language,omitempty" maxLength:"10" pattern:"^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$" doc:"New BCP 47 language tag."`
	URLs              []SiteURL        `json:"urls" required:"false" maxItems:"10" doc:"When present, replaces every address. Same rules as createAdminWebsite."`
	Founded           *string          `json:"founded,omitempty" maxLength:"20" doc:"New founding text; empty string clears it. Free text; never use it as a decision input."`
}

type AdminWebsiteCategory struct {
	Object      string         `json:"object" enum:"admin_website_category" maxLength:"22" doc:"Type discriminant. Always admin_website_category."`
	ID          repr.DecimalID `json:"id" doc:"Category id."`
	Slug        string         `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key."`
	Label       string         `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description string         `json:"description" maxLength:"300" doc:"Plain-text description. Free text; never use it as a decision input."`
	SortOrder   int            `json:"sort_order" minimum:"0" maximum:"9999" doc:"Position among categories, ascending."`
	CreatedAt   repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt   repr.DateTime  `json:"updated_at" doc:"Time of the latest write."`
}

type AdminWebsiteTag struct {
	Object            string          `json:"object" enum:"admin_website_tag" maxLength:"17" doc:"Type discriminant. Always admin_website_tag."`
	ID                repr.DecimalID  `json:"id" doc:"Tag id."`
	Slug              string          `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key."`
	Label             string          `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description       string          `json:"description" maxLength:"300" doc:"Plain-text description. Free text; never use it as a decision input."`
	Level             int             `json:"level" minimum:"-100" maximum:"20" doc:"Weight of the tag."`
	WebsiteTagGroupID *repr.DecimalID `json:"website_tag_group_id" doc:"Group id. null when ungrouped."`
	CreatedAt         repr.DateTime   `json:"created_at" doc:"Creation time."`
	UpdatedAt         repr.DateTime   `json:"updated_at" doc:"Time of the latest write."`
}

type AdminWebsiteTagGroup struct {
	Object        string         `json:"object" enum:"admin_website_tag_group" maxLength:"23" doc:"Type discriminant. Always admin_website_tag_group."`
	ID            repr.DecimalID `json:"id" doc:"Group id."`
	Slug          string         `json:"slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"Stable key."`
	Label         string         `json:"label" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description   string         `json:"description" maxLength:"300" doc:"Plain-text description. Free text; never use it as a decision input."`
	SortOrder     int            `json:"sort_order" minimum:"0" maximum:"9999" doc:"Position among groups, ascending."`
	IsMultiSelect bool           `json:"is_multi_select" doc:"Whether a website may carry more than one tag of this group."`
	CreatedAt     repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt     repr.DateTime  `json:"updated_at" doc:"Time of the latest write."`
}

type CategoryCreate struct {
	Slug        string `json:"slug" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"URL key. Taken is ALREADY_EXISTS."`
	Label       string `json:"label" minLength:"1" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description string `json:"description" required:"false" maxLength:"300" doc:"Plain-text description. Absent means empty. Free text; never use it as a decision input."`
	SortOrder   int    `json:"sort_order" required:"false" minimum:"0" maximum:"9999" doc:"Position, ascending. Absent means 0."`
}

type CategoryPatch struct {
	Slug        *string `json:"slug,omitempty" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"New URL key."`
	Label       *string `json:"label,omitempty" minLength:"1" maxLength:"30" doc:"New display name. Free text; never use it as a decision input."`
	Description *string `json:"description,omitempty" maxLength:"300" doc:"New description. Free text; never use it as a decision input."`
	SortOrder   *int    `json:"sort_order,omitempty" minimum:"0" maximum:"9999" doc:"New position."`
}

type TagCreate struct {
	Slug              string          `json:"slug" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"URL key. Taken is ALREADY_EXISTS."`
	Label             string          `json:"label" minLength:"1" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description       string          `json:"description" required:"false" maxLength:"300" doc:"Plain-text description. Absent means empty. Free text; never use it as a decision input."`
	Level             int             `json:"level" minimum:"-100" maximum:"20" doc:"Weight of the tag."`
	WebsiteTagGroupID TagGroupRef     `json:"website_tag_group_id" required:"false" doc:"Group id. Absent or null means ungrouped; an unknown id is UNKNOWN_REFERENCE."`
}

// TagGroupRef tells an absent field (keep) from an explicit null (ungroup).
type TagGroupRef struct {
	Present bool
	Value   *repr.DecimalID
}

func (r *TagGroupRef) UnmarshalJSON(data []byte) error {
	r.Present = true
	if string(data) == "null" {
		r.Value = nil
		return nil
	}
	var v repr.DecimalID
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	r.Value = &v
	return nil
}

func (TagGroupRef) Schema(reg huma.Registry) *huma.Schema {
	s := repr.DecimalID("").Schema(reg)
	s.Nullable = true
	s.Description = "New group id. null moves the tag out of every group; absent keeps it."
	return s
}

type TagPatch struct {
	Slug              *string     `json:"slug,omitempty" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"New URL key."`
	Label             *string     `json:"label,omitempty" minLength:"1" maxLength:"30" doc:"New display name. Free text; never use it as a decision input."`
	Description       *string     `json:"description,omitempty" maxLength:"300" doc:"New description. Free text; never use it as a decision input."`
	Level             *int        `json:"level,omitempty" minimum:"-100" maximum:"20" doc:"New weight."`
	WebsiteTagGroupID TagGroupRef `json:"website_tag_group_id" required:"false" doc:"New group id. null moves the tag out of every group; absent keeps it."`
}

type TagGroupCreate struct {
	Slug          string `json:"slug" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"Stable key. Taken is ALREADY_EXISTS."`
	Label         string `json:"label" minLength:"1" maxLength:"30" doc:"Display name. Free text; never use it as a decision input."`
	Description   string `json:"description" required:"false" maxLength:"300" doc:"Plain-text description. Absent means empty. Free text; never use it as a decision input."`
	SortOrder     int    `json:"sort_order" required:"false" minimum:"0" maximum:"9999" doc:"Position, ascending. Absent means 0."`
	IsMultiSelect bool   `json:"is_multi_select" required:"false" doc:"Whether a website may carry more than one tag of this group. Absent means false."`
}

type TagGroupPatch struct {
	Slug          *string `json:"slug,omitempty" minLength:"1" maxLength:"30" pattern:"^[a-z0-9_-]+$" doc:"New key."`
	Label         *string `json:"label,omitempty" minLength:"1" maxLength:"30" doc:"New display name. Free text; never use it as a decision input."`
	Description   *string `json:"description,omitempty" maxLength:"300" doc:"New description. Free text; never use it as a decision input."`
	SortOrder     *int    `json:"sort_order,omitempty" minimum:"0" maximum:"9999" doc:"New position."`
	IsMultiSelect *bool   `json:"is_multi_select,omitempty" doc:"New multi-select flag. Turning it off does not touch websites that already carry several tags of the group."`
}

type listWebsitesInput struct {
	collect.Page
	collect.Total
	IncludeNSFW       bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW sites are included. Default false."`
	WebsiteCategoryID string `query:"website_category_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only sites in this category."`
	WebsiteTagID      string `query:"website_tag_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only sites carrying this tag."`
}

type listWebsitesOutput struct {
	Body repr.CountedList[WebsiteSummary]
}

type hostInput struct {
	WebsiteHost string `path:"website_host" maxLength:"253" pattern:"^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$" doc:"Main host name of the site, lower case."`
}

type websiteOutput struct {
	Body Website
}

type engagementOutput struct {
	Body WebsiteEngagement
}

type websiteIDInput struct {
	WebsiteID string `path:"website_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Website id."`
}

type adminWebsiteOutput struct {
	Body AdminWebsite
}

type createWebsiteInput struct {
	Body WebsiteCreate
}

type createWebsiteOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new site's edit source, such as /api/v1/admin/websites/96."`
	Body     AdminWebsite
}

type patchWebsiteInput struct {
	WebsiteID string `path:"website_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Website id."`
	Body      WebsitePatch
}

type noContentOutput struct{}

type listVocabularyInput struct {
	collect.Page
}

type listCategoriesOutput struct {
	Body repr.List[WebsiteCategory]
}

type categorySlugInput struct {
	WebsiteCategorySlug string `path:"website_category_slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key of the category."`
}

type categoryOutput struct {
	Body WebsiteCategory
}

type categoryIDInput struct {
	WebsiteCategoryID string `path:"website_category_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Category id."`
}

type adminCategoryOutput struct {
	Body AdminWebsiteCategory
}

type createCategoryInput struct {
	Body CategoryCreate
}

type createCategoryOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new category, such as /api/v1/admin/website-categories/5."`
	Body     AdminWebsiteCategory
}

type patchCategoryInput struct {
	WebsiteCategoryID string `path:"website_category_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Category id."`
	Body              CategoryPatch
}

type listTagsOutput struct {
	Body repr.List[WebsiteTag]
}

type tagSlugInput struct {
	WebsiteTagSlug string `path:"website_tag_slug" pattern:"^[a-z0-9_-]+$" maxLength:"30" doc:"URL key of the tag."`
}

type tagOutput struct {
	Body WebsiteTag
}

type tagIDInput struct {
	WebsiteTagID string `path:"website_tag_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Tag id."`
}

type adminTagOutput struct {
	Body AdminWebsiteTag
}

type createTagInput struct {
	Body TagCreate
}

type createTagOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new tag, such as /api/v1/admin/website-tags/72."`
	Body     AdminWebsiteTag
}

type patchTagInput struct {
	WebsiteTagID string `path:"website_tag_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Tag id."`
	Body         TagPatch
}

type listTagGroupsOutput struct {
	Body repr.List[WebsiteTagGroup]
}

type tagGroupIDInput struct {
	WebsiteTagGroupID string `path:"website_tag_group_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Group id."`
}

type adminTagGroupOutput struct {
	Body AdminWebsiteTagGroup
}

type createTagGroupInput struct {
	Body TagGroupCreate
}

type createTagGroupOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new group, such as /api/v1/admin/website-tag-groups/19."`
	Body     AdminWebsiteTagGroup
}

type patchTagGroupInput struct {
	WebsiteTagGroupID string `path:"website_tag_group_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Group id."`
	Body              TagGroupPatch
}
