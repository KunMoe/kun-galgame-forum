package repository

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
