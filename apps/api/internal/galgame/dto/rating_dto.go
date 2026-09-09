package dto

import "encoding/json"

type RatingListRequest struct {
	Page         int    `query:"page" validate:"min=1"`
	Limit        int    `query:"limit" validate:"min=1,max=50"`
	SortField    string `query:"sort_field"`
	SortOrder    string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	SpoilerLevel string `query:"spoiler_level"`
	PlayStatus   string `query:"play_status"`
	GalgameType  string `query:"galgame_type"`
}

type CreateRatingRequest struct {
	GalgameID    int      `json:"galgame_id" validate:"required,min=1"`
	Recommend    string   `json:"recommend" validate:"required"`
	Overall      int      `json:"overall" validate:"required,min=1,max=10"`
	GalgameType  []string `json:"galgame_type" validate:"required,min=1"`
	PlayStatus   string   `json:"play_status" validate:"required,oneof=wish doing done_one_route done_main done_all on_hold dropped"`
	ShortSummary string   `json:"short_summary" validate:"max=1314"`
	SpoilerLevel string   `json:"spoiler_level"`
	Art          int      `json:"art" validate:"min=0,max=10"`
	Story        int      `json:"story" validate:"min=0,max=10"`
	Music        int      `json:"music" validate:"min=0,max=10"`
	Character    int      `json:"character" validate:"min=0,max=10"`
	Route        int      `json:"route" validate:"min=0,max=10"`
	System       int      `json:"system" validate:"min=0,max=10"`
	Voice        int      `json:"voice" validate:"min=0,max=10"`
	ReplayValue  int      `json:"replay_value" validate:"min=0,max=10"`
}

type UpdateRatingRequest struct {
	GalgameRatingID int      `json:"galgame_rating_id" validate:"required,min=1"`
	Recommend       string   `json:"recommend" validate:"required"`
	Overall         int      `json:"overall" validate:"required,min=1,max=10"`
	GalgameType     []string `json:"galgame_type" validate:"required,min=1"`
	PlayStatus      string   `json:"play_status" validate:"required,oneof=wish doing done_one_route done_main done_all on_hold dropped"`
	ShortSummary    string   `json:"short_summary" validate:"max=1314"`
	SpoilerLevel    string   `json:"spoiler_level" validate:"required"`
	Art             int      `json:"art" validate:"min=0,max=10"`
	Story           int      `json:"story" validate:"min=0,max=10"`
	Music           int      `json:"music" validate:"min=0,max=10"`
	Character       int      `json:"character" validate:"min=0,max=10"`
	Route           int      `json:"route" validate:"min=0,max=10"`
	System          int      `json:"system" validate:"min=0,max=10"`
	Voice           int      `json:"voice" validate:"min=0,max=10"`
	ReplayValue     int      `json:"replay_value" validate:"min=0,max=10"`
}

type DeleteRatingRequest struct {
	GalgameRatingID int `query:"galgame_rating_id" validate:"required,min=1"`
}

type ToggleRatingLikeRequest struct {
	GalgameRatingID int `json:"galgame_rating_id" validate:"required,min=1"`
}

type CreatedRating struct {
	ID           int             `json:"id"`
	User         UserBrief       `json:"user"`
	Recommend    string          `json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `json:"view"`
	GalgameType  json.RawMessage `json:"galgame_type"`
	PlayStatus   string          `json:"play_status"`
	ShortSummary string          `json:"short_summary"`
	SpoilerLevel string          `json:"spoiler_level"`
	RatingScores
	LikeCount int                `json:"like_count"`
	IsLiked   bool               `json:"is_liked"`
	Created   string             `json:"created"`
	Updated   string             `json:"updated"`
	Galgame   RatingGalgameBrief `json:"galgame"`
}

type RatingScores struct {
	Art         int `json:"art"`
	Story       int `json:"story"`
	Music       int `json:"music"`
	Character   int `json:"character"`
	Route       int `json:"route"`
	System      int `json:"system"`
	Voice       int `json:"voice"`
	ReplayValue int `json:"replay_value"`
}

type RatingGalgameBrief struct {
	ID           int    `json:"id"`
	ContentLimit string `json:"content_limit"`
	Name         string `json:"name"`
}

type RatingCard struct {
	ID           int             `json:"id"`
	User         UserBrief       `json:"user"`
	Recommend    string          `json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `json:"view"`
	GalgameType  json.RawMessage `json:"galgame_type"`
	PlayStatus   string          `json:"play_status"`
	ShortSummary string          `json:"short_summary"`
	SpoilerLevel string          `json:"spoiler_level"`
	RatingScores
	LikeCount int                `json:"like_count"`
	Created   string             `json:"created"`
	Updated   string             `json:"updated"`
	Galgame   RatingGalgameBrief `json:"galgame"`
}

type RatingListPage struct {
	RatingData []RatingCard `json:"rating_data"`
	Total      int64        `json:"total"`
}

type RatingOfficial struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type RatingGalgameDetail struct {
	ID                       int              `json:"id"`
	ContentLimit             string           `json:"content_limit"`
	EffectiveBannerHash      string           `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string           `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int              `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int              `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string           `json:"effective_banner_thumbhash,omitempty"`
	AgeLimit                 string           `json:"age_limit"`
	OriginalLanguage         string           `json:"original_language"`
	Rating                   int64            `json:"rating"`
	RatingCount              int64            `json:"rating_count"`
	Official                 []RatingOfficial `json:"official"`
	Name                     string           `json:"name"`
	NameOriginal             string           `json:"name_original"`
}

type RatingDetail struct {
	ID           int             `json:"id"`
	User         UserBrief       `json:"user"`
	Recommend    string          `json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `json:"view"`
	GalgameType  json.RawMessage `json:"galgame_type"`
	PlayStatus   string          `json:"play_status"`
	ShortSummary string          `json:"short_summary"`
	SpoilerLevel string          `json:"spoiler_level"`
	RatingScores
	LikeCount  int                 `json:"like_count"`
	IsLiked    bool                `json:"is_liked"`
	LikedUsers []UserBrief         `json:"liked_users"`
	Created    string              `json:"created"`
	Updated    string              `json:"updated"`
	Galgame    RatingGalgameDetail `json:"galgame"`
}
