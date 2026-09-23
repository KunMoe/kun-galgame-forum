package repository

import "gorm.io/gorm"

const rankingBayesianPriorC = 10.0

type RankingRepository struct {
	db *gorm.DB
}

func NewRankingRepository(db *gorm.DB) *RankingRepository {
	return &RankingRepository{db: db}
}
