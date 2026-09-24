package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/calendarapiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/testdb"
	legacyErrors "kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	geCDN = "https://image.test.example"

	geWorkMin = 941000001
	geWorkMax = 941000040

	geTagMain    = 5101
	geTagSexual  = 5102
	geTagHidden  = 5103
	geTagEmpty   = 5104
	geCompany    = 6101
	geImprint    = 6102
	geMerged     = 6103
	geEngine     = 7101
	geSeries     = 8101
	geSeriesNSFW = 8102
	geCreditName = 9101
	geCharacter  = 9201
	geUser       = 941000900
)

func geHash(n int) string {
	return fmt.Sprintf("%064x", n)
}

func geImageURL(n int) string {
	h := geHash(n)
	return fmt.Sprintf("https://image.other.example/%s/%s/%s.webp", h[:2], h[2:4], h)
}

// geWork is one catalog work and, when local is set, the forum's row for it.
type geWork struct {
	id         int
	name       string
	release    string
	limit      string // claim content_limit; "" means no claim
	rating     string // catalog content_rating
	tags       []int
	local      bool
	view       int
	created    time.Time
	platforms  []string
	languages  []string
	legacyPlat string
	ratings    []int
	gameTypes  []string
}

// Ties on purpose: view 7 three times among the local rows, 2026-01-01 twice,
// one work the forum has no row for, one NSFW by claim that is all_ages by age
// rating and one r18 by age rating that the claim marks sfw.
func geWorks() []geWork {
	t0 := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	return []geWork{
		{id: geWorkMin + 0, name: "Alpha", release: "2026-01-01", limit: "sfw", rating: "all_ages", tags: []int{geTagMain}, local: true, view: 7, created: t0, platforms: []string{"win", "and"}, languages: []string{"zh-cn"}, legacyPlat: "windows", ratings: []int{8, 9}, gameTypes: []string{"plot"}},
		{id: geWorkMin + 1, name: "Beta", release: "2026-01-01", limit: "sfw", rating: "r18", tags: []int{geTagMain, geTagSexual}, local: true, view: 7, created: t0.Add(time.Hour), platforms: []string{"win"}, languages: []string{"ja-jp", "zh-cn"}, legacyPlat: "windows"},
		{id: geWorkMin + 2, name: "Gamma", release: "2025-06", limit: "nsfw", rating: "all_ages", tags: []int{geTagMain}, local: true, view: 7, created: t0.Add(2 * time.Hour), platforms: []string{"and"}, languages: []string{"zh-tw"}, legacyPlat: "app"},
		{id: geWorkMin + 3, name: "Delta", release: "2024", limit: "sfw", rating: "all_ages", tags: []int{geTagMain}, local: true, view: 3, created: t0.Add(3 * time.Hour), platforms: []string{"mac"}, languages: []string{"en-us"}, legacyPlat: "windows", ratings: []int{5}},
		{id: geWorkMin + 4, name: "Epsilon", release: "", limit: "", rating: "all_ages", tags: []int{geTagMain}},
		{id: geWorkMin + 5, name: "Zeta", release: "2023-02-02", limit: "sfw", rating: "all_ages", tags: []int{geTagMain}, local: true, view: 1, created: t0.Add(4 * time.Hour), platforms: []string{"win"}, languages: []string{"zh-cn"}, legacyPlat: "windows", gameTypes: []string{"moe"}},
	}
}

func geRowJSON(w geWork) string {
	claim := "null"
	if w.limit != "" {
		claim = fmt.Sprintf(`{"site":"kungal","site_work_id":%d,"state":"live","content_limit":%q}`, w.id, w.limit)
	}
	release := "null"
	if w.release != "" {
		release = strconv.Quote(w.release)
	}
	return fmt.Sprintf(`{"id":%d,"display_name":%q,"latin":%q,
		"localized":{"zh-Hans":{"value":%q,"machine":true}},
		"content_rating":%q,"content_limit":%q,"release_date":%s,"claim":%s,
		"cover_slots":{"portrait":{"url":%q,"width":256,"height":361,"thumbhash":"pUgK"},"banner":{"url":%q,"width":800,"height":450,"thumbhash":"sigO"}},
		"labels":[{"id":%d,"display_name":"Publisher Co","role":"publisher","kind":"publisher"},{"id":%d,"display_name":"Maker Brand","localized":{"zh-Hans":{"value":"制作品牌"}},"role":"developer","kind":"game_brand"}]}`,
		w.id, w.name, strings.ToLower(w.name), w.name+"（中）", w.rating, geShelf(w), release, claim,
		geImageURL(w.id), geImageURL(w.id+1000), geImprint, geCompany)
}

