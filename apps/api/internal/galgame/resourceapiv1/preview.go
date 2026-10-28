package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/userclient"
)

type PreviewFunc func(ctx context.Context, ids []int) (map[int]content.GalgameResourcePreview, error)

func Previews(store *repository.ResourceV1Store, users *userclient.Client, getCatalog func() Catalog, cdn string) PreviewFunc {
	s := &Service{store: store, users: users, getCatalog: getCatalog, cdn: cdn}
	return s.previews
}

func (s *Service) previews(ctx context.Context, ids []int) (map[int]content.GalgameResourcePreview, error) {
	rows, err := s.store.FindMany(ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int]content.GalgameResourcePreview, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	authorIDs := make([]int, 0, len(rows))
	workIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		authorIDs = append(authorIDs, r.UserID)
		workIDs = append(workIDs, r.WorkID)
	}
	visible, err := s.store.VisibleWorks(workIDs)
	if err != nil {
		return nil, err
	}
	users, err := s.users.Users(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	catRows, p := s.catalogRows(ctx, workIDs)
	if p != nil {
		return nil, p
	}
	works := s.workRefs(ctx, catRows, workIDs)
	for _, r := range rows {
		work, ok := works[r.WorkID]
		if !ok || !visible[r.WorkID] || !renderable(users, r.UserID) {
			continue
		}
		plats, _ := platformsOf(r)
		out[r.ID] = content.GalgameResourcePreview{
			Object:            "galgame_resource",
			ID:                repr.ID(r.ID),
			Work:              &work,
			Author:            repr.NewUserRef(s.cdn, users[r.UserID]),
			ResourceType:      resourcevocab.CompatType(r.Type),
			ResourceLanguages: typed[workrepr.ResourceLanguage](languagesOf(r)),
			ResourcePlatforms: typed[workrepr.ResourcePlatform](plats),
			Title:             r.Title,
			Size:              r.Size,
			State:             stateOf(r.Status),
			DownloadCount:     r.Download,
			LikeCount:         r.LikeCount,
			CreatedAt:         repr.Timestamp(r.CreatedAt),
		}
	}
	return out, nil
}

func typed[T ~string](in []string) []T {
	out := make([]T, len(in))
	for i, v := range in {
		out[i] = T(v)
	}
	return out
}
