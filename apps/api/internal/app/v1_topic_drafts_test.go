package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	w5cDraftBase = 930000600
	w5cDraftTieA = 930000611
	w5cDraftTieB = 930000612
	w5cDraftTieC = 930000613
	w5cDraftBob  = 930000620
)

const draftsPath = "/api/v1/me/topic-drafts"

func newDraftFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.seedDrafts(t)
	return f
}

// Every seeded draft of alice's, oldest first, so a test can name the order it
// expects without recomputing it.
func (f *writeFix) seedDrafts(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	run := func(id, userID int, title, content string, at time.Time) {
		t.Helper()
		if err := f.db.Exec(`INSERT INTO topic_draft
			(id, user_id, title, content, category, sections, cover_images, is_nsfw, created, updated)
			VALUES (?, ?, ?, ?, '', '', '', false, ?, ?)`,
			id, userID, title, content, at, at).Error; err != nil {
			t.Fatalf("seed drafts: %v", err)
		}
	}
	for i := 0; i < 4; i++ {
		run(w5cDraftBase+i, w3UserAlice,
			fmt.Sprintf("draft %d", i), fmt.Sprintf("body %d", i),
			base.Add(time.Duration(i)*time.Minute))
	}
	// Three drafts sharing one second. Without an id tie-breaker in the keyset
	// their order is whatever the plan happens to emit, and a page boundary
	// between them drops or repeats a row. Production has no ties on
	// (user_id, updated), so a seed shaped like production cannot catch it —
	// which is how the same bug survived three green pagination tests in W4.
	tie := base.Add(10 * time.Minute)
	for _, id := range []int{w5cDraftTieA, w5cDraftTieB, w5cDraftTieC} {
		run(id, w3UserAlice, "tie "+strconv.Itoa(id), "tied body", tie)
	}
	run(w5cDraftBob, w3UserBob, "bob's draft", "bob's body", base.Add(20*time.Minute))
}

func (f *writeFix) draftList(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodGet, draftsPath+query, session, "/me/topic-drafts", "", nil, nil)
	return resp, problemMap(t, body)
}

func (f *writeFix) draftIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		row, _ := item.(map[string]any)
		out = append(out, strID(row["id"]))
	}
	return out
}

func (f *writeFix) postDraft(t *testing.T, session, key string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodPost, draftsPath, session, "/me/topic-drafts", key, nil, payload)
	return resp, problemMap(t, body)
}

func (f *writeFix) draftRowCount(t *testing.T, query string, args ...any) int {
	t.Helper()
	return f.scalar(t, query, args...)
}

