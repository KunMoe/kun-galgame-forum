package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
)

const defaultLimit = 24

type ToolsetSummary struct {
	Object                    string         `json:"object" enum:"toolset" maxLength:"7" doc:"Type discriminant. Always toolset."`
	ID                        repr.DecimalID `json:"id" doc:"Toolset id."`
	Name                      string         `json:"name" maxLength:"500" doc:"Display name. Free text; never use it as a decision input."`
	Aliases                   []string       `json:"aliases" maxItems:"17" doc:"Alternate names. Empty array, never null."`
	Type                      string         `json:"type" enum:"emulator,translator,extractor,converter,debug,launcher,script,docs,others" maxLength:"10" doc:"Kind of tool."`
	Language                  string         `json:"interface_language" enum:"zh-cn,zh-tw,ja-jp,en-us,others" maxLength:"7" doc:"The tool's interface language. Lower-case BCP 47, plus others."`
	Platform                  string         `json:"platform" enum:"windows,mac,linux,emulator,others" maxLength:"8" doc:"Platform the tool runs on."`
	Version                   string         `json:"version" enum:"stable,beta,alpha,rc" maxLength:"6" doc:"Release channel."`
	HomepageURLs              []string       `json:"homepage_urls" maxItems:"10" doc:"http or https homepage URLs. Empty array, never null."`
	Author                    repr.UserRef   `json:"author" doc:"Author of the toolset."`
	ViewCount                 int            `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	DownloadCount             int            `json:"download_count" minimum:"0" doc:"Sum of download counts of every resource. 0 when there are none."`
	CommentCount              int            `json:"comment_count" minimum:"0" doc:"Comments on the toolset's wall."`
	PracticalityAverage       *float64       `json:"practicality_average" doc:"Mean rating, two decimal places. null when nobody has rated."`
	PracticalityCount         int            `json:"practicality_count" minimum:"0" doc:"Number of ratings."`
	PracticalityDistribution  []int          `json:"practicality_distribution" minItems:"5" maxItems:"5" doc:"Counts per star. Index 0 is 1 star. Length 5, never null."`
	CreatedAt                 repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt                 repr.DateTime  `json:"updated_at" doc:"Time of the latest write to the row."`
	EditedAt                  *repr.DateTime `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	ResourceUpdatedAt         repr.DateTime  `json:"resource_updated_at" doc:"Time a resource of this toolset was last added."`
}

