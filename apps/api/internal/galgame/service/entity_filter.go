package service

import (
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
)

func buildEntityFilter(q url.Values) model.GalgameListFilter {
	sortOrder := q.Get("sortOrder")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	return model.GalgameListFilter{
		Type:           q.Get("type"),
		Language:       q.Get("language"),
		Platform:       q.Get("platform"),
		GameType:       q.Get("gameType"),
		SortField:      q.Get("sortField"),
		SortOrder:      sortOrder,
		Page:           atoiOr(q.Get("page"), 1),
		Limit:          atoiOr(q.Get("limit"), 24),
		ShowNoResource: q.Get("showNoResource") == "true",
	}
}

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

func entityMemberQuery(key, id string, f model.GalgameListFilter) url.Values {
	q := url.Values{key: {id}}
	if sort := catalogMemberSort(f); sort != "" {
		q.Set("sort", sort)
	}
	return q
}

func listCardsToEntityCards(cards []dto.GalgameListCard) []dto.GalgameCard {
	out := make([]dto.GalgameCard, len(cards))
	for i, c := range cards {
		out[i] = dto.GalgameCard{
			ID:                         c.ID,
			Name:                       c.Name,
			NameOriginal:               c.NameOriginal,
			User:                       c.User,
			ContentLimit:               c.ContentLimit,
			View:                       c.View,
			LikeCount:                  c.LikeCount,
			Rating:                     c.Rating,
			RatingCount:                c.RatingCount,
			ResourceUpdateTime:         c.ResourceUpdateTime,
			Platform:                   c.Platform,
			Language:                   c.Language,
			ReleaseDate:                c.ReleaseDate,
			ReleaseDateTBA:             c.ReleaseDateTBA,
			EffectiveBannerHash:        c.EffectiveBannerHash,
			EffectiveBannerURL:         c.EffectiveBannerURL,
			EffectiveBannerWidth:       c.EffectiveBannerWidth,
			EffectiveBannerHeight:      c.EffectiveBannerHeight,
			EffectiveBannerThumbhash:   c.EffectiveBannerThumbhash,
			EffectivePortraitHash:      c.EffectivePortraitHash,
			EffectivePortraitURL:       c.EffectivePortraitURL,
			EffectivePortraitWidth:     c.EffectivePortraitWidth,
			EffectivePortraitHeight:    c.EffectivePortraitHeight,
			EffectivePortraitThumbhash: c.EffectivePortraitThumbhash,
			IsOnForum:                  c.IsOnForum,
			Company:                    c.Company,
		}
	}
	return out
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
