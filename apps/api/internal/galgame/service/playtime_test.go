package service

import (
	"sort"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

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

func TestAttachWorkStates_JoinsStatusAndLeavesGapsEmpty(t *testing.T) {
	folded := []foldedPlaytime{
		{workID: 7, minutes: 720},
		{workID: 9, minutes: 60},
		{workID: 11, minutes: 100},
	}
	attachWorkStates(folded, map[int64]catalogclient.WorkStateRecord{
		7: {WorkID: 7, State: catalogclient.WorkStateDone},
		9: {WorkID: 9, State: catalogclient.WorkStateDoing},
	})
	if folded[0].status != catalogclient.WorkStateDone {
		t.Errorf("work 7 status = %q, want done", folded[0].status)
	}
	if folded[1].status != catalogclient.WorkStateDoing {
		t.Errorf("work 9 status = %q, want doing", folded[1].status)
	}
	if folded[2].status != "" {
		t.Errorf("work 11 status = %q, want empty when missing", folded[2].status)
	}
	finished := 0
	for _, f := range folded {
		if f.status == catalogclient.WorkStateDone {
			finished++
		}
	}
	if finished != 1 {
		t.Errorf("finished = %d, want 1", finished)
	}
}
