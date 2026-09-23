package repository

import (
	"gorm.io/gorm"
)

type TopicListRepository struct {
	db *gorm.DB
}

func NewTopicListRepository(db *gorm.DB) *TopicListRepository {
	return &TopicListRepository{db: db}
}