// geShelf is catalog's verdict for a fixture work: the claim's limit, else what
// an unclaimed work with ordinary cover art gets.
func geShelf(w geWork) string {
	if w.limit != "" {
		return w.limit
	}
	if w.rating == "r18" {
		return "nsfw"
	}
	return "sfw"
}

type geMember struct {
	workID int
	via    *client.CatalogLabelVia
}

// fakeCatalog answers entityapiv1.Catalog from in-memory fixtures decoded from
// JSON, because the client's wire types carry unexported fields.
type fakeCatalog struct {
	sfwHidden  map[int]bool
	rowsGate   chan struct{}
	rowsIn     chan struct{}
	mu         sync.Mutex
	fail       atomic.Bool
	throttle   atomic.Bool
	calls      atomic.Int32
	rows       map[int]client.CatalogWorkListItem
	details    map[int]*client.CatalogWorkDetail
	movedWorks map[int]int64
	media      map[int64]client.CatalogEntityMedia
	works      []geWork
	taxonomy   map[string][]client.CatalogTaxonomyItem
	hits       map[string][]client.CatalogEntityHit
	tags       map[string]*client.CatalogTagDetail
	labels     map[string]*client.CatalogLabelDetail
	moved      map[string]int64
	engines    map[string]*client.CatalogEngineDetail
	series     map[string]*client.CatalogSeriesDetail
	graphs     map[string]*client.CatalogLabelRelationGraph
	wiki       map[int]int64
	names      map[int64]*client.CatalogName
	chars      map[int64]*client.CatalogCharacter
	members    map[string][]int
	rollup     map[string][]geMember
	seriesMem  map[int][]int
	gotLimits  []string
	searched   []url.Values
	omitIDs    map[int]bool
	calItems   map[string][]client.CatalogWorkListItem
	calToday   map[string]string
	calMin     map[string]string
	calMax     map[string]string
	calFail    map[string]bool
	calQ       []url.Values
	calBucket  []string
}

func decodeInto(t *testing.T, raw string, out any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		t.Fatalf("fixture %s: %v", raw, err)
	}
}

func (f *fakeCatalog) err() *legacyErrors.AppError {
	f.calls.Add(1)
	if f.fail.Load() {
		return legacyErrors.New(233, "catalog down", http.StatusServiceUnavailable)
	}
	if f.throttle.Load() {
		return legacyErrors.New(233, "Short-window rate limit exceeded.", http.StatusTooManyRequests)
	}
	return nil
}

func (f *fakeCatalog) visible(id int, contentLimit string) bool {
	row, ok := f.rows[id]
	if !ok {
		return false
	}
	if contentLimit != "sfw" {
		return true
	}
	if f.sfwHidden[id] {
		return false
	}
	return client.CatalogItemToBrief(context.Background(), &row).ContentLimit == "sfw"
}

func (f *fakeCatalog) CatalogWorkExists(ctx context.Context, workID int) (bool, *legacyErrors.AppError) {
	rows, err := f.CatalogRowsByWorkIDs(ctx, []int{workID}, "", "all")
	if err != nil {
		return false, err
	}
	_, ok := rows[workID]
	return ok, nil
}

func (f *fakeCatalog) CatalogRowsByWorkIDs(_ context.Context, ids []int, _, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	f.mu.Lock()
	f.gotLimits = append(f.gotLimits, contentLimit)
	gate := f.rowsGate
	f.rowsGate = nil
	f.mu.Unlock()
	if gate != nil {
		f.rowsIn <- struct{}{}
		<-gate
	}
	out := map[int]client.CatalogWorkListItem{}
	for _, id := range ids {
		if f.omitIDs[id] {
			continue
		}
		if f.visible(id, contentLimit) {
			row := f.rows[id]
			if !client.CatalogItemRenderable(&row) {
				continue
			}
			out[id] = row
		}
	}
	return out, nil
}

