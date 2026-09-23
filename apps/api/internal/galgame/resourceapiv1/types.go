package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/resourcevocab"

	"github.com/danielgtaylor/huma/v2"
)

const defaultLimit = 50

const downloadURLPattern = "^(https?|ftps?|magnet|ed2k|thunder):"

type ResourceLanguage string

func (ResourceLanguage) Schema(huma.Registry) *huma.Schema {
	n := 5
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"zh-cn", "zh-tw", "ja-jp", "en-us", "other"},
		MaxLength:   &n,
		Description: "A resource language. Lower-case BCP 47, plus other.",
	}
}

type ResourcePlatform string

func (ResourcePlatform) Schema(huma.Registry) *huma.Schema {
	n := 3
	enum := make([]any, len(resourcevocab.PlatformKeys))
	for i, k := range resourcevocab.PlatformKeys {
		enum[i] = k
	}
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &n,
		Description: "A resource platform token, such as win or and.",
	}
}

type ResourceRuntime string

func (ResourceRuntime) Schema(huma.Registry) *huma.Schema {
	n := 13
	enum := make([]any, len(resourcevocab.RuntimeKeys))
	for i, k := range resourcevocab.RuntimeKeys {
		enum[i] = k
	}
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &n,
		Description: "A resource runtime token, such as native-win. Lower-case kebab-case, plus other.",
	}
}

type DownloadURL string

func (DownloadURL) Schema(huma.Registry) *huma.Schema {
	n := 4096
	return &huma.Schema{
		Type:        huma.TypeString,
		MaxLength:   &n,
		Pattern:     downloadURLPattern,
		Description: "A download URL: http, https, ftp, ftps, magnet, ed2k or thunder, at most 4096 characters.",
	}
}

type ProviderName string

func (ProviderName) Schema(huma.Registry) *huma.Schema {
	n := 256
	return &huma.Schema{
		Type:        huma.TypeString,
		MaxLength:   &n,
		Description: "A download-host name derived from the URLs. Free text; never use it as a decision input.",
	}
}

type GalgameResource struct {
	Object            string                  `json:"object" enum:"galgame_resource" maxLength:"16" doc:"Type discriminant. Always galgame_resource."`
	ID                repr.DecimalID          `json:"id" doc:"Resource id."`
	Work              *repr.WorkRef           `json:"work" doc:"The work this resource belongs to. Never null on this face; nullable to match WorkRef elsewhere."`
	Author            repr.UserRef            `json:"author" doc:"Uploader of the resource."`
	ResourceType      string                  `json:"resource_type" enum:"game,patch,collection,crack_fix,mod,tool,walkthrough,ost,voice,cg,wallpaper,artbook,video,other" maxLength:"11" doc:"Kind of download."`
	ResourceLanguages []ResourceLanguage      `json:"resource_languages" minItems:"1" doc:"Languages of the resource. Never empty, never null."`
	ResourcePlatforms []ResourcePlatform      `json:"resource_platforms" doc:"Platforms of the resource. Empty array when only runtimes apply. Never null."`
	ResourceRuntimes  []ResourceRuntime       `json:"resource_runtimes" doc:"Runtimes of the resource. Empty array when none. Never null."`
	Title             string                  `json:"title" maxLength:"200" doc:"Optional title. Empty string when none. Free text; never use it as a decision input."`
	VersionLabel      *string                 `json:"version_label" enum:"official_latest,stable,mirror,localized,unknown" maxLength:"15" doc:"Version token. null when none."`
	Size              string                  `json:"size" maxLength:"64" doc:"Size as the uploader wrote it, such as 1.5 GB. Free text; never use it as a decision input."`
	ProviderNames     []ProviderName          `json:"provider_names" doc:"Host names derived from the download URLs. Empty array, never null."`
	Content           content.ContentDocument `json:"content" doc:"Note as a Markdown document. An empty document when there is no note."`
	State             string                  `json:"state" enum:"valid,expired" maxLength:"7" doc:"Whether the links are currently treated as working."`
	ViewCount         int                     `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	DownloadCount     int                     `json:"download_count" minimum:"0" doc:"Times download secrets were issued."`
	LikeCount         int                     `json:"like_count" minimum:"0" doc:"Number of likes."`
	CommentCount      int                     `json:"comment_count" minimum:"0" doc:"Comments on the resource's wall."`
	CreatedAt         repr.DateTime           `json:"created_at" doc:"Creation time."`
	UpdatedAt         repr.DateTime           `json:"updated_at" doc:"Time of the latest write to the row."`
	EditedAt          *repr.DateTime          `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	Dlsite            *DlsiteOffer            `json:"dlsite" doc:"DLsite purchase offer for the work. null when there is no DLsite workno."`
	Viewer            *GalgameResourceViewer  `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type GalgameResourceViewer struct {
	HasLiked  bool `json:"has_liked" doc:"Whether the caller liked this resource."`
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may edit this resource. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete this resource. Requests authenticated with a Bearer token never carry staff powers."`
}

