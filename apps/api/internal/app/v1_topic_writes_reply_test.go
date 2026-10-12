package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
)

func replyBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return out
}

func (f *writeFix) createReply(t *testing.T, topicID int, session, idem, content string) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/topics/%d/replies", topicID), session,
		"/topics/{topic_id}/replies", idem, nil, map[string]any{"content_markdown": content})
	if resp.StatusCode != http.StatusCreated {
		return resp, problemMap(t, raw)
	}
	return resp, replyBody(t, raw)
}

func (f *writeFix) floorOf(t *testing.T, replyID int) int {
	t.Helper()
	var floor int
	if err := f.db.Raw(`SELECT floor FROM topic_reply WHERE id = ?`, replyID).Row().Scan(&floor); err != nil {
		t.Fatal(err)
	}
	return floor
}

func TestV1CreateReplyValuesAndSideEffects(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)

	resp, out := f.createReply(t, w3TopicFloors, "sess-bob", "11111111-1111-4111-8111-111111111111", "hello @nobody")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	if got := resp.Header.Get("Location"); got != "/api/v1/replies/"+fmt.Sprint(asInt(out["id"])) {
		t.Fatalf("location %q", got)
	}
	if out["floor"] != float64(4) {
		t.Fatalf("floor %v, want 4 after the seeded three", out["floor"])
	}
	if out["topic_id"] != fmt.Sprint(w3TopicFloors) {
		t.Fatalf("topic_id %v", out["topic_id"])
	}
	author, _ := out["author"].(map[string]any)
	if author["id"] != fmt.Sprint(w3UserBob) {
		t.Fatalf("author %v", out["author"])
	}
	viewer, _ := out["viewer"].(map[string]any)
	if viewer == nil || viewer["has_liked"] != false {
		t.Fatalf("viewer %v", out["viewer"])
	}

	var replyCount, lastFloor int
	if err := f.db.Raw(`SELECT reply_count, last_reply_floor FROM topic WHERE id = ?`, w3TopicFloors).
		Row().Scan(&replyCount, &lastFloor); err != nil {
		t.Fatal(err)
	}
	if replyCount != 4 || lastFloor != 4 {
		t.Fatalf("reply_count %d last_reply_floor %d", replyCount, lastFloor)
	}

	var msgType, link string
	var receiver int
	if err := f.db.Raw(`SELECT receiver_id, type, link FROM message WHERE sender_id = ? AND type = 'replied' ORDER BY id DESC LIMIT 1`,
		w3UserBob).Row().Scan(&receiver, &msgType, &link); err != nil {
		t.Fatal(err)
	}
	if receiver != w3UserAlice || link != fmt.Sprintf("/topic/%d?reply=4", w3TopicFloors) {
		t.Fatalf("replied message to %d link %q", receiver, link)
	}
	wantNotifRefs(t, f.db, []string{notifRef("replied", "reply", asInt(out["id"]), "topic", w3TopicFloors)},
		"sender_id = ? AND type = 'replied'", w3UserBob)

	aw := f.snapshotAwards()
	replyID := asInt(out["id"])
	if len(aw) != 1 || aw[0].userID != w3UserAlice || aw[0].delta != constants.RewardReply ||
		aw[0].reason != moemoepoint.ReasonContentApproved ||
		aw[0].ref != moemoepoint.Ref("topic_reply", replyID) ||
		aw[0].key != moemoepoint.Key("replied", fmt.Sprintf("topic_reply_%d", replyID)) {
		t.Fatalf("awards %+v", aw)
	}
}

func TestV1CreateReplyByTopicAuthorEarnsNothing(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	resp, out := f.createReply(t, w3TopicFloors, "sess-alice", "11111111-1111-4111-8111-111111111112", "mine")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	if aw := f.snapshotAwards(); len(aw) != 0 {
		t.Fatalf("awards %+v", aw)
	}
	var n int64
	f.db.Raw(`SELECT COUNT(*) FROM message WHERE type = 'replied' AND sender_id = ?`, w3UserAlice).Scan(&n)
	if n != 0 {
		t.Fatalf("replied messages %d", n)
	}
}

