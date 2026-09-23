package app

import (
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

func (f *wallFix) authoredOf(ids ...int64) []communityclient.AuthorPostView {
	out := make([]communityclient.AuthorPostView, 0, len(ids))
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	for _, id := range ids {
		p := f.cm.posts[id]
		out = append(out, communityclient.AuthorPostView{
			Post: p.PostView,
			Thread: communityclient.PostThreadContext{
				ThreadID: p.ThreadID, AnchorKind: p.anchorKind, AnchorID: p.anchorID,
			},
		})
	}
	return out
}

func rcViewer(id int, roles ...string) *middleware.UserInfo {
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	return &middleware.UserInfo{ID: id, Name: "n", Roles: roles}
}

func TestV1WallRenderAuthoredOnePerSubject(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	ids := []int64{
		f.cm.seed(anchorGame, strconvI(rcGalgame), rcAlice, "g", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("rating", rcRating), rcAlice, "r", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("resource", rcResource), rcAlice, "s", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("quiz", rcQuizOpen), rcAlice, "q", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcAlice, "t", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("website", rcWebsite), rcAlice, "w", 0, 0, now),
	}
	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, f.authoredOf(ids...))
	if p != nil {
		t.Fatalf("render %v", p)
	}
	if len(got) != 6 {
		t.Fatalf("got %d comments, want 6", len(got))
	}
	want := []string{"galgame", "galgame_rating", "galgame_resource", "galgame_quiz", "toolset", "website"}
	for i, item := range got {
		if string(item.SubjectType) != want[i] || string(item.ID) != pid(ids[i]) {
			t.Errorf("item %d type=%s id=%s, want %s %s", i, item.SubjectType, item.ID, want[i], pid(ids[i]))
		}
	}
}

func TestV1WallRenderAuthoredDropsSpoilerQuizForStrangers(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorRes, resAnchor("quiz", rcQuizSpoiler), rcBob, "spoiler talk", 0, 0, time.Now())
	rows := f.authoredOf(id)

	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, rows)
	if p != nil || len(got) != 0 {
		t.Fatalf("anonymous: %d %v", len(got), p)
	}
	got, p = f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcAlice), rows)
	if p != nil || len(got) != 0 {
		t.Fatalf("stranger: %d %v", len(got), p)
	}
	got, p = f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcBob), rows)
	if p != nil || len(got) != 1 || string(got[0].ID) != pid(id) {
		t.Fatalf("author: %d %v %+v", len(got), p, got)
	}
	got, p = f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcCarol), rows)
	if p != nil || len(got) != 1 || string(got[0].ID) != pid(id) {
		t.Fatalf("answered: %d %v %+v", len(got), p, got)
	}
}

func TestV1WallRenderAuthoredHeldOnlyForAuthor(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorGame, strconvI(rcGalgame), rcAlice, "held", communityclient.PostHeld, 0, time.Now())
	rows := f.authoredOf(id)

	got, p := f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcAlice), rows)
	if p != nil || len(got) != 1 || got[0].State != "held" {
		t.Fatalf("author held: %d %v %+v", len(got), p, got)
	}
	got, p = f.app.WallV1.RenderAuthored(t.Context(), rcViewer(rcBob), rows)
	if p != nil || len(got) != 0 {
		t.Fatalf("other held: %d %v", len(got), p)
	}
	got, p = f.app.WallV1.RenderAuthored(t.Context(), nil, rows)
	if p != nil || len(got) != 0 {
		t.Fatalf("anon held: %d %v", len(got), p)
	}
}

func TestV1WallRenderAuthoredDropsUnknownAnchor(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(0, "board-1", rcAlice, "board post", 0, 0, time.Now())
	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, f.authoredOf(id))
	if p != nil || len(got) != 0 {
		t.Fatalf("unknown anchor: %d %v", len(got), p)
	}
}

func TestV1WallRenderAuthoredUnavailableWallFails(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorGame, strconvI(rcGalgame), rcAlice, "g", 0, 0, time.Now())
	f.failGC.Store(true)
	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, f.authoredOf(id))
	if p == nil || p.Code != problem.CodeServiceUnavailable {
		t.Fatalf("catalog down: got %d items, problem %+v; want SERVICE_UNAVAILABLE", len(got), p)
	}
}

func TestV1WallRenderAuthoredChecksGalgameWallsInOneBatch(t *testing.T) {
	f := newWallFix(t)
	now := time.Now()
	ids := []int64{
		f.cm.seed(anchorGame, strconvI(rcGalgame), rcAlice, "g1", 0, 0, now),
		f.cm.seed(anchorRes, resAnchor("rating", rcRating), rcAlice, "r", 0, 0, now),
		f.cm.seed(anchorGame, strconvI(rcGalgame+1), rcAlice, "gone1", 0, 0, now),
		f.cm.seed(anchorGame, strconvI(rcGalgame), rcAlice, "g2", 0, 0, now),
		f.cm.seed(anchorGame, strconvI(rcGalgame+2), rcAlice, "gone2", 0, 0, now),
	}
	f.resolveCalls.Store(0)
	f.existCalls.Store(0)
	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, f.authoredOf(ids...))
	if p != nil {
		t.Fatalf("render %v", p)
	}
	if n, d := f.existCalls.Load(), f.resolveCalls.Load(); n != 1 || d != 0 {
		t.Fatalf("three galgame walls: %d batch calls and %d detail calls, want 1 and 0", n, d)
	}
	want := []string{pid(ids[0]), pid(ids[1]), pid(ids[3])}
	if len(got) != len(want) {
		t.Fatalf("got %d comments, want %d (missing works dropped)", len(got), len(want))
	}
	for i, item := range got {
		if string(item.ID) != want[i] {
			t.Errorf("item %d id=%s, want %s (input order)", i, item.ID, want[i])
		}
	}
}

func TestV1WallRenderAuthoredUnavailableOwnerFails(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorRes, resAnchor("rating", rcRating), rcAlice, "r", 0, 0, time.Now())
	f.failOA.Store(true)
	got, p := f.app.WallV1.RenderAuthored(t.Context(), nil, f.authoredOf(id))
	if p == nil || p.Code != problem.CodeServiceUnavailable {
		t.Fatalf("account service down: got %d items, problem %+v; want SERVICE_UNAVAILABLE", len(got), p)
	}
}
