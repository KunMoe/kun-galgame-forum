package dto

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

type UserGalgamesRequest struct {
	Type           string `query:"type" validate:"required"`
	Page           int    `query:"page" validate:"min=1"`
	Limit          int    `query:"limit" validate:"min=1,max=50"`
	ShowNoResource bool   `query:"show_no_resource"`
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

type BanUserRequest struct {
	Status int `json:"status" validate:"oneof=0 1"`
}
