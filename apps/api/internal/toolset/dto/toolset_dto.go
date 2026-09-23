package dto

import (
	userModel "kun-galgame-api/internal/user/model"
)

type ToolsetListRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
	Type      string `query:"type"`
	Language  string `query:"language"`
	Platform  string `query:"platform"`
	Version   string `query:"version"`
	SortField string `query:"sort_field"`
	SortOrder string `query:"sort_order"`
	Query     string `query:"query" validate:"max=100"`
	UserID    int    `query:"-"`
}

type ToolsetCard struct {
	ID                 int                 `json:"id"`
	Name               string              `json:"name"`
	User               userModel.UserBrief `json:"user"`
	Type               string              `json:"type"`
	Platform           string              `json:"platform"`
	Language           string              `json:"language"`
	Version            string              `json:"version"`
	View               int                 `json:"view"`
	Download           int                 `json:"download"`
	CommentCount       int                 `json:"comment_count"`
	PracticalityAvg    any                 `json:"practicality_avg"`
	ResourceUpdateTime any                 `json:"resource_update_time"`
}
