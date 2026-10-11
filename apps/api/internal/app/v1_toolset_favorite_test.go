package app

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func (f *writeFix) favoriteToolset(t *testing.T, method string, id int, session string) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, "/api/v1/toolsets/"+idStr(id)+"/favorite", "/toolsets/{toolset_id}/favorite", session, "", nil)
}

func (f *writeFix) favoriteRowID(t *testing.T, toolsetID, userID int) int {
	t.Helper()
	return f.scalar(t, `SELECT id FROM galgame_toolset_favorite WHERE toolset_id = ? AND user_id = ?`, toolsetID, userID)
}

func wantEngagement(t *testing.T, resp *http.Response, body map[string]any, count int, favorited bool) {
	t.Helper()
	v, _ := body["viewer"].(map[string]any)
	if resp.StatusCode != http.StatusOK || body["object"] != "toolset_engagement" ||
		asInt(body["favorite_count"]) != count || v == nil || v["has_favorited"] != favorited {
		t.Fatalf("engagement %d %+v, want count %d favorited %v", resp.StatusCode, body, count, favorited)
	}
}

func (f *writeFix) wantFavoriteCountConsistent(t *testing.T, toolsetID, want int) {
	t.Helper()
	cached := f.scalar(t, `SELECT favorite_count FROM galgame_toolset WHERE id = ?`, toolsetID)
	rows := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_favorite WHERE toolset_id = ?`, toolsetID)
	if cached != want || rows != want {
		t.Errorf("toolset %d favorite_count %d, rows %d, want %d", toolsetID, cached, rows, want)
	}
}

func TestV1PutDeleteToolsetFavorite(t *testing.T) {
	f := newToolsetFix(t, nil)
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset WHERE id = ?`, g1TSMain)

	resp, got := f.favoriteToolset(t, http.MethodPut, g1TSMain, "sess-bob")
	wantEngagement(t, resp, got, 1, true)
	if strID(got["toolset_id"]) != idStr(g1TSMain) {
		t.Errorf("toolset_id %v", got["toolset_id"])
	}
	f.wantFavoriteCountConsistent(t, g1TSMain, 1)
	rowID := strconv.Itoa(f.favoriteRowID(t, g1TSMain, w3UserBob))
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].userID != w3UserAlice || aw[0].delta != 1 || aw[0].reason != moemoepoint.ReasonLiked ||
		aw[0].ref != moemoepoint.Ref("toolset", g1TSMain) ||
		aw[0].key != moemoepoint.Key("favorited", "toolset_favorite_"+rowID) {
		t.Errorf("favorite award %+v", aw)
	}

	resp, got = f.favoriteToolset(t, http.MethodPut, g1TSMain, "sess-bob")
	wantEngagement(t, resp, got, 1, true)
	f.wantFavoriteCountConsistent(t, g1TSMain, 1)
	if n := len(f.snapshotAwards()); n != 1 {
		t.Errorf("a repeated favorite awarded again: %+v", f.snapshotAwards())
	}

	resp, got = f.favoriteToolset(t, http.MethodDelete, g1TSMain, "sess-bob")
	wantEngagement(t, resp, got, 0, false)
	f.wantFavoriteCountConsistent(t, g1TSMain, 0)
	aw = f.snapshotAwards()
	if len(aw) != 2 || aw[1].userID != w3UserAlice || aw[1].delta != -1 || aw[1].reason != moemoepoint.ReasonLiked ||
		aw[1].key != moemoepoint.Key("unfavorited", "toolset_favorite_"+rowID) {
		t.Errorf("unfavorite award %+v", aw)
	}

	resp, got = f.favoriteToolset(t, http.MethodDelete, g1TSMain, "sess-bob")
	wantEngagement(t, resp, got, 0, false)
	f.wantFavoriteCountConsistent(t, g1TSMain, 0)
	if n := len(f.snapshotAwards()); n != 2 {
		t.Errorf("a repeated unfavorite moved points: %+v", f.snapshotAwards())
	}

	resp, got = f.favoriteToolset(t, http.MethodPut, g1TSMain, "sess-alice")
	wantEngagement(t, resp, got, 1, true)
	if n := len(f.snapshotAwards()); n != 2 {
		t.Errorf("a self-favorite awarded: %+v", f.snapshotAwards())
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset WHERE id = ?`, g1TSMain); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("a favorite moved updated: %v -> %v", updated, got)
	}
}

func TestV1ToolsetFavoriteConcurrent(t *testing.T) {
	f := newToolsetFix(t, nil)
	var wg sync.WaitGroup
	codes := make([]int, 4)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, _ := f.favoriteToolset(t, http.MethodPut, g1TSMain, "sess-bob")
			codes[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()
	for _, c := range codes {
		if c != http.StatusOK {
			t.Errorf("concurrent favorites answered %v", codes)
			break
		}
	}
	f.wantFavoriteCountConsistent(t, g1TSMain, 1)
	if n := len(f.snapshotAwards()); n != 1 {
		t.Errorf("concurrent favorites awarded %d times", n)
	}
}

func TestV1ToolsetFavoriteRefusals(t *testing.T) {
	f := newToolsetFix(t, nil)
	hidden := g1TSTied + 5
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		resp, got := f.favoriteToolset(t, method, hidden, "sess-bob")
		wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
		resp, got = f.favoriteToolset(t, method, g1TSGone, "sess-bob")
		wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
		resp, got = f.favoriteToolset(t, method, g1TSMain, "")
		wantCode(t, resp, got, http.StatusUnauthorized, problem.CodeMissingCredential)
		resp, got = f.favoriteToolset(t, method, g1TSMain, "sess-banned")
		wantCode(t, resp, got, http.StatusForbidden, problem.CodeAccountBanned)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_favorite WHERE toolset_id IN (?, ?)`, hidden, g1TSMain); n != 0 {
		t.Errorf("a refused favorite wrote %d rows", n)
	}
	f.wantFavoriteCountConsistent(t, hidden, 0)
	if n := len(f.snapshotAwards()); n != 0 {
		t.Errorf("a refused favorite moved points: %+v", f.snapshotAwards())
	}
}