func (f *fakeCatalog) CatalogTaxonomyList(_ context.Context, entity string, q url.Values) (*client.CatalogTaxonomyPage, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	items := []client.CatalogTaxonomyItem{}
	for _, it := range f.taxonomy[entity] {
		if q.Get("has_works") == "1" && it.WorkCount == 0 {
			continue
		}
		items = append(items, it)
	}
	return &client.CatalogTaxonomyPage{Items: items}, nil
}

func (f *fakeCatalog) CatalogEntitySearch(_ context.Context, searchType, _ string, _, _ int) ([]client.CatalogEntityHit, int64, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, 0, e
	}
	hits := f.hits[searchType]
	return hits, int64(len(hits)), nil
}

func (f *fakeCatalog) CatalogSexualTagIDs(_ context.Context, ids []int) map[int]bool {
	out := map[int]bool{}
	for _, id := range ids {
		if t, ok := f.tags[strconv.Itoa(id)]; ok && t.Sexual {
			out[id] = true
		}
	}
	return out
}

func (f *fakeCatalog) CatalogTag(_ context.Context, id string) (*client.CatalogTagDetail, bool, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, e
	}
	t, ok := f.tags[id]
	return t, ok, nil
}

func (f *fakeCatalog) CatalogLabel(_ context.Context, id string) (*client.CatalogLabelDetail, bool, int64, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, 0, e
	}
	if to, ok := f.moved[id]; ok {
		return &client.CatalogLabelDetail{}, false, to, nil
	}
	o, ok := f.labels[id]
	return o, ok, 0, nil
}

func (f *fakeCatalog) CatalogWorkDetail(_ context.Context, workID int) (*client.CatalogWorkDetail, bool, int64, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, 0, e
	}
	if to, ok := f.movedWorks[workID]; ok {
		return nil, false, to, nil
	}
	if d, ok := f.details[workID]; ok {
		return d, true, 0, nil
	}
	row, ok := f.rows[workID]
	if !ok || !client.CatalogItemRenderable(&row) {
		return nil, false, 0, nil
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return nil, false, 0, legacyErrors.New(233, err.Error(), http.StatusInternalServerError)
	}
	var d client.CatalogWorkDetail
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, false, 0, legacyErrors.New(233, err.Error(), http.StatusInternalServerError)
	}
	return &d, true, 0, nil
}

func (f *fakeCatalog) CatalogEntityMediaBatch(_ context.Context, entity string, ids []int64) (map[int64]client.CatalogEntityMedia, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	out := map[int64]client.CatalogEntityMedia{}
	for _, id := range ids {
		if m, ok := f.media[id]; ok {
			out[id] = m
		}
	}
	return out, nil
}

func (f *fakeCatalog) CatalogEngine(_ context.Context, id string) (*client.CatalogEngineDetail, bool, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, e
	}
	e, ok := f.engines[id]
	return e, ok, nil
}

func (f *fakeCatalog) CatalogSeries(_ context.Context, id string) (*client.CatalogSeriesDetail, bool, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, e
	}
	s, ok := f.series[id]
	return s, ok, nil
}

func (f *fakeCatalog) CatalogLabelRelationGraph(_ context.Context, id string) (*client.CatalogLabelRelationGraph, bool, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, e
	}
	g, ok := f.graphs[id]
	return g, ok, nil
}

func (f *fakeCatalog) LookupWikiLabel(_ context.Context, wikiID int) (int64, bool, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return 0, false, e
	}
	id, ok := f.wiki[wikiID]
	return id, ok, nil
}

func (f *fakeCatalog) CatalogNameDetail(_ context.Context, id int64, limit, offset int) (*client.CatalogName, bool, int64, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, 0, e
	}
	if id == geCreditName+1 {
		return nil, false, geCreditName, nil
	}
	n, ok := f.names[id]
	if !ok {
		return nil, false, 0, nil
	}
	cp := *n
	cp.Credits = n.Credits[min(offset, len(n.Credits)):min(offset+limit, len(n.Credits))]
	cp.NextOffset = nil
	if offset+limit < len(n.Credits) {
		next := offset + limit
		cp.NextOffset = &next
	}
	return &cp, true, 0, nil
}

