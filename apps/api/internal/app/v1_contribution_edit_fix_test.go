package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
)

const (
	g7bWork        = 957000001
	g7bWorkHidden  = 957000002
	g7bWorkMerged  = 957000003
	g7bWorkPlain   = 957000004
	g7bWorkUnknown = 957000099

	g7bSessAlice   = "g7b-sess-alice"
	g7bSessBob     = "g7b-sess-bob"
	g7bSessStaff   = "g7b-sess-staff"
	g7bSessOther   = "g7b-sess-other"
	g7bSessTrusted = "g7b-sess-trusted"
	g7bSessBanned  = "g7b-sess-banned"
	g7bSessNoscope = "g7b-sess-noscope"
	g7bSessQuota   = "g7b-sess-quota"
	g7bSessNoToken = "g7b-sess-notoken"
)

type g7bFix struct {
	*writeFix
	cat *fakeCatalog
	up  *g7bCatalog

	pOpenAlice, pOpenOther, pOpenBanned, pMergedBob, pDeclined, pWithdrawn int64
	pEmptyState, pOtherSite, pTag, pHidden, pPlainAlice, pStaffOwn        int64
	rev1, rev2, rev3                                                       int64
}

func newG7bFix(t *testing.T) *g7bFix {
	t.Helper()
	f := newWriteFix(t, nil)
	for _, s := range []struct {
		cookie, token string
		uid           int
		roles         []string
	}{
		{g7bSessAlice, "g7b-alice", w3UserAlice, nil},
		{g7bSessBob, "g7b-bob", w3UserBob, nil},
		{g7bSessStaff, "g7b-staff", w3UserStaff, []string{"user", "moderator"}},
		{g7bSessOther, "g7b-other", w3UserOther, nil},
		{g7bSessTrusted, "g7b-trusted", w3UserGrant, nil},
		{g7bSessBanned, "g7b-banned", w3UserBanned, nil},
		{g7bSessNoscope, "g7b-noscope", w3UserAlice, nil},
		{g7bSessQuota, "g7b-quota", w3UserAlice, nil},
		{g7bSessNoToken, "", w3UserAlice, nil},
	} {
		g7bSession(t, f, s.cookie, s.uid, s.token, s.roles...)
	}

	cat := &fakeCatalog{
		rows:       map[int]client.CatalogWorkListItem{},
		details:    map[int]*client.CatalogWorkDetail{},
		movedWorks: map[int]int64{g7bWorkMerged: g7bWork},
		works: []geWork{
			{id: g7bWork, name: "EditLive", limit: "sfw", rating: "all_ages"},
			{id: g7bWorkPlain, name: "EditPlain", limit: "nsfw", rating: "r18"},
		},
	}
	for _, w := range cat.works {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		cat.rows[w.id] = row
	}
	var hidden client.CatalogWorkListItem
	decodeInto(t, resourceHiddenJSON(g7bWorkHidden, "EditHidden"), &hidden)
	cat.rows[g7bWorkHidden] = hidden

	up := newG7bCatalog(t)
	up.actors = map[string]g7bActor{
		"g7b-alice":   {uid: w3UserAlice},
		"g7b-bob":     {uid: w3UserBob},
		"g7b-staff":   {uid: w3UserStaff, moderator: true},
		"g7b-other":   {uid: w3UserOther},
		"g7b-trusted": {uid: w3UserGrant, trusted: true},
		"g7b-banned":  {uid: w3UserBanned},
		"g7b-noscope": {uid: w3UserAlice, noScope: true},
		"g7b-quota":   {uid: w3UserAlice, quota: true},
		"staff-token": {uid: w3UserStaff, moderator: true},
		"bob-token":   {uid: w3UserBob},
	}
	up.owners = map[int64]int{g7bWork: w3UserBob}
	cc := catalogclient.New(catalogclient.Config{BaseURL: up.srv.URL, AppKey: g7bAppKey})

	f.GalgameV1 = galgameapiv1.New(cat, nil, f.UserClient, f.rdb, geCDN).
		WithWork(f.db, cc, nil, f.recordAward).
		WithEditing(msgService.NewNotifier(msgRepo.NewMessageRepository(f.db)))
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)

	gf := &g7bFix{writeFix: f, cat: cat, up: up}
	gf.cleanupG7b(t)
	t.Cleanup(func() { gf.cleanupG7b(t) })
	gf.seedG7b(t)
	t.Cleanup(func() {
		perm.SetUserOverrides(nil)
	})
	return gf
}

