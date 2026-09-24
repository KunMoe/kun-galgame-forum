package repository

import (
	"gorm.io/gorm"
)

type GalgameResourceMetaRepository struct {
	db *gorm.DB
}

func NewGalgameResourceMetaRepository(db *gorm.DB) *GalgameResourceMetaRepository {
	return &GalgameResourceMetaRepository{db: db}
}

type ResourceAxes struct {
	Platforms map[string]bool
	Languages map[string]bool
}

func (r *GalgameResourceMetaRepository) FindResourceAxesBatch(workIDs []int) (map[int]ResourceAxes, error) {
	out := make(map[int]ResourceAxes, len(workIDs))
	if len(workIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		WorkID int    `gorm:"column:work_id"`
		Axis   string `gorm:"column:axis"`
		Key    string `gorm:"column:key"`
	}
	err := r.db.Raw(`SELECT DISTINCT gr.work_id, 'platform' AS axis, k.key
		FROM galgame_resource gr, jsonb_array_elements_text(gr.platforms) AS k(key)
		WHERE gr.work_id IN ?
		UNION
		SELECT DISTINCT gr.work_id, 'language' AS axis, k.key
		FROM galgame_resource gr, jsonb_array_elements_text(gr.languages) AS k(key)
		WHERE gr.work_id IN ?`, workIDs, workIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		axes, ok := out[row.WorkID]
		if !ok {
			axes = ResourceAxes{Platforms: map[string]bool{}, Languages: map[string]bool{}}
			out[row.WorkID] = axes
		}
		if row.Axis == "platform" {
			axes.Platforms[row.Key] = true
		} else {
			axes.Languages[row.Key] = true
		}
	}
	return out, nil
}
