package handler

import (
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/rss/dto"
	"kun-galgame-api/internal/rss/repository"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v3"
)

type RSSHandler struct {
	repo          *repository.RSSRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
}

func NewRSSHandler(
	repo *repository.RSSRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
) *RSSHandler {
	return &RSSHandler{repo: repo, galgameClient: galgameClient, userClient: userClient}
}

func (h *RSSHandler) GetTopicRSS(c fiber.Ctx) error {
	rows := h.repo.FindRecentSFWTopics()
	uids := userclient.CollectIDs(rows, func(r dto.TopicRSSItem) int { return r.UserID })
	userMap := h.userClient.Hydrate(c.Context(), uids)
	items := make([]dto.TopicRSSItem, 0, len(rows))
	for i := range rows {
		u := userMap[rows[i].UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		rows[i].UserName = u.Name
		items = append(items, rows[i])
	}
	return response.OK(c, items)
}

func (h *RSSHandler) GetGalgameRSS(c fiber.Ctx) error {
	rows := h.repo.FindRecentWorkIDs(10)
	if len(rows) == 0 {
		return response.OK(c, []dto.GalgameRSSItem{})
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}

	briefMap, _ := h.galgameClient.GetBatchPublic(c.Context(), ids, true)
	if briefMap == nil {
		briefMap = map[int]client.GalgameBrief{}
	}

	userMap := h.userClient.Hydrate(c.Context(), userclient.CollectIDs(rows,
		func(r repository.RecentGalgameRow) int { return userclient.DerefID(r.CreatorUserID) }))

	items := make([]dto.GalgameRSSItem, 0, len(rows))
	for _, row := range rows {
		b, ok := briefMap[row.ID]
		if !ok {
			continue
		}
		u := userMap[userclient.DerefID(row.CreatorUserID)]
		items = append(items, dto.GalgameRSSItem{
			ID:     row.ID,
			Name:   b.Name,
			Banner: b.EffectiveBannerURL,
			User: dto.GalgameRSSUser{
				ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			},
			Description: "",
			Created:     row.Created,
		})
	}
	return response.OK(c, items)
}
