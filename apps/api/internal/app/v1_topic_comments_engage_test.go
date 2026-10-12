package app

import (
	"net/http"
	"strconv"
	"testing"
)

func TestV1UpdateComment(t *testing.T) {
	ck := &countingChecker{}
	f := newCommentFix(t, ck)

	resp, out := f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-alice", map[string]any{"text": "edited"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit %d %+v", resp.StatusCode, out)
	}
	if docPlainText(t, commentContent(t, out)) != "edited" || out["edited_at"] == nil {
		t.Fatalf("%+v", out)
	}
	if out["parent_comment_id"] != nil || out["reply_id"] != strconv.Itoa(w3ReplyMin) {
		t.Fatalf("edit response dropped a field: %+v", out)
	}
	if n := ck.calls.Load(); n != 1 {
		t.Fatalf("trust calls %d, want 1", n)
	}

	// K18: no new text, no check.
	resp, out = f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-alice", map[string]any{"text": "edited"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unchanged edit %d %+v", resp.StatusCode, out)
	}
	if n := ck.calls.Load(); n != 1 {
		t.Fatalf("trust calls %d after an unchanged body, want 1", n)
	}

	resp, out = f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-alice", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty patch %d %+v", resp.StatusCode, out)
	}
	if n := ck.calls.Load(); n != 1 {
		t.Fatalf("trust calls %d after an empty patch, want 1", n)
	}
}

func TestV1UpdateCommentAuthorization(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-bob", map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("another user %d %+v", resp.StatusCode, out)
	}
	resp, out = f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-staff", map[string]any{"text": "moderated"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff %d %+v", resp.StatusCode, out)
	}
	if docPlainText(t, commentContent(t, out)) != "moderated" {
		t.Fatalf("%+v", out)
	}
}

