package app

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const (
	t4TopicTieMin = 930000260
	t4TopicTieMax = 930000266
)

func newAdminTopicFix(t *testing.T) *writeFix {
	t.Helper()
	f := newLotteryFix(t)
	f.putSession(t, "sess-admin", w3UserStaff, "user", "admin")
	// Seven hidden topics that share one bump second, so a page boundary lands
	// inside the tie and only the id tie-breaker keeps the walk stable.
	tie := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	for id := t4TopicTieMin; id <= t4TopicTieMax; id++ {
		hiddenBy := "author"
		if id%2 == 0 {
			hiddenBy = "moderator"
		}
		if err := f.db.Exec(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, 'body', 0, 1, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			id, fmt.Sprintf("tied hidden %d", id), tie, tie, tie, w3UserBob, hiddenBy).Error; err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *writeFix) hiddenTopics(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.lotteryCall(t, http.MethodGet, "/api/v1/admin/hidden-topics?"+q.Encode(), "/admin/hidden-topics", session, "", nil)
}

func adminItemIDs(body map[string]any) []string {
	raw, _ := body["items"].([]any)
	out := make([]string, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		out = append(out, strID(m["id"]))
	}
	return out
}

func (f *writeFix) sqlIDs(t *testing.T, query string, args ...any) []string {
	t.Helper()
	rows, err := f.db.Raw(query, args...).Rows()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

func TestV1AdminHiddenTopicsWalk(t *testing.T) {
	f := newAdminTopicFix(t)
	want := f.sqlIDs(t, `SELECT id::text FROM topic WHERE status = 1 ORDER BY status_update_time DESC, id DESC`)
	if len(want) < 8 {
		t.Fatalf("seed too thin: %v", want)
	}
	for _, limit := range []int{2, 3} {
		var got []string
		for page := 1; page <= 20; page++ {
			resp, body := f.hiddenTopics(t, "sess-staff", url.Values{"page": {fmt.Sprint(page)}, "limit": {fmt.Sprint(limit)}})
			if resp.StatusCode != http.StatusOK || body["object"] != "list" {
				t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
			}
			if asInt(body["total"]) != len(want) || body["total_relation"] != "eq" {
				t.Fatalf("total %v %v, want %d eq", body["total"], body["total_relation"], len(want))
			}
			if _, has := body["next_cursor"]; has {
				t.Fatal("a page-number collection has no next_cursor")
			}
			ids := adminItemIDs(body)
			if len(ids) == 0 {
				break
			}
			got = append(got, ids...)
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1AdminHiddenTopicsFilter(t *testing.T) {
	f := newAdminTopicFix(t)
	resp, body := f.hiddenTopics(t, "sess-staff", url.Values{"hidden_by": {"moderator"}, "limit": {"100"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filter %d %+v", resp.StatusCode, body)
	}
	want := f.sqlIDs(t, `SELECT id::text FROM topic WHERE status = 1 AND hidden_by = 'moderator' ORDER BY status_update_time DESC, id DESC`)
	if fmt.Sprint(adminItemIDs(body)) != fmt.Sprint(want) || asInt(body["total"]) != len(want) {
		t.Errorf("moderator-hidden %v total %v, want %v", adminItemIDs(body), body["total"], want)
	}
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["state"] != "hidden" || m["hidden_by"] != "moderator" || m["object"] != "topic" {
			t.Errorf("item %+v", m)
		}
	}

	resp, body = f.hiddenTopics(t, "sess-staff", url.Values{"q": {"tied hidden 93000026"}, "limit": {"100"}})
	if resp.StatusCode != http.StatusOK || asInt(body["total"]) != 7 {
		t.Errorf("title search %d total %v", resp.StatusCode, body["total"])
	}
	resp, body = f.hiddenTopics(t, "sess-staff", url.Values{"q": {"100%_"}, "limit": {"100"}})
	if resp.StatusCode != http.StatusOK || asInt(body["total"]) != 0 {
		t.Errorf("wildcards are literal: %d total %v", resp.StatusCode, body["total"])
	}
	resp, body = f.hiddenTopics(t, "sess-staff", url.Values{"hidden_by": {"nobody"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown hidden_by %d %+v", resp.StatusCode, body)
	}
}

func TestV1AdminHiddenTopicsDepth(t *testing.T) {
	f := newAdminTopicFix(t)
	resp, body := f.hiddenTopics(t, "sess-staff", url.Values{"page": {"101"}, "limit": {"100"}})
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("too deep %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	e, _ := errs[0].(map[string]any)
	params, _ := e["params"].(map[string]any)
	if e["parameter"] != "page" || e["reason"] != "OUT_OF_RANGE" || asInt(params["maximum"]) != 100 {
		t.Errorf("depth error %+v", e)
	}
	if resp, _ = f.hiddenTopics(t, "sess-staff", url.Values{"page": {"100"}, "limit": {"100"}}); resp.StatusCode != http.StatusOK {
		t.Errorf("exactly at the cap %d", resp.StatusCode)
	}
}

func TestV1AdminTopicFacesNeedTheirPermissions(t *testing.T) {
	f := newAdminTopicFix(t)
	topic := fmt.Sprint(w3TopicHidden)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	for _, c := range []struct {
		name, method, url, spec, session string
		hdr                              http.Header
		status                           int
	}{
		{"list as a user", http.MethodGet, "/api/v1/admin/hidden-topics", "/admin/hidden-topics", "sess-alice", nil, http.StatusForbidden},
		{"list anonymously", http.MethodGet, "/api/v1/admin/hidden-topics", "/admin/hidden-topics", "", nil, http.StatusUnauthorized},
		{"list as a Bearer moderator", http.MethodGet, "/api/v1/admin/hidden-topics", "/admin/hidden-topics", "", bearer, http.StatusForbidden},
		{"view as a user", http.MethodGet, "/api/v1/admin/topics/" + topic, "/admin/topics/{topic_id}", "sess-alice", nil, http.StatusForbidden},
		{"view as a moderator, who may list but not purge", http.MethodGet, "/api/v1/admin/topics/" + topic, "/admin/topics/{topic_id}", "sess-staff", nil, http.StatusForbidden},
		{"purge as a moderator", http.MethodDelete, "/api/v1/admin/topics/" + topic, "/admin/topics/{topic_id}", "sess-staff", nil, http.StatusForbidden},
		{"purge as a user", http.MethodDelete, "/api/v1/admin/topics/" + topic, "/admin/topics/{topic_id}", "sess-alice", nil, http.StatusForbidden},
		{"purge as a Bearer moderator", http.MethodDelete, "/api/v1/admin/topics/" + topic, "/admin/topics/{topic_id}", "", bearer, http.StatusForbidden},
	} {
		resp, body := f.doJSON(t, c.method, c.url, c.session, c.spec, "", c.hdr, nil)
		if resp.StatusCode != c.status {
			t.Errorf("%s: %d %s", c.name, resp.StatusCode, body)
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic WHERE id = ?`, w3TopicHidden); n != 1 {
		t.Fatal("a refused purge deleted the topic")
	}
}

func TestV1AdminPurgeRefundsOpenLotteryEscrow(t *testing.T) {
	f := newAdminTopicFix(t)
	lottery := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", pointPrize("fixed", 10, 2)))
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 980 {
		t.Fatalf("escrow not taken: %d", got)
	}

	resp, preview := f.lotteryCall(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/topics/%d", w3TopicPub),
		"/admin/topics/{topic_id}", "sess-admin", "", nil)
	if resp.StatusCode != http.StatusOK || preview["object"] != "admin_topic" ||
		asInt(preview["lottery_count"]) != 1 || asInt(preview["open_lottery_escrow"]) != 20 {
		t.Fatalf("preview %d %+v", resp.StatusCode, preview)
	}

	resp, body := f.lotteryCall(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/topics/%d", w3TopicPub),
		"/admin/topics/{topic_id}", "sess-admin", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("purge %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic WHERE id = ?`, w3TopicPub); n != 0 {
		t.Error("the topic survived its purge")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_lottery WHERE id = ?`, lottery); n != 0 {
		t.Error("the lottery survived its topic's purge")
	}
	if got := f.scalar(t, `SELECT moemoepoint FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); got != 1000 {
		t.Errorf("the author's escrow did not come back: %d", got)
	}
	var refund *awardCall
	for _, a := range f.lotteryAwards(lottery) {
		if a.delta > 0 {
			a := a
			refund = &a
		}
	}
	if refund == nil || refund.delta != 20 || refund.userID != w3UserAlice || refund.key != "kungal:lottery_escrow_refund:topic_lottery_"+lottery {
		t.Errorf("refund %+v", refund)
	}

	resp, _ = f.lotteryCall(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/topics/%d", w3TopicPub),
		"/admin/topics/{topic_id}", "sess-admin", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("purging twice %d", resp.StatusCode)
	}
}

func TestV1AdminPurgeWaitsForADraw(t *testing.T) {
	f := newAdminTopicFix(t)
	lottery := f.mustCreateLottery(t, w3TopicPub, "sess-alice", lotteryBody("signup", "manual", offlinePrize(1)))
	if err := f.db.Exec(`UPDATE topic_lottery SET status = 'drawing' WHERE id = ?`, lottery).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.lotteryCall(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/topics/%d", w3TopicPub),
		"/admin/topics/{topic_id}", "sess-admin", "", nil)
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_DRAWN" {
		t.Fatalf("purge during a draw %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic WHERE id = ?`, w3TopicPub); n != 1 {
		t.Error("the topic was purged while its lottery was being drawn")
	}
}
