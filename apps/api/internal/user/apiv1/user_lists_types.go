package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	"kun-galgame-api/internal/galgame/workrepr"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
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

type UserWorksPage struct {
	UserID      string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW works are included. Default false."`
}

func (p UserWorksPage) number() collect.PageNumber {
	return collect.PageNumber{Page: p.Page, Limit: p.Limit}
}

type listUserWorksInput struct {
	UserWorksPage
	Relation string `query:"relation" required:"true" enum:"published,contributed,liked" maxLength:"11" doc:"How the listed works relate to the user: published, contributed, or liked."`
}

type UserResourcesPage struct {
	UserID      string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, resources of NSFW works are included. Default false."`
}

func (p UserResourcesPage) number() collect.PageNumber {
	return collect.PageNumber{Page: p.Page, Limit: p.Limit}
}

type listUserGalgameResourcesInput struct {
	UserResourcesPage
	Relation string `query:"relation" required:"true" enum:"published,liked" maxLength:"9" doc:"How the listed resources relate to the user: published or liked."`
	State    string `query:"state" enum:"valid,expired" required:"false" maxLength:"7" doc:"When set, only this state. Omitted means every state."`
}

type listUserWallCommentsInput struct {
	UserID      string                `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Relation    string                `query:"relation" required:"true" enum:"authored,liked" maxLength:"8" doc:"How the listed wall comments relate to the user: authored or liked."`
	SubjectType wallapiv1.SubjectType `query:"subject_type" required:"false" doc:"Kind of page whose wall the comments are on. Omitted means every wall."`
	Cursor      string                `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit       int                   `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type listUserWorksOutput struct {
	Body repr.PageList[workrepr.WorkSummary]
}

type listUserGalgameResourcesOutput struct {
	Body repr.PageList[resourceapiv1.GalgameResource]
}

type listUserWallCommentsOutput struct {
	Body repr.List[wallapiv1.WallComment]
}

type UserTopicItem struct {
	Object    string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID        repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title     string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type UserReplyItem struct {
	Object     string         `json:"object" enum:"reply" maxLength:"5" doc:"Type discriminant. Always reply."`
	ID         repr.DecimalID `json:"id" doc:"Reply id. JSON string of a decimal integer."`
	TopicID    repr.DecimalID `json:"topic_id" doc:"Id of the topic the reply belongs to."`
	TopicTitle string         `json:"topic_title" maxLength:"233" doc:"That topic's title. Free text; never use it as a decision input."`
	Floor      int            `json:"floor" minimum:"1" doc:"Floor number, assigned when the reply was created and never renumbered. Deleted and hidden replies leave gaps; a floor is not a position."`
	Excerpt    string         `json:"excerpt" maxLength:"200" doc:"Plain-text excerpt of the reply body, at most 200 characters. When longer, cut to 199 characters and terminated with an ellipsis. Free text; never use it as a decision input."`
	Author     repr.UserRef   `json:"author" doc:"Reply author."`
	CreatedAt  repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type UserCommentItem struct {
	Object     string         `json:"object" enum:"comment" maxLength:"7" doc:"Type discriminant. Always comment."`
	ID         repr.DecimalID `json:"id" doc:"Comment id. JSON string of a decimal integer."`
	TopicID    repr.DecimalID `json:"topic_id" doc:"Id of the topic the comment belongs to."`
	TopicTitle string         `json:"topic_title" maxLength:"233" doc:"That topic's title. Free text; never use it as a decision input."`
	Excerpt    string         `json:"excerpt" maxLength:"200" doc:"Plain-text excerpt of the comment body, at most 200 characters. When longer, cut to 199 characters and terminated with an ellipsis. Free text; never use it as a decision input."`
	Author     repr.UserRef   `json:"author" doc:"Comment author."`
	CreatedAt  repr.DateTime  `json:"created_at" doc:"Creation time."`
}
