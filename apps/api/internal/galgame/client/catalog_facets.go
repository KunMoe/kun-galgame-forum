package client

import (
	"context"
	"net/url"
	"slices"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

const (
	EntityFacets   = "tag_id,company_id"
	entityFacetCap = 30
	companyPageCap = 50
)

type CatalogFacetCount struct {
	ID    int
	Count int
}

type catFacetBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type catFacets map[string][]catFacetBucket

func (f catFacets) top(name string) []CatalogFacetCount {
	out := make([]CatalogFacetCount, 0, len(f[name]))
	for _, b := range f[name] {
		if id, err := strconv.Atoi(b.Value); err == nil && id > 0 && b.Count > 0 {
			out = append(out, CatalogFacetCount{ID: id, Count: b.Count})
		}
	}
	slices.SortFunc(out, func(a, b CatalogFacetCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return a.ID - b.ID
	})
	return out[:min(len(out), entityFacetCap)]
}

func (f catFacets) Tags() []CatalogFacetCount      { return f.top("tag_id") }
func (f catFacets) Companies() []CatalogFacetCount { return f.top("company_id") }

// ok is false when neither filter is set; ok with no ids matches nothing.
func (c *GalgameClient) EntityWorkIDs(ctx context.Context, tagIDs []int64, companyID int64) (ids []int, ok bool, appErr *errors.AppError) {
	var sets [][]int
	for _, tag := range tagIDs {
		got, appErr := c.ExcludedWorkIDs(ctx, NewTagExclusion([]int64{tag}, nil))
		if appErr != nil {
			return nil, false, appErr
		}
		sets = append(sets, got)
	}
	if companyID > 0 {
		got, appErr := c.CatalogMemberWorkIDs(ctx, url.Values{"company_id": {strconv.FormatInt(companyID, 10)}}, false, companyPageCap)
		if appErr != nil {
			return nil, false, appErr
		}
		sets = append(sets, got)
	}
	if len(sets) == 0 {
		return nil, false, nil
	}
	return intersect(sets), true, nil
}

func intersect(sets [][]int) []int {
	slices.SortFunc(sets, func(a, b []int) int { return len(a) - len(b) })
	keep := make(map[int]bool, len(sets[0]))
	for _, id := range sets[0] {
		keep[id] = true
	}
	for _, set := range sets[1:] {
		next := make(map[int]bool, len(keep))
		for _, id := range set {
			if keep[id] {
				next[id] = true
			}
		}
		keep = next
	}
	out := make([]int, 0, len(keep))
	for id := range keep {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
