package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var activityBlocks = []string{
	"topic", "topic_digest", "reply", "comment", "work", "work_digest", "work_stats", "work_revision",
	"galgame_rating", "resource", "quiz", "toolset", "todo", "update_log",
}

var allowedBlocks = map[string][]string{
	"topic_creation":            {"topic", "topic_digest"},
	"topic_upvote":              {"topic", "topic_digest"},
	"topic_reply_creation":      {"reply"},
	"best_answer_set":           {"reply"},
	"topic_comment_creation":    {"comment"},
	"galgame_creation":          {"work", "work_digest", "work_stats"},
	"galgame_resource_creation": {"work", "resource"},
	"galgame_edit":              {"work", "work_digest", "work_revision"},
	"galgame_rating_creation":   {"work", "galgame_rating"},
}

func (f *activityFix) expected(t *testing.T, extra string, args ...any) []string {
	t.Helper()
	var ids []int64
	q := `SELECT id FROM feed_activity
		WHERE (user_id BETWEEN 950000001 AND 950000999 OR source_id BETWEEN 950000001 AND 950000999 OR work_id BETWEEN 950000001 AND 950000999)
		AND type <> 'MESSAGE_UPVOTE' AND user_id <> ? AND work_id NOT IN (?, ?) ` + extra +
		` ORDER BY created DESC, type DESC, source_id DESC`
	if err := f.db.Raw(q, append([]any{acUserBanned, acWorkGone, acWorkAdult}, args...)...).Scan(&ids).Error; err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.FormatInt(id, 10)
	}
	return out
}

func TestV1ActivitiesWalk(t *testing.T) {
	f := newActivityFix(t)
	want := f.expected(t, `AND NOT is_nsfw AND NOT (type = 'TOPIC_CREATION' AND source_id = ?)`, acTopicHelp)
	if len(want) < 8 {
		t.Fatalf("seed too thin: %v", want)
	}
	var ties int
	if err := f.db.Raw(`SELECT COUNT(*) FROM feed_activity WHERE created = ? AND source_id BETWEEN 950000001 AND 950000999`, acTie).Scan(&ties).Error; err != nil || ties < 5 {
		t.Fatalf("the walk needs tied timestamps across pages, got %d", ties)
	}
	mine := f.ourRows(t)
	for _, limit := range []int{2, 3} {
		got := activityIDs(f.walk(t, url.Values{}, limit), mine)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
	solved := f.rowID(t, "MESSAGE_SOLUTION", acMsgSolved)
	if !strings.Contains(fmt.Sprint(want), solved) {
		t.Error("best_answer_set is part of the stream")
	}
}

func TestV1ActivitiesNoUpvoteEcho(t *testing.T) {
	f := newActivityFix(t)
	echo := f.rowID(t, "MESSAGE_UPVOTE", acMsgUpvote)
	for _, it := range f.walk(t, url.Values{"include_nsfw": {"true"}, "topic_sections": {"all"}}, 50) {
		if fmt.Sprint(it["id"]) == echo {
			t.Fatalf("the notification echo of an upvote reached the stream: %+v", it)
		}
	}
}

func TestV1ActivitiesFilters(t *testing.T) {
	f := newActivityFix(t)
	mine := f.ourRows(t)
	adultTopic := f.rowID(t, "TOPIC_CREATION", acTopicNSFW)
	helpTopic := f.rowID(t, "TOPIC_CREATION", acTopicHelp)
	normalTopic := f.rowID(t, "TOPIC_CREATION", acTopicNormal)

	plain := activityIDs(f.walk(t, url.Values{}, 50), mine)
	if strings.Contains(fmt.Sprint(plain), adultTopic) || strings.Contains(fmt.Sprint(plain), helpTopic) {
		t.Errorf("default request shows NSFW or help topics: %v", plain)
	}
	adult := activityIDs(f.walk(t, url.Values{"include_nsfw": {"true"}}, 50), mine)
	if !strings.Contains(fmt.Sprint(adult), adultTopic) {
		t.Errorf("include_nsfw=true misses the NSFW topic: %v", adult)
	}
	adultWork := 0
	for _, it := range f.walk(t, url.Values{"include_nsfw": {"true"}}, 50) {
		if w, _ := it["work"].(map[string]any); w != nil && fmt.Sprint(w["id"]) == fmt.Sprint(acWorkAdult) {
			adultWork++
		}
	}
	if adultWork == 0 {
		t.Error("include_nsfw=true misses the adult work")
	}

	help := activityIDs(f.walk(t, url.Values{"activity_types": {"topic_creation"}, "topic_sections": {"help"}}, 50), mine)
	if fmt.Sprint(help) != fmt.Sprint([]string{helpTopic}) {
		t.Errorf("help topics %v, want [%s]", help, helpTopic)
	}
	both := activityIDs(f.walk(t, url.Values{"activity_types": {"topic_creation"}, "topic_sections": {"all"}}, 50), mine)
	if len(both) != 2 || !strings.Contains(fmt.Sprint(both), normalTopic) || !strings.Contains(fmt.Sprint(both), helpTopic) {
		t.Errorf("all sections %v", both)
	}
	replies := f.walk(t, url.Values{"activity_types": {"topic_reply_creation,topic_comment_creation"}}, 50)
	for _, it := range replies {
		if at := it["activity_type"]; at != "topic_reply_creation" && at != "topic_comment_creation" {
			t.Errorf("activity_types filter let through %v", at)
		}
	}
}

func TestV1ActivitiesBumped(t *testing.T) {
	f := newActivityFix(t)
	if err := f.db.Exec(`UPDATE topic SET status_update_time = ? WHERE id BETWEEN ? AND ?`, acTie, acTopicMin, acTopicMax).Error; err != nil {
		t.Fatal(err)
	}
	mine := f.ourRows(t)
	got := activityIDs(f.walk(t, url.Values{"activity_types": {"topic_creation"}, "sort": {"bumped_desc"}, "topic_sections": {"all"}}, 1), mine)
	want := []string{f.rowID(t, "TOPIC_CREATION", acTopicHelp), f.rowID(t, "TOPIC_CREATION", acTopicNormal)}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("bumped walk %v, want %v", got, want)
	}
}

