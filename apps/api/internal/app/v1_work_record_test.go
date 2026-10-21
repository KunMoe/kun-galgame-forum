package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/model"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
)

func (f *grFix) record(t *testing.T, method, session string, workID int, body any) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, method, "/api/v1/works/"+strconv.Itoa(workID)+"/record", session, "/works/{work_id}/record", "", body)
}

func TestV1WorkRecord(t *testing.T) {
	f := newGRFix(t)

	resp, out := f.record(t, http.MethodGet, "gr-bob", grWorkFresh, nil)
	if resp.StatusCode != http.StatusOK || out["play_state"] != nil || out["minutes"].(float64) != 0 || out["rating_summary"] != nil {
		t.Fatalf("empty record: %d %v", resp.StatusCode, out)
	}

	full := map[string]any{
		"play_state": "done_main",
		"minutes":    600,
		"rating_input": map[string]any{
			"recommend": "yes", "overall": 8, "game_types": []string{"plot"}, "spoiler_level": "none",
			"short_summary": "good",
		},
	}
	resp, out = f.record(t, http.MethodPut, "gr-bob", grWorkFresh, full)
	if resp.StatusCode != http.StatusOK || out["play_state"] != "done_main" || out["minutes"].(float64) != 600 {
		t.Fatalf("put record: %d %v", resp.StatusCode, out)
	}
	rating, _ := out["rating_summary"].(map[string]any)
	if rating == nil || rating["overall"].(float64) != 8 || rating["play_status"] != "done_main" ||
		len(rating["game_types"].([]any)) != 1 {
		t.Fatalf("rating in the record: %v", rating)
	}
	id, _ := strconv.Atoi(rating["id"].(string))
	ref := moemoepoint.Ref("galgame_rating", id)
	if got := f.takeAwards(); len(got) != 1 || got[0].delta != constants.RatingRewardLow || got[0].ref != ref ||
		got[0].reason != moemoepoint.ReasonContentApproved {
		t.Errorf("first rating awards %v", got)
	}
	if scan := f.nextScan(t); scan.SubjectID != strconv.Itoa(id) || scan.Text != "good" {
		t.Errorf("scan %+v", scan)
	}
	if f.recordWrites.Load() != 1 {
		t.Errorf("the playtime cache was not dropped")
	}

	checks := f.checker.calls.Load()
	resp, out = f.record(t, http.MethodPut, "gr-bob", grWorkFresh, full)
	if resp.StatusCode != http.StatusOK || f.checker.calls.Load() != checks || len(f.takeAwards()) != 0 {
		t.Errorf("an unchanged record was checked or paid: %d %v", resp.StatusCode, out)
	}

	refused := map[string]any{"rating_input": map[string]any{
		"recommend": "yes", "overall": 8, "game_types": []string{"plot"}, "spoiler_level": "none", "short_summary": "forbidden",
	}}
	if resp, out := f.record(t, http.MethodPut, "gr-bob", grWorkFresh, refused); resp.StatusCode != http.StatusUnprocessableEntity ||
		grCode(out) != "CONTENT_REJECTED" {
		t.Errorf("refused summary: %d %v", resp.StatusCode, out)
	}

	resp, out = f.record(t, http.MethodPut, "gr-bob", grWorkFresh, map[string]any{"play_state": "dropped"})
	if resp.StatusCode != http.StatusOK || out["play_state"] != "dropped" || out["minutes"].(float64) != 0 || out["rating_summary"] != nil {
		t.Fatalf("state only: %d %v", resp.StatusCode, out)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, id); n != 0 {
		t.Errorf("the dropped rating is still mirrored")
	}
	if got := f.takeAwards(); len(got) != 1 || got[0].delta != -constants.RatingRewardLow || got[0].ref != ref ||
		got[0].reason != moemoepoint.ReasonContentRemoved {
		t.Errorf("deleting the rating: awards %v", got)
	}

	if resp, _ := f.record(t, http.MethodPut, "gr-bob", grWorkNone, map[string]any{}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("a work catalog does not show: %d", resp.StatusCode)
	}
	if resp, out := f.record(t, http.MethodGet, "", grWorkFresh, nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d %v", resp.StatusCode, out)
	}
}

