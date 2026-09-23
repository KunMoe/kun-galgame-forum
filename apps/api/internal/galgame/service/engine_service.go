package service

import (
	"context"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type EngineService struct {
	galgameClient *client.GalgameClient
	galgameSvc    *GalgameService
}

func NewEngineService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *EngineService {
	return &EngineService{galgameClient: galgameClient, galgameSvc: galgameSvc}
}

const engineIndexPageCap = 20

func (s *EngineService) GetList(ctx context.Context) ([]dto.EngineListItem, *errors.AppError) {
	items := []dto.EngineListItem{}
	cursor := ""
	for page := 0; page < engineIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.galgameClient.CatalogTaxonomyList(ctx, "engines", q)
		if appErr != nil {
			return nil, appErr
		}
		for _, e := range res.Items {
			name := e.Label(ctx)
			items = append(items, dto.EngineListItem{
				ID:           int(e.ID),
				Name:         name,
				Description:  e.Description,
				Alias:        e.Aliases.Values(name),
				GalgameCount: e.WorkCount,
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	return items, nil
}

func (s *EngineService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.EngineDetail, *errors.AppError) {
	e, found, appErr := s.galgameClient.CatalogEngine(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该引擎")
	}

	filter := buildEntityFilter(rawQuery)
	memberIDs, appErr := s.galgameClient.CatalogMemberWorkIDs(ctx,
		entityMemberQuery("engine_id", id, filter), isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	filter.RestrictIDs = memberIDs
	page, appErr := s.galgameSvc.hydrateListCards(ctx, filter, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	name := e.Label(ctx)
	return &dto.EngineDetail{
		ID:           int(e.ID),
		Name:         name,
		Description:  e.Description,
		Alias:        e.Aliases.Values(name),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

func emptyStrSliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
