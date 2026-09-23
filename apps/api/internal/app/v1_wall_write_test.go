package app

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/perm"
)

func createBody(subjectType string, subjectID int, text string) map[string]any {
	return map[string]any{"subject_type": subjectType, "subject_id": strconv.Itoa(subjectID), "content_markdown": text}
}

func TestV1WallCreateOnEachWall(t *testing.T) {
	f := newWallFix(t)
	cases := []struct {
		typ     string
		id      int
		anchorK int32
		anchor  string
		counter string
	}{
		{"galgame", rcGalgame, anchorGame, strconv.Itoa(rcGalgame), "SELECT comment_count FROM galgame WHERE id = ?"},
		{"galgame_rating", rcRating, anchorRes, resAnchor("rating", rcRating), ""},
		{"galgame_resource", rcResource, anchorRes, resAnchor("resource", rcResource), "SELECT comment_count FROM galgame_resource WHERE id = ?"},
		{"galgame_quiz", rcQuizOpen, anchorRes, resAnchor("quiz", rcQuizOpen), "SELECT comment_count FROM galgame_quiz WHERE id = ?"},
		{"toolset", rcToolsetA, anchorRes, resAnchor("toolset", rcToolsetA), "SELECT comment_count FROM galgame_toolset WHERE id = ?"},
		{"website", rcWebsite, anchorRes, resAnchor("website", rcWebsite), "SELECT comment_count FROM galgame_website WHERE id = ?"},
	}
	for i, c := range cases {
		resp, out := f.create(t, "rc-alice", keyUUID(100+i), createBody(c.typ, c.id, "**hi** "+c.typ))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("%s: %d %v", c.typ, resp.StatusCode, out)
		}
		id := out["id"].(string)
		if resp.Header.Get("Location") != "/api/v1/wall-comments/"+id {
			t.Fatalf("%s location %q", c.typ, resp.Header.Get("Location"))
		}
		if out["subject_type"] != c.typ || out["subject_id"] != strconv.Itoa(c.id) || out["state"] != "visible" {
			t.Fatalf("%s shape %v", c.typ, out)
		}
		if f.cm.lastReq.AnchorKind != c.anchorK || f.cm.lastReq.AnchorID != c.anchor {
			t.Fatalf("%s sent anchor %d %q", c.typ, f.cm.lastReq.AnchorKind, f.cm.lastReq.AnchorID)
		}
		if c.counter != "" {
			if n := f.count(t, c.counter, c.id); n != 1 {
				t.Fatalf("%s counter %d", c.typ, n)
			}
		}
		n := f.count(t, `SELECT COUNT(*) FROM feed_activity WHERE source_id = ? AND user_id = ?`, asInt(id), rcAlice)
		if n != 1 {
			t.Fatalf("%s feed rows %d", c.typ, n)
		}
	}
	var link string
	var nsfw bool
	if err := f.db.Raw(`SELECT link, is_nsfw FROM feed_activity WHERE type = 'GALGAME_WEBSITE_COMMENT_CREATION' AND user_id = ?`, rcAlice).
		Row().Scan(&link, &nsfw); err != nil {
		t.Fatal(err)
	}
	if link != "/website/"+rcWebsiteSlug || !nsfw {
		t.Fatalf("website feed link %q nsfw %v", link, nsfw)
	}
	// Toolset, resource and quiz owners hear about top-level comments; rating and
	// website owners never did.
	if n := f.count(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND sender_id = ? AND type = 'commented'`, rcBob, rcAlice); n != 3 {
		t.Fatalf("owner notifications %d, want 3", n)
	}
}

func TestV1WallCreateRatingAddressesTheRatingAuthor(t *testing.T) {
	f := newWallFix(t)
	resp, out := f.create(t, "rc-alice", keyUUID(1), createBody("galgame_rating", rcRating, "nice rating"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %v", resp.StatusCode, out)
	}
	if a, _ := out["addressee"].(map[string]any); a["id"] != strconv.Itoa(rcBob) {
		t.Fatalf("addressee %v, want the rating author", out["addressee"])
	}
	parent := out["id"].(string)
	resp, out = f.create(t, "rc-bob", keyUUID(2), map[string]any{
		"subject_type": "galgame_rating", "subject_id": strconv.Itoa(rcRating), "content_markdown": "thanks", "parent_comment_id": parent,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("reply %d %v", resp.StatusCode, out)
	}
	if a, _ := out["addressee"].(map[string]any); a["id"] != strconv.Itoa(rcAlice) || out["parent_comment_id"] != parent {
		t.Fatalf("reply %v", out)
	}
	resp, out = f.create(t, "rc-alice", keyUUID(3), createBody("toolset", rcToolsetA, "top"))
	if resp.StatusCode != http.StatusCreated || out["addressee"] != nil {
		t.Fatalf("toolset top-level addressee %v", out["addressee"])
	}
}

func TestV1WallCreateRefuses(t *testing.T) {
	f := newWallFix(t)
	onB := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetB), rcOther, "b", 0, 0, time.Now())
	gone := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcOther, "g", communityclient.PostDeleted, 0, time.Now())

	expect := func(label string, resp *http.Response, out map[string]any, status int, code, reason string) {
		t.Helper()
		if resp.StatusCode != status || wallCode(out) != code {
			t.Fatalf("%s: %d %v, want %d %s", label, resp.StatusCode, out, status, code)
		}
		if reason != "" {
			errs, _ := out["errors"].([]any)
			if len(errs) != 1 || errs[0].(map[string]any)["reason"] != reason {
				t.Fatalf("%s errors %v, want %s", label, out["errors"], reason)
			}
		}
	}

	resp, out := f.create(t, "", keyUUID(1), createBody("toolset", rcToolsetA, "x"))
	expect("anonymous", resp, out, http.StatusUnauthorized, "MISSING_CREDENTIAL", "")
	resp, out = f.create(t, "rc-banned", keyUUID(2), createBody("toolset", rcToolsetA, "x"))
	expect("banned", resp, out, http.StatusForbidden, "ACCOUNT_BANNED", "")
	resp, out = f.create(t, "rc-alice", "", createBody("toolset", rcToolsetA, "x"))
	expect("no key", resp, out, http.StatusBadRequest, "INVALID_PARAMETER", "")
	resp, out = f.create(t, "rc-alice", keyUUID(3), createBody("toolset", rcToolsetA, "   "))
	expect("blank", resp, out, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "TOO_SHORT")

	long := strings.Repeat("字", 1008)
	resp, out = f.create(t, "rc-alice", keyUUID(4), createBody("toolset", rcToolsetA, long))
	expect("toolset over 1007", resp, out, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "TOO_LONG")
	if p := out["errors"].([]any)[0].(map[string]any)["params"].(map[string]any); asInt(p["max_length"]) != 1007 {
		t.Fatalf("max_length %v", p)
	}
	resp, out = f.create(t, "rc-alice", keyUUID(5), createBody("galgame_rating", rcRating, strings.Repeat("a", 1315)))
	expect("rating over 1314", resp, out, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "TOO_LONG")
	if resp, out := f.create(t, "rc-alice", keyUUID(6), createBody("galgame", rcGalgame, long)); resp.StatusCode != http.StatusCreated {
		t.Fatalf("galgame allows 5000: %d %v", resp.StatusCode, out)
	}
	// K19: the limit is on the value as sent, surrounding whitespace included.
	resp, out = f.create(t, "rc-alice", keyUUID(7), createBody("toolset", rcToolsetA, " "+strings.Repeat("a", 1006)+" "))
	expect("whitespace counts", resp, out, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "TOO_LONG")

	for i, parent := range []string{pid(onB), pid(gone), "999999999"} {
		body := createBody("toolset", rcToolsetA, "reply")
		body["parent_comment_id"] = parent
		resp, out = f.create(t, "rc-alice", keyUUID(20+i), body)
		expect("parent "+parent, resp, out, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "UNKNOWN_REFERENCE")
	}

	resp, out = f.create(t, "rc-alice", keyUUID(30), createBody("toolset", rcToolsetA, "BLOCKED word"))
	expect("word list", resp, out, http.StatusUnprocessableEntity, "CONTENT_REJECTED", "")

	resp, out = f.create(t, "rc-alice", keyUUID(31), createBody("galgame", rcGalgameMissing, "x"))
	expect("missing galgame", resp, out, http.StatusNotFound, "NOT_FOUND", "")

	resp, out = f.create(t, "rc-alice", keyUUID(32), createBody("toolset", rcToolsetA, "same key"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %v", resp.StatusCode, out)
	}
	resp, out = f.create(t, "rc-alice", keyUUID(32), createBody("toolset", rcToolsetA, "same key, other body"))
	expect("key reused", resp, out, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "")

	f.cm.limited.Store(true)
	resp, out = f.create(t, "rc-alice", keyUUID(33), createBody("toolset", rcToolsetA, "new account"))
	expect("rate limited", resp, out, http.StatusTooManyRequests, "RATE_LIMITED", "")
	f.cm.limited.Store(false)

	f.cm.closed.Store(true)
	resp, out = f.create(t, "rc-alice", keyUUID(35), createBody("toolset", rcToolsetA, "closed wall"))
	expect("closed wall", resp, out, http.StatusConflict, "INVALID_STATE_TRANSITION", "")
	f.cm.closed.Store(false)

	before := f.cm.comments.Load()
	f.failOA.Store(true)
	resp, out = f.create(t, "rc-carol", keyUUID(34), createBody("toolset", rcToolsetA, "oauth down"))
	expect("oauth down", resp, out, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "")
	f.failOA.Store(false)
	if f.cm.comments.Load() != before {
		t.Fatal("the comment reached the community service although the author could not be checked")
	}
}

func TestV1WallCreateGalgameMentions(t *testing.T) {
	f := newWallFix(t)
	body := "hi [@other](kungal-user:" + strconv.Itoa(rcOther) + ") and [@me](kungal-user:" + strconv.Itoa(rcAlice) + ")"
	resp, out := f.create(t, "rc-alice", keyUUID(1), createBody("galgame", rcGalgame, body))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %v", resp.StatusCode, out)
	}
	if got := f.cm.lastReq.MentionUserIDs; len(got) != 1 || got[0] != rcOther {
		t.Fatalf("mentions sent %v, want only the other user", got)
	}
	resp, _ = f.create(t, "rc-alice", keyUUID(2), createBody("toolset", rcToolsetA, body))
	if resp.StatusCode != http.StatusCreated || len(f.cm.lastReq.MentionUserIDs) != 0 {
		t.Fatalf("toolset wall passed mentions on: %v", f.cm.lastReq.MentionUserIDs)
	}
	var many strings.Builder
	for i := 0; i < 21; i++ {
		many.WriteString("[@u](kungal-user:" + strconv.Itoa(950000900+i) + ") ")
	}
	many.WriteString("[@other](kungal-user:" + strconv.Itoa(rcOther) + ")")
	resp, out = f.create(t, "rc-alice", keyUUID(3), createBody("galgame", rcGalgame, many.String()))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unknown users are dropped before the cap: %d %v", resp.StatusCode, out)
	}
}

func TestV1WallUpdate(t *testing.T) {
	f := newWallFix(t)
	own := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "old", 0, 0, time.Now())
	onTool := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcAlice, "old", 0, 0, time.Now())
	gone := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "x", communityclient.PostDeleted, 0, time.Now())

	resp, out := f.onComment(t, http.MethodPatch, "rc-alice", own, "", map[string]any{"content_markdown": "new [@o](kungal-user:" + strconv.Itoa(rcOther) + ")"})
	if resp.StatusCode != http.StatusOK || out["edited_at"] == nil || out["is_edited_by_moderator"] != false {
		t.Fatalf("author edit %d %v", resp.StatusCode, out)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND type = 'mentioned'`, rcOther); n != 1 {
		t.Fatalf("newly mentioned notified %d times", n)
	}
	resp, out = f.onComment(t, http.MethodPatch, "rc-staff", own, "", map[string]any{"content_markdown": "moderated"})
	if resp.StatusCode != http.StatusOK || out["is_edited_by_moderator"] != true {
		t.Fatalf("staff edit %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPatch, "rc-other", own, "", map[string]any{"content_markdown": "hijack"})
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "PERMISSION_REQUIRED" {
		t.Fatalf("other edit %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPatch, "bearer:rc-staff-token", own, "", map[string]any{"content_markdown": "bearer"})
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "PERMISSION_REQUIRED" {
		t.Fatalf("Bearer staff must hold no staff power: %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPatch, "rc-alice", onTool, "", map[string]any{"content_markdown": strings.Repeat("a", 1008)})
	if resp.StatusCode != http.StatusUnprocessableEntity || wallCode(out) != "VALIDATION_FAILED" {
		t.Fatalf("edit over the toolset limit %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPatch, "rc-alice", gone, "", map[string]any{"content_markdown": "revive"})
	if resp.StatusCode != http.StatusConflict || wallCode(out) != "INVALID_STATE_TRANSITION" {
		t.Fatalf("edit a tombstone %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPatch, "rc-alice", own, "", map[string]any{"content_markdown": "BLOCKED"})
	if resp.StatusCode != http.StatusUnprocessableEntity || wallCode(out) != "CONTENT_REJECTED" {
		t.Fatalf("word list on edit %d %v", resp.StatusCode, out)
	}

	resp, out = f.onComment(t, http.MethodGet, "rc-alice", own, "/source", nil)
	if resp.StatusCode != http.StatusOK || out["content_markdown"] != "moderated" || out["wall_comment_id"] != pid(own) {
		t.Fatalf("source %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodGet, "rc-other", own, "/source", nil)
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "PERMISSION_REQUIRED" {
		t.Fatalf("source for a non-editor %d %v", resp.StatusCode, out)
	}
}

func TestV1WallDeletePermissions(t *testing.T) {
	f := newWallFix(t)
	perm.SetUserOverrides(map[int][]perm.Override{rcGalMod: {{Permission: perm.CommentGalgameDelete, Effect: perm.EffectGrant}}})
	t.Cleanup(func() { perm.SetUserOverrides(nil) })

	post := func(kind int32, anchor string) int64 {
		return f.cm.seed(kind, anchor, rcAlice, "c", 0, 0, time.Now())
	}
	cases := []struct {
		label   string
		session string
		kind    int32
		anchor  string
		status  int
	}{
		{"galgame delete holder on a galgame wall", "rc-galmod", anchorGame, strconv.Itoa(rcGalgame), http.StatusNoContent},
		{"galgame delete holder on a rating wall", "rc-galmod", anchorRes, resAnchor("rating", rcRating), http.StatusForbidden},
		{"galgame delete holder on a quiz wall", "rc-galmod", anchorRes, resAnchor("quiz", rcQuizOpen), http.StatusForbidden},
		{"toolset owner on own toolset", "rc-bob", anchorRes, resAnchor("toolset", rcToolsetA), http.StatusNoContent},
		{"toolset owner on another toolset", "rc-bob", anchorRes, resAnchor("toolset", rcToolsetB), http.StatusForbidden},
		{"rating author on own rating", "rc-bob", anchorRes, resAnchor("rating", rcRating), http.StatusNoContent},
		{"galgame creator on a rating of it", "rc-creator", anchorRes, resAnchor("rating", rcRating), http.StatusForbidden},
		{"resource publisher", "rc-bob", anchorRes, resAnchor("resource", rcResource), http.StatusNoContent},
		{"quiz author", "rc-bob", anchorRes, resAnchor("quiz", rcQuizOpen), http.StatusNoContent},
		{"website has no owner", "rc-bob", anchorRes, resAnchor("website", rcWebsite), http.StatusForbidden},
		{"staff anywhere", "rc-staff", anchorRes, resAnchor("website", rcWebsite), http.StatusNoContent},
		{"author", "rc-alice", anchorRes, resAnchor("toolset", rcToolsetB), http.StatusNoContent},
		{"stranger", "rc-other", anchorRes, resAnchor("toolset", rcToolsetA), http.StatusForbidden},
	}
	for _, c := range cases {
		id := post(c.kind, c.anchor)
		resp, out := f.onComment(t, http.MethodDelete, c.session, id, "", nil)
		if resp.StatusCode != c.status {
			t.Fatalf("%s: %d %v, want %d", c.label, resp.StatusCode, out, c.status)
		}
		if c.status == http.StatusForbidden && wallCode(out) != "PERMISSION_REQUIRED" {
			t.Fatalf("%s code %v", c.label, out)
		}
		wantStatus := int32(communityclient.PostVisible)
		if c.status == http.StatusNoContent {
			wantStatus = communityclient.PostDeleted
		}
		if got := f.cm.post(id).Status; got != wantStatus {
			t.Fatalf("%s: upstream status %d", c.label, got)
		}
	}
}

func TestV1WallDeleteTwiceCountsOnce(t *testing.T) {
	f := newWallFix(t)
	resp, out := f.create(t, "rc-alice", keyUUID(1), createBody("galgame_resource", rcResource, "one"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("%d %v", resp.StatusCode, out)
	}
	f.create(t, "rc-alice", keyUUID(2), createBody("galgame_resource", rcResource, "two"))
	id := int64(asInt(out["id"]))
	for i := 0; i < 2; i++ {
		if resp, out := f.onComment(t, http.MethodDelete, "rc-alice", id, "", nil); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("delete #%d %d %v", i, resp.StatusCode, out)
		}
	}
	if n := f.count(t, `SELECT comment_count FROM galgame_resource WHERE id = ?`, rcResource); n != 1 {
		t.Fatalf("comment_count %d, want 1", n)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM feed_activity WHERE source_id = ?`, id); n != 0 {
		t.Fatalf("feed row survived the delete")
	}
	resp, out = f.onComment(t, http.MethodGet, "", id, "", nil)
	if resp.StatusCode != http.StatusOK || out["state"] != "deleted" {
		t.Fatalf("tombstone %d %v", resp.StatusCode, out)
	}
}

func TestV1WallCreateHeldAndFollowedOwner(t *testing.T) {
	f := newWallFix(t)
	resp, out := f.create(t, "rc-alice", keyUUID(1), createBody("toolset", rcToolsetA, "HOLDME please"))
	if resp.StatusCode != http.StatusCreated || out["state"] != "held" {
		t.Fatalf("held create %d %v", resp.StatusCode, out)
	}
	f.cm.mu.Lock()
	f.cm.follows[rcBob] = true
	f.cm.mu.Unlock()
	before := f.count(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ?`, rcBob)
	f.create(t, "rc-other", keyUUID(2), createBody("toolset", rcToolsetA, "followed"))
	if n := f.count(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ?`, rcBob); n != before {
		t.Fatal("an owner who follows the wall is notified by the community service, not twice")
	}
}
