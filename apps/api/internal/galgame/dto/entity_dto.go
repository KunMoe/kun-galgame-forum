package dto

type GalgameCard struct {
	ID                         int       `json:"id"`
	Name                       string    `json:"name"`
	NameOriginal               string    `json:"name_original"`
	User                       UserBrief `json:"user"`
	ContentLimit               string    `json:"content_limit"`
	View                       int       `json:"view"`
	LikeCount                  int       `json:"like_count"`
	Rating                     float64   `json:"rating"`
	RatingCount                int       `json:"rating_count"`
	ResourceUpdateTime         string    `json:"resource_update_time"`
	Platform                   []string  `json:"platform"`
	Language                   []string  `json:"language"`
	ReleaseDate                *string   `json:"release_date"`
	ReleaseDateTBA             bool      `json:"release_date_tba"`
	ReleasePrecision           string    `json:"release_precision,omitempty"`
	EffectiveBannerHash        string    `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL         string    `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth       int       `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight      int       `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash   string    `json:"effective_banner_thumbhash,omitempty"`
	EffectivePortraitHash      string    `json:"effective_portrait_hash,omitempty"`
	EffectivePortraitURL       string    `json:"effective_portrait_url,omitempty"`
	EffectivePortraitWidth     int       `json:"effective_portrait_width,omitempty"`
	EffectivePortraitHeight    int       `json:"effective_portrait_height,omitempty"`
	EffectivePortraitThumbhash string    `json:"effective_portrait_thumbhash,omitempty"`
	IsOnForum                  bool      `json:"is_on_forum"`
	Status                     int       `json:"status,omitempty"`
	Company                    string    `json:"company,omitempty"`
}

type TagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
}

type EntitySearchItem struct {
	ID        int    `json:"id"`
	Family    string `json:"family"`
	Name      string `json:"name"`
	Alias     string `json:"alias,omitempty"`
	Image     string `json:"image,omitempty"`
	WorkCount int    `json:"work_count,omitempty"`
}

type EntitySearchGroup struct {
	Family string             `json:"family"`
	Total  int64              `json:"total"`
	Items  []EntitySearchItem `json:"items"`
}
