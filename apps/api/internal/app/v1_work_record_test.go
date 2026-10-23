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
	"kun-galgame-api/pkg/userclient"
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

type currentOf = func(context.Context, int64, int64) (int64, bool, error)

func runMirrorAsking(t *testing.T, f *grFix, current currentOf, items ...catalogclient.RatingChange) {
	t.Helper()
	f.rdb.Del(context.Background(), galgameService.RatingMirrorCursorKey)
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{Items: items}}}
	galgameService.NewRatingMirror(feed, galgameRepo.NewRatingStore(f.db), f.rdb, []string{grForumClient}).
		WithWorkFate(fatesOf(nil)).WithCurrentRating(current).Run()
}

func (f *grFix) parked(t *testing.T) map[int]int64 {
	t.Helper()
	got, err := galgameRepo.NewRatingStore(f.db).ParkedGoneRatings()
	if err != nil {
		t.Fatal(err)
	}
	return got
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

func TestRatingMirrorParksARatingItCannotSettleAndMovesOn(t *testing.T) {
	f := newGRFix(t)
	f.run(t, `DELETE FROM galgame_rating_held WHERE id IN (?, ?)`, grRatingAlice, grRatingGone)
	f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, now())`, grRatingAlice, grBob)
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{NextCursor: "1", Items: []catalogclient.RatingChange{
		{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
		{ID: int64(grRatingGone), WorkID: grWorkSFW2, Gone: true},
	}}}}
	down := true
	fate := func(_ context.Context, work int64) (string, error) {
		if work == grWorkSFW && down {
			return "", fmt.Errorf("catalog down")
		}
		if work == grWorkSFW {
			return galgameRepo.HeldDeleted, nil
		}
		return "", nil
	}
	mirror := galgameService.NewRatingMirror(feed, galgameRepo.NewRatingStore(f.db), f.rdb, []string{grForumClient}).WithWorkFate(fate)
	mirror.Run()

	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingAlice); n != 1 {
		t.Errorf("the rating went while catalog could not say what happened to its work")
	}
	if got := f.parked(t); len(got) != 1 || got[grRatingAlice] != grWorkSFW {
		t.Errorf("parked %v, want the one rating catalog could not answer about", got)
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingGone); n != 0 {
		t.Errorf("one work catalog could not answer about stopped the ratings behind it")
	}
	if cursor, _ := f.rdb.Get(context.Background(), galgameService.RatingMirrorCursorKey).Result(); cursor != "1" {
		t.Errorf("cursor %q, want it past the page", cursor)
	}

	mirror.Run()
	if len(f.parked(t)) != 1 {
		t.Errorf("a rating left the retry set while catalog was still down")
	}
	down = false
	mirror.Run()
	f.assertHeld(t, grRatingAlice, galgameRepo.HeldDeleted)
	if got := f.parked(t); len(got) != 0 {
		t.Errorf("the settled rating is still parked: %v", got)
	}
}

func TestRatingMirrorRetriesAParkedRatingAfterRedisIsLost(t *testing.T) {
	f := newGRFix(t)
	store := galgameRepo.NewRatingStore(f.db)
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{NextCursor: "1", Items: []catalogclient.RatingChange{
		{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
	}}}}
	down := func(context.Context, int64) (string, error) { return "", fmt.Errorf("catalog down") }
	galgameService.NewRatingMirror(feed, store, f.rdb, []string{grForumClient}).WithWorkFate(down).Run()
	if len(f.parked(t)) != 1 {
		t.Fatalf("the rating catalog could not answer about was not parked")
	}

	if err := f.rdb.FlushAll(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	none := func(context.Context, int64, int64) (int64, bool, error) { return 0, false, nil }
	galgameService.NewRatingMirror(&grFeed{}, store, f.rdb, []string{grForumClient}).
		WithWorkFate(fatesOf(nil)).WithCurrentRating(none).Run()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingAlice) +
		f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, grRatingAlice); n != 0 {
		t.Errorf("a rating its author deleted is still kept after Redis was lost")
	}
	if got := f.parked(t); len(got) != 0 {
		t.Errorf("the settled rating is still parked: %v", got)
	}
}

func TestRatingMirrorAsksNothingAboutARatingItNeverHad(t *testing.T) {
	f := newGRFix(t)
	never := grRatingRaters + 90
	if err := galgameRepo.NewRatingStore(f.db).ParkGoneRating(never, grWorkSFW); err != nil {
		t.Fatal(err)
	}
	feed := &grFeed{pages: []*catalogclient.RatingChangePage{{Items: []catalogclient.RatingChange{
		{ID: int64(never), WorkID: grWorkSFW, Gone: true},
	}}}}
	asked := 0
	fate := func(context.Context, int64) (string, error) { asked++; return "", fmt.Errorf("catalog down") }
	current := func(context.Context, int64, int64) (int64, bool, error) { asked++; return 0, false, fmt.Errorf("down") }
	galgameService.NewRatingMirror(feed, galgameRepo.NewRatingStore(f.db), f.rdb, []string{grForumClient}).
		WithWorkFate(fate).WithCurrentRating(current).Run()

	if asked != 0 {
		t.Errorf("asked NextMoe %d times about a rating the forum never had", asked)
	}
	if got := f.parked(t); len(got) != 0 {
		t.Errorf("a rating the forum never had is parked: %v", got)
	}
}

func TestRatingMirrorAsksNextMoeForTheRatingAMergeKept(t *testing.T) {
	merged := int64(grRatingRaters + 61)
	for name, tc := range map[string]struct {
		current currentOf
		want    string
	}{
		"another rating of the person's": {
			current: func(context.Context, int64, int64) (int64, bool, error) { return merged, true, nil },
			want:    "held",
		},
		"none": {
			current: func(context.Context, int64, int64) (int64, bool, error) { return 0, false, nil },
			want:    "dropped",
		},
		"this very rating": {
			current: func(context.Context, int64, int64) (int64, bool, error) { return int64(grRatingAlice), true, nil },
			want:    "kept",
		},
		"NextMoe down": {
			current: func(context.Context, int64, int64) (int64, bool, error) { return 0, false, fmt.Errorf("down") },
			want:    "parked",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newGRFix(t)
			f.run(t, `DELETE FROM galgame_rating_held WHERE id = ?`, grRatingAlice)
			f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, now())`, grRatingAlice, grBob)
			f.run(t, `UPDATE galgame_rating SET like_count = 1 WHERE id = ?`, grRatingAlice)
			var asked [2]int64
			current := func(ctx context.Context, user, work int64) (int64, bool, error) {
				asked = [2]int64{user, work}
				return tc.current(ctx, user, work)
			}

			runMirrorAsking(t, f, current, catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true})
			if asked != [2]int64{grAlice, grWorkSFW} {
				t.Errorf("asked about %v, want the author's rating of the work", asked)
			}
			live := f.count(t, `SELECT count(*) FROM galgame_rating WHERE id = ?`, grRatingAlice)
			held := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, grRatingAlice)
			parked := len(f.parked(t))
			switch tc.want {
			case "held":
				f.assertHeld(t, grRatingAlice, galgameRepo.HeldMerged)
			case "dropped":
				if live+held+parked != 0 {
					t.Errorf("a rating its author deleted was kept: live %d held %d parked %d", live, held, parked)
				}
			case "kept", "parked":
				if live != 1 || held != 0 || parked != map[string]int{"kept": 0, "parked": 1}[tc.want] {
					t.Errorf("live %d held %d parked %d", live, held, parked)
				}
			}
		})
	}
}

