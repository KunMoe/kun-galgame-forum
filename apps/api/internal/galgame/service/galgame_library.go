package service

import "kun-galgame-api/internal/galgame/model"

// EntityUsesLocalList reports whether an entity page's work list leaves the
// catalog membership for local SQL, which only knows resource-carrying rows.
func EntityUsesLocalList(f model.GalgameListFilter) bool { return entityUsesLocalList(f) }

func entityUsesLocalList(f model.GalgameListFilter) bool {
	if f.HasResourcePredicate() {
		return true
	}
	if f.GameType != "" && f.GameType != "all" {
		return true
	}
	if f.MinRatingCount > 0 || f.MinRating > 0 {
		return true
	}
	return false
}
