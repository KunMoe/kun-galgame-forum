package client

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestRewriteV2JSON_CoverSlotsDoNotRecurse(t *testing.T) {
	raw := []byte(`{"object":"work","id":"4242","display_name":"Kun","cover":{"url":"https://cdn.example/p.webp","hash":"aa","width":600,"height":800},"banner":{"url":"https://cdn.example/b.webp","width":1280,"height":720,"thumbhash":"B"},"covers":[{"url":"https://cdn.example/p.webp","kind":"main"}],"cover_slots":{"portrait":{"url":"https://cdn.example/p.webp","width":600,"height":800},"banner":{"url":"https://cdn.example/b.webp","width":1280,"height":720,"thumbhash":"B"}}}`)
	out := rewriteV2JSON(raw, "")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("rewrite produced invalid json: %v\n%s", err, out)
	}
	slots, _ := got["cover_slots"].(map[string]any)
	if slots == nil {
		t.Fatalf("cover_slots missing: %s", out)
	}
	if _, nested := slots["cover_slots"]; nested {
		t.Fatalf("cover_slots was rewritten as a work: %s", out)
	}
	covers, ok := got["covers"].([]any)
	if !ok || len(covers) != 1 {
		t.Fatalf("covers array was overwritten: %s", out)
	}
}

func TestRewriteV2JSON_CoverAndBannerBecomeSlots(t *testing.T) {
	raw := []byte(`{"object":"work","id":"1","display_name":"Kun","cover":{"url":"https://cdn.example/p.webp","width":600,"height":800,"thumbhash":"P"},"banner":{"url":"https://cdn.example/b.webp","width":1280,"height":720,"thumbhash":"B"}}`)
	out := rewriteV2JSON(raw, "")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("rewrite produced invalid json: %v", err)
	}
	if _, ok := got["cover"].(string); !ok {
		t.Fatalf("cover Image was not flattened to a URL: %s", out)
	}
	slots, _ := got["cover_slots"].(map[string]any)
	banner, _ := slots["banner"].(map[string]any)
	if banner["url"] != "https://cdn.example/b.webp" || banner["thumbhash"] != "B" {
		t.Fatalf("banner slot = %v", banner)
	}
}

func TestV2CatalogQuery_EmptySearchGetsFacets(t *testing.T) {
	q := v2CatalogQuery("/catalog/works/search", url.Values{"limit": {"24"}})
	if q.Get("facets") != "olang,tag_id" {
		t.Fatalf("facets = %q, want olang,tag_id so empty browse uses the search total", q.Get("facets"))
	}
	if q.Get("include_total") != "true" {
		t.Fatalf("include_total = %q", q.Get("include_total"))
	}
}

func TestV2CatalogQuery_PageBecomesCursor(t *testing.T) {
	q := v2CatalogQuery("/catalog/works/search", url.Values{"q": {"kun"}, "page": {"2"}})
	if q.Get("page") != "" {
		t.Fatalf("page leaked: %q", q.Get("page"))
	}
	if !strings.HasPrefix(q.Get("cursor"), "cur_") {
		t.Fatalf("cursor = %q, want cur_…", q.Get("cursor"))
	}
}

// The entity search face grew a cursor in catalog 2.4.0 and reuses the works
// lane's page-number encoding, so the forum's paginator addresses it the same
// way. Before that it answered exactly one page and the extra rows were
// unreachable.
func TestV2CatalogQuery_SearchPageBecomesCursor(t *testing.T) {
	q := v2CatalogQuery("/catalog/search", url.Values{
		"type": {"characters"}, "q": {"kun"}, "page": {"3"},
	})
	if q.Get("page") != "" {
		t.Fatalf("page leaked: %q", q.Get("page"))
	}
	if got, want := q.Get("cursor"), encodePageCursor(3); got != want {
		t.Fatalf("cursor = %q, want %q", got, want)
	}
	if q.Get("object") != "character" {
		t.Fatalf("object = %q, want character", q.Get("object"))
	}
}