func (f *fakeCatalog) CatalogCharacterDetail(_ context.Context, id int64, limit, offset int, withWorks bool) (*client.CatalogCharacter, bool, int64, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, false, 0, e
	}
	if id == geCharacter+1 {
		return nil, false, geCharacter, nil
	}
	c, ok := f.chars[id]
	if !ok {
		return nil, false, 0, nil
	}
	cp := *c
	cp.Works = nil
	cp.NextOffset = nil
	if withWorks {
		cp.Works = c.Works[min(offset, len(c.Works)):min(offset+limit, len(c.Works))]
		if offset+limit < len(c.Works) {
			next := offset + limit
			cp.NextOffset = &next
		}
	}
	return &cp, true, 0, nil
}

func (f *fakeCatalog) sortMembers(ids []int, catalogSort string) []int {
	out := slices.Clone(ids)
	if catalogSort == "" {
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := "", ""
		if r := f.rows[out[i]].ReleaseDate; r != nil {
			a = *r
		}
		if r := f.rows[out[j]].ReleaseDate; r != nil {
			b = *r
		}
		if catalogSort == "released_asc" {
			return a < b
		}
		return a > b
	})
	return out
}

func (f *fakeCatalog) CatalogMemberWorkIDs(_ context.Context, filter url.Values, isSFW bool, _ int) ([]int, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	var key string
	for _, k := range []string{"tag_id", "engine_id", "series_id"} {
		if v := filter.Get(k); v != "" {
			key = k + "=" + v
		}
	}
	limit := "all"
	if isSFW {
		limit = "sfw"
	}
	out := []int{}
	for _, id := range f.sortMembers(f.members[key], filter.Get("sort")) {
		if f.visible(id, limit) {
			out = append(out, id)
		}
	}
	return out, nil
}

func (f *fakeCatalog) CatalogLabelRollupMembers(_ context.Context, labelID, sort string, isSFW bool, _ int) ([]client.CatalogRollupMember, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	limit := "all"
	if isSFW {
		limit = "sfw"
	}
	byID := map[int]*client.CatalogLabelVia{}
	ids := []int{}
	for _, m := range f.rollup[labelID] {
		byID[m.workID] = m.via
		ids = append(ids, m.workID)
	}
	out := []client.CatalogRollupMember{}
	for _, id := range f.sortMembers(ids, sort) {
		if f.visible(id, limit) {
			out = append(out, client.CatalogRollupMember{WorkID: id, Via: byID[id]})
		}
	}
	return out, nil
}

func (f *fakeCatalog) CatalogRowsByCatalogIDs(_ context.Context, ids []int64, isSFW bool) (map[int64]client.CatalogWorkListItem, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	limit := "all"
	if isSFW {
		limit = "sfw"
	}
	out := map[int64]client.CatalogWorkListItem{}
	for _, id := range ids {
		if f.visible(int(id), limit) {
			out[id] = f.rows[int(id)]
		}
	}
	return out, nil
}

