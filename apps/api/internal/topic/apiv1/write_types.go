package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type AccessRole string

func (AccessRole) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"creator", "moderator", "admin", "ren"},
		MaxLength:   &n,
		Description: "Role that may read a topic whose access_scope is role.",
	}
}

type ImageHash string

func (ImageHash) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     "^[0-9a-f]{64}$",
		MinLength:   &n,
		MaxLength:   &n,
		Description: "Image-service content hash of an uploaded image.",
	}
}

type TopicCreate struct {
	Title            string            `json:"title" minLength:"1" maxLength:"233" doc:"Topic title. Leading and trailing whitespace is removed before it is stored, and a title of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	ContentMarkdown  string            `json:"content_markdown" minLength:"1" maxLength:"100007" doc:"Topic body as Markdown source, stored as sent. A body of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	Category         string            `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Topic category."`
	Sections         []SectionSlug     `json:"sections" minItems:"1" maxItems:"3" uniqueItems:"true" doc:"Section slugs. Each must belong to category: g- slugs to galgame, t- slugs to technique, o- slugs to others."`
	IsNSFW           bool              `json:"is_nsfw" doc:"Whether the topic is NSFW."`
	CoverImageHashes *[]ImageHash      `json:"cover_image_hashes,omitempty" maxItems:"9" uniqueItems:"true" doc:"Cover images by image-service hash, in display order. When absent, the covers are the first nine distinct /image/{hash} tokens of the body in body order, stickers skipped; an empty array means no covers."`
	AccessScope      string            `json:"access_scope" enum:"public,login,role,users" maxLength:"6" doc:"Who may read the topic: everyone, signed-in users, holders of granted roles, or granted users. The author and staff always may."`
	AccessRoles      *[]AccessRole     `json:"access_roles,omitempty" minItems:"1" maxItems:"4" uniqueItems:"true" doc:"Roles granted to read the topic. Required when access_scope is role; must be absent otherwise."`
	AccessUserIDs    *[]repr.DecimalID `json:"access_user_ids,omitempty" minItems:"1" maxItems:"50" uniqueItems:"true" doc:"Users granted to read the topic. Required when access_scope is users; must be absent otherwise. The author always reads their own topic and is dropped from the list, so a list of only the author leaves the topic readable by its author and staff alone."`
}

type TopicPatch struct {
	Title            *string           `json:"title,omitempty" minLength:"1" maxLength:"233" doc:"New title. Trimmed and checked as in createTopic. Free text; never use it as a decision input."`
	ContentMarkdown  *string           `json:"content_markdown,omitempty" minLength:"1" maxLength:"100007" doc:"New body as Markdown source. Checked as in createTopic. Free text; never use it as a decision input."`
	Category         *string           `json:"category,omitempty" enum:"galgame,technique,others" maxLength:"9" doc:"New category. The resulting sections must belong to the resulting category."`
	Sections         *[]SectionSlug    `json:"sections,omitempty" minItems:"1" maxItems:"3" uniqueItems:"true" doc:"New section slugs, replacing the stored ones."`
	IsNSFW           *bool             `json:"is_nsfw,omitempty" doc:"New NSFW flag."`
	CoverImageHashes *[]ImageHash      `json:"cover_image_hashes,omitempty" maxItems:"9" uniqueItems:"true" doc:"New cover images, replacing the stored ones. An empty array removes every cover. Unlike createTopic, nothing is derived from the body."`
	AccessScope      *string           `json:"access_scope,omitempty" enum:"public,login,role,users" maxLength:"6" doc:"New access scope. The resulting scope and grants are checked together as in createTopic. When the scope changes, the stored grants are dropped and access_roles or access_user_ids supplies the new ones."`
	AccessRoles      *[]AccessRole     `json:"access_roles,omitempty" minItems:"1" maxItems:"4" uniqueItems:"true" doc:"New granted roles, replacing the stored ones. Only with a resulting access_scope of role."`
	AccessUserIDs    *[]repr.DecimalID `json:"access_user_ids,omitempty" minItems:"1" maxItems:"50" uniqueItems:"true" doc:"New granted users, replacing the stored ones. Only with a resulting access_scope of users. The author is dropped as in createTopic."`
	State            *string           `json:"state,omitempty" enum:"published,hidden" maxLength:"9" doc:"hidden hides the topic; published shows it again. Hiding needs can_hide and showing needs can_unhide. Sending the current state changes nothing."`
}

type AccessGrants struct {
	Roles []AccessRole   `json:"roles" maxItems:"4" doc:"Granted roles when access_scope is role. Empty array otherwise."`
	Users []repr.UserRef `json:"users" maxItems:"50" doc:"Granted users when access_scope is users, in grant order. The author is never listed. Banned and deleted users keep their entry with name null. Empty array otherwise."`
}

type TopicSource struct {
	Object          string         `json:"object" enum:"topic_source" maxLength:"12" doc:"Type discriminant. Always topic_source."`
	TopicID         repr.DecimalID `json:"topic_id" doc:"Id of the topic."`
	Title           string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"100007" doc:"Topic body as the stored Markdown source. Free text; never use it as a decision input."`
	Category        string         `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Topic category."`
	Sections        []SectionSlug  `json:"sections" maxItems:"3" doc:"Section slugs, in stored order. Empty array if none. Hyphenated URL segments of /section/{key}."`
	IsNSFW          bool           `json:"is_nsfw" doc:"Whether the topic is NSFW."`
	CoverImages     []repr.Image   `json:"cover_images" maxItems:"9" doc:"Cover images in stored token order. Tokens that do not parse are skipped, and so are stickers: a sticker is never a cover. Empty array if none."`
	AccessScope     string         `json:"access_scope" enum:"public,login,role,users" maxLength:"6" doc:"Who may read the topic: everyone, signed-in users, holders of granted roles, or granted users. The author and staff always may."`
	AccessGrants    AccessGrants   `json:"access_grants" doc:"The stored grants."`
}

type ReplyCreate struct {
	ContentMarkdown string `json:"content_markdown" minLength:"1" maxLength:"10007" doc:"Reply body as Markdown source, stored as sent. A body of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
}

type ReplyPatch struct {
	ContentMarkdown *string `json:"content_markdown,omitempty" minLength:"1" maxLength:"10007" doc:"New body as Markdown source. Checked as in createReply. Free text; never use it as a decision input."`
}

type ReplySource struct {
	Object          string         `json:"object" enum:"reply_source" maxLength:"12" doc:"Type discriminant. Always reply_source."`
	ReplyID         repr.DecimalID `json:"reply_id" doc:"Id of the reply."`
	TopicID         repr.DecimalID `json:"topic_id" doc:"Id of the topic the reply belongs to."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"10007" doc:"Reply body as the stored Markdown source. Free text; never use it as a decision input."`
}
