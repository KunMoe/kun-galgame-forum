package repository

import (
	"time"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

func (s *WorkV1Store) CreatorsOf(workIDs []int) (map[int]int, error) {
	out := make(map[int]int, len(workIDs))
	if len(workIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID            int
		CreatorUserID int
	}
	err := s.db.Table("galgame").Select("id, creator_user_id").
		Where("id IN ? AND creator_user_id IS NOT NULL", workIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.CreatorUserID
	}
	return out, nil
}

func (s *WorkV1Store) TouchEdited(workID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.EnsureLocal(tx, workID); err != nil {
			return err
		}
		return tx.Model(&model.GalgameLocal{}).Where("id = ?", workID).
			UpdateColumn("resource_update_time", time.Now()).Error
	})
}

// wiki_pr_id holds the catalog proposal id since the cutover; the column keeps
// its old name (G7 §8 O2).
func (s *WorkV1Store) InsertProposalActivity(proposalID int64, workID, userID int) error {
	return s.db.Exec(`
		INSERT INTO galgame_activity (wiki_pr_id, work_id, user_id, type, created)
		VALUES (?, ?, ?, 'GALGAME_PR_CREATION', now())
		ON CONFLICT (wiki_pr_id) DO NOTHING
	`, proposalID, workID, userID).Error
}