func TestV1ActivitiesRejects(t *testing.T) {
	f := newActivityFix(t)
	cases := []struct {
		name string
		q    url.Values
		code string
	}{
		{"bumped with mixed types", url.Values{"sort": {"bumped_desc"}, "activity_types": {"topic_creation,topic_upvote"}}, "INVALID_PARAMETER"},
		{"bumped with every type", url.Values{"sort": {"bumped_desc"}}, "INVALID_PARAMETER"},
		{"unknown type", url.Values{"activity_types": {"message_upvote"}}, "UNKNOWN_ENUM_VALUE"},
		{"unknown sort", url.Values{"sort": {"hot"}}, "UNKNOWN_SORT"},
		{"limit too large", url.Values{"limit": {"101"}}, "LIMIT_TOO_LARGE"},
		{"garbage cursor", url.Values{"cursor": {"cur_nope"}}, "INVALID_CURSOR"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := f.call(t, c.q)
			if resp.StatusCode != http.StatusBadRequest || body["code"] != c.code {
				t.Fatalf("%d %v, want 400 %s: %+v", resp.StatusCode, body["code"], c.code, body)
			}
		})
	}
	_, first := f.call(t, url.Values{"activity_types": {"topic_reply_creation"}, "limit": {"1"}})
	next, _ := first["next_cursor"].(string)
	if next == "" {
		t.Fatal("expected a second page")
	}
	resp, body := f.call(t, url.Values{"activity_types": {"topic_comment_creation"}, "limit": {"1"}, "cursor": {next}})
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Errorf("a cursor reused under other filters: %d %+v", resp.StatusCode, body)
	}
}

