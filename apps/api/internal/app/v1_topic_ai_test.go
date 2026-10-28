package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/setting"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/aiclient"
)

// fakeModel answers chat completions with whatever the test last set.
type fakeModel struct {
	mu     sync.Mutex
	status int
	answer string
	calls  int
}

func (m *fakeModel) set(status int, answer string) {
	m.mu.Lock()
	m.status, m.answer = status, answer
	m.mu.Unlock()
}

func (m *fakeModel) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	status, answer := m.status, m.answer
	m.calls++
	m.mu.Unlock()
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": answer}, "finish_reason": "stop"}},
	})
}

func newTopicAIFix(t *testing.T, classify bool) (*writeFix, *fakeModel) {
	t.Helper()
	model := &fakeModel{status: http.StatusOK, answer: `{"section":"daily","reason":"r"}`}
	srv := httptest.NewServer(model)
	t.Cleanup(srv.Close)
	f := newWriteFixCommunity(t, nil, nil, func(a *App) {
		store := setting.NewStore(a.DB)
		if classify {
			if err := store.Set(map[string]any{setting.TopicAIClassifyEnabled: true}, w3UserAlice); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() { _ = a.DB.Exec(`DELETE FROM site_setting`).Error })
		a.Settings = store
		a.TopicAI = topicapiv1.NewTopicAI(aiclient.New(srv.URL, "test"), store, topicRepo.NewAIRepository(a.DB))
	})
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserGrant, "user", "admin")
	return f, model
}

func newTopicBody(title string, sections ...string) map[string]any {
	return map[string]any{
		"title":            title,
		"content_markdown": "body of " + title,
		"sections":         sections,
		"is_nsfw":          false,
		"access_scope":     "public",
	}
}

func (f *writeFix) classification(t *testing.T, id string) (section, source string) {
	t.Helper()
	if err := f.db.Raw(`SELECT section, source FROM topic_classification WHERE topic_id = ?`, id).Row().Scan(&section, &source); err != nil {
		t.Fatalf("classification of %s: %v", id, err)
	}
	return section, source
}

func TestV1TopicClassifyIsUnavailableWhileOff(t *testing.T) {
	f, model := newTopicAIFix(t, false)
	resp, body := f.callJSON(t, http.MethodPost, "/api/v1/topics/classify", "/topics/classify", "sess-alice", "",
		map[string]any{"title": "t", "content_markdown": "b"})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("status %d %v", resp.StatusCode, body)
	}
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9101), newTopicBody("off", "daily"))
	if resp.StatusCode != http.StatusCreated || out["section"] != "daily" {
		t.Fatalf("create while off: %d %v", resp.StatusCode, out)
	}
	if model.calls != 0 {
		t.Fatalf("model called %d times while classification is off", model.calls)
	}
	var n int64
	f.db.Raw(`SELECT COUNT(*) FROM topic_classification WHERE topic_id = ?`, strID(out["id"])).Scan(&n)
	if n != 0 {
		t.Fatalf("classification rows %d while off", n)
	}
}

