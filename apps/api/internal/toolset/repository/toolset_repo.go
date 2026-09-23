package repository

import (
	"strings"

	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type ToolsetRepository struct {
	db *gorm.DB
}

func NewToolsetRepository(db *gorm.DB) *ToolsetRepository {
	return &ToolsetRepository{db: db}
}

type ListFilters struct {
	Type     string
	Language string
	Platform string
	Version  string
	UserID   int
	Query    string
}

type ListOptions struct {
	SortField string
	SortOrder string
	Offset    int
	Limit     int
}

func (r *ToolsetRepository) buildListQuery(f ListFilters) *gorm.DB {
	q := r.db.Model(&model.GalgameToolset{}).Where("status != 1")
	if f.Type != "" && f.Type != "all" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Language != "" && f.Language != "all" {
		q = q.Where("language = ?", f.Language)
	}
	if f.Platform != "" && f.Platform != "all" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.Version != "" && f.Version != "all" {
		q = q.Where("version = ?", f.Version)
	}
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if kw := strings.TrimSpace(f.Query); kw != "" {
		q = q.Where("name ILIKE ?", "%"+escapeLike(kw)+"%")
	}
	return q
}

func escapeLike(s string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	).Replace(s)
}

func (r *ToolsetRepository) CountFiltered(f ListFilters) int64 {
	var total int64
	r.buildListQuery(f).Count(&total)
	return total
}

func (r *ToolsetRepository) ListFiltered(f ListFilters, o ListOptions) []model.GalgameToolset {
	var toolsets []model.GalgameToolset
	r.buildListQuery(f).
		Order(o.SortField + " " + o.SortOrder).
		Offset(o.Offset).Limit(o.Limit).
		Find(&toolsets)
	return toolsets
}