func TestV1CreateReplyBumpsWithinCutoffOnly(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	var before string
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicOld).Row().Scan(&before); err != nil {
		t.Fatal(err)
	}
	if resp, out := f.createReply(t, w3TopicOld, "sess-bob", "11111111-1111-4111-8111-111111111113", "old topic"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	var after string
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicOld).Row().Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("a topic older than the cutoff was bumped: %s -> %s", before, after)
	}

	var pubBefore string
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&pubBefore); err != nil {
		t.Fatal(err)
	}
	if resp, out := f.createReply(t, w3TopicPub, "sess-bob", "11111111-1111-4111-8111-111111111114", "fresh topic"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	var pubAfter string
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&pubAfter); err != nil {
		t.Fatal(err)
	}
	if pubAfter == pubBefore {
		t.Fatal("a topic within the cutoff was not bumped")
	}
}

func TestV1ReplyFloorsAreNeverReused(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	_, out := f.createReply(t, w3TopicFloors, "sess-bob", "11111111-1111-4111-8111-111111111115", "four")
	first := asInt(out["id"])
	if got := f.floorOf(t, first); got != 4 {
		t.Fatalf("first floor %d", got)
	}
	resp, raw := f.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/replies/%d", first), "sess-bob",
		"/replies/{reply_id}", "", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %s", resp.StatusCode, raw)
	}
	_, out = f.createReply(t, w3TopicFloors, "sess-bob", "11111111-1111-4111-8111-111111111116", "five")
	if got := f.floorOf(t, asInt(out["id"])); got != 5 {
		t.Fatalf("floor after deleting the top reply is %d, want 5 (floors are never reused)", got)
	}
}

func TestV1ReplyFloorsSurviveConcurrentCreates(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	const n = 4
	var wg sync.WaitGroup
	ids := make([]int, n)
	codes := make([]int, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, out := f.createReply(t, w3TopicFloors, "sess-bob",
				fmt.Sprintf("11111111-1111-4111-8111-1111111112%02d", i), fmt.Sprintf("concurrent %d", i))
			codes[i] = resp.StatusCode
			if resp.StatusCode == http.StatusCreated {
				ids[i] = asInt(out["id"])
			}
		}(i)
	}
	wg.Wait()
	seen := map[int]bool{}
	for i := range n {
		if codes[i] != http.StatusCreated {
			t.Fatalf("create %d answered %d", i, codes[i])
		}
		floor := f.floorOf(t, ids[i])
		if seen[floor] {
			t.Fatalf("floor %d assigned twice", floor)
		}
		seen[floor] = true
	}
	if len(seen) != n {
		t.Fatalf("floors %v", seen)
	}
}

func TestV1UpdateReplyPermissionsAndEdits(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	reply := w3ReplyMin // author bob

	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-other",
		"/replies/{reply_id}", "", nil, map[string]any{"content_markdown": "nope"})
	if resp.StatusCode != http.StatusForbidden || problemMap(t, raw)["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("stranger edit %d %s", resp.StatusCode, raw)
	}

	hdr := http.Header{"Authorization": []string{"Bearer staff-token"}}
	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", reply), "",
		"/replies/{reply_id}", "", hdr, map[string]any{"content_markdown": "bearer staff"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bearer staff edit %d %s: a Bearer request never holds staff powers", resp.StatusCode, raw)
	}

	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-bob",
		"/replies/{reply_id}", "", nil, map[string]any{"content_markdown": "   "})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("blank edit %d %s", resp.StatusCode, raw)
	}

	var bumpBefore string
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicFloors).Row().Scan(&bumpBefore); err != nil {
		t.Fatal(err)
	}
	resp, raw = f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-staff",
		"/replies/{reply_id}", "", nil, map[string]any{"content_markdown": "edited by staff"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff edit %d %s", resp.StatusCode, raw)
	}
	var content, bumpAfter string
	var edited *string
	if err := f.db.Raw(`SELECT content, edited FROM topic_reply WHERE id = ?`, reply).Row().Scan(&content, &edited); err != nil {
		t.Fatal(err)
	}
	if content != "edited by staff" || edited == nil {
		t.Fatalf("content %q edited %v", content, edited)
	}
	if err := f.db.Raw(`SELECT status_update_time FROM topic WHERE id = ?`, w3TopicFloors).Row().Scan(&bumpAfter); err != nil {
		t.Fatal(err)
	}
	if bumpAfter != bumpBefore {
		t.Fatal("editing a reply bumped its topic")
	}
}

