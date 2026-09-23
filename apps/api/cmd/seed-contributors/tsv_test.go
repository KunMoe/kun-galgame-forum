package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseContributorTSV(t *testing.T) {
	in := strings.Join([]string{
		"1\t1\t2\t2025-07-20 21:58:23.086+00",
		"1\t1\t2\t2025-07-19 10:00:00+00",
		"1\t1\t356\t2025-07-21 02:55:42.374+00",
		"9682\t0\t112505\t2026-03-25 18:10:31.662+00",
	}, "\n")

	rows, stats, err := parseContributorTSV(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if stats.Lines != 4 || stats.Skipped != 0 || stats.Folded != 1 {
		t.Fatalf("stats = %+v, want 4 lines / 0 skipped / 1 folded", stats)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 distinct pairs", len(rows))
	}
	if rows[0].WorkID != 1 || rows[0].UserID != 2 {
		t.Errorf("first pair = (%d, %d), want (1, 2) — column 2 is the catalog work id, not the user",
			rows[0].WorkID, rows[0].UserID)
	}
	want := time.Date(2025, 7, 19, 10, 0, 0, 0, time.UTC)
	if !rows[0].Created.Equal(want) {
		t.Errorf("folded created = %v, want the EARLIER %v", rows[0].Created.UTC(), want)
	}
	if rows[2].WorkID != 9682 || rows[2].UserID != 112505 {
		t.Errorf("last pair = (%d, %d), want (9682, 112505)", rows[2].WorkID, rows[2].UserID)
	}
}

func TestParseContributorTSVSkipsUnusableRows(t *testing.T) {
	in := strings.Join([]string{
		"1\t1\t0\t2025-07-20 21:58:23.086+00",
		"0\t1\t5\t2025-07-20 21:58:23.086+00",
		"1\t1\t5\tnot-a-timestamp",
		"1\t1",
		"",
		"7\t1\t5\t2025-07-20 21:58:23.086+00",
	}, "\n")

	rows, stats, err := parseContributorTSV(strings.NewReader(in))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 || rows[0].WorkID != 7 {
		t.Fatalf("rows = %+v, want only the well-formed pair", rows)
	}
	if stats.Skipped != 4 {
		t.Errorf("skipped = %d, want 4", stats.Skipped)
	}
}
