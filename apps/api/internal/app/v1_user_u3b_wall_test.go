package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/collect"
)

func (f *wallFix) userWall(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(rcAlice)+"/wall-comments?"+query,
		session, "/users/{user_id}/wall-comments", "", nil)
}

func TestV1UserWallCommentsAuthoredWalk(t *testing.T) {
	f := newWallFix(t)
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var want []string
	for i := 0; i < 5; i++ {
		id := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "g"+strconv.Itoa(i), 0, 0, base)
		want = append([]string{pid(id)}, want...)
	}
	var got []string
	cursor := ""
	for pages := 0; pages < 20; pages++ {
		q := "relation=authored&limit=2"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, body := f.userWall(t, "", q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", pages, resp.StatusCode, body)
		}
		ids := itemIDs(t, body)
		got = append(got, ids...)
		cursor, _ = body["next_cursor"].(string)
		if cursor == "" {
			break
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("walked %v, want %v", got, want)
	}
}

func TestV1UserWallCommentsSubjectTypeFilter(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	g := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "game", 0, 0, now)
	f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcAlice, "res", 0, 0, now)
	resp, body := f.userWall(t, "", "relation=authored&subject_type=galgame&limit=100")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filter %d %+v", resp.StatusCode, body)
	}
	ids := itemIDs(t, body)
	if len(ids) != 1 || ids[0] != pid(g) {
		t.Fatalf("filtered %v, want [%s]", ids, pid(g))
	}
}

func TestV1UserWallCommentsUpstreamCallCap(t *testing.T) {
	f := newWallFix(t)
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcAlice, "old-resource", 0, 0, base)
	for i := 0; i < 20; i++ {
		f.cm.seed(anchorRes, resAnchor("quiz", rcQuizOpen), rcAlice, "q"+strconv.Itoa(i), 0, 0, base)
	}
	f.cm.authorPosts.Store(0)
	resp, body := f.userWall(t, "", "relation=authored&subject_type=galgame_resource&limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cap %d %+v", resp.StatusCode, body)
	}
	n := int(f.cm.authorPosts.Load())
	if n > 3 {
		t.Fatalf("upstream calls %d, want ≤ 3", n)
	}
}

func TestV1UserWallCommentsCursorBoundToSubjectType(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "g"+strconv.Itoa(i), 0, 0, now)
		f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcAlice, "r"+strconv.Itoa(i), 0, 0, now)
	}
	resp, body := f.userWall(t, "", "relation=authored&subject_type=galgame&limit=1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first %d %+v", resp.StatusCode, body)
	}
	cur, _ := body["next_cursor"].(string)
	if cur == "" {
		t.Fatal("expected cursor")
	}
	resp, body = f.userWall(t, "", "relation=authored&subject_type=galgame_resource&limit=1&cursor="+url.QueryEscape(cur))
	if resp.StatusCode != http.StatusBadRequest || wallCode(body) != "INVALID_CURSOR" {
		t.Fatalf("reused cursor %d %+v", resp.StatusCode, body)
	}
}

func TestV1UserWallCommentsMalformedCursorIs400(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.userWall(t, "", "relation=authored&cursor=nope")
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
	cur := collect.EncodeCursor("id_desc", collect.Fingerprint("authored", "galgame", strconv.Itoa(rcAlice)), "not-int")
	resp, body = f.userWall(t, "", "relation=authored&subject_type=galgame&cursor="+url.QueryEscape(cur))
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1UserWallCommentsOwnerMissingBannedUnavailable(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.do(t, http.MethodGet, "/api/v1/users/971000099/wall-comments?relation=authored",
		"", "/users/{user_id}/wall-comments", "", nil)
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	resp, body = f.do(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(rcBanned)+"/wall-comments?relation=authored",
		"", "/users/{user_id}/wall-comments", "", nil)
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	f.failOA.Store(true)
	resp, body = f.userWall(t, "", "relation=authored")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	f.failOA.Store(false)
	f.cm.down.Store(true)
	resp, body = f.userWall(t, "", "relation=authored")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserWallCommentsUnknownEnumAndLimit(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.userWall(t, "", "relation=shared")
	if resp.StatusCode != http.StatusBadRequest || wallCode(body) != "UNKNOWN_ENUM_VALUE" {
		t.Fatalf("relation %d %+v", resp.StatusCode, body)
	}
	resp, body = f.userWall(t, "", "relation=authored&subject_type=forum")
	if resp.StatusCode != http.StatusBadRequest || wallCode(body) != "UNKNOWN_ENUM_VALUE" {
		t.Fatalf("subject_type %d %+v", resp.StatusCode, body)
	}
	resp, body = f.userWall(t, "", "relation=authored&limit=101")
	if resp.StatusCode != http.StatusBadRequest || wallCode(body) != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit %d %+v", resp.StatusCode, body)
	}
}

func TestV1UserWallCommentsLikedWalk(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	a := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBob, "a", 0, 0, now)
	b := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBob, "b", 0, 0, now)
	c := f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcBob, "c", 0, 0, now)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	run(`INSERT INTO galgame_post_like (id, post_id, user_id, created) VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`,
		971000501, a, rcAlice, now, 971000502, b, rcAlice, now, 971000503, c, rcAlice, now)

	var got []string
	cursor := ""
	for pages := 0; pages < 20; pages++ {
		q := "relation=liked&limit=2"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		resp, body := f.userWall(t, "", q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("liked page %d: %d %+v", pages, resp.StatusCode, body)
		}
		got = append(got, itemIDs(t, body)...)
		cursor, _ = body["next_cursor"].(string)
		if cursor == "" {
			break
		}
	}
	want := []string{pid(c), pid(b), pid(a)}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("liked walked %v, want %v", got, want)
	}

	resp, body := f.userWall(t, "", "relation=liked&subject_type=galgame&limit=100")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("liked filter %d %+v", resp.StatusCode, body)
	}
	ids := itemIDs(t, body)
	if len(ids) != 2 || ids[0] != pid(b) || ids[1] != pid(a) {
		t.Fatalf("liked galgame filter %v", ids)
	}
}

func TestV1UserWallCommentsLikedCallCap(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	var posts []int64
	for i := 0; i < 10; i++ {
		posts = append(posts, f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcBob, "r"+strconv.Itoa(i), 0, 0, now))
	}
	for i, postID := range posts {
		if err := f.db.Exec(`INSERT INTO galgame_post_like (id, post_id, user_id, created) VALUES (?, ?, ?, ?)`,
			971000601+i, postID, rcAlice, now).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.cm.resolves.Store(0)
	resp, body := f.userWall(t, "", "relation=liked&subject_type=galgame&limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("liked cap %d %+v", resp.StatusCode, body)
	}
	n := int(f.cm.resolves.Load())
	if n > 3 {
		t.Fatalf("resolve calls %d, want ≤ 3", n)
	}
}
