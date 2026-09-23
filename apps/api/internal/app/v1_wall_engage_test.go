package app

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

func TestV1WallLikeIsASlot(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBob, "likeable", 0, 0, time.Now())

	for i := 0; i < 2; i++ {
		resp, out := f.onComment(t, http.MethodPut, "rc-alice", id, "/like", nil)
		if resp.StatusCode != http.StatusOK || asInt(out["like_count"]) != 1 || wallViewer(t, out)["has_liked"] != true {
			t.Fatalf("PUT #%d %d %v", i, resp.StatusCode, out)
		}
	}
	awards := f.awardsSnapshot()
	if len(awards) != 1 || awards[0].userID != rcBob || awards[0].delta != 1 || awards[0].reason != "liked" {
		t.Fatalf("awards after two PUTs %+v", awards)
	}
	for i := 0; i < 2; i++ {
		resp, out := f.onComment(t, http.MethodDelete, "rc-alice", id, "/like", nil)
		if resp.StatusCode != http.StatusOK || asInt(out["like_count"]) != 0 || wallViewer(t, out)["has_liked"] != false {
			t.Fatalf("DELETE #%d %d %v", i, resp.StatusCode, out)
		}
	}
	awards = f.awardsSnapshot()
	if len(awards) != 2 || awards[1].delta != -1 {
		t.Fatalf("awards after two DELETEs %+v", awards)
	}
	if f.cm.reactions[[2]int64{id, rcAlice}] {
		t.Fatal("upstream still holds the like")
	}
}

func TestV1WallLikeRepairsADriftedMirror(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBob, "drift", 0, 0, time.Now())
	f.cm.mu.Lock()
	f.cm.reactions[[2]int64{id, rcAlice}] = true
	f.cm.mu.Unlock()

	resp, out := f.onComment(t, http.MethodPut, "rc-alice", id, "/like", nil)
	if resp.StatusCode != http.StatusOK || wallViewer(t, out)["has_liked"] != true {
		t.Fatalf("PUT %d %v", resp.StatusCode, out)
	}
	if !f.cm.reactions[[2]int64{id, rcAlice}] {
		t.Fatal("the upstream like was toggled away")
	}
	if len(f.awardsSnapshot()) != 0 {
		t.Fatal("a like that already existed upstream earned a second award")
	}
}

func TestV1WallLikeRefuses(t *testing.T) {
	f := newWallFix(t)
	own := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcAlice, "mine", 0, 0, time.Now())
	gone := f.cm.seed(anchorGame, strconv.Itoa(rcGalgame), rcBob, "x", communityclient.PostDeleted, 0, time.Now())
	resp, out := f.onComment(t, http.MethodPut, "rc-alice", own, "/like", nil)
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "SELF_LIKE_FORBIDDEN" {
		t.Fatalf("self like %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPut, "rc-alice", gone, "/like", nil)
	if resp.StatusCode != http.StatusConflict || wallCode(out) != "INVALID_STATE_TRANSITION" {
		t.Fatalf("like a tombstone %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodDelete, "rc-alice", gone, "/like", nil)
	if resp.StatusCode != http.StatusConflict || wallCode(out) != "INVALID_STATE_TRANSITION" {
		t.Fatalf("unlike a tombstone %d %v", resp.StatusCode, out)
	}
}

func TestV1WallFlag(t *testing.T) {
	f := newWallFix(t)
	id := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcBob, "spam", 0, 0, time.Now())
	gone := f.cm.seed(anchorRes, resAnchor("toolset", rcToolsetA), rcBob, "x", communityclient.PostDeleted, 0, time.Now())

	resp, out := f.onComment(t, http.MethodPost, "rc-alice", id, "/flags", map[string]any{"flag_reason": "off_topic", "note": "n"})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("flag %d %v", resp.StatusCode, out)
	}
	if len(f.cm.flags) != 1 || f.cm.flags[0].reason != communityclient.FlagReasonOffTopic || f.cm.flags[0].note != "n" || f.cm.flags[0].flagger != rcAlice {
		t.Fatalf("flags %+v", f.cm.flags)
	}
	resp, out = f.onComment(t, http.MethodPost, "rc-bob", id, "/flags", map[string]any{"flag_reason": "spam"})
	if resp.StatusCode != http.StatusForbidden || wallCode(out) != "PERMISSION_REQUIRED" {
		t.Fatalf("flag own %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPost, "rc-alice", gone, "/flags", map[string]any{"flag_reason": "spam"})
	if resp.StatusCode != http.StatusConflict || wallCode(out) != "INVALID_STATE_TRANSITION" {
		t.Fatalf("flag a tombstone %d %v", resp.StatusCode, out)
	}
	resp, out = f.onComment(t, http.MethodPost, "rc-alice", id, "/flags", map[string]any{"flag_reason": "rude"})
	if resp.StatusCode != http.StatusUnprocessableEntity || wallCode(out) != "VALIDATION_FAILED" {
		t.Fatalf("unknown reason %d %v", resp.StatusCode, out)
	}
}