func TestV1TopicModelDecidesTheSection(t *testing.T) {
	f, model := newTopicAIFix(t, true)

	model.set(http.StatusOK, `{"section":"tech-help","reason":"短求助"}`)
	resp, verdict := f.callJSON(t, http.MethodPost, "/api/v1/topics/classify", "/topics/classify", "sess-alice", "",
		map[string]any{"title": "打不开", "content_markdown": "body of 打不开"})
	if resp.StatusCode != http.StatusOK || verdict["section"] != "tech-help" || verdict["reason"] != "短求助" {
		t.Fatalf("classify %d %v", resp.StatusCode, verdict)
	}
	token, _ := verdict["classification_token"].(string)

	model.set(http.StatusOK, `{"section":"acgn","reason":"would differ"}`)
	body := newTopicBody("打不开", "daily")
	body["classification_token"] = token
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9102), body)
	if resp.StatusCode != http.StatusCreated || out["section"] != "tech-help" || out["category"] != "technique" {
		t.Fatalf("create with token: %d section %v category %v", resp.StatusCode, out["section"], out["category"])
	}
	if sec, src := f.classification(t, strID(out["id"])); sec != "tech-help" || src != "model" {
		t.Fatalf("classification %s %s", sec, src)
	}
	paid := false
	for _, a := range f.snapshotAwards() {
		if a.userID == w3UserAlice && a.delta == -10 {
			paid = true
		}
	}
	if !paid {
		t.Fatalf("no -10 award for a tech-help topic: %v", f.snapshotAwards())
	}

	body = newTopicBody("no token", "daily")
	model.set(http.StatusOK, `{"section":"development","reason":"开发"}`)
	resp, out = f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9103), body)
	if resp.StatusCode != http.StatusCreated || out["section"] != "development" {
		t.Fatalf("create without token: %d %v", resp.StatusCode, out["section"])
	}

	stale := newTopicBody("edited after classify", "daily")
	stale["classification_token"] = token
	model.set(http.StatusInternalServerError, "")
	resp, out = f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9104), stale)
	if resp.StatusCode != http.StatusCreated || out["section"] != "daily" {
		t.Fatalf("model down: %d %v", resp.StatusCode, out["section"])
	}
	fallbackID := strID(out["id"])
	if sec, src := f.classification(t, fallbackID); sec != "daily" || src != "fallback" {
		t.Fatalf("fallback classification %s %s", sec, src)
	}

	resp, prob := f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+fallbackID, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"sections": []string{"acgn"}})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("author moving the section: %d %v", resp.StatusCode, prob)
	}
	resp, moved := f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+fallbackID, "/topics/{topic_id}", "sess-staff", "",
		map[string]any{"sections": []string{"acgn"}})
	if resp.StatusCode != http.StatusOK || moved["section"] != "acgn" {
		t.Fatalf("staff moving the section: %d %v", resp.StatusCode, moved["section"])
	}
}

func TestV1TopicRecheckFilesARequestWhenTheModelDisagrees(t *testing.T) {
	f, model := newTopicAIFix(t, true)
	model.set(http.StatusInternalServerError, "")
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9111), newTopicBody("recheck", "daily"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", resp.StatusCode, out)
	}
	id := strID(out["id"])
	if err := f.db.Exec(`UPDATE topic_classification SET created = now() - interval '5 minutes' WHERE topic_id = ?`, id).Error; err != nil {
		t.Fatal(err)
	}
	model.set(http.StatusOK, `{"section":"game-running","reason":"运行问题"}`)
	f.App.TopicAI.Sweep()
	if sec, src := f.classification(t, id); sec != "game-running" || src != "model" {
		t.Fatalf("after recheck %s %s", sec, src)
	}
	var to string
	var requester *int
	if err := f.db.Raw(`SELECT to_section, requested_by FROM topic_section_request WHERE topic_id = ? AND state = 'pending'`, id).Row().Scan(&to, &requester); err != nil {
		t.Fatalf("no request filed: %v", err)
	}
	if to != "game-running" || requester != nil {
		t.Fatalf("request to %s by %v", to, requester)
	}
	resp, topic := f.callJSON(t, http.MethodGet, "/api/v1/topics/"+id, "/topics/{topic_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || topic["section"] != "daily" {
		t.Fatalf("recheck moved the topic by itself: %v", topic["section"])
	}
}

