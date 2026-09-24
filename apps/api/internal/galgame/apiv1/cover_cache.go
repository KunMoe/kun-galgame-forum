package apiv1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/pkg/catalogclient"
)

const (
	coverTalliesTTL = time.Minute
	myCoverVotesTTL = 10 * time.Minute
)

func coverTalliesKey(workID int) string { return "kungal:cover-tallies:v1:" + strconv.Itoa(workID) }

// Keyed by token, not only by user: /v2/me/cover-votes needs catalog:edit, and
// a per-user entry would hand one session's votes to another session of the
// same person whose token was never granted it.
func myCoverVotesKey(userID int, token string) string {
	sum := sha256.Sum256([]byte(token))
	return "kungal:my-cover-votes:v2:" + strconv.Itoa(userID) + ":" + hex.EncodeToString(sum[:8])
}

// Tallies are the same for every reader, so they are read with the app key.
func (s *Service) coverTallies(ctx context.Context, workID int) []catalogclient.CoverTally {
	if s.catalog == nil {
		return nil
	}
	var cached []catalogclient.CoverTally
	if s.cacheGet(ctx, coverTalliesKey(workID), &cached) {
		return cached
	}
	tallies, err := s.catalog.WorkCoverVotes(ctx, int64(workID))
	if err != nil {
		slog.Warn("galgame detail: cover vote tallies unavailable", "work_id", workID, "upstream_status", upstreamStatus(err), "err", err)
		return nil
	}
	s.cacheSet(ctx, coverTalliesKey(workID), tallies, coverTalliesTTL)
	return tallies
}

// Catalog lists every vote the caller ever cast in one unpaginated page (one
// ballot per work; prod max 6 on 2026-09-24), so one read serves every work.
func (s *Service) myCoverVotes(ctx context.Context, userID int, token string) ([]catalogclient.MyCoverVote, error) {
	var cached []catalogclient.MyCoverVote
	if s.cacheGet(ctx, myCoverVotesKey(userID, token), &cached) {
		return cached, nil
	}
	votes, err := s.catalog.MyCoverVotes(ctx, token)
	if err != nil {
		return nil, err
	}
	s.cacheSet(ctx, myCoverVotesKey(userID, token), votes, myCoverVotesTTL)
	return votes, nil
}

func (s *Service) dropCoverVoteCaches(ctx context.Context, userID int, token string, workID int) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.Del(context.WithoutCancel(ctx), coverTalliesKey(workID), myCoverVotesKey(userID, token)).Err(); err != nil {
		slog.Warn("work cover vote: cache drop failed", "work_id", workID, "user_id", userID, "err", err)
	}
}

func (s *Service) cacheGet(ctx context.Context, key string, out any) bool {
	if s.rdb == nil {
		return false
	}
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, out) == nil
}

func (s *Service) cacheSet(ctx context.Context, key string, v any, ttl time.Duration) {
	if s.rdb == nil {
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	if err := s.rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		slog.Warn("galgame: cache write failed", "key", key, "err", err)
	}
}
