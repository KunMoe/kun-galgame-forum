package repository

import (
	"time"

	"gorm.io/gorm"
)

type UserContentRepository struct {
	db *gorm.DB
}

func NewUserContentRepository(db *gorm.DB) *UserContentRepository {
	return &UserContentRepository{db: db}
}

func (r *UserContentRepository) FindUserWorkIDs(userID int, queryType string, page, limit int, showNoResource bool) ([]int, int64, error) {
	offset := (page - 1) * limit
	var total int64

	baseQuery := r.db.Table("galgame").Select("galgame.id").Where("galgame.published")

	switch queryType {
	case "galgame_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_like ON galgame_like.work_id = galgame.id").
			Where("galgame_like.user_id = ?", userID)
	case "galgame_favorite":
		baseQuery = baseQuery.
			Joins("JOIN galgame_favorite ON galgame_favorite.work_id = galgame.id").
			Where("galgame_favorite.user_id = ?", userID)
	default:
		return []int{}, 0, nil
	}

	if !showNoResource {
		baseQuery = baseQuery.Where("EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = galgame.id)")
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type idRow struct {
		ID int `gorm:"column:id"`
	}
	var rows []idRow
	err := baseQuery.Order("galgame.created DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, total, nil
}

type LikedPostRow struct {
	ID     int64 `gorm:"column:id"`
	PostID int64 `gorm:"column:post_id"`
}

func (r *UserContentRepository) FindUserLikedPostIDs(userID int) ([]LikedPostRow, error) {
	var rows []LikedPostRow
	err := r.db.Table("galgame_post_like").
		Select("id, post_id").
		Where("user_id = ?", userID).
		Order("id DESC").
		Scan(&rows).Error
	return rows, err
}

type UserResource struct {
	ID       int    `gorm:"column:id" json:"id"`
	WorkID   int    `gorm:"column:work_id" json:"galgame_id"`
	Type     string `gorm:"column:type" json:"type"`
	Language string `gorm:"column:language" json:"language"`
	Platform string `gorm:"column:platform" json:"platform"`
	Size     string `gorm:"column:size" json:"size"`
	Code     string `gorm:"column:code" json:"code"`
	Password string `gorm:"column:password" json:"password"`
	Note     string `gorm:"column:note" json:"note"`
	Status   int    `gorm:"column:status" json:"status"`
	Created  string `gorm:"column:created" json:"created"`
}

type ResourceLink struct {
	ResourceID int    `gorm:"column:galgame_resource_id"`
	URL        string `gorm:"column:url"`
}

func (r *UserContentRepository) FindUserResources(userID int, queryType string, page, limit int) ([]UserResource, int64, error) {
	offset := (page - 1) * limit
	var results []UserResource
	var total int64

	baseQuery := r.db.Table("galgame_resource").
		Select("galgame_resource.id, galgame_resource.work_id, galgame_resource.type, galgame_resource.language, galgame_resource.platform, galgame_resource.size, galgame_resource.code, galgame_resource.password, galgame_resource.note, galgame_resource.status, galgame_resource.created")

	switch queryType {
	case "expire":
		baseQuery = baseQuery.Where("galgame_resource.user_id = ? AND galgame_resource.status = 1", userID)
	case "galgame_resource_like":
		baseQuery = baseQuery.
			Joins("JOIN galgame_resource_like ON galgame_resource_like.galgame_resource_id = galgame_resource.id").
			Where("galgame_resource_like.user_id = ?", userID)
	default:
		baseQuery = baseQuery.Where("galgame_resource.user_id = ? AND galgame_resource.status = 0", userID)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := baseQuery.Order("galgame_resource.created DESC").Offset(offset).Limit(limit).Scan(&results).Error
	return results, total, err
}

func (r *UserContentRepository) FindResourceLinks(resourceIDs []int) (map[int][]string, error) {
	var links []ResourceLink
	err := r.db.Table("galgame_resource_link").
		Select("galgame_resource_id, url").
		Where("galgame_resource_id IN ?", resourceIDs).
		Scan(&links).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]string)
	for _, l := range links {
		result[l.ResourceID] = append(result[l.ResourceID], l.URL)
	}
	return result, nil
}

