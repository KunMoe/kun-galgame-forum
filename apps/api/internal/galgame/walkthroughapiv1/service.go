package walkthroughapiv1

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

var errUnconfigured = errors.New("walkthrough faces are not configured")

type Catalog interface {
	CatalogWalkthroughs(ctx context.Context, q url.Values) (*client.CatalogWalkthroughPage, *legacyErrors.AppError)
	CatalogWalkthrough(ctx context.Context, id string) (*client.CatalogWalkthrough, bool, *legacyErrors.AppError)
}

type Service struct {
	catalog Catalog
}

func New(catalog Catalog) *Service {
	return &Service{catalog: catalog}
}

type listInput struct {
	WorkID      string `query:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only walkthroughs of this work."`
	CharacterID string `query:"character_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only walkthroughs whose chart routes to this character."`
	Cursor      string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque cursor from a previous page."`
	Limit       int    `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Page size."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, walkthroughs of works shown as adult content (is_nsfw) are included."`
}

type listOutput struct {
	Body repr.List[Walkthrough]
}

func (s *Service) listWalkthroughs(ctx context.Context, in *listInput) (*listOutput, error) {
	if s == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	// nsfw= is catalog's age gate (content_rating), not kungal's SFW gate
	// (content_limit). Leaving it off for SFW readers hid every r18 work's
	// walkthroughs: 31 were left in the whole catalog (2026-10-25).
	q := client.OpenPopulation(url.Values{"limit": {strconv.Itoa(in.Limit)}})
	if !in.IncludeNSFW {
		q.Set("content_limit", "sfw")
	}
	for k, v := range map[string]string{"work_id": in.WorkID, "character_id": in.CharacterID, "cursor": in.Cursor} {
		if v != "" {
			q.Set(k, v)
		}
	}
	page, appErr := s.catalog.CatalogWalkthroughs(ctx, q)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	items := make([]Walkthrough, 0, len(page.Items))
	for _, w := range page.Items {
		w.Chart = nil
		items = append(items, walkthroughFrom(w))
	}
	return &listOutput{Body: repr.NewList(items, page.NextCursor)}, nil
}

type detailInput struct {
	WalkthroughID string `path:"walkthrough_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Walkthrough id."`
}

type detailOutput struct {
	Body Walkthrough
}

