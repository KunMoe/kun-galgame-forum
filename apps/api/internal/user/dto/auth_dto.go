package dto

import (
	"time"
)

type OAuthCallbackRequest struct {
	Code         string `json:"code" validate:"required,max=2048"`
	CodeVerifier string `json:"code_verifier" validate:"required,max=256"`
}

type SessionResponse struct {
	Token string       `json:"-"`
	User  *UserProfile `json:"user"`
}

// AdultConfirmed and NSFWDisplay are the account's raw pair, never the folded
// stance: the web folds them itself so one function on each side owns the rule.
type UserProfile struct {
	ID             int      `json:"id"`
	Sub            string   `json:"sub"`
	Name           string   `json:"name"`
	Avatar         string   `json:"avatar"`
	Roles          []string `json:"roles"`
	Moemoepoint    int      `json:"moemoepoint"`
	Bio            string   `json:"bio"`
	AdultConfirmed bool     `json:"adult_confirmed"`
	NSFWDisplay    string   `json:"nsfw_display"`
}

type UserProfileDetail struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar"`
	Roles       []string  `json:"roles"`
	Status      int       `json:"status"`
	Moemoepoint int       `json:"moemoepoint"`
	Bio         string    `json:"bio"`
	CreatedAt   time.Time `json:"created"`

	Topic                  int64 `json:"topic"`
	TopicPoll              int64 `json:"topic_poll"`
	TopicLottery           int64 `json:"topic_lottery"`
	ReplyCreated           int64 `json:"reply_created"`
	CommentCreated         int64 `json:"comment_created"`
	Galgame                int64 `json:"galgame"`
	ContributeGalgame      int64 `json:"contribute_galgame"`
	GalgameComment         int64 `json:"galgame_comment"`
	GalgameRating          int64 `json:"galgame_rating"`
	GalgameResource        int64 `json:"galgame_resource"`
	GalgameToolset         int64 `json:"galgame_toolset"`
	GalgameToolsetResource int64 `json:"galgame_toolset_resource"`

	Upvote  int64 `json:"upvote"`
	Like    int64 `json:"like"`
	Dislike int64 `json:"dislike"`

	DailyTopicCount   int64 `json:"daily_topic_count"`
	DailyGalgameCount int64 `json:"daily_galgame_count"`
}

type UserGalgamesRequest struct {
	Type           string `query:"type" validate:"required"`
	Page           int    `query:"page" validate:"min=1"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	ShowNoResource bool   `query:"show_no_resource"`
}

type UserTopicsRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserRepliesRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserCommentsRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserResourcesRequest struct {
	Type  string `query:"type" validate:"required"`
	Page  int    `query:"page" validate:"min=1"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}

type UserRatingsRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type UserTopic struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `gorm:"column:created" json:"created"`
}

type BanUserRequest struct {
	Status int `json:"status" validate:"oneof=0 1"`
}
