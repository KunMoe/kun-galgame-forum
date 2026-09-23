package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"
)

type GalgameUserStats struct {
	Published      int64 `json:"published"`
	PublishedToday int   `json:"published_today"`
	Contributed    int   `json:"contributed"`
	MergedEdits    int64 `json:"merged_edits"`
}

type GalgameUserStatsService struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
}

func NewGalgameUserStatsService(
	catalog *catalogclient.Client,
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
) *GalgameUserStatsService {
	return &GalgameUserStatsService{
		catalog: catalog, galgameClient: galgameClient, galgameRepo: galgameRepo,
	}
}

const contributedScan = 200

func (s *GalgameUserStatsService) Stats(ctx context.Context, uid int64) GalgameUserStats {
	var out GalgameUserStats

	if _, total, err := s.galgameRepo.PublishedIDsByCreator(int(uid), 1, 1); err != nil {
		slog.Warn("user stats: 读取已发布计数失败", "uid", uid, "error", err)
	} else {
		out.Published = total
	}
	out.PublishedToday = s.galgameRepo.CountPublishedByCreatorSince(int(uid), startOfToday())

	if s.catalog == nil || !s.catalog.AppConfigured() {
		return out
	}

	if total, err := s.catalog.CountEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid, Status: "merged",
	}); err != nil {
		slog.Warn("user stats: 读取合并提案计数失败", "uid", uid, "error", err)
	} else {
		out.MergedEdits = total
	}

	if items, err := s.catalog.ListEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid,
		Status: "merged", Limit: contributedScan,
	}); err != nil {
		slog.Warn("user stats: 读取贡献条目失败", "uid", uid, "error", err)
	} else {
		out.Contributed = len(distinctEntityIDs(items))
	}
	return out
}

func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func distinctEntityIDs(items []catalogclient.EditProposal) []int64 {
	seen := make(map[int64]struct{}, len(items))
	out := make([]int64, 0, len(items))
	for i := range items {
		id := items[i].EntityID
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *GalgameUserStatsService) PublishedWorkIDs(
	_ context.Context, uid int64, page, limit int,
) ([]int, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.galgameRepo.PublishedIDsByCreator(int(uid), page, limit)
}

func (s *GalgameUserStatsService) ContributedWorkIDs(ctx context.Context, uid int64) ([]int, error) {
	items, err := s.catalog.ListEditProposals(ctx, catalogclient.EditProposalFilter{
		EntityType: catalogclient.EntityTypeWork, ProposerUID: uid,
		Status: "merged", Limit: contributedScan,
	})
	if err != nil {
		return nil, err
	}
	workIDs := distinctEntityIDs(items)
	ids := make([]int, 0, len(workIDs))
	for _, id := range workIDs {
		if id > 0 {
			ids = append(ids, int(id))
		}
	}
	return ids, nil
}
