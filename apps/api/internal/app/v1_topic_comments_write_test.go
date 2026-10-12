package app

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestV1CreateCommentValuesAndSideEffects(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(1), map[string]any{"text": "hello"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	id := asInt(out["id"])
	if got := resp.Header.Get("Location"); got != "/api/v1/comments/"+strconv.Itoa(id) {
		t.Fatalf("location %q", got)
	}
	if out["object"] != "comment" || out["reply_id"] != strconv.Itoa(w3ReplyMin) {
		t.Fatalf("object/reply_id %+v", out)
	}
	if out["parent_comment_id"] != nil {
		t.Fatalf("parent_comment_id %v, want null", out["parent_comment_id"])
	}
	author, _ := out["author"].(map[string]any)
	if author["id"] != strconv.Itoa(w3UserAlice) {
		t.Fatalf("author %v", out["author"])
	}
	// The reply is bob's, so a top-level comment answers bob. The caller never
	// sent this; it is derived.
	target, _ := out["in_reply_to_user"].(map[string]any)
	if target["id"] != strconv.Itoa(w3UserBob) {
		t.Fatalf("in_reply_to_user %v, want the reply author %d", out["in_reply_to_user"], w3UserBob)
	}
	if docPlainText(t, commentContent(t, out)) != "hello" {
		t.Fatalf("content %v", out["content"])
	}
	if asInt(out["like_count"]) != 0 || out["edited_at"] != nil {
		t.Fatalf("like_count/edited_at %+v", out)
	}
	v := viewerOf(t, out)
	if v["has_liked"] != false || v["can_edit"] != true || v["can_delete"] != true || v["can_like"] != false {
		t.Fatalf("viewer %v", v)
	}

	var storedTarget, storedTopic int
	var updated any
	if err := f.db.Raw(`SELECT target_user_id, topic_id, updated FROM topic_comment WHERE id = ?`, id).
		Row().Scan(&storedTarget, &storedTopic, &updated); err != nil {
		t.Fatal(err)
	}
	if storedTarget != w3UserBob || storedTopic != w3TopicFloors {
		t.Fatalf("stored target %d topic %d", storedTarget, storedTopic)
	}
	if updated == nil {
		t.Fatal("updated is NOT NULL with no default; the insert must set it")
	}

	if n := f.commentRowCount(t, `SELECT comment_count FROM topic WHERE id = ?`, w3TopicFloors); n != 6 {
		t.Fatalf("topic comment_count %d, want 6 (5 seeded visible + 1 new)", n)
	}

	var receiver int
	var link string
	if err := f.db.Raw(`SELECT receiver_id, link FROM message WHERE sender_id = ? AND type = 'commented' ORDER BY id DESC LIMIT 1`,
		w3UserAlice).Row().Scan(&receiver, &link); err != nil {
		t.Fatal(err)
	}
	if receiver != w3UserBob || link != fmt.Sprintf("/topic/%d?comment=%d", w3TopicFloors, id) {
		t.Fatalf("commented message to %d link %q", receiver, link)
	}
	wantNotifRefs(t, f.db, []string{notifRef("commented", "topic_comment", id, "topic", w3TopicFloors)},
		"sender_id = ? AND type = 'commented'", w3UserAlice)

	got := awardsFor(f.snapshotAwards(), "topic_comment:"+strconv.Itoa(id))
	want := awardCall{w3UserBob, 1, "content_approved", "topic_comment:" + strconv.Itoa(id),
		"kungal:commented:topic_comment_" + strconv.Itoa(id)}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("awards %+v, want %+v", got, want)
	}
}

