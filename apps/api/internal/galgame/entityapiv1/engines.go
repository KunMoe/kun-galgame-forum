package entityapiv1

import (
	"cmp"
	"context"
	"net/url"
	"sort"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

const (
	engineIndexTTL     = 30 * time.Minute
	engineIndexPageCap = 20
)

func (s *Service) engineIndex(ctx context.Context) ([]Engine, error) {
	return s.engines.get(ctx, engineIndexTTL, s.buildEngineIndex)
}

func (s *Service) buildEngineIndex(ctx context.Context) ([]Engine, error) {
	rows := []Engine{}
	cursor := ""
	for page := 0; page < engineIndexPageCap; page++ {
		q := client.OpenPopulation(url.Values{"limit": {"100"}})
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := s.catalog.CatalogTaxonomyList(ctx, "engines", q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			e := &res.Items[i]
			name := workrepr.Name(cmp.Or(e.DisplayName, e.Name), e.Latin, client.LocalizedValues(e.Localized))
			rows = append(rows, Engine{
				Object:           "engine",
				ID:               repr.ID(int(e.ID)),
				CatalogName:      name,
				Aliases:          aliases(e.Aliases.Values(name.DisplayName)),
				Description:      e.Description,
				CatalogWorkCount: max(e.WorkCount, 0),
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

type listEnginesInput struct {
	Q     string `query:"q" maxLength:"100" doc:"Case-insensitive substring of any of the engine's names or aliases. Free text; never use it as a decision input."`
	Page  int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit int    `query:"limit" minimum:"1" maximum:"100" default:"100" doc:"Page size. 1–100, default 100. Values above 100 are rejected, not clamped."`
}

type listEnginesOutput struct {
	Body repr.PageList[Engine]
}

func (s *Service) listEngines(ctx context.Context, in *listEnginesInput) (*listEnginesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if prob := checkDepth(in.Page, in.Limit, false); prob != nil {
		return nil, prob
	}
	index, err := s.engineIndex(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	rows := index
	if q := trimQuery(in.Q); q != "" {
		rows = make([]Engine, 0, len(index))
		for _, e := range index {
			extra := make([]string, len(e.Aliases))
			for i, a := range e.Aliases {
				extra[i] = string(a)
			}
			if nameMatches(q, e.CatalogName, extra...) {
				rows = append(rows, e)
			}
		}
	}
	return &listEnginesOutput{Body: pageList(rows, in.Page, in.Limit)}, nil
}

type enginePathInput struct {
	EngineID string `path:"engine_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Engine id."`
}

type getEngineOutput struct {
	Body Engine
}

func (s *Service) engineHead(ctx context.Context, rawID string) (*client.CatalogEngineDetail, *problem.Problem) {
	if _, ok := pathID(rawID); !ok {
		return nil, notFound()
	}
	e, found, appErr := s.catalog.CatalogEngine(ctx, rawID)
	if appErr != nil {
		return nil, unavailable(appErr)
	}
	if !found {
		return nil, notFound()
	}
	return e, nil
}

func (s *Service) getEngine(ctx context.Context, in *enginePathInput) (*getEngineOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	e, prob := s.engineHead(ctx, in.EngineID)
	if prob != nil {
		return nil, prob
	}
	name := workrepr.Name(cmp.Or(e.DisplayName, e.Name), "", client.LocalizedValues(e.Localized))
	return &getEngineOutput{Body: Engine{
		Object:           "engine",
		ID:               repr.ID(int(e.ID)),
		CatalogName:      name,
		Aliases:          aliases(e.Aliases.Values(name.DisplayName)),
		Description:      e.Description,
		CatalogWorkCount: max(e.WorkCount, 0),
	}}, nil
}

type listEngineWorksInput struct {
	EngineID string `path:"engine_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Engine id."`
	WorksQuery
}

func (s *Service) listEngineWorks(ctx context.Context, in *listEngineWorksInput) (*workPageOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	if _, prob := s.engineHead(ctx, in.EngineID); prob != nil {
		return nil, prob
	}
	page, prob := s.worksPage(ctx, s.walkMembers("engine_id", in.EngineID), in.WorksQuery)
	if prob != nil {
		return nil, prob
	}
	return &workPageOutput{Body: page}, nil
}
