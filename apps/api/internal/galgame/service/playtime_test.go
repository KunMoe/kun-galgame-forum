package service

import (
	"sort"
	"testing"

	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/pkg/catalogclient"
)

func strPtr(s string) *string { return &s }

func TestFoldRecords_CollapsesClientsTheWayCatalogDoes(t *testing.T) {
	rows := []catalogclient.PlaytimeRecord{
		{WorkID: 7, Minutes: 300},
		{WorkID: 7, Minutes: 720},
		{WorkID: 9, Minutes: 60},
	}

	order, byWork := foldRecords(rows)
	if len(order) != 2 || order[0] != 7 || order[1] != 9 {
		t.Fatalf("order = %v, want [7 9]", order)
	}

	seven := byWork[7]
	if seven.minutes != 720 {
		t.Errorf("minutes = %d, want the larger 720", seven.minutes)
	}
	if seven.clients != 2 {
		t.Errorf("clients = %d, want 2", seven.clients)
	}
	if seven.lastIndex != 1 {
		t.Errorf("lastIndex = %d, want 1", seven.lastIndex)
	}

	nine := byWork[9]
	if nine.clients != 1 {
		t.Errorf("clients = %d, want 1", nine.clients)
	}
	if nine.lastIndex != 2 {
		t.Errorf("lastIndex = %d, want 2", nine.lastIndex)
	}

	folded := []foldedPlaytime{seven, nine}
	sort.Slice(folded, func(i, j int) bool { return folded[i].lastIndex > folded[j].lastIndex })
	if folded[0].workID != 9 || folded[1].workID != 7 {
		t.Errorf("order by last occurrence = %d,%d want 9 then 7", folded[0].workID, folded[1].workID)
	}
}

func TestPlaytimeWithdrawn_TreatsASubFloorReportAsAbsent(t *testing.T) {
	cases := []struct {
		minutes int
		want    bool
	}{
		{0, true},
		{catalogclient.PlaytimeMinutesFloor - 1, true},
		{catalogclient.PlaytimeMinutesFloor, false},
		{720, false},
	}
	for _, c := range cases {
		if got := playtimeWithdrawn(c.minutes); got != c.want {
			t.Errorf("playtimeWithdrawn(%d) = %v, want %v", c.minutes, got, c.want)
		}
	}
}

func TestFoldRecords_KeepsAWorkAnotherClientStillReports(t *testing.T) {
	rows := []catalogclient.PlaytimeRecord{
		{WorkID: 7, Minutes: 0},
		{WorkID: 7, Minutes: 480},
	}
	_, byWork := foldRecords(rows)
	if playtimeWithdrawn(byWork[7].minutes) {
		t.Errorf("folded minutes = %d, want the other client's 480 to keep the work visible", byWork[7].minutes)
	}
}

func TestAssembleMine_StateOnlyWorkHasZeroMinutes(t *testing.T) {
	out := assembleMine(
		nil,
		nil,
		[]int64{7},
		map[int64]catalogclient.WorkStateRecord{
			7: {WorkID: 7, State: catalogclient.WorkStateWish},
		},
		map[int64]int{7: 1},
	)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0].minutes != 0 || out[0].clients != 0 || out[0].status != playstate.Wish || out[0].gid != 1 {
		t.Errorf("got %+v, want minutes 0 clients 0 status wish gid 1", out[0])
	}
}

func TestAssembleMine_FinishedWorksCountsDoneMain(t *testing.T) {
	out := assembleMine(
		[]int64{7, 9},
		map[int64]foldedPlaytime{
			7: {workID: 7, minutes: 120, lastIndex: 0},
			9: {workID: 9, minutes: 60, lastIndex: 1},
		},
		[]int64{7, 9, 11},
		map[int64]catalogclient.WorkStateRecord{
			7:  {WorkID: 7, State: catalogclient.WorkStateDone, Completion: strPtr("main")},
			9:  {WorkID: 9, State: catalogclient.WorkStateDoing},
			11: {WorkID: 11, State: catalogclient.WorkStateDone, Completion: strPtr("all")},
		},
		map[int64]int{7: 1, 9: 2, 11: 3},
	)
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	if out[0].workID != 9 || out[1].workID != 7 || out[2].workID != 11 {
		t.Errorf("order = %d,%d,%d want 9 (playtime lastIndex), 7, then state-only 11",
			out[0].workID, out[1].workID, out[2].workID)
	}
	if out[1].status != playstate.DoneMain {
		t.Errorf("work 7 status = %q, want done_main", out[1].status)
	}
	if out[2].minutes != 0 || out[2].clients != 0 || out[2].status != playstate.DoneAll {
		t.Errorf("state-only = %+v, want minutes 0 done_all", out[2])
	}
	finished := 0
	for _, f := range out {
		if playStateFinished(f.status) {
			finished++
		}
	}
	if finished != 2 {
		t.Errorf("finished = %d, want 2 (done_main + done_all)", finished)
	}
}

func TestAssembleMine_KeepsWithdrawnPlaytimeWhenAStateRemains(t *testing.T) {
	out := assembleMine(
		[]int64{7, 9},
		map[int64]foldedPlaytime{
			7: {workID: 7, minutes: 0, lastIndex: 0},
			9: {workID: 9, minutes: 0, lastIndex: 1},
		},
		[]int64{7},
		map[int64]catalogclient.WorkStateRecord{
			7: {WorkID: 7, State: catalogclient.WorkStateWish},
		},
		map[int64]int{7: 1, 9: 2},
	)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1 — withdrawn with no state must drop", len(out))
	}
	if out[0].workID != 7 || out[0].minutes != 0 || out[0].status != playstate.Wish {
		t.Errorf("got %+v, want work 7 minutes 0 wish", out[0])
	}
}

func TestPlayStateFinished_IgnoresDisplayOnlyDone(t *testing.T) {
	if playStateFinished("done") {
		t.Error("display-only done must not count as finished")
	}
	if !playStateFinished(playstate.DoneMain) {
		t.Error("done_main must count as finished")
	}
}
