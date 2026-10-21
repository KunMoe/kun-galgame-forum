package app

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

func (f *wallFix) setRendered(t *testing.T, workID int, rendered bool) {
	t.Helper()
	if err := f.db.Exec(`UPDATE galgame SET catalog_rendered = ? WHERE id = ?`, rendered, workID).Error; err != nil {
		t.Fatal(err)
	}
}

func (f *wallFix) followedKeys(t *testing.T, session string) []string {
	t.Helper()
	resp, body := f.followedWalls(t, session, url.Values{"limit": {"50"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("followed walls %d %+v", resp.StatusCode, body)
	}
	var out []string
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		out = append(out, fmt.Sprint(m["subject_type"], ":", m["subject_id"]))
	}
	slices.Sort(out)
	return out
}

func TestV1WallsOnUnrenderedWorkHide(t *testing.T) {
	f := newWallFix(t)
	if err := f.db.Exec(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, rcQuizOpen, rcGalgame).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	walls := []struct {
		typ    string
		id     int
		anchor string
	}{
		{"galgame_rating", rcRating, resAnchor("rating", rcRating)},
		{"galgame_resource", rcResource, resAnchor("resource", rcResource)},
		{"galgame_quiz", rcQuizOpen, resAnchor("quiz", rcQuizOpen)},
	}
	var posts []int64
	for _, w := range walls {
		posts = append(posts, f.cm.seed(anchorRes, w.anchor, rcAlice, "on "+w.typ, 0, 0, now))
		if resp, body := f.wallOp(t, http.MethodPut, "rc-alice", w.typ, w.id, "/follow"); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow %s: %d %+v", w.typ, resp.StatusCode, body)
		}
	}
	toolsetPost := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcAlice, "toolset", 0, 0, now)
	f.wallOp(t, http.MethodPut, "rc-alice", "toolset", rcToolsetA, "/follow")
	f.cm.mu.Lock()
	f.cm.anchorSubs[rcAlice][anchorKey(anchorGame, strconv.Itoa(rcGalgameMissing))] = communityclient.NotificationWatching
	f.cm.mu.Unlock()

	every := []string{
		"galgame_quiz:" + strconv.Itoa(rcQuizOpen),
		"galgame_rating:" + strconv.Itoa(rcRating),
		"galgame_resource:" + strconv.Itoa(rcResource),
		"toolset:" + strconv.Itoa(rcToolsetA),
	}
	if got := f.followedKeys(t, "rc-alice"); fmt.Sprint(got) != fmt.Sprint(every) {
		t.Fatalf("followed %v, want %v (a galgame wall catalog does not show stays out)", got, every)
	}

	f.setRendered(t, rcGalgame, false)
	for i, w := range walls {
		resp, body := f.list(t, "rc-alice", w.typ, w.id, "")
		if resp.StatusCode != http.StatusNotFound || wallCode(body) != problem.CodeNotFound {
			t.Errorf("list %s on a hidden work: %d %+v", w.typ, resp.StatusCode, body)
		}
		resp, body = f.onComment(t, http.MethodGet, "rc-alice", posts[i], "", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("get comment on %s of a hidden work: %d %+v", w.typ, resp.StatusCode, body)
		}
		resp, body = f.wallOp(t, http.MethodGet, "rc-alice", w.typ, w.id, "")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("wall state %s of a hidden work: %d %+v", w.typ, resp.StatusCode, body)
		}
	}
	rendered, p := f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcAlice), f.authoredOf(append(posts, toolsetPost)...))
	if p != nil {
		t.Fatalf("render %v", p)
	}
	if len(rendered) != 1 || string(rendered[0].ID) != pid(toolsetPost) {
		t.Errorf("authored on a hidden work rendered %+v, want only the toolset comment", rendered)
	}
	if got := f.followedKeys(t, "rc-alice"); fmt.Sprint(got) != fmt.Sprint([]string{"toolset:" + strconv.Itoa(rcToolsetA)}) {
		t.Errorf("followed while hidden %v", got)
	}
	f.cm.mu.Lock()
	level := f.cm.anchorSubs[rcAlice][anchorKey(anchorRes, resAnchor("resource", rcResource))]
	f.cm.mu.Unlock()
	if level != communityclient.NotificationWatching {
		t.Errorf("hiding unfollowed the wall: level %d", level)
	}

	f.setRendered(t, rcGalgame, true)
	for _, w := range walls {
		if resp, body := f.list(t, "rc-alice", w.typ, w.id, ""); resp.StatusCode != http.StatusOK || len(itemIDs(t, body)) != 1 {
			t.Errorf("list %s restored: %d %+v", w.typ, resp.StatusCode, body)
		}
	}
	rendered, p = f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcAlice), f.authoredOf(append(posts, toolsetPost)...))
	if p != nil || len(rendered) != 4 {
		t.Errorf("authored restored: %d %v", len(rendered), p)
	}
	if got := f.followedKeys(t, "rc-alice"); fmt.Sprint(got) != fmt.Sprint(every) {
		t.Errorf("followed restored %v", got)
	}
}
