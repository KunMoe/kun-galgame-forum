package dto

import (
	"time"

	"kun-galgame-api/pkg/imageclient"
)

type ActivityRequest struct {
	Cursor         string `query:"cursor"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	Type           string `query:"type" validate:"required"`
	ShowNoResource bool   `query:"show_no_resource"`
}

type TimelineRequest struct {
	Cursor         string `query:"cursor"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	ShowNoResource bool   `query:"show_no_resource"`
}

type TabRequest struct {
	Tab            string `query:"tab" validate:"omitempty,oneof=all topic galgame resource others"`
	Types          string `query:"types"`
	Cursor         string `query:"cursor"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	ShowNoResource bool   `query:"show_no_resource"`
	ForceSfw       bool   `query:"force_sfw"`
}

type Actor struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type ActivityItem struct {
	ID        int       `json:"-"`
	UniqueID  string    `json:"unique_id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     Actor     `json:"actor"`
	Link      string    `json:"link"`
	Content   string    `json:"content"`
	Data      any       `json:"data,omitempty"`
}

type TopicActivityData struct {
	TopicID        int                              `json:"topic_id"`
	Title          string                           `json:"title,omitempty"`
	AuthorID       int                              `json:"author_id,omitempty"`
	Excerpt        string                           `json:"excerpt"`
	Sections       []string                         `json:"sections"`
	CoverImages    []string                         `json:"cover_images"`
	CoverImageMeta map[string]imageclient.ImageMeta `json:"cover_image_meta,omitempty"`
	View           int                              `json:"view"`
	LikeCount      int                              `json:"like_count"`
	FavoriteCount  int                              `json:"favorite_count"`
	ReplyCount     int                              `json:"reply_count"`
	CommentCount   int                              `json:"comment_count"`
	UpvoteTime     *time.Time                       `json:"upvote_time"`
	Edited         *time.Time                       `json:"edited"`
	HasBestAnswer  bool                             `json:"has_best_answer"`
	MiniApps       []string                         `json:"mini_apps"`
	IsNSFW         bool                             `json:"is_nsfw"`
	TopReply       *TopReply                        `json:"top_reply,omitempty"`
	BestAnswer     *TopReply                        `json:"best_answer,omitempty"`
	Upvotes        []TopicUpvote                    `json:"upvotes,omitempty"`
	LatestActivity *LatestActivity                  `json:"latest_activity,omitempty"`
	Reactions      []TopicReactionCount             `json:"reactions"`
}

type TopicReactionCount struct {
	Reaction string  `json:"reaction"`
	Count    int     `json:"count"`
	Reactors []Actor `json:"reactors,omitempty"`
}

type TopReply struct {
	ReplyID   int    `json:"reply_id"`
	Floor     int    `json:"floor"`
	User      Actor  `json:"user"`
	Content   string `json:"content"`
	LikeCount int    `json:"like_count"`
}

type TopicUpvote struct {
	ID          int       `json:"id"`
	User        Actor     `json:"user"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
}

type LatestActivity struct {
	Kind      string    `json:"kind"`
	ReplyID   int       `json:"reply_id"`
	Floor     int       `json:"floor"`
	CommentId int       `json:"comment_id"`
	User      Actor     `json:"user"`
	Content   string    `json:"content"`
	Created   time.Time `json:"created"`
}

type NoteActivityData struct {
	Version string `json:"version,omitempty"`
	Status  *int   `json:"status,omitempty"`
}

type EntityRefActivityData struct {
	ParentName string `json:"parent_name"`
}

type QuizActivityData struct {
	Category      string `json:"category"`
	Type          string `json:"type"`
	Difficulty    int    `json:"difficulty"`
	AnswerCount   int    `json:"answer_count"`
	CorrectCount  int    `json:"correct_count"`
	FavoriteCount int    `json:"favorite_count"`
	Description   string `json:"description"`
}

type SolutionActivityData struct {
	TopicTitle string `json:"topic_title"`
	Floor      int    `json:"floor"`
}

type ReplyActivityData struct {
	TopicTitle  string       `json:"topic_title"`
	Floor       int          `json:"floor"`
	QuotedReply *QuotedReply `json:"quoted_reply,omitempty"`
}

type QuotedReply struct {
	Floor   int    `json:"floor"`
	Content string `json:"content"`
}

type TopicCommentActivityData struct {
	TopicTitle  string       `json:"topic_title"`
	CommentId   int          `json:"comment_id"`
	QuotedReply *QuotedReply `json:"quoted_reply,omitempty"`
}

type GalgameActivityData struct {
	Name           string                  `json:"name"`
	CoverHash      string                  `json:"cover_hash"`
	Language       string                  `json:"language"`
	AgeLimit       string                  `json:"age_limit"`
	ReleaseDate    *string                 `json:"release_date"`
	WorkID         int                     `json:"galgame_id,omitempty"`
	RevisionID     int                     `json:"revision_id,omitempty"`
	RevisionNumber int                     `json:"revision_number,omitempty"`
	Developer      string                  `json:"developer,omitempty"`
	Intro          string                  `json:"intro,omitempty"`
	ResourceCount  int                     `json:"resource_count,omitempty"`
	LikeCount      int                     `json:"like_count,omitempty"`
	FavoriteCount  int                     `json:"favorite_count,omitempty"`
	Rating         *RatingInfo             `json:"rating,omitempty"`
	ParentComment  *CommentContext         `json:"parent_comment,omitempty"`
	Resource       *GalgameResourceDetails `json:"resource,omitempty"`
}

type CommentContext struct {
	Content string `json:"content"`
}

type GalgameResourceDetails struct {
	Type      string `json:"type"`
	Language  string `json:"language"`
	Platform  string `json:"platform"`
	Size      string `json:"size"`
	Note      string `json:"note"`
	LikeCount int    `json:"like_count"`
}

type RatingInfo struct {
	RatingID     int    `json:"rating_id"`
	Overall      int    `json:"overall"`
	PlayStatus   string `json:"play_status"`
	Recommend    string `json:"recommend"`
	ShortSummary string `json:"short_summary"`
	SpoilerLevel string `json:"spoiler_level"`
	LikeCount    int    `json:"like_count"`
	AuthorID     int    `json:"author_id"`
}
