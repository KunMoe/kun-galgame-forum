package dto

import "time"

type CreateCollectionRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=60"`
	Description string `json:"description" validate:"max=500"`
	Visibility  string `json:"visibility" validate:"required,oneof=public private"`
}

type UpdateCollectionRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=60"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Visibility  *string `json:"visibility" validate:"omitempty,oneof=public private"`
}

type SetCollectionMembershipRequest struct {
	CollectionIDs []int `json:"collection_ids" validate:"omitempty,max=50,dive,gt=0"`
}

type CollectionSummary struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Visibility    string    `json:"visibility"`
	IsDefault     bool      `json:"is_default"`
	ItemCount     int       `json:"item_count"`
	PreviewCovers []string  `json:"preview_covers"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
}

type CollectionDetail struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Visibility  string            `json:"visibility"`
	IsDefault   bool              `json:"is_default"`
	ItemCount   int               `json:"item_count"`
	IsOwner     bool              `json:"is_owner"`
	Owner       UserBrief         `json:"owner"`
	Galgames    []GalgameListCard `json:"galgames"`
	Total       int64             `json:"total"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
}

type MyCollectionForGalgame struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
	IsDefault  bool   `json:"is_default"`
	ItemCount  int    `json:"item_count"`
	Contains   bool   `json:"contains"`
}
