package entityapiv1

import (
	"context"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

const (
	companyIndexTTL     = 30 * time.Minute
	companyIndexPageCap = 500
)

var companyKinds = map[string]bool{
	"game_brand": true, "bunko": true, "publisher": true,
	"anime_studio": true, "doujin_circle": true, "group": true,
}

var companyRelations = map[string]bool{
	"parent": true, "subsidiary": true, "imprint": true, "imprint_of": true,
	"succeeded_by": true, "formerly": true, "spawned": true, "origin": true,
}

func (s *Service) companyIndex(ctx context.Context) ([]CompanySummary, error) {
	return s.companies.get(ctx, companyIndexTTL, s.buildCompanyIndex)
}

func (s *Service) buildCompanyIndex(ctx context.Context) ([]CompanySummary, error) {
	rows := make([]CompanySummary, 0, 4096)
	cursor := ""
	for page := 0; page < companyIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"has_works": {"1"}, "limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.catalog.CatalogTaxonomyList(ctx, "labels", q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			o := &res.Items[i]
			if !companyKinds[o.Kind] {
				slog.Warn("company index: unknown company_kind, row dropped", "company_id", o.ID, "company_kind", o.Kind)
				continue
			}
			name := workrepr.Name(o.DisplayName, o.Latin, client.LocalizedValues(o.Localized))
			rows = append(rows, CompanySummary{
				Object:           "company",
				ID:               repr.ID(int(o.ID)),
				CatalogName:      name,
				CompanyKind:      o.Kind,
				Logo:             workrepr.ImageFromHash(s.cdn, o.LogoHash),
				Aliases:          aliases(o.Aliases.Values(name.DisplayName)),
				CatalogWorkCount: max(o.WorkCount, 0),
			})
		}
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].CatalogWorkCount != rows[j].CatalogWorkCount {
			return rows[i].CatalogWorkCount > rows[j].CatalogWorkCount
		}
		return idLess(rows[i].ID, rows[j].ID)
	})
	return rows, nil
}

