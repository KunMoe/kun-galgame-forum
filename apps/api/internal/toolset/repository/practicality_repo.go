package repository

import (
	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type PracticalityRepository struct {
	db *gorm.DB
}

func NewPracticalityRepository(db *gorm.DB) *PracticalityRepository {
	return &PracticalityRepository{db: db}
}

type RateCount struct {
	Rate  int   `json:"rate"`
	Count int64 `json:"count"`
}

func (r *PracticalityRepository) AveragesForToolsets(toolsetIDs []int) map[int]float64 {
	if len(toolsetIDs) == 0 {
		return map[int]float64{}
	}
	type row struct {
		ToolsetID int
		Avg       float64
	}
	var rows []row
	r.db.Model(&model.GalgameToolsetPracticality{}).
		Select("toolset_id, COALESCE(AVG(rate), 0) AS avg").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows)
	out := make(map[int]float64, len(rows))
	for _, r := range rows {
		out[r.ToolsetID] = r.Avg
	}
	return out
}