func TestV1WorkRecordKeepsAnUnstatedDone(t *testing.T) {
	f := newGRFix(t)
	f.plane.mu.Lock()
	f.plane.states[grKey{grBob, grWorkFresh}] = catalogclient.WorkStateRecord{WorkID: grWorkFresh, State: "done"}
	f.plane.mu.Unlock()

	resp, out := f.record(t, http.MethodGet, "gr-bob", grWorkFresh, nil)
	if resp.StatusCode != http.StatusOK || out["play_state"] != "done" {
		t.Fatalf("read: %d %v", resp.StatusCode, out)
	}

	rating := map[string]any{"recommend": "yes", "overall": 8, "game_types": []string{"plot"}, "spoiler_level": "none"}
	for _, tc := range []struct{ sent, wire string }{
		{"done", `{"state":"done"}`},
		{"done_one_route", `{"state":"done","completion":"one_route"}`},
	} {
		resp, out = f.record(t, http.MethodPut, "gr-bob", grWorkFresh, map[string]any{"play_state": tc.sent, "rating_input": rating})
		if resp.StatusCode != http.StatusOK || out["play_state"] != tc.sent {
			t.Fatalf("put %s: %d %v", tc.sent, resp.StatusCode, out)
		}
		f.plane.mu.Lock()
		wire, err := json.Marshal(f.plane.lastRecord.WorkState)
		f.plane.mu.Unlock()
		if err != nil || string(wire) != tc.wire {
			t.Errorf("%s reached NextMoe as work_state %s, want %s", tc.sent, wire, tc.wire)
		}
	}
}

type grFeed struct {
	pages []*catalogclient.RatingChangePage
}

func (f *grFeed) RatingChanges(_ context.Context, cursor string) (*catalogclient.RatingChangePage, error) {
	i, _ := strconv.Atoi(cursor)
	if i >= len(f.pages) {
		return &catalogclient.RatingChangePage{}, nil
	}
	return f.pages[i], nil
}

func TestRatingMirror(t *testing.T) {
	f := newGRFix(t)
	store := galgameRepo.NewRatingStore(f.db)
	at := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	rating := func(id, work, user int, client, summary string) *catalogclient.Rating {
		return &catalogclient.Rating{
			ID: int64(id), WorkID: int64(work), UserID: int64(user), Overall: 6, Recommend: "neutral",
			SpoilerLevel: "none", ShortSummary: summary, ClientID: client, Site: "kungal", CreatedAt: at, UpdatedAt: at,
			WorkState: &catalogclient.WorkStateRecord{State: "done"},
		}
	}
	f.run(t, `UPDATE galgame_rating SET like_count = 3 WHERE id = ?`, grRatingAlice)

	replaced := grRatingRaters + 50
	foreign := grRatingRaters + 51
	unknownWork := grRatingRaters + 52
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{
		{NextCursor: "1", Items: []catalogclient.RatingChange{
			{ID: int64(grRatingAlice), Rating: rating(grRatingAlice, grWorkSFW, grAlice, grForumClient, "edited elsewhere")},
			{ID: int64(replaced), Rating: rating(replaced, grWorkSFW2, grGone, grForumClient, "")},
		}},
		{Items: []catalogclient.RatingChange{
			{ID: int64(foreign), Rating: rating(foreign, grWorkSFW, grBob, "letmoe-client", "from letmoe")},
			{ID: int64(unknownWork), Rating: rating(unknownWork, grWorkMax, grBob, grForumClient, "no local row")},
			{ID: int64(grRatingBanned), Gone: true},
		}},
	}}
	mirror := galgameService.NewRatingMirror(feed, store, f.rdb, []string{grForumClient})
	mirror.Run()

	got, err := store.Get(grRatingAlice)
	if err != nil || got.ShortSummary != "edited elsewhere" || got.LikeCount != 3 || got.AuthorState == nil || *got.AuthorState != "done" {
		t.Errorf("upsert kept or lost the wrong columns: %+v %v", got, err)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingGone); n != 0 {
		t.Errorf("the replaced rating of the same person and work is still there")
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingBanned); n != 0 {
		t.Errorf("a gone rating is still there")
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ? AND NOT written_here`, foreign); n != 1 {
		t.Errorf("another site's rating is missing or marked written here")
	}
	if n := f.count(t, `SELECT count(*) FROM feed_activity WHERE type = 'GALGAME_RATING_CREATION' AND source_id = ?`, foreign); n != 0 {
		t.Errorf("another site's rating reached the feed")
	}
	if n := f.count(t, `SELECT count(*) FROM feed_activity WHERE type = 'GALGAME_RATING_CREATION' AND source_id = ?`, replaced); n != 1 {
		t.Errorf("a rating written here did not reach the feed")
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, unknownWork); n != 1 {
		t.Errorf("a rating on a work without a local row was not mirrored")
	}
	if cursor, _ := f.rdb.Get(context.Background(), galgameService.RatingMirrorCursorKey).Result(); cursor != "1" {
		t.Errorf("cursor %q, want the last full page's", cursor)
	}

	var row model.GalgameRating
	if err := f.db.Raw(`SELECT id, written_here FROM galgame_rating WHERE id = ?`, grRatingAlice).Scan(&row).Error; err != nil || !row.WrittenHere {
		t.Errorf("a rating written here lost written_here: %+v %v", row, err)
	}

	queued := func() int {
		return f.count(t, `SELECT count(*) FROM activity_push_queue WHERE type = 'GALGAME_RATING_CREATION' AND source_id IN (?, ?)`,
			grRatingAlice, replaced)
	}
	f.run(t, `DELETE FROM activity_push_queue WHERE type = 'GALGAME_RATING_CREATION' AND source_id IN (?, ?)`, grRatingAlice, replaced)
	f.run(t, `UPDATE galgame_rating SET written_here = true WHERE id = ?`, replaced)
	f.rdb.Del(context.Background(), galgameService.RatingMirrorCursorKey)
	mirror.Run()
	if n := queued(); n != 0 {
		t.Errorf("replaying unchanged ratings re-pushed %d activities", n)
	}

	feed.pages[0].Items[0].Rating.WorkState = &catalogclient.WorkStateRecord{State: "dropped"}
	f.rdb.Del(context.Background(), galgameService.RatingMirrorCursorKey)
	mirror.Run()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ? AND author_state = 'dropped'`, grRatingAlice); n != 1 {
		t.Errorf("the author's new play state was not mirrored")
	}
	if n := queued(); n != 0 {
		t.Errorf("a play state change re-pushed %d activities", n)
	}
}

