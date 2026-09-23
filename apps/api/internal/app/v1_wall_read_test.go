package app

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

// Page edges are where dropped posts hide: a banned author's post or someone
// else's held post sitting last on a page must not make the cursor skip or
// repeat the post after it.
func TestV1WallListWalksEveryPageInPostOrder(t *testing.T) {
	f := newWallFix(t)
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	anchor := resAnchor("toolset", rcToolsetA)
	type seeded struct {
		id     int64
		author int64
		status int32
	}
	var all []seeded
	plan := []struct {
		author int64
		status int32
	}{
		{rcAlice, 0}, {rcBob, 0}, {rcBanned, 0}, {rcAlice, 0}, {rcOther, communityclient.PostHeld},
		{rcBob, 0}, {rcAlice, 0}, {rcBanned, 0}, {rcAlice, communityclient.PostHeld}, {rcBob, communityclient.PostDeleted},
		{rcOther, 0}, {rcAlice, 0}, {rcBanned, 0},
	}
	for i, p := range plan {
		// Every post shares one timestamp: the order must come from the post
		// number, not from time.
		id := f.cm.seed(anchorRes, anchor, p.author, "body "+strconv.Itoa(i), p.status, 0, base)
		all = append(all, seeded{id, p.author, p.status})
	}

	for _, viewer := range []struct {
		session string
		uid     int64
	}{{"", 0}, {"rc-alice", rcAlice}, {"rc-other", rcOther}} {
		var want []string
		for _, s := range all {
			if s.author == rcBanned {
				continue
			}
			if s.status == communityclient.PostHeld && s.author != viewer.uid {
				continue
			}
			want = append(want, pid(s.id))
		}
		for _, limit := range []int{1, 2, 3, 5, 100} {
			var got []string
			cursor := ""
			for pages := 0; ; pages++ {
				if pages > 20 {
					t.Fatalf("viewer %q limit %d: no end after 20 pages", viewer.session, limit)
				}
				extra := "&limit=" + strconv.Itoa(limit)
				if cursor != "" {
					extra += "&cursor=" + url.QueryEscape(cursor)
				}
				resp, out := f.list(t, viewer.session, "toolset", rcToolsetA, extra)
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("list %d %v", resp.StatusCode, out)
				}
				ids := itemIDs(t, out)
				if len(ids) > limit {
					t.Fatalf("page of %d over limit %d", len(ids), limit)
				}
				got = append(got, ids...)
				next, _ := out["next_cursor"].(string)
				if next == "" {
					break
				}
				cursor = next
			}
			if !slices.Equal(got, want) {
				t.Fatalf("viewer %q limit %d:\n got %v\nwant %v", viewer.session, limit, got, want)
			}
		}
	}
}

