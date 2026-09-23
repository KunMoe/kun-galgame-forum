package app

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestV1TopicRankingVisibility(t *testing.T) {
	f := newRankingFix(t)
	resp, body := f.get(t, "/rankings/topics", url.Values{"limit": {"5"}})
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if _, has := body["next_cursor"]; has {
		t.Error("a top-N list has no next_cursor")
	}
	got := rkEntries(body, "topic")
	if fmt.Sprint(rkIDs(got)) != rkWant(rkTopicTieHigh, rkTopicTieMid, rkTopicTieLow) {
		t.Fatalf("topics %v: hidden, login-only, NSFW, banned-author and missing-author topics must be dropped, ties broken by id DESC", rkIDs(got))
	}
	for i, e := range got {
		if e.rank != i+1 || e.value != 1900000100 || e.raw["object"] != "topic_ranking_entry" {
			t.Errorf("entry %d: %+v", i, e.raw)
		}
	}
	topic, _ := got[0].raw["topic"].(map[string]any)
	author, _ := topic["author"].(map[string]any)
	if topic["object"] != "topic" || topic["title"] != fmt.Sprintf("topic %d", rkTopicTieHigh) || author["id"] != fmt.Sprint(rkUserC) {
		t.Errorf("topic %+v", topic)
	}

	resp, body = f.get(t, "/rankings/topics", url.Values{"limit": {"10"}, "include_nsfw": {"true"}})
	if ids := rkIDs(rkEntries(body, "topic")); resp.StatusCode != http.StatusOK || fmt.Sprint(ids) != rkWant(rkTopicNSFW, rkTopicTieHigh, rkTopicTieMid, rkTopicTieLow) {
		t.Errorf("include_nsfw %d %v", resp.StatusCode, ids)
	}
}

func TestV1TopicRankingTieWalk(t *testing.T) {
	f := newRankingFix(t)
	var want []string
	rows, err := f.db.Raw(`SELECT id::text FROM topic WHERE status != 1 AND access_scope = 'public' AND is_nsfw = false
		AND user_id NOT IN (?, ?) ORDER BY view DESC, id DESC`, rkUserBanned, rkUserMissing).Rows()
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		want = append(want, id)
	}
	_ = rows.Close()
	_, body := f.get(t, "/rankings/topics", url.Values{"limit": {"100"}})
	if got := rkIDs(rkEntries(body, "topic")); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order %v, want %v", got, want)
	}
}

func TestV1UserRanking(t *testing.T) {
	f := newRankingFix(t)
	resp, body := f.get(t, "/rankings/users", url.Values{"limit": {"10"}})
	got := rkEntries(body, "member")
	if resp.StatusCode != http.StatusOK || fmt.Sprint(rkIDs(got)) != rkWant(rkUserA, rkUserB, rkUserC) {
		t.Fatalf("moemoepoint %d %v: banned and deleted accounts are dropped", resp.StatusCode, rkIDs(got))
	}
	if got[0].rank != 1 || got[0].value != 2000000003 || got[0].raw["bio"] != fmt.Sprintf("bio of %d", rkUserA) || got[0].raw["object"] != "user_ranking_entry" {
		t.Errorf("first %+v", got[0].raw)
	}

	_, body = f.get(t, "/rankings/users", url.Values{"sort": {"replies_desc"}})
	got = rkEntries(body, "member")
	counts := map[string]float64{}
	for _, e := range got {
		counts[e.id] = e.value
	}
	if fmt.Sprint(rkIDs(got)) != rkWant(rkUserA, rkUserB, rkUserC) ||
		counts[fmt.Sprint(rkUserA)] != 3 || counts[fmt.Sprint(rkUserB)] != 2 || counts[fmt.Sprint(rkUserC)] != 1 {
		t.Errorf("replies %v %v: hidden replies and replies in hidden topics do not count", rkIDs(got), counts)
	}

	for _, sort := range []string{"topics_desc", "comments_desc", "resources_desc"} {
		if resp, body := f.get(t, "/rankings/users", url.Values{"sort": {sort}}); resp.StatusCode != http.StatusOK {
			t.Errorf("%s: %d %+v", sort, resp.StatusCode, body)
		}
	}
	_, body = f.get(t, "/rankings/users", url.Values{"sort": {"resources_desc"}})
	if got := rkEntries(body, "member"); len(got) == 0 || got[0].id != fmt.Sprint(rkUserA) || got[0].value != 5 {
		t.Errorf("resources %+v", got)
	}
}