type sweepAuthors struct {
	users map[int]userclient.User
	err   error
}

func (s sweepAuthors) Users(_ context.Context, ids []int) (map[int]userclient.User, error) {
	out := map[int]userclient.User{}
	for _, id := range ids {
		if u, ok := s.users[id]; ok {
			out[id] = u
		}
	}
	return out, s.err
}

func TestHeldRatingSweepDropsTheHeldRatingsOfDeletedAccounts(t *testing.T) {
	f := newGRFix(t)
	anonymized, erased, alive := grRatingRaters+80, grRatingRaters+81, grRatingRaters+82
	ids := []int{anonymized, erased, alive}
	t.Cleanup(func() { f.run(t, `DELETE FROM galgame_rating_held WHERE id IN ?`, ids) })
	f.run(t, `INSERT INTO galgame_rating_held (id, user_id, work_id, reason, row) VALUES
		(?, ?, ?, 'deleted', '{}'), (?, ?, ?, 'merged', '{}'), (?, ?, ?, 'deleted', '{}')`,
		anonymized, grAlice, grWorkSFW, erased, grGone, grWorkSFW, alive, grBob, grWorkSFW)
	when := "2026-10-20T00:00:00Z"
	authors := sweepAuthors{users: map[int]userclient.User{
		grAlice: {ID: grAlice, AnonymizedAt: &when},
		grBob:   {ID: grBob},
	}}
	store := galgameRepo.NewRatingStore(f.db)

	galgameService.NewHeldRatingSweep(sweepAuthors{users: authors.users, err: fmt.Errorf("oauth down")}, store).Run()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id IN ?`, ids); n != 3 {
		t.Errorf("the sweep dropped %d held ratings while the account service was down", 3-n)
	}

	galgameService.NewHeldRatingSweep(authors, store).Run()
	for _, id := range []int{anonymized, erased} {
		if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, id); n != 0 {
			t.Errorf("held rating %d outlived its author's account", id)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, alive); n != 1 {
		t.Errorf("the held rating of a live account went")
	}
}

func TestHeldRatingSweepDropsWhatAnAuthorDeletedWhileTheWorkWasGone(t *testing.T) {
	f := newGRFix(t)
	kept := grRatingRaters
	f.run(t, `DELETE FROM galgame_rating_held WHERE id IN (?, ?)`, grRatingAlice, kept)
	f.run(t, `INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, now()), (?, ?, now())`,
		grRatingAlice, grBob, kept, grBob)
	f.run(t, `UPDATE galgame_rating SET like_count = 1 WHERE id IN (?, ?)`, grRatingAlice, kept)
	runMirror(t, f, map[int64]string{grWorkSFW: galgameRepo.HeldDeleted},
		catalogclient.RatingChange{ID: int64(grRatingAlice), WorkID: grWorkSFW, Gone: true},
		catalogclient.RatingChange{ID: int64(kept), WorkID: grWorkSFW, Gone: true})
	f.assertHeld(t, grRatingAlice, galgameRepo.HeldDeleted)
	f.assertHeld(t, kept, galgameRepo.HeldDeleted)

	var fateAsked, currentAsked int
	fate := galgameRepo.HeldDeleted
	var fateErr error
	sweep := galgameService.NewHeldRatingSweep(nil, galgameRepo.NewRatingStore(f.db)).WithRestoredWorks(
		func(context.Context, int64) (string, error) { fateAsked++; return fate, fateErr },
		func(_ context.Context, user, work int64) (int64, bool, error) {
			currentAsked++
			if work != grWorkSFW {
				t.Errorf("asked about work %d", work)
			}
			if user == grRater {
				return int64(kept), true, nil
			}
			return 0, false, nil
		})
	held := func() int {
		return f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id IN (?, ?)`, grRatingAlice, kept)
	}

	f.run(t, `UPDATE galgame SET catalog_rendered = false WHERE id = ?`, grWorkSFW)
	sweep.Run()
	if fateAsked+currentAsked != 0 || held() != 2 {
		t.Errorf("an unlisted work was asked about (fate %d, current %d) or lost holds (%d left)", fateAsked, currentAsked, held())
	}

	f.run(t, `UPDATE galgame SET catalog_rendered = true WHERE id = ?`, grWorkSFW)
	sweep.Run()
	if fateAsked != 1 || currentAsked != 0 || held() != 2 {
		t.Errorf("a work catalog still calls deleted lost holds (%d left) or its authors were asked (%d)", held(), currentAsked)
	}

	fateAsked, fateErr = 0, fmt.Errorf("catalog down")
	sweep.Run()
	if fateAsked != 1 || currentAsked != 0 || held() != 2 {
		t.Errorf("a work catalog could not answer about lost holds (%d left)", held())
	}

	fate, fateErr = "", nil
	sweep.Run()
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, grRatingAlice); n != 0 {
		t.Errorf("a rating its author deleted while the work was deleted is still held")
	}
	if n := f.count(t, `SELECT count(*) FROM galgame_rating_held WHERE id = ?`, kept); n != 1 {
		t.Errorf("a rating NextMoe still holds was dropped on restore")
	}

	runMirror(t, f, nil, catalogclient.RatingChange{ID: int64(kept), WorkID: grWorkSFW,
		Rating: liveRating(kept, grWorkSFW, grRater, time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC))})
	f.assertRestored(t, kept)
}