func (f *fakeCatalog) CatalogWorksSearch(_ context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	f.mu.Lock()
	f.searched = append(f.searched, maps.Clone(q))
	f.mu.Unlock()
	limit := q.Get("content_limit")
	var ids []int
	switch {
	case q.Get("q") != "":
		kw := strings.ToLower(q.Get("q"))
		for _, w := range f.works {
			if strings.Contains(strings.ToLower(w.name), kw) {
				ids = append(ids, w.id)
			}
		}
		if len(ids) == 0 {
			for id, row := range f.rows {
				if strings.Contains(strings.ToLower(row.DisplayName), kw) {
					ids = append(ids, id)
				}
			}
		}
	case q.Get("series_id") != "":
		sid, _ := strconv.Atoi(q.Get("series_id"))
		ids = f.sortMembers(f.seriesMem[sid], q.Get("sort"))
	case q.Get("tag_id") != "":
		want := []int{}
		for _, raw := range strings.Split(q.Get("tag_id"), ",") {
			n, _ := strconv.Atoi(raw)
			want = append(want, n)
		}
		for _, w := range f.works {
			all := true
			for _, tag := range want {
				if !slices.Contains(w.tags, tag) {
					all = false
				}
			}
			if all {
				ids = append(ids, w.id)
			}
		}
		ids = f.sortMembers(ids, "released_desc")
	default:
		for _, w := range f.works {
			ids = append(ids, w.id)
		}
		if len(ids) == 0 {
			for id := range f.rows {
				ids = append(ids, id)
			}
			sort.Ints(ids)
		}
	}
	after, before := q.Get("released_after"), q.Get("released_before")
	if after != "" || before != "" {
		kept := ids[:0]
		for _, id := range ids {
			d := ""
			if row := f.rows[id]; row.ReleaseDate != nil {
				d = *row.ReleaseDate
			}
			if after != "" && d < after {
				continue
			}
			if before != "" && d > before {
				continue
			}
			kept = append(kept, id)
		}
		ids = kept
	}
	visible := []int{}
	for _, id := range ids {
		if f.visible(id, limit) {
			visible = append(visible, id)
		}
	}
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("limit"))
	page, size = max(page, 1), max(size, 1)
	start := min((page-1)*size, len(visible))
	res := &client.CatalogWorksPage{Total: int64(len(visible))}
	for _, id := range visible[start:min(start+size, len(visible))] {
		res.Items = append(res.Items, f.rows[id])
	}
	return res, nil
}

func (f *fakeCatalog) CatalogCalendar(_ context.Context, bucket string, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError) {
	if e := f.err(); e != nil {
		return nil, e
	}
	f.mu.Lock()
	f.calQ = append(f.calQ, maps.Clone(q))
	f.calBucket = append(f.calBucket, bucket)
	f.mu.Unlock()
	key := bucket
	switch bucket {
	case "":
		key = q.Get("month")
	case "/pending":
		key = "pending:" + q.Get("year")
	case "/tba":
		key = "tba"
	}
	if f.calFail[key] {
		return nil, legacyErrors.New(233, "calendar down", http.StatusInternalServerError)
	}
	items := f.calItems[key]
	start := 0
	if cur := q.Get("cursor"); strings.HasPrefix(cur, "p") {
		start, _ = strconv.Atoi(cur[1:])
	}
	limit := 100
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = n
	}
	end := min(start+limit, len(items))
	if start > len(items) {
		start = len(items)
		end = start
	}
	page := &client.CatalogWorksPage{
		Items: append([]client.CatalogWorkListItem{}, items[start:end]...),
		Month: q.Get("month"),
		Year:  q.Get("year"),
		Count: int64(len(items)),
		Total: int64(len(items)),
	}
	page.Meta.Today = f.calToday[key]
	page.Meta.MinMonth = f.calMin[key]
	page.Meta.MaxMonth = f.calMax[key]
	if f.calHasPrev(key) {
		v := true
		page.Meta.HasPrev = &v
	}
	if f.calHasNext(key) {
		v := true
		page.Meta.HasNext = &v
	}
	if end < len(items) {
		page.NextCursor = "p" + strconv.Itoa(end)
	}
	return page, nil
}

func (f *fakeCatalog) calHasPrev(key string) bool {
	return f.calMin[key] != "" && f.calMin[key] < key
}

func (f *fakeCatalog) calHasNext(key string) bool {
	return f.calMax[key] != "" && f.calMax[key] > key
}

var _ entityapiv1.Catalog = (*fakeCatalog)(nil)
var _ calendarapiv1.Catalog = (*fakeCatalog)(nil)

type geFix struct {
	app  *App
	db   *gorm.DB
	cat  *fakeCatalog
	spec *specConformance
}

