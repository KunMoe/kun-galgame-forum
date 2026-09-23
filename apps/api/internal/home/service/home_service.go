package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/home/dto"
	"kun-galgame-api/internal/home/repository"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

const (
	homeGalgameLimit      = 12
	homeGalgameFetchLimit = 24
	homeTopicLimit        = 10
)

const homeCacheTTL = 60 * time.Second

type HomeService struct {
	repo          *repository.HomeRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
	rdb           *redis.Client
}

func NewHomeService(
	repo *repository.HomeRepository,
	gc *client.GalgameClient,
	userClient *userclient.Client,
	rdb *redis.Client,
) *HomeService {
	return &HomeService{repo: repo, galgameClient: gc, userClient: userClient, rdb: rdb}
}

func (s *HomeService) GetHome(ctx context.Context, isSFW bool) (*dto.HomeResponse, error) {
	cacheKey := fmt.Sprintf("home:v1:%t", isSFW)
	if cached := s.getCachedHome(ctx, cacheKey); cached != nil {
		return cached, nil
	}

	galgames, gErr := s.getHomeGalgames(ctx, isSFW)
	if gErr != nil {
		slog.Warn("首页 galgame 获取失败, 降级为空列表", "error", gErr)
		galgames = []dto.HomeGalgame{}
	}
	topics, err := s.getHomeTopics(ctx, isSFW)
	if err != nil {
		return nil, err
	}
	resp := &dto.HomeResponse{Galgames: galgames, Topics: topics}
	if gErr == nil {
		s.cacheHome(ctx, cacheKey, resp)
	}
	return resp, nil
}

func (s *HomeService) getCachedHome(ctx context.Context, key string) *dto.HomeResponse {
	if s.rdb == nil {
		return nil
	}
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}
	var resp dto.HomeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	return &resp
}

func (s *HomeService) cacheHome(ctx context.Context, key string, resp *dto.HomeResponse) {
	if s.rdb == nil || resp == nil {
		return
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, key, raw, homeCacheTTL).Err()
}

func (s *HomeService) getHomeGalgames(ctx context.Context, isSFW bool) ([]dto.HomeGalgame, error) {
	localRows, err := s.repo.FindRecentGalgames(homeGalgameFetchLimit)
	if err != nil {
		return nil, err
	}
	if len(localRows) == 0 {
		return []dto.HomeGalgame{}, nil
	}

	workIDs := make([]int, len(localRows))
	for i, r := range localRows {
		workIDs[i] = r.ID
	}
	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, workIDs, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	userMap := s.userClient.Hydrate(ctx, userclient.CollectIDs(localRows,
		func(lr repository.GalgameLocalRow) int { return userclient.DerefID(lr.CreatorUserID) }))

	resources := s.repo.FindResourcePlatformLanguage(workIDs)
	platformMap := map[int]map[string]bool{}
	languageMap := map[int]map[string]bool{}
	for _, r := range resources {
		if platformMap[r.WorkID] == nil {
			platformMap[r.WorkID] = map[string]bool{}
		}
		if languageMap[r.WorkID] == nil {
			languageMap[r.WorkID] = map[string]bool{}
		}
		platformMap[r.WorkID][r.Platform] = true
		languageMap[r.WorkID][r.Language] = true
	}

	result := make([]dto.HomeGalgame, 0, homeGalgameLimit)
	for _, lr := range localRows {
		if len(result) >= homeGalgameLimit {
			break
		}
		b, ok := briefMap[lr.ID]
		if !ok {
			continue
		}
		u := userMap[userclient.DerefID(lr.CreatorUserID)]
		result = append(result, dto.HomeGalgame{
			ID:                  lr.ID,
			Name:                b.Name,
			User:                dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ContentLimit:        b.ContentLimit,
			View:                lr.View,
			LikeCount:           lr.LikeCount,
			ResourceUpdateTime:  lr.ResourceUpdateTime.Format(time.RFC3339),
			Platform:            mapKeys(platformMap[lr.ID]),
			Language:            mapKeys(languageMap[lr.ID]),
			EffectiveBannerHash: b.EffectiveBannerHash,
			EffectiveBannerURL:  b.EffectiveBannerURL,
		})
	}

	return result, nil
}

func (s *HomeService) getHomeTopics(ctx context.Context, isSFW bool) ([]dto.HomeTopic, error) {
	rows, err := s.repo.FindHomeTopics(homeTopicLimit, isSFW)
	if err != nil {
		return nil, err
	}

	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}

	sections := s.repo.FindTopicSections(topicIDs)
	sectionMap := map[int][]string{}
	for _, sct := range sections {
		sectionMap[sct.TopicID] = append(sectionMap[sct.TopicID], sct.SectionName)
	}

	miniApps := s.repo.FindTopicMiniApps(topicIDs)

	uids := userclient.CollectIDs(rows, func(r repository.TopicRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	result := make([]dto.HomeTopic, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		topicSections := sectionMap[r.ID]
		if topicSections == nil {
			topicSections = []string{}
		}

		result = append(result, dto.HomeTopic{
			ID:               r.ID,
			Title:            r.Title,
			View:             r.View,
			LikeCount:        r.LikeCount,
			ReplyCount:       r.ReplyCount,
			CommentCount:     r.CommentCount,
			HasBestAnswer:    r.BestAnswerID != nil,
			MiniApps:         miniApps[r.ID],
			IsNSFWTopic:      r.IsNSFW,
			Section:          topicSections,
			User:             dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Status:           r.Status,
			UpvoteTime:       r.UpvoteTime,
			StatusUpdateTime: r.StatusUpdateTime,
		})
	}

	return result, nil
}

func mapKeys(m map[string]bool) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
