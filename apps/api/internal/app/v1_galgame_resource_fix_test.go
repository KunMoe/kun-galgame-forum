package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/trust/gate"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/linkcheck"
)

const (
	g3HiddenA = 930004101
	g3HiddenB = 930004102

	g3ResTied    = 930004201
	g3ResMain    = 930004210
	g3ResExpired = 930004211
	g3ResCount0  = 930004212
	g3ResNSFW    = 930004213
	g3ResBanned  = 930004214
	g3ResGone    = 930004299

	g3WorkSFW     = 943000001
	g3WorkNSFW    = 943000002
	g3WorkUnpub   = 943000003
	g3WorkBanned  = 943000004
	g3WorkHidden  = 943000005
	g3WorkNoLocal = 943000006
	g3WorkCount0  = 943000007
	g3WorkMiss    = 943000099

	g3SecretURL = "https://example.invalid/secret-dl-aaaa"
	g3OtherURL  = "https://pan.example.invalid/file"
)

type resourceFix struct {
	*writeFix
	cat    *fakeCatalog
	claim  *fakeClaim
	shares *fakeShare
}

type fakeClaim struct {
	n   atomic.Int32
	err *legacyErrors.AppError
}

func (f *fakeClaim) call(_ context.Context, _ string, _ int64) *legacyErrors.AppError {
	f.n.Add(1)
	return f.err
}

type fakeShare struct {
	status linkcheck.Status
}

func (f *fakeShare) CheckShare(context.Context, []string, string) linkcheck.Status {
	if f == nil {
		return linkcheck.StatusUnknown
	}
	return f.status
}

func newResourceFix(t *testing.T, checker gate.Checker) *resourceFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	f.putSession(t, "sess-grant", w3UserGrant)
	f.addOAuthUser(g3HiddenA, "hidden-a", 1, nil)
	f.addOAuthUser(g3HiddenB, "hidden-b", 1, nil)
	cat := newResourceCatalog(t)
	claim := &fakeClaim{}
	shares := &fakeShare{status: linkcheck.StatusDead}
	f.ResourceCatalog = cat
	f.ResourceClaim = claim.call
	f.ResourceChecker = shares
	f.GalgameV1 = galgameapiv1.New(cat, nil, f.UserClient, f.rdb, geCDN)
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	rf := &resourceFix{writeFix: f, cat: cat, claim: claim, shares: shares}
	rf.cleanupResources(t)
	t.Cleanup(func() { rf.cleanupResources(t) })
	rf.seedResources(t)
	return rf
}

func resourceHiddenJSON(id int, name string) string {
	return fmt.Sprintf(`{"id":%d,"display_name":%q,"latin":%q,
		"localized":{"zh-Hans":{"value":%q,"machine":true}},
		"content_rating":"all_ages","release_date":"2026-01-01",
		"claim":{"site":"kungal","site_work_id":%d,"state":"hidden","content_limit":"sfw"},
		"cover_slots":{"portrait":{"url":%q,"width":256,"height":361,"thumbhash":"pUgK"}}}`,
		id, name, name, name+"（中）", id, geImageURL(id))
}

func newResourceCatalog(t *testing.T) *fakeCatalog {
	t.Helper()
	cat := &fakeCatalog{
		rows: map[int]client.CatalogWorkListItem{},
		works: []geWork{
			{id: g3WorkSFW, name: "AlphaRes", limit: "sfw", rating: "all_ages"},
			{id: g3WorkNSFW, name: "BetaNSFWRes", limit: "nsfw", rating: "r18"},
			{id: g3WorkUnpub, name: "GammaUnpub", limit: "sfw", rating: "all_ages"},
			{id: g3WorkBanned, name: "DeltaBanned", limit: "sfw", rating: "all_ages"},
			{id: g3WorkNoLocal, name: "EpsilonGhost", limit: "sfw", rating: "all_ages"},
			{id: g3WorkCount0, name: "ZetaCount", limit: "sfw", rating: "all_ages"},
		},
	}
	for _, w := range cat.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		cat.rows[w.id] = row
	}
	var hidden client.CatalogWorkListItem
	decodeInto(t, resourceHiddenJSON(g3WorkHidden, "HiddenRes"), &hidden)
	cat.rows[g3WorkHidden] = hidden
	cat.works = append(cat.works, geWork{id: g3WorkHidden, name: "HiddenRes", limit: "sfw", rating: "all_ages"})
	return cat
}

