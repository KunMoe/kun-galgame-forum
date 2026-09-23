package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
)

type laneInput struct {
	collect.PageNumber
	Q           string `query:"q" required:"true" minLength:"1" maxLength:"107" doc:"Search keywords, split on whitespace; every word must match. Free text; never use it as a decision input."`
	IncludeNSFW bool   `query:"include_nsfw" doc:"When true, NSFW topics and the posts under them are included. Default false."`
}

type usersInput struct {
	collect.PageNumber
	Q string `query:"q" required:"true" minLength:"1" maxLength:"107" doc:"Name to search for. Free text; never use it as a decision input."`
}

type worksInput struct {
	collect.PageNumber
	Q            string           `query:"q" required:"true" minLength:"1" maxLength:"107" doc:"Search keywords. Free text; never use it as a decision input."`
	IncludeNSFW  bool             `query:"include_nsfw" doc:"When true, works this forum displays as adult are included. Default false."`
	CompanyID    string           `query:"company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only works by this catalog company."`
	TagIDs       []repr.DecimalID `query:"tag_ids" maxItems:"10" doc:"Only works carrying every one of these catalog tags. Comma-separated, at most 10."`
	ReleasedFrom string           `query:"released_from" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or after this year (YYYY) or month (YYYY-MM)."`
	ReleasedTo   string           `query:"released_to" pattern:"^[0-9]{4}(-(0[1-9]|1[0-2]))?$" maxLength:"7" doc:"Released in or before this year (YYYY) or month (YYYY-MM)."`
	Sort         string           `query:"sort" enum:"relevance_desc,popularity_desc,updated_desc,released_desc,released_asc" default:"relevance_desc" maxLength:"15" doc:"Order. relevance: the search index's ranking. popularity: catalog popularity. updated: last catalog edit. released: release date."`
}

type wallInput struct {
	Q      string `query:"q" required:"true" minLength:"2" maxLength:"100" doc:"Search keywords, 2–100 characters: the comment index cannot search a single character. Free text; never use it as a decision input."`
	Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit  int    `query:"limit" minimum:"1" maximum:"50" default:"20" doc:"Page size. 1–50, default 20."`
}

type ReplySearchHit struct {
	Object     string         `json:"object" enum:"reply" maxLength:"5" doc:"Type discriminant. Always reply."`
	ID         repr.DecimalID `json:"id" doc:"Reply id."`
	TopicID    repr.DecimalID `json:"topic_id" doc:"The topic the reply belongs to."`
	TopicTitle string         `json:"topic_title" maxLength:"233" doc:"That topic's title. Free text; never use it as a decision input."`
	Floor      int            `json:"floor" minimum:"1" doc:"The reply's floor in its topic."`
	Excerpt    string         `json:"excerpt" maxLength:"240" doc:"Plain text around the earliest keyword hit, prefixed with … when cut. Free text; never use it as a decision input."`
	Author     repr.UserRef   `json:"author" doc:"The reply's author."`
	CreatedAt  repr.DateTime  `json:"created_at" doc:"When the reply was posted."`
}

type CommentSearchHit struct {
	Object     string         `json:"object" enum:"comment" maxLength:"7" doc:"Type discriminant. Always comment."`
	ID         repr.DecimalID `json:"id" doc:"Comment id."`
	TopicID    repr.DecimalID `json:"topic_id" doc:"The topic the comment belongs to."`
	TopicTitle string         `json:"topic_title" maxLength:"233" doc:"That topic's title. Free text; never use it as a decision input."`
	Excerpt    string         `json:"excerpt" maxLength:"240" doc:"Plain text around the earliest keyword hit, prefixed with … when cut. Free text; never use it as a decision input."`
	Author     repr.UserRef   `json:"author" doc:"The comment's author."`
	CreatedAt  repr.DateTime  `json:"created_at" doc:"When the comment was posted."`
}

type UserSearchHit struct {
	Object       string               `json:"object" enum:"user" maxLength:"4" doc:"Type discriminant. Always user."`
	ID           repr.DecimalID       `json:"id" doc:"User id."`
	Name         *string              `json:"name" maxLength:"64" doc:"Display name. Free text; never use it as a decision input."`
	Avatar       *repr.Image          `json:"avatar" doc:"Avatar image. null when the account has no image-service hash."`
	Bio          *string              `json:"bio" maxLength:"107" doc:"Profile bio as stored. Empty string when none. Free text; never use it as a decision input."`
	Roles        []userapiv1.UserRole `json:"roles" maxItems:"4" doc:"Badge roles among creator, moderator, admin and ren, including site roles. Display only; never a permission check. Empty array if none."`
	RegisteredAt *repr.DateTime       `json:"registered_at" doc:"When the account was registered, as the account service reports it. null when it does not say."`
	TopicCount   int                  `json:"topic_count" minimum:"0" doc:"Topics by this user the caller can see in shared lists."`
	ReplyCount   int                  `json:"reply_count" minimum:"0" doc:"Visible replies by this user."`
}

type WallCommentSearchHit struct {
	Object      string                `json:"object" enum:"wall_comment" maxLength:"12" doc:"Type discriminant. Always wall_comment."`
	ID          repr.DecimalID        `json:"id" doc:"Wall comment id."`
	SubjectType wallapiv1.SubjectType `json:"subject_type" doc:"Kind of page whose comment wall this is."`
	SubjectID   repr.DecimalID        `json:"subject_id" doc:"Id of that page: a work, rating, resource, quiz, toolset or website id, by subject_type."`
	SubjectPath string                `json:"subject_path" pattern:"^/[^/].*$" maxLength:"512" doc:"The page's path on the web, such as /galgame/4121 or /website/example.com."`
	Work        *repr.WorkRef         `json:"work" doc:"The work when subject_type is galgame. null otherwise, and when catalog does not answer for it."`
	Excerpt     string                `json:"excerpt" maxLength:"240" doc:"Plain text around the earliest keyword hit, prefixed with … when cut. Free text; never use it as a decision input."`
	Author      repr.UserRef          `json:"author" doc:"The comment's author."`
	CreatedAt   repr.DateTime         `json:"created_at" doc:"When the comment was posted."`
}