func TestV1TopicSectionRequests(t *testing.T) {
	f, _ := newTopicAIFix(t, false)
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9121), newTopicBody("ask", "daily"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", resp.StatusCode, out)
	}
	id := strID(out["id"])
	path := "/api/v1/topics/" + id + "/section-requests"
	spec := "/topics/{topic_id}/section-requests"

	for _, c := range []struct {
		session string
		body    map[string]any
		status  int
	}{
		{"sess-bob", map[string]any{"section": "acgn", "reason": "x"}, http.StatusForbidden},
		{"sess-alice", map[string]any{"section": "daily", "reason": "x"}, http.StatusUnprocessableEntity},
		{"sess-alice", map[string]any{"section": "resource-release", "reason": "x"}, http.StatusUnprocessableEntity},
		{"sess-alice", map[string]any{"section": "acgn", "reason": "   "}, http.StatusUnprocessableEntity},
		{"sess-alice", map[string]any{"section": "acgn", "reason": "这是动画话题"}, http.StatusCreated},
		{"sess-alice", map[string]any{"section": "site", "reason": "again"}, http.StatusConflict},
	} {
		resp, body := f.callJSON(t, http.MethodPost, path, spec, c.session, "", c.body)
		if resp.StatusCode != c.status {
			t.Fatalf("%s %v: %d %v", c.session, c.body, resp.StatusCode, body)
		}
	}

	resp, list := f.callJSON(t, http.MethodGet, path, spec, "sess-alice", "", nil)
	items, _ := list["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) != 1 {
		t.Fatalf("author's list %d %v", resp.StatusCode, list)
	}
	req, _ := items[0].(map[string]any)
	reqID := strID(req["id"])
	if req["state"] != "pending" || req["from_section"] != "daily" || req["to_section"] != "acgn" {
		t.Fatalf("request %v", req)
	}

	resp, _ = f.callJSON(t, http.MethodGet, "/api/v1/admin/section-requests", "/admin/section-requests", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("author reading the staff queue: %d", resp.StatusCode)
	}
	resp, queue := f.callJSON(t, http.MethodGet, "/api/v1/admin/section-requests", "/admin/section-requests", "sess-staff", "", nil)
	queued, _ := queue["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(queued) == 0 {
		t.Fatalf("staff queue %d %v", resp.StatusCode, queue)
	}

	decide := "/api/v1/admin/section-requests/" + reqID
	decideSpec := "/admin/section-requests/{section_request_id}"
	resp, decided := f.callJSON(t, http.MethodPatch, decide, decideSpec, "sess-staff", "", map[string]any{"state": "approved"})
	if resp.StatusCode != http.StatusOK || decided["state"] != "approved" || decided["decided_at"] == nil {
		t.Fatalf("approve %d %v", resp.StatusCode, decided)
	}
	resp, topic := f.callJSON(t, http.MethodGet, "/api/v1/topics/"+id, "/topics/{topic_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || topic["section"] != "acgn" || topic["category"] != "others" {
		t.Fatalf("after approval section %v category %v", topic["section"], topic["category"])
	}
	var notices int64
	f.db.Raw(`SELECT COUNT(*) FROM message WHERE receiver_id = ? AND type = 'notice'`, w3UserAlice).Scan(&notices)
	if notices == 0 {
		t.Fatal("the author was not notified")
	}
	resp, _ = f.callJSON(t, http.MethodPatch, decide, decideSpec, "sess-staff", "", map[string]any{"state": "rejected"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("deciding twice: %d", resp.StatusCode)
	}

	resp, second := f.callJSON(t, http.MethodPost, path, spec, "sess-alice", "", map[string]any{"section": "tech-help", "reason": "其实是求助"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second request %d %v", resp.StatusCode, second)
	}
	before := len(f.snapshotAwards())
	resp, _ = f.callJSON(t, http.MethodPatch, "/api/v1/admin/section-requests/"+strID(second["id"]), decideSpec, "sess-staff", "", map[string]any{"state": "approved"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve paid move %d", resp.StatusCode)
	}
	awards := f.snapshotAwards()[before:]
	if len(awards) != 1 || awards[0].userID != w3UserAlice || awards[0].delta != -13 {
		t.Fatalf("moving into a paid section should cost the reward plus the fee: %v", awards)
	}
}

func TestV1AdminSettings(t *testing.T) {
	f, _ := newTopicAIFix(t, false)
	resp, _ := f.callJSON(t, http.MethodGet, "/api/v1/admin/settings", "/admin/settings", "sess-staff", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("moderator reading settings: %d", resp.StatusCode)
	}
	resp, got := f.callJSON(t, http.MethodGet, "/api/v1/admin/settings", "/admin/settings", "sess-admin", "", nil)
	if resp.StatusCode != http.StatusOK || got["is_topic_classify_enabled"] != false || got["topic_summary_min_length"] != float64(500) {
		t.Fatalf("defaults %d %v", resp.StatusCode, got)
	}
	resp, got = f.callJSON(t, http.MethodPatch, "/api/v1/admin/settings", "/admin/settings", "sess-admin", "",
		map[string]any{"is_topic_classify_enabled": true, "topic_summary_min_length": 800})
	if resp.StatusCode != http.StatusOK || got["is_topic_classify_enabled"] != true || got["topic_summary_min_length"] != float64(800) {
		t.Fatalf("patch %d %v", resp.StatusCode, got)
	}
	resp, _ = f.callJSON(t, http.MethodPatch, "/api/v1/admin/settings", "/admin/settings", "sess-admin", "",
		map[string]any{"topic_ai_timeout_ms": 10})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("out of range timeout: %d", resp.StatusCode)
	}
}

func (f *writeFix) waitSummary(t *testing.T, id, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got *string
		if err := f.db.Raw(`SELECT ai_summary FROM topic WHERE id = ?`, id).Row().Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != nil && *got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("summary of %s is %v, want %q", id, got, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestV1TopicSummaryIsWrittenAndRewrittenAfterEdits(t *testing.T) {
	f, model := newTopicAIFix(t, false)
	if err := f.App.Settings.Set(map[string]any{
		setting.TopicAISummaryEnabled:           true,
		setting.TopicSummaryMinLength:           10,
		setting.TopicSummaryRegenerateThreshold: 20,
	}, w3UserAlice); err != nil {
		t.Fatal(err)
	}
	model.set(http.StatusOK, `{"summary":"第一版摘要"}`)
	body := newTopicBody("long", "daily")
	body["content_markdown"] = "这是一段足够长的正文, 用来生成摘要。\n第二段"
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9131), body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", resp.StatusCode, out)
	}
	id := strID(out["id"])
	f.waitSummary(t, id, "第一版摘要")
	resp, topic := f.callJSON(t, http.MethodGet, "/api/v1/topics/"+id, "/topics/{topic_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || topic["ai_summary"] != "第一版摘要" {
		t.Fatalf("detail ai_summary %v", topic["ai_summary"])
	}

	model.set(http.StatusOK, `{"summary":"第二版摘要"}`)
	resp, _ = f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+id, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"content_markdown": "这是一段足够长的正文, 用来生成摘要。\n第三段"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("small edit %d", resp.StatusCode)
	}
	var drift int
	f.db.Raw(`SELECT ai_summary_drift FROM topic WHERE id = ?`, id).Scan(&drift)
	if drift != 6 {
		t.Fatalf("drift after a three-rune rewrite is %d, want 6", drift)
	}
	time.Sleep(200 * time.Millisecond)
	f.waitSummary(t, id, "第一版摘要")

	resp, _ = f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+id, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"content_markdown": "这是一段足够长的正文, 用来生成摘要。\n第三段\n新加了一整段很长很长的内容, 足够超过阈值"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("large edit %d", resp.StatusCode)
	}
	f.waitSummary(t, id, "第二版摘要")
	f.db.Raw(`SELECT ai_summary_drift FROM topic WHERE id = ?`, id).Scan(&drift)
	if drift != 0 {
		t.Fatalf("drift after rewriting the summary is %d", drift)
	}
}

