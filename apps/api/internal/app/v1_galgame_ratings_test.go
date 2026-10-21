package app

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var grSorts = []string{"created_desc", "created_asc", "view_desc", "view_asc", "overall_desc", "overall_asc"}

func grWalk(t *testing.T, f *grFix, session, query string, limit int) ([]string, int) {
	t.Helper()
	var ids []string
	total := -1
	for page := 1; ; page++ {
		resp, out := f.list(t, session, fmt.Sprintf("%s&limit=%d&page=%d", query, limit, page))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s page %d: %d %v", query, page, resp.StatusCode, out)
		}
		n := int(out["total"].(float64))
		if total >= 0 && n != total {
			t.Fatalf("%s: total moved from %d to %d between pages", query, total, n)
		}
		total = n
		ids = append(ids, itemIDs(t, out)...)
		if page*limit >= total {
			return ids, total
		}
	}
}

func grCode(out map[string]any) string {
	s, _ := out["code"].(string)
	return s
}

func grFieldReason(out map[string]any) (pointer, reason string) {
	errs, _ := out["errors"].([]any)
	if len(errs) == 0 {
		return "", ""
	}
	e, _ := errs[0].(map[string]any)
	pointer, _ = e["pointer"].(string)
	reason, _ = e["reason"].(string)
	return pointer, reason
}

func TestV1RatingsListWalk(t *testing.T) {
	f := newGRFix(t)
	for _, nsfw := range []bool{false, true} {
		for _, sort := range grSorts {
			cut := strings.LastIndexByte(sort, '_')
			want, wantTotal := grExpected(nsfw, sort[:cut], sort[cut+1:] == "desc", nil)
			for _, limit := range []int{1, 2, 3, 5} {
				got, total := grWalk(t, f, "", fmt.Sprintf("sort=%s&include_nsfw=%t", sort, nsfw), limit)
				if total != wantTotal {
					t.Errorf("nsfw=%t %s limit %d: total %d, want %d", nsfw, sort, limit, total, wantTotal)
				}
				if !slices.Equal(got, want) {
					t.Errorf("nsfw=%t %s limit %d:\n got %v\nwant %v", nsfw, sort, limit, got, want)
				}
			}
		}
	}
	got, _ := grWalk(t, f, "", "include_nsfw=false", 24)
	want, _ := grExpected(false, "created", true, nil)
	if !slices.Equal(got, want) {
		t.Errorf("default sort is not created_desc:\n got %v\nwant %v", got, want)
	}
}

func TestV1RatingsListFilters(t *testing.T) {
	f := newGRFix(t)
	cases := []struct {
		query string
		keep  func(grRating) bool
	}{
		{"work_id=" + strconv.Itoa(grWorkSFW), func(r grRating) bool { return r.workID == grWorkSFW }},
		{"work_id=" + strconv.Itoa(grWorkHidden), func(r grRating) bool { return r.workID == grWorkHidden }},
		{"author_id=" + strconv.Itoa(grAlice), func(r grRating) bool { return r.userID == grAlice }},
		{"spoiler_level=portion", func(r grRating) bool { return r.spoiler == "portion" }},
		{"play_status=doing", func(r grRating) bool { return r.playStatus == "doing" }},
		{"game_type=moe", func(r grRating) bool { return slices.Contains(r.gameTypes, "moe") }},
		{"game_type=plot&spoiler_level=none", func(r grRating) bool { return slices.Contains(r.gameTypes, "plot") && r.spoiler == "none" }},
	}
	for _, c := range cases {
		for _, nsfw := range []bool{false, true} {
			want, wantTotal := grExpected(nsfw, "overall", true, c.keep)
			got, total := grWalk(t, f, "", fmt.Sprintf("%s&sort=overall_desc&include_nsfw=%t", c.query, nsfw), 2)
			if total != wantTotal || !slices.Equal(got, want) {
				t.Errorf("%s nsfw=%t: got %v (total %d), want %v (total %d)", c.query, nsfw, got, total, want, wantTotal)
			}
		}
	}

	_, out := f.list(t, "", "work_id="+strconv.Itoa(grWorkNSFW))
	if out["total"].(float64) != 0 || len(itemIDs(t, out)) != 0 {
		t.Errorf("NSFW work listed without include_nsfw: %v", out)
	}

	for _, c := range []struct{ query, code string }{
		{"sort=score_desc", "UNKNOWN_SORT"},
		{"spoiler_level=huge", "UNKNOWN_ENUM_VALUE"},
		{"game_type=otome", "UNKNOWN_ENUM_VALUE"},
		{"play_status=finished", "UNKNOWN_ENUM_VALUE"},
		{"work_id=0", "INVALID_PARAMETER"},
		{"author_id=abc", "INVALID_PARAMETER"},
		{"limit=101", "LIMIT_TOO_LARGE"},
		{"limit=100&page=101", "INVALID_PARAMETER"},
	} {
		resp, out := f.list(t, "", c.query)
		if resp.StatusCode != http.StatusBadRequest || grCode(out) != c.code {
			t.Errorf("%s: %d %s, want 400 %s", c.query, resp.StatusCode, grCode(out), c.code)
		}
	}

	f.cat.fail.Store(true)
	resp, out := f.list(t, "", "")
	if resp.StatusCode != http.StatusServiceUnavailable || grCode(out) != "SERVICE_UNAVAILABLE" {
		t.Errorf("catalog down: %d %v", resp.StatusCode, out)
	}
}

