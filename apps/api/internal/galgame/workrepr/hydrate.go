package workrepr

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

// RowInclude is what a WorkSummary reads off a catalog works row.
const RowInclude = "names,covers,refs,labels"

type Rows interface {
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *errors.AppError)
}

type Hydrator struct {
	rows     Rows
	galgames *repository.GalgameRepository
	lists    *repository.GalgameListRepository
	meta     *repository.GalgameResourceMetaRepository
	cdn      string
}

func NewHydrator(rows Rows, db *gorm.DB, cdn string) *Hydrator {
	return &Hydrator{
		rows:     rows,
		galgames: repository.NewGalgameRepository(db),
		lists:    repository.NewGalgameListRepository(db),
		meta:     repository.NewGalgameResourceMetaRepository(db),
		cdn:      cdn,
	}
}

func ContentLimit(includeNSFW bool) string {
	if includeNSFW {
		return "all"
	}
	return "sfw"
}

// ByIDs keeps the order of ids and drops the ones catalog did not return: an id
// the reader's content limit hides, or one catalog no longer renders.
func (h *Hydrator) ByIDs(ctx context.Context, ids []int, includeNSFW bool) ([]WorkSummary, *problem.Problem) {
	if len(ids) == 0 {
		return []WorkSummary{}, nil
	}
	byID, appErr := h.rows.CatalogRowsByWorkIDs(ctx, ids, RowInclude, ContentLimit(includeNSFW))
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	rows := make([]client.CatalogWorkListItem, 0, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok {
			slog.Warn("workrepr: catalog did not render work, dropped", "work_id", id, "include_nsfw", includeNSFW)
			continue
		}
		if !client.CatalogItemRenderable(&row) {
			slog.Warn("workrepr: catalog row not renderable, dropped", "work_id", id, "include_nsfw", includeNSFW)
			continue
		}
		rows = append(rows, row)
	}
	return h.FromRows(ctx, rows)
}

func (h *Hydrator) FromRows(ctx context.Context, rows []client.CatalogWorkListItem) ([]WorkSummary, *problem.Problem) {
	out := make([]WorkSummary, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]int, len(rows))
	for i := range rows {
		ids[i] = int(rows[i].ID)
	}
	local := h.galgames.FindLocalBatch(ids)
	ratings := h.lists.BayesianRatings(ids)
	axes, err := h.meta.FindResourceAxesBatch(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	for i := range rows {
		out = append(out, h.summary(ctx, &rows[i], local[ids[i]], ratings[ids[i]], axes[ids[i]]))
	}
	return out, nil
}

func (h *Hydrator) summary(
	ctx context.Context,
	it *client.CatalogWorkListItem,
	local repository.GalgameLocalRow,
	rating repository.RatingInfo,
	axes repository.ResourceAxes,
) WorkSummary {
	date, precision := Release(it.ReleaseDate)
	s := WorkSummary{
		WorkRef:              Ref(ctx, it, h.cdn),
		Banner:               Banner(it, h.cdn),
		ReleaseDate:          date,
		ReleaseDatePrecision: precision,
		Maker:                Maker(it),
		ViewCount:            max(local.View, 0),
		LikeCount:            max(local.LikeCount, 0),
		RatingCount:          rating.Count,
		ResourcePlatforms:    inVocabOrder[ResourcePlatform](resourcevocab.PlatformKeys, axes.Platforms),
		ResourceLanguages:    inVocabOrder[ResourceLanguage](resourcevocab.LanguageKeys, axes.Languages),
		IsPublished:          local.Published,
	}
	if rating.Count > 0 {
		score := rating.Score
		s.RatingScore = &score
	}
	if !local.ResourceUpdateTime.IsZero() {
		t := repr.Timestamp(local.ResourceUpdateTime)
		s.ResourceUpdatedAt = &t
	}
	return s
}

func inVocabOrder[T ~string](vocab []string, present map[string]bool) []T {
	out := make([]T, 0, len(present))
	for _, key := range vocab {
		if present[key] {
			out = append(out, T(key))
		}
	}
	return out
}