func TestV1TopicsCanLeaveOutHiddenSections(t *testing.T) {
	f, _ := newTopicAIFix(t, false)
	ids := map[string]string{}
	for i, sec := range []string{"tech-help", "daily"} {
		resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9141+i), newTopicBody("hidden "+sec, sec))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create %s: %d %v", sec, resp.StatusCode, out)
		}
		ids[sec] = strID(out["id"])
	}
	listed := func(query string) map[string]bool {
		resp, body := f.callJSON(t, http.MethodGet, "/api/v1/topics?sort=created_desc&limit=50"+query, "/topics", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list %s: %d %v", query, resp.StatusCode, body)
		}
		seen := map[string]bool{}
		items, _ := body["items"].([]any)
		for _, it := range items {
			m, _ := it.(map[string]any)
			seen[strID(m["id"])] = true
		}
		return seen
	}
	all := listed("")
	if !all[ids["tech-help"]] || !all[ids["daily"]] {
		t.Fatalf("default list misses a topic: %v", all)
	}
	shown := listed("&hidden_sections=exclude")
	if shown[ids["tech-help"]] || !shown[ids["daily"]] {
		t.Fatalf("exclude: tech-help %v daily %v", shown[ids["tech-help"]], shown[ids["daily"]])
	}
}

func TestV1TopicNSFWIsJudgedAndOnlyTightenedByTheAuthor(t *testing.T) {
	f, model := newTopicAIFix(t, true)

	model.set(http.StatusOK, `{"section":"daily","nsfw":true,"reason":"r"}`)
	resp, verdict := f.callJSON(t, http.MethodPost, "/api/v1/topics/classify", "/topics/classify", "sess-alice", "",
		map[string]any{"title": "adult", "content_markdown": "body of adult"})
	if resp.StatusCode != http.StatusOK || verdict["is_nsfw"] != true {
		t.Fatalf("classify %d %v", resp.StatusCode, verdict)
	}
	model.set(http.StatusOK, `{"section":"daily","nsfw":false,"reason":"would differ"}`)
	body := newTopicBody("adult", "daily")
	body["classification_token"] = verdict["classification_token"]
	resp, out := f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9131), body)
	if resp.StatusCode != http.StatusCreated || out["is_nsfw"] != true {
		t.Fatalf("the token's NSFW verdict was dropped: %d %v", resp.StatusCode, out["is_nsfw"])
	}
	adultID := strID(out["id"])

	model.set(http.StatusOK, `{"section":"daily","nsfw":false,"reason":"r"}`)
	marked := newTopicBody("author marks", "daily")
	marked["is_nsfw"] = true
	resp, out = f.callJSON(t, http.MethodPost, "/api/v1/topics", "/topics", "sess-alice", keyUUID(9132), marked)
	if resp.StatusCode != http.StatusCreated || out["is_nsfw"] != true {
		t.Fatalf("the author's own mark was dropped: %d %v", resp.StatusCode, out["is_nsfw"])
	}
	markedID := strID(out["id"])

	model.set(http.StatusOK, `{"section":"daily","nsfw":true,"reason":"r"}`)
	resp, out = f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+adultID, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"is_nsfw": false})
	if resp.StatusCode != http.StatusOK || out["is_nsfw"] != true {
		t.Fatalf("an author cleared a judged NSFW mark: %d %v", resp.StatusCode, out["is_nsfw"])
	}

	model.set(http.StatusInternalServerError, "")
	resp, out = f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+markedID, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"is_nsfw": false})
	if resp.StatusCode != http.StatusOK || out["is_nsfw"] != true {
		t.Fatalf("a mark was cleared while the model was down: %d %v", resp.StatusCode, out["is_nsfw"])
	}

	model.set(http.StatusOK, `{"section":"daily","nsfw":false,"reason":"r"}`)
	resp, out = f.callJSON(t, http.MethodPatch, "/api/v1/topics/"+markedID, "/topics/{topic_id}", "sess-alice", "",
		map[string]any{"is_nsfw": false})
	if resp.StatusCode != http.StatusOK || out["is_nsfw"] != false {
		t.Fatalf("an author could not clear their own mark: %d %v", resp.StatusCode, out["is_nsfw"])
	}
}
