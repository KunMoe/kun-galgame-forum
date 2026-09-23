package service

import (
	"context"

	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/repository"
	userModel "kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/userclient"
)

type ToolsetService struct {
	toolsetRepo      *repository.ToolsetRepository
	resourceRepo     *repository.ResourceRepository
	practicalityRepo *repository.PracticalityRepository
	userClient       *userclient.Client
}

func NewToolsetService(
	toolsetRepo *repository.ToolsetRepository,
	resourceRepo *repository.ResourceRepository,
	practicalityRepo *repository.PracticalityRepository,
	userClient *userclient.Client,
) *ToolsetService {
	return &ToolsetService{
		toolsetRepo:      toolsetRepo,
		resourceRepo:     resourceRepo,
		practicalityRepo: practicalityRepo,
		userClient:       userClient,
	}
}

func userBriefFromClient(u userclient.User) userModel.UserBrief {
	return userModel.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

func (s *ToolsetService) GetList(ctx context.Context, req *dto.ToolsetListRequest) ([]dto.ToolsetCard, int64) {
	filters := repository.ListFilters{
		Type:     req.Type,
		Language: req.Language,
		Platform: req.Platform,
		Version:  req.Version,
		UserID:   req.UserID,
		Query:    req.Query,
	}
	total := s.toolsetRepo.CountFiltered(filters)

	opts := repository.ListOptions{
		SortField: allowedSortField(req.SortField),
		SortOrder: sortOrder(req.SortOrder),
		Offset:    (req.Page - 1) * req.Limit,
		Limit:     req.Limit,
	}
	toolsets := s.toolsetRepo.ListFiltered(filters, opts)

	toolsetIDs := make([]int, len(toolsets))
	userIDs := make([]int, len(toolsets))
	for i, t := range toolsets {
		toolsetIDs[i] = t.ID
		userIDs[i] = t.UserID
	}

	avgMap := s.practicalityRepo.AveragesForToolsets(toolsetIDs)
	dlMap := s.resourceRepo.DownloadSumsForToolsets(toolsetIDs)
	ccMap := make(map[int]int, len(toolsets))
	for _, t := range toolsets {
		ccMap[t.ID] = t.CommentCount
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.ToolsetCard, 0, len(toolsets))
	for _, t := range toolsets {
		if !userclient.IsRenderable(userMap[t.UserID]) {
			continue
		}
		cards = append(cards, toolsetCardFromRow(t, userMap, avgMap, dlMap, ccMap))
	}

	return cards, total
}