func TestV1GetToolsetFavoriteState(t *testing.T) {
	f := newToolsetFix(t, nil)
	f.favoriteToolset(t, http.MethodPut, g1TSMain, "sess-bob")

	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["favorite_count"]) != 1 || body["viewer"] != nil {
		t.Fatalf("anonymous detail %d %+v", resp.StatusCode, body)
	}
	for session, want := range map[string]bool{"sess-bob": true, "sess-alice": false} {
		_, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", session, "", nil)
		v, _ := body["viewer"].(map[string]any)
		if v == nil || v["has_favorited"] != want {
			t.Errorf("%s viewer %+v, want has_favorited %v", session, body["viewer"], want)
		}
	}
}

func TestV1ListUserFavoritedToolsets(t *testing.T) {
	f := newToolsetFix(t, nil)
	at := time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)
	for _, fav := range []struct {
		toolset int
		created time.Time
	}{
		{g1TSMain, at},
		{g1TSTied, at.Add(time.Hour)},
		{g1TSTied + 1, at.Add(time.Hour)},
		{g1TSTied + 5, at.Add(2 * time.Hour)},
		{g1TSBob, at.Add(-time.Hour)},
	} {
		if err := f.db.Exec(`INSERT INTO galgame_toolset_favorite (toolset_id, user_id, created) VALUES (?, ?, ?)`,
			fav.toolset, w3UserBob, fav.created).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := f.db.Exec(`INSERT INTO galgame_toolset_favorite (toolset_id, user_id, created) VALUES (?, ?, ?)`,
		g1TSBad, w3UserOther, at).Error; err != nil {
		t.Fatal(err)
	}
	want := f.sqlIDs(t, `SELECT fav.toolset_id::text FROM galgame_toolset_favorite fav
		JOIN galgame_toolset t ON t.id = fav.toolset_id
		WHERE fav.user_id = ? AND t.user_id IN (`+renderableAuthorsSQL()+`)
		ORDER BY fav.created DESC, fav.id DESC`, w3UserBob)
	if len(want) != 4 {
		t.Fatalf("seed gave %v", want)
	}

	var walked []string
	for page := 1; page <= 3; page++ {
		url := fmt.Sprintf("/api/v1/users/%d/toolsets?relation=favorited&limit=2&page=%d", w3UserBob, page)
		resp, body := f.ts(t, http.MethodGet, url, "/users/{user_id}/toolsets", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("favorited page %d: %d %+v", page, resp.StatusCode, body)
		}
		if asInt(body["total"]) != len(want) {
			t.Errorf("favorited total %v, want %d", body["total"], len(want))
		}
		walked = append(walked, adminItemIDs(body)...)
	}
	if fmt.Sprint(walked) != fmt.Sprint(want) {
		t.Errorf("favorited walked %v, want %v", walked, want)
	}

	resp, body := f.ts(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/toolsets?limit=100", w3UserBob),
		"/users/{user_id}/toolsets", "", "", nil)
	authored := f.sqlIDs(t, `SELECT id::text FROM galgame_toolset
		WHERE user_id = ? AND id BETWEEN 930002201 AND 930002299 ORDER BY created DESC, id DESC`, w3UserBob)
	if resp.StatusCode != http.StatusOK || fmt.Sprint(adminItemIDs(body)) != fmt.Sprint(authored) {
		t.Errorf("default relation %d %v, want authored %v", resp.StatusCode, adminItemIDs(body), authored)
	}

	resp, body = f.ts(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/toolsets?relation=liked", w3UserBob),
		"/users/{user_id}/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
	resp, body = f.ts(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/toolsets?relation=favorited", g1HiddenA),
		"/users/{user_id}/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1DeleteToolsetTakesItsFavorites(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.favoriteToolset(t, http.MethodPut, g1TSBad, "sess-bob")
	wantEngagement(t, resp, got, 1, true)
	resp, got = f.ts(t, http.MethodDelete, "/api/v1/toolsets/"+idStr(g1TSBad), "/toolsets/{toolset_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_favorite WHERE toolset_id = ?`, g1TSBad); n != 0 {
		t.Errorf("%d favorites outlived their toolset", n)
	}
}
