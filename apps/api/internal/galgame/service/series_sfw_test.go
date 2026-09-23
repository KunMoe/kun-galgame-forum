package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

func seriesStub(t *testing.T, hasNSFWField bool) *httptest.Server {
	t.Helper()
	row := func(id int, name string, count int, nsfw bool) map[string]any {
		r := map[string]any{"id": id, "display_name": name, "work_count": count}
		if hasNSFWField {
			r["has_nsfw"] = nsfw
		}
		return r
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/catalog/series") {
			items := []map[string]any{
				row(1, "全年龄系列", 2, false),
				row(2, "成人系列", 3, true),
				row(3, "无已发布作品", 0, false),
				row(4, "无可展示成员", 2, false),
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list", "items": items, "next_cursor": "", "total": len(items),
			})
			return
		}
		member := func(id int) map[string]any {
			return map[string]any{
				"id": id, "display_name": "作品",
				"claim": map[string]any{"site": "kungal", "site_work_id": id, "state": "live"},
			}
		}
		items := []map[string]any{}
		switch req.URL.Query().Get("series_id") {
		case "1":
			items = append(items, member(11), member(12))
		case "2":
			items = append(items, member(21), member(22), member(23))
		case "4":
			items = append(items, map[string]any{"id": 4001, "display_name": "别处的作品"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list", "items": items, "total": len(items),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func seriesNames(t *testing.T, svc *SeriesService, isSFW bool) []string {
	t.Helper()
	page, appErr := svc.GetCards(context.Background(), nil, 1, 12, isSFW)
	if appErr != nil {
		t.Fatalf("GetCards: %v", appErr)
	}
	names := make([]string, 0, len(page.Series))
	for _, c := range page.Series {
		names = append(names, c.Name)
	}
	if int(page.Total) != len(names) {
		t.Errorf("total = %d but %d rows — the pager promises pages that filter away", page.Total, len(names))
	}
	return names
}

func TestSeriesIndex_HidesAdultSeriesFromSFWReaders(t *testing.T) {
	svc := NewSeriesService(client.New(seriesStub(t, true).URL, "nm_test_key", ""), nil, nil)

	if got := seriesNames(t, svc, true); len(got) != 2 {
		t.Errorf("SFW index = %v, want the all-ages series and unclaimed membership", got)
	}
	if got := seriesNames(t, svc, false); len(got) != 3 {
		t.Errorf("open index = %v, want every series that has catalog members", got)
	}
}

func TestSeriesIndex_HidesSeriesWithNoWorks(t *testing.T) {
	svc := NewSeriesService(client.New(seriesStub(t, true).URL, "nm_test_key", ""), nil, nil)

	for _, name := range seriesNames(t, svc, false) {
		if name == "无已发布作品" {
			t.Errorf("open index lists %q, which has no catalog members", name)
		}
	}
}

func TestSeriesIndex_UnansweredHasNSFWIsNotSafe(t *testing.T) {
	svc := NewSeriesService(client.New(seriesStub(t, false).URL, "nm_test_key", ""), nil, nil)

	if got := seriesNames(t, svc, true); len(got) != 0 {
		t.Errorf("SFW index = %v, want nothing — the catalog answered nothing", got)
	}
	if got := seriesNames(t, svc, false); len(got) != 3 {
		t.Errorf("open index = %v, want every series that has catalog members", got)
	}
}
