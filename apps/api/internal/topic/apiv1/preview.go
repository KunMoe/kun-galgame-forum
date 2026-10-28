package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"
)

type PreviewFunc func(ctx context.Context, ids []int) (map[int]content.TopicPreview, error)

func Previews(topics *repository.TopicRepository, users *userclient.Client, images func([]string) map[string]imageclient.ImageMeta, cdn string) PreviewFunc {
	return func(ctx context.Context, ids []int) (map[int]content.TopicPreview, error) {
		rows, err := topics.FindByIDs(ids)
		if err != nil {
			return nil, err
		}
		var authorIDs []int
		var hashes []string
		for _, r := range rows {
			if r.Status != 0 || r.AccessScope != "public" {
				continue
			}
			authorIDs = append(authorIDs, r.UserID)
			for _, tok := range withoutStickers(r.CoverImages) {
				if h, _, ok := markdown.ParseContentImageRef(tok); ok {
					hashes = append(hashes, h)
				}
			}
		}
		out := make(map[int]content.TopicPreview, len(rows))
		if len(authorIDs) == 0 {
			return out, nil
		}
		authors, err := users.Users(ctx, authorIDs)
		if err != nil {
			return nil, err
		}
		var meta map[string]imageclient.ImageMeta
		if len(hashes) > 0 && images != nil {
			meta = images(hashes)
		}
		for _, r := range rows {
			author, ok := authors[r.UserID]
			if r.Status != 0 || r.AccessScope != "public" || !ok || !userclient.IsRenderable(author) {
				continue
			}
			out[r.ID] = content.TopicPreview{
				Object:      "topic",
				ID:          repr.ID(r.ID),
				Title:       r.Title,
				CoverImages: coverImagesWithMeta(cdn, r.CoverImages, meta),
				Author:      repr.NewUserRef(cdn, author),
				ViewCount:   r.View,
				LikeCount:   r.LikeCount,
				ReplyCount:  r.ReplyCount,
				IsNSFW:      r.IsNSFW,
				CreatedAt:   repr.Timestamp(r.CreatedAt),
			}
		}
		return out, nil
	}
}