func TestV1CreateCommentDerivesTargetFromTheParent(t *testing.T) {
	f := newCommentFix(t, nil)

	// w5CommentBob is bob's comment under bob's own reply: a child of it
	// answers bob, not the reply author and not the caller.
	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(2), map[string]any{
		"text":              "child",
		"parent_comment_id": strconv.Itoa(w5CommentBob),
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	if out["parent_comment_id"] != strconv.Itoa(w5CommentBob) {
		t.Fatalf("parent_comment_id %v", out["parent_comment_id"])
	}
	target, _ := out["in_reply_to_user"].(map[string]any)
	if target["id"] != strconv.Itoa(w3UserBob) {
		t.Fatalf("in_reply_to_user %v, want the parent author %d", out["in_reply_to_user"], w3UserBob)
	}

	// A child of alice's own comment answers alice, so nothing is awarded and
	// nobody is notified.
	before := len(f.snapshotAwards())
	resp, out = f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(3), map[string]any{
		"text":              "self",
		"parent_comment_id": strconv.Itoa(w5CommentAlice),
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	target, _ = out["in_reply_to_user"].(map[string]any)
	if target["id"] != strconv.Itoa(w3UserAlice) {
		t.Fatalf("in_reply_to_user %v", out["in_reply_to_user"])
	}
	if got := f.snapshotAwards(); len(got) != before {
		t.Fatalf("a comment answering yourself awarded %+v", got[before:])
	}
}

func TestV1CreateCommentRejectsAForeignParent(t *testing.T) {
	f := newCommentFix(t, nil)

	cases := []struct{ name, parent string }{
		{"other reply", strconv.Itoa(w5CommentOnR2)},
		{"hidden", strconv.Itoa(w5CommentHidden)},
		{"nonexistent", "930000499"},
		{"not positive", "0"},
	}
	for i, c := range cases {
		name, parent := c.name, c.parent
		resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(10+i), map[string]any{
			"text": "x", "parent_comment_id": parent,
		})
		if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
			t.Fatalf("%s: %d %+v", name, resp.StatusCode, out)
		}
		errs, _ := out["errors"].([]any)
		if len(errs) != 1 {
			t.Fatalf("%s: errors %v", name, out["errors"])
		}
		e, _ := errs[0].(map[string]any)
		if e["pointer"] != "/parent_comment_id" {
			t.Fatalf("%s: errors[0] %v", name, e)
		}
	}
}

func TestV1CreateCommentVisibility(t *testing.T) {
	f := newCommentFix(t, nil)

	// w3TopicRole grants the creator role only, and other holds none.
	resp, out := f.postComment(t, w5ReplyRole, "sess-other", keyUUID(20), map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusNotFound || out["code"] != "NOT_FOUND" {
		t.Fatalf("comment into an unreadable topic %d %+v", resp.StatusCode, out)
	}
	resp, out = f.postComment(t, 930000399, "sess-alice", keyUUID(21), map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("comment on a missing reply %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE topic_id = ?`, w3TopicRole); n != 1 {
		t.Fatalf("role topic gained a comment: %d rows", n)
	}
}

func TestV1CreateCommentRequirements(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/replies/%d/comments", w3ReplyMin), "sess-alice",
		"/replies/{reply_id}/comments", "", nil, map[string]any{"text": "x"})
	body := problemMap(t, out)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("missing Idempotency-Key %d %+v", resp.StatusCode, body)
	}

	resp, body = f.postComment(t, w3ReplyMin, "", keyUUID(30), map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Fatalf("anonymous %d %+v", resp.StatusCode, body)
	}
}

// K19: the limit is checked against the raw value, before the handler trims.
func TestV1CommentTextLength(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(40), map[string]any{
		"text": strings.Repeat("a", 1007),
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("1007 characters %d %+v", resp.StatusCode, out)
	}

	resp, out = f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(41), map[string]any{
		"text": " " + strings.Repeat("a", 1007) + " ",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "VALIDATION_FAILED" {
		t.Fatalf("1007 characters plus surrounding spaces %d %+v", resp.StatusCode, out)
	}
	errs, _ := out["errors"].([]any)
	e, _ := errs[0].(map[string]any)
	if e["pointer"] != "/text" || e["reason"] != "TOO_LONG" {
		t.Fatalf("errors[0] %v", e)
	}
	params, _ := e["params"].(map[string]any)
	if asInt(params["max_length"]) != 1007 {
		t.Fatalf("params %v", params)
	}

	resp, out = f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(42), map[string]any{"text": "   "})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("whitespace only %d %+v", resp.StatusCode, out)
	}
	errs, _ = out["errors"].([]any)
	e, _ = errs[0].(map[string]any)
	if e["pointer"] != "/text" || e["reason"] != "TOO_SHORT" {
		t.Fatalf("errors[0] %v", e)
	}
}

