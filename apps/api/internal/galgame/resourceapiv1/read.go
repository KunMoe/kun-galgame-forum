package apiv1

import (
	"context"
	"errors"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) getGalgameResource(ctx context.Context, in *resourceIDInput) (*resourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if err := s.store.IncrementView(row.ID); err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	row.View++
	out, p := s.one(ctx, row, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &resourceOutput{Body: out}, nil
}

func (s *Service) createGalgameResourceDownload(ctx context.Context, in *resourceIDInput) (*downloadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	links, err := s.store.Links(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if err := s.store.IncrementDownload(row.ID); err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	return &downloadOutput{Body: GalgameResourceDownload{
		Object: "galgame_resource_download",
		DownloadURLs: typedURLs(links),
		ExtractionCode: row.Code, ArchivePassword: row.Password,
	}}, nil
}

func (s *Service) getGalgameResourceSource(ctx context.Context, in *resourceIDInput) (*sourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if !canEditResource(row.UserID, user) {
		return nil, permissionRequired()
	}
	links, err := s.store.Links(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	langs := languagesOf(*row)
	plats, runs := platformsOf(*row)
	return &sourceOutput{Body: GalgameResourceSource{
		Object: "galgame_resource_source", ResourceID: repr.ID(row.ID),
		ResourceType: resourcevocab.CompatType(row.Type), Title: row.Title, VersionLabel: versionToken(row.VersionLabel),
		ResourceLanguages: typedLangs(langs), ResourcePlatforms: typedPlats(plats), ResourceRuntimes: typedRuns(runs),
		Size: row.Size, DownloadURLs: typedURLs(links),
		ExtractionCode: row.Code, ArchivePassword: row.Password, ContentMarkdown: row.Note,
	}}, nil
}
