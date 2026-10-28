package ratingapiv1

import (
	"context"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
)

const previewSummaryRunes = 200

func (s *Service) Previews(ctx context.Context, ids []int) (map[int]content.RatingPreview, error) {
	if !s.ready() {
		return nil, errUnconfigured
	}
	rows, err := s.Store.GetMany(ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int]content.RatingPreview, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	userIDs := make([]int, 0, len(rows))
	workIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
		workIDs = append(workIDs, r.WorkID)
	}
	users, err := s.Users.Users(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	works, p := s.visibleWorks(ctx, workIDs)
	if p != nil {
		return nil, p
	}
	for _, r := range rows {
		author, ok := s.authorRef(users, r.UserID)
		if !ok {
			continue
		}
		work, ok := works[r.WorkID]
		if !ok {
			continue
		}
		ref := workrepr.Ref(ctx, &work, s.CDN)
		out[r.ID] = content.RatingPreview{
			Object:       "rating",
			ID:           repr.ID(r.ID),
			Work:         &ref,
			Author:       author,
			Overall:      r.Overall,
			ShortSummary: truncate(r.ShortSummary, previewSummaryRunes),
			HasSpoilers:  r.SpoilerLevel != "none",
			CreatedAt:    repr.Timestamp(r.Created),
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
