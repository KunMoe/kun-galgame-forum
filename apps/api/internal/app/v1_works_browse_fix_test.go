package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	calendarapiv1 "kun-galgame-api/internal/galgame/calendarapiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/testdb"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	g5WorkMin  = 945000001
	g5WorkMax  = 945000099
	g5CalMin   = 946000001
	g5CalMax   = 946000250
	g5SFW0     = 945000001
	g5NULL     = 945000011
	g5NSFW     = 945000012
	g5None     = 945000013
	g5TieLo    = 945000014
	g5TieHi    = 945000015
	g5Drop     = 945000016
	g5Hidden   = 945000017
	g5CalToday = 946000101
)

type g5Fix struct {
	app  *App
	db   *gorm.DB
	cat  *fakeCatalog
	spec *specConformance
}

func newG5Fix(t *testing.T) *g5Fix {
	t.Helper()
	db := testdb.Open(t)
	cat := &fakeCatalog{
		rows:     map[int]client.CatalogWorkListItem{},
		omitIDs:  map[int]bool{},
		calItems: map[string][]client.CatalogWorkListItem{},
		calToday: map[string]string{},
		calMin:   map[string]string{},
		calMax:   map[string]string{},
		calFail:  map[string]bool{},
	}
	f := &g5Fix{db: db, cat: cat}
	f.seedCatalog(t)
	f.seedLocal(t)
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = geCDN
	f.app = &App{
		Fiber:           newFiber(),
		Config:          cfg,
		DB:              db,
		GalgameV1:       galgameapiv1.New(cat, nil, nil, nil, geCDN).WithWork(db, nil, nil, nil),
		GalgameEntityV1: entityapiv1.New(cat, db, geCDN),
		GalgameCalendarV1: calendarapiv1.New(cat, db, geCDN).WithClock(func() time.Time {
			loc, err := time.LoadLocation("Asia/Tokyo")
			if err != nil {
				loc = time.UTC
			}
			return time.Date(2026, 9, 23, 12, 0, 0, 0, loc)
		}),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	return f
}

func (f *g5Fix) seedCatalog(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	works := []geWork{}
	for i := 0; i < 10; i++ {
		works = append(works, geWork{
			id: g5SFW0 + i, name: fmt.Sprintf("SFW%d", i), release: "2026-01-15",
			limit: "sfw", rating: "all_ages", local: true, created: base.Add(time.Duration(i) * time.Hour),
			platforms: []string{"win"}, languages: []string{"zh-cn"},
		})
	}
	works[0].runtimes = []string{"emulator"}
	works[1].runtimes = []string{"kirikiroid2"}
	works = append(works,
		geWork{id: g5NULL, name: "NullLimit", release: "2026-02-01", limit: "", rating: "all_ages", local: true, created: base.Add(20 * time.Hour), platforms: []string{"win"}, languages: []string{"ja-jp"}},
		geWork{id: g5NSFW, name: "NewestNSFW", release: "2026-03-01", limit: "nsfw", rating: "r18", local: true, created: base.Add(40 * time.Hour), platforms: []string{"and"}, languages: []string{"zh-cn"}},
		geWork{id: g5None, name: "NoResource", release: "2026-01-01", limit: "sfw", rating: "all_ages", local: true, created: base.Add(5 * time.Hour)},
		geWork{id: g5TieLo, name: "TieLo", release: "2025-06-01", limit: "sfw", rating: "all_ages", local: true, created: base.Add(3 * time.Hour), platforms: []string{"win"}, languages: []string{"zh-cn"}},
		geWork{id: g5TieHi, name: "TieHi", release: "2025-06-01", limit: "sfw", rating: "all_ages", local: true, created: base.Add(3 * time.Hour), platforms: []string{"mac"}, languages: []string{"en-us"}},
		geWork{id: g5Drop, name: "Dropped", release: "2026-04-01", limit: "sfw", rating: "all_ages", local: true, created: base, platforms: []string{"win"}, languages: []string{"zh-cn"}},
		geWork{id: g5Hidden, name: "HiddenWork", release: "2026-01-01", limit: "sfw", rating: "all_ages", local: true, created: base},
	)
	f.cat.works = works
	for _, w := range works {
		var row client.CatalogWorkListItem
		if w.id == g5Hidden {
			decodeInto(t, resourceHiddenJSON(w.id, w.name), &row)
		} else {
			decodeInto(t, geRowJSON(w), &row)
		}
		f.cat.rows[w.id] = row
	}
	f.cat.omitIDs[g5Drop] = true

	monthItems := make([]client.CatalogWorkListItem, 0, 101)
	for i := 0; i < 100; i++ {
		w := geWork{id: g5CalMin + i, name: fmt.Sprintf("Cal%d", i), release: "2026-09-01", limit: "sfw", rating: "all_ages"}
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		monthItems = append(monthItems, row)
		f.cat.rows[w.id] = row
	}
	todayWork := geWork{id: g5CalToday, name: "TodayRelease", release: "2026-09-23", limit: "sfw", rating: "all_ages"}
	var todayRow client.CatalogWorkListItem
	decodeInto(t, geRowJSON(todayWork), &todayRow)
	monthItems = append(monthItems, todayRow)
	f.cat.rows[g5CalToday] = todayRow
	f.cat.calItems["2026-09"] = monthItems
	f.cat.calToday["2026-09"] = "2026-09-23"
	f.cat.calMin["2026-09"] = "2026-01"
	f.cat.calMax["2026-09"] = "2026-11"

	oct := geWork{id: g5CalMin + 200, name: "Oct", release: "2026-10-01", limit: "sfw", rating: "all_ages"}
	var octRow client.CatalogWorkListItem
	decodeInto(t, geRowJSON(oct), &octRow)
	f.cat.calItems["2026-10"] = []client.CatalogWorkListItem{octRow}
	f.cat.calFail["2026-10"] = true

	yearWork := geWork{id: g5CalMin + 210, name: "YearPending", release: "2026", limit: "sfw", rating: "all_ages"}
	var yearRow client.CatalogWorkListItem
	decodeInto(t, geRowJSON(yearWork), &yearRow)
	f.cat.calItems["pending:2026"] = []client.CatalogWorkListItem{yearRow}
	f.cat.rows[yearWork.id] = yearRow

	tbaWork := geWork{id: g5CalMin + 211, name: "TBAYear", release: "2024", limit: "sfw", rating: "all_ages"}
	var tbaRow client.CatalogWorkListItem
	decodeInto(t, geRowJSON(tbaWork), &tbaRow)
	f.cat.calItems["tba"] = []client.CatalogWorkListItem{tbaRow}
	f.cat.rows[tbaWork.id] = tbaRow
}

func (f *g5Fix) seedLocal(t *testing.T) {
	t.Helper()
	wipe := func() {
		for _, q := range []string{
			`DELETE FROM galgame_resource WHERE work_id BETWEEN ? AND ?`,
			`DELETE FROM galgame WHERE id BETWEEN ? AND ?`,
			`DELETE FROM galgame_view_daily WHERE entity_id BETWEEN ? AND ?`,
		} {
			if err := f.db.Exec(q, g5WorkMin, g5WorkMax).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	wipe()
	t.Cleanup(wipe)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for _, w := range f.cat.works {
		if !w.local {
			continue
		}
		updated := w.created.Add(time.Duration(w.id%50) * time.Minute)
		if w.limit == "" {
			run(`INSERT INTO galgame (id, view, created, updated, like_count, published, content_limit, resource_update_time) VALUES (?, ?, ?, ?, 0, true, NULL, ?)`,
				w.id, w.view, w.created, w.created, updated)
		} else {
			run(`INSERT INTO galgame (id, view, created, updated, like_count, published, content_limit, resource_update_time) VALUES (?, ?, ?, ?, 0, true, ?, ?)`,
				w.id, w.view, w.created, w.created, w.limit, updated)
		}
		if w.id == g5None {
			continue
		}
		plat, _ := json.Marshal(w.platforms)
		lang, _ := json.Marshal(w.languages)
		if len(w.platforms) == 0 {
			plat = []byte("[]")
		}
		if len(w.languages) == 0 {
			lang = []byte("[]")
		}
		run(`INSERT INTO galgame_resource (work_id, user_id, type, platform, language, platforms, languages, runtimes, provider, updated) VALUES (?, ?, 'game', 'windows', 'zh-cn', ?::jsonb, ?::jsonb, ?::jsonb, '{baidu}', now())`,
			w.id, geUser, string(plat), string(lang), geJSONKeys(w.runtimes))
	}
}

func (f *g5Fix) get(t *testing.T, rawURL, specPath string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.check(t, http.MethodGet, specPath, resp, body)
	var m map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("%s: %v: %s", rawURL, err, body)
		}
	}
	return resp, m
}

func g5ID(id int) string { return fmt.Sprintf("%d", id) }

func g5HasID(body map[string]any, id int) bool {
	for _, raw := range body["items"].([]any) {
		if raw.(map[string]any)["id"] == g5ID(id) {
			return true
		}
	}
	return false
}

var errScanFailed = errors.New("scan failed")
