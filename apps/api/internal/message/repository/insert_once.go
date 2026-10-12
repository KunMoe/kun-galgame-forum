package repository

import (
	"kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
)

func InsertOnce(tx *gorm.DB, m *model.Message) error {
	var n int64
	if err := tx.Model(&model.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND subject_kind = ? AND subject_id = ?",
			m.SenderID, m.ReceiverID, m.Type, m.SubjectKind, m.SubjectID).
		Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return tx.Create(m).Error
}
