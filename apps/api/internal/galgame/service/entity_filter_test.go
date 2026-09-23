package service

import (
	"testing"

	"kun-galgame-api/internal/galgame/model"
)

// The sort chips on an entity page are the forum's, and only the release date is
// a column catalog also holds. Answering the rest out of catalog's vocabulary
// would silently rank the page by something the reader did not ask for.
func TestCatalogMemberSortOnlyClaimsTheReleaseDate(t *testing.T) {
	for _, tc := range []struct{ field, order, want string }{
		{"release_date", "desc", "released_desc"},
		{"release_date", "asc", "released_asc"},
		{"view", "desc", ""},
		{"rating", "desc", ""},
		{"time", "desc", ""},
		{"", "desc", ""},
	} {
		f := model.GalgameListFilter{SortField: tc.field, SortOrder: tc.order}
		if got := catalogMemberSort(f); got != tc.want {
			t.Errorf("catalogMemberSort(%q/%q) = %q, want %q", tc.field, tc.order, got, tc.want)
		}
	}
}

// Entity pages list every catalog member since 方案③, so this may not assert
// true: a member with no forum resource drew a "0 浏览 0 点赞" strip and an
// empty byline, the same card defect galgame_enricher.go records.
