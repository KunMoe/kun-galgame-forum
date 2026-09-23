package dto

type GalgameLink struct {
	ID     int       `json:"id"`
	User   UserBrief `json:"user"`
	WorkID int       `json:"galgame_id"`
	Name   string    `json:"name"`
	Link   string    `json:"link"`
}
