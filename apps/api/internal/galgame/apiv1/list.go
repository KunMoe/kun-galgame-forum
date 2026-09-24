package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/utils"
)

type workSummaryPageOutput struct {
	Body repr.PageList[workrepr.WorkSummary]
}

func browsePage(page, limit int) collect.PageNumber {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultBrowseLimit
	}
	return collect.PageNumber{Page: page, Limit: limit}
}

func (s *Service) listWorks(ctx context.Context, in *listWorksInput) (*workSummaryPageOutput, error) {
	if s == nil || s.lists == nil || s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	field, order, ok := entityapiv1.WorkSortParts(string(in.Sort))
	if !ok {
		return nil, problem.New(problem.CodeUnknownSort, "The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "not in this collection's closed vocabulary", nil))
	}
	releasedFrom, releasedTo, prob := boundRange(in.ReleasedFrom, in.ReleasedTo, "released_from", "released_to", true)
	if prob != nil {
		return nil, prob
	}
	collectedFrom, collectedTo, prob := boundRange(in.CollectedFrom, in.CollectedTo, "collected_from", "collected_to", false)
	if prob != nil {
		return nil, prob
	}
	f := model.GalgameListFilter{
		Type:                 string(in.ResourceType),
		PlatformAxes:         strKeys(in.ResourcePlatforms),
		LanguageAxes:         strKeys(in.ResourceLanguages),
		RuntimeAxes:          strKeys(in.ResourceRuntimes),
		GameType:             string(in.GameType),
		SortField:            field,
		SortOrder:            order,
		IncludeProviders:     providerKeys(in.ResourceProviders),
		ExcludeOnlyProviders: providerKeys(in.ExcludedSoleProviders),
		ReleasedFrom:         releasedFrom,
		ReleasedTo:           releasedTo,
		ReleasedMonths:       monthNums(in.ReleasedMonths),
		CollectedFrom:        collectedFrom,
		CollectedTo:          collectedTo,
		CollectedMonths:      monthNums(in.CollectedMonths),
		MinRating:            in.MinRating,
		MinRatingCount:       in.MinRatingCount,
		ShowNoResource:       in.IncludeResourceless,
		Indexed:              in.IncludeResourceless,
		SFWOnly:              !in.IncludeNSFW,
		Page:                 pg.Page,
		Limit:                pg.Limit,
	}
	ids, total, err := s.lists.ListIDs(f)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.hydrator.ByIDs(ctx, ids, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	// The total is what the paginator divides into pages, so it must not depend
	// on which page came back: subtracting the rows this page hid made the page
	// count change every time the reader moved. Pages that lost rows are short.
	n, rel := collect.ClampTotal(int(total))
	return &workSummaryPageOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func boundRange(from, to, fromName, toName string, release bool) (lo, hi string, prob *problem.Problem) {
	var err error
	if release {
		lo, err = utils.ParseReleaseLowerBound(from)
	} else {
		lo, err = utils.ParseDateLowerBound(from, fromName)
	}
	if err != nil {
		return "", "", problem.New(problem.CodeInvalidParameter, fromName+" is not a year or year-month.",
			problem.AtParameter(fromName, problem.ReasonInvalidFormat, "must be YYYY or YYYY-MM", nil))
	}
	if release {
		hi, err = utils.ParseReleaseUpperBound(to)
	} else {
		hi, err = utils.ParseDateUpperBound(to, toName)
	}
	if err != nil {
		return "", "", problem.New(problem.CodeInvalidParameter, toName+" is not a year or year-month.",
			problem.AtParameter(toName, problem.ReasonInvalidFormat, "must be YYYY or YYYY-MM", nil))
	}
	if lo != "" && hi != "" && lo > hi {
		return "", "", problem.New(problem.CodeInvalidParameter, fromName+" is after "+toName+".",
			problem.AtParameter(fromName, problem.ReasonInconsistentWith, toName, nil))
	}
	return lo, hi, nil
}

func monthNums(in []MonthNumber) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(in))
	for _, m := range in {
		n := int(m)
		if n < 1 || n > 12 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func strKeys[T ~string](in []T) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		s := string(v)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func providerKeys(in []workrepr.ResourceProvider) []string {
	return strKeys(in)
}
