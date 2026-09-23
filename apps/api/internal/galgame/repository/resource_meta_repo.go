package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type GalgameResourceMetaRepository struct {
	db *gorm.DB
}

func NewGalgameResourceMetaRepository(db *gorm.DB) *GalgameResourceMetaRepository {
	return &GalgameResourceMetaRepository{db: db}
}

func (r *GalgameResourceMetaRepository) FindResourceMetaByWork(workID int) (platforms, languages, types []string) {
	type row struct {
		Platform string `gorm:"column:platform"`
		Language string `gorm:"column:language"`
		Type     string `gorm:"column:type"`
	}
	var rows []row
	r.db.Table("galgame_resource").
		Select("DISTINCT platform, language, type").
		Where("work_id = ?", workID).Scan(&rows)

	pSet, lSet, tSet := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, x := range rows {
		if x.Platform != "" {
			pSet[x.Platform] = true
		}
		if x.Language != "" {
			lSet[x.Language] = true
		}
		if x.Type != "" {
			tSet[x.Type] = true
		}
	}
	return mapKeys(pSet), mapKeys(lSet), mapKeys(tSet)
}

func (r *GalgameResourceMetaRepository) FindResourceMetaBatch(workIDs []int) []model.GalgameResourceMeta {
	if len(workIDs) == 0 {
		return nil
	}
	var rows []model.GalgameResourceMeta
	r.db.Table("galgame_resource").
		Select("DISTINCT work_id, platform, language").
		Where("work_id IN ?", workIDs).Scan(&rows)
	return rows
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

func mapKeys(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
