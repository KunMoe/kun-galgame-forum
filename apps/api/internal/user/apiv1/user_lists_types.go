package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
)

type UserListPage struct {
	UserID      string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW topics are included. Default false."`
}

func (p UserListPage) number() collect.PageNumber {
	return collect.PageNumber{Page: p.Page, Limit: p.Limit}
}

type listUserTopicsInput struct {
	UserListPage
	Relation string `query:"relation" required:"true" enum:"authored,liked,upvoted,favorited,hidden" maxLength:"9" doc:"How the listed topics relate to the user: authored, liked, upvoted, favorited, or hidden."`
}

type listUserRepliesInput struct {
	UserListPage
	Relation string `query:"relation" required:"true" enum:"authored,received,liked" maxLength:"8" doc:"How the listed replies relate to the user: authored, received, or liked."`
}

type listUserCommentsInput struct {
	UserListPage
	Relation string `query:"relation" required:"true" enum:"authored,received,liked" maxLength:"8" doc:"How the listed comments relate to the user: authored, received, or liked."`
}

type listUserTopicsOutput struct {
	Body repr.PageList[UserTopicItem]
}

type listUserRepliesOutput struct {
	Body repr.PageList[UserReplyItem]
}

type listUserCommentsOutput struct {
	Body repr.PageList[UserCommentItem]
}

type UserTopicItem struct {
	Object    string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID        repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title     string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type UserReplyItem struct {
	Object    string         `json:"object" enum:"reply" maxLength:"5" doc:"Type discriminant. Always reply."`
	ID        repr.DecimalID `json:"id" doc:"Reply id. JSON string of a decimal integer."`
	TopicID   repr.DecimalID `json:"topic_id" doc:"Id of the topic the reply belongs to."`
	Floor     int            `json:"floor" minimum:"1" doc:"Floor number, assigned when the reply was created and never renumbered. Deleted and hidden replies leave gaps; a floor is not a position."`
	Excerpt   string         `json:"excerpt" maxLength:"800" doc:"Plain-text excerpt of the reply body, at most 200 characters. When longer, cut to 199 characters and terminated with an ellipsis. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type UserCommentItem struct {
	Object    string         `json:"object" enum:"comment" maxLength:"7" doc:"Type discriminant. Always comment."`
	ID        repr.DecimalID `json:"id" doc:"Comment id. JSON string of a decimal integer."`
	TopicID   repr.DecimalID `json:"topic_id" doc:"Id of the topic the comment belongs to."`
	Excerpt   string         `json:"excerpt" maxLength:"800" doc:"Plain-text excerpt of the comment body, at most 200 characters. When longer, cut to 199 characters and terminated with an ellipsis. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}