// Page 1 must not carry a cursor: catalog rejects anything that is not a cur_
// token, and "the first page" has no token to send.
func TestV2CatalogQuery_SearchFirstPageSendsNoCursor(t *testing.T) {
	q := v2CatalogQuery("/catalog/search", url.Values{
		"type": {"tags"}, "q": {"kun"}, "page": {"1"},
	})
	if q.Get("cursor") != "" || q.Get("page") != "" {
		t.Fatalf("page 1 sent cursor=%q page=%q", q.Get("cursor"), q.Get("page"))
	}
}

// Its own variables, never the application's: internal/testdb/rule_test.go
// fails the whole suite on os.Getenv("KUN_ in a test, so that a smoke run can
// never silently inherit a live catalog from a stray .env.
//
// This is the only check that catches the failure this package keeps having.
// The 2026-08 cutover compiled and its suite was green because it rewrote the
// assertions alongside the code; nothing failed, the pages just went blank. So
// every face the forum reads gets driven here against a real catalog, and each
// assertion names a field that a page actually renders.
func liveClient(t *testing.T) *GalgameClient {
	t.Helper()
	base, key := os.Getenv("SMOKE_CATALOG_BASE"), os.Getenv("SMOKE_CATALOG_KEY")
	if base == "" || key == "" || os.Getenv("SMOKE_CATALOG_V2") == "" {
		t.Skip("set SMOKE_CATALOG_V2=1 with SMOKE_CATALOG_BASE and SMOKE_CATALOG_KEY")
	}
	return New(base, key, os.Getenv("SMOKE_IMAGE_PUBLIC_BASE_URL"))
}

func TestLiveV2Works(t *testing.T) {
	c, ctx := liveClient(t), context.Background()

	q := OpenPopulation(url.Values{"limit": {"3"}, "include": {"names,covers,refs,labels"}})
	page, appErr := c.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		t.Fatalf("CatalogWorksSearch: %v", appErr)
	}
	if len(page.Items) == 0 {
		t.Fatal("empty works page")
	}
	if page.Total == 0 {
		t.Error("total = 0 — include_total did not reach the registry lane")
	}
	b := CatalogItemToBrief(ctx, &page.Items[0])
	if b.ID <= 0 || b.Name == "" || b.EffectiveBannerURL == "" {
		t.Fatalf("card fields missing: %+v", b)
	}

	d, found, appErr := c.CatalogWorkDetail(ctx, b.ID)
	if appErr != nil || !found || d == nil {
		t.Fatalf("CatalogWorkDetail(%d) = (%v, %v)", b.ID, appErr, found)
	}
	full := CatalogDetailToFull(ctx, d, b.ID)
	if full.Name == "" {
		t.Error("detail name empty")
	}
	if full.EffectiveBannerURL == "" {
		t.Error("detail hero empty — the cover slots did not survive the rewrite")
	}
	if len(full.Official) == 0 {
		t.Error("no 制作方 rows — the companies block decoded to nothing")
	}
	// A tag row the forum drops when canonical_id is 0, which is what v2 naming
	// the same column `id` would produce.
	if len(full.Tag) == 0 {
		t.Error("no tag rows — a canonical_id of 0 drops every one of them")
	}
	for i := range full.Tag {
		if full.Tag[i].Tag.ID == 0 || full.Tag[i].Tag.Name == "" {
			t.Errorf("tag %d = %+v, want an id and a name", i, full.Tag[i].Tag)
			break
		}
	}
}