func grViewers(t *testing.T, f *grFix, session string) map[string]map[string]any {
	t.Helper()
	_, out := f.list(t, session, "limit=100&include_nsfw=true")
	viewers := map[string]map[string]any{}
	for _, it := range out["items"].([]any) {
		m := it.(map[string]any)
		v, _ := m["viewer"].(map[string]any)
		viewers[m["id"].(string)] = v
	}
	return viewers
}

func TestV1RatingsViewer(t *testing.T) {
	f := newGRFix(t)
	alice, onSFW, onSFW2 := strconv.Itoa(grRatingAlice), strconv.Itoa(grRatingRaters), strconv.Itoa(grRatingRaters+1)
	for id, v := range grViewers(t, f, "") {
		if v != nil {
			t.Fatalf("anonymous viewer on %s: %v", id, v)
		}
	}
	check := func(session, id string, edit, del bool) {
		t.Helper()
		v := grViewers(t, f, session)[id]
		if v == nil || v["can_edit"] != edit || v["can_delete"] != del || v["has_liked"] != false {
			t.Errorf("%s on %s: viewer %v, want can_edit=%t can_delete=%t", session, id, v, edit, del)
		}
	}
	check("gr-alice", alice, true, true)
	check("gr-alice", onSFW, false, false)
	check("gr-creator", onSFW, false, false)
	check("gr-creator", onSFW2, false, false)
	check("gr-staff", onSFW2, false, true)
	check("bearer:gr-staff-token", onSFW2, false, false)
}