func fatesOf(byWork map[int64]string) func(context.Context, int64) (string, error) {
	return func(_ context.Context, workID int64) (string, error) { return byWork[workID], nil }
}

func liveRating(id, work, user int, at time.Time) *catalogclient.Rating {
	return &catalogclient.Rating{
		ID: int64(id), WorkID: int64(work), UserID: int64(user), Overall: 8, Recommend: "yes",
		SpoilerLevel: "none", ClientID: grForumClient, Site: "kungal", CreatedAt: at, UpdatedAt: at,
	}
}

func runMirror(t *testing.T, f *grFix, fate map[int64]string, items ...catalogclient.RatingChange) {
	t.Helper()
	f.rdb.Del(context.Background(), galgameService.RatingMirrorCursorKey)
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{Items: items}}}
	galgameService.NewRatingMirror(feed, galgameRepo.NewRatingStore(f.db), f.rdb, []string{grForumClient}).
		WithWorkFate(fatesOf(fate)).Run()
}

func (f *grFix) assertRestored(t *testing.T, id int) {
	t.Helper()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ? AND galgame_type::text LIKE '%plot%'`, id); n != 1 {
		t.Errorf("rating %d is not back with its game types", id)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ? AND like_count = 1`, id); n != 1 {
		t.Errorf("rating %d is not back with its like count", id)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_like WHERE galgame_rating_id = ?`, id); n != 1 {
		t.Errorf("rating %d is not back with its like", id)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, id); n != 0 {
		t.Errorf("rating %d is still held", id)
	}
}

func (f *grFix) assertHeld(t *testing.T, id int, reason string) {
	t.Helper()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, id); n != 0 {
		t.Errorf("rating %d is still readable", id)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ? AND reason = ? AND jsonb_array_length(likes) = 1`, id, reason); n != 1 {
		t.Errorf("rating %d is not held as %s with its like", id, reason)
	}
}

