package service

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/moemoepoint"
)

var (
	ErrAlreadyCheckedIn = errors.New("already checked in today")
	ErrUpstream         = errors.New("account service unavailable")
)

type CheckInResult struct {
	Date    string
	Awarded int
	Balance int
}

func (s *UserService) CheckIn(_ context.Context, userID int) (CheckInResult, error) {
	// Check-in date is the Asia/Shanghai calendar day; UTC keys missed ~20% of daily check-ins.
	date := s.now().In(cron.ScheduleLocation()).Format("2006-01-02")
	key := moemoepoint.Key("daily_checkin", fmt.Sprintf("%d_%s", userID, date))
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	delta := int(h.Sum32() % uint32(constants.CheckinMaxReward+1))

	if err := s.stateRepo.Ensure(userID); err != nil {
		return CheckInResult{}, err
	}
	applied, err := s.stateRepo.CheckIn(userID)
	if err != nil {
		return CheckInResult{}, err
	}
	if !applied {
		return CheckInResult{}, ErrAlreadyCheckedIn
	}

	if delta != 0 {
		if err := moemoepoint.AwardSync(userID, delta, moemoepoint.ReasonDailyCheckin, "", key); err != nil {
			if rerr := s.stateRepo.ResetDailyCheckIn(userID); rerr != nil {
				slog.Error("check-in gate restore failed", "user_id", userID, "err", rerr)
			}
			return CheckInResult{}, fmt.Errorf("%w: %w", ErrUpstream, err)
		}
	}

	state, err := s.stateRepo.FindByID(userID)
	if err != nil {
		return CheckInResult{}, err
	}
	return CheckInResult{Date: date, Awarded: delta, Balance: state.Moemoepoint}, nil
}
