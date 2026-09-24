package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// SubmitLocal stamps the submitter at mint time rather than when the claim feed
// reports it ten minutes later: the stamp is what lets the submitter open their
// own unpublished entry. The row stays published=false.
func (s *WorkV1Store) SubmitLocal(workID, userID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.EnsureLocal(tx, workID); err != nil {
			return err
		}
		return tx.Model(&model.GalgameLocal{}).
			Where("id = ? AND creator_user_id IS NULL", workID).
			UpdateColumn("creator_user_id", userID).Error
	})
}

// DeleteLocalDraft is the only cleanup a deleted draft will ever get — catalog's
// delete writes no claim event, so no cron comes along behind it. It refuses
// rather than cascades: galgame_resource is ON DELETE CASCADE, and a draft claim
// carrying a published resource is reachable, because publishing a resource sets
// `published` without moving the claim state.
func (s *WorkV1Store) DeleteLocalDraft(workID int) error {
	return s.db.Exec(`DELETE FROM galgame WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM galgame_resource r WHERE r.work_id = galgame.id)`,
		workID).Error
}
