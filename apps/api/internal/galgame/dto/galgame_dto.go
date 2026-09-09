package dto

import "encoding/json"

type MyGalgameInteractions struct {
	Liked     []int `json:"liked"`
	Favorited []int `json:"favorited"`
}

type GalgameListRequest struct {
	Page                 int     `query:"page" validate:"min=1"`
	Limit                int     `query:"limit" validate:"min=1,max=50"`
	Type                 string  `query:"type"`
	Language             string  `query:"language"`
	Platform             string  `query:"platform"`
	GameType             string  `query:"game_type" validate:"omitempty,oneof=all ba_saku plot moe daily uncategorized"`
	SortField            string  `query:"sort_field"`
	SortOrder            string  `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	IncludeProviders     string  `query:"include_providers"`
	ExcludeOnlyProviders string  `query:"exclude_only_providers"`
	ReleasedFrom         string  `query:"released_from"`
	ReleasedTo           string  `query:"released_to"`
	ReleasedMonths       string  `query:"released_months"`
	MinRatingCount       int     `query:"min_rating_count" validate:"omitempty,min=0"`
	MinRating            float64 `query:"min_rating" validate:"omitempty,min=0,max=10"`
	ShowNoResource       bool    `query:"show_no_resource"`
	Indexed              bool    `query:"indexed"`
	Library              bool    `query:"library"`
}

type GalgameCover struct {
	ImageHash string `json:"image_hash"`
	SortOrder int    `json:"sort_order"`
	Sexual    int    `json:"sexual"`
	Violence  int    `json:"violence"`
	Source    string `json:"source"`
	SourceKey string `json:"source_key"`
	Kind      string `json:"kind,omitempty"`
	CDNURL    string `json:"cdn_url,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Thumbhash string `json:"thumbhash,omitempty"`
	ID        int64  `json:"id,omitempty"`
	VoteCount int    `json:"vote_count"`
	Voted     bool   `json:"voted"`
}

type GalgameScreenshot struct {
	ImageHash string `json:"image_hash"`
	SortOrder int    `json:"sort_order"`
	Caption   string `json:"caption"`
	Sexual    int    `json:"sexual"`
	Violence  int    `json:"violence"`
	Source    string `json:"source"`
	SourceKey string `json:"source_key"`
	CDNURL    string `json:"cdn_url,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

type GalgameListCard struct {
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
	IsOnForum                  bool      `json:"is_on_forum"`
	Platform                   []string  `json:"platform"`
	Language                   []string  `json:"language"`
	ReleaseDate                *string   `json:"release_date"`
	ReleaseDateTBA             bool      `json:"release_date_tba"`
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
	Company                    string    `json:"company,omitempty"`
}

type GalgameListPage struct {
	Galgames []GalgameListCard `json:"galgames"`
	Total    int64             `json:"total"`
}

type DraftsPage struct {
	Items []GalgameCard `json:"items"`
	Total int64         `json:"total"`
}

type GalgameDetailOfficial struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Roles        []string `json:"roles"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type GalgameDetailSeries struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GalgameDetailEngine struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type GalgameDetailTag struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
	SpoilerLevel int    `json:"spoiler_level"`
}

type GalgameDetailStaff struct {
	RoleKey  string                   `json:"role_key"`
	RoleName string                   `json:"role_name"`
	People   []GalgameDetailStaffName `json:"people"`
}

type GalgameDetailStaffName struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Latin      string   `json:"latin,omitempty"`
	Characters []string `json:"characters,omitempty"`
}

type GalgameDetailCharacter struct {
	ID           int                           `json:"id"`
	Name         string                        `json:"name"`
	NameOriginal string                        `json:"name_original,omitempty"`
	Latin        string                        `json:"latin,omitempty"`
	Kind         string                        `json:"kind"`
	Spoiler      int                           `json:"spoiler"`
	Identity     string                        `json:"identity,omitempty"`
	Image        string                        `json:"image,omitempty"`
	Figure       string                        `json:"figure,omitempty"`
	ImageMeta    *GalgameArtMeta               `json:"image_meta,omitempty"`
	FigureMeta   *GalgameArtMeta               `json:"figure_meta,omitempty"`
	Voices       []GalgameDetailCharacterVoice `json:"voices"`
}

type GalgameArtMeta struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

type GalgameDetailCharacterVoice struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Lang  string `json:"lang,omitempty"`
	Latin string `json:"latin,omitempty"`
}

type GalgameDetailRatingGalgame struct {
	ID           int    `json:"id"`
	ContentLimit string `json:"content_limit"`
	Name         string `json:"name"`
}

type GalgameDetailRating struct {
	ID           int                        `json:"id"`
	User         UserBrief                  `json:"user"`
	Recommend    string                     `json:"recommend"`
	Overall      int                        `json:"overall"`
	View         int                        `json:"view"`
	GalgameType  json.RawMessage            `json:"galgame_type"`
	PlayStatus   string                     `json:"play_status"`
	ShortSummary string                     `json:"short_summary"`
	SpoilerLevel string                     `json:"spoiler_level"`
	Art          int                        `json:"art"`
	Story        int                        `json:"story"`
	Music        int                        `json:"music"`
	Character    int                        `json:"character"`
	Route        int                        `json:"route"`
	System       int                        `json:"system"`
	Voice        int                        `json:"voice"`
	ReplayValue  int                        `json:"replay_value"`
	LikeCount    int                        `json:"like_count"`
	IsLiked      bool                       `json:"is_liked"`
	GalgameID    int                        `json:"galgame_id"`
	Created      string                     `json:"created"`
	Updated      string                     `json:"updated"`
	Galgame      GalgameDetailRatingGalgame `json:"galgame"`
}