func TestLiveV2DetailFaces(t *testing.T) {
	c, ctx := liveClient(t), context.Background()

	labels, appErr := c.CatalogTaxonomyList(ctx, "labels", OpenPopulation(url.Values{"has_works": {"1"}, "limit": {"3"}}))
	if appErr != nil || len(labels.Items) == 0 {
		t.Fatalf("company list: err=%v items=%d total=%d", appErr, len(labels.Items), labels.Total)
	}
	id := strconv.FormatInt(labels.Items[0].ID, 10)

	label, found, _, appErr := c.CatalogLabel(ctx, id)
	if appErr != nil || !found {
		t.Fatalf("CatalogLabel(%s) = (%v, %v)", id, appErr, found)
	}
	if label.DisplayName == "" || label.WorkCount == 0 {
		t.Errorf("company %s = %+v, want a name and a work count", id, label)
	}

	graph, found, appErr := c.CatalogLabelRelationGraph(ctx, id)
	if appErr != nil {
		t.Fatalf("CatalogLabelRelationGraph(%s): %v", id, appErr)
	}
	if found && len(graph.Nodes) == 0 {
		t.Errorf("company %s graph answered with no nodes", id)
	}

	tags, appErr := c.CatalogTaxonomyList(ctx, "tags", OpenPopulation(url.Values{"has_works": {"1"}, "limit": {"3"}}))
	if appErr != nil || len(tags.Items) == 0 {
		t.Fatalf("tag list: err=%v items=%d total=%d", appErr, len(tags.Items), tags.Total)
	}
	tag, found, appErr := c.CatalogTag(ctx, strconv.FormatInt(tags.Items[0].ID, 10))
	if appErr != nil || !found || tag.Label() == "" {
		t.Fatalf("CatalogTag = (%v, %v, %q)", appErr, found, tag.Label())
	}

	series, appErr := c.CatalogTaxonomyList(ctx, "series", OpenPopulation(url.Values{"limit": {"3"}}))
	if appErr != nil || len(series.Items) == 0 {
		t.Fatalf("series list: err=%v items=%d", appErr, len(series.Items))
	}

	// The staff page: the credits block lives on its own sub-face and is spliced
	// back in, inheriting the parent's population gate.
	hits, _, appErr := c.CatalogEntitySearch(ctx, "names", "田村", 1, 3)
	if appErr != nil {
		t.Fatalf("CatalogEntitySearch names: %v", appErr)
	}
	if len(hits) == 0 {
		t.Fatal("staff search found nothing")
	}
	if hits[0].EntityType != "name" {
		t.Errorf("hit entity_type = %q, want name", hits[0].EntityType)
	}
	name, found, _, appErr := c.CatalogNameDetail(ctx, hits[0].ID, 50, 0)
	if appErr != nil || !found {
		t.Fatalf("CatalogNameDetail(%d) = (%v, %v)", hits[0].ID, appErr, found)
	}
	if CatalogEntityName(ctx, name.Localized, name.DisplayName, name.Latin) == "" {
		t.Error("staff name empty")
	}
	if len(name.Credits) == 0 {
		t.Errorf("staff %d has no credits — the sub-face splice produced nothing", hits[0].ID)
	}

	chars, _, appErr := c.CatalogEntitySearch(ctx, "characters", "a", 1, 3)
	if appErr != nil || len(chars) == 0 {
		t.Fatalf("character search: err=%v hits=%d", appErr, len(chars))
	}
	ch, found, _, appErr := c.CatalogCharacterDetail(ctx, chars[0].ID, 50, 0, true)
	if appErr != nil || !found {
		t.Fatalf("CatalogCharacterDetail(%d) = (%v, %v)", chars[0].ID, appErr, found)
	}
	if CatalogEntityName(ctx, ch.Localized, ch.DisplayName, ch.Latin) == "" {
		t.Error("character name empty")
	}
	if ch.Image == "" && ch.Figure == "" {
		t.Errorf("character %d has neither art — the image objects did not fold to URLs", chars[0].ID)
	}
	for i := range ch.Traits {
		if ch.Traits[i].LocalName() == "" {
			t.Errorf("trait %d renders as an empty chip: %+v", i, ch.Traits[i])
			break
		}
	}
}

