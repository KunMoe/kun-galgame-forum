package dto

import (
	"time"

	galgameDto "kun-galgame-api/internal/galgame/dto"
	toolsetDto "kun-galgame-api/internal/toolset/dto"
)

type SearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
	Type     string `query:"type" validate:"required,oneof=topic galgame resource user reply comment"`
	Page     int    `query:"page" validate:"min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=24"`
	GalgameFilter
}

// GalgameFilter narrows the Galgame lane. Catalog keeps these attributes live
// when q= switches /catalog/works/search to the search index, so a filtered
// search is the same single request an unfiltered one is.
//
// 评分 is missing on purpose and the panel says so: the works index carries no
// rating attribute and no rating sort, unlike /galgame, which ranks on the
// forum's own bayesian column.
type GalgameFilter struct {
	CompanyID    string `query:"company_id" validate:"omitempty,number"`
	TagIDs       string `query:"tag_ids" validate:"omitempty,max=107"`
	ReleasedFrom string `query:"released_from" validate:"omitempty,max=7"`
	ReleasedTo   string `query:"released_to" validate:"omitempty,max=7"`
	Sort         string `query:"sort" validate:"omitempty,oneof=relevance popularity updated released_desc released_asc"`
}

type OverviewRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
}

type EntitySearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
	Family   string `query:"family" validate:"omitempty,oneof=character company staff tag series engine"`
	Page     int    `query:"page" validate:"omitempty,min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=60"`
}

// A filter chip keys on a catalog id, so a shared link arrives holding ids and
// no names.
type EntityResolveRequest struct {
	Family string `query:"family" validate:"required,oneof=company tag"`
	IDs    string `query:"ids" validate:"required,max=107"`
}

type EntityResolveResult struct {
	Items []galgameDto.EntitySearchItem `json:"items"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type TopicItem struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	View             int        `json:"view"`
	Status           int        `json:"status"`
	LikeCount        int        `json:"like_count"`
	ReplyCount       int        `json:"reply_count"`
	CommentCount     int        `json:"comment_count"`
	HasBestAnswer    bool       `json:"has_best_answer"`
	MiniApps         []string   `json:"mini_apps"`
	IsNSFWTopic      bool       `json:"is_nsfw_topic"`
	Section          []string   `json:"section"`
	UpvoteTime       *time.Time `json:"upvote_time"`
	StatusUpdateTime time.Time  `json:"status_update_time"`
	User             UserBrief  `json:"user"`
}

type UserItem struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Avatar string   `json:"avatar"`
	Bio    string   `json:"bio"`
	Roles  []string `json:"roles"`
	// OAuth owns the registration date and /users/search may answer without it,
	// so this is a pointer: a zero time.Time serialises as year 1 and the card
	// printed 0001年1月1日 for every account.
	Created    *time.Time `json:"created"`
	TopicCount int        `json:"topic_count"`
	ReplyCount int        `json:"reply_count"`
}

type ReplyItem struct {
	ID         int       `json:"id"`
	TopicID    int       `json:"topic_id"`
	TopicTitle string    `json:"topic_title"`
	Content    string    `json:"content"`
	Floor      int       `json:"floor"`
	User       UserBrief `json:"user"`
	Created    time.Time `json:"created"`
}

type CommentItem struct {
	ID         int       `json:"id"`
	TopicID    int       `json:"topic_id"`
	TopicTitle string    `json:"topic_title"`
	Content    string    `json:"content"`
	User       UserBrief `json:"user"`
	Created    time.Time `json:"created"`
}

type QuickSearchRequest struct {
	Keywords string `query:"keywords" validate:"required,max=107"`
}

type QuickSearchTotals struct {
	Topic   int64 `json:"topic"`
	Galgame int64 `json:"galgame"`
	User    int64 `json:"user"`
}

type QuickSearchResult struct {
	Topics   []TopicItem              `json:"topics"`
	Galgames []galgameDto.GalgameCard `json:"galgames"`
	Users    []UserItem               `json:"users"`
	Totals   QuickSearchTotals        `json:"totals"`
}

type PaginatedResult[T any] struct {
	Items []T
	Total int64
}

type OverviewTotals struct {
	Topic    int64 `json:"topic"`
	Galgame  int64 `json:"galgame"`
	Entity   int64 `json:"entity"`
	Resource int64 `json:"resource"`
	User     int64 `json:"user"`
	Reply    int64 `json:"reply"`
	Comment  int64 `json:"comment"`
	Toolset  int64 `json:"toolset"`
}

type OverviewResult struct {
	Topics    []TopicItem                    `json:"topics"`
	Galgames  []galgameDto.GalgameCard       `json:"galgames"`
	Entities  []galgameDto.EntitySearchGroup `json:"entities"`
	Resources []galgameDto.ResourceCard      `json:"resources"`
	Users     []UserItem                     `json:"users"`
	Replies   []ReplyItem                    `json:"replies"`
	Comments  []CommentItem                  `json:"comments"`
	Toolsets  []toolsetDto.ToolsetCard       `json:"toolsets"`
	Totals    OverviewTotals                 `json:"totals"`
}

type EntitySearchResult struct {
	Groups []galgameDto.EntitySearchGroup `json:"groups"`
	Total  int64                          `json:"total"`
}