func TestV1RatingsDetail(t *testing.T) {
	f := newGRFix(t)
	resp, out := f.onRating(t, http.MethodGet, "", grRatingAlice, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail: %d %v", resp.StatusCode, out)
	}
	if out["view_count"].(float64) != 3 {
		t.Errorf("view_count %v, want the seeded 2 plus this read", out["view_count"])
	}
	scores := out["aspect_scores"].(map[string]any)
	want := map[string]any{"art": 9.0, "story": nil, "music": 7.0, "character": nil, "route": nil, "system": nil, "voice": 10.0, "replay_value": 1.0}
	for k, v := range want {
		if scores[k] != v {
			t.Errorf("aspect_scores.%s = %v, want %v", k, scores[k], v)
		}
	}
	if w := out["work"].(map[string]any); w["id"] != strconv.Itoa(grWorkSFW) {
		t.Errorf("work %v", w)
	}
	if ws := out["work_summary"].(map[string]any); ws["id"] != strconv.Itoa(grWorkSFW) {
		t.Errorf("work_summary %v", ws)
	}
	f.onRating(t, http.MethodGet, "", grRatingAlice, "", nil)
	if n := f.count(t, `SELECT view FROM galgame_rating WHERE id = ?`, grRatingAlice); n != 4 {
		t.Errorf("stored view %d after two reads, want 4", n)
	}

	for _, id := range []int{grRatingBanned, grRatingHidden, grRatingMax} {
		resp, out := f.onRating(t, http.MethodGet, "", id, "", nil)
		if resp.StatusCode != http.StatusNotFound || grCode(out) != "NOT_FOUND" {
			t.Errorf("rating %d: %d %v, want 404", id, resp.StatusCode, out)
		}
	}

	resp, out = f.onRating(t, http.MethodGet, "", grRatingGone, "", nil)
	if resp.StatusCode != http.StatusOK || out["author"].(map[string]any)["id"] != strconv.Itoa(grGone) {
		t.Errorf("deleted author's rating: %d %v", resp.StatusCode, out)
	}
	nsfwRating := grRatingRaters + 2
	if resp, _ := f.onRating(t, http.MethodGet, "", nsfwRating, "", nil); resp.StatusCode != http.StatusOK {
		t.Errorf("adult work's rating: %d, want 200 whatever the preference", resp.StatusCode)
	}

	for i := range 60 {
		f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, created, updated) VALUES (?, ?, now() + (? * interval '1 second'), now())`,
			grRatingAlice, grLiker+i, i)
	}
	_, out = f.onRating(t, http.MethodGet, "", grRatingAlice, "", nil)
	var likers []string
	for _, l := range out["likers"].([]any) {
		likers = append(likers, l.(map[string]any)["id"].(string))
	}
	if len(likers) != 50 || likers[0] != strconv.Itoa(grLiker+59) || likers[49] != strconv.Itoa(grLiker+10) {
		t.Errorf("likers %v, want the 50 newest, newest first", likers)
	}
}

func TestV1RatingsCarryMakerAndTopTags(t *testing.T) {
	f := newGRFix(t)
	tagIDs := func(item map[string]any) []string {
		var out []string
		tags, _ := item["top_tags"].([]any)
		for _, tag := range tags {
			m, _ := tag.(map[string]any)
			out = append(out, strID(m["id"]))
		}
		return out
	}
	for _, tc := range []struct {
		nsfw bool
		want []string
	}{{false, []string{"21"}}, {true, []string{"21", "23"}}} {
		resp, body := f.list(t, "", fmt.Sprintf("work_id=%d&include_nsfw=%t", grWorkSFW, tc.nsfw))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("nsfw=%t: %d %+v", tc.nsfw, resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		if len(items) == 0 {
			t.Fatalf("nsfw=%t: no ratings for the work", tc.nsfw)
		}
		for _, it := range items {
			item, _ := it.(map[string]any)
			maker, _ := item["maker"].(map[string]any)
			if maker == nil || maker["display_name"] != "Maker Brand" || strID(maker["id"]) == "" {
				t.Errorf("maker = %+v, want the developer with its id", item["maker"])
			}
			if item["release_date"] != "2026-01-01" || item["release_date_precision"] != "day" {
				t.Errorf("release = %v %v, want 2026-01-01 day", item["release_date"], item["release_date_precision"])
			}
			if got := tagIDs(item); !slices.Equal(got, tc.want) {
				t.Errorf("nsfw=%t top_tags = %v, want %v: no spoiler tag, adult only with include_nsfw", tc.nsfw, got, tc.want)
			}
		}
	}
	resp, body := f.list(t, "", fmt.Sprintf("work_id=%d&include_nsfw=false", grWorkSFW2))
	items, _ := body["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) == 0 {
		t.Fatalf("second work: %d %+v", resp.StatusCode, body)
	}
	first, _ := items[0].(map[string]any)
	if first["top_tags"] == nil || len(tagIDs(first)) != 0 {
		t.Errorf("a work without tags: top_tags = %#v, want []", first["top_tags"])
	}
	if first["release_date"] != "2025-06-01" || first["release_date_precision"] != "month" {
		t.Errorf("a month-precise release = %v %v, want 2025-06-01 month", first["release_date"], first["release_date_precision"])
	}
}
