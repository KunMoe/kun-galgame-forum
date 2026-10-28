package entityapiv1

import (
	"cmp"
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"

	"golang.org/x/sync/errgroup"
)

const (
	previewFanout       = 8
	previewWorks        = 3
	previewNameCredits  = 20
	previewDescriptionN = 120
)

func (s *Service) CompanyPreviews(ctx context.Context, ids []int) (map[int]content.CompanyPreview, error) {
	return fanOut(ctx, ids, func(ctx context.Context, id int) (content.CompanyPreview, bool, error) {
		o, found, movedTo, appErr := s.catalog.CatalogLabel(ctx, strconv.Itoa(id))
		if appErr != nil {
			return content.CompanyPreview{}, false, appErr
		}
		if !found || movedTo != 0 || !companyKinds[o.Kind] {
			return content.CompanyPreview{}, false, nil
		}
		return content.CompanyPreview{
			Object:           "company",
			ID:               repr.ID(int(o.ID)),
			CatalogName:      workrepr.Name(o.DisplayName, o.Latin, client.LocalizedValues(o.Localized)),
			CompanyKind:      o.Kind,
			Logo:             workrepr.ImageFromHash(s.cdn, o.LogoHash),
			CatalogWorkCount: max(o.WorkCount, 0),
		}, true, nil
	})
}

func (s *Service) TagPreviews(ctx context.Context, ids []int) (map[int]content.TagPreview, error) {
	return fanOut(ctx, ids, func(ctx context.Context, id int) (content.TagPreview, bool, error) {
		t, found, appErr := s.catalog.CatalogTag(ctx, strconv.Itoa(id))
		if appErr != nil {
			return content.TagPreview{}, false, appErr
		}
		if !found || !tagKinds[t.Kind] {
			return content.TagPreview{}, false, nil
		}
		return content.TagPreview{
			Object:           "tag",
			ID:               repr.ID(int(t.ID)),
			CatalogName:      workrepr.Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized)),
			TagKind:          t.Kind,
			IsSexual:         t.Sexual,
			CatalogWorkCount: max(t.WorkCount, 0),
			Description:      introExcerpt(t.Intros),
		}, true, nil
	})
}

func (s *Service) SeriesPreviews(ctx context.Context, ids []int) (map[int]content.SeriesPreview, error) {
	return fanOut(ctx, ids, func(ctx context.Context, id int) (content.SeriesPreview, bool, error) {
		rec, found, appErr := s.catalog.CatalogSeries(ctx, strconv.Itoa(id))
		if appErr != nil {
			return content.SeriesPreview{}, false, appErr
		}
		if !found {
			return content.SeriesPreview{}, false, nil
		}
		card := s.card(ctx, seriesRow{
			id:      int(rec.ID),
			name:    workrepr.Name(cmp.Or(rec.DisplayName, rec.Name), "", client.LocalizedValues(rec.Localized)),
			count:   max(rec.WorkCount, 0),
			hasNSFW: rec.HasNSFW,
		})
		p := content.SeriesPreview{
			Object: "series", ID: card.ID, CatalogName: card.CatalogName,
			HasNSFWWorks: card.HasNSFWWorks, CatalogWorkCount: card.CatalogWorkCount,
			Works: []repr.WorkRef{},
		}
		for _, w := range card.SampleWorks {
			if len(p.Works) == previewWorks {
				break
			}
			p.Works = append(p.Works, w.WorkRef)
		}
		if len(card.SampleWorks) > 0 {
			p.Cover = card.SampleWorks[0].Cover
		}
		return p, true, nil
	})
}

func (s *Service) CharacterPreviews(ctx context.Context, ids []int) (map[int]content.CharacterPreview, error) {
	workIDs := map[int][]int{}
	var mu sync.Mutex
	out, err := fanOut(ctx, ids, func(ctx context.Context, id int) (content.CharacterPreview, bool, error) {
		ch, found, movedTo, appErr := s.catalog.CatalogCharacterDetail(ctx, int64(id), previewWorks, 0, true)
		if appErr != nil {
			return content.CharacterPreview{}, false, appErr
		}
		if !found || movedTo != 0 {
			return content.CharacterPreview{}, false, nil
		}
		works := make([]int, 0, len(ch.Works))
		for _, w := range ch.Works {
			works = append(works, int(w.Work.ID))
		}
		mu.Lock()
		workIDs[id] = works
		mu.Unlock()
		return content.CharacterPreview{
			Object:      "character",
			ID:          repr.ID(int(ch.ID)),
			CatalogName: workrepr.Name(ch.DisplayName, ch.Latin, client.LocalizedValues(ch.Localized)),
			Image:       workrepr.GatedImageFromURL(s.cdn, ch.Image, ch.ImageMeta.Width, ch.ImageMeta.Height, ch.ImageMeta.Thumbhash, ch.ImageSexual, false),
		}, true, nil
	})
	if err != nil {
		return nil, err
	}
	refs, err := s.previewWorkRefs(ctx, workIDs)
	if err != nil {
		return nil, err
	}
	for id, p := range out {
		p.Works = pickWorks(workIDs[id], refs)
		out[id] = p
	}
	return out, nil
}