func (f *resourceFix) cleanupResources(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM message WHERE link LIKE '/galgame/943000%' OR sender_id IN (930004101, 930004102) OR receiver_id IN (930004101, 930004102)`,
		`DELETE FROM galgame_resource_like WHERE galgame_resource_id BETWEEN 930004201 AND 930004299 OR user_id BETWEEN 930000001 AND 930004199`,
		`DELETE FROM galgame_resource_link WHERE galgame_resource_id BETWEEN 930004201 AND 930004299`,
		`DELETE FROM galgame_resource WHERE id BETWEEN 930004201 AND 930004299 OR work_id BETWEEN 943000001 AND 943000099 OR user_id BETWEEN 930004101 AND 930004199`,
		`DELETE FROM galgame WHERE id BETWEEN 943000001 AND 943000099`,
		`DELETE FROM kungal_user_state WHERE user_id IN (930004101, 930004102)`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *resourceFix) seedResources(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("resource seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	tied := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)
	run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, 7, ?, ?), (?, 7, ?, ?)
		ON CONFLICT (user_id) DO NOTHING`, g3HiddenA, base, base, g3HiddenB, base, base)
	run(`INSERT INTO galgame (id, view, created, updated, published, content_limit, resource_update_time, resource_count, resource_publish_banned)
		VALUES (?, 0, ?, ?, true, 'sfw', ?, 8, false),
		       (?, 0, ?, ?, true, 'nsfw', ?, 1, false),
		       (?, 0, ?, ?, false, 'sfw', ?, 0, false),
		       (?, 0, ?, ?, true, 'sfw', ?, 1, true),
		       (?, 0, ?, ?, true, 'sfw', ?, 0, false)`,
		g3WorkSFW, base, base, base,
		g3WorkNSFW, base, base, base,
		g3WorkUnpub, base, base, base,
		g3WorkBanned, base, base, base,
		g3WorkCount0, base, base, base)

	authors := []int{w3UserAlice, w3UserBob, w3UserOther, w3UserGrant, w3UserStaff, g3HiddenA, g3HiddenB}
	note := "tied note seed"
	for i := 0; i < 7; i++ {
		id := g3ResTied + i
		run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
			VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', ?, 0, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
			id, note, g3WorkSFW, authors[i], tied, tied)
	}

	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', 'Main pack', '官方最新', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '2.5 GB', 'abcd', 'zip', 'A **note**', 0, ?, ?, 0, 0, 3, 1, '["Example"]'::jsonb, ?, ?)`,
		g3ResMain, g3WorkSFW, w3UserAlice, base.Add(time.Hour), base)
	run(`INSERT INTO galgame_resource_link (url, galgame_resource_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		g3SecretURL, g3ResMain, base, base, g3OtherURL, g3ResMain, base, base)

	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', 'expired pack', 1, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
		g3ResExpired, g3WorkSFW, w3UserBob, base, base)
	run(`INSERT INTO galgame_resource_link (url, galgame_resource_id, created, updated) VALUES (?, ?, ?, ?)`,
		"https://dead.example.invalid/x", g3ResExpired, base, base)

	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', 'count zero', 0, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
		g3ResCount0, g3WorkCount0, w3UserOther, base, base)

	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', 'nsfw pack', 0, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
		g3ResNSFW, g3WorkNSFW, w3UserBob, base, base)

	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', 'banned work pack', 0, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
		g3ResBanned, g3WorkBanned, w3UserGrant, base, base)
}

func (f *resourceFix) rs(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, idem, payload)
}

func (f *resourceFix) walkResources(t *testing.T, query string, limit int) []string {
	t.Helper()
	var got []string
	for page := 1; page <= 40; page++ {
		url := fmt.Sprintf("/api/v1/galgame-resources?page=%d&limit=%d%s", page, limit, query)
		resp, body := f.rs(t, http.MethodGet, url, "/galgame-resources", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, body)
		}
		ids := adminItemIDs(body)
		got = append(got, ids...)
		if len(ids) < limit {
			return got
		}
	}
	t.Fatal("walk did not end")
	return nil
}

func createResourceBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"resource_type":      "game",
		"resource_languages": []string{"zh-cn"},
		"resource_platforms": []string{"win"},
		"resource_runtimes":  []string{"native-win"},
		"size":               "1.5 GB",
		"download_urls":      []string{"https://cdn.example.invalid/a"},
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func renderableResourceAuthorsSQL() string {
	return strconv.Itoa(w3UserAlice) + "," + strconv.Itoa(w3UserBob) + "," +
		strconv.Itoa(w3UserOther) + "," + strconv.Itoa(w3UserGrant) + "," + strconv.Itoa(w3UserStaff)
}
