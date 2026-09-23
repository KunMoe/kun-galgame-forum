package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/problem"
)

type getWorkInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id, which is also the forum galgame page id."`
}

type getWorkOutput struct {
	Body repr.WorkRef
}

func (s *Service) getWork(ctx context.Context, in *getWorkInput) (*getWorkOutput, error) {
	if s == nil || s.works == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	workID, ok := repr.ParseID(repr.DecimalID(in.WorkID))
	if !ok {
		return nil, notFound()
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, []int{workID}, "names,covers", "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	row, ok := rows[workID]
	if !ok || !client.CatalogItemRenderable(&row) {
		return nil, notFound()
	}
	ref := WorkRefOf(ctx, &row, s.cdn)
	return &getWorkOutput{Body: ref}, nil
}
