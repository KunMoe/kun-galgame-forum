package repository

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/model"
	msgModel "kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrResourceNotFound = errors.New("galgame resource: row not found")

type ResourceV1Store struct {
	db *gorm.DB
}

func NewResourceV1Store(db *gorm.DB) *ResourceV1Store {
	return &ResourceV1Store{db: db}
}

func (s *ResourceV1Store) Ready() bool {
	return s != nil && s.db != nil
}

func (s *ResourceV1Store) InTx(fn func(tx *gorm.DB) error) error {
	return s.db.Transaction(fn)
}

func resourceNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrResourceNotFound
	}
	return err
}

type ResourceListFilter struct {
	Q              string
	IncludeNSFW    bool
	State          string
	WorkID         int
	CatalogWorkIDs []int
	AuthorIDs      []int
	SkipNSFW       bool
}

func escapeLike(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(s)
}

func (f ResourceListFilter) apply(q *gorm.DB) *gorm.DB {
	q = q.Joins("JOIN galgame g ON g.id = r.work_id").Where("g.published = ?", true)
	if !f.SkipNSFW && !f.IncludeNSFW {
		q = q.Where("g.content_limit IS NULL OR g.content_limit <> ?", "nsfw")
	}
	switch f.State {
	case "valid":
		q = q.Where("r.status = ?", 0)
	case "expired":
		q = q.Where("r.status = ?", 1)
	}
	if f.WorkID > 0 {
		q = q.Where("r.work_id = ?", f.WorkID)
	}
	if kw := strings.TrimSpace(f.Q); kw != "" {
		like := "%" + escapeLike(kw) + "%"
		if len(f.CatalogWorkIDs) > 0 {
			q = q.Where("r.note ILIKE ? ESCAPE '\\' OR r.work_id IN ?", like, f.CatalogWorkIDs)
		} else {
			q = q.Where("r.note ILIKE ? ESCAPE '\\'", like)
		}
	}
	if f.AuthorIDs != nil {
		q = q.Where("r.user_id IN ?", f.AuthorIDs)
	}
	return q
}

func (s *ResourceV1Store) scoped(f ResourceListFilter) *gorm.DB {
	return f.apply(s.db.Table("galgame_resource AS r"))
}

func (s *ResourceV1Store) DistinctAuthors(f ResourceListFilter) ([]int, error) {
	f.AuthorIDs = nil
	var ids []int
	err := s.scoped(f).Distinct("r.user_id").Pluck("r.user_id", &ids).Error
	return ids, err
}

func (s *ResourceV1Store) Count(f ResourceListFilter) (int, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := s.scoped(f).Count(&n).Error
	return int(n), err
}

func (s *ResourceV1Store) List(f ResourceListFilter, sort string, offset, limit int) ([]model.GalgameResource, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return nil, nil
	}
	q := s.scoped(f).Select("r.*")
	switch sort {
	case "created_asc":
		q = q.Order("r.created ASC, r.id ASC")
	case "work":
		q = q.Order("r.status ASC, r.created DESC, r.id DESC")
	case "relevance":
		q = q.Order(clause.Expr{
			SQL:  "CASE WHEN r.work_id IN ? THEN 1 ELSE 0 END DESC, r.created DESC, r.id DESC",
			Vars: []any{f.CatalogWorkIDs},
		})
	default:
		q = q.Order("r.created DESC, r.id DESC")
	}
	var rows []model.GalgameResource
	err := q.Offset(offset).Limit(limit).Scan(&rows).Error
	if rows == nil {
		rows = []model.GalgameResource{}
	}
	return rows, err
}

func (s *ResourceV1Store) Find(id int) (*model.GalgameResource, error) {
	var row model.GalgameResource
	err := s.db.Where("id = ?", id).Take(&row).Error
	if err != nil {
		return nil, resourceNotFound(err)
	}
	return &row, nil
}

func (s *ResourceV1Store) WorkPublished(workID int) (bool, error) {
	var row model.GalgameLocal
	err := s.db.Select("id", "published").Where("id = ?", workID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return row.Published, err
}

func (s *ResourceV1Store) IsPublishBanned(workID int) (bool, error) {
	var row struct {
		Banned bool `gorm:"column:resource_publish_banned"`
	}
	err := s.db.Table("galgame").Select("resource_publish_banned").Where("id = ?", workID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return row.Banned, err
}

func (s *ResourceV1Store) LocalPublished(tx *gorm.DB, workID int) (bool, error) {
	var row model.GalgameLocal
	err := tx.Select("id", "published").Where("id = ?", workID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return row.Published, err
}

func (s *ResourceV1Store) PublishLocal(tx *gorm.DB, workID int) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{"published": true}),
	}).Create(&model.GalgameLocal{ID: workID, Published: true}).Error
}

func (s *ResourceV1Store) IncrementView(id int) error {
	res := s.db.Exec(`UPDATE galgame_resource SET view = view + 1 WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (s *ResourceV1Store) IncrementDownload(id int) error {
	res := s.db.Exec(`UPDATE galgame_resource SET download = download + 1 WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (s *ResourceV1Store) ProviderNames(ids []int) (map[int][]string, error) {
	out := map[int][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID           int             `gorm:"column:id"`
		ProviderName json.RawMessage `gorm:"column:provider_name"`
	}
	err := s.db.Table("galgame_resource").Select("id, provider_name").Where("id IN ?", ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		var names []string
		if len(r.ProviderName) > 0 {
			_ = json.Unmarshal(r.ProviderName, &names)
		}
		if names == nil {
			names = []string{}
		}
		out[r.ID] = names
	}
	return out, nil
}

func (s *ResourceV1Store) Links(resourceID int) ([]string, error) {
	var rows []model.GalgameResourceLink
	err := s.db.Where("galgame_resource_id = ?", resourceID).Order("id ASC").Find(&rows).Error
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.URL
	}
	return out, err
}

func (s *ResourceV1Store) LikedSet(userID int, ids []int) (map[int]bool, error) {
	out := map[int]bool{}
	if userID == 0 || len(ids) == 0 {
		return out, nil
	}
	var liked []int
	err := s.db.Table("galgame_resource_like").
		Where("user_id = ? AND galgame_resource_id IN ?", userID, ids).
		Pluck("galgame_resource_id", &liked).Error
	for _, id := range liked {
		out[id] = true
	}
	return out, err
}

func (s *ResourceV1Store) HasLiked(resourceID, userID int) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var n int64
	err := s.db.Table("galgame_resource_like").
		Where("galgame_resource_id = ? AND user_id = ?", resourceID, userID).
		Count(&n).Error
	return n > 0, err
}

