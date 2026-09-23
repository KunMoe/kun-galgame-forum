package repository

import (
	"gorm.io/gorm"
)

type FriendLinkRepository struct {
	db *gorm.DB
}

func NewFriendLinkRepository(db *gorm.DB) *FriendLinkRepository {
	return &FriendLinkRepository{db: db}
}