type DlsiteOffer struct {
	PurchaseURL  string  `json:"purchase_url" format:"uri" maxLength:"4096" doc:"Short link or affiliate template for the DLsite product."`
	CouponURL    *string `json:"coupon_url" format:"uri" maxLength:"4096" doc:"Coupon or campaign landing URL. null when none."`
	CampaignName *string `json:"campaign_name" maxLength:"256" doc:"Name of a running campaign. null on the static coupon page. Free text; never use it as a decision input."`
}

type GalgameResourceDownload struct {
	Object          string        `json:"object" enum:"galgame_resource_download" maxLength:"25" doc:"Type discriminant. Always galgame_resource_download."`
	DownloadURLs    []DownloadURL `json:"download_urls" minItems:"1" maxItems:"20" doc:"Stored download links, in row order. Never null."`
	ExtractionCode  string        `json:"extraction_code" maxLength:"1007" doc:"Extraction code. Empty string when none. Free text; never use it as a decision input."`
	ArchivePassword string        `json:"archive_password" maxLength:"1007" doc:"Archive password. Empty string when none. Free text; never use it as a decision input."`
}

type GalgameResourceSource struct {
	Object            string             `json:"object" enum:"galgame_resource_source" maxLength:"23" doc:"Type discriminant. Always galgame_resource_source."`
	ResourceID        repr.DecimalID     `json:"resource_id" doc:"Id of the resource."`
	ResourceType      string             `json:"resource_type" enum:"game,patch,collection,crack_fix,mod,tool,walkthrough,ost,voice,cg,wallpaper,artbook,video,other" maxLength:"11" doc:"Kind of download."`
	Title             string             `json:"title" maxLength:"200" doc:"Stored title. Empty string when none. Free text; never use it as a decision input."`
	VersionLabel      *string            `json:"version_label" enum:"official_latest,stable,mirror,localized,unknown" maxLength:"15" doc:"Version token. null when none."`
	ResourceLanguages []ResourceLanguage `json:"resource_languages" minItems:"1" doc:"Languages of the resource. Never empty, never null."`
	ResourcePlatforms []ResourcePlatform `json:"resource_platforms" doc:"Platforms of the resource. Empty array when only runtimes apply. Never null."`
	ResourceRuntimes  []ResourceRuntime  `json:"resource_runtimes" doc:"Runtimes of the resource. Empty array when none. Never null."`
	Size              string             `json:"size" maxLength:"64" doc:"Size as the uploader wrote it. Free text; never use it as a decision input."`
	DownloadURLs      []DownloadURL      `json:"download_urls" minItems:"1" maxItems:"20" doc:"Stored download links, in row order. Never null."`
	ExtractionCode    string             `json:"extraction_code" maxLength:"1007" doc:"Extraction code. Empty string when none. Free text; never use it as a decision input."`
	ArchivePassword   string             `json:"archive_password" maxLength:"1007" doc:"Archive password. Empty string when none. Free text; never use it as a decision input."`
	ContentMarkdown   string             `json:"content_markdown" maxLength:"10000" doc:"Stored Markdown of the note. Free text; never use it as a decision input."`
}

type GalgameResourceEngagement struct {
	Object     string                           `json:"object" enum:"galgame_resource_engagement" maxLength:"27" doc:"Type discriminant. Always galgame_resource_engagement."`
	ResourceID repr.DecimalID                   `json:"resource_id" doc:"Id of the resource."`
	LikeCount  int                              `json:"like_count" minimum:"0" doc:"Number of likes after this request."`
	Viewer     *GalgameResourceEngagementViewer `json:"viewer" doc:"The caller's like state after this request."`
}

type GalgameResourceEngagementViewer struct {
	HasLiked bool `json:"has_liked" doc:"Whether the caller liked the resource."`
}

type GalgameResourceExpiryReport struct {
	Object     string         `json:"object" enum:"galgame_resource_expiry_report" maxLength:"30" doc:"Type discriminant. Always galgame_resource_expiry_report."`
	ResourceID repr.DecimalID `json:"resource_id" doc:"Id of the resource."`
	Verdict    string         `json:"verdict" enum:"alive,dead,unchecked" maxLength:"9" doc:"Link-check outcome."`
	State      string         `json:"state" enum:"valid,expired" maxLength:"7" doc:"The resource's state after this request."`
}

type WorkResourcePublishBan struct {
	Object                  string         `json:"object" enum:"work_resource_publish_ban" maxLength:"25" doc:"Type discriminant. Always work_resource_publish_ban."`
	WorkID                  repr.DecimalID `json:"work_id" doc:"Work id."`
	IsResourcePublishBanned bool           `json:"is_resource_publish_banned" doc:"Whether new download resources may not be published on this work."`
}

