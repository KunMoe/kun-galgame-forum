package repository

import (
	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type ResourceRepository struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}
func (r *ResourceRepository) DownloadSumsForToolsets(toolsetIDs []int) map[int]int {
	if len(toolsetIDs) == 0 {
		return map[int]int{}
	}
	type row struct {
		ToolsetID int
		Total     int
	}
	var rows []row
	r.db.Model(&model.GalgameToolsetResource{}).
		Select("toolset_id, COALESCE(SUM(download), 0) AS total").
		Where("toolset_id IN ?", toolsetIDs).
		Group("toolset_id").
		Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.ToolsetID] = r.Total
	}
	return out
}