func g7bSession(t *testing.T, f *writeFix, cookie string, uid int, token string, roles ...string) {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: uid, Name: "n", Roles: roles},
		OAuthAccessToken: token,
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(cookie), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *g7bFix) cleanupG7b(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM galgame_activity WHERE work_id BETWEEN 957000001 AND 957000099`,
		`DELETE FROM message WHERE link LIKE '/galgame/957000%'`,
		`DELETE FROM galgame WHERE id BETWEEN 957000001 AND 957000099`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *g7bFix) seedG7b(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if err := f.db.Exec(`INSERT INTO galgame (id, view, like_count, favorite_count, created, updated, published, content_limit, creator_user_id, resource_update_time, resource_publish_banned)
		VALUES (?, 0, 0, 0, ?, ?, true, 'sfw', ?, ?, false),
		       (?, 0, 0, 0, ?, ?, false, 'nsfw', NULL, ?, false)`,
		g7bWork, base, base, w3UserBob, base,
		g7bWorkPlain, base, base, base).Error; err != nil {
		t.Fatalf("g7b seed: %v", err)
	}

	up := f.up
	f.rev1 = up.addRevision(g7bWork, "created", w3UserAlice, 0,
		map[string]any{"catalog.work.display_name": "Edit One", "catalog.work.content_rating": float64(0)},
		"catalog.work.display_name", "catalog.work.content_rating").id
	f.rev2 = up.addRevision(g7bWork, "direct", w3UserStaff, 0,
		map[string]any{"catalog.work.display_name": "Edit Two"}, "catalog.work.display_name").id
	up.addRevision(g7bWorkHidden, "created", w3UserAlice, 0, map[string]any{"catalog.work.display_name": "Hidden"})

	add := func(p *g7bProposal) int64 { return up.addProposal(p).id }
	f.pOpenAlice = add(&g7bProposal{workID: g7bWork, proposer: w3UserAlice, note: "first",
		patch: map[string]any{"catalog.work.display_name": "Alice Name"}})
	f.pOpenOther = add(&g7bProposal{workID: g7bWork, proposer: w3UserOther,
		patch: map[string]any{"catalog.work.tag_ids": []any{"12"}}})
	f.pOpenBanned = add(&g7bProposal{workID: g7bWork, proposer: w3UserBanned,
		patch: map[string]any{"catalog.work.display_name": "Spam"}})
	decided := base.Add(time.Hour)
	f.pMergedBob = add(&g7bProposal{workID: g7bWork, proposer: w3UserBob, state: "merged", decider: w3UserStaff, decidedAt: &decided,
		patch:      map[string]any{"catalog.work.display_name": "Bob Name"},
		amendments: []g7bAmendment{{id: 957399001, seq: 1, amender: w3UserStaff, note: "tidy", created: base}}})
	f.pDeclined = add(&g7bProposal{workID: g7bWork, proposer: w3UserAlice, state: "declined", decider: w3UserStaff, decidedAt: &decided,
		patch: map[string]any{"catalog.work.display_name": "Nope"}})
	f.pWithdrawn = add(&g7bProposal{workID: g7bWork, proposer: w3UserAlice, state: "withdrawn",
		patch: map[string]any{"catalog.work.display_name": "Later"}})
	f.pEmptyState = add(&g7bProposal{workID: g7bWork, proposer: w3UserAlice,
		patch: map[string]any{"catalog.work.display_name": "Blank"}})
	up.emptyState = f.pEmptyState
	f.pOtherSite = add(&g7bProposal{workID: g7bWork, proposer: w3UserAlice, site: "moyu",
		patch: map[string]any{"catalog.work.display_name": "Moyu"}})
	f.pTag = add(&g7bProposal{workID: 88, proposer: w3UserAlice, entityType: "catalog.tag",
		patch: map[string]any{"catalog.tag.intros": []any{}}})
	f.pHidden = add(&g7bProposal{workID: g7bWorkHidden, proposer: w3UserAlice,
		patch: map[string]any{"catalog.work.display_name": "Hidden Two"}})
	f.pPlainAlice = add(&g7bProposal{workID: g7bWorkPlain, proposer: w3UserAlice,
		patch: map[string]any{"catalog.work.display_name": "Plain"}})
	f.pStaffOwn = add(&g7bProposal{workID: g7bWorkPlain, proposer: w3UserStaff,
		patch: map[string]any{"catalog.work.display_name": "Staff"}})

	up.mu.Lock()
	f.rev3 = up.addRevisionLocked(g7bWork, "merged", w3UserBob, w3UserStaff, f.pMergedBob,
		map[string]any{"catalog.work.display_name": "Bob Name"}, []string{"catalog.work.display_name"}).id
	up.mu.Unlock()
}

func (f *g7bFix) call(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, idem, payload)
}

func (f *g7bFix) callHdr(t *testing.T, method, rawURL, spec, session, idem string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, method, rawURL, session, spec, idem, hdr, payload)
	if len(raw) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, raw)
}

func (f *g7bFix) bearer(t *testing.T, method, rawURL, spec, token, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.callHdr(t, method, rawURL, spec, "", idem, http.Header{"Authorization": {"Bearer " + token}}, payload)
}

func g7bWorkPath(id int) string { return "/api/v1/works/" + strconv.Itoa(id) }

func g7bProposalPath(id int64) string { return "/api/v1/edit-proposals/" + strconv.FormatInt(id, 10) }

func itemByID(body map[string]any, id int64) map[string]any {
	items, _ := body["items"].([]any)
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		if strID(it["id"]) == strconv.FormatInt(id, 10) {
			return it
		}
	}
	return nil
}

func g7bID(id int64) string { return strconv.FormatInt(id, 10) }

// denyEditReview revokes the edit-review key from the moderator by override, the
// way an admin does on /admin/permission; the role alone must then open nothing.
func (f *g7bFix) denyEditReview(t *testing.T) {
	t.Helper()
	perm.SetUserOverrides(map[int][]perm.Override{
		w3UserStaff: {{Permission: perm.GalgameEditProposalReview, Effect: perm.EffectRevoke}},
	})
}
