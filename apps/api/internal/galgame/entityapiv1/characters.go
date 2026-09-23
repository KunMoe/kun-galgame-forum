package entityapiv1

import (
	"cmp"
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

var spoilerLevels = []string{"none", "minor", "major"}

type listCharactersOutput struct {
	Body repr.PageList[CharacterRef]
}

func (s *Service) listCharacters(ctx context.Context, in *searchInput) (*listCharactersOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q, prob := in.query()
	if prob != nil {
		return nil, prob
	}
	hits, _, appErr := s.catalog.CatalogEntitySearch(ctx, "characters", q, 1, searchDepth)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	rows := make([]CharacterRef, 0, len(hits))
	for i := range hits {
		h := &hits[i]
		rows = append(rows, CharacterRef{
			Object:      "character",
			ID:          repr.ID(int(h.ID)),
			CatalogName: workrepr.Name(h.DisplayName, h.Latin, client.LocalizedValues(h.Localized)),
		})
	}
	return &listCharactersOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

type characterPathInput struct {
	CharacterID string `path:"character_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Character id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult traits are included. Default false."`
}

type getCharacterOutput struct {
	Body Character
}

func (s *Service) character(ctx context.Context, rawID string, limit, offset int, withWorks bool) (*client.CatalogCharacter, *problem.Problem) {
	id, ok := pathID(rawID)
	if !ok {
		return nil, notFound()
	}
	ch, found, movedTo, appErr := s.catalog.CatalogCharacterDetail(ctx, int64(id), limit, offset, withWorks)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if movedTo != 0 {
		return nil, merged("character", movedTo)
	}
	if !found {
		return nil, notFound()
	}
	return ch, nil
}

// traitName folds in the superseded name_zh column, which is still the only
// Chinese name many traits have.
func traitName(display, zh string, localized map[string]client.LocalizedValue) repr.CatalogName {
	if _, ok := localized["zh-Hans"]; !ok && zh != "" {
		localized["zh-Hans"] = client.LocalizedValue{Value: zh}
	}
	return workrepr.Name(display, "", localized)
}

func (s *Service) getCharacter(ctx context.Context, in *characterPathInput) (*getCharacterOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	ch, prob := s.character(ctx, in.CharacterID, 1, 0, false)
	if prob != nil {
		return nil, prob
	}
	traits := make([]CharacterTrait, 0, len(ch.Traits))
	for _, t := range ch.Traits {
		if t.Sexual && !in.IncludeNSFW {
			continue
		}
		level := spoilerLevels[len(spoilerLevels)-1]
		if t.Spoiler >= 0 && t.Spoiler < len(spoilerLevels) {
			level = spoilerLevels[t.Spoiler]
		}
		traits = append(traits, CharacterTrait{
			Object:      "trait",
			ID:          repr.ID(int(t.ID)),
			CatalogName: traitName(cmp.Or(t.DisplayName, t.Name), t.NameZh, client.LocalizedValues(t.Localized)),
			TraitGroup:  traitName(t.Group, t.GroupZh, client.LocalizedValues(t.GroupLocalized)),
			Spoiler:     level,
			IsLie:       t.Lie,
			IsSexual:    t.Sexual,
		})
	}
	links := make([]workrepr.CatalogLink, 0, len(ch.Refs))
	for _, ref := range ch.Refs {
		if tpl, ok := characterPage[ref.Source]; ok && ref.ExternalID != "" {
			links = workrepr.AppendLink(links, ref.Source, tpl(ref.ExternalID))
		}
	}
	return &getCharacterOutput{Body: Character{
		Object:      "character",
		ID:          repr.ID(int(ch.ID)),
		CatalogName: workrepr.Name(ch.DisplayName, ch.Latin, client.LocalizedValues(ch.Localized)),
		Lang:        workrepr.Lang(ch.Lang),
		Image:       workrepr.ImageFromURL(s.cdn, ch.Image, ch.ImageMeta.Width, ch.ImageMeta.Height, ch.ImageMeta.Thumbhash),
		Figure:      workrepr.ImageFromURL(s.cdn, ch.Figure, ch.FigureMeta.Width, ch.FigureMeta.Height, ch.FigureMeta.Thumbhash),
		Intros:      workrepr.Intros(ch.Intros),
		Traits:      traits,
		Links:       links,
	}}, nil
}

type listAppearancesInput struct {
	CharacterID string `path:"character_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Character id."`
	StreamQuery
}

type listAppearancesOutput struct {
	Body repr.List[Appearance]
}

func (s *Service) listCharacterAppearances(ctx context.Context, in *listAppearancesInput) (*listAppearancesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	offset, prob := in.offset("character", in.CharacterID)
	if prob != nil {
		return nil, prob
	}
	ch, prob := s.character(ctx, in.CharacterID, in.Limit, offset, true)
	if prob != nil {
		return nil, prob
	}
	ids := make([]int64, 0, len(ch.Works))
	for _, w := range ch.Works {
		ids = append(ids, w.Work.ID)
	}
	byID, prob := s.visibleRows(ctx, ids, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	items := make([]Appearance, 0, len(ch.Works))
	for _, w := range ch.Works {
		work, ok := byID[w.Work.ID]
		if !ok {
			continue
		}
		voices := make([]CreditNameRef, 0, len(w.Voices))
		for i := range w.Voices {
			voices = append(voices, creditNameRef(&w.Voices[i]))
		}
		items = append(items, Appearance{Object: "appearance", WorkSummary: work, Voices: voices})
	}
	return &listAppearancesOutput{Body: repr.NewList(items, in.next("character", in.CharacterID, ch.NextOffset))}, nil
}