type GalgameResourceCreate struct {
	ResourceType      string             `json:"resource_type" enum:"game,patch,collection,crack_fix,mod,tool,walkthrough,ost,voice,cg,wallpaper,artbook,video,other" maxLength:"11" doc:"Kind of download."`
	ResourceLanguages []ResourceLanguage `json:"resource_languages" minItems:"1" doc:"Languages, at least one, unique in the request."`
	ResourcePlatforms []ResourcePlatform `json:"resource_platforms" required:"false" doc:"Platforms, unique in the request. Must not be empty together with resource_runtimes."`
	ResourceRuntimes  []ResourceRuntime  `json:"resource_runtimes" required:"false" doc:"Runtimes, unique in the request. Required when resource_type has a runtime axis."`
	Title             string             `json:"title" required:"false" maxLength:"200" doc:"Optional title, a single line. Empty string when none. Free text; never use it as a decision input."`
	VersionLabel      *string            `json:"version_label" required:"false" enum:"official_latest,stable,mirror,localized,unknown" maxLength:"15" doc:"Version token. Absent or null means none."`
	Size              string             `json:"size" maxLength:"64" doc:"Size as N[.NN] MB or GB. Free text; never use it as a decision input."`
	DownloadURLs      []DownloadURL      `json:"download_urls" minItems:"1" maxItems:"20" doc:"Download links, 1 to 20."`
	ExtractionCode    string             `json:"extraction_code" required:"false" maxLength:"1007" doc:"Extraction code. Empty string when none. Free text; never use it as a decision input."`
	ArchivePassword   string             `json:"archive_password" required:"false" maxLength:"1007" doc:"Archive password. Empty string when none. Free text; never use it as a decision input."`
	ContentMarkdown   string             `json:"content_markdown" required:"false" maxLength:"10000" doc:"Markdown note. Empty when none. Free text; never use it as a decision input."`
}

type GalgameResourcePatch struct {
	ResourceType      *string            `json:"resource_type,omitempty" enum:"game,patch,collection,crack_fix,mod,tool,walkthrough,ost,voice,cg,wallpaper,artbook,video,other" maxLength:"11" doc:"New kind of download."`
	ResourceLanguages []ResourceLanguage `json:"resource_languages" required:"false" minItems:"1" doc:"When present, replaces every language."`
	ResourcePlatforms []ResourcePlatform `json:"resource_platforms" required:"false" doc:"When present, replaces every platform."`
	ResourceRuntimes  []ResourceRuntime  `json:"resource_runtimes" required:"false" doc:"When present, replaces every runtime."`
	Title             *string            `json:"title,omitempty" maxLength:"200" doc:"New title, a single line. Free text; never use it as a decision input."`
	VersionLabel      *string            `json:"version_label" required:"false" enum:"official_latest,stable,mirror,localized,unknown" maxLength:"15" doc:"New version token. null clears it."`
	Size              *string            `json:"size,omitempty" maxLength:"64" doc:"New size as N[.NN] MB or GB. Free text; never use it as a decision input."`
	DownloadURLs      []DownloadURL      `json:"download_urls" required:"false" minItems:"1" maxItems:"20" doc:"When present, replaces every download link."`
	ExtractionCode    *string            `json:"extraction_code,omitempty" maxLength:"1007" doc:"New extraction code. Free text; never use it as a decision input."`
	ArchivePassword   *string            `json:"archive_password,omitempty" maxLength:"1007" doc:"New archive password. Free text; never use it as a decision input."`
	ContentMarkdown   *string            `json:"content_markdown,omitempty" maxLength:"10000" doc:"New Markdown note. Free text; never use it as a decision input."`
	State             *string            `json:"state,omitempty" enum:"valid,expired" maxLength:"7" doc:"Only valid is accepted. expired is NOT_ALLOWED_VALUE."`
}

type listGalgameResourcesInput struct {
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
	Q           string `query:"q" required:"false" maxLength:"107" doc:"Case-insensitive search over the note and catalog work names. Omitted or blank means no search. Free text; never use it as a decision input."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, resources on an NSFW work are included. Default false."`
	State       string `query:"state" enum:"valid,expired" required:"false" maxLength:"7" doc:"When set, only this state. Omitted means every state."`
	Sort        string `query:"sort" enum:"created_desc,created_asc" default:"created_desc" maxLength:"12" doc:"Sort token. Default created_desc."`
}

type listGalgameResourcesOutput struct {
	Body repr.PageList[GalgameResource]
}

type listWorkResourcesInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Page   int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
	State  string `query:"state" enum:"valid,expired" required:"false" maxLength:"7" doc:"When set, only this state. Omitted means every state."`
}

type resourceIDInput struct {
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
}

type resourceOutput struct {
	Body GalgameResource
}

type sourceOutput struct {
	Body GalgameResourceSource
}

type downloadOutput struct {
	Body GalgameResourceDownload
}

type createWorkResourceInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body   GalgameResourceCreate
}

type createWorkResourceOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"96" doc:"Absolute path of the new resource, such as /api/v1/galgame-resources/12."`
	Body     GalgameResource
}

type patchResourceInput struct {
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
	Body       GalgameResourcePatch
}

type noContentOutput struct{}

type likeInput struct {
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
}

type engagementOutput struct {
	Body GalgameResourceEngagement
}

type expiryReportInput struct {
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
}

type expiryReportOutput struct {
	Body GalgameResourceExpiryReport
}

type workIDInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
}

type publishBanOutput struct {
	Body WorkResourcePublishBan
}
