package service

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type GalgameProxyService struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
	userClient    *userclient.Client
}

func NewGalgameProxyService(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	userClient *userclient.Client,
) *GalgameProxyService {
	return &GalgameProxyService{galgameClient: galgameClient, galgameRepo: galgameRepo, userClient: userClient}
}

func (s *GalgameProxyService) GetGalgameLinks(
	ctx context.Context,
	workID string,
) ([]dto.GalgameLink, *errors.AppError) {
	gidInt, err := strconv.Atoi(workID)
	if err != nil || gidInt <= 0 {
		return nil, errors.ErrBadRequest("无效的 Galgame ID")
	}
	rows, appErr := s.galgameClient.CatalogWorkLinks(ctx, gidInt)
	if appErr != nil {
		return nil, appErr
	}
	out := make([]dto.GalgameLink, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.GalgameLink{
			WorkID: gidInt,
			Name:   r.Name,
			Link:   r.Link,
		})
	}
	return out, nil
}
