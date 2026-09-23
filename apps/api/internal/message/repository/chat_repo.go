package repository

import "gorm.io/gorm"

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) DB() *gorm.DB {
	return r.db
}