type Toolset struct {
	Object                   string                   `json:"object" enum:"toolset" maxLength:"7" doc:"Type discriminant. Always toolset."`
	ID                       repr.DecimalID           `json:"id" doc:"Toolset id."`
	Name                     string                   `json:"name" maxLength:"500" doc:"Display name. Free text; never use it as a decision input."`
	Aliases                  []string                 `json:"aliases" maxItems:"17" doc:"Alternate names. Empty array, never null."`
	Type                     string                   `json:"type" enum:"emulator,translator,extractor,converter,debug,launcher,script,docs,others" maxLength:"10" doc:"Kind of tool."`
	Language                 string                   `json:"interface_language" enum:"zh-cn,zh-tw,ja-jp,en-us,others" maxLength:"7" doc:"The tool's interface language. Lower-case BCP 47, plus others."`
	Platform                 string                   `json:"platform" enum:"windows,mac,linux,emulator,others" maxLength:"8" doc:"Platform the tool runs on."`
	Version                  string                   `json:"version" enum:"stable,beta,alpha,rc" maxLength:"6" doc:"Release channel."`
	HomepageURLs             []string                 `json:"homepage_urls" maxItems:"10" doc:"http or https homepage URLs. Empty array, never null."`
	Author                   repr.UserRef             `json:"author" doc:"Author of the toolset."`
	ViewCount                int                      `json:"view_count" minimum:"0" doc:"Lifetime view count. Each read of this operation adds one."`
	DownloadCount            int                      `json:"download_count" minimum:"0" doc:"Sum of download counts of every resource. 0 when there are none."`
	CommentCount             int                      `json:"comment_count" minimum:"0" doc:"Comments on the toolset's wall."`
	PracticalityAverage      *float64                 `json:"practicality_average" doc:"Mean rating, two decimal places. null when nobody has rated."`
	PracticalityCount        int                      `json:"practicality_count" minimum:"0" doc:"Number of ratings."`
	PracticalityDistribution []int                    `json:"practicality_distribution" minItems:"5" maxItems:"5" doc:"Counts per star. Index 0 is 1 star. Length 5, never null."`
	CreatedAt                repr.DateTime            `json:"created_at" doc:"Creation time."`
	UpdatedAt                repr.DateTime            `json:"updated_at" doc:"Time of the latest write to the row."`
	EditedAt                 *repr.DateTime           `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	ResourceUpdatedAt        repr.DateTime            `json:"resource_updated_at" doc:"Time a resource of this toolset was last added."`
	Content                  content.ContentDocument  `json:"content" doc:"Description as a node tree. An empty document when there is no description."`
	Contributors             []repr.UserRef           `json:"contributors" doc:"Users who have contributed a resource. Unrenderable accounts are omitted. Empty array, never null."`
	Resources                []ToolsetResourceSummary `json:"resources" doc:"Resources of the toolset, newest first. Empty array, never null."`
	Viewer                   *ToolsetViewer           `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type ToolsetViewer struct {
	CanEdit            bool `json:"can_edit" doc:"Whether the caller may edit this toolset. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete          bool `json:"can_delete" doc:"Whether the caller may delete this toolset. Requests authenticated with a Bearer token never carry staff powers."`
	PracticalityRating *int `json:"practicality_rating" minimum:"1" maximum:"5" doc:"The caller's rating, 1–5. null when they have not rated."`
}

type ToolsetSource struct {
	Object          string         `json:"object" enum:"toolset_source" maxLength:"14" doc:"Type discriminant. Always toolset_source."`
	ToolsetID       repr.DecimalID `json:"toolset_id" doc:"Id of the toolset."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"2000" doc:"Stored Markdown of the description. Free text; never use it as a decision input."`
}

type ToolsetResourceSummary struct {
	Object        string         `json:"object" enum:"toolset_resource" maxLength:"16" doc:"Type discriminant. Always toolset_resource."`
	ID            repr.DecimalID `json:"id" doc:"Resource id."`
	ResourceType  string         `json:"resource_type" enum:"file,link" maxLength:"4" doc:"file is a hosted archive; link is an external URL."`
	FileSize      *int64         `json:"file_size" minimum:"0" doc:"Size in bytes of a file resource. null for a link."`
	SizeLabel     *string        `json:"size_label" maxLength:"107" doc:"Free-text size of a link resource. null for a file. Free text; never use it as a decision input."`
	Note          string         `json:"note" maxLength:"1007" doc:"Note shown with the resource. Empty string when none. Free text; never use it as a decision input."`
	DownloadCount int            `json:"download_count" minimum:"0" doc:"Times this resource's download secrets were issued."`
	Poster        repr.UserRef   `json:"poster" doc:"User who added the resource."`
	CreatedAt     repr.DateTime  `json:"created_at" doc:"Creation time."`
	Viewer        *ResourceViewer `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type ResourceViewer struct {
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may edit this resource. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete this resource. Requests authenticated with a Bearer token never carry staff powers."`
}

type ToolsetDownload struct {
	Object           string         `json:"object" enum:"toolset_download" maxLength:"16" doc:"Type discriminant. Always toolset_download."`
	URL              string         `json:"url" maxLength:"4096" doc:"Download URL. A link resource returns the stored URL; a file resource returns a presigned URL."`
	ExpiresAt        *repr.DateTime `json:"expires_at" doc:"When the presigned URL expires. null for a link."`
	ExtractionCode   string         `json:"extraction_code" maxLength:"1007" doc:"Extraction code. Empty string when none."`
	ArchivePassword  string         `json:"archive_password" maxLength:"1007" doc:"Archive password. Empty string when none."`
}

type ToolsetUpload struct {
	Object        string          `json:"object" enum:"toolset_upload" maxLength:"14" doc:"Type discriminant. Always toolset_upload."`
	ID            string          `json:"id" format:"uuid" maxLength:"36" doc:"Upload id, equal to the artifact UUID."`
	ToolsetID     repr.DecimalID  `json:"toolset_id" doc:"Toolset this upload belongs to."`
	Filename      string          `json:"filename" maxLength:"1007" doc:"Original filename."`
	FileSize      int64           `json:"file_size" minimum:"1" doc:"Declared size in bytes."`
	State         string          `json:"state" enum:"pending,completed" maxLength:"9" doc:"pending until complete; completed afterwards."`
	Multipart     bool            `json:"multipart" doc:"Whether the upload is split into parts."`
	UploadURL     *string         `json:"upload_url" format:"uri" maxLength:"4096" doc:"Single-shot upload URL. null when multipart or completed."`
	PartSize      *int64          `json:"part_size" minimum:"1" doc:"Part size in bytes when multipart. null otherwise."`
	Parts         []UploadPart    `json:"parts" doc:"Presigned part URLs. Empty array, never null."`
	UploadedParts []UploadedPart  `json:"uploaded_parts" doc:"Parts already uploaded, on a pending multipart resume. Empty array otherwise."`
	ExpiresAt     *repr.DateTime  `json:"expires_at" doc:"When the upload URLs expire. null when completed."`
	CreatedAt     repr.DateTime   `json:"created_at" doc:"Creation time."`
	CompletedAt   *repr.DateTime  `json:"completed_at" doc:"Time the upload completed. null while pending."`
}

type UploadPart struct {
	PartNumber int    `json:"part_number" minimum:"1" doc:"1-based part number."`
	URL        string `json:"url" format:"uri" maxLength:"4096" doc:"Presigned URL for this part."`
}

type UploadedPart struct {
	PartNumber int    `json:"part_number" minimum:"1" doc:"1-based part number."`
	Etag       string `json:"etag" maxLength:"256" doc:"ETag returned by the object store."`
	Size       int64  `json:"size" minimum:"0" doc:"Bytes uploaded in this part."`
}

type ToolsetPracticality struct {
	Object                   string                   `json:"object" enum:"toolset_practicality" maxLength:"20" doc:"Type discriminant. Always toolset_practicality."`
	ToolsetID                repr.DecimalID           `json:"toolset_id" doc:"Id of the toolset."`
	PracticalityAverage      *float64                 `json:"practicality_average" doc:"Mean rating, two decimal places. null when nobody has rated."`
	PracticalityCount        int                      `json:"practicality_count" minimum:"0" doc:"Number of ratings."`
	PracticalityDistribution []int                    `json:"practicality_distribution" minItems:"5" maxItems:"5" doc:"Counts per star. Index 0 is 1 star. Length 5, never null."`
	Viewer                   *PracticalityViewer      `json:"viewer" doc:"The caller's rating after this request."`
}

type PracticalityViewer struct {
	PracticalityRating int `json:"practicality_rating" minimum:"1" maximum:"5" doc:"The rating just written."`
}

type ToolsetCreate struct {
	Name             string   `json:"name" minLength:"1" maxLength:"500" doc:"Display name. Length is checked on the raw value; only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
	ContentMarkdown  string   `json:"content_markdown" required:"false" maxLength:"2000" doc:"Markdown description. May be empty. Free text; never use it as a decision input."`
	Type             string   `json:"type" enum:"emulator,translator,extractor,converter,debug,launcher,script,docs,others" maxLength:"10" doc:"Kind of tool."`
	Language         string   `json:"interface_language" enum:"zh-cn,zh-tw,ja-jp,en-us,others" maxLength:"7" doc:"The tool's interface language."`
	Platform         string   `json:"platform" enum:"windows,mac,linux,emulator,others" maxLength:"8" doc:"Platform the tool runs on."`
	Version          string   `json:"version" enum:"stable,beta,alpha,rc" maxLength:"6" doc:"Release channel."`
	Aliases          []string `json:"aliases" required:"false" maxItems:"17" doc:"Alternate names, at most 17, each 1–500 after trimming, unique in the request."`
	HomepageURLs     []string `json:"homepage_urls" required:"false" maxItems:"10" doc:"http or https URLs, at most 10, each at most 500 characters."`
}

type ToolsetPatch struct {
	Name            *string  `json:"name,omitempty" minLength:"1" maxLength:"500" doc:"New display name. Free text; never use it as a decision input."`
	ContentMarkdown *string  `json:"content_markdown,omitempty" maxLength:"2000" doc:"New Markdown description. Free text; never use it as a decision input."`
	Type            *string  `json:"type,omitempty" enum:"emulator,translator,extractor,converter,debug,launcher,script,docs,others" maxLength:"10" doc:"New kind of tool."`
	Language        *string  `json:"interface_language,omitempty" enum:"zh-cn,zh-tw,ja-jp,en-us,others" maxLength:"7" doc:"New interface language."`
	Platform        *string  `json:"platform,omitempty" enum:"windows,mac,linux,emulator,others" maxLength:"8" doc:"New platform."`
	Version         *string  `json:"version,omitempty" enum:"stable,beta,alpha,rc" maxLength:"6" doc:"New release channel."`
	Aliases         []string `json:"aliases" required:"false" maxItems:"17" doc:"When present, replaces every alias."`
	HomepageURLs    []string `json:"homepage_urls" required:"false" maxItems:"10" doc:"When present, replaces every homepage URL."`
}

type ToolsetResourceCreate struct {
	ResourceType     string  `json:"resource_type" enum:"file,link" maxLength:"4" doc:"file needs artifact_id; link needs url and size_label."`
	ArtifactID       *string `json:"artifact_id,omitempty" format:"uuid" maxLength:"36" doc:"Completed upload of the caller on this toolset. Required for file; inconsistent on link."`
	URL              *string `json:"url,omitempty" maxLength:"1007" doc:"External download URL. Required for link; inconsistent on file."`
	SizeLabel        *string `json:"size_label,omitempty" maxLength:"107" doc:"Free-text size. Required for link; inconsistent on file."`
	ExtractionCode   *string `json:"extraction_code,omitempty" maxLength:"1007" doc:"Extraction code."`
	ArchivePassword  *string `json:"archive_password,omitempty" maxLength:"1007" doc:"Archive password."`
	Note             *string `json:"note,omitempty" maxLength:"1007" doc:"Note. Free text; never use it as a decision input."`
}

type ToolsetResourcePatch struct {
	URL             *string `json:"url,omitempty" maxLength:"1007" doc:"New URL. Immutable on a file resource."`
	SizeLabel       *string `json:"size_label,omitempty" maxLength:"107" doc:"New size label. Immutable on a file resource."`
	ExtractionCode  *string `json:"extraction_code,omitempty" maxLength:"1007" doc:"New extraction code. Immutable on a file resource."`
	ArchivePassword *string `json:"archive_password,omitempty" maxLength:"1007" doc:"New archive password."`
	Note            *string `json:"note,omitempty" maxLength:"1007" doc:"New note. Free text; never use it as a decision input."`
}

type ToolsetUploadCreate struct {
	Filename    string  `json:"filename" minLength:"1" maxLength:"1007" doc:"Original filename. Must end in .7z, .zip, or .rar."`
	FileSize    int64   `json:"file_size" minimum:"1" maximum:"2147483648" doc:"Size in bytes, 1–2147483648."`
	ContentType *string `json:"content_type,omitempty" maxLength:"100" doc:"MIME type of the file."`
}

type ToolsetUploadPatch struct {
	State string             `json:"state" enum:"pending,completed" maxLength:"9" doc:"Must be completed. Any other value is INVALID_STATE_TRANSITION."`
	Parts []CompletePartBody `json:"parts" required:"false" doc:"Completed multipart parts."`
}

type CompletePartBody struct {
	PartNumber int    `json:"part_number" minimum:"1" doc:"1-based part number."`
	Etag       string `json:"etag" minLength:"1" maxLength:"256" doc:"ETag returned by the object store."`
}

type PracticalityPut struct {
	Rating int `json:"rating" minimum:"1" maximum:"5" doc:"Star rating, 1–5."`
}

type listToolsetsInput struct {
	Page     int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit    int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	Type     string `query:"type" enum:"emulator,translator,extractor,converter,debug,launcher,script,docs,others" required:"false" maxLength:"10" doc:"When set, only this type. Omitted means every type."`
	Language string `query:"interface_language" enum:"zh-cn,zh-tw,ja-jp,en-us,others" required:"false" maxLength:"7" doc:"When set, only tools with this interface language. Omitted means every language."`
	Platform string `query:"platform" enum:"windows,mac,linux,emulator,others" required:"false" maxLength:"8" doc:"When set, only this platform. Omitted means every platform."`
	Version  string `query:"version" enum:"stable,beta,alpha,rc" required:"false" maxLength:"6" doc:"When set, only this channel. Omitted means every channel."`
	Sort     string `query:"sort" enum:"resource_updated_desc,resource_updated_asc,created_desc,created_asc,view_desc,view_asc,name_asc,name_desc" default:"resource_updated_desc" doc:"Sort token. Default resource_updated_desc."`
	Q        string `query:"q" required:"false" maxLength:"100" doc:"Case-insensitive search over the toolset name. Omitted or blank means no search."`
}

type listToolsetsOutput struct {
	Body repr.PageList[ToolsetSummary]
}

type listUserToolsetsInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Page   int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type toolsetIDInput struct {
	ToolsetID string `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
}

type toolsetOutput struct {
	Body Toolset
}

type sourceOutput struct {
	Body ToolsetSource
}

type createToolsetInput struct {
	Body ToolsetCreate
}

type createToolsetOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new toolset, such as /api/v1/toolsets/12."`
	Body     Toolset
}

type patchToolsetInput struct {
	ToolsetID string       `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	Body      ToolsetPatch
}

type resourceIDInput struct {
	ToolsetID  string `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
}

type resourceOutput struct {
	Body ToolsetResourceSummary
}

type createResourceInput struct {
	ToolsetID string                `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	Body      ToolsetResourceCreate
}

type createResourceOutput struct {
	Location string                  `header:"Location" format:"uri-reference" maxLength:"96" doc:"Absolute path of the new resource, such as /api/v1/toolsets/12/resources/4."`
	Body     ToolsetResourceSummary
}

type patchResourceInput struct {
	ToolsetID  string               `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	ResourceID string               `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Resource id."`
	Body       ToolsetResourcePatch
}

type downloadOutput struct {
	Body ToolsetDownload
}

type createUploadInput struct {
	ToolsetID string              `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	Body      ToolsetUploadCreate
}

type createUploadOutput struct {
	Location string        `header:"Location" format:"uri-reference" maxLength:"96" doc:"Absolute path of the new upload, such as /api/v1/toolsets/12/uploads/550e8400-e29b-41d4-a716-446655440000."`
	Body     ToolsetUpload
}

type uploadIDInput struct {
	ToolsetID string `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	UploadID  string `path:"upload_id" format:"uuid" maxLength:"36" doc:"Upload id, equal to the artifact UUID."`
}

type uploadOutput struct {
	Body ToolsetUpload
}

type patchUploadInput struct {
	ToolsetID string             `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	UploadID  string             `path:"upload_id" format:"uuid" maxLength:"36" doc:"Upload id, equal to the artifact UUID."`
	Body      ToolsetUploadPatch
}

type putPracticalityInput struct {
	ToolsetID string          `path:"toolset_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Toolset id."`
	Body      PracticalityPut
}

type practicalityOutput struct {
	Body ToolsetPracticality
}

type noContentOutput struct{}