// The 资料库 tab paginates every family through this one face. A second page
// that repeats the first would look like it worked and quietly show the same
// rows forever, so identity of the two pages is the assertion.
func TestLiveV2EntitySearchPages(t *testing.T) {
	c, ctx := liveClient(t), context.Background()

	first, total, appErr := c.CatalogEntitySearch(ctx, "characters", "a", 1, 5)
	if appErr != nil {
		t.Fatalf("page 1: %v", appErr)
	}
	if total <= 5 || len(first) != 5 {
		t.Fatalf("page 1: total=%d items=%d — too few rows to page", total, len(first))
	}
	second, _, appErr := c.CatalogEntitySearch(ctx, "characters", "a", 2, 5)
	if appErr != nil {
		t.Fatalf("page 2: %v", appErr)
	}
	seen := map[int64]bool{}
	for _, h := range first {
		seen[h.ID] = true
	}
	for _, h := range second {
		if seen[h.ID] {
			t.Fatalf("page 2 repeats id %d from page 1", h.ID)
		}
	}

	// series and engines joined the closed object vocabulary in catalog 2.4.0.
	// Their indices are built only by the nightly reindex, so a deployment that
	// has never run it answers an empty page rather than an error — which is
	// why this asserts the call, not the row count.
	for _, family := range []string{"series", "engines"} {
		if _, _, appErr := c.CatalogEntitySearch(ctx, family, "a", 1, 5); appErr != nil {
			t.Errorf("%s search: %v", family, appErr)
		}
	}
}

func TestLiveV2Calendar(t *testing.T) {
	c, ctx := liveClient(t), context.Background()
	for _, bucket := range []string{"", "/pending", "/tba"} {
		page, appErr := c.CatalogCalendar(ctx, bucket, OpenPopulation(url.Values{"limit": {"3"}}))
		if appErr != nil {
			t.Errorf("CatalogCalendar(%q): %v", bucket, appErr)
			continue
		}
		if len(page.Items) == 0 {
			t.Errorf("CatalogCalendar(%q) answered no rows", bucket)
		}
	}
}

// The gid bridge is the one lane that was already on v2, and it is how every
// legacy forum id finds its catalog work. HydrateCardsByIDs drops an id it
// cannot resolve, so a break here is a short page, never an error.
func TestLiveV2GIDBridge(t *testing.T) {
	c, ctx := liveClient(t), context.Background()
	raw := os.Getenv("SMOKE_GIDS")
	if raw == "" {
		t.Skip("set SMOKE_GIDS to a comma-separated list of published forum gids")
	}
	var gids []int
	for _, part := range strings.Split(raw, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			gids = append(gids, n)
		}
	}
	ids, appErr := c.CatalogWorkIDs(ctx, gids)
	if appErr != nil {
		t.Fatalf("CatalogWorkIDs: %v", appErr)
	}
	// A handful of unresolved gids are forum rows whose catalog work was deleted;
	// they are data drift and always have been. A PROPORTION of them is the read
	// regression this test exists for — the 2026-08 olang bug lost 1.6% of the
	// catalogue this way, and because HydrateCardsByIDs skips what it cannot
	// resolve, the only symptom was pages that came back one row short.
	if lost := len(gids) - len(ids); lost > 0 {
		var sample []int
		for _, gid := range gids {
			if _, ok := ids[gid]; !ok && len(sample) < 20 {
				sample = append(sample, gid)
			}
		}
		t.Logf("bridge resolved %d of %d gids; unresolved: %v", len(ids), len(gids), sample)
		if lost*200 > len(gids) {
			t.Errorf("%d of %d gids (%.1f%%) resolve to nothing — that is a lane fault, not drift",
				lost, len(gids), float64(lost)*100/float64(len(gids)))
		}
	}
	rows, appErr := c.CatalogRowsByGIDs(ctx, gids, "names,covers,refs,labels", "all")
	if appErr != nil {
		t.Fatalf("CatalogRowsByGIDs: %v", appErr)
	}
	// A shortfall here is normally the forum's own gates doing their job: catalog
	// hides a work the forum still has published, or the claim's site_work_id no
	// longer round-trips to the gid the anchor names. Those are drift, not a read
	// regression. Only a LIVE row that failed to hydrate means the lane is broken,
	// and that is the case worth a red test — HydrateCardsByIDs skips what it
	// cannot resolve, so upstream breakage shows up as a short page, never an error.
	if len(rows) != len(gids) {
		var missingIDs []int64
		byCatalogID := map[int64]int{}
		for _, gid := range gids {
			if _, ok := rows[gid]; !ok && ids[gid] != 0 {
				missingIDs = append(missingIDs, ids[gid])
				byCatalogID[ids[gid]] = gid
			}
		}
		explained, dropped := 0, map[int64]string{}
		if len(missingIDs) > 0 {
			back, appErr := c.CatalogRowsByCatalogIDs(ctx, missingIDs, false)
			if appErr != nil {
				t.Fatalf("re-reading the missing rows: %v", appErr)
			}
			for id := range back {
				row := back[id]
				switch {
				case row.Claim == nil:
					dropped[id] = "no claim"
				case row.Claim.State == "live":
					dropped[id] = "LIVE and still lost"
				default:
					explained++
				}
			}
		}
		t.Logf("hydrated %d of %d gids; %d explained by claim state, %d with no catalog row",
			len(rows), len(gids), explained, len(gids)-len(ids))
		for id, why := range dropped {
			t.Errorf("gid %d -> catalog %d: %s", byCatalogID[id], id, why)
		}
	}
	for gid := range rows {
		row := rows[gid]
		b := CatalogItemToBrief(ctx, &row)
		if b.Name == "" {
			t.Errorf("gid %d hydrated with no name", gid)
			break
		}
	}
}