func TestV1UpdateCommentContentRejected(t *testing.T) {
	f := newCommentFix(t, denyChecker{})

	resp, out := f.commentCall(t, http.MethodPatch, "", w5CommentAlice, "sess-alice", map[string]any{"text": "bad"})
	if resp.StatusCode != http.StatusUnprocessableEntity || out["code"] != "CONTENT_REJECTED" {
		t.Fatalf("%d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE id = ? AND content = 'alice-comment'`, w5CommentAlice); n != 1 {
		t.Fatal("a rejected edit was written")
	}
}

// A3: the charge is best effort. A moderator could not remove a comment whose
// author had nothing left to pay with.
func TestV1DeleteCommentNeverWaitsOnTheBalance(t *testing.T) {
	f := newCommentFix(t, nil)
	if err := f.db.Exec(`UPDATE kungal_user_state SET moemoepoint = 0 WHERE user_id = ?`, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}

	resp, out := f.commentCall(t, http.MethodDelete, "", w5CommentAlice, "sess-staff", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("staff delete of a comment by a user with 0 moemoepoint: %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment WHERE id = ?`, w5CommentAlice); n != 0 {
		t.Fatal("the comment survived")
	}
	if got := awardsFor(f.snapshotAwards(), "topic_comment:"+strconv.Itoa(w5CommentAlice)); len(got) != 0 {
		t.Fatalf("charged a user with nothing: %+v", got)
	}

	// The same for the author deleting their own.
	resp, out = f.commentCall(t, http.MethodDelete, "", w5CommentBob, "sess-bob", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("author delete %d %+v", resp.StatusCode, out)
	}
}

func TestV1DeleteComment(t *testing.T) {
	f := newCommentFix(t, nil)

	// A child keeps the parent alive in the tree only through its id, which
	// the FK sets to NULL when the parent goes.
	resp, out := f.postComment(t, w3ReplyMin, "sess-bob", keyUUID(90), map[string]any{
		"text": "child", "parent_comment_id": strconv.Itoa(w5CommentAlice),
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("seed child %d %+v", resp.StatusCode, out)
	}
	child := asInt(out["id"])

	resp, out = f.commentCall(t, http.MethodDelete, "", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusForbidden || out["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("another user %d %+v", resp.StatusCode, out)
	}

	before := f.commentRowCount(t, `SELECT comment_count FROM topic WHERE id = ?`, w3TopicFloors)
	resp, out = f.commentCall(t, http.MethodDelete, "", w5CommentAlice, "sess-alice", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("author delete %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT comment_count FROM topic WHERE id = ?`, w3TopicFloors); n != before-1 {
		t.Fatalf("comment_count %d, want %d", n, before-1)
	}
	if n := f.commentRowCount(t,
		`SELECT COUNT(*) FROM topic_comment WHERE id = ? AND parent_comment_id IS NULL`, child); n != 1 {
		t.Fatal("the child should have become top-level")
	}

	got := awardsFor(f.snapshotAwards(), "topic_comment:"+strconv.Itoa(w5CommentAlice))
	if len(got) != 1 || got[0].delta != -3 || got[0].reason != "content_removed" {
		t.Fatalf("delete award %+v", got)
	}

	resp, out = f.getComment(t, w5CommentAlice, "sess-alice")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("after delete %d %+v", resp.StatusCode, out)
	}
}

func TestV1CommentLikeSlot(t *testing.T) {
	f := newCommentFix(t, nil)
	ref := "topic_comment:" + strconv.Itoa(w5CommentAlice)

	resp, out := f.commentCall(t, http.MethodPut, "/like", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("like %d %+v", resp.StatusCode, out)
	}
	if asInt(out["like_count"]) != 1 || viewerOf(t, out)["has_liked"] != true {
		t.Fatalf("%+v", out)
	}
	if got := awardsFor(f.snapshotAwards(), ref); len(got) != 1 || got[0].delta != 1 || got[0].userID != w3UserAlice {
		t.Fatalf("like award %+v", got)
	}

	// Setting a slot already set changes nothing: no second row, no second
	// award, no second notification.
	resp, out = f.commentCall(t, http.MethodPut, "/like", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusOK || asInt(out["like_count"]) != 1 {
		t.Fatalf("replayed like %d %+v", resp.StatusCode, out)
	}
	if got := awardsFor(f.snapshotAwards(), ref); len(got) != 1 {
		t.Fatalf("replayed like awarded twice: %+v", got)
	}
	if n := f.commentRowCount(t,
		`SELECT COUNT(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked'`,
		w3UserBob, w3UserAlice); n != 1 {
		t.Fatalf("liked messages %d, want 1", n)
	}
	wantNotifRefs(t, f.db, []string{notifRef("liked", "topic_comment", w5CommentAlice, "topic", w3TopicFloors)},
		"sender_id = ? AND receiver_id = ? AND type = 'liked'", w3UserBob, w3UserAlice)

	resp, out = f.commentCall(t, http.MethodDelete, "/like", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusOK || asInt(out["like_count"]) != 0 {
		t.Fatalf("unlike %d %+v", resp.StatusCode, out)
	}
	if viewerOf(t, out)["has_liked"] != false {
		t.Fatalf("viewer %+v", out["viewer"])
	}
	if got := awardsFor(f.snapshotAwards(), ref); len(got) != 2 || got[1].delta != -1 {
		t.Fatalf("unlike award %+v", got)
	}

	resp, out = f.commentCall(t, http.MethodDelete, "/like", w5CommentAlice, "sess-bob", nil)
	if resp.StatusCode != http.StatusOK || asInt(out["like_count"]) != 0 {
		t.Fatalf("replayed unlike %d %+v", resp.StatusCode, out)
	}
	if got := awardsFor(f.snapshotAwards(), ref); len(got) != 2 {
		t.Fatalf("replayed unlike awarded again: %+v", got)
	}
}

func TestV1CommentSelfLikeForbidden(t *testing.T) {
	f := newCommentFix(t, nil)

	resp, out := f.getComment(t, w5CommentAlice, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, out)
	}
	if viewerOf(t, out)["can_like"] != false {
		t.Fatalf("can_like on your own comment: %v", out["viewer"])
	}

	resp, out = f.commentCall(t, http.MethodPut, "/like", w5CommentAlice, "sess-alice", nil)
	if resp.StatusCode != http.StatusForbidden || out["code"] != "SELF_LIKE_FORBIDDEN" {
		t.Fatalf("self like %d %+v", resp.StatusCode, out)
	}
	if n := f.commentRowCount(t, `SELECT COUNT(*) FROM topic_comment_like WHERE topic_comment_id = ?`, w5CommentAlice); n != 0 {
		t.Fatal("a self like was written")
	}
}

func TestV1CommentLikeVisibility(t *testing.T) {
	f := newCommentFix(t, nil)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		resp, out := f.commentCall(t, method, "/like", w5CommentRole, "sess-other", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s like on an unreadable comment %d %+v", method, resp.StatusCode, out)
		}
		resp, out = f.commentCall(t, method, "/like", w5CommentHidden, "sess-bob", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s like on a hidden comment %d %+v", method, resp.StatusCode, out)
		}
	}
}
