package apiv1

import (
	"context"
	"errors"
	"sort"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type listCollectionWorksInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	Page         int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit        int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type collectionWorkInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	WorkID       string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, an adult work is returned. Default false: an adult work in the collection is NOT_FOUND."`
}

type collectionWorkOutput struct {
	Body workrepr.WorkSummary
}

type collectionWorkEngagementOutput struct {
	Body CollectionWorkEngagement
}

func (s *Service) listCollectionWorks(ctx context.Context, in *listCollectionWorksInput) (*workSummaryPageOutput, error) {
	folder, token, _, err := s.visibleFolder(ctx, in.CollectionID)
	if err != nil {
		return nil, err
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	if s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	items, iErr := s.loadFolderItems(ctx, folder, token)
	if iErr != nil {
		return nil, mapUserPlane(iErr, token != "")
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].WorkID > items[j].WorkID
	})
	population, p := s.folderPopulation(ctx, folder, items, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	n, rel := collect.ClampTotal(len(population))
	start := pg.Offset()
	pageIDs := []int{}
	if start < len(population) {
		end := min(start+pg.Limit, len(population))
		pageIDs = population[start:end]
	}
	summaries, p := s.hydrator.ByIDs(ctx, pageIDs, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	return &workSummaryPageOutput{Body: repr.NewPageList(summaries, n, rel)}, nil
}

func (s *Service) getCollectionWork(ctx context.Context, in *collectionWorkInput) (*collectionWorkOutput, error) {
	folder, token, _, err := s.visibleFolder(ctx, in.CollectionID)
	if err != nil {
		return nil, err
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	items, iErr := s.loadFolderItems(ctx, folder, token)
	if iErr != nil {
		return nil, mapUserPlane(iErr, token != "")
	}
	found := false
	for _, it := range items {
		if int(it.WorkID) == workID {
			found = true
			break
		}
	}
	if !found {
		return nil, notFound()
	}
	if s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	sums, p := s.hydrator.ByIDs(ctx, []int{workID}, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	if len(sums) != 1 {
		return nil, notFound()
	}
	if !in.IncludeNSFW && sums[0].IsNSFW {
		return nil, notFound()
	}
	return &collectionWorkOutput{Body: sums[0]}, nil
}

func (s *Service) putCollectionWork(ctx context.Context, in *collectionWorkInput) (*collectionWorkEngagementOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	folderID, ok := parseCollectionID(in.CollectionID)
	if !ok {
		return nil, notFound()
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	if _, err := s.catalog.MyFolder(ctx, token, folderID); err != nil {
		return nil, mapUserPlane(err, false)
	}
	holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(workID))
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	firstAdd := len(holding) == 0
	if err := s.catalog.PutFolderItem(ctx, token, folderID, int64(workID)); err != nil {
		return nil, mapUserPlane(err, true)
	}
	if firstAdd {
		s.firstAddSideEffects(ctx, user.ID, workID)
	}
	return &collectionWorkEngagementOutput{Body: CollectionWorkEngagement{
		Object: "collection_work_engagement", CollectionID: repr.ID(int(folderID)), WorkID: repr.ID(workID),
		Viewer: &CollectionWorkEngagementViewer{HasWork: true},
	}}, nil
}

func (s *Service) deleteCollectionWork(ctx context.Context, in *collectionWorkInput) (*collectionWorkEngagementOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	folderID, ok := parseCollectionID(in.CollectionID)
	if !ok {
		return nil, notFound()
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	if _, err := s.catalog.MyFolder(ctx, token, folderID); err != nil {
		return nil, mapUserPlane(err, false)
	}
	holding, err := s.catalog.MyFoldersContaining(ctx, token, int64(workID))
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	inThis := false
	onlyThis := true
	for _, f := range holding {
		if f.ID == folderID {
			inThis = true
			continue
		}
		onlyThis = false
	}
	lastRemove := inThis && onlyThis
	if err := s.catalog.DeleteFolderItem(ctx, token, folderID, int64(workID)); err != nil && !errors.Is(err, catalogclient.ErrNotFound) {
		return nil, mapUserPlane(err, true)
	}
	if lastRemove {
		s.lastRemoveSideEffects(user.ID, workID)
	}
	return &collectionWorkEngagementOutput{Body: CollectionWorkEngagement{
		Object: "collection_work_engagement", CollectionID: repr.ID(int(folderID)), WorkID: repr.ID(workID),
		Viewer: &CollectionWorkEngagementViewer{HasWork: false},
	}}, nil
}
