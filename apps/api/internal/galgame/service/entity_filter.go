package service

import (
	"strconv"

	"kun-galgame-api/internal/galgame/model"
)

// catalogMemberSort is the walk order catalog can answer for itself. The release
// date is the only sort field on these pages that is catalog's own data — every
// member carries one, while view/rating/update exist only for the members that
// are also forum rows — so it is the only one worth asking catalog for.
//
// /v2/catalog/works takes a closed vocabulary (id, updated, relevance,
// released_desc, released_asc, popularity) and answers 400 UNKNOWN_SORT to
// anything else; there is no page parameter, so the whole membership is walked
// either way and asking for an order costs nothing.
//
// A non-empty result means the walk has already ordered the page and the local
// ranking pass must leave it alone, or it reshuffles what the walk established.
// CatalogMemberSort is the catalog walk order for f, empty when the local
// ranking pass orders the members instead.
func CatalogMemberSort(f model.GalgameListFilter) string { return catalogMemberSort(f) }

func catalogMemberSort(f model.GalgameListFilter) string {
	if f.SortField != "release_date" {
		return ""
	}
	if f.SortOrder == "asc" {
		return "released_asc"
	}
	return "released_desc"
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}
