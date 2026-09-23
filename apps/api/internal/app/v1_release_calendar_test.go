package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestV1ReleaseCalendarMonthFollowsCursor(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar?month=2026-09", "/release-calendar")
	geStatus(t, resp, body, http.StatusOK, "")
	items := body["items"].([]any)
	if len(items) <= 100 {
		t.Fatalf("month walk stopped at first page: %d", len(items))
	}
	if body["is_truncated"] != false {
		t.Fatalf("101 items should not hit the 2000 cap: %+v", body["is_truncated"])
	}
	if body["calendar_month"] != "2026-09" {
		t.Fatalf("calendar_month %v", body["calendar_month"])
	}
	if body["item_count"] != float64(len(items)) {
		t.Fatalf("item_count %v len %d", body["item_count"], len(items))
	}
	if int(body["item_count"].(float64)) < 101 {
		t.Fatalf("item_count %v", body["item_count"])
	}
}

func TestV1ReleaseCalendarTodayUsesWalkedMonth(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar/today", "/release-calendar/today")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["has_release"] != true {
		t.Fatalf("today at position 101 must count: %+v", body)
	}
	if body["today_date"] != "2026-09-23" {
		t.Fatalf("today %v", body["today_date"])
	}
	if body["expires_in"].(float64) < 0 {
		t.Fatalf("expires_in %v", body["expires_in"])
	}
}

func TestV1ReleaseCalendarUpcomingFailedMonth(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar/upcoming", "/release-calendar/upcoming")
	geStatus(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1ReleaseCalendarPendingPrecision(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar/pending?year=2026", "/release-calendar/pending")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["year"] != float64(2026) {
		t.Fatalf("year %v", body["year"])
	}
	found := false
	for _, q := range f.cat.calQ {
		if q.Get("precision") == "year" && q.Get("year") == "2026" {
			found = true
		}
		if q.Get("precision") == "month" {
			t.Fatalf("pending sent precision=month: %v", q)
		}
	}
	if !found {
		t.Fatalf("pending did not send precision=year: %+v", f.cat.calQ)
	}
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("pending items %v", geItemIDs(body))
	}
	if items[0].(map[string]any)["release_date_precision"] != "year" {
		t.Fatalf("pending row %+v", items[0])
	}
}

func TestV1ReleaseCalendarTBAStatusUnknown(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar/tba", "/release-calendar/tba")
	geStatus(t, resp, body, http.StatusOK, "")
	found := false
	for i, q := range f.cat.calQ {
		if f.cat.calBucket[i] != "/tba" {
			continue
		}
		if q.Get("status") != "unknown" {
			t.Fatalf("tba status=%q", q.Get("status"))
		}
		found = true
	}
	if !found {
		t.Fatalf("tba query missing: buckets=%v", f.cat.calBucket)
	}
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("tba items %v", geItemIDs(body))
	}
	if items[0].(map[string]any)["release_date_precision"] != "year" {
		t.Fatalf("tba must keep year-precision rows: %+v", items[0])
	}
}

func TestV1ReleaseCalendarBadMonth(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/release-calendar?month=2026-13", "/release-calendar")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	f.cat.fail.Store(true)
	resp, body = f.get(t, "/api/v1/release-calendar?month=2026-09", "/release-calendar")
	geStatus(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1ReleaseCalendarTruncationWarn(t *testing.T) {
	f := newG5Fix(t)
	rows := f.cat.calItems["2026-09"]
	for len(f.cat.calItems["2026-09"]) < 2001 {
		f.cat.calItems["2026-09"] = append(f.cat.calItems["2026-09"], rows[0])
	}
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.get(t, "/api/v1/release-calendar?month=2026-09", "/release-calendar")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["is_truncated"] != true {
		t.Fatalf("2001 items must set is_truncated: %+v", body["is_truncated"])
	}
	if int(body["item_count"].(float64)) != 2000 {
		t.Fatalf("capped item_count %v", body["item_count"])
	}
	if !strings.Contains(buf.String(), "month truncated at page cap") {
		t.Fatalf("missing truncation WARN: %s", buf.String())
	}
}
