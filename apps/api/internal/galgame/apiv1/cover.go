package apiv1

import (
	"context"
	"log/slog"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/problem"
)

type workCoverInput struct {
	WorkID  string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	CoverID string `path:"cover_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog cover row id."`
}

type workCoverOutput struct {
	Body WorkCover
}

type workCoverEngagementOutput struct {
	Body WorkCoverEngagement
}

func (s *Service) getWorkCover(ctx context.Context, in *workCoverInput) (*workCoverOutput, error) {
	workID, coverID, d, p := s.coverOnWork(ctx, in.WorkID, in.CoverID)
	if p != nil {
		return nil, p
	}
	user := v1.User(ctx)
	var viewer *WorkCoverViewer
	if user != nil {
		viewer = &WorkCoverViewer{}
	}
	tallies := s.coverTallies(ctx, workID, accessToken(ctx))
	if tallies == nil && s.catalog != nil {
		slog.Warn("work cover: vote tallies unavailable", "work_id", workID, "cover_id", coverID)
	}
	covers := coversOf(d, s.cdn, tallies, viewer)
	for _, c := range covers {
		id, ok := repr.ParseID(c.ID)
		if ok && id == coverID {
			return &workCoverOutput{Body: c}, nil
		}
	}
	return nil, notFound()
}

func (s *Service) putWorkCoverVote(ctx context.Context, in *workCoverInput) (*workCoverEngagementOutput, error) {
	return s.castCoverVote(ctx, in, true)
}

func (s *Service) deleteWorkCoverVote(ctx context.Context, in *workCoverInput) (*workCoverEngagementOutput, error) {
	return s.castCoverVote(ctx, in, false)
}

func (s *Service) castCoverVote(ctx context.Context, in *workCoverInput, vote bool) (*workCoverEngagementOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	workID, coverID, _, p := s.coverOnWork(ctx, in.WorkID, in.CoverID)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	var err error
	if vote {
		_, err = s.catalog.VoteCover(ctx, token, int64(workID), int64(coverID))
	} else {
		_, err = s.catalog.UnvoteCover(ctx, token, int64(workID), int64(coverID))
	}
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	tallies, err := s.catalog.WorkCoverVotes(ctx, int64(workID))
	if err != nil {
		slog.Warn("work cover vote: public tallies unavailable after write",
			"work_id", workID, "cover_id", coverID, "upstream_status", upstreamStatus(err), "err", err)
		return nil, problem.Unavailable(err)
	}
	count := 0
	hasVoted := false
	for _, t := range tallies {
		if int(t.ID) == coverID {
			count = max(t.VoteCount, 0)
			hasVoted = t.Voted
			break
		}
	}
	if vote {
		hasVoted = true
	}
	return &workCoverEngagementOutput{Body: WorkCoverEngagement{
		Object: "work_cover_engagement", WorkID: repr.ID(workID), CoverID: repr.ID(coverID),
		VoteCount: count, Viewer: &WorkCoverEngagementViewer{HasVoted: hasVoted},
	}}, nil
}

func (s *Service) coverOnWork(ctx context.Context, rawWork, rawCover string) (int, int, *client.CatalogWorkDetail, *problem.Problem) {
	workID, ok := parseWorkID(rawWork)
	if !ok {
		return 0, 0, nil, notFound()
	}
	coverID, ok := parseWorkID(rawCover)
	if !ok {
		return 0, 0, nil, notFound()
	}
	detail := s.detailCatalog()
	if detail == nil {
		return 0, 0, nil, problem.Internal(errUnconfigured)
	}
	d, found, movedTo, appErr := detail.CatalogWorkDetail(ctx, workID)
	if appErr != nil {
		return 0, 0, nil, catalogUnavailable(appErr)
	}
	if movedTo != 0 {
		return 0, 0, nil, mergedWork(movedTo)
	}
	if !found || d == nil {
		return 0, 0, nil, notFound()
	}
	for _, c := range d.Covers {
		if int(c.ID) == coverID {
			return workID, coverID, d, nil
		}
	}
	return 0, 0, nil, notFound()
}
