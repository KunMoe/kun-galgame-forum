package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"kun-galgame-api/internal/apiv1/content"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	"kun-galgame-api/internal/galgame/workrepr"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"

	"golang.org/x/sync/errgroup"
)

var errCatalogUnconfigured = errors.New("entity previews: catalog is not configured")

type previewLookup func(ctx context.Context, ids []int) (map[int]content.EntityTarget, error)

func lookup[T any](get func(context.Context, []int) (map[int]T, error), wrap func(T) content.EntityTarget) previewLookup {
	return func(ctx context.Context, ids []int) (map[int]content.EntityTarget, error) {
		rows, err := get(ctx, ids)
		if err != nil {
			return nil, err
		}
		out := make(map[int]content.EntityTarget, len(rows))
		for id, row := range rows {
			out[id] = wrap(row)
		}
		return out, nil
	}
}

const catalogPreviewTTL = 10 * time.Minute

// cached keeps catalog previews, which change rarely, for a while. Forum rows
// (topics, resources, ratings) are never cached: hiding or deleting one has to
// take its card down on the next read.
func cached(get previewLookup, ttl time.Duration) previewLookup {
	type entry struct {
		target content.EntityTarget
		found  bool
		until  time.Time
	}
	var mu sync.Mutex
	entries := map[int]entry{}
	return func(ctx context.Context, ids []int) (map[int]content.EntityTarget, error) {
		now := time.Now()
		out := make(map[int]content.EntityTarget, len(ids))
		var misses []int
		mu.Lock()
		for _, id := range ids {
			e, ok := entries[id]
			switch {
			case !ok || now.After(e.until):
				misses = append(misses, id)
			case e.found:
				out[id] = e.target
			}
		}
		mu.Unlock()
		if len(misses) == 0 {
			return out, nil
		}
		rows, err := get(ctx, misses)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		if len(entries) > 20000 {
			entries = map[int]entry{}
		}
		for _, id := range misses {
			t, found := rows[id]
			entries[id] = entry{target: t, found: found, until: now.Add(ttl)}
			if found {
				out[id] = t
			}
		}
		return out, nil
	}
}

func (a *App) entityResolver(cdn string) content.EntityResolver {
	a.entityOnce.Do(func() { a.entityResolve = a.buildEntityResolver(cdn) })
	return a.entityResolve
}

func (a *App) buildEntityResolver(cdn string) content.EntityResolver {
	if a.DB == nil || a.UserClient == nil {
		return nil
	}
	catalog := func() resourceapiv1.Catalog { return a.ResourceCatalog }
	works := func(ctx context.Context, ids []int) (map[int]content.EntityTarget, error) {
		cat := catalog()
		if cat == nil {
			return nil, errCatalogUnconfigured
		}
		rows, appErr := cat.CatalogRowsByWorkIDs(ctx, ids, workrepr.RowInclude+",tags", "all")
		if appErr != nil {
			return nil, appErr
		}
		out := make(map[int]content.EntityTarget, len(rows))
		for id := range rows {
			row := rows[id]
			if !client.CatalogItemRenderable(&row) {
				continue
			}
			date, precision := workrepr.Release(row.ReleaseDate)
			out[id] = content.WorkTarget(content.WorkPreview{
				WorkRef:              galgameapiv1.WorkRefOf(ctx, &row, cdn),
				Maker:                workrepr.Maker(&row),
				TopTags:              workrepr.TopTags(&row, false, 4),
				ReleaseDate:          date,
				ReleaseDatePrecision: precision,
			})
		}
		return out, nil
	}
	lookups := map[content.EntityType]previewLookup{
		content.EntityTopic: lookup(topicapiv1.Previews(topicRepo.NewTopicRepository(a.DB), a.UserClient, a.ImageMeta, cdn), content.TopicTarget),
		content.EntityGalgameResource: lookup(resourceapiv1.Previews(galgameRepo.NewResourceV1Store(a.DB), a.UserClient, catalog, cdn),
			content.GalgameResourceTarget),
		content.EntityWork: works,
	}
	if e := a.GalgameEntityV1; e != nil {
		lookups[content.EntityCharacter] = cached(lookup(e.CharacterPreviews, content.CharacterTarget), catalogPreviewTTL)
		lookups[content.EntityCompany] = cached(lookup(e.CompanyPreviews, content.CompanyTarget), catalogPreviewTTL)
		lookups[content.EntityTag] = cached(lookup(e.TagPreviews, content.TagTarget), catalogPreviewTTL)
		lookups[content.EntitySeries] = cached(lookup(e.SeriesPreviews, content.SeriesTarget), catalogPreviewTTL)
		lookups[content.EntityCreditName] = cached(lookup(e.CreditNamePreviews, content.CreditNameTarget), catalogPreviewTTL)
	}
	if r := a.GalgameRatingV1; r != nil {
		lookups[content.EntityRating] = lookup(r.Previews, content.RatingTarget)
	}
	return func(ctx context.Context, refs []content.EntityRef) (map[content.EntityRef]content.EntityTarget, error) {
		byType := map[content.EntityType][]int{}
		for _, r := range refs {
			byType[r.Type] = append(byType[r.Type], r.ID)
		}
		out := make(map[content.EntityRef]content.EntityTarget, len(refs))
		var mu sync.Mutex
		g, gctx := errgroup.WithContext(ctx)
		for typ, ids := range byType {
			get, ok := lookups[typ]
			if !ok {
				continue
			}
			g.Go(func() error {
				rows, err := get(gctx, ids)
				if err != nil {
					return err
				}
				mu.Lock()
				defer mu.Unlock()
				for id, t := range rows {
					out[content.EntityRef{Type: typ, ID: id}] = t
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return out, nil
	}
}
