package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/repository"
)

func TestV1WorksDefaultSFWGateFailsClosedOnNULL(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/works", "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	if g5HasID(body, g5NSFW) {
		t.Fatalf("default page included NSFW: %v", geItemIDs(body))
	}
	if g5HasID(body, g5NULL) {
		t.Fatalf("default page included a work whose content_limit has not synced: %v", geItemIDs(body))
	}
	if g5HasID(body, g5None) {
		t.Fatalf("resourceless published work on the default page: %v", geItemIDs(body))
	}
	if g5HasID(body, g5Drop) || g5HasID(body, g5Hidden) {
		t.Fatalf("hydrate-dropped ids still in items: %v", geItemIDs(body))
	}
	if int(body["total"].(float64)) != 14 {
		t.Fatalf("total %v want 14 SFW resource-bearing published rows (including hydrate-dropped ones, excluding NSFW and NULL)", body["total"])
	}
}

func TestV1WorksNSFWNotOnCreatedDescLimit10(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/works?sort=created_desc&limit=10", "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	ids := geItemIDs(body)
	if len(ids) != 10 {
		t.Fatalf("len=%d want 10: %v", len(ids), ids)
	}
	for _, id := range ids {
		if id == g5ID(g5NSFW) {
			t.Fatalf("newest NSFW leaked onto created_desc limit=10: %v", ids)
		}
	}
	if g5HasID(body, g5NSFW) {
		t.Fatal("NSFW on the page")
	}
}

func TestV1WorksCreatedAscIDTieBreak(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/works?sort=created_asc&limit=24", "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	ids := geItemIDs(body)
	lo, hi := -1, -1
	for i, id := range ids {
		if id == g5ID(g5TieLo) {
			lo = i
		}
		if id == g5ID(g5TieHi) {
			hi = i
		}
	}
	if lo < 0 || hi < 0 || lo > hi {
		t.Fatalf("created_asc tie: lo=%d hi=%d ids=%v", lo, hi, ids)
	}
}

func TestV1WorksUnknownSortAndEnumAndDate(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/works?sort=popularity_desc", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_SORT")
	resp, body = f.get(t, "/api/v1/works?resource_platforms=windows", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE")
	resp, body = f.get(t, "/api/v1/works?released_from=2026-13", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/works?released_from=2026-01-01", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/works?page=101&limit=100", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/works?limit=101", "/works")
	geStatus(t, resp, body, http.StatusBadRequest, "LIMIT_TOO_LARGE")
}

func TestV1WorksIncludeResourceless(t *testing.T) {
	f := newG5Fix(t)
	_, body := f.get(t, "/api/v1/works", "/works")
	if g5HasID(body, g5None) {
		t.Fatal("default included resourceless")
	}
	_, body = f.get(t, "/api/v1/works?include_resourceless=true", "/works")
	if !g5HasID(body, g5None) {
		t.Fatalf("include_resourceless=true missing %d: %v", g5None, geItemIDs(body))
	}
}

func TestV1WorksHydrateDropWarnsAndKeepsTotal(t *testing.T) {
	f := newG5Fix(t)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.get(t, "/api/v1/works?limit=24", "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	if g5HasID(body, g5Drop) {
		t.Fatal("omitted catalog id appeared in items")
	}
	total := int(body["total"].(float64))
	if total <= len(geItemIDs(body)) {
		t.Fatalf("total %d should exceed items %d when a paged id is dropped", total, len(geItemIDs(body)))
	}
	if !strings.Contains(buf.String(), "catalog did not render work, dropped") || !strings.Contains(buf.String(), g5ID(g5Drop)) {
		t.Fatalf("missing WARN for dropped work: %s", buf.String())
	}
}

func TestV1WorksNoLibraryParam(t *testing.T) {
	f := newG5Fix(t)
	before := len(f.cat.searched)
	resp, body := f.get(t, "/api/v1/works?library=true", "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	if len(f.cat.searched) != before {
		t.Fatal("/works with library=true still called catalog search")
	}
}

func TestV1LibraryWorksIgnoresForumFilters(t *testing.T) {
	f := newG5Fix(t)
	_, a := f.get(t, "/api/v1/library-works", "/library-works")
	n := len(f.cat.searched)
	if n == 0 {
		t.Fatal("library-works did not call catalog search")
	}
	_, b := f.get(t, "/api/v1/library-works?resource_type=game&game_type=plot&collected_from=2026", "/library-works")
	if len(f.cat.searched) != n+1 {
		t.Fatalf("forum filters must still hit catalog, searches=%d", len(f.cat.searched))
	}
	last := f.cat.searched[len(f.cat.searched)-1]
	if last.Get("resource_type") != "" || last.Get("game_type") != "" || last.Get("collected_from") != "" {
		t.Fatalf("forum filters leaked into catalog query: %v", last)
	}
	if last.Get("sort") != "popularity" {
		t.Fatalf("sort=%q", last.Get("sort"))
	}
	if a["total"] != b["total"] || len(geItemIDs(a)) != len(geItemIDs(b)) {
		t.Fatalf("forum filters changed the library page: %v vs %v", geItemIDs(a), geItemIDs(b))
	}
	resp, body := f.get(t, "/api/v1/library-works?sort=hot", "/library-works")
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_SORT")
	resp, body = f.get(t, "/api/v1/library-works?q=%20%20", "/library-works")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
}

func TestV1LibraryWorksReleasedLadder(t *testing.T) {
	f := newG5Fix(t)
	_, body := f.get(t, "/api/v1/library-works?released_from=2026-02&released_to=2026-02", "/library-works")
	last := f.cat.searched[len(f.cat.searched)-1]
	if last.Get("released_after") != "2026-02-01" || last.Get("released_before") != "2026-02-28" {
		t.Fatalf("ladder %v", last)
	}
	_ = body
}

func TestV1WorkCollectedMonths(t *testing.T) {
	f := newG5Fix(t)
	resp, body := f.get(t, "/api/v1/works/collected-months", "/works/collected-months")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["object"] != "work_collected_months" {
		t.Fatalf("object %v", body["object"])
	}
	items := body["items"].([]any)
	if len(items) == 0 {
		t.Fatal("empty collected months")
	}
	f.app.GalgameV1.WithCollectedMonths(func(bool) ([]repository.CollectedMonth, error) {
		return nil, errScanFailed
	})
	resp, body = f.get(t, "/api/v1/works/collected-months", "/works/collected-months")
	geStatus(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1WorksCatalogDown(t *testing.T) {
	f := newG5Fix(t)
	f.cat.fail.Store(true)
	resp, body := f.get(t, "/api/v1/works", "/works")
	geStatus(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.get(t, "/api/v1/library-works", "/library-works")
	geStatus(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1WorksQueryURLValues(t *testing.T) {
	f := newG5Fix(t)
	q := url.Values{"resource_platforms": {"win"}, "limit": {"24"}}
	resp, body := f.get(t, "/api/v1/works?"+q.Encode(), "/works")
	geStatus(t, resp, body, http.StatusOK, "")
	if !g5HasID(body, g5SFW0) {
		t.Fatalf("win filter missed SFW0: %v", geItemIDs(body))
	}
}