type GalgameRatingBucket struct {
	Score int `json:"score"`
	Count int `json:"count"`
}

type GalgameRatingStats struct {
	Average *float64 `json:"average,omitempty"`
	Stdev   *float64 `json:"stdev,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
}

type GalgameExternalRating struct {
	Source       string                `json:"source"`
	Score        float64               `json:"score"`
	VoteCount    int                   `json:"vote_count"`
	Rank         *int                  `json:"rank,omitempty"`
	Distribution []GalgameRatingBucket `json:"distribution,omitempty"`
	Stats        *GalgameRatingStats   `json:"stats,omitempty"`
}

type GalgamePlaytime struct {
	Source    string `json:"source"`
	Minutes   int    `json:"minutes"`
	VoteCount int    `json:"vote_count"`
}

// GalgameIntro is one language's introduction. The language list is whatever
// catalog actually carries rather than a fixed set: the four product slots this
// replaces always shipped an empty 繁體中文, because catalog has never held a
// zh-Hant intro row, and the reader got a tab that could only say 暂无对应翻译.
type GalgameIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Machine bool   `json:"machine"`
}

type GalgameDetail struct {
	ID                         int                      `json:"id"`
	MovedTo                    int                      `json:"moved_to,omitempty"`
	VndbID                     string                   `json:"vndb_id"`
	User                       UserBrief                `json:"user"`
	Name                       string                   `json:"name"`
	NameOriginal               string                   `json:"name_original"`
	Introduction               []GalgameIntro           `json:"introduction"`
	IntroText                  string                   `json:"intro_text"`
	ContentLimit               string                   `json:"content_limit"`
	ResourceUpdateTime         string                   `json:"resource_update_time"`
	View                       int                      `json:"view"`
	IsOnForum                  bool                     `json:"is_on_forum"`
	Indexed                    bool                     `json:"indexed"`
	Status                     int                      `json:"status"`
	OriginalLanguage           string                   `json:"original_language"`
	AgeLimit                   string                   `json:"age_limit"`
	ReleaseDate                *string                  `json:"release_date"`
	ReleaseDateTBA             bool                     `json:"release_date_tba"`
	EffectiveBannerHash        string                   `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL         string                   `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth       int                      `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight      int                      `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash   string                   `json:"effective_banner_thumbhash,omitempty"`
	EffectivePortraitHash      string                   `json:"effective_portrait_hash,omitempty"`
	EffectivePortraitURL       string                   `json:"effective_portrait_url,omitempty"`
	EffectivePortraitWidth     int                      `json:"effective_portrait_width,omitempty"`
	EffectivePortraitHeight    int                      `json:"effective_portrait_height,omitempty"`
	EffectivePortraitThumbhash string                   `json:"effective_portrait_thumbhash,omitempty"`
	Covers                     []GalgameCover           `json:"covers"`
	Screenshots                []GalgameScreenshot      `json:"screenshots"`
	Platform                   []string                 `json:"platform"`
	Language                   []string                 `json:"language"`
	Type                       []string                 `json:"type"`
	Contributor                []UserBrief              `json:"contributor"`
	LikeCount                  int                      `json:"like_count"`
	IsLiked                    bool                     `json:"is_liked"`
	FavoriteCount              int                      `json:"favorite_count"`
	IsFavorited                bool                     `json:"is_favorited"`
	ResourcePublishBanned      bool                     `json:"resource_publish_banned"`
	Alias                      []string                 `json:"alias"`
	Engine                     []GalgameDetailEngine    `json:"engine"`
	Official                   []GalgameDetailOfficial  `json:"official"`
	Series                     []GalgameDetailSeries    `json:"series"`
	Tag                        []GalgameDetailTag       `json:"tag"`
	Staff                      []GalgameDetailStaff     `json:"staff"`
	Characters                 []GalgameDetailCharacter `json:"characters"`
	Ratings                    []GalgameDetailRating    `json:"ratings"`
	Rating                     float64                  `json:"rating"`
	RatingCount                int                      `json:"rating_count"`
	ExternalRatings            []GalgameExternalRating  `json:"external_ratings"`
	Playtimes                  []GalgamePlaytime        `json:"playtimes"`
	Refs                       map[string]string        `json:"refs,omitempty"`
	Created                    string                   `json:"created"`
	Updated                    string                   `json:"updated"`
	DlsitePurchaseURL          string                   `json:"dlsite_purchase_url,omitempty"`
	DlsiteCouponURL            string                   `json:"dlsite_coupon_url,omitempty"`
	DlsiteCampaignName         string                   `json:"dlsite_campaign_name,omitempty"`
	MyPlaytime                 *GalgameMyPlaytime       `json:"my_playtime,omitempty"`
}

// Minutes may be 0 while Status is set: a state-only record, which is the
// normal case when the user marked a state without a duration. Status is the
// flat play-state value (or empty when the user has none).
type GalgameMyPlaytime struct {
	Minutes int    `json:"minutes"`
	Status  string `json:"status"`
}
