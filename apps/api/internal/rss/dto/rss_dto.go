package dto

type GalgameRSSUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type GalgameRSSItem struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Banner      string         `json:"banner"`
	User        GalgameRSSUser `json:"user"`
	Description string         `json:"description"`
	Created     string         `json:"created"`
}
