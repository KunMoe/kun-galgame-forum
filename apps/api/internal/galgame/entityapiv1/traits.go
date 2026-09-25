package entityapiv1

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/problem"
)

const traitIndexTTL = time.Hour

type traitNode struct {
	id          int
	name        repr.CatalogName
	sexual      bool
	searchable  bool
	count       int
	sfwCount    int
	order       int
	parents     []int
	children    []int
	group       int
	aliases     []string
	description string
	haystack    []string
}

type traitVocab struct {
	byID  map[int]*traitNode
	roots []int
}

func (s *Service) traitVocab(ctx context.Context) (*traitVocab, error) {
	rows, err := s.traits.get(ctx, traitIndexTTL, s.buildTraitVocab)
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (s *Service) buildTraitVocab(ctx context.Context) ([]traitVocab, error) {
	wire, appErr := s.catalog.CatalogTraitVocabulary(ctx)
	if appErr != nil {
		return nil, appErr
	}
	return []traitVocab{newTraitVocab(wire)}, nil
}

func newTraitVocab(wire []client.CatalogTrait) traitVocab {
	v := traitVocab{byID: make(map[int]*traitNode, len(wire))}
	for i := range wire {
		t := &wire[i]
		order := math.MaxInt
		if t.RootOrder != nil {
			order = *t.RootOrder
		}
		n := &traitNode{
			id:          int(t.ID),
			name:        traitName(t.DisplayName, t.NameZh, client.LocalizedValues(t.Localized)),
			sexual:      t.Sexual,
			searchable:  t.Searchable,
			count:       max(t.CharacterCount, 0),
			sfwCount:    max(t.SFWCharacterCount, 0),
			order:       order,
			aliases:     t.Aliases,
			description: t.Description,
		}
		for _, p := range t.ParentIDs() {
			n.parents = append(n.parents, int(p))
		}
		n.haystack = traitHaystack(n)
		v.byID[n.id] = n
	}
	for _, n := range v.byID {
		n.parents = slices.DeleteFunc(n.parents, func(p int) bool { return v.byID[p] == nil })
		if len(n.parents) == 0 {
			v.roots = append(v.roots, n.id)
		}
		for _, p := range n.parents {
			v.byID[p].children = append(v.byID[p].children, n.id)
		}
	}
	v.sortByOrder(v.roots)
	for _, n := range v.byID {
		v.sortByCount(n.children)
	}
	for _, root := range v.roots {
		v.assignGroup(root, root)
	}
	return v
}

func (v *traitVocab) sortByOrder(ids []int) {
	slices.SortFunc(ids, func(a, b int) int {
		return cmp.Or(cmp.Compare(v.byID[a].order, v.byID[b].order), cmp.Compare(a, b))
	})
}

func (v *traitVocab) sortByCount(ids []int) {
	slices.SortFunc(ids, func(a, b int) int {
		return cmp.Or(cmp.Compare(v.byID[b].count, v.byID[a].count), cmp.Compare(a, b))
	})
}

func (n *traitNode) countFor(includeNSFW bool) int {
	if includeNSFW {
		return n.count
	}
	return n.sfwCount
}

func (v *traitVocab) assignGroup(id, group int) {
	n := v.byID[id]
	if n.group != 0 {
		return
	}
	n.group = group
	for _, c := range n.children {
		v.assignGroup(c, group)
	}
}

func traitHaystack(n *traitNode) []string {
	out := []string{strings.ToLower(n.name.DisplayName)}
	if n.name.Latin != nil {
		out = append(out, strings.ToLower(*n.name.Latin))
	}
	for _, l := range n.name.Localized {
		out = append(out, strings.ToLower(l.Value))
	}
	for _, a := range n.aliases {
		out = append(out, strings.ToLower(a))
	}
	return out
}

func (n *traitNode) matchRank(q string) int {
	best := -1
	for _, h := range n.haystack {
		switch {
		case h == q:
			return 0
		case strings.HasPrefix(h, q):
			best = 1
		case best < 0 && strings.Contains(h, q):
			best = 2
		}
	}
	return best
}

func (v *traitVocab) visible(id int, includeNSFW bool) (*traitNode, bool) {
	n, ok := v.byID[id]
	if !ok || (n.sexual && !includeNSFW) {
		return nil, false
	}
	return n, true
}

func (v *traitVocab) ref(n *traitNode) TraitRef {
	return TraitRef{Object: "trait", ID: repr.ID(n.id), CatalogName: n.name}
}

func (v *traitVocab) summary(n *traitNode, includeNSFW bool) TraitSummary {
	parents := make([]TraitRef, 0, len(n.parents))
	for _, p := range n.parents {
		if pn, ok := v.visible(p, includeNSFW); ok {
			parents = append(parents, v.ref(pn))
		}
	}
	children := 0
	for _, c := range n.children {
		if _, ok := v.visible(c, includeNSFW); ok {
			children++
		}
	}
	return TraitSummary{
		Object:         "trait",
		ID:             repr.ID(n.id),
		CatalogName:    n.name,
		TraitGroup:     v.byID[n.group].name,
		TraitGroupID:   repr.ID(n.group),
		Parents:        parents,
		IsSexual:       n.sexual,
		IsSearchable:   n.searchable,
		CharacterCount: n.countFor(includeNSFW),
		ChildCount:     children,
	}
}

func (v *traitVocab) subtraits(id int, includeNSFW bool) []TraitSummary {
	out := []TraitSummary{}
	seen := map[int]bool{id: true}
	level := []int{id}
	for depth := 0; depth < 2; depth++ {
		var next []int
		for _, p := range level {
			for _, c := range v.byID[p].children {
				n, ok := v.visible(c, includeNSFW)
				if !ok || seen[c] {
					continue
				}
				seen[c] = true
				out = append(out, v.summary(n, includeNSFW))
				next = append(next, c)
			}
		}
		level = next
	}
	return out
}

type listTraitsInput struct {
	Q           string           `query:"q" maxLength:"100" doc:"Name search over every name and alias, case-insensitive: exact names first, then prefixes, then other matches, most characters first within each. Reaches the first 100 matches. Free text; never use it as a decision input. Mutually exclusive with ids and parent_id."`
	IDs         []repr.DecimalID `query:"ids" maxItems:"100" doc:"Trait ids to resolve, comma-separated. 1 to 100 of them, answered in request order. Absent ids are omitted. Mutually exclusive with q and parent_id."`
	ParentID    string           `query:"parent_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only the traits directly below this one, most characters first. Mutually exclusive with q and ids."`
	Page        int              `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000, or 100 when q is set."`
	Limit       int              `query:"limit" minimum:"1" maximum:"100" default:"100" doc:"Page size. 1–100, default 100. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool             `query:"include_nsfw" default:"false" doc:"When true, adult traits are included. Default false."`
}

type listTraitsOutput struct {
	Body repr.PageList[TraitSummary]
}

func (s *Service) listTraits(ctx context.Context, in *listTraitsInput) (*listTraitsOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q := strings.ToLower(trimQuery(in.Q))
	set := 0
	for _, on := range []bool{q != "", len(in.IDs) > 0, in.ParentID != ""} {
		if on {
			set++
		}
	}
	if set > 1 {
		return nil, problem.New(problem.CodeInvalidParameter, "Only one of q, ids and parent_id may be set.",
			problem.AtParameter("q", problem.ReasonInconsistentWith, "ids, parent_id", nil))
	}
	if prob := checkDepth(in.Page, in.Limit, q != ""); prob != nil {
		return nil, prob
	}
	v, err := s.traitVocab(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	var nodes []*traitNode
	switch {
	case len(in.IDs) > 0:
		ids, prob := parseEntityIDs(in.IDs)
		if prob != nil {
			return nil, prob
		}
		for _, id := range uniqueIDs(ids) {
			if n, ok := v.visible(id, in.IncludeNSFW); ok {
				nodes = append(nodes, n)
			}
		}
	case in.ParentID != "":
		id, _ := pathID(in.ParentID)
		if p, ok := v.visible(id, in.IncludeNSFW); ok {
			for _, c := range p.children {
				if n, ok := v.visible(c, in.IncludeNSFW); ok {
					nodes = append(nodes, n)
				}
			}
		}
	case q != "":
		nodes = v.search(q, in.IncludeNSFW)
	default:
		for _, id := range v.roots {
			if n, ok := v.visible(id, in.IncludeNSFW); ok {
				nodes = append(nodes, n)
			}
		}
	}
	rows := make([]TraitSummary, 0, len(nodes))
	for _, n := range nodes {
		rows = append(rows, v.summary(n, in.IncludeNSFW))
	}
	return &listTraitsOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

func (v *traitVocab) search(q string, includeNSFW bool) []*traitNode {
	type hit struct {
		n    *traitNode
		rank int
	}
	var hits []hit
	for _, n := range v.byID {
		if n.sexual && !includeNSFW {
			continue
		}
		if r := n.matchRank(q); r >= 0 {
			hits = append(hits, hit{n, r})
		}
	}
	slices.SortFunc(hits, func(a, b hit) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(b.n.countFor(includeNSFW), a.n.countFor(includeNSFW)), cmp.Compare(a.n.id, b.n.id))
	})
	out := make([]*traitNode, 0, min(len(hits), searchDepth))
	for _, h := range hits[:min(len(hits), searchDepth)] {
		out = append(out, h.n)
	}
	return out
}

type traitPathInput struct {
	TraitID     string `path:"trait_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Trait id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, an adult trait answers. Default false: it is NOT_FOUND."`
}

type getTraitOutput struct {
	Body Trait
}

func (s *Service) getTrait(ctx context.Context, in *traitPathInput) (*getTraitOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	id, ok := pathID(in.TraitID)
	if !ok {
		return nil, notFound()
	}
	v, err := s.traitVocab(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	n, ok := v.visible(id, in.IncludeNSFW)
	if !ok {
		return nil, notFound()
	}
	sum := v.summary(n, in.IncludeNSFW)
	return &getTraitOutput{Body: Trait{
		Object:         sum.Object,
		ID:             sum.ID,
		CatalogName:    sum.CatalogName,
		TraitGroup:     sum.TraitGroup,
		TraitGroupID:   sum.TraitGroupID,
		Parents:        sum.Parents,
		IsSexual:       sum.IsSexual,
		IsSearchable:   sum.IsSearchable,
		CharacterCount: sum.CharacterCount,
		ChildCount:     sum.ChildCount,
		Aliases:        aliases(n.aliases),
		Description:    n.description,
		Subtraits:      v.subtraits(n.id, in.IncludeNSFW),
	}}, nil
}
