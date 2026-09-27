package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/userclient"
)

// ModerationProfile is getUser's body for an account in any state, plus whether
// the account service still counts it active. It is nil when the account no
// longer exists.
func (s *Users) ModerationProfile(ctx context.Context, userID int) (*UserProfile, bool, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, false, prob
	}
	p, err := s.users.ModerationProfile(ctx, userID)
	if errors.Is(err, service.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	body := mapUserProfile(s.cdn, p)
	body.Counts.FollowerCount, body.Counts.FollowingCount = s.followCounts(ctx, userID)
	return &body, userclient.IsRenderable(p.Account), nil
}