func TestV1DeleteReplyPenaltyAndPointers(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	reply := w3ReplyMin // bob's reply, like_count 2
	if err := f.db.Exec(`INSERT INTO topic_comment (id, content, user_id, target_user_id, topic_id, topic_reply_id, status, created, updated)
		VALUES (?, 'visible', ?, ?, ?, ?, 0, now(), now()), (?, 'hidden', ?, ?, ?, ?, 1, now(), now())`,
		w3CommMin, w3UserAlice, w3UserBob, w3TopicFloors, reply,
		w3CommMin+1, w3UserAlice, w3UserBob, w3TopicFloors, reply).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`UPDATE topic SET best_answer_id = ?, pinned_reply_id = ? WHERE id = ?`,
		reply, reply, w3TopicFloors).Error; err != nil {
		t.Fatal(err)
	}

	// 3 * (2 likes + 1 visible comment + 1) = 12, and bob holds 5.
	resp, raw := f.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-bob",
		"/replies/{reply_id}", "", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("poor author delete %d %s", resp.StatusCode, raw)
	}
	p := problemMap(t, raw)
	if p["code"] != "MOEMOEPOINT_INSUFFICIENT" || p["required"] != float64(12) {
		t.Fatalf("problem %+v", p)
	}
	if aw := f.snapshotAwards(); len(aw) != 0 {
		t.Fatalf("a refused delete awarded %+v", aw)
	}

	// Staff delete charges the author 3 and is never blocked by their balance.
	resp, raw = f.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-staff",
		"/replies/{reply_id}", "", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("staff delete %d %s", resp.StatusCode, raw)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].userID != w3UserBob || aw[0].delta != -3 ||
		aw[0].reason != moemoepoint.ReasonContentRemoved ||
		aw[0].key != moemoepoint.Key("reply_deleted", fmt.Sprintf("topic_reply_%d", reply)) {
		t.Fatalf("awards %+v", aw)
	}

	var rows, comments, replyCount int64
	var best, pinned *int
	f.db.Raw(`SELECT COUNT(*) FROM topic_reply WHERE id = ?`, reply).Scan(&rows)
	f.db.Raw(`SELECT COUNT(*) FROM topic_comment WHERE topic_reply_id = ?`, reply).Scan(&comments)
	if err := f.db.Raw(`SELECT best_answer_id, pinned_reply_id, reply_count FROM topic WHERE id = ?`, w3TopicFloors).
		Row().Scan(&best, &pinned, &replyCount); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || comments != 0 {
		t.Fatalf("reply rows %d comments %d", rows, comments)
	}
	if best != nil || pinned != nil {
		t.Fatalf("best %v pinned %v must be cleared by the delete", best, pinned)
	}
	if replyCount != 2 {
		t.Fatalf("reply_count %d", replyCount)
	}
}

func TestV1DeleteReplyByItsAuthorPaysTheFormula(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	reply := w3ReplyMin + 1 // alice's reply, no likes, no comments
	resp, raw := f.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/replies/%d", reply), "sess-alice",
		"/replies/{reply_id}", "", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("author delete %d %s", resp.StatusCode, raw)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].userID != w3UserAlice || aw[0].delta != -3 {
		t.Fatalf("awards %+v", aw)
	}
}