func TestV1CreateTopicDraftRoundTrips(t *testing.T) {
	f := newDraftFix(t)

	hash := strings.Repeat("ab", 32)
	resp, body := f.postDraft(t, "sess-alice", keyUUID(700), map[string]any{
		"title":              "  spaced title  ",
		"content_markdown":   "# hello\n\nworld",
		"category":           "galgame",
		"sections":           []string{"g-news", "g-other"},
		"is_nsfw":            true,
		"cover_image_hashes": []string{hash},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "topic_draft" {
		t.Errorf("object %v", body["object"])
	}
	if got := resp.Header.Get("Location"); got != draftsPath[len("/api/v1"):]+"/"+strID(body["id"]) &&
		got != "/api/v1/me/topic-drafts/"+strID(body["id"]) {
		t.Errorf("Location %q", got)
	}
	// The title is stored as sent; only the blank check trims.
	if body["title"] != "  spaced title  " {
		t.Errorf("title %q", body["title"])
	}
	if body["category"] != "galgame" {
		t.Errorf("category %v", body["category"])
	}

	id := strID(body["id"])
	resp, got := f.doJSONMap(t, http.MethodGet, draftsPath+"/"+id, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read back %d %+v", resp.StatusCode, got)
	}
	for _, field := range []string{"title", "content_markdown", "category", "is_nsfw"} {
		if fmt.Sprint(got[field]) != fmt.Sprint(body[field]) {
			t.Errorf("%s: created %v, read back %v", field, body[field], got[field])
		}
	}
	if fmt.Sprint(got["sections"]) != "[news]" {
		t.Errorf("sections %v", got["sections"])
	}
	if fmt.Sprint(got["cover_image_hashes"]) != "["+hash+"]" {
		t.Errorf("cover_image_hashes %v", got["cover_image_hashes"])
	}
}

func TestV1CreateTopicDraftOptionalFields(t *testing.T) {
	f := newDraftFix(t)

	resp, body := f.postDraft(t, "sess-alice", keyUUID(701), map[string]any{
		"content_markdown": "just a body",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	// Most production drafts have never had a category chosen. The stored empty
	// string is not a member of the enum, so it travels as null.
	if body["category"] != nil {
		t.Errorf("an unchosen category is null, got %v", body["category"])
	}
	if _, ok := body["category"]; !ok {
		t.Error("category must be present and null, not absent")
	}
	if fmt.Sprint(body["sections"]) != "[]" {
		t.Errorf("sections must be [], got %v", body["sections"])
	}
	if fmt.Sprint(body["cover_image_hashes"]) != "[]" {
		t.Errorf("cover_image_hashes must be [], got %v", body["cover_image_hashes"])
	}
	if body["title"] != "" {
		t.Errorf("an unwritten title is the empty string, got %v", body["title"])
	}
	for _, field := range []string{"polls", "lotteries"} {
		if fmt.Sprint(body[field]) != "[]" {
			t.Errorf("%s must be [], got %v", field, body[field])
		}
	}
}

func TestV1TopicDraftKeepsMiniAppsWithoutCodes(t *testing.T) {
	f := newDraftFix(t)

	image := strings.Repeat("cd", 32)
	poll := map[string]any{
		"title": "which route", "description": "", "choice_type": "multiple",
		"min_choice": 1, "max_choice": 2, "closes_at": "2026-12-31T15:59:59Z",
		"result_visibility": "after_vote", "is_anonymous": true, "can_change_vote": false,
		"options": []map[string]any{{"text": "a"}, {"text": "b"}, {"text": "c"}},
	}
	lottery := map[string]any{
		"title": "keys", "description": "two keys", "entry_mode": "signup",
		"floor_rule": nil, "draw_mode": "deadline", "draw_threshold": nil,
		"closes_at": "2026-12-31T15:59:59Z", "min_account_age_days": 3,
		"min_moemoepoint": 20, "is_entry_list_public": true,
		"prizes": []map[string]any{{
			"title": "steam key", "description": "", "image_hashes": []string{image},
			"adult_image_hashes": []string{}, "delivery": "code",
			"point_mode": nil, "point_amount": nil, "slot_count": 2,
			"codes": []string{"AAAA-BBBB", "CCCC-DDDD"},
		}},
	}
	resp, body := f.postDraft(t, "sess-alice", keyUUID(704), map[string]any{
		"content_markdown": "lottery soon",
		"polls":            []any{poll},
		"lotteries":        []any{lottery},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])

	resp, got := f.doJSONMap(t, http.MethodGet, draftsPath+"/"+id, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read back %d %+v", resp.StatusCode, got)
	}
	polls, _ := got["polls"].([]any)
	if len(polls) != 1 {
		t.Fatalf("polls %v", got["polls"])
	}
	readPoll, _ := polls[0].(map[string]any)
	for _, field := range []string{"title", "choice_type", "max_choice", "closes_at", "result_visibility", "is_anonymous", "can_change_vote"} {
		if fmt.Sprint(readPoll[field]) != fmt.Sprint(poll[field]) {
			t.Errorf("poll %s: sent %v, read back %v", field, poll[field], readPoll[field])
		}
	}
	if options, _ := readPoll["options"].([]any); len(options) != 3 {
		t.Errorf("poll options %v", readPoll["options"])
	}

	lotteries, _ := got["lotteries"].([]any)
	if len(lotteries) != 1 {
		t.Fatalf("lotteries %v", got["lotteries"])
	}
	readLottery, _ := lotteries[0].(map[string]any)
	if readLottery["floor_rule"] != nil || readLottery["closes_at"] != lottery["closes_at"] {
		t.Errorf("lottery read back %+v", readLottery)
	}
	prizes, _ := readLottery["prizes"].([]any)
	if len(prizes) != 1 {
		t.Fatalf("prizes %v", readLottery["prizes"])
	}
	prize, _ := prizes[0].(map[string]any)
	if _, ok := prize["codes"]; ok {
		t.Errorf("a draft must never hand codes back, got %v", prize["codes"])
	}
	if fmt.Sprint(prize["slot_count"]) != "2" || fmt.Sprint(prize["image_hashes"]) != "["+image+"]" {
		t.Errorf("prize read back %+v", prize)
	}

	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_draft WHERE id = ? AND lotteries::text LIKE '%AAAA-BBBB%'`, id); n != 0 {
		t.Fatal("a redemption code was stored in the draft")
	}
	// The reference ping reads prize_image_hashes, not the nested lotteries.
	if n := f.scalar(t, `SELECT COUNT(*) FROM topic_draft WHERE id = ? AND prize_image_hashes = ?::jsonb`, id, `["`+image+`"]`); n != 1 {
		t.Fatal("prize_image_hashes does not hold the drafted prize image")
	}
}

func TestV1CreateTopicDraftNeedsSomething(t *testing.T) {
	f := newDraftFix(t)

	before := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice)
	resp, body := f.postDraft(t, "sess-alice", keyUUID(702), map[string]any{
		"title": "   ", "content_markdown": "  \n\t ",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
		t.Fatalf("blank draft %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors %v", body["errors"])
	}
	e, _ := errs[0].(map[string]any)
	if e["pointer"] != "/content_markdown" || e["reason"] != "REQUIRED" {
		t.Fatalf("field error %+v", e)
	}
	if after := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice); after != before {
		t.Fatalf("a refused draft was persisted: %d -> %d", before, after)
	}
}

// K19: the limit is checked against the raw value, before anything trims.
func TestV1TopicDraftTitleLength(t *testing.T) {
	f := newDraftFix(t)

	resp, body := f.postDraft(t, "sess-alice", keyUUID(703), map[string]any{
		"title": strings.Repeat("x", 234), "content_markdown": "body",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("234 characters %d %+v", resp.StatusCode, body)
	}
	// K19 again, from the other side: 231 characters plus three spaces is 234
	// raw. Trimming first would bring it under the limit, so this is the case
	// that tells "checked on the raw value" apart from "checked after trimming".
	resp, body = f.postDraft(t, "sess-alice", keyUUID(704), map[string]any{
		"title": strings.Repeat("x", 231) + "   ", "content_markdown": "body",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("234 raw characters must be refused before trimming: %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors %v", body["errors"])
	}
	if e, _ := errs[0].(map[string]any); e["pointer"] != "/title" || e["reason"] != "TOO_LONG" {
		t.Fatalf("field error %+v", errs[0])
	}

	resp, body = f.postDraft(t, "sess-alice", keyUUID(705), map[string]any{
		"title": strings.Repeat("x", 233), "content_markdown": "body",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("exactly 233 characters is the limit, not over it: %d %+v", resp.StatusCode, body)
	}
}

func TestV1ListTopicDraftsOrderAndTraversal(t *testing.T) {
	f := newDraftFix(t)

	resp, body := f.draftList(t, "sess-alice", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	full := f.draftIDs(t, body)
	if len(full) != 7 {
		t.Fatalf("alice has 7 drafts, got %d: %v (bob's must not appear)", len(full), full)
	}
	if body["next_cursor"] != nil {
		t.Errorf("the last page carries no next_cursor, got %v", body["next_cursor"])
	}
	for _, id := range full {
		if id == strconv.Itoa(w5cDraftBob) {
			t.Fatalf("bob's draft leaked into alice's list: %v", full)
		}
	}
	// updated DESC, id DESC. The three tied drafts sort among themselves by id.
	want := []string{
		strconv.Itoa(w5cDraftTieC), strconv.Itoa(w5cDraftTieB), strconv.Itoa(w5cDraftTieA),
		strconv.Itoa(w5cDraftBase + 3), strconv.Itoa(w5cDraftBase + 2),
		strconv.Itoa(w5cDraftBase + 1), strconv.Itoa(w5cDraftBase),
	}
	if fmt.Sprint(full) != fmt.Sprint(want) {
		t.Fatalf("order %v\nwant %v", full, want)
	}

	// Walk the whole list two at a time, so a page boundary falls between two
	// drafts sharing a second.
	var walked []string
	cursor := ""
	for page := 0; page < 10; page++ {
		q := "?limit=2"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		resp, body := f.draftList(t, "sess-alice", q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		walked = append(walked, f.draftIDs(t, body)...)
		next, ok := body["next_cursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if fmt.Sprint(walked) != fmt.Sprint(full) {
		t.Fatalf("cursor traversal dropped or repeated a row\n got %v\nwant %v", walked, full)
	}
}

func TestV1ListTopicDraftsLimit(t *testing.T) {
	f := newDraftFix(t)

	resp, body := f.draftList(t, "sess-alice", "?limit=101")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit=101 %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicDraftCursorIsPerUser(t *testing.T) {
	f := newDraftFix(t)

	_, body := f.draftList(t, "sess-alice", "?limit=2")
	cursor, ok := body["next_cursor"].(string)
	if !ok {
		t.Fatalf("no cursor to reuse: %+v", body)
	}
	resp, body := f.draftList(t, "sess-bob", "?limit=2&cursor="+cursor)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Fatalf("alice's cursor must not open bob's list: %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicDraftIsPrivate(t *testing.T) {
	f := newDraftFix(t)

	path := draftsPath + "/" + strconv.Itoa(w5cDraftBob)
	resp, body := f.doJSONMap(t, http.MethodGet, path, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("reading bob's draft %d %+v", resp.StatusCode, body)
	}
	resp, body = f.doJSONMap(t, http.MethodDelete, path, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("deleting bob's draft %d %+v", resp.StatusCode, body)
	}
	if n := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE id = ?`, w5cDraftBob); n != 1 {
		t.Fatalf("bob's draft is gone: %d rows", n)
	}
}

func TestV1DeleteTopicDraft(t *testing.T) {
	f := newDraftFix(t)

	path := draftsPath + "/" + strconv.Itoa(w5cDraftBase)
	resp, _ := f.doJSONMap(t, http.MethodDelete, path, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE id = ?`, w5cDraftBase); n != 0 {
		t.Fatalf("still there: %d rows", n)
	}
	resp, body := f.doJSONMap(t, http.MethodDelete, path, "sess-alice", "/me/topic-drafts/{draft_id}")
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("deleting twice %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicDraftCap(t *testing.T) {
	f := newDraftFix(t)

	// Alice already holds 7; fill to exactly 30.
	for i := 0; i < 23; i++ {
		resp, body := f.postDraft(t, "sess-alice", keyUUID(800+i), map[string]any{
			"content_markdown": fmt.Sprintf("filler %d", i),
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("filler %d: %d %+v", i, resp.StatusCode, body)
		}
	}
	if n := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice); n != 30 {
		t.Fatalf("expected 30 drafts before the cap test, got %d", n)
	}

	resp, body := f.postDraft(t, "sess-alice", keyUUID(900), map[string]any{
		"content_markdown": "the thirty-first",
	})
	if resp.StatusCode != http.StatusConflict || body["code"] != "DRAFT_LIMIT_REACHED" {
		t.Fatalf("the 31st draft %d %+v", resp.StatusCode, body)
	}
	if asInt(body["limit"]) != 30 {
		t.Errorf("limit extension %v", body["limit"])
	}
	if n := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice); n != 30 {
		t.Fatalf("the refused draft was persisted: %d rows", n)
	}
	// The cap is the caller's own, not the table's.
	resp, body = f.postDraft(t, "sess-bob", keyUUID(901), map[string]any{
		"content_markdown": "bob is not capped",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("bob %d %+v", resp.StatusCode, body)
	}
}

func TestV1TopicDraftRequirements(t *testing.T) {
	f := newDraftFix(t)

	for _, c := range []struct {
		method, path, spec string
	}{
		{http.MethodGet, draftsPath, "/me/topic-drafts"},
		{http.MethodGet, draftsPath + "/" + strconv.Itoa(w5cDraftBase), "/me/topic-drafts/{draft_id}"},
		{http.MethodDelete, draftsPath + "/" + strconv.Itoa(w5cDraftBase), "/me/topic-drafts/{draft_id}"},
	} {
		resp, body := f.doJSONMap(t, c.method, c.path, "", c.spec)
		if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
			t.Fatalf("anonymous %s %s: %d %+v", c.method, c.path, resp.StatusCode, body)
		}
	}
	resp, body := f.postDraft(t, "", keyUUID(902), map[string]any{"content_markdown": "x"})
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Fatalf("anonymous POST %d %+v", resp.StatusCode, body)
	}

	resp, raw := f.doJSON(t, http.MethodPost, draftsPath, "sess-alice", "/me/topic-drafts", "", nil,
		map[string]any{"content_markdown": "x"})
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("missing Idempotency-Key %d %+v", resp.StatusCode, body)
	}
}

func TestV1CreateTopicDraftIdempotency(t *testing.T) {
	f := newDraftFix(t)

	before := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice)
	key := keyUUID(903)
	payload := map[string]any{"content_markdown": "replayed"}

	resp, first := f.postDraft(t, "sess-alice", key, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first %d %+v", resp.StatusCode, first)
	}
	resp, second := f.postDraft(t, "sess-alice", key, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("replay %d %+v", resp.StatusCode, second)
	}
	if strID(first["id"]) != strID(second["id"]) {
		t.Fatalf("a replay minted a second draft: %v vs %v", first["id"], second["id"])
	}
	if after := f.draftRowCount(t, `SELECT COUNT(*) FROM topic_draft WHERE user_id = ?`, w3UserAlice); after != before+1 {
		t.Fatalf("a replay wrote a row: %d -> %d", before, after)
	}
}

// The list carries a cut of the body, not the body.
func TestV1TopicDraftSummaryIsTruncated(t *testing.T) {
	f := newDraftFix(t)

	long := strings.Repeat("字", 200)
	resp, created := f.postDraft(t, "sess-alice", keyUUID(904), map[string]any{"content_markdown": long})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, created)
	}
	_, body := f.draftList(t, "sess-alice", "?limit=1")
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items %v", body["items"])
	}
	row, _ := items[0].(map[string]any)
	summary, _ := row["summary"].(string)
	if n := len([]rune(summary)); n != 120 {
		t.Fatalf("summary is %d characters, want 120 — LEFT() counts characters, not bytes", n)
	}
	if _, ok := row["content_markdown"]; ok {
		t.Errorf("the list item must not carry the whole body: %+v", row)
	}
}

func (f *writeFix) doJSONMap(t *testing.T, method, path, session, spec string) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, path, session, spec, "", nil, nil)
	if len(body) == 0 {
		return resp, map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("json: %v\n%s", err, body)
	}
	return resp, m
}