type listCompaniesInput struct {
	Q           string           `query:"q" maxLength:"100" doc:"Name search. Set, the collection is catalog's 100 best name matches in relevance order. Free text; never use it as a decision input. Mutually exclusive with ids."`
	IDs         []repr.DecimalID `query:"ids" maxItems:"100" doc:"Company ids to resolve, comma-separated. 1 to 100 of them. Mutually exclusive with q. Absent ids are omitted."`
	CompanyKind string           `query:"company_kind" enum:"game_brand,bunko,publisher,anime_studio,doujin_circle,group" maxLength:"13" doc:"Only companies of this kind. Omitted means every kind."`
	Page        int              `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000, or 100 when q is set."`
	Limit       int              `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
}

type listCompaniesOutput struct {
	Body repr.PageList[CompanySummary]
}

func (s *Service) listCompanies(ctx context.Context, in *listCompaniesInput) (*listCompaniesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q := trimQuery(in.Q)
	if q != "" && len(in.IDs) > 0 {
		return nil, problem.New(problem.CodeInvalidParameter, "q and ids cannot both be set.",
			problem.AtParameter("ids", problem.ReasonInconsistentWith, "q", nil))
	}
	if prob := checkDepth(in.Page, in.Limit, q != ""); prob != nil {
		return nil, prob
	}
	index, err := s.companyIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	var rows []CompanySummary
	if len(in.IDs) > 0 {
		found, prob := s.companiesByIDs(ctx, in.IDs, index)
		if prob != nil {
			return nil, prob
		}
		rows = found
	} else if q == "" {
		rows = index
	} else {
		hits, _, appErr := s.catalog.CatalogEntitySearch(ctx, "labels", q, 1, searchDepth)
		if appErr != nil {
			return nil, unavailable(appErr)
		}
		byID := make(map[repr.DecimalID]CompanySummary, len(index))
		for _, r := range index {
			byID[r.ID] = r
		}
		rows = make([]CompanySummary, 0, len(hits))
		for i := range hits {
			h := &hits[i]
			row, ok := byID[repr.ID(int(h.ID))]
			if !ok {
				if !companyKinds[h.Kind] {
					continue
				}
				row = CompanySummary{
					Object:      "company",
					ID:          repr.ID(int(h.ID)),
					CatalogName: workrepr.Name(h.DisplayName, h.Latin, client.LocalizedValues(h.Localized)),
					CompanyKind: h.Kind,
					Aliases:     []AliasName{},
				}
			}
			rows = append(rows, row)
		}
	}
	if in.CompanyKind != "" {
		kept := make([]CompanySummary, 0, len(rows))
		for _, r := range rows {
			if r.CompanyKind == in.CompanyKind {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	return &listCompaniesOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

func (s *Service) companiesByIDs(ctx context.Context, parts []repr.DecimalID, index []CompanySummary) ([]CompanySummary, *problem.Problem) {
	ids, prob := parseEntityIDs(parts)
	if prob != nil {
		return nil, prob
	}
	ids = uniqueIDs(ids)
	byID := make(map[repr.DecimalID]CompanySummary, len(index))
	for _, r := range index {
		byID[r.ID] = r
	}
	out := make([]CompanySummary, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[repr.ID(id)]; ok {
			out = append(out, row)
			continue
		}
		o, found, movedTo, appErr := s.catalog.CatalogLabel(ctx, strconv.Itoa(id))
		if appErr != nil {
			return nil, unavailable(appErr)
		}
		if movedTo != 0 || !found {
			continue
		}
		if !companyKinds[o.Kind] {
			slog.Warn("list companies ids: unknown company_kind, row dropped", "company_id", o.ID, "company_kind", o.Kind)
			continue
		}
		name := workrepr.Name(o.DisplayName, o.Latin, client.LocalizedValues(o.Localized))
		out = append(out, CompanySummary{
			Object:           "company",
			ID:               repr.ID(int(o.ID)),
			CatalogName:      name,
			CompanyKind:      o.Kind,
			Logo:             workrepr.ImageFromHash(s.cdn, o.LogoHash),
			Aliases:          aliases(o.Aliases.Values(name.DisplayName)),
			CatalogWorkCount: max(o.WorkCount, 0),
		})
	}
	return out, nil
}

type companyPathInput struct {
	CompanyID string `path:"company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Company id."`
}

type getCompanyOutput struct {
	Body Company
}

func (s *Service) companyHead(ctx context.Context, rawID string) (*client.CatalogLabelDetail, *problem.Problem) {
	if _, ok := pathID(rawID); !ok {
		return nil, notFound()
	}
	o, found, movedTo, appErr := s.catalog.CatalogLabel(ctx, rawID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if movedTo != 0 {
		return nil, merged("company", movedTo)
	}
	if !found {
		return nil, notFound()
	}
	return o, nil
}

func (s *Service) getCompany(ctx context.Context, in *companyPathInput) (*getCompanyOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	o, prob := s.companyHead(ctx, in.CompanyID)
	if prob != nil {
		return nil, prob
	}
	if !companyKinds[o.Kind] {
		return nil, problem.Internal(errUnknownVocab("company_kind", o.Kind))
	}
	name := workrepr.Name(o.DisplayName, o.Latin, client.LocalizedValues(o.Localized))
	links := make([]workrepr.CatalogLink, 0, len(o.Links))
	for _, l := range o.Links {
		links = workrepr.AppendLink(links, l.Source, l.URL)
	}
	var lang *string
	if o.Lang != "" {
		lang = &o.Lang
	}
	return &getCompanyOutput{Body: Company{
		Object:           "company",
		ID:               repr.ID(int(o.ID)),
		CatalogName:      name,
		CompanyKind:      o.Kind,
		Logo:             workrepr.ImageFromHash(s.cdn, o.LogoHash),
		Aliases:          aliases(o.Aliases.Values(name.DisplayName)),
		CatalogWorkCount: max(o.WorkCount, 0),
		Lang:             lang,
		Links:            links,
		Intros:           workrepr.Intros(o.Intros),
	}}, nil
}

type listCompanyWorksInput struct {
	CompanyID string `path:"company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Company id."`
	Via       string `query:"via" enum:"own,imprint" maxLength:"7" doc:"own: only the company's own works. imprint: only works credited to one of its imprints. Omitted means both."`
	WorksQuery
}

type listCompanyWorksOutput struct {
	Body repr.PageList[CompanyWork]
}

func (s *Service) listCompanyWorks(ctx context.Context, in *listCompanyWorksInput) (*listCompanyWorksOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if _, prob := s.companyHead(ctx, in.CompanyID); prob != nil {
		return nil, prob
	}
	if prob := checkDepth(in.Page, in.Limit, false); prob != nil {
		return nil, prob
	}
	walk := func(ctx context.Context, catalogSort string, isSFW bool) ([]member, *legacyErrors.AppError) {
		rollup, appErr := s.catalog.CatalogLabelRollupMembers(ctx, in.CompanyID, catalogSort, isSFW, memberPageCap)
		if appErr != nil {
			return nil, appErr
		}
		out := make([]member, len(rollup))
		for i, m := range rollup {
			out[i] = member{workID: m.WorkID, via: m.Via}
		}
		return out, nil
	}
	var keep func(member) bool
	switch in.Via {
	case "own":
		keep = func(m member) bool { return m.via == nil }
	case "imprint":
		keep = func(m member) bool { return m.via != nil }
	}
	ids, via, count, prob := s.memberPage(ctx, walk, in.WorksQuery, keep)
	if prob != nil {
		return nil, prob
	}
	works, prob := s.works.ByIDs(ctx, ids, in.IncludeNSFW)
	if prob != nil {
		return nil, prob
	}
	items := make([]CompanyWork, len(works))
	for i, w := range works {
		item := CompanyWork{Object: "company_work", WorkSummary: w}
		if id, ok := repr.ParseID(w.ID); ok {
			if v := via[id]; v != nil {
				ref := workrepr.CompanyRefOf(v.ID, v.DisplayName, "", client.LocalizedValues(v.Localized))
				item.ViaCompany = &ref
			}
		}
		items[i] = item
	}
	total, relation := collect.ClampTotal(count)
	return &listCompanyWorksOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

type companyGraphOutput struct {
	Body CompanyGraph
}

func (s *Service) getCompanyGraph(ctx context.Context, in *companyPathInput) (*companyGraphOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if _, ok := pathID(in.CompanyID); !ok {
		return nil, notFound()
	}
	graph, found, appErr := s.catalog.CatalogLabelRelationGraph(ctx, in.CompanyID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if !found {
		return nil, notFound()
	}
	out := CompanyGraph{
		Object:    "company_graph",
		CompanyID: repr.DecimalID(in.CompanyID),
		Nodes:     make([]CompanyGraphNode, 0, len(graph.Nodes)),
		Edges:     make([]CompanyGraphEdge, 0, len(graph.Edges)),
	}
	for _, n := range graph.Nodes {
		display := n.DisplayName
		if display == "" {
			display = n.Name
		}
		out.Nodes = append(out.Nodes, CompanyGraphNode{
			Object:           "company",
			ID:               repr.ID(int(n.ID)),
			CatalogName:      workrepr.Name(display, "", client.LocalizedValues(n.Localized)),
			Logo:             workrepr.ImageFromHash(s.cdn, n.LogoHash),
			CatalogWorkCount: max(n.WorkCount, 0),
		})
	}
	for _, e := range graph.Edges {
		if !companyRelations[e.Relation] {
			slog.Warn("company graph: unknown relation, edge dropped", "company_id", in.CompanyID, "relation", e.Relation)
			continue
		}
		out.Edges = append(out.Edges, CompanyGraphEdge{
			FromCompanyID: repr.ID(int(e.From)),
			ToCompanyID:   repr.ID(int(e.To)),
			Relation:      e.Relation,
		})
	}
	return &companyGraphOutput{Body: out}, nil
}

type wikiCompanyInput struct {
	WikiCompanyID string `path:"wiki_company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The company id the retired galgame wiki used."`
}

type wikiCompanyOutput struct {
	Body WikiCompanyRedirect
}

func (s *Service) getWikiCompanyRedirect(ctx context.Context, in *wikiCompanyInput) (*wikiCompanyOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	wikiID, ok := pathID(in.WikiCompanyID)
	if !ok {
		return nil, notFound()
	}
	id, found, appErr := s.catalog.LookupWikiLabel(ctx, wikiID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if !found || id <= 0 {
		return nil, notFound()
	}
	return &wikiCompanyOutput{Body: WikiCompanyRedirect{
		Object:        "wiki_company_redirect",
		WikiCompanyID: repr.ID(wikiID),
		CompanyID:     repr.DecimalID(strconv.FormatInt(id, 10)),
	}}, nil
}
