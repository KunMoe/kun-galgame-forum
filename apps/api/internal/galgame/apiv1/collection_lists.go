package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type listMyCollectionsInput struct {
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	WorkID      string `query:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"When set, each item's viewer.has_work is whether this collection holds that work."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included in preview_covers. Default false."`
}

type collectionSummaryPageOutput struct {
	Body repr.PageList[CollectionSummary]
}

type listUserCollectionsInput struct {
	UserID      string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Page        int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included in preview_covers. Default false."`
}

type collectionAliasInput struct {
	AliasID string `path:"alias_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The frozen forum collection id."`
}

type collectionAliasOutput struct {
	Body CollectionAlias
}

func (s *Service) listMyCollections(ctx context.Context, in *listMyCollectionsInput) (*collectionSummaryPageOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	// No default folder is created here. Catalog's POST /v2/me/folders refuses a
	// blank name (deviation 112: only the backfill may make an unnamed default),
	// and the lazy create this face used to do sent name:"" — every reader with
	// no folders got 422 → 233 instead of an empty picker.
	folders, err := s.catalog.MyFolders(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	contains := map[int64]bool{}
	if in.WorkID != "" {
		workID, ok := parseWorkID(in.WorkID)
		if !ok {
			return nil, invalidParameter(problem.AtParameter("work_id", problem.ReasonInvalidFormat,
				"work_id must be a positive decimal integer", nil))
		}
		holding, hErr := s.catalog.MyFoldersContaining(ctx, token, int64(workID))
		if hErr != nil {
			return nil, mapUserPlane(hErr, true)
		}
		for _, f := range holding {
			contains[f.ID] = true
		}
	}
	sortFolders(folders)
	n, rel := collect.ClampTotal(len(folders))
	pageFolders := pageOfFolders(folders, pg.Page, pg.Limit)
	items := make([]CollectionSummary, 0, len(pageFolders))
	for _, f := range pageFolders {
		sum, ok, p := s.toSummary(ctx, f, user, token, contains[f.ID], in.IncludeNSFW)
		if p != nil {
			return nil, p
		}
		if !ok {
			continue
		}
		items = append(items, *sum)
	}
	return &collectionSummaryPageOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) listUserCollections(ctx context.Context, in *listUserCollectionsInput) (*collectionSummaryPageOutput, error) {
	ownerID, ok := parseWorkID(in.UserID)
	if !ok {
		return nil, notFound()
	}
	if s.users == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	_, renderable, p := s.lookupOwner(ctx, int64(ownerID))
	if p != nil {
		return nil, p
	}
	user := callerUser(ctx)
	isOwner := user != nil && user.ID == ownerID
	if !renderable && !isOwner {
		return nil, notFound()
	}
	var folders []catalogclient.Folder
	previewToken := ""
	if isOwner {
		token, tp := requireToken(ctx)
		if tp != nil {
			return nil, tp
		}
		got, lErr := s.catalog.MyFolders(ctx, token)
		if lErr != nil {
			return nil, mapUserPlane(lErr, true)
		}
		folders = got
		previewToken = token
	} else {
		got, lErr := s.catalog.PublicFolders(ctx, int64(ownerID))
		if lErr != nil {
			return nil, mapUserPlane(lErr, false)
		}
		folders = got
	}
	sortFolders(folders)
	n, rel := collect.ClampTotal(len(folders))
	pageFolders := pageOfFolders(folders, pg.Page, pg.Limit)
	items := make([]CollectionSummary, 0, len(pageFolders))
	for _, f := range pageFolders {
		sum, ok, p := s.toSummary(ctx, f, user, previewToken, false, in.IncludeNSFW)
		if p != nil {
			return nil, p
		}
		if !ok {
			continue
		}
		items = append(items, *sum)
	}
	return &collectionSummaryPageOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) getCollectionAlias(ctx context.Context, in *collectionAliasInput) (*collectionAliasOutput, error) {
	aliasID, ok := parseWorkID(in.AliasID)
	if !ok {
		return nil, notFound()
	}
	if s.aliases == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	alias, err := s.aliases.AliasByID(aliasID)
	if err != nil {
		return nil, notFound()
	}
	if _, _, _, visErr := s.visibleFolder(ctx, string(repr.ID(int(alias.CatalogFolderID)))); visErr != nil {
		if errors.Is(visErr, catalogclient.ErrNotFound) {
			return nil, notFound()
		}
		var p *problem.Problem
		if errors.As(visErr, &p) && p.Code == problem.CodeNotFound {
			return nil, notFound()
		}
		return nil, visErr
	}
	return &collectionAliasOutput{Body: CollectionAlias{
		Object: "collection_alias", AliasID: repr.ID(alias.ID), CollectionID: repr.ID(int(alias.CatalogFolderID)),
	}}, nil
}
