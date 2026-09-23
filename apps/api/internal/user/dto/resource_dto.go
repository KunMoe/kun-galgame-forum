package dto

type UserResourceItem struct {
	ID          int      `json:"id"`
	WorkID      int      `json:"galgame_id"`
	GalgameName string   `json:"galgame_name"`
	Type        string   `json:"type"`
	Language    string   `json:"language"`
	Platform    string   `json:"platform"`
	Size        string   `json:"size"`
	Link        []string `json:"link"`
	Code        string   `json:"code"`
	Password    string   `json:"password"`
	Note        string   `json:"note"`
	Status      int      `json:"status"`
	Created     string   `json:"created"`
}

type UserResourcesResponse struct {
	Resources []UserResourceItem `json:"resources"`
	Total     int64              `json:"total"`
}