type UserRating struct {
	ID           int    `gorm:"column:id" json:"id"`
	WorkID       int    `gorm:"column:work_id" json:"galgame_id"`
	Recommend    string `gorm:"column:recommend" json:"recommend"`
	Overall      int    `gorm:"column:overall" json:"overall"`
	View         int    `gorm:"column:view" json:"view"`
	Art          int    `gorm:"column:art" json:"art"`
	Story        int    `gorm:"column:story" json:"story"`
	Music        int    `gorm:"column:music" json:"music"`
	Character    int    `gorm:"column:character" json:"character"`
	Route        int    `gorm:"column:route" json:"route"`
	System       int    `gorm:"column:system" json:"system"`
	Voice        int    `gorm:"column:voice" json:"voice"`
	ReplayValue  int    `gorm:"column:replay_value" json:"replay_value"`
	GalgameType  string `gorm:"column:galgame_type" json:"-"`
	PlayStatus   string `gorm:"column:play_status" json:"play_status"`
	ShortSummary string `gorm:"column:short_summary" json:"short_summary"`
	SpoilerLevel string `gorm:"column:spoiler_level" json:"spoiler_level"`
	LikeCount    int    `gorm:"column:like_count" json:"like_count"`
	UserID       int    `gorm:"column:user_id" json:"-"`
	Created      string `gorm:"column:created" json:"created"`
	Updated      string `gorm:"column:updated" json:"updated"`
}

func (r *UserContentRepository) FindUserRatings(userID int, page, limit int) ([]UserRating, int64, error) {
	offset := (page - 1) * limit
	var results []UserRating
	var total int64

	if err := r.db.Table("galgame_rating").Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Table("galgame_rating").
		Select(`galgame_rating.id, galgame_rating.work_id, galgame_rating.recommend, galgame_rating.overall, galgame_rating.view,
			galgame_rating.art, galgame_rating.story, galgame_rating.music, galgame_rating.character, galgame_rating.route, galgame_rating.system, galgame_rating.voice, galgame_rating.replay_value,
			galgame_rating.galgame_type, galgame_rating.play_status, galgame_rating.short_summary, galgame_rating.spoiler_level, galgame_rating.like_count,
			galgame_rating.user_id,
			galgame_rating.created, galgame_rating.updated`).
		Where("galgame_rating.user_id = ?", userID).
		Order("galgame_rating.created DESC").Offset(offset).Limit(limit).
		Scan(&results).Error
	return results, total, err
}

type GalgameLocalStats struct {
	ID                 int       `gorm:"column:id"`
	View               int       `gorm:"column:view"`
	LikeCount          int       `gorm:"column:like_count"`
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time"`
	CreatorUserID      *int      `gorm:"column:creator_user_id"`
}

func (r *UserContentRepository) FindGalgameLocalStats(ids []int) map[int]GalgameLocalStats {
	if len(ids) == 0 {
		return map[int]GalgameLocalStats{}
	}
	var rows []GalgameLocalStats
	r.db.Table("galgame").Select("id, view, like_count, resource_update_time, creator_user_id").
		Where("id IN ?", ids).Where("published").Scan(&rows)
	out := make(map[int]GalgameLocalStats, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

type GalgameResourceMeta struct {
	WorkID   int    `gorm:"column:work_id"`
	Platform string `gorm:"column:platform"`
	Language string `gorm:"column:language"`
}

func (r *UserContentRepository) FindResourceMetaByWorkIDs(ids []int) []GalgameResourceMeta {
	if len(ids) == 0 {
		return nil
	}
	var rows []GalgameResourceMeta
	r.db.Table("galgame_resource").
		Select("DISTINCT work_id, platform, language").
		Where("work_id IN ?", ids).Scan(&rows)
	return rows
}
