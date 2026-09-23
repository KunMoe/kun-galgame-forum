package app

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const (
	secAnime   = 933000001
	secLinux   = 933000002
	secTopicLo = 933000101
)

type secTopic struct {
	id       int
	user     int
	status   int
	access   string
	category string
	nsfw     bool
	created  time.Time
	view     int
}

// o-anime holds topics of every kind the counts must tell apart; t-linux stays
// empty. Two published topics share one created second across page
// boundaries so only the id tie-breaker keeps the walk stable.
func newSectionFix(t *testing.T) (*writeFix, []secTopic) {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	base := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	tie := base.Add(3 * time.Hour)
	topics := []secTopic{
		{secTopicLo + 0, w3UserAlice, 0, "public", "others", false, base, 10},
		{secTopicLo + 1, w3UserBob, 0, "public", "others", false, tie, 20},
		{secTopicLo + 2, w3UserAlice, 0, "public", "others", false, tie, 30},
		{secTopicLo + 3, w3UserBob, 0, "public", "others", true, base.Add(4 * time.Hour), 40},
		{secTopicLo + 4, w3UserAlice, 1, "public", "others", false, base.Add(5 * time.Hour), 50},
		{secTopicLo + 5, w3UserAlice, 0, "login", "others", false, base.Add(6 * time.Hour), 60},
		{secTopicLo + 6, w3UserAlice, 0, "public", "galgame", false, base.Add(7 * time.Hour), 70},
		{secTopicLo + 7, w3UserBanned, 0, "public", "others", false, base.Add(8 * time.Hour), 80},
		{secTopicLo + 8, w3UserBob, 0, "public", "others", false, tie, 90},
	}
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	clean := func() {
		_ = f.db.Exec(`DELETE FROM topic WHERE id BETWEEN ? AND ?`, secTopicLo, secTopicLo+99).Error
		_ = f.db.Exec(`DELETE FROM topic_section WHERE id IN (?, ?)`, secAnime, secLinux).Error
	}
	clean()
	t.Cleanup(clean)
	run(`INSERT INTO topic_section (id, name, created, updated) VALUES (?, 'o-anime', now(), now()), (?, 't-linux', now(), now())`, secAnime, secLinux)
	for _, tp := range topics {
		hiddenBy := ""
		if tp.status == 1 {
			hiddenBy = "author"
		}
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, 'body', ?, ?, ?, ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			tp.id, fmt.Sprintf("anime %d", tp.id), tp.view, tp.status, tp.category, tp.created, tp.created, tp.created,
			tp.user, tp.nsfw, tp.access, hiddenBy)
		run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, now(), now())`, tp.id, secAnime)
	}
	return f, topics
}

func (f *writeFix) sectionList(t *testing.T, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodGet, "/api/v1/sections?"+q.Encode(), "/sections", "", "", nil, nil)
}

func sectionByName(t *testing.T, body map[string]any, name string) map[string]any {
	t.Helper()
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["section"] == name {
			return m
		}
	}
	return nil
}

func TestV1SectionsCountOnlyPublishedPublicTopicsOfTheirCategory(t *testing.T) {
	f, topics := newSectionFix(t)
	resp, body := f.sectionList(t, url.Values{"category": {"others"}})
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("sections %d %+v", resp.StatusCode, body)
	}
	if _, has := body["next_cursor"]; has {
		t.Error("the section list is not paged")
	}
	anime := sectionByName(t, body, "o-anime")
	var wantCount, wantViews int
	for _, tp := range topics {
		if tp.status == 0 && tp.access == "public" && tp.category == "others" {
			wantCount++
			wantViews += tp.view
		}
	}
	if anime == nil || asInt(anime["topic_count"]) != wantCount || asInt(anime["view_count"]) != wantViews ||
		anime["category"] != "others" || anime["object"] != "section" {
		t.Fatalf("o-anime %+v, want count %d views %d", anime, wantCount, wantViews)
	}
	latest, _ := anime["latest_topic"].(map[string]any)
	if strID(latest["id"]) != fmt.Sprint(secTopicLo+3) || latest["object"] != "topic" {
		t.Errorf("latest %+v: the banned author's newer topic is skipped, the NSFW one counts", latest)
	}
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["category"] != "others" {
			t.Errorf("category filter let through %+v", m)
		}
	}
	if sectionByName(t, body, "t-linux") != nil {
		t.Error("t-linux is not an others section")
	}
}

func TestV1SectionsKeepEmptySections(t *testing.T) {
	f, _ := newSectionFix(t)
	_, body := f.sectionList(t, url.Values{})
	linux := sectionByName(t, body, "t-linux")
	if linux == nil || asInt(linux["topic_count"]) != 0 || linux["latest_topic"] != nil {
		t.Errorf("empty section %+v", linux)
	}
	if sectionByName(t, body, "o-anime") == nil {
		t.Error("no category means every section")
	}
}

func TestV1SectionsNeedAccounts(t *testing.T) {
	f, _ := newSectionFix(t)
	f.failOA.Store(true)
	if resp, body := f.sectionList(t, url.Values{"category": {"others"}}); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("accounts down: %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicsSectionFilterWalk(t *testing.T) {
	f, topics := newSectionFix(t)
	var want []string
	for _, id := range f.sqlIDs(t, `SELECT t.id::text FROM topic t
		JOIN topic_section_relation tsr ON tsr.topic_id = t.id
		WHERE tsr.topic_section_id = ? AND t.status != 1 AND t.access_scope = 'public'
		ORDER BY t.created DESC, t.id DESC`, secAnime) {
		if id != fmt.Sprint(secTopicLo+7) {
			want = append(want, id)
		}
	}
	if len(want) < 5 {
		t.Fatalf("seed too thin %v (%d topics)", want, len(topics))
	}
	for _, limit := range []int{1, 2, 3} {
		q := url.Values{"section": {"o-anime"}, "sort": {"created_desc"}, "include_nsfw": {"true"}, "limit": {fmt.Sprint(limit)}}
		var got []string
		for range 30 {
			resp, body := f.docCall(t, http.MethodGet, "/api/v1/topics?"+q.Encode(), "/topics", "", "", nil, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("topics %d %+v", resp.StatusCode, body)
			}
			got = append(got, adminItemIDs(body)...)
			next, _ := body["next_cursor"].(string)
			if next == "" {
				break
			}
			q.Set("cursor", next)
		}
		if joinIDs(got) != joinIDs(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}

	_, body := f.docCall(t, http.MethodGet, "/api/v1/topics?section=o-anime&sort=created_desc&limit=1", "/topics", "", "", nil, nil)
	cur, _ := body["next_cursor"].(string)
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/topics?section=o-daily&sort=created_desc&limit=1&cursor="+cur, "/topics", "", "", nil, nil)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Errorf("cursor reused on another section: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.docCall(t, http.MethodGet, "/api/v1/topics?section=o-nothing", "/topics", "", "", nil, nil)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
		t.Errorf("unknown section %d %+v", resp.StatusCode, body)
	}
	_, body = f.docCall(t, http.MethodGet, "/api/v1/topics?section=o-anime&limit=100", "/topics", "", "", nil, nil)
	for _, id := range adminItemIDs(body) {
		if id == fmt.Sprint(secTopicLo+3) {
			t.Error("include_nsfw defaults to false inside a section too")
		}
	}
}
