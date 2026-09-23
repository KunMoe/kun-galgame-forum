package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
)

type WallComment struct {
	Object              string                  `json:"object" enum:"wall_comment" maxLength:"12" doc:"Type discriminant. Always wall_comment."`
	ID                  repr.DecimalID          `json:"id" doc:"Wall comment id. JSON string of a decimal integer. Its own id space, unrelated to topic comments."`
	SubjectType         SubjectType             `json:"subject_type" doc:"Kind of page whose wall the comment is on."`
	SubjectID           repr.DecimalID          `json:"subject_id" doc:"Id of that page: a galgame, rating, resource, quiz, toolset or website id, by subject_type."`
	ParentCommentID     *repr.DecimalID         `json:"parent_comment_id" doc:"Id of the wall comment this one answers. null for a top-level comment. The parent may be absent from a list."`
	RootCommentID       *repr.DecimalID         `json:"root_comment_id" doc:"Id of the top-level comment this one's reply chain starts from. null for a top-level comment. Clients that draw two levels group by it."`
	Author              repr.UserRef            `json:"author" doc:"Comment author."`
	Addressee           *repr.UserRef           `json:"addressee" doc:"The user the comment is addressed to: the parent comment's author for a reply, the rating's author for a top-level comment on a rating wall, null otherwise."`
	State               string                  `json:"state" enum:"visible,held,deleted" maxLength:"7" doc:"visible: shown to everyone. held: waiting for review, shown only to its author. deleted: a tombstone kept so replies keep their parent; content is an empty document."`
	Content             content.ContentDocument `json:"content" doc:"Comment body as a node tree. An empty document when state is deleted."`
	LikeCount           int                     `json:"like_count" minimum:"0" doc:"Like count."`
	CreatedAt           repr.DateTime           `json:"created_at" doc:"Creation time."`
	EditedAt            *repr.DateTime          `json:"edited_at" doc:"Time of the latest edit. null when never edited."`
	IsEditedByModerator bool                    `json:"is_edited_by_moderator" doc:"Whether the latest edit was made by staff rather than the author."`
	Viewer              *WallCommentViewer      `json:"viewer" doc:"The caller's own state on this comment. null for an anonymous caller."`
}

type WallCommentViewer struct {
	HasLiked  bool `json:"has_liked" doc:"Whether the caller liked the comment."`
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may edit the comment: its author, or staff holding this wall's edit permission. Requests authenticated with a Bearer token never carry staff powers. Always false on a tombstone."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete the comment: its author, staff holding this wall's delete permission, or the owner of the page the wall belongs to (a resource's publisher, a toolset's owner, a quiz's author, a rating's author). Always false on a tombstone."`
	CanLike   bool `json:"can_like" doc:"Whether the caller may like the comment: anyone but its author, unless it is a tombstone."`
	CanFlag   bool `json:"can_flag" doc:"Whether the caller may flag the comment: anyone but its author, unless it is a tombstone."`
}

type WallCommentSource struct {
	Object          string         `json:"object" enum:"wall_comment_source" maxLength:"19" doc:"Type discriminant. Always wall_comment_source."`
	WallCommentID   repr.DecimalID `json:"wall_comment_id" doc:"Id of the wall comment."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"10000" doc:"Comment body as the stored Markdown source. Free text; never use it as a decision input."`
}

type WallCommentCreate struct {
	SubjectType     SubjectType     `json:"subject_type" doc:"Kind of page whose wall to comment on."`
	SubjectID       repr.DecimalID  `json:"subject_id" doc:"Id of that page."`
	ContentMarkdown string          `json:"content_markdown" minLength:"1" maxLength:"5000" doc:"Comment body as Markdown source. The wall sets the real limit: 5000 characters on a galgame, 1314 on a rating, 1007 elsewhere, counted on the value as sent. A body of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	ParentCommentID *repr.DecimalID `json:"parent_comment_id" required:"false" doc:"Id of a comment on the same wall that this one answers. Absent or null for a top-level comment."`
}

type WallCommentPatch struct {
	ContentMarkdown string `json:"content_markdown" minLength:"1" maxLength:"5000" doc:"New body as Markdown source, checked as in createWallComment. Free text; never use it as a decision input."`
}

type WallCommentFlag struct {
	FlagReason string  `json:"flag_reason" enum:"spam,abuse,off_topic,other,nsfw_mislabel" maxLength:"13" doc:"Why the comment is flagged."`
	Note       *string `json:"note" required:"false" maxLength:"500" doc:"Free-text note for the moderators. Absent or null for none. Free text; never use it as a decision input."`
}

type listInput struct {
	collect.Page
	SubjectType SubjectType `query:"subject_type" required:"true" doc:"Kind of page whose wall to list."`
	SubjectID   string      `query:"subject_id" required:"true" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Id of that page."`
}

type listOutput struct {
	Body repr.List[WallComment]
}

type createInput struct {
	Body WallCommentCreate
}

type createOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new comment, such as /api/v1/wall-comments/10523."`
	Body     WallComment
}

type commentInput struct {
	WallCommentID string `path:"wall_comment_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Wall comment id."`
}

type updateInput struct {
	WallCommentID string `path:"wall_comment_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Wall comment id."`
	Body          WallCommentPatch
}

type flagInput struct {
	WallCommentID string `path:"wall_comment_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Wall comment id."`
	Body          WallCommentFlag
}

type commentOutput struct {
	Body WallComment
}

type sourceOutput struct {
	Body WallCommentSource
}
