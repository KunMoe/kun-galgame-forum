package apiv1

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/pkg/catalogclient"
)

// Catalog's per-uid bucket is 100 calls a minute across every app, and one
// /me/playtimes read sweeps up to 20 pages of it. Paging through the list must
// not re-sweep.
const myPlaytimesCacheTTL = time.Minute

type playtimeSweep struct {
	Playtimes []catalogclient.PlaytimeRecord  `json:"p"`
	States    []catalogclient.WorkStateRecord `json:"s"`
	Truncated bool                            `json:"t"`
}

func myPlaytimesCacheKey(userID int) string {
	return "kungal:me-playtimes:v1:" + strconv.Itoa(userID)
}

func (s *Service) myPlaytimeSweep(ctx context.Context, userID int, token string) (*playtimeSweep, error) {
	if s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, myPlaytimesCacheKey(userID)).Bytes(); err == nil {
			var cached playtimeSweep
			if json.Unmarshal(raw, &cached) == nil {
				return &cached, nil
			}
		}
	}
	playtimes, playTrunc, err := s.sweepPlaytimes(ctx, token)
	if err != nil {
		return nil, err
	}
	states, stateTrunc, err := s.sweepWorkStates(ctx, token)
	if err != nil {
		return nil, err
	}
	if playTrunc || stateTrunc {
		slog.Warn("me playtimes: sweep hit page cap", "pages", playtimeSweepPages, "playtime_truncated", playTrunc, "state_truncated", stateTrunc)
	}
	sweep := &playtimeSweep{Playtimes: playtimes, States: states, Truncated: playTrunc || stateTrunc}
	if s.rdb != nil {
		if raw, err := json.Marshal(sweep); err == nil {
			if err := s.rdb.Set(ctx, myPlaytimesCacheKey(userID), raw, myPlaytimesCacheTTL).Err(); err != nil {
				slog.Warn("me playtimes: cache write failed", "user_id", userID, "err", err)
			}
		}
	}
	return sweep, nil
}

func (s *Service) dropMyPlaytimeSweep(ctx context.Context, userID int) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.Del(context.WithoutCancel(ctx), myPlaytimesCacheKey(userID)).Err(); err != nil {
		slog.Warn("me playtimes: cache drop failed", "user_id", userID, "err", err)
	}
}