func (s *Service) getWalkthrough(ctx context.Context, in *detailInput) (*detailOutput, error) {
	if s == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	w, found, appErr := s.catalog.CatalogWalkthrough(ctx, in.WalkthroughID)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	if !found || w.Chart == nil {
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	return &detailOutput{Body: walkthroughFrom(*w)}, nil
}

func localized(m map[string]client.LocalizedValue) Localized {
	out := Localized{}
	for tag, v := range m {
		out[tag] = repr.LocalizedName{Value: v.Value, IsMachine: v.Machine}
	}
	return out
}

func keys(in []string) []ChartKey {
	out := make([]ChartKey, 0, len(in))
	for _, k := range in {
		out = append(out, ChartKey(k))
	}
	return out
}

func keyPtr(k *string) *ChartKey {
	if k == nil {
		return nil
	}
	c := ChartKey(*k)
	return &c
}

func walkthroughFrom(w client.CatalogWalkthrough) Walkthrough {
	out := Walkthrough{
		Object: "walkthrough", ID: repr.DecimalID(w.ID), WorkID: repr.DecimalID(w.WorkID), IsNSFW: w.WorkContentLimit != "sfw",
		SourceSite: w.Source, SourcePageURL: w.SourceURL, SourceAuthor: w.SourceAuthor,
		TextLang: w.Lang, EndingCount: w.EndingCount, SaveCount: w.SaveCount, SegmentCount: w.SegmentCount,
		ChartCharacters: make([]WalkthroughCharacter, 0, len(w.Characters)),
	}
	if t, err := time.Parse(time.RFC3339, w.UpdatedAt); err == nil {
		out.UpdatedAt = repr.Timestamp(t)
	}
	if w.SourceUpdatedOn != nil {
		d := repr.CalendarDate(*w.SourceUpdatedOn)
		out.SourceUpdatedDate = &d
	}
	for _, ch := range w.Characters {
		c := WalkthroughCharacter{Key: ChartKey(ch.Key), SourceName: ch.Name}
		if ch.CharacterID != nil && ch.DisplayName != nil {
			c.CatalogCharacter = &WalkthroughCharacterRef{
				Object:      "character",
				ID:          repr.DecimalID(*ch.CharacterID),
				CatalogName: repr.NewCatalogName(*ch.DisplayName, "", localized(client.LocalizedValues(ch.Localized))),
			}
		}
		out.ChartCharacters = append(out.ChartCharacters, c)
	}
	if w.Chart != nil {
		out.Chart = chartFrom(*w.Chart)
	}
	return out
}

func chartFrom(c client.CatalogWalkthroughChart) *WalkthroughChart {
	out := &WalkthroughChart{
		RouteSections: make([]WalkthroughSection, 0, len(c.Sections)),
		EntryPoints:   make([]WalkthroughEntry, 0, len(c.Entries)),
		SavePoints:    make([]WalkthroughSave, 0, len(c.Saves)),
		ChartEndings:  make([]WalkthroughEnding, 0, len(c.Endings)),
		ChartSegments: make([]WalkthroughSegment, 0, len(c.Segments)),
	}
	for _, s := range c.Sections {
		out.RouteSections = append(out.RouteSections, WalkthroughSection{
			Key: ChartKey(s.Key), Title: s.Title, Localized: localized(client.LocalizedValues(s.Localized)), CharacterKeys: keys(s.CharacterKeys),
		})
	}
	for _, e := range c.Entries {
		out.EntryPoints = append(out.EntryPoints, WalkthroughEntry{Key: ChartKey(e.Key), Label: e.Label, UnlockedByEndingKey: keyPtr(e.UnlockedByEndingKey)})
	}
	for _, s := range c.Saves {
		out.SavePoints = append(out.SavePoints, WalkthroughSave{Key: ChartKey(s.Key), Label: s.Label})
	}
	for _, e := range c.Endings {
		out.ChartEndings = append(out.ChartEndings, WalkthroughEnding{
			Key: ChartKey(e.Key), EndingName: e.Name, Localized: localized(client.LocalizedValues(e.Localized)),
			EndingType: e.EndingType, CharacterKeys: keys(e.CharacterKeys),
		})
	}
	node := func(n client.CatalogWalkthroughNode) WalkthroughNode {
		return WalkthroughNode{NodeType: n.NodeType, NodeKey: keyPtr(n.Key)}
	}
	for _, s := range c.Segments {
		seg := WalkthroughSegment{
			Key: ChartKey(s.Key), SectionKey: keyPtr(s.SectionKey), FromNode: node(s.FromNode), ToNode: node(s.ToNode),
			Condition: s.Condition, ConditionLocalized: localized(client.LocalizedValues(s.ConditionLocalized)),
			Steps: make([]WalkthroughStep, 0, len(s.Steps)),
		}
		for _, st := range s.Steps {
			step := WalkthroughStep{
				StepType: st.StepType, StepText: st.Text, Localized: localized(client.LocalizedValues(st.Localized)),
				IsAchievementOnly: st.Achievement, SaveKey: keyPtr(st.SaveKey),
				Alternatives: make([]WalkthroughAlternative, 0, len(st.Alternatives)),
			}
			for _, a := range st.Alternatives {
				step.Alternatives = append(step.Alternatives, WalkthroughAlternative{Text: a.Text, Localized: localized(client.LocalizedValues(a.Localized))})
			}
			seg.Steps = append(seg.Steps, step)
		}
		out.ChartSegments = append(out.ChartSegments, seg)
	}
	return out
}
