package apiv1

import "kun-galgame-api/internal/apiv1/repr"

type TopicDraft struct {
	Object           string         `json:"object" enum:"topic_draft" maxLength:"11" doc:"Type discriminant. Always topic_draft."`
	ID               repr.DecimalID `json:"id" doc:"Draft id. JSON string of a decimal integer."`
	Title            string         `json:"title" maxLength:"233" doc:"Draft title as stored, empty string when the author has not written one. Free text; never use it as a decision input."`
	ContentMarkdown  string         `json:"content_markdown" maxLength:"100007" doc:"Draft body as Markdown source, stored as sent. Empty string when the author has not written one. Free text; never use it as a decision input."`
	Category         *string        `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Chosen category. null when the author has not chosen one, which is the stored state of most drafts."`
	Sections         []SectionSlug  `json:"sections" maxItems:"3" doc:"Chosen section slugs, in stored order. Empty array if none. A draft is unfinished, so they are not checked against category."`
	IsNSFW           bool           `json:"is_nsfw" doc:"Whether the author marked the draft NSFW."`
	CoverImageHashes []ImageHash    `json:"cover_image_hashes" maxItems:"9" doc:"Cover images by image-service hash, in stored order. Tokens that do not parse are skipped. Empty array if none."`
	CreatedAt        repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt        repr.DateTime  `json:"updated_at" doc:"Last write time. A draft is never rewritten, so it equals created_at for every draft written by this API."`
}

type TopicDraftSummary struct {
	Object    string         `json:"object" enum:"topic_draft_summary" maxLength:"19" doc:"Type discriminant. Always topic_draft_summary."`
	ID        repr.DecimalID `json:"id" doc:"Draft id. JSON string of a decimal integer."`
	Title     string         `json:"title" maxLength:"233" doc:"Draft title as stored, empty string when the author has not written one. Free text; never use it as a decision input."`
	Summary   string         `json:"summary" maxLength:"120" doc:"First 120 characters of the body as stored: raw Markdown, image tokens included, cut without regard for word or token boundaries. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt repr.DateTime  `json:"updated_at" doc:"Last write time."`
}

type TopicDraftCreate struct {
	Title            string        `json:"title" required:"false" maxLength:"233" doc:"Draft title. Stored as sent. Free text; never use it as a decision input."`
	ContentMarkdown  string        `json:"content_markdown" required:"false" maxLength:"100007" doc:"Draft body as Markdown source. Stored as sent. Free text; never use it as a decision input."`
	Category         *string       `json:"category,omitempty" enum:"galgame,technique,others" maxLength:"9" doc:"Chosen category. Absent or null when the author has not chosen one."`
	Sections         []SectionSlug `json:"sections" required:"false" maxItems:"3" uniqueItems:"true" doc:"Chosen section slugs. Unlike createTopic they are not checked against category, and an empty list is allowed: a draft is unfinished by definition."`
	IsNSFW           bool          `json:"is_nsfw" required:"false" doc:"Whether to mark the draft NSFW. Defaults to false."`
	CoverImageHashes []ImageHash   `json:"cover_image_hashes" required:"false" maxItems:"9" uniqueItems:"true" doc:"Cover images by image-service hash, in display order. Unlike createTopic, nothing is derived from the body."`
}
