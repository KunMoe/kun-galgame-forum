package service

import "context"

func (s *UserService) ProfileSignInRequired(ctx context.Context, userID int) (bool, error) {
	return s.stateRepo.ProfileSignInRequired(ctx, userID)
}

func (s *UserService) SetProfileSignInRequired(userID int, required bool) error {
	if err := s.stateRepo.Ensure(userID); err != nil {
		return err
	}
	return s.stateRepo.SetProfileSignInRequired(userID, required)
}
