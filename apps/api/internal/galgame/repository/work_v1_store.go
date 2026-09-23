package repository

import (
	"strconv"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/viewstats"
	msgModel "kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkV1Store struct {
	db *gorm.DB
}

func NewWorkV1Store(db *gorm.DB) *WorkV1Store {
	return &WorkV1Store{db: db}
}

func (s *WorkV1Store) Ready() bool {
	return s != nil && s.db != nil
}

func (s *WorkV1Store) InTx(fn func(tx *gorm.DB) error) error {
	return s.db.Transaction(fn)
}

func (s *WorkV1Store) IncrementView(workID int) error {
	res := s.db.Exec(`UPDATE galgame SET view = view + 1 WHERE id = ?`, workID)
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}
	return viewstats.BumpDaily(s.db, viewstats.GalgameDaily, workID)
}

type workLocal struct {
	ID                    int  `gorm:"column:id"`
	View                  int  `gorm:"column:view"`
	LikeCount             int  `gorm:"column:like_count"`
	Published             bool `gorm:"column:published"`
	ResourcePublishBanned bool `gorm:"column:resource_publish_banned"`
	CreatorUserID         *int `gorm:"column:creator_user_id"`
}

func (s *WorkV1Store) FindLocal(workID int) (workLocal, bool, error) {
	var row workLocal
	err := s.db.Table("galgame").
		Select("id, view, like_count, published, resource_publish_banned, creator_user_id").
		Where("id = ?", workID).Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return workLocal{}, false, nil
	}
	return row, err == nil, err
}

func (s *WorkV1Store) EnsureLocal(tx *gorm.DB, workID int) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.GalgameLocal{ID: workID}).Error
}

func (s *WorkV1Store) InsertLike(tx *gorm.DB, userID, workID int) (int64, bool, error) {
	return returningInt64(tx, `
		INSERT INTO galgame_like (user_id, work_id, created, updated)
		VALUES (?, ?, now(), now())
		ON CONFLICT (work_id, user_id) DO NOTHING
		RETURNING id`, userID, workID)
}

func (s *WorkV1Store) DeleteLike(tx *gorm.DB, userID, workID int) (int64, bool, error) {
	return returningInt64(tx, `
		DELETE FROM galgame_like
		WHERE user_id = ? AND work_id = ?
		RETURNING id`, userID, workID)
}

func (s *WorkV1Store) AdjustLikeCount(tx *gorm.DB, workID, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE galgame SET like_count = like_count + ? WHERE id = ?`, delta, workID).Error
	}
	return tx.Exec(`UPDATE galgame SET like_count = GREATEST(like_count + ?, 0) WHERE id = ?`, delta, workID).Error
}

func (s *WorkV1Store) HasLiked(userID, workID int) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var n int64
	err := s.db.Table("galgame_like").
		Where("galgame_like.user_id = ? AND galgame_like.work_id = ?", userID, workID).
		Count(&n).Error
	return n > 0, err
}

func (s *WorkV1Store) LikedSet(userID int, workIDs []int) (map[int]bool, error) {
	out := map[int]bool{}
	if userID == 0 || len(workIDs) == 0 {
		return out, nil
	}
	var liked []int
	err := s.db.Table("galgame_like").
		Where("galgame_like.user_id = ? AND galgame_like.work_id IN ?", userID, workIDs).
		Pluck("galgame_like.work_id", &liked).Error
	for _, id := range liked {
		out[id] = true
	}
	return out, err
}

func (s *WorkV1Store) LikeCount(workID int) (int, error) {
	var n int
	err := s.db.Raw(`SELECT like_count FROM galgame WHERE id = ?`, workID).Scan(&n).Error
	return n, err
}

func (s *WorkV1Store) ContributorIDs(workID, limit int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_contributor").
		Select("galgame_contributor.user_id").
		Where("galgame_contributor.work_id = ?", workID).
		Order("galgame_contributor.revision_count DESC, galgame_contributor.first_at ASC").
		Limit(limit).
		Pluck("galgame_contributor.user_id", &ids).Error
	return ids, err
}

func (s *WorkV1Store) ResourceTypes(workID int) ([]string, error) {
	var types []string
	err := s.db.Table("galgame_resource").
		Where("galgame_resource.work_id = ?", workID).
		Distinct("galgame_resource.type").
		Pluck("galgame_resource.type", &types).Error
	return types, err
}

func (s *WorkV1Store) CreateLikedMessage(tx *gorm.DB, senderID, receiverID int, content string, workID int) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	link := "/galgame/" + strconv.Itoa(workID)
	var count int64
	if err := tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, "liked", link).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "liked", Content: content, Link: link, Status: "unread",
	}).Error
}
