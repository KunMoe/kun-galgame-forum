package entityapiv1

import (
	"cmp"
	"context"
	"strconv"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

var spoilerLevels = []string{"none", "minor", "major"}

type CharacterGender string

func (CharacterGender) Schema(huma.Registry) *huma.Schema {
	n := 6
	return &huma.Schema{Type: huma.TypeString, Enum: []any{"female", "male", "other"}, MaxLength: &n, Description: "A character's recorded gender."}
}

var characterSorts = map[string]string{
	"popularity_desc": "popularity",
	"relevance_desc":  "relevance",
	"id_desc":         "newest",
}

type listCharactersInput struct {
	Q           string            `query:"q" maxLength:"100" doc:"Name search over every name and alias catalog records for the character. Free text; never use it as a decision input."`
	TraitIDs    []repr.DecimalID  `query:"trait_ids" maxItems:"10" doc:"Trait ids, comma-separated, 1–10. A trait also matches its descendants, so boots finds knee-high boots too. Only trait links without a spoiler count. An adult trait needs include_nsfw=true."`
	TraitMatch  string            `query:"trait_match" enum:"all,any" default:"all" maxLength:"3" doc:"all: a character must match every trait in trait_ids. any: at least one. No effect without trait_ids."`
	Genders     []CharacterGender `query:"genders" maxItems:"3" doc:"Only characters of any of these genders. Comma-separated. Omitted means no filter."`
	Sort        string            `query:"sort" enum:"popularity_desc,relevance_desc,id_desc" maxLength:"15" doc:"Order. popularity: how widely the works a character appears in are collected, lead roles weighing double. relevance: the name search's ranking, and needs q. id: newest in catalog first. Omitted: relevance_desc when q is set, popularity_desc otherwise."`
	Page        int               `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int               `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool              `query:"include_nsfw" default:"false" doc:"When true, adult traits may be named in trait_ids. Default false."`
}

type listCharactersOutput struct {
	Body repr.PageList[CharacterSummary]
}

func (s *Service) listCharacters(ctx context.Context, in *listCharactersInput) (*listCharactersOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := checkDepth(in.Page, in.Limit, false); prob != nil {
		return nil, prob
	}
	q := trimQuery(in.Q)
	sort := in.Sort
	if sort == "" {
		sort = "popularity_desc"
		if q != "" {
			sort = "relevance_desc"
		}
	}
	if sort == "relevance_desc" && q == "" {
		return nil, problem.New(problem.CodeInvalidParameter, "sort=relevance_desc ranks a name search and needs q.",
			problem.AtParameter("sort", problem.ReasonInconsistentWith, "q", nil))
	}
	traitIDs, prob := s.characterTraitFilter(ctx, in.TraitIDs, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	genders := make([]string, 0, len(in.Genders))
	for _, g := range in.Genders {
		genders = append(genders, string(g))
	}
	page, appErr := s.catalog.CatalogCharacterList(ctx, client.CatalogCharacterQuery{
		Q:        q,
		TraitIDs: traitIDs,
		MatchAny: in.TraitMatch == "any",
		Genders:  genders,
		Sort:     characterSorts[sort],
		Page:     in.Page,
		Limit:    in.Limit,
		NSFW:     in.IncludeNSFW,
	})
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	var vocab *traitVocab
	if len(traitIDs) > 0 {
		v, err := s.traitVocab(ctx)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		vocab = v
	}
	rows := make([]CharacterSummary, 0, len(page.Items))
	for i := range page.Items {
		rows = append(rows, s.characterSummary(&page.Items[i], vocab, in.IncludeNSFW))
	}
	total, relation := collect.ClampTotal(page.Total)
	return &listCharactersOutput{Body: repr.NewPageList(rows, total, relation)}, nil
}

func (s *Service) characterTraitFilter(ctx context.Context, raw []repr.DecimalID, includeNSFW bool) ([]int, *problem.Problem) {
	if len(raw) == 0 {
		return nil, nil
	}
	ids, prob := parseEntityIDs(raw)
	if prob != nil {
		return nil, prob
	}
	ids = uniqueIDs(ids)
	if includeNSFW {
		return ids, nil
	}
	v, err := s.traitVocab(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	for _, id := range ids {
		if n, ok := v.byID[id]; ok && n.sexual {
			return nil, problem.New(problem.CodeInvalidParameter, "An adult trait needs include_nsfw=true.",
				problem.AtParameter("trait_ids", problem.ReasonNotAllowedValue, "trait "+strconv.Itoa(id)+" is adult content", nil))
		}
	}
	return ids, nil
}

var imageSexualLevel = map[string]int16{"safe": 0, "suggestive": 1, "explicit": 2}

func (s *Service) characterSummary(row *client.CatalogCharacterRow, vocab *traitVocab, includeNSFW bool) CharacterSummary {
	latin := ""
	if row.Latin != nil {
		latin = *row.Latin
	}
	out := CharacterSummary{
		CharacterRef: CharacterRef{
			Object:      "character",
			ID:          repr.ID(int(row.ID)),
			CatalogName: workrepr.Name(row.DisplayName, latin, client.LocalizedValues(row.Localized)),
		},
		CatalogWorkCount: max(row.WorkCount, 0),
		MatchedTraits:    []TraitRef{},
	}
	if vocab != nil {
		for _, id := range row.MatchedIDs() {
			if n, ok := vocab.visible(id, includeNSFW); ok {
				out.MatchedTraits = append(out.MatchedTraits, vocab.ref(n))
			}
		}
	}
	if img := row.Image; img != nil {
		meta := &imageclient.ImageMeta{}
		if img.Width != nil {
			meta.Width = *img.Width
		}
		if img.Height != nil {
			meta.Height = *img.Height
		}
		if img.Thumbhash != nil {
			meta.Thumbhash = *img.Thumbhash
		}
		if img.Sexual != nil {
			if level, ok := imageSexualLevel[*img.Sexual]; ok {
				meta.Sexual = &level
			}
		}
		out.Image = repr.NewImage(s.cdn, img.Hash, meta)
	}
	return out
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
		Image:       workrepr.ImageFromURL(s.cdn, ch.Image, ch.ImageMeta.Width, ch.ImageMeta.Height, ch.ImageMeta.Thumbhash, nil),
		Figure:      workrepr.ImageFromURL(s.cdn, ch.Figure, ch.FigureMeta.Width, ch.FigureMeta.Height, ch.FigureMeta.Thumbhash, nil),
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