func TestRatingMirrorHoldsTheRatingsOfADeletedWork(t *testing.T) {
	f := newGRFix(t)
	f.run(t, `DELETE FROM galgame_rating_held WHERE id IN (?, ?)`, grRatingAlice, grRatingGone)
	f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, now())`, grRatingAlice, grBob)
	f.run(t, `UPDATE galgame_rating SET like_count = 1 WHERE id = ?`, grRatingAlice)
	fate := map[int64]string{grWorkSFW: "deleted"}

	runMirror(t, f, fate,
		catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
		catalogclient.RatingChange{ID: int64(grRatingGone), WorkID: grWorkSFW2, Gone: true})
	f.assertHeld(t, grRatingAlice, "deleted")
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingGone) +
		f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, grRatingGone); n != 0 {
		t.Errorf("a rating its author deleted on a live work was kept")
	}

	runMirror(t, f, nil, catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW,
		Rating: liveRating(grRatingAlice, grWorkSFW, grAlice, time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC))})
	f.assertRestored(t, grRatingAlice)
}

func TestRatingMirrorHoldsTheOlderRatingAMergeFolded(t *testing.T) {
	merged := grRatingRaters + 60
	at := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	for name, items := range map[string][]catalogclient.RatingChange{
		"on the merged work": {
			{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
			{ID: int64(merged), WorkID: grWorkFresh, Rating: liveRating(merged, grWorkFresh, grAlice, at)},
		},
		"on the survivor, gone first": {
			{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
			{ID: int64(merged), WorkID: grWorkSFW, Rating: liveRating(merged, grWorkSFW, grAlice, at)},
		},
		"on the survivor, merged rating first": {
			{ID: int64(merged), WorkID: grWorkSFW, Rating: liveRating(merged, grWorkSFW, grAlice, at)},
			{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newGRFix(t)
			f.run(t, `DELETE FROM galgame_rating_held WHERE id IN (?, ?)`, grRatingAlice, merged)
			t.Cleanup(func() { f.run(t, `DELETE FROM galgame_rating WHERE id = ?`, merged) })
			f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, now())`, grRatingAlice, grBob)
			f.run(t, `UPDATE galgame_rating SET like_count = 1 WHERE id = ?`, grRatingAlice)
			fate := map[int64]string{}
			if name == "on the merged work" {
				fate[grWorkSFW] = "merged"
			}

			runMirror(t, f, fate, items...)
			f.assertHeld(t, grRatingAlice, "merged")

			runMirror(t, f, nil,
				catalogclient.RatingChange{ID: int64(merged), WorkID: grWorkFresh, Rating: liveRating(merged, grWorkFresh, grAlice, at)},
				catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW, Rating: liveRating(grRatingAlice, grWorkSFW, grAlice, at)})
			f.assertRestored(t, grRatingAlice)
		})
	}
}

func TestRatingMirrorPurgesMergeHoldsOnceTheUndoWindowIsOver(t *testing.T) {
	f := newGRFix(t)
	ids := []int{grRatingRaters + 70, grRatingRaters + 71}
	t.Cleanup(func() { f.run(t, `DELETE FROM galgame_rating_held WHERE id IN ?`, ids) })
	f.run(t, `INSERT INTO galgame_rating_held (id, user_id, work_id, reason, row, held_at) VALUES
		(?, ?, ?, 'merged', '{}', now() - interval '40 days'), (?, ?, ?, 'deleted', '{}', now() - interval '40 days')`,
		ids[0], grAlice, grWorkSFW, ids[1], grAlice, grWorkSFW)

	runMirror(t, f, nil)
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, ids[0]); n != 0 {
		t.Errorf("a merge hold outlived the 30-day undo window")
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, ids[1]); n != 1 {
		t.Errorf("a deleted work's hold went: a deletion can be undone at any time")
	}
}

func TestRatingMirrorNeverHoldsAPurgedAuthorsRating(t *testing.T) {
	f := newGRFix(t)
	f.run(t, `DELETE FROM galgame_rating_held WHERE id = ?`, grRatingAlice)
	f.run(t, `INSERT INTO user_purge_archive (purge_id, operator_id, target_user_id, table_name, operation, row_pk, row_data)
		VALUES (gen_random_uuid(), 1, ?, 'galgame_rating', 'delete', '{}', '{}')`, grAlice)
	t.Cleanup(func() { f.run(t, `DELETE FROM user_purge_archive WHERE target_user_id = ?`, grAlice) })

	runMirror(t, f, map[int64]string{grWorkSFW: "deleted"},
		catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true})
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingAlice) +
		f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, grRatingAlice); n != 0 {
		t.Errorf("a purged author's rating was kept")
	}
}

func TestRatingMirrorHoldsTheCursorWhenItCannotTellWhyARatingWent(t *testing.T) {
	f := newGRFix(t)
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{Items: []catalogclient.RatingChange{
		{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
	}}}}
	down := func(context.Context, int64) (string, error) { return "", fmt.Errorf("catalog down") }
	galgameService.NewRatingMirror(feed, galgameRepo.NewRatingStore(f.db), f.rdb, []string{grForumClient}).WithWorkFate(down).Run()

	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingAlice); n != 1 {
		t.Errorf("the rating went while catalog could not say what happened to its work")
	}
}