func (s *ResourceV1Store) LikeCount(resourceID int) (int, error) {
	var n int
	err := s.db.Raw(`SELECT like_count FROM galgame_resource WHERE id = ?`, resourceID).Scan(&n).Error
	return n, err
}

func (s *ResourceV1Store) Create(tx *gorm.DB, row *model.GalgameResource) error {
	return tx.Create(row).Error
}

func (s *ResourceV1Store) ReplaceProviders(tx *gorm.DB, resourceID int, providers []string) error {
	return tx.Exec(
		"UPDATE galgame_resource SET provider = ?::text[] WHERE id = ?",
		pgTextArrayLiteral(providers), resourceID,
	).Error
}

func (s *ResourceV1Store) ReplaceProviderNames(tx *gorm.DB, resourceID int, names []string) error {
	if names == nil {
		names = []string{}
	}
	encoded, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Exec(
		"UPDATE galgame_resource SET provider_name = ?::jsonb WHERE id = ?",
		string(encoded), resourceID,
	).Error
}

func (s *ResourceV1Store) CreateLinks(tx *gorm.DB, resourceID int, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	links := make([]model.GalgameResourceLink, len(urls))
	now := time.Now()
	for i, u := range urls {
		links[i] = model.GalgameResourceLink{GalgameResourceID: resourceID, URL: u, CreatedAt: now, UpdatedAt: now}
	}
	return tx.Create(&links).Error
}

func (s *ResourceV1Store) DeleteLinks(tx *gorm.DB, resourceID int) error {
	return tx.Where("galgame_resource_id = ?", resourceID).Delete(&model.GalgameResourceLink{}).Error
}

func (s *ResourceV1Store) Patch(tx *gorm.DB, id int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	res := tx.Model(&model.GalgameResource{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (s *ResourceV1Store) Delete(tx *gorm.DB, id int) error {
	res := tx.Where("id = ?", id).Delete(&model.GalgameResource{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrResourceNotFound
	}
	return nil
}

func (s *ResourceV1Store) AdjustResourceCount(tx *gorm.DB, workID, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE galgame SET resource_count = resource_count + ? WHERE id = ?`, delta, workID).Error
	}
	return tx.Exec(`UPDATE galgame SET resource_count = GREATEST(resource_count + ?, 0) WHERE id = ?`, delta, workID).Error
}

func (s *ResourceV1Store) TouchResourceUpdate(tx *gorm.DB, workID int) error {
	return tx.Exec(`UPDATE galgame SET resource_update_time = ? WHERE id = ?`, time.Now(), workID).Error
}

func returningInt64(tx *gorm.DB, q string, args ...any) (int64, bool, error) {
	rows, err := tx.Raw(q, args...).Rows()
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, false, rows.Err()
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, false, err
	}
	return id, true, rows.Err()
}

func (s *ResourceV1Store) InsertLike(tx *gorm.DB, resourceID, userID int) (int64, bool, error) {
	return returningInt64(tx, `
		INSERT INTO galgame_resource_like (galgame_resource_id, user_id, created, updated)
		VALUES (?, ?, now(), now())
		ON CONFLICT (galgame_resource_id, user_id) DO NOTHING
		RETURNING id`, resourceID, userID)
}

func (s *ResourceV1Store) DeleteLike(tx *gorm.DB, resourceID, userID int) (int64, bool, error) {
	return returningInt64(tx, `
		DELETE FROM galgame_resource_like
		WHERE galgame_resource_id = ? AND user_id = ?
		RETURNING id`, resourceID, userID)
}

func (s *ResourceV1Store) AdjustLikeCount(tx *gorm.DB, resourceID, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE galgame_resource SET like_count = like_count + ? WHERE id = ?`, delta, resourceID).Error
	}
	return tx.Exec(`UPDATE galgame_resource SET like_count = GREATEST(like_count + ?, 0) WHERE id = ?`, delta, resourceID).Error
}

func (s *ResourceV1Store) SetStatus(tx *gorm.DB, resourceID, status int) error {
	return tx.Exec(`UPDATE galgame_resource SET status = ? WHERE id = ?`, status, resourceID).Error
}

func (s *ResourceV1Store) CreateGalgameMessage(tx *gorm.DB, senderID, receiverID int, msgType, content string, workID int) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	link := "/galgame/" + strconv.Itoa(workID)
	var count int64
	if err := tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: content, Link: link, Status: "unread",
	}).Error
}

func (s *ResourceV1Store) PutPublishBan(workID int) error {
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{"resource_publish_banned": true}),
	}).Create(&model.GalgameLocal{ID: workID, ResourcePublishBanned: true}).Error
}

func (s *ResourceV1Store) ClearPublishBan(workID int) error {
	res := s.db.Table("galgame").Where("id = ?", workID).
		Update("resource_publish_banned", false)
	return res.Error
}


