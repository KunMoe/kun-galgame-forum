package apiv1

import (
	"context"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listGalgameResources(ctx context.Context, in *listGalgameResourcesInput) (*listGalgameResourcesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	filter := repository.ResourceListFilter{
		Q: in.Q, IncludeNSFW: in.IncludeNSFW, State: in.State,
	}
	sort := in.Sort
	if q := strings.TrimSpace(in.Q); q != "" {
		ids, p := s.searchWorkIDs(ctx, q, in.IncludeNSFW)
		if p != nil {
			return nil, p
		}
		filter.Q = q
		filter.CatalogWorkIDs = ids
		if len(ids) > 0 && (sort == "" || sort == "created_desc") {
			sort = "relevance"
		}
	} else {
		filter.Q = ""
	}
	authors, err := s.store.DistinctAuthors(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	keep, p := s.keepAuthors(ctx, authors)
	if p != nil {
		return nil, p
	}
	filter.AuthorIDs = keep
	total, err := s.store.Count(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.List(filter, sort, pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.assemble(ctx, rows, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &listGalgameResourcesOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) listWorkResources(ctx context.Context, in *listWorkResourcesInput) (*listGalgameResourcesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	workID, ok := parseID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.lookupWork(ctx, workID); p != nil {
		return nil, p
	}
	published, err := s.store.WorkPublished(workID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !published {
		return nil, notFound()
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	filter := repository.ResourceListFilter{
		WorkID: workID, State: in.State, SkipNSFW: true,
	}
	authors, err := s.store.DistinctAuthors(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	keep, p := s.keepAuthors(ctx, authors)
	if p != nil {
		return nil, p
	}
	filter.AuthorIDs = keep
	total, err := s.store.Count(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.List(filter, "work", pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.assemble(ctx, rows, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &listGalgameResourcesOutput{Body: repr.NewPageList(items, n, rel)}, nil
}
