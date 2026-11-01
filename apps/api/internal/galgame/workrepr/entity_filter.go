package workrepr

import (
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
)

type EntityFilterInput struct {
	CompanyID string           `query:"company_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only works by this catalog company."`
	TagIDs    []repr.DecimalID `query:"tag_ids" maxItems:"10" doc:"Only works carrying every one of these catalog tags. Comma-separated, at most 10."`
}

func (in EntityFilterInput) Tags() []int64 {
	return parseTagIDs(in.TagIDs)
}

func (in EntityFilterInput) Company() int64 {
	id, _ := strconv.ParseInt(in.CompanyID, 10, 64)
	return id
}

func (in EntityFilterInput) Apply(q url.Values) url.Values {
	if id := in.Company(); id > 0 {
		q.Set("company_id", strconv.FormatInt(id, 10))
	}
	if tags := in.Tags(); len(tags) > 0 {
		parts := make([]string, len(tags))
		for i, id := range tags {
			parts[i] = strconv.FormatInt(id, 10)
		}
		q.Set("tag_id", strings.Join(parts, ","))
	}
	return q
}

type EntityCount struct {
	ID        repr.DecimalID `json:"id" doc:"Catalog id of the tag or company."`
	WorkCount int            `json:"work_count" minimum:"0" doc:"Works in the filtered result set that carry it."`
}

type WorkFacets struct {
	Tags      []EntityCount `json:"tag_counts" doc:"The tags the most works in the filtered result set carry, most first, at most 30. Empty array, never null."`
	Companies []EntityCount `json:"company_counts" doc:"The companies credited on the most works in the filtered result set, most first, at most 30. Empty array, never null."`
}

func NewWorkFacets(tags, companies []client.CatalogFacetCount) WorkFacets {
	return WorkFacets{Tags: entityCounts(tags), Companies: entityCounts(companies)}
}

func entityCounts(in []client.CatalogFacetCount) []EntityCount {
	out := make([]EntityCount, len(in))
	for i, f := range in {
		out[i] = EntityCount{ID: repr.ID(f.ID), WorkCount: f.Count}
	}
	return out
}

type FacetedWorkPage struct {
	Object        string        `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items         []WorkSummary `json:"items" doc:"Members of this page. Empty array, never null."`
	Total         int           `json:"total" minimum:"0" doc:"Members matching the filters, under the same predicate as items. Counted up to the depth limit when total_relation is gte."`
	TotalRelation string        `json:"total_relation" enum:"eq,gte" maxLength:"3" doc:"eq when total is exact, gte when it stopped at the depth limit and there are at least that many."`
	Facets        WorkFacets    `json:"facets" doc:"The tags and companies most common in the whole filtered result set, not only this page."`
}

func NewFacetedWorkPage(items []WorkSummary, total int, relation string, facets WorkFacets) FacetedWorkPage {
	if items == nil {
		items = []WorkSummary{}
	}
	return FacetedWorkPage{Object: "list", Items: items, Total: total, TotalRelation: relation, Facets: facets}
}