func TestV1ActivitiesShape(t *testing.T) {
	f := newActivityFix(t)
	items := f.walk(t, url.Values{}, 50)
	for _, it := range items {
		allowed := map[string]bool{}
		for _, b := range allowedBlocks[fmt.Sprint(it["activity_type"])] {
			allowed[b] = true
		}
		for _, b := range activityBlocks {
			if it[b] != nil && !allowed[b] {
				t.Errorf("%v %v carries %s", it["activity_type"], it["id"], b)
			}
		}
	}

	topic := activityByID(items, f.rowID(t, "TOPIC_CREATION", acTopicNormal))
	digest, _ := topic["topic_digest"].(map[string]any)
	top, _ := digest["top_reply"].(map[string]any)
	best, _ := digest["best_answer_excerpt"].(map[string]any)
	if top == nil || fmt.Sprint(top["reply_id"]) != fmt.Sprint(acReplyLiked) || best == nil || fmt.Sprint(best["reply_id"]) != fmt.Sprint(acReplyLiked) {
		t.Errorf("top_reply %+v best %+v: the hidden reply with 99 likes must not win", top, best)
	}
	up, _ := digest["latest_upvote"].(map[string]any)
	if up == nil || up["note"] != "push" {
		t.Errorf("latest_upvote %+v", up)
	}
	reactions, _ := digest["reactions"].([]any)
	if len(reactions) != 1 {
		t.Fatalf("reactions %+v", reactions)
	}
	heart, _ := reactions[0].(map[string]any)
	reactors, _ := heart["reactors"].([]any)
	if asInt(heart["count"]) != 2 || len(reactors) != 1 || heart["viewer"] != nil {
		t.Errorf("heart %+v: counted twice, the banned reactor left out, no viewer", heart)
	}
	if summary, _ := topic["topic"].(map[string]any); summary["object"] != "topic" || fmt.Sprint(summary["id"]) != fmt.Sprint(acTopicNormal) {
		t.Errorf("topic %+v", summary)
	}

	quote := activityByID(items, f.rowID(t, "TOPIC_REPLY_CREATION", acReplyQuote))
	reply, _ := quote["reply"].(map[string]any)
	quoted, _ := reply["quoted"].(map[string]any)
	if quoted == nil || asInt(quoted["floor"]) != 1 || asInt(reply["floor"]) != 2 {
		t.Errorf("reply %+v", reply)
	}
	raw := fmt.Sprint(reply["content"])
	if !strings.Contains(raw, "object:mention") || !strings.Contains(raw, strconv.Itoa(acUserBob)) {
		t.Errorf("the mention is not a node: %s", raw)
	}
	if strings.Contains(raw, "kungal-user:") {
		t.Errorf("token text leaked into the document: %s", raw)
	}

	solved := activityByID(items, f.rowID(t, "MESSAGE_SOLUTION", acMsgSolved))
	if r, _ := solved["reply"].(map[string]any); solved["activity_type"] != "best_answer_set" || fmt.Sprint(r["reply_id"]) != fmt.Sprint(acReplyLiked) {
		t.Errorf("best_answer_set %+v", solved)
	}
	comment := activityByID(items, f.rowID(t, "TOPIC_COMMENT_CREATION", acComment))
	if c, _ := comment["comment"].(map[string]any); c == nil || c["quoted"] == nil || comment["excerpt_markdown"] != "nice" {
		t.Errorf("comment %+v", comment)
	}

	work := activityByID(items, f.rowID(t, "GALGAME_CREATION", acWorkShown))
	if work == nil {
		t.Fatal("galgame_creation of the shown work is missing")
	}
	performer, _ := work["performer"].(map[string]any)
	wd, _ := work["work_digest"].(map[string]any)
	ws, _ := work["work_stats"].(map[string]any)
	ref, _ := work["work"].(map[string]any)
	if fmt.Sprint(performer["id"]) != fmt.Sprint(acUserAlice) || wd["release"] != "2024-05" || asInt(ws["favorite_count"]) != 5 ||
		ref["object"] != "work" || ref["display_name"] != fmt.Sprintf("Work %d", acWorkShown) {
		t.Errorf("galgame_creation %+v", work)
	}
	revisionOf := func(source int) any {
		it := activityByID(items, f.rowID(t, "GALGAME_EDIT", source))
		if it == nil {
			t.Fatalf("galgame_edit %d is missing", source)
		}
		return it["work_revision"]
	}
	if rv, _ := revisionOf(acEditEngine).(map[string]any); asInt(rv["revision_number"]) != 3 || rv["legacy_revision_id"] != nil {
		t.Errorf("engine edit revision %+v", rv)
	}
	if rv, _ := revisionOf(acEditWiki).(map[string]any); rv["revision_number"] != nil || rv["legacy_revision_id"] != "950000877" {
		t.Errorf("wiki edit revision %+v", rv)
	}
	if rv := revisionOf(acEditBare); rv != nil {
		t.Errorf("an edit with no revision recorded got %+v", rv)
	}
	for _, it := range items {
		if w, _ := it["work"].(map[string]any); w != nil && fmt.Sprint(w["id"]) == fmt.Sprint(acWorkGone) {
			t.Errorf("a work catalog does not know reached the stream: %+v", it)
		}
		if p, _ := it["performer"].(map[string]any); p != nil && fmt.Sprint(p["id"]) == fmt.Sprint(acUserBanned) {
			t.Errorf("a banned performer reached the stream: %+v", it)
		}
	}
}