func TestV1CommentAwardsWaitForTheCommit(t *testing.T) {
	f := newCommentFix(t, nil)

	const cb = "comment_award_commit_test"
	_ = f.db.Callback().Create().Before("gorm:create").Register(cb, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "message" {
			return
		}
		_ = tx.AddError(errors.New("message insert refused by the test"))
	})
	t.Cleanup(func() { _ = f.db.Callback().Create().Remove(cb) })

	before := len(f.snapshotAwards())
	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(50), map[string]any{"text": "boom"})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("create with a failing message insert %d %+v", resp.StatusCode, out)
	}
	if got := f.snapshotAwards(); len(got) != before {
		t.Fatalf("awards escaped a rolled-back transaction: %+v", got[before:])
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE content = 'boom'`); n != 0 {
		t.Fatal("comment row survived the rollback")
	}
}

func TestV1CreateCommentContentRejected(t *testing.T) {
	f := newCommentFix(t, denyChecker{})

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(60), map[string]any{"text": "bad"})
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "CONTENT_REJECTED" {
		t.Fatalf("denied content %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE content = 'bad'`); n != 0 {
		t.Fatal("a rejected comment was written")
	}
}

// A18: an unreadable target costs no trust round trip.
func TestV1CreateCommentChecksVisibilityBeforeContent(t *testing.T) {
	ck := &countingChecker{}
	f := newCommentFix(t, ck)

	resp, _ := f.postComment(t, w5ReplyRole, "sess-other", keyUUID(70), map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unreadable topic %d", resp.StatusCode)
	}
	if n := ck.calls.Load(); n != 0 {
		t.Fatalf("trust calls %d before the visibility decision", n)
	}
}

// K17: the author-renderability decision comes from OAuth, so a write closes
// when that call fails rather than going ahead on an unknown.
func TestV1CommentWritesFailClosedWhenOAuthIsDown(t *testing.T) {
	f := newCommentFix(t, nil)
	f.failOA.Store(true)
	f.UserClient.Invalidate(w3UserAlice, w3UserBob, w3UserStaff, w3UserOther)
	t.Cleanup(func() { f.failOA.Store(false) })

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", keyUUID(100), map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusServiceUnavailable || out["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("create %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE content = 'x'`); n != 0 {
		t.Fatal("a comment was written while the author decision was unknown")
	}
	resp, out = f.commentCall(t, http.MethodPut, "/like", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusServiceUnavailable || out["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("like %d %+v", resp.StatusCode, out)
	}
}

func TestV1CreateCommentIdempotency(t *testing.T) {
	f := newCommentFix(t, nil)
	key := keyUUID(110)

	resp, first := f.postComment(t, w3ReplyMin, "sess-alice", key, map[string]any{"text": "once"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, first)
	}
	resp, replay := f.postComment(t, w3ReplyMin, "sess-alice", key, map[string]any{"text": "once"})
	if resp.StatusCode != http.StatusCreated || replay["id"] != first["id"] {
		t.Fatalf("replay %d %+v", resp.StatusCode, replay)
	}
	if resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("Idempotency-Replayed %q", resp.Header.Get("Idempotency-Replayed"))
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE content = 'once'`); n != 1 {
		t.Fatalf("comment rows %d, want 1", n)
	}

	resp, out := f.postComment(t, w3ReplyMin, "sess-alice", key, map[string]any{"text": "twice"})
	if resp.StatusCode != http.StatusConflict || out["code"] != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("reused key %d %+v", resp.StatusCode, out)
	}
}