func TestV1WorkRanking(t *testing.T) {
	f := newRankingFix(t)
	resp, body := f.get(t, "/rankings/works", url.Values{"limit": {"10"}})
	got := rkEntries(body, "work")
	if resp.StatusCode != http.StatusOK || fmt.Sprint(rkIDs(got)) != rkWant(rkWorkTop, rkWorkNoCreator, rkWorkBannedMaker) {
		t.Fatalf("works %d %v: unpublished, resourceless and NSFW works are dropped", resp.StatusCode, rkIDs(got))
	}
	first := got[0].raw
	work, _ := first["work"].(map[string]any)
	creator, _ := first["creator"].(map[string]any)
	if first["object"] != "work_ranking_entry" || got[0].rank != 1 || got[0].value != 1900000050 ||
		work["object"] != "work" || work["display_name"] != fmt.Sprintf("作品%d", rkWorkTop) || work["is_nsfw"] != false ||
		creator["id"] != fmt.Sprint(rkUserA) {
		t.Errorf("first %+v", first)
	}
	if got[1].raw["creator"] != nil || got[2].raw["creator"] != nil {
		t.Errorf("no creator and a banned creator are both null: %+v / %+v", got[1].raw["creator"], got[2].raw["creator"])
	}

	_, body = f.get(t, "/rankings/works", url.Values{"limit": {"10"}, "include_resourceless": {"true"}, "include_nsfw": {"true"}})
	if ids := rkIDs(rkEntries(body, "work")); fmt.Sprint(ids) != rkWant(rkWorkBare, rkWorkNSFW, rkWorkTop, rkWorkNoCreator, rkWorkBannedMaker) {
		t.Errorf("with resourceless and NSFW %v", ids)
	}
}

func TestV1WorkRankingFillsTheLimitWhenNSFWLeads(t *testing.T) {
	f := newRankingFix(t)
	_, body := f.get(t, "/rankings/works", url.Values{"limit": {"2"}})
	if ids := rkIDs(rkEntries(body, "work")); fmt.Sprint(ids) != rkWant(rkWorkTop, rkWorkNoCreator) {
		t.Errorf("an SFW top 2 must hold 2 works even though an NSFW work outranks them: %v", ids)
	}
}

func TestV1WorkRankingRatingTie(t *testing.T) {
	f := newRankingFix(t)
	_, body := f.get(t, "/rankings/works", url.Values{"sort": {"rating_desc"}})
	got := rkEntries(body, "work")
	if fmt.Sprint(rkIDs(got)) != rkWant(rkWorkNoCreator, rkWorkTop) || got[0].value != got[1].value {
		t.Errorf("rating tie %v: equal weighted ratings break on id DESC", got)
	}

	// Five more works on the same weighted rating, rated out of id order, so the
	// join's natural output order is not id DESC by accident.
	now := time.Now().Add(-time.Hour)
	extra := []int{rkWorkBannedMaker + 3, rkWorkBannedMaker + 1, rkWorkBannedMaker + 4, rkWorkBannedMaker + 2, rkWorkBannedMaker + 5}
	for _, id := range extra {
		f.sql(t, `INSERT INTO galgame (id, view, updated, published, content_limit, resource_count) VALUES (?, 0, ?, true, 'sfw', 0)`, id, now)
		f.sql(t, `INSERT INTO galgame_rating (recommend, overall, user_id, work_id, updated) VALUES ('yes', 10, ?, ?, ?)`, rkUserA, id, now)
	}
	_, body = f.get(t, "/rankings/works", url.Values{"sort": {"rating_desc"}, "include_resourceless": {"true"}})
	want := rkWant(rkWorkBannedMaker+5, rkWorkBannedMaker+4, rkWorkBannedMaker+3, rkWorkBannedMaker+2, rkWorkBannedMaker+1, rkWorkNoCreator, rkWorkTop)
	if ids := rkIDs(rkEntries(body, "work")); fmt.Sprint(ids) != want {
		t.Errorf("seven tied works %v, want %s", ids, want)
	}
}

func TestV1RankingUpstreamDown(t *testing.T) {
	f := newRankingFix(t)
	f.failOA.Store(true)
	for _, path := range []string{"/rankings/topics", "/rankings/users", "/rankings/works"} {
		resp, body := f.get(t, path, nil)
		if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
			t.Errorf("%s with OAuth down: %d %+v", path, resp.StatusCode, body)
		}
	}
	g := newRankingFix(t)
	g.works.fail.Store(true)
	if resp, body := g.get(t, "/rankings/works", nil); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("catalog down: %d %+v", resp.StatusCode, body)
	}
}

func TestV1RankingRejects(t *testing.T) {
	f := newRankingFix(t)
	for _, c := range []struct {
		path string
		q    url.Values
		code string
	}{
		{"/rankings/topics", url.Values{"sort": {"views_asc"}}, "UNKNOWN_SORT"},
		{"/rankings/users", url.Values{"sort": {"reply_created"}}, "UNKNOWN_SORT"},
		{"/rankings/works", url.Values{"limit": {"101"}}, "LIMIT_TOO_LARGE"},
		{"/rankings/topics", url.Values{"limit": {"0"}}, "INVALID_PARAMETER"},
		{"/rankings/works", url.Values{"include_nsfw": {"yes"}}, "INVALID_PARAMETER"},
	} {
		resp, body := f.get(t, c.path, c.q)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != c.code {
			t.Errorf("%s %v: %d %+v, want 400 %s", c.path, c.q, resp.StatusCode, body, c.code)
		}
	}
}
