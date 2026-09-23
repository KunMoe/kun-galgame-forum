package service

import (
	"errors"

	"gorm.io/gorm"
)

func (s *UserService) MutedNotificationTypes(userID int) ([]string, error) {
	state, err := s.stateRepo.FindByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if state.MutedNotificationTypes == nil {
		return []string{}, nil
	}
	return state.MutedNotificationTypes, nil
}

func (s *UserService) ReplaceMutedNotificationTypes(userID int, dbKeys []string) error {
	if err := s.stateRepo.Ensure(userID); err != nil {
		return err
	}
	return s.stateRepo.UpdateMutedTypes(userID, dbKeys)
}
