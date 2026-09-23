package dto

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type UserGalgameCard struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	NameOriginal        string    `json:"name_original"`
	User                UserBrief `json:"user"`
	ContentLimit        string    `json:"content_limit"`
	View                int       `json:"view"`
	LikeCount           int       `json:"like_count"`
	ResourceUpdateTime  string    `json:"resource_update_time"`
	Platform            []string  `json:"platform"`
	Language            []string  `json:"language"`
	ReleaseDate         *string   `json:"release_date"`
	ReleaseDateTBA      bool      `json:"release_date_tba"`
	EffectiveBannerHash string    `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL  string    `json:"effective_banner_url,omitempty"`

	EffectivePortraitHash      string `json:"effective_portrait_hash,omitempty"`
	EffectivePortraitURL       string `json:"effective_portrait_url,omitempty"`
	EffectivePortraitWidth     int    `json:"effective_portrait_width,omitempty"`
	EffectivePortraitHeight    int    `json:"effective_portrait_height,omitempty"`
	EffectivePortraitThumbhash string `json:"effective_portrait_thumbhash,omitempty"`

	Company string `json:"company,omitempty"`
}

type UserGalgameComment struct {
	ID          int64     `json:"id"`
	WorkID      int       `json:"galgame_id"`
	Content     string    `json:"content"`
	ContentHtml string    `json:"content_html"`
	User        UserBrief `json:"user"`
	Created     string    `json:"created"`
	Deleted     bool      `json:"deleted"`
}

type UserGalgameCommentsRequest struct {
	Type  string `query:"type" validate:"required"`
	After string `query:"after"`
	Limit int    `query:"limit" validate:"min=1,max=50"`
}