func TestV2CatalogPath(t *testing.T) {
	cases := map[string]string{
		"/catalog/works/search":            "/v2/catalog/works",
		"/catalog/works/12":                "/v2/catalog/works/12",
		"/catalog/labels/3":                "/v2/catalog/companies/3",
		"/catalog/labels/3/relation-graph": "/v2/catalog/companies/3/graph",
		"/catalog/names/9":                 "/v2/catalog/credit-names/9",
		"/catalog/calendar/pending":        "/v2/catalog/calendar",
	}
	for in, want := range cases {
		if got := v2CatalogPath(in); got != want {
			t.Errorf("v2CatalogPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// The mirror channel replaced a nightly full sweep, so its two silent failure
// modes are the ones to drive live: a cursor that does not advance re-reads page
// one forever, and a hydrated row keyed by its catalog id writes one game's
// display verdict onto another game's row.
func TestLiveMirrorChannel(t *testing.T) {
	c, ctx := liveClient(t), context.Background()

	pages := 5
	if n, err := strconv.Atoi(os.Getenv("SMOKE_MIRROR_PAGES")); err == nil && n > 0 {
		pages = n
	}

	cursor := ""
	seen := map[int64]bool{}
	limits := map[int]string{}
	var rows, gone int
	for range pages {
		page, appErr := c.CatalogChanges(ctx, cursor, CatalogChangesLimit)
		if appErr != nil {
			t.Fatalf("CatalogChanges(%q): %v", cursor, appErr)
		}
		if len(page.Items) == 0 {
			t.Fatalf("empty page at cursor %q", cursor)
		}
		ids := make([]int64, 0, len(page.Items))
		for _, it := range page.Items {
			if seen[it.ID] {
				t.Fatalf("id %d served twice — the cursor is inclusive, "+
					"and a mirror that re-reads its own page never reaches the tail", it.ID)
			}
			seen[it.ID] = true
			rows++
			if it.Gone {
				gone++
				continue
			}
			ids = append(ids, it.ID)
		}
		got, appErr := c.MirrorByCatalogIDs(ctx, ids)
		if appErr != nil {
			t.Fatalf("MirrorByCatalogIDs: %v", appErr)
		}
		for gid, row := range got {
			limits[gid] = row.ContentLimit
		}
		if page.NextCursor == "" {
			break
		}
		if page.NextCursor == cursor {
			t.Fatalf("next_cursor did not move off %q", cursor)
		}
		cursor = page.NextCursor
	}

	if len(limits) == 0 {
		t.Fatalf("read %d changed works and matched none to a forum row — "+
			"the claim block is how a catalog id becomes a gid", rows)
	}
	for gid, limit := range limits {
		if limit != "sfw" && limit != "nsfw" {
			t.Fatalf("gid %d carries verdict %q", gid, limit)
		}
	}
	t.Logf("changes: rows=%d gone=%d matched=%d", rows, gone, len(limits))
	for gid, limit := range limits {
		t.Logf("gid=%d limit=%s", gid, limit)
	}
}