func newGEFix(t *testing.T) *geFix {
	t.Helper()
	db := testdb.Open(t)
	cat := &fakeCatalog{
		rows: map[int]client.CatalogWorkListItem{}, works: geWorks(),
		taxonomy: map[string][]client.CatalogTaxonomyItem{}, hits: map[string][]client.CatalogEntityHit{},
		tags: map[string]*client.CatalogTagDetail{}, labels: map[string]*client.CatalogLabelDetail{},
		moved: map[string]int64{strconv.Itoa(geMerged): geCompany}, engines: map[string]*client.CatalogEngineDetail{},
		series: map[string]*client.CatalogSeriesDetail{}, graphs: map[string]*client.CatalogLabelRelationGraph{},
		wiki: map[int]int64{42: geCompany}, names: map[int64]*client.CatalogName{}, chars: map[int64]*client.CatalogCharacter{},
		members: map[string][]int{}, rollup: map[string][]geMember{}, seriesMem: map[int][]int{},
	}
	f := &geFix{db: db, cat: cat}
	f.seedCatalog(t)
	f.seedLocal(t)
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = geCDN
	f.app = &App{Fiber: newFiber(), Config: cfg, DB: db, GalgameEntityV1: entityapiv1.New(cat, db, geCDN)}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	return f
}