func TestV1ActivitiesUpstream(t *testing.T) {
	f := newActivityFix(t)
	f.failOA.Store(true)
	resp, body := f.call(t, url.Values{"activity_types": {"topic_comment_creation"}})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("OAuth down, no document to convert: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.call(t, url.Values{"activity_types": {"topic_reply_creation"}})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("OAuth down: %d %+v", resp.StatusCode, body)
	}
	g := newActivityFix(t)
	g.catalog.fail.Store(true)
	resp, body = g.call(t, url.Values{"activity_types": {"galgame_resource_creation"}})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("catalog down: %d %+v", resp.StatusCode, body)
	}
}

func TestV1ActivitiesCacheKeepsStancesApart(t *testing.T) {
	f := newActivityFix(t)
	shows := func(items []map[string]any, work int) bool {
		for _, it := range items {
			if w, _ := it["work"].(map[string]any); w != nil && fmt.Sprint(w["id"]) == strconv.Itoa(work) {
				return true
			}
		}
		return false
	}
	adult := f.walk(t, url.Values{"include_nsfw": {"true"}}, 50)
	if !shows(adult, acWorkAdult) || !shows(adult, acWorkShown) {
		t.Fatal("the NSFW read misses a work")
	}
	plain := f.walk(t, url.Values{}, 50)
	if shows(plain, acWorkAdult) {
		t.Error("an adult work cached for an NSFW reader reached an SFW reader")
	}
	if !shows(plain, acWorkShown) {
		t.Error("the cache filled by an NSFW reader blanked an SFW work for an SFW reader")
	}
}

func TestV1ActivitiesRatingMatchesTheRatingShape(t *testing.T) {
	f := newActivityFix(t)
	ratings := map[string]map[string]any{}
	for _, it := range f.walk(t, url.Values{"activity_types": {"galgame_rating_creation"}}, 50) {
		if r, _ := it["galgame_rating"].(map[string]any); r != nil {
			ratings[fmt.Sprint(r["rating_id"])] = r
		}
	}
	if hold := ratings[strconv.Itoa(acRatingHold)]; hold == nil || hold["play_status"] != "on_hold" || hold["short_summary"] != "worth a second try" {
		t.Errorf("on_hold rating: %+v", hold)
	}
	if spoiler := ratings[strconv.Itoa(acRatingSpoiler)]; spoiler == nil || spoiler["spoiler_level"] != "serious" || spoiler["short_summary"] != "" {
		t.Errorf("a spoiler rating withholds its review as an empty string: %+v", spoiler)
	}
}

func TestV1ActivitiesResourceUsesTheResourceVocabulary(t *testing.T) {
	f := newActivityFix(t)
	items := f.walk(t, url.Values{"activity_types": {"galgame_resource_creation"}}, 50)
	it := activityByID(items, f.rowID(t, "GALGAME_RESOURCE_CREATION", acResShown))
	res, _ := it["resource"].(map[string]any)
	if res == nil || res["resource_type"] != "game" ||
		fmt.Sprint(res["resource_platforms"]) != "[win and]" || fmt.Sprint(res["resource_languages"]) != "[zh-cn ja-jp]" {
		t.Errorf("the resource block reads the jsonb axes in vocabulary order: %+v", res)
	}

	f.run(t, `UPDATE galgame_resource SET type = 'image' WHERE id = ?`, acResShown)
	resp, body := f.call(t, url.Values{"activity_types": {"galgame_resource_creation"}})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("a type outside the vocabulary is a data error, not a remap: %d %+v", resp.StatusCode, body)
	}
}