func TestV1ReplyWritesVisibility(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	if err := f.db.Exec(`UPDATE topic_reply SET status = 1 WHERE id = ?`, w3ReplyMin+2).Error; err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		method  string
		url     string
		spec    string
		session string
		payload any
		want    int
	}{
		{"hidden reply edit", http.MethodPatch, fmt.Sprintf("/api/v1/replies/%d", w3ReplyMin+2), "/replies/{reply_id}", "sess-bob", map[string]any{"content_markdown": "x"}, 404},
		{"hidden reply delete", http.MethodDelete, fmt.Sprintf("/api/v1/replies/%d", w3ReplyMin+2), "/replies/{reply_id}", "sess-bob", nil, 404},
		{"hidden reply source", http.MethodGet, fmt.Sprintf("/api/v1/replies/%d/source", w3ReplyMin+2), "/replies/{reply_id}/source", "sess-bob", nil, 404},
		{"stranger on a hidden topic", http.MethodPost, fmt.Sprintf("/api/v1/topics/%d/replies", w3TopicHidden), "/topics/{topic_id}/replies", "sess-other", map[string]any{"content_markdown": "x"}, 404},
		{"missing topic", http.MethodPost, "/api/v1/topics/930000999/replies", "/topics/{topic_id}/replies", "sess-bob", map[string]any{"content_markdown": "x"}, 404},
	}
	for _, c := range cases {
		idem := ""
		if c.method == http.MethodPost {
			idem = "22222222-2222-4222-8222-222222222222"
		}
		resp, raw := f.doJSON(t, c.method, c.url, c.session, c.spec, idem, nil, c.payload)
		if resp.StatusCode != c.want {
			t.Fatalf("%s: %d %s", c.name, resp.StatusCode, raw)
		}
	}

	// The author may still reply to their own hidden topic, as the legacy API allowed.
	resp, raw := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/topics/%d/replies", w3TopicHidden), "sess-alice",
		"/topics/{topic_id}/replies", "33333333-3333-4333-8333-333333333333", nil,
		map[string]any{"content_markdown": "own hidden topic"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("author reply to own hidden topic %d %s", resp.StatusCode, raw)
	}
}

func TestV1ReplySourceAndTopicSource(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)

	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/replies/%d/source", w3ReplyMin), "sess-bob",
		"/replies/{reply_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reply source %d %s", resp.StatusCode, raw)
	}
	src := replyBody(t, raw)
	if src["object"] != "reply_source" || src["content_markdown"] != "r1" ||
		src["reply_id"] != fmt.Sprint(w3ReplyMin) || src["topic_id"] != fmt.Sprint(w3TopicFloors) {
		t.Fatalf("reply source %+v", src)
	}

	resp, raw = f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/replies/%d/source", w3ReplyMin), "sess-other",
		"/replies/{reply_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger reply source %d %s", resp.StatusCode, raw)
	}

	resp, raw = f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/source", w3TopicRole), "sess-alice",
		"/topics/{topic_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("topic source %d %s", resp.StatusCode, raw)
	}
	ts := replyBody(t, raw)
	if ts["object"] != "topic_source" || ts["content_markdown"] != "role-body" || ts["access_scope"] != "role" {
		t.Fatalf("topic source %+v", ts)
	}
	grants, _ := ts["access_grants"].(map[string]any)
	roles, _ := grants["roles"].([]any)
	users, _ := grants["users"].([]any)
	if len(roles) != 1 || roles[0] != "creator" || len(users) != 0 {
		t.Fatalf("grants %+v", grants)
	}

	resp, raw = f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/source", w3TopicPub), "sess-other",
		"/topics/{topic_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger topic source %d %s", resp.StatusCode, raw)
	}

	resp, raw = f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/source", w3TopicHidden), "sess-other",
		"/topics/{topic_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger source of a hidden topic %d %s", resp.StatusCode, raw)
	}
}