func (f *geFix) seedCatalog(t *testing.T) {
	t.Helper()
	c := f.cat
	ids := []int{}
	for _, w := range c.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		c.rows[w.id] = row
		ids = append(ids, w.id)
	}
	c.members["tag_id="+strconv.Itoa(geTagMain)] = ids
	c.members["engine_id="+strconv.Itoa(geEngine)] = ids[:3]
	c.members["series_id="+strconv.Itoa(geSeries)] = ids[:2]
	c.seriesMem[geSeries] = ids[:2]
	c.seriesMem[geSeriesNSFW] = []int{ids[2]}
	via := &client.CatalogLabelVia{ID: geImprint, DisplayName: "Imprint Label"}
	c.rollup[strconv.Itoa(geCompany)] = []geMember{{ids[0], nil}, {ids[1], via}, {ids[3], nil}, {ids[5], via}}

	var tagItems []client.CatalogTaxonomyItem
	decodeInto(t, `[
		{"id":5101,"display_name":"Plot twist","localized":{"zh-Hans":{"value":"剧情反转"}},"kind":"content","tier":"core","work_count":40,"sexual":false},
		{"id":5102,"display_name":"Adult tag","kind":"content","tier":"core","work_count":40,"sexual":true},
		{"id":5103,"display_name":"Hidden tag","kind":"meta","tier":"hidden","work_count":90,"sexual":false},
		{"id":5105,"display_name":"Meta tag","kind":"meta","tier":"core","work_count":40,"sexual":false},
		{"id":5106,"display_name":"Rare tag","kind":"content","tier":"core","work_count":2,"sexual":false},
		{"id":5104,"display_name":"Empty tag","kind":"content","tier":"core","work_count":0,"sexual":false}]`, &tagItems)
	c.taxonomy["tags"] = tagItems
	for _, raw := range []string{
		`{"id":5101,"display_name":"Plot twist","localized":{"zh-Hans":{"value":"剧情反转"}},"kind":"content","tier":"core","work_count":40,"sexual":false,"intros":[{"lang":"zh-Hans","intro":"反转","source":"vndb"},{"lang":"en","intro":"  "}]}`,
		`{"id":5102,"display_name":"Adult tag","kind":"content","tier":"core","work_count":40,"sexual":true}`,
		`{"id":5103,"display_name":"Hidden tag","kind":"meta","tier":"hidden","work_count":90,"sexual":false}`,
		`{"id":5104,"display_name":"Empty tag","kind":"content","tier":"core","work_count":0,"sexual":false}`,
		`{"id":5107,"display_name":"Empty adult","kind":"content","tier":"core","work_count":0,"sexual":true}`,
	} {
		var d client.CatalogTagDetail
		decodeInto(t, raw, &d)
		c.tags[strconv.FormatInt(d.ID, 10)] = &d
	}
	c.hits["tags"] = decodeHits(t, `[
		{"id":5103,"display_name":"Hidden tag","tier":"hidden","kind":"meta"},
		{"id":5102,"display_name":"Adult tag","tier":"core","kind":"content"},
		{"id":5104,"display_name":"Empty tag","tier":"core","kind":"content"},
		{"id":5107,"display_name":"Empty adult","tier":"core","kind":"content"},
		{"id":5101,"display_name":"Plot twist","tier":"core","kind":"content"}]`)

	c.taxonomy["labels"] = decodeTax(t, `[
		{"id":6101,"display_name":"Maker Brand","latin":"maker brand","kind":"game_brand","work_count":4,"logo_hash":"`+geHash(61)+`","aliases":[{"value":"MB"},{"value":"Maker Brand"}]},
		{"id":6102,"display_name":"Imprint Label","kind":"game_brand","work_count":4},
		{"id":6104,"display_name":"Circle One","kind":"doujin_circle","work_count":9},
		{"id":6105,"display_name":"Odd Kind","kind":"conglomerate","work_count":50}]`)
	c.hits["labels"] = decodeHits(t, `[
		{"id":6106,"display_name":"New Circle","kind":"doujin_circle"},
		{"id":6101,"display_name":"Maker Brand","kind":"game_brand"}]`)
	var label client.CatalogLabelDetail
	decodeInto(t, `{"id":6101,"display_name":"Maker Brand","latin":"maker brand","kind":"game_brand","lang":"ja","work_count":4,
		"logo_hash":"`+geHash(61)+`","aliases":[{"value":"MB"}],
		"links":[{"source":"official_site","url":"https://maker.example"},{"source":"dlsite","url":""},{"source":"Bad Source!","url":"https://x.example"}],
		"intros":[{"lang":"ja","intro":"メーカー","machine":false}]}`, &label)
	c.labels[strconv.Itoa(geCompany)] = &label
	var graph client.CatalogLabelRelationGraph
	decodeInto(t, `{"nodes":[{"id":6101,"display_name":"Maker Brand","work_count":4},{"id":6102,"name":"Imprint Label","work_count":2}],
		"edges":[{"from":6101,"to":6102,"relation":"imprint"},{"from":6102,"to":6101,"relation":"imprint_of"},{"from":6101,"to":6102,"relation":"sister"}]}`, &graph)
	c.graphs[strconv.Itoa(geCompany)] = &graph

	c.taxonomy["engines"] = decodeTax(t, `[
		{"id":7101,"display_name":"KiriKiri","description":"A scripting engine.","work_count":3,"aliases":[{"value":"krkr"}]},
		{"id":7102,"display_name":"Ren'Py","work_count":3}]`)
	var engine client.CatalogEngineDetail
	decodeInto(t, `{"id":7101,"display_name":"KiriKiri","description":"A scripting engine.","work_count":3,"aliases":[{"value":"krkr"}]}`, &engine)
	c.engines[strconv.Itoa(geEngine)] = &engine

	c.taxonomy["series"] = decodeTax(t, `[
		{"id":8101,"display_name":"Saga","work_count":2,"has_nsfw":false},
		{"id":8102,"display_name":"Night Saga","work_count":1,"has_nsfw":true},
		{"id":8103,"display_name":"Unknown Saga","work_count":0}]`)
	var series client.CatalogSeriesDetail
	decodeInto(t, `{"id":8101,"display_name":"Saga","work_count":2,"has_nsfw":false,"intros":[{"lang":"zh-Hans","intro":"系列"}]}`, &series)
	c.series[strconv.Itoa(geSeries)] = &series

	c.hits["names"] = decodeHits(t, `[{"id":9101,"display_name":"瀬戸","latin":"Seto"}]`)
	c.hits["characters"] = decodeHits(t, `[{"id":9201,"display_name":"夏帆","latin":"Kaho"}]`)
	c.media = map[int64]client.CatalogEntityMedia{
		9201: {Image: "https://image.other.example/aa/bb/" + geHash(9201) + ".webp", WorkCount: 3},
	}
	var name client.CatalogName
	decodeInto(t, fmt.Sprintf(`{"id":9101,"display_name":"瀬戸","latin":"Seto","lang":"ja","gender":2,"birth_m":4,"birth_d":1,
		"refs":[{"source":"vndb","external_id":"2099"},{"source":"dlsite","external_id":"x"}],
		"siblings":[{"id":9102,"display_name":"せと","lang":"ja"}],
		"credits":[
			{"work":{"id":%d},"roles":[{"role_key":"scenario","role_name":"剧本"},{"role_key":"other-staff","role_name":"其他"}]},
			{"work":{"id":%d},"roles":[{"role_key":"voice-actor","role_name":"声优","character_id":9201,"character":"夏帆"},{"role_key":"voice-actor","role_name":"声优","character":"モブ"}]},
			{"work":{"id":%d},"roles":[{"role_key":"music","role_name":"音乐"}]},
			{"work":{"id":%d},"roles":[{"role_key":"scenario","role_name":"剧本"}]}]}`,
		geWorkMin, geWorkMin+1, geWorkMin+2, geWorkMin+3), &name)
	c.names[geCreditName] = &name
	var char client.CatalogCharacter
	decodeInto(t, fmt.Sprintf(`{"id":9201,"display_name":"夏帆","latin":"Kaho","lang":"ja","image":%q,
		"intros":[{"lang":"ja","intro":"養母","source":"erogamescape"}],
		"traits":[{"id":1,"display_name":"Blonde","name_zh":"金发","group":"Hair","group_zh":"毛发","spoiler":0},
			{"id":2,"display_name":"Adult trait","group":"Body","spoiler":2,"sexual":true},
			{"id":3,"display_name":"Liar","group":"Role","spoiler":1,"lie":true}],
		"refs":[{"source":"vndb","external_id":"c9990"}],
		"works":[{"work":{"id":%d},"voices":[{"id":9101,"display_name":"瀬戸","lang":"ja"}]},{"work":{"id":%d},"voices":[]},{"work":{"id":%d},"voices":[]}]}`,
		geImageURL(9201), geWorkMin, geWorkMin+2, geWorkMin+5), &char)
	c.chars[geCharacter] = &char
}

