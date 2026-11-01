package apiv1

import (
	"testing"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
)

func ratedIn(mut func(*ratedWorkRankingInput)) *ratedWorkRankingInput {
	in := &ratedWorkRankingInput{Sort: "lower_desc", Page: 1, Limit: 50}
	if mut != nil {
		mut(in)
	}
	return in
}

func TestRatedQueryDefaultsToTheGatedConsensus(t *testing.T) {
	q, prob := ratedQuery(ratedIn(nil))
	if prob != nil {
		t.Fatal(prob)
	}
	want := map[string]string{
		"metric": "lower", "order": "desc", "page": "1", "limit": "50",
		"include": workrepr.RowInclude, "nsfw": "true", "content_limit": "sfw", "olang": "ja,zh,zh-Hans,zh-Hant",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	for _, k := range []string{"by", "scope", "min_sources", "min_votes", "released_after", "platform"} {
		if q.Has(k) {
			t.Errorf("%s sent without being asked: %s", k, q.Encode())
		}
	}
}

func TestRatedQueryMapsEveryFilter(t *testing.T) {
	q, prob := ratedQuery(ratedIn(func(in *ratedWorkRankingInput) {
		in.Site = "erogamescape"
		in.Sort = "upper_asc"
		in.ReleasedFrom, in.ReleasedTo = "2015", "2020-06"
		in.Circulation = "doujin"
		in.Platforms = []workrepr.ResourcePlatform{"win", "swi"}
		in.MinVoteCount, in.MaxVoteCount = 5, 50
		in.MaxPlaytimeMinutes = 600
		in.IncludeNSFW = true
		in.IncludeAllOriginalLanguages = true
		in.IncludeAllGames = true
		in.CompanyID = "301"
		in.TagIDs = []repr.DecimalID{"32", "1547"}
	}))
	if prob != nil {
		t.Fatal(prob)
	}
	want := map[string]string{
		"by": "erogamescape", "metric": "upper", "order": "asc",
		"released_after": "2015-01-01", "released_before": "2020-06-30",
		"circulation": "doujin", "platform": "win,swi",
		"min_votes": "5", "max_votes": "50", "playtime_max": "600", "nsfw": "true", "scope": "all",
		"company_id": "301", "tag_id": "32,1547", "facets": "tag_id,company_id",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	if q.Has("content_limit") || q.Has("olang") || q.Has("playtime_min") {
		t.Errorf("the open population carries no gate: %s", q.Encode())
	}
}

func TestRatedQueryRefusesWhatCatalogWouldRefuse(t *testing.T) {
	cases := map[string]func(*ratedWorkRankingInput){
		"native without site":       func(in *ratedWorkRankingInput) { in.Sort = "native_desc" },
		"divergence with site":      func(in *ratedWorkRankingInput) { in.Site = "vndb"; in.Sort = "divergence_desc" },
		"min_site_count with site":  func(in *ratedWorkRankingInput) { in.Site = "vndb"; in.MinSiteCount = 2 },
		"vote floor above ceiling":  func(in *ratedWorkRankingInput) { in.MinVoteCount, in.MaxVoteCount = 60, 50 },
		"playtime floor above ceil": func(in *ratedWorkRankingInput) { in.MinPlaytimeMinutes, in.MaxPlaytimeMinutes = 700, 600 },
		"years reversed":            func(in *ratedWorkRankingInput) { in.ReleasedFrom, in.ReleasedTo = "2021", "2020" },
	}
	for name, mut := range cases {
		if _, prob := ratedQuery(ratedIn(mut)); prob == nil || prob.Code != problem.CodeInvalidParameter {
			t.Errorf("%s: %+v", name, prob)
		}
	}
	if _, prob := ratedQuery(ratedIn(func(in *ratedWorkRankingInput) { in.Sort = "rating_desc" })); prob == nil || prob.Code != problem.CodeUnknownSort {
		t.Errorf("unknown sort: %+v", prob)
	}
}