func TestV1WallListShapesATombstoneAndAHeldPost(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	anchor := resAnchor("resource", rcResource)
	root := f.cm.seed(anchorRes, anchor, rcAlice, "**root**", 0, 0, now)
	reply := f.cm.seed(anchorRes, anchor, rcBob, "reply", 0, root, now)
	f.cm.seed(anchorRes, anchor, rcAlice, "HOLDME", communityclient.PostHeld, 0, now)
	gone := f.cm.seed(anchorRes, anchor, rcOther, "gone", communityclient.PostDeleted, 0, now)

	resp, out := f.list(t, "rc-alice", "galgame_resource", rcResource, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %v", resp.StatusCode, out)
	}
	items, _ := out["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("items %d, want 4 (author sees own held post)", len(items))
	}
	first := items[0].(map[string]any)
	if first["object"] != "wall_comment" || first["subject_type"] != "galgame_resource" || first["subject_id"] != strconv.Itoa(rcResource) {
		t.Fatalf("first %v", first)
	}
	if first["parent_comment_id"] != nil || first["root_comment_id"] != nil || first["addressee"] != nil || first["state"] != "visible" {
		t.Fatalf("top-level shape %v", first)
	}
	second := items[1].(map[string]any)
	if second["id"] != pid(reply) || second["parent_comment_id"] != pid(root) || second["root_comment_id"] != pid(root) {
		t.Fatalf("reply %v", second)
	}
	if a, _ := second["addressee"].(map[string]any); a["id"] != strconv.Itoa(rcAlice) {
		t.Fatalf("reply addressee %v", second["addressee"])
	}
	if items[2].(map[string]any)["state"] != "held" {
		t.Fatalf("held %v", items[2])
	}
	tomb := items[3].(map[string]any)
	doc, _ := tomb["content"].(map[string]any)
	if tomb["id"] != pid(gone) || tomb["state"] != "deleted" || len(doc["children"].([]any)) != 0 {
		t.Fatalf("tombstone %v", tomb)
	}
	if v := tomb["viewer"].(map[string]any); v["can_edit"] != false || v["can_delete"] != false || v["can_like"] != false || v["can_flag"] != false {
		t.Fatalf("tombstone viewer %v", v)
	}
	if v := first["viewer"].(map[string]any); v["can_edit"] != true || v["can_like"] != false || v["can_flag"] != false {
		t.Fatalf("own comment viewer %v", v)
	}

	_, out = f.list(t, "rc-other", "galgame_resource", rcResource, "")
	if n := len(out["items"].([]any)); n != 3 {
		t.Fatalf("another viewer sees %d items, want 3 (no held post)", n)
	}
	_, out = f.list(t, "", "galgame_resource", rcResource, "")
	if first := out["items"].([]any)[0].(map[string]any); first["viewer"] != nil {
		t.Fatalf("anonymous viewer %v", first["viewer"])
	}
}

func TestV1WallListRefusesBadInput(t *testing.T) {
	f := newWallFix(t)
	resp, out := f.do(t, http.MethodGet, "/api/v1/wall-comments?subject_type=toolset", "", "/wall-comments", "", nil)
	if resp.StatusCode != http.StatusBadRequest || wallCode(out) != "INVALID_PARAMETER" {
		t.Fatalf("missing subject_id %d %v", resp.StatusCode, out)
	}
	resp, out = f.list(t, "", "forum", 1, "")
	if resp.StatusCode != http.StatusBadRequest || wallCode(out) != "UNKNOWN_ENUM_VALUE" {
		t.Fatalf("unknown subject_type %d %v", resp.StatusCode, out)
	}
	resp, out = f.list(t, "", "toolset", rcToolsetA, "&limit=101")
	if resp.StatusCode != http.StatusBadRequest || wallCode(out) != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit %d %v", resp.StatusCode, out)
	}

	for i := 0; i < 3; i++ {
		f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcAlice, "a", 0, 0, time.Now())
		f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetB), rcAlice, "b", 0, 0, time.Now())
	}
	_, out = f.list(t, "", "toolset", rcToolsetA, "&limit=1")
	cursor, _ := out["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("no cursor")
	}
	resp, out = f.list(t, "", "toolset", rcToolsetB, "&limit=1&cursor="+url.QueryEscape(cursor))
	if resp.StatusCode != http.StatusBadRequest || wallCode(out) != "INVALID_CURSOR" {
		t.Fatalf("cursor from another wall %d %v", resp.StatusCode, out)
	}
}