func decodeTax(t *testing.T, raw string) []client.CatalogTaxonomyItem {
	t.Helper()
	var items []client.CatalogTaxonomyItem
	decodeInto(t, raw, &items)
	return items
}

func decodeHits(t *testing.T, raw string) []client.CatalogEntityHit {
	t.Helper()
	var items []client.CatalogEntityHit
	decodeInto(t, raw, &items)
	return items
}

func (f *geFix) seedLocal(t *testing.T) {
	t.Helper()
	wipe := func() {
		for _, q := range []string{
			`DELETE FROM galgame_rating WHERE work_id BETWEEN ? AND ?`,
			`DELETE FROM galgame_resource WHERE work_id BETWEEN ? AND ?`,
			`DELETE FROM galgame WHERE id BETWEEN ? AND ?`,
		} {
			if err := f.db.Exec(q, geWorkMin, geWorkMax).Error; err != nil {
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
		run(`INSERT INTO galgame (id, view, created, updated, like_count, published, content_limit, resource_update_time) VALUES (?, ?, ?, ?, ?, true, ?, ?)`,
			w.id, w.view, w.created, w.created, w.view*2, w.limit, w.created.Add(24*time.Hour))
		plat, _ := json.Marshal(w.platforms)
		lang, _ := json.Marshal(w.languages)
		run(`INSERT INTO galgame_resource (work_id, user_id, type, platform, language, platforms, languages, updated) VALUES (?, ?, 'game', ?, 'zh-cn', ?::jsonb, ?::jsonb, now())`,
			w.id, geUser, w.legacyPlat, string(plat), string(lang))
		gameTypes := w.gameTypes
		if gameTypes == nil {
			gameTypes = []string{}
		}
		for i, overall := range w.ratings {
			gt, _ := json.Marshal(gameTypes)
			run(`INSERT INTO galgame_rating (work_id, user_id, recommend, overall, galgame_type, updated) VALUES (?, ?, 'yes', ?, ?::jsonb, now())`,
				w.id, geUser+i, overall, string(gt))
		}
		if len(w.ratings) == 0 && len(w.gameTypes) > 0 {
			gt, _ := json.Marshal(w.gameTypes)
			run(`INSERT INTO galgame_rating (work_id, user_id, recommend, overall, galgame_type, updated) VALUES (?, ?, 'yes', 6, ?::jsonb, now())`,
				w.id, geUser, string(gt))
		}
	}
}

func (f *geFix) get(t *testing.T, rawURL, specPath string) (*http.Response, map[string]any) {
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

func geItemIDs(body map[string]any) []string {
	items, _ := body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, raw := range items {
		it := raw.(map[string]any)
		if w, ok := it["work_summary"].(map[string]any); ok {
			it = w
		}
		out = append(out, it["id"].(string))
	}
	return out
}

func wid(offset int) string {
	return strconv.Itoa(geWorkMin + offset)
}