func TestV1TopicSourceListsGrantedUsers(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	if err := f.db.Exec(`UPDATE topic SET access_scope = 'users' WHERE id = ?`, w3TopicPub).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`INSERT INTO topic_access_grant (topic_id, subject_type, subject_value)
		VALUES (?, 'user', ?), (?, 'user', ?)`,
		w3TopicPub, fmt.Sprint(w3UserGrant), w3TopicPub, fmt.Sprint(w3UserBanned)).Error; err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/source", w3TopicPub), "sess-alice",
		"/topics/{topic_id}/source", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("source %d %s", resp.StatusCode, raw)
	}
	grants, _ := replyBody(t, raw)["access_grants"].(map[string]any)
	users, _ := grants["users"].([]any)
	if len(users) != 2 {
		t.Fatalf("users %+v", users)
	}
	// Grants have no stored order, so the source lists them by id.
	first, _ := users[0].(map[string]any)
	second, _ := users[1].(map[string]any)
	if first["id"] != fmt.Sprint(w3UserBanned) || first["name"] != nil {
		t.Fatalf("a banned grantee keeps its entry with name null: %+v", first)
	}
	if second["id"] != fmt.Sprint(w3UserGrant) || second["name"] != "grantee" {
		t.Fatalf("second grantee %+v", second)
	}
}

func TestV1PatchTopicKeepsGrantsOfAnAuthorOnlyUsersScope(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	// A users-scoped topic granted to nobody but its author stores no grant row.
	if err := f.db.Exec(`UPDATE topic SET access_scope = 'users' WHERE id = ?`, w3TopicPub).Error; err != nil {
		t.Fatal(err)
	}
	resp, raw := f.doJSON(t, http.MethodPatch, fmt.Sprintf("/api/v1/topics/%d", w3TopicPub), "sess-alice",
		"/topics/{topic_id}", "", nil, map[string]any{"title": "renamed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch of an author-only users topic %d %s", resp.StatusCode, raw)
	}
	var scope, title string
	if err := f.db.Raw(`SELECT access_scope, title FROM topic WHERE id = ?`, w3TopicPub).Row().Scan(&scope, &title); err != nil {
		t.Fatal(err)
	}
	if scope != "users" || title != "renamed" {
		t.Fatalf("scope %q title %q", scope, title)
	}
}

func TestV1ReplyFloorSkipsAheadOfRowsTheCounterMissed(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	// Between the migration and the deploy, code that does not bump the counter
	// can insert a reply; the next floor must clear that row, not collide with it.
	if err := f.db.Exec(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'stale writer', 9, ?, ?, 0, now(), now())`,
		w3ReplyMin+50, w3UserBob, w3TopicFloors).Error; err != nil {
		t.Fatal(err)
	}
	_, out := f.createReply(t, w3TopicFloors, "sess-bob", "44444444-4444-4444-8444-444444444444", "after the stale row")
	if got := f.floorOf(t, asInt(out["id"])); got != 10 {
		t.Fatalf("floor %d, want 10: the counter must catch up to the highest stored floor", got)
	}
}

func TestV1ReplyAwardsWaitForTheCommit(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	// A reply to another user's topic writes the "replied" message, queues the
	// award, and only then writes the mention message. Failing that second
	// insert rolls the reply back with an award already queued.
	const cb = "w3_fail_second_message_insert"
	var inserts atomic.Int32
	if err := f.db.Callback().Create().Before("gorm:create").Register(cb, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "message" {
			return
		}
		if inserts.Add(1) >= 2 {
			_ = tx.AddError(errors.New("message insert refused by the test"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.db.Callback().Create().Remove(cb) })

	resp, out := f.createReply(t, w3TopicFloors, "sess-bob", "55555555-5555-4555-8555-555555555555",
		mentionBody(w3MentionMin))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	if aw := f.snapshotAwards(); len(aw) != 0 {
		t.Fatalf("a rolled-back reply awarded %+v", aw)
	}
	var replies, count int64
	f.db.Raw(`SELECT COUNT(*) FROM topic_reply WHERE topic_id = ?`, w3TopicFloors).Scan(&replies)
	f.db.Raw(`SELECT reply_count FROM topic WHERE id = ?`, w3TopicFloors).Scan(&count)
	if replies != 3 || count != 3 {
		t.Fatalf("rows %d reply_count %d", replies, count)
	}
}
