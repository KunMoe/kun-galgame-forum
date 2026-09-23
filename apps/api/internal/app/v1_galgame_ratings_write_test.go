package app

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
)

const (
	grKey1 = "01J9ZQ3V8K5N2M4P6R8T0V2X4Y"
	grKey2 = "01J9ZQ3V8K5N2M4P6R8T0V2X4Z"
	grKey3 = "01J9ZQ3V8K5N2M4P6R8T0V2X50"
	grKey4 = "01J9ZQ3V8K5N2M4P6R8T0V2X51"
)

func grCreateBody(workID int, summary string) map[string]any {
	return map[string]any{
		"work_id": strconv.Itoa(workID), "recommend": "strong_yes", "overall": 9,
		"game_types": []string{"plot", "moe"}, "play_status": "done_main", "spoiler_level": "portion",
		"short_summary": summary,
		"aspect_scores": map[string]any{"art": 8, "story": 10, "music": nil, "character": nil, "route": 6, "system": nil, "voice": nil, "replay_value": nil},
	}
}

func (f *grFix) create(t *testing.T, session, idem string, body map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodPost, "/api/v1/ratings", session, "/ratings", idem, body)
}

func grRatingRef(id int) string { return moemoepoint.Ref("galgame_rating", id) }

func TestV1RatingsCreate(t *testing.T) {
	f := newGRFix(t)
	body := grCreateBody(grWorkFresh, "short and sweet")

	resp, out := f.create(t, "gr-alice", "", body)
	if resp.StatusCode != http.StatusBadRequest || grCode(out) != "INVALID_PARAMETER" {
		t.Fatalf("no Idempotency-Key: %d %v", resp.StatusCode, out)
	}

	resp, out = f.create(t, "gr-alice", grKey1, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %v", resp.StatusCode, out)
	}
	id, _ := strconv.Atoi(out["id"].(string))
	if loc := resp.Header.Get("Location"); loc != "/api/v1/ratings/"+out["id"].(string) {
		t.Errorf("Location %q", loc)
	}
	if out["work"].(map[string]any)["id"] != strconv.Itoa(grWorkFresh) || out["overall"].(float64) != 9 {
		t.Errorf("created %v", out)
	}
	if s := out["aspect_scores"].(map[string]any); s["story"] != 10.0 || s["music"] != nil {
		t.Errorf("aspect_scores %v", s)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame WHERE id = ?`, grWorkFresh); n != 1 {
		t.Errorf("the work's local row was not created: %d", n)
	}
	wantAward := awardCall{grAlice, constants.RatingRewardLow, moemoepoint.ReasonContentApproved, grRatingRef(id), ""}
	if got := f.takeAwards(); len(got) != 1 || got[0].userID != wantAward.userID || got[0].delta != wantAward.delta ||
		got[0].reason != wantAward.reason || got[0].ref != wantAward.ref {
		t.Errorf("awards %v, want %v", got, wantAward)
	}
	if got := f.takeSyncs(); !slices.Equal(got, []grSync{{grWorkFresh, "access-gr-alice", "done_main"}}) {
		t.Errorf("syncs %v", got)
	}
	if scan := f.nextScan(t); scan.SubjectKind != gate.SubjectKindGalgameRating || scan.SubjectID != strconv.Itoa(id) || scan.Text != "short and sweet" {
		t.Errorf("scan %+v", scan)
	}

	resp, again := f.create(t, "gr-alice", grKey1, body)
	if resp.StatusCode != http.StatusCreated || again["id"] != out["id"] || len(f.takeAwards()) != 0 {
		t.Errorf("replayed create: %d %v", resp.StatusCode, again)
	}
	resp, out = f.create(t, "gr-alice", grKey2, body)
	if resp.StatusCode != http.StatusConflict || grCode(out) != "ALREADY_EXISTS" {
		t.Errorf("second rating of the work: %d %v", resp.StatusCode, out)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE user_id = ? AND work_id = ?`, grAlice, grWorkFresh); n != 1 {
		t.Errorf("%d ratings of one work by one user", n)
	}

	resp, out = f.create(t, "gr-bob", grKey1, grCreateBody(grWorkNone, ""))
	if p, r := grFieldReason(out); resp.StatusCode != http.StatusUnprocessableEntity || p != "/work_id" || r != "UNKNOWN_REFERENCE" {
		t.Errorf("unknown work: %d %v", resp.StatusCode, out)
	}
	resp, out = f.create(t, "gr-bob", grKey2, grCreateBody(grWorkFresh, "a forbidden word"))
	if resp.StatusCode != http.StatusUnprocessableEntity || grCode(out) != "CONTENT_REJECTED" {
		t.Errorf("refused summary: %d %v", resp.StatusCode, out)
	}
	resp, out = f.create(t, "gr-banned", grKey1, grCreateBody(grWorkFresh, ""))
	if resp.StatusCode != http.StatusForbidden || grCode(out) != "ACCOUNT_BANNED" {
		t.Errorf("banned author: %d %v", resp.StatusCode, out)
	}

	bad := grCreateBody(grWorkFresh, "")
	bad["game_types"] = []string{"plot", "plot"}
	if resp, _ := f.create(t, "gr-bob", grKey3, bad); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("duplicate game type: %d", resp.StatusCode)
	}
	bad = grCreateBody(grWorkFresh, "")
	delete(bad["aspect_scores"].(map[string]any), "voice")
	if resp, _ := f.create(t, "gr-bob", grKey4, bad); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("aspect_scores without voice: %d", resp.StatusCode)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE user_id IN (?, ?)`, grBob, grBanned); n != 1 {
		t.Errorf("refused creates wrote rows: %d (the banned user's seeded rating is the one)", n)
	}

	f.failOA.Store(true)
	f.putSession(t, "gr-fresh", grBob+900, "user")
	resp, out = f.create(t, "gr-fresh", grKey1, grCreateBody(grWorkFresh, ""))
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("OAuth down: %d %v", resp.StatusCode, out)
	}
}

func TestV1RatingsUpdate(t *testing.T) {
	f := newGRFix(t)
	patch := func(session string, id int, body map[string]any) (*http.Response, map[string]any) {
		t.Helper()
		return f.onRating(t, http.MethodPatch, session, id, "", body)
	}

	resp, out := patch("gr-bob", grRatingAlice, map[string]any{"overall": 1})
	if resp.StatusCode != http.StatusForbidden || grCode(out) != "PERMISSION_REQUIRED" {
		t.Errorf("someone else's rating: %d %v", resp.StatusCode, out)
	}
	resp, out = patch("gr-alice", grRatingHidden, map[string]any{"overall": 1})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("rating of a hidden work: %d %v", resp.StatusCode, out)
	}
	resp, out = patch("gr-banned", grRatingBanned, map[string]any{"overall": 1})
	if resp.StatusCode != http.StatusForbidden || grCode(out) != "ACCOUNT_BANNED" {
		t.Errorf("banned author: %d %v", resp.StatusCode, out)
	}

	checks := f.checker.calls.Load()
	resp, out = patch("gr-alice", grRatingAlice, map[string]any{"overall": 2, "short_summary": "short"})
	if resp.StatusCode != http.StatusOK || out["overall"].(float64) != 2 || out["recommend"] != "yes" || out["play_status"] != "done_all" {
		t.Fatalf("partial patch: %d %v", resp.StatusCode, out)
	}
	if f.checker.calls.Load() != checks || len(f.takeAwards()) != 0 || len(f.takeSyncs()) != 0 {
		t.Errorf("an unchanged summary was checked, paid or synced")
	}

	long := strings.Repeat("a", constants.RatingLenThresholdHigh)
	resp, out = patch("gr-alice", grRatingAlice, map[string]any{"short_summary": long, "play_status": "dropped"})
	if resp.StatusCode != http.StatusOK || out["short_summary"] != long {
		t.Fatalf("summary patch: %d %v", resp.StatusCode, out)
	}
	if f.checker.calls.Load() != checks+1 {
		t.Errorf("a changed summary was not checked")
	}
	if scan := f.nextScan(t); scan.SubjectID != strconv.Itoa(grRatingAlice) || scan.Text != long {
		t.Errorf("scan after edit %+v", scan)
	}
	awards := f.takeAwards()
	if len(awards) != 1 || awards[0].userID != grAlice || awards[0].delta != constants.RatingRewardHigh-constants.RatingRewardLow ||
		awards[0].reason != moemoepoint.ReasonContentApproved {
		t.Errorf("awards %v", awards)
	}
	if got := f.takeSyncs(); !slices.Equal(got, []grSync{{grWorkSFW, "access-gr-alice", "dropped"}}) {
		t.Errorf("syncs %v", got)
	}

	resp, out = patch("gr-alice", grRatingAlice, map[string]any{"aspect_scores": map[string]any{
		"art": nil, "story": nil, "music": nil, "character": nil, "route": nil, "system": nil, "voice": nil, "replay_value": 4,
	}})
	if s := out["aspect_scores"].(map[string]any); resp.StatusCode != http.StatusOK || s["art"] != nil || s["replay_value"] != 4.0 {
		t.Errorf("aspect_scores replace: %d %v", resp.StatusCode, out)
	}
	if n := f.count(t, `SELECT art FROM galgame_rating WHERE id = ?`, grRatingAlice); n != 0 {
		t.Errorf("null art stored as %d", n)
	}

	resp, out = patch("gr-alice", grRatingAlice, map[string]any{"short_summary": "forbidden"})
	if resp.StatusCode != http.StatusUnprocessableEntity || grCode(out) != "CONTENT_REJECTED" {
		t.Errorf("refused summary: %d %v", resp.StatusCode, out)
	}
}

func TestV1RatingsDelete(t *testing.T) {
	f := newGRFix(t)
	del := func(session string, id int) (*http.Response, map[string]any) {
		t.Helper()
		return f.onRating(t, http.MethodDelete, session, id, "", nil)
	}
	onSFW, onSFW2, alsoSFW := grRatingRaters, grRatingRaters+1, grRatingRaters+3
	for _, c := range []struct {
		session string
		id      int
	}{{"gr-bob", grRatingAlice}, {"gr-creator", onSFW2}, {"bearer:gr-staff-token", alsoSFW}} {
		resp, out := del(c.session, c.id)
		if resp.StatusCode != http.StatusForbidden || grCode(out) != "PERMISSION_REQUIRED" {
			t.Errorf("%s deleting %d: %d %v", c.session, c.id, resp.StatusCode, out)
		}
	}
	if len(f.takeAwards()) != 0 {
		t.Fatal("a refused delete moved points")
	}

	for _, c := range []struct {
		session string
		id      int
		author  int
	}{{"gr-creator", onSFW, grRater}, {"gr-staff", alsoSFW, grRater + 3}, {"gr-alice", grRatingAlice, grAlice}} {
		resp, out := del(c.session, c.id)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s deleting %d: %d %v", c.session, c.id, resp.StatusCode, out)
		}
		awards := f.takeAwards()
		if len(awards) != 1 || awards[0].userID != c.author || awards[0].delta != -constants.RatingRewardLow ||
			awards[0].reason != moemoepoint.ReasonContentRemoved || awards[0].ref != grRatingRef(c.id) {
			t.Errorf("deleting %d: awards %v", c.id, awards)
		}
		if resp, _ := f.onRating(t, http.MethodGet, "", c.id, "", nil); resp.StatusCode != http.StatusNotFound {
			t.Errorf("deleted rating %d still reads %d", c.id, resp.StatusCode)
		}
	}
	if resp, _ := del("gr-alice", grRatingAlice); resp.StatusCode != http.StatusNotFound {
		t.Errorf("deleting twice: %d", resp.StatusCode)
	}
}

func TestV1RatingsLike(t *testing.T) {
	f := newGRFix(t)
	like := func(method, session string, id int) (*http.Response, map[string]any) {
		t.Helper()
		return f.onRating(t, method, session, id, "/like", nil)
	}

	var likeID int
	for i := range 2 {
		resp, out := like(http.MethodPut, "gr-bob", grRatingAlice)
		if resp.StatusCode != http.StatusOK || out["like_count"].(float64) != 1 || out["viewer"].(map[string]any)["has_liked"] != true {
			t.Fatalf("like #%d: %d %v", i+1, resp.StatusCode, out)
		}
		awards := f.takeAwards()
		if i == 0 {
			likeID = f.count(t, `SELECT id FROM galgame_rating_like WHERE galgame_rating_id = ? AND user_id = ?`, grRatingAlice, grBob)
			want := awardCall{grAlice, 1, moemoepoint.ReasonLiked, grRatingRef(grRatingAlice), moemoepoint.Key("liked", "galgame_rating_like_"+strconv.Itoa(likeID))}
			if !slices.Equal(awards, []awardCall{want}) {
				t.Errorf("first like: awards %v, want %v", awards, want)
			}
		} else if len(awards) != 0 {
			t.Errorf("replayed like moved points: %v", awards)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked'`, grBob, grAlice); n != 1 {
		t.Errorf("%d like notifications, want 1", n)
	}
	_, out := f.onRating(t, http.MethodGet, "gr-bob", grRatingAlice, "", nil)
	if out["viewer"].(map[string]any)["has_liked"] != true || len(out["likers"].([]any)) != 1 {
		t.Errorf("detail after like: %v", out)
	}

	for i := range 2 {
		resp, out := like(http.MethodDelete, "gr-bob", grRatingAlice)
		if resp.StatusCode != http.StatusOK || out["like_count"].(float64) != 0 || out["viewer"].(map[string]any)["has_liked"] != false {
			t.Fatalf("unlike #%d: %d %v", i+1, resp.StatusCode, out)
		}
		awards := f.takeAwards()
		if i == 0 {
			want := awardCall{grAlice, -1, moemoepoint.ReasonLiked, grRatingRef(grRatingAlice), moemoepoint.Key("unliked", "galgame_rating_like_"+strconv.Itoa(likeID))}
			if !slices.Equal(awards, []awardCall{want}) {
				t.Errorf("first unlike: awards %v, want %v", awards, want)
			}
		} else if len(awards) != 0 {
			t.Errorf("replayed unlike moved points: %v", awards)
		}
	}

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		resp, out := like(method, "gr-alice", grRatingAlice)
		if resp.StatusCode != http.StatusForbidden || grCode(out) != "SELF_LIKE_FORBIDDEN" {
			t.Errorf("%s own like: %d %v", method, resp.StatusCode, out)
		}
	}
	for _, id := range []int{grRatingHidden, grRatingBanned} {
		if resp, _ := like(http.MethodPut, "gr-bob", id); resp.StatusCode != http.StatusNotFound {
			t.Errorf("liking hidden rating %d: %d", id, resp.StatusCode)
		}
	}
	if resp, out := like(http.MethodPut, "gr-banned", grRatingAlice); resp.StatusCode != http.StatusForbidden || grCode(out) != "ACCOUNT_BANNED" {
		t.Errorf("banned liker: %d %v", resp.StatusCode, out)
	}
	if len(f.takeAwards()) != 0 {
		t.Error("a refused like moved points")
	}
}
