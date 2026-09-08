package dto

type ResourceListRequest struct {
	Page     int    `query:"page" validate:"min=1"`
	Limit    int    `query:"limit" validate:"min=1,max=50"`
	Keywords string `query:"keywords" validate:"omitempty,max=107"`
}

type GalgameResourcesRequest struct {
	GalgameID int `query:"galgame_id" validate:"required,min=1"`
}

type CreateGalgameResourceRequest struct {
	GalgameID int      `json:"galgame_id" validate:"required,min=1"`
	Type      string   `json:"type" validate:"required"`
	Language  string   `json:"language" validate:"required"`
	Platform  string   `json:"platform" validate:"required"`
	Size      string   `json:"size" validate:"required,max=107"`
	Code      string   `json:"code" validate:"max=1007"`
	Password  string   `json:"password" validate:"max=1007"`
	Note      string   `json:"note" validate:"max=10000"`
	Link      []string `json:"link" validate:"required,min=1,max=20,dive,downloadlink"`
}

type UpdateGalgameResourceRequest struct {
	GalgameResourceID int      `json:"galgame_resource_id" validate:"required,min=1"`
	GalgameID         int      `json:"galgame_id"`
	Type              string   `json:"type" validate:"required"`
	Language          string   `json:"language" validate:"required"`
	Platform          string   `json:"platform" validate:"required"`
	Size              string   `json:"size" validate:"required,max=107"`
	Code              string   `json:"code" validate:"max=1007"`
	Password          string   `json:"password" validate:"max=1007"`
	Note              string   `json:"note" validate:"max=10000"`
	Link              []string `json:"link" validate:"required,min=1,max=20,dive,downloadlink"`
}

type DeleteGalgameResourceRequest struct {
	GalgameResourceID int `query:"galgame_resource_id" validate:"required,min=1"`
}

type ToggleResourceLikeRequest struct {
	GalgameResourceID int `json:"galgame_resource_id" validate:"required,min=1"`
}

type ResourceStatusRequest struct {
	GalgameResourceID int `json:"galgame_resource_id" validate:"required,min=1"`
}

type ReportExpireResult struct {
	Verdict string `json:"verdict"`
	Marked  bool   `json:"marked"`
}

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type ResourceCard struct {
	ID                 int       `json:"id"`
	View               int       `json:"view"`
	GalgameID          int       `json:"galgame_id"`
	User               UserBrief `json:"user"`
	Type               string    `json:"type"`
	Language           string    `json:"language"`
	Platform           string    `json:"platform"`
	Size               string    `json:"size"`
	Status             int       `json:"status"`
	Download           int       `json:"download"`
	LikeCount          int       `json:"like_count"`
	IsLiked            bool      `json:"is_liked"`
	CommentCount       int       `json:"comment_count"`
	DlsitePurchaseURL  string    `json:"dlsite_purchase_url,omitempty"`
	DlsiteCouponURL    string    `json:"dlsite_coupon_url,omitempty"`
	DlsiteCampaignName string    `json:"dlsite_campaign_name,omitempty"`
	LinkDomain         string    `json:"link_domain"`
	ProviderNames      []string  `json:"provider_names"`
	Note               string    `json:"note"`
	NoteHtml           string    `json:"note_html"`
	Created            string    `json:"created"`
	Edited             *string   `json:"edited"`
	GalgameName        string    `json:"galgame_name,omitempty"`
}

type ResourceMeta struct {
	ID                 int       `json:"id"`
	View               int       `json:"view"`
	GalgameID          int       `json:"galgame_id"`
	User               UserBrief `json:"user"`
	Type               string    `json:"type"`
	Language           string    `json:"language"`
	Platform           string    `json:"platform"`
	Size               string    `json:"size"`
	Status             int       `json:"status"`
	Download           int       `json:"download"`
	LikeCount          int       `json:"like_count"`
	IsLiked            bool      `json:"is_liked"`
	CommentCount       int       `json:"comment_count"`
	DlsitePurchaseURL  string    `json:"dlsite_purchase_url,omitempty"`
	DlsiteCouponURL    string    `json:"dlsite_coupon_url,omitempty"`
	DlsiteCampaignName string    `json:"dlsite_campaign_name,omitempty"`
	LinkDomain         string    `json:"link_domain"`
	ProviderNames      []string  `json:"provider_names"`
	Note               string    `json:"note"`
	NoteHtml           string    `json:"note_html"`
	Created            string    `json:"created"`
	Edited             *string   `json:"edited"`
}

type ResourceDownloadDetail struct {
	ResourceMeta
	Link     []string `json:"link"`
	Code     string   `json:"code"`
	Password string   `json:"password"`
}

type ResourceGalgameSummary struct {
	ID                       int      `json:"id"`
	Name                     string   `json:"name"`
	EffectiveBannerHash      string   `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string   `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int      `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int      `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string   `json:"effective_banner_thumbhash,omitempty"`
	ContentLimit             string   `json:"content_limit"`
	View                     int      `json:"view"`
	ResourceUpdateTime       string   `json:"resource_update_time"`
	OriginalLanguage         string   `json:"original_language"`
	AgeLimit                 string   `json:"age_limit"`
	Platform                 []string `json:"platform"`
	Language                 []string `json:"language"`
	Type                     []string `json:"type"`
}

type ResourceDetailPage struct {
	Galgame         ResourceGalgameSummary `json:"galgame"`
	Resource        ResourceMeta           `json:"resource"`
	Recommendations []ResourceCard         `json:"recommendations"`
}

type ResourceListPage struct {
	Resources []ResourceCard `json:"resources"`
	Total     int64          `json:"total"`
}
