package push

import (
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestSentRowsNeverMoveBackToAnOlderRevision(t *testing.T) {
	db := testdb.Open(t)
	const source = 960209901
	key := Key{Type: "TOPIC_CREATION", SourceID: source}
	anchor := presentationKey{Kind: 2, ID: "resource:960209901"}
	clean := func() {
		_ = db.Exec(`DELETE FROM activity_push_sent WHERE type = ? AND source_id = ?`, key.Type, key.SourceID).Error
		_ = db.Exec(`DELETE FROM anchor_presentation_sent WHERE anchor_kind = ? AND anchor_id = ?`, anchor.Kind, anchor.ID).Error
	}
	clean()
	t.Cleanup(clean)

	type row struct {
		Revision int64
		Removed  bool
	}
	p := &Pusher{db: db}
	pr := &Presenter{db: db}
	writers := []struct {
		name  string
		write func(rev int64, removed bool) error
		read  string
		args  []any
	}{
		{"activity", func(rev int64, removed bool) error { return p.upsertSent(key, 7, rev, removed) },
			`SELECT revision, removed FROM activity_push_sent WHERE type = ? AND source_id = ?`, []any{key.Type, key.SourceID}},
		{"presentation", func(rev int64, removed bool) error { return pr.upsertPresentationSent(anchor, rev, removed) },
			`SELECT revision, removed FROM anchor_presentation_sent WHERE anchor_kind = ? AND anchor_id = ?`, []any{anchor.Kind, anchor.ID}},
	}
	for _, w := range writers {
		for _, step := range []struct {
			rev     int64
			removed bool
			want    row
		}{
			{200, false, row{200, false}},
			{100, true, row{200, false}},
			{300, true, row{300, true}},
		} {
			if err := w.write(step.rev, step.removed); err != nil {
				t.Fatal(err)
			}
			var got row
			if err := db.Raw(w.read, w.args...).Scan(&got).Error; err != nil {
				t.Fatal(err)
			}
			if got != step.want {
				t.Fatalf("%s sent after writing revision %d = %+v, want %+v", w.name, step.rev, got, step.want)
			}
		}
	}
}
