package app

import (
	"fmt"
	"net/url"
	"slices"
	"testing"
	"time"
)

const (
	acQuizOnShown  = 950000951
	acQuizNoRow    = 950000952
	acWorkNoRow    = 950000699
	acResComment   = 950000961
	acRateComment  = 950000962
	acQuizComment  = 950000963
	acQuizNoRowCmt = 950000964
)

func (f *activityFix) seedSubjectsOfShownWork(t *testing.T) {
	t.Helper()
	clean := func() {
		_ = f.db.Exec(`DELETE FROM galgame_quiz WHERE id IN (?, ?)`, acQuizOnShown, acQuizNoRow).Error
	}
	clean()
	t.Cleanup(clean)
	at := acTie.Add(8 * time.Hour)
	for _, q := range []struct{ id, work int }{{acQuizOnShown, acWorkShown}, {acQuizNoRow, acWorkNoRow}} {
		f.run(t, `INSERT INTO galgame_quiz (id, user_id, type, question, status_update_time, created, updated)
			VALUES (?, ?, 'single', 'Which route', ?, ?, ?)`, q.id, acUserAlice, at, at, at)
		f.run(t, `INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, q.id, q.work)
	}
	for _, c := range []struct {
		typ    string
		source int
		link   string
	}{
		{"GALGAME_RESOURCE_COMMENT_CREATION", acResComment, fmt.Sprintf("/galgame/resource/%d", acResShown)},
		{"GALGAME_RATING_COMMENT_CREATION", acRateComment, fmt.Sprintf("/galgame-rating/%d", acRatingHold)},
		{"GALGAME_QUIZ_COMMENT_CREATION", acQuizComment, fmt.Sprintf("/galgame-quiz/%d", acQuizOnShown)},
		{"GALGAME_QUIZ_COMMENT_CREATION", acQuizNoRowCmt, fmt.Sprintf("/galgame-quiz/%d", acQuizNoRow)},
	} {
		f.run(t, `INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
			VALUES (?, ?, ?, 0, 'a comment', ?, false, ?)`, c.typ, c.source, acUserBob, c.link, at)
	}
}

func TestV1ActivitiesHideWhileTheirWorkIsNotRendered(t *testing.T) {
	f := newActivityFix(t)
	f.seedSubjectsOfShownWork(t)
	mine := f.ourRows(t)
	tied := []string{
		f.rowID(t, "GALGAME_QUIZ_CREATION", acQuizOnShown),
		f.rowID(t, "GALGAME_RESOURCE_COMMENT_CREATION", acResComment),
		f.rowID(t, "GALGAME_RATING_COMMENT_CREATION", acRateComment),
		f.rowID(t, "GALGAME_QUIZ_COMMENT_CREATION", acQuizComment),
	}
	var onWork []int64
	if err := f.db.Raw(`SELECT id FROM feed_activity WHERE work_id = ?`, acWorkShown).Scan(&onWork).Error; err != nil || len(onWork) < 3 {
		t.Fatalf("rows carrying the work id: %v %v", onWork, err)
	}
	for _, id := range onWork {
		tied = append(tied, fmt.Sprint(id))
	}
	loose := []string{
		f.rowID(t, "GALGAME_QUIZ_CREATION", acQuizNoRow),
		f.rowID(t, "GALGAME_QUIZ_COMMENT_CREATION", acQuizNoRowCmt),
		f.rowID(t, "TOPIC_CREATION", acTopicNormal),
	}
	setRendered := func(rendered bool) {
		f.run(t, `UPDATE galgame SET catalog_rendered = ? WHERE id = ?`, rendered, acWorkShown)
	}
	stream := func(limit int) []string {
		return activityIDs(f.walk(t, url.Values{}, limit), mine)
	}

	before := stream(50)
	for _, id := range append(slices.Clone(tied), loose...) {
		if !slices.Contains(before, id) {
			t.Fatalf("row %s missing before the work is hidden: %v", id, before)
		}
	}

	setRendered(false)
	for _, limit := range []int{2, 50} {
		got := stream(limit)
		for _, id := range tied {
			if slices.Contains(got, id) {
				t.Errorf("limit %d: row %s of a work catalog no longer renders is in the stream", limit, id)
			}
		}
		for _, id := range loose {
			if !slices.Contains(got, id) {
				t.Errorf("limit %d: row %s not tied to the hidden work went missing", limit, id)
			}
		}
		if len(got) != len(before)-len(tied) {
			t.Errorf("limit %d: %d rows while hidden, want %d", limit, len(got), len(before)-len(tied))
		}
	}

	setRendered(true)
	if after := stream(50); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Errorf("after restore %v, want %v", after, before)
	}
}
