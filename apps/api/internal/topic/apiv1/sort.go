package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"

	"github.com/danielgtaylor/huma/v2"
)

type sortKind int

const (
	sortKindInt sortKind = iota
	sortKindTime
)

type sortSpec struct {
	Token     string
	Key       string
	Direction string
	Kind      sortKind
}

var sortSpecs = []sortSpec{
	{Token: "bumped_asc", Key: "status_update_time", Direction: "asc", Kind: sortKindTime},
	{Token: "bumped_desc", Key: "status_update_time", Direction: "desc", Kind: sortKindTime},
	{Token: "created_asc", Key: "created", Direction: "asc", Kind: sortKindTime},
	{Token: "created_desc", Key: "created", Direction: "desc", Kind: sortKindTime},
	{Token: "views_asc", Key: "view", Direction: "asc", Kind: sortKindInt},
	{Token: "views_desc", Key: "view", Direction: "desc", Kind: sortKindInt},
	{Token: "views_1d_asc", Key: "view_1d", Direction: "asc", Kind: sortKindInt},
	{Token: "views_1d_desc", Key: "view_1d", Direction: "desc", Kind: sortKindInt},
	{Token: "views_7d_asc", Key: "view_7d", Direction: "asc", Kind: sortKindInt},
	{Token: "views_7d_desc", Key: "view_7d", Direction: "desc", Kind: sortKindInt},
	{Token: "views_30d_asc", Key: "view_30d", Direction: "asc", Kind: sortKindInt},
	{Token: "views_30d_desc", Key: "view_30d", Direction: "desc", Kind: sortKindInt},
	{Token: "likes_asc", Key: "like_count", Direction: "asc", Kind: sortKindInt},
	{Token: "likes_desc", Key: "like_count", Direction: "desc", Kind: sortKindInt},
	{Token: "favorites_asc", Key: "favorite_count", Direction: "asc", Kind: sortKindInt},
	{Token: "favorites_desc", Key: "favorite_count", Direction: "desc", Kind: sortKindInt},
	{Token: "upvotes_asc", Key: "upvote_count", Direction: "asc", Kind: sortKindInt},
	{Token: "upvotes_desc", Key: "upvote_count", Direction: "desc", Kind: sortKindInt},
}

var sortByToken = func() map[string]sortSpec {
	m := make(map[string]sortSpec, len(sortSpecs))
	for _, s := range sortSpecs {
		m[s.Token] = s
	}
	return m
}()

const defaultSort = "bumped_desc"

const sortDoc = "Sort order; ties break on id in the same direction. Default bumped_desc. " +
	"bumped: bumped_at. created: created_at. views: view_count. " +
	"views_1d: views today. views_7d and views_30d: views in the last 7 or 30 days, recomputed daily. " +
	"likes: like_count. favorites: times favorited. upvotes: times upvoted."

type SortToken string

func (SortToken) Schema(huma.Registry) *huma.Schema {
	enum := make([]any, len(sortSpecs))
	maxLen := 0
	for i, s := range sortSpecs {
		enum[i] = s.Token
		if n := len(s.Token); n > maxLen {
			maxLen = n
		}
	}
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		Description: sortDoc,
		MaxLength:   &maxLen,
		Default:     defaultSort,
	}
}

func lookupSort(token string) (sortSpec, bool) {
	if token == "" {
		token = defaultSort
	}
	s, ok := sortByToken[token]
	return s, ok
}

// The section joins the fingerprint only when set, so cursors issued before
// the filter existed stay valid.
func listFingerprint(sort, category, section string, includeNSFW, authenticated bool) string {
	values := []string{sort, category, boolKey(includeNSFW), boolKey(authenticated)}
	if section != "" {
		values = append(values, "section:"+section)
	}
	return collect.Fingerprint(values...)
}

func boolKey(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
