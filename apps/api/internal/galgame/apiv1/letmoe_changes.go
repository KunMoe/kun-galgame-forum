package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/letmoeclient"

	"github.com/redis/go-redis/v9"
)

const (
	letmoeChangesCursorKey = "letmoe:changes:cursor"
	letmoeChangesInterval  = 10 * time.Second
	letmoeChangesLimit     = 100
	letmoeChangesMaxPages  = 50
	letmoeChangesTickWait  = time.Minute
)

// StartLetmoeChanges drops a work's cached letmoe list as soon as letmoe's
// changes feed names it, so an upload shows here in seconds rather than when
// letmoeCacheTTL runs out.
func (s *Service) StartLetmoeChanges() func() {
	if s == nil || !s.letmoe.Configured() || s.rdb == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.RunLetmoeChanges(ctx)
		t := time.NewTicker(letmoeChangesInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.RunLetmoeChanges(ctx)
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func (s *Service) RunLetmoeChanges(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, letmoeChangesTickWait)
	defer cancel()

	cursor, err := s.rdb.Get(runCtx, letmoeChangesCursorKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		slog.Warn("letmoe changes: cursor read failed", "error", err)
		return
	}
	for range letmoeChangesMaxPages {
		page, err := s.letmoe.Changes(runCtx, cursor, letmoeChangesLimit)
		if errors.Is(err, letmoeclient.ErrInvalidCursor) {
			s.letmoeFetchErr = ""
			slog.Warn("letmoe changes: cursor rejected, dropping every cached letmoe list", "cursor", cursor)
			s.dropAllLetmoeLists(runCtx)
			return
		}
		if err != nil {
			// 127.0.0.1:9291 refused the connection on every 10s tick (2026-11-02),
			// and each warning repeated the whole URL.
			if err.Error() != s.letmoeFetchErr {
				s.letmoeFetchErr = err.Error()
				slog.Warn("letmoe changes: page fetch failed", "error", err)
			}
			return
		}
		if s.letmoeFetchErr != "" {
			s.letmoeFetchErr = ""
			slog.Info("letmoe changes: page fetch recovered")
		}
		keys := make([]string, 0, len(page.Items))
		for _, c := range page.Items {
			keys = append(keys, letmoeCacheKey+c.WorkID)
		}
		if len(keys) > 0 {
			if err := s.rdb.Del(runCtx, keys...).Err(); err != nil {
				slog.Warn("letmoe changes: cache drop failed", "error", err)
				return
			}
		}
		cursor = page.NextCursor
		if err := s.rdb.Set(runCtx, letmoeChangesCursorKey, cursor, 0).Err(); err != nil {
			slog.Warn("letmoe changes: cursor write failed", "error", err)
			return
		}
		if len(page.Items) < letmoeChangesLimit {
			return
		}
	}
}

func (s *Service) dropAllLetmoeLists(ctx context.Context) {
	iter := s.rdb.Scan(ctx, 0, letmoeCacheKey+"*", 500).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) == 500 {
			s.rdb.Unlink(ctx, keys...)
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		s.rdb.Unlink(ctx, keys...)
	}
	if err := iter.Err(); err != nil {
		slog.Warn("letmoe changes: cache scan failed", "error", err)
		return
	}
	s.rdb.Del(ctx, letmoeChangesCursorKey)
}