func TestV1WallListFollowsItsPage(t *testing.T) {
	f := newWallFix(t)
	for _, c := range []struct {
		typ  string
		id   int
		code string
	}{
		{"galgame", rcGalgameMissing, "NOT_FOUND"},
		{"galgame_rating", 950000299, "NOT_FOUND"},
		{"galgame_rating", rcRatingBanned, "NOT_FOUND"},
		{"website", 950000699, "NOT_FOUND"},
		{"toolset", 950000499, "NOT_FOUND"},
		{"galgame_quiz", 950000599, "NOT_FOUND"},
	} {
		resp, out := f.list(t, "rc-alice", c.typ, c.id, "")
		if resp.StatusCode != http.StatusNotFound || wallCode(out) != c.code {
			t.Fatalf("%s %d: %d %v", c.typ, c.id, resp.StatusCode, out)
		}
	}
	for _, c := range []struct {
		typ string
		id  int
	}{{"galgame", rcGalgame}, {"galgame_rating", rcRating}, {"galgame_resource", rcResource}, {"toolset", rcToolsetB}, {"galgame_quiz", rcQuizOpen}, {"website", rcWebsite}} {
		if resp, out := f.list(t, "", c.typ, c.id, ""); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %v", c.typ, resp.StatusCode, out)
		}
	}

	f.failGC.Store(true)
	resp, out := f.list(t, "", "galgame", rcGalgame, "")
	if resp.StatusCode != http.StatusServiceUnavailable || wallCode(out) != "SERVICE_UNAVAILABLE" {
		t.Fatalf("catalog down %d %v", resp.StatusCode, out)
	}
	f.failGC.Store(false)
	f.cm.down.Store(true)
	resp, out = f.list(t, "", "galgame", rcGalgame, "")
	if resp.StatusCode != http.StatusServiceUnavailable || wallCode(out) != "SERVICE_UNAVAILABLE" {
		t.Fatalf("community down %d %v, want 503 rather than an empty wall", resp.StatusCode, out)
	}
	f.cm.down.Store(false)
	f.cm.bogus.Store(true)
	resp, out = f.list(t, "", "galgame", rcGalgame, "")
	if resp.StatusCode != http.StatusInternalServerError || wallCode(out) != "INTERNAL_ERROR" {
		t.Fatalf("upstream refused our request %d %v", resp.StatusCode, out)
	}
}

func TestV1WallQuizGate(t *testing.T) {
	f := newWallFix(t)
	f.cm.seed(anchorRes, resAnchor("quiz", rcQuizSpoiler), rcBob, "spoiler talk", 0, 0, time.Now())
	for _, c := range []struct {
		session string
		status  int
	}{
		{"", http.StatusForbidden},
		{"rc-alice", http.StatusForbidden},
		{"rc-carol", http.StatusOK},
		{"rc-bob", http.StatusOK},
		{"rc-staff", http.StatusOK},
	} {
		resp, out := f.list(t, c.session, "galgame_quiz", rcQuizSpoiler, "")
		if resp.StatusCode != c.status {
			t.Fatalf("%q: %d %v, want %d", c.session, resp.StatusCode, out, c.status)
		}
		if c.status == http.StatusForbidden && wallCode(out) != "QUIZ_ANSWER_REQUIRED" {
			t.Fatalf("%q code %v", c.session, out)
		}
	}
	resp, out := f.create(t, "rc-alice", keyUUID(901), map[string]any{
		"subject_type": "galgame_quiz", "subject_id": strconv.Itoa(rcQuizSpoiler), "content_markdown": "hi",
	})
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "QUIZ_ANSWER_REQUIRED" {
		t.Fatalf("create on a locked quiz %d %v", resp.StatusCode, out)
	}
	if resp, _ := f.list(t, "", "galgame_quiz", rcQuizOpen, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("an open quiz is readable by anyone, got %d", resp.StatusCode)
	}
}

func TestV1WallGetComment(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	ok := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "hello", 0, 0, now)
	held := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "held", communityclient.PostHeld, 0, now)
	banned := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBanned, "x", 0, 0, now)
	foreign := f.cm.seed(0, "board-1", rcAlice, "board post", 0, 0, now)

	resp, out := f.onComment(t, http.MethodGet, "", ok, "", nil)
	if resp.StatusCode != http.StatusOK || out["id"] != pid(ok) || out["subject_type"] != "galgame" || out["subject_id"] != strconv.Itoa(rcGalgame) {
		t.Fatalf("get %d %v", resp.StatusCode, out)
	}
	if resp, out := f.onComment(t, http.MethodGet, "rc-alice", held, "", nil); resp.StatusCode != http.StatusOK || out["state"] != "held" {
		t.Fatalf("author reads own held post %d %v", resp.StatusCode, out)
	}
	for _, c := range []struct {
		session string
		id      int64
	}{{"rc-other", held}, {"", banned}, {"", foreign}, {"", 999999999}} {
		if resp, out := f.onComment(t, http.MethodGet, c.session, c.id, "", nil); resp.StatusCode != http.StatusNotFound || wallCode(out) != "NOT_FOUND" {
			t.Fatalf("get %d as %q: %d %v", c.id, c.session, resp.StatusCode, out)
		}
	}
}