func (s *Service) CreditNamePreviews(ctx context.Context, ids []int) (map[int]content.CreditNamePreview, error) {
	workIDs := map[int][]int{}
	var mu sync.Mutex
	out, err := fanOut(ctx, ids, func(ctx context.Context, id int) (content.CreditNamePreview, bool, error) {
		n, found, movedTo, appErr := s.catalog.CatalogNameDetail(ctx, int64(id), previewNameCredits, 0)
		if appErr != nil {
			return content.CreditNamePreview{}, false, appErr
		}
		if !found || movedTo != 0 {
			return content.CreditNamePreview{}, false, nil
		}
		roleCount := map[string]int{}
		works := make([]int, 0, len(n.Credits))
		for _, c := range n.Credits {
			works = append(works, int(c.Work.ID))
			for _, r := range c.Roles {
				if r.RoleName != "" {
					roleCount[r.RoleName]++
				}
			}
		}
		mu.Lock()
		workIDs[id] = works
		mu.Unlock()
		return content.CreditNamePreview{
			Object:      "credit_name",
			ID:          repr.ID(int(n.ID)),
			CatalogName: workrepr.Name(n.DisplayName, n.Latin, client.LocalizedValues(n.Localized)),
			Photo:       workrepr.ImageFromHash(s.cdn, n.PhotoHash),
			RoleNames:   topRoles(roleCount),
		}, true, nil
	})
	if err != nil {
		return nil, err
	}
	refs, err := s.previewWorkRefs(ctx, workIDs)
	if err != nil {
		return nil, err
	}
	for id, p := range out {
		p.Works = pickWorks(workIDs[id], refs)
		out[id] = p
	}
	return out, nil
}

func (s *Service) previewWorkRefs(ctx context.Context, byOwner map[int][]int) (map[int]repr.WorkRef, error) {
	seen := map[int]bool{}
	var ids []int
	for _, ws := range byOwner {
		for _, id := range ws {
			if id > 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	out := map[int]repr.WorkRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, appErr := s.catalog.CatalogRowsByWorkIDs(ctx, ids, workrepr.RowInclude, "all")
	if appErr != nil {
		return nil, appErr
	}
	for id := range rows {
		row := rows[id]
		if client.CatalogItemRenderable(&row) {
			out[id] = workrepr.Ref(ctx, &row, s.cdn)
		}
	}
	return out, nil
}

func pickWorks(ids []int, refs map[int]repr.WorkRef) []repr.WorkRef {
	out := []repr.WorkRef{}
	seen := map[int]bool{}
	for _, id := range ids {
		ref, ok := refs[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, ref)
		if len(out) == previewWorks {
			break
		}
	}
	return out
}

func topRoles(count map[string]int) []content.CreditRoleName {
	roles := make([]string, 0, len(count))
	for r := range count {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool {
		if count[roles[i]] != count[roles[j]] {
			return count[roles[i]] > count[roles[j]]
		}
		return roles[i] < roles[j]
	})
	out := make([]content.CreditRoleName, 0, 3)
	for _, r := range roles {
		if len(out) == 3 {
			break
		}
		out = append(out, content.CreditRoleName(r))
	}
	return out
}

func introExcerpt(intros []client.CatalogIntro) string {
	pick := ""
	for _, in := range intros {
		text := strings.TrimSpace(in.Intro)
		if text == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(in.Lang), "zh") {
			pick = text
			break
		}
		if pick == "" {
			pick = text
		}
	}
	pick = strings.Join(strings.Fields(pick), " ")
	if utf8.RuneCountInString(pick) > previewDescriptionN {
		pick = string([]rune(pick)[:previewDescriptionN]) + "…"
	}
	return pick
}

func fanOut[T any](ctx context.Context, ids []int, get func(context.Context, int) (T, bool, error)) (map[int]T, error) {
	out := make(map[int]T, len(ids))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(previewFanout)
	for _, id := range ids {
		g.Go(func() error {
			v, ok, err := get(gctx, id)
			if err != nil || !ok {
				return err
			}
			mu.Lock()
			out[id] = v
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
