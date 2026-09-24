package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	legacyErrors "kun-galgame-api/pkg/errors"
)

const (
	g7aDraft      = 948000001
	g7aPending    = 948000002
	g7aDeclined   = 948000003
	g7aHiddenLive = 948000004
	g7aHiddenDraf = 948000005
	g7aHiddenBare = 948000006
	g7aBobPending = 948000007
	g7aLive       = 948000008
	g7aDraftRes   = 948000009
	g7aWizNSFW    = 948000010
	g7aWizSFW     = 948000011
	g7aTwin       = 948000012
	g7aWizDraft   = 948000013
	g7aWizDecl    = 948000014
	g7aStaffOwn   = 948000015
	g7aGhostOwner = 948000016
	g7aGhostUser  = 930000777
	g7aFirstMint  = 948000100
	g7aResource   = 948500001

	g7aTokAlice   = "g7-alice"
	g7aTokBob     = "g7-bob"
	g7aTokStaff   = "g7-staff"
	g7aTokNoscope = "g7-noscope"
	g7aTok429     = "g7-429"
	g7aTokTrusted = "g7-trusted"
)

type g7aEvent struct {
	id     int64
	from   *string
	to     string
	reason *string
	actor  int64
	at     time.Time
}

type g7aClaim struct {
	id      int64
	name    string
	state   string
	owner   int64
	nsfw    bool
	rating  int
	version int
	events  []g7aEvent
}

type g7aMutation struct {
	workID  int64
	state   string
	ifMatch string
	token   string
}

type g7aUser struct {
	*g6User
	cat *fakeCatalog

	lock          sync.Mutex
	claims        map[int64]*g7aClaim
	nextWork      int64
	nextEvent     int64
	clock         time.Time
	modCalls      int
	decisions     []g7aMutation
	patches       []g7aMutation
	deletes       []g7aMutation
	mints         []catalogclient.UserWorkSubmitRequest
	proposals     []catalogclient.UserEditCreateRequest
	merges        []g7aMutation
	proposeErr    error
	mergeErr      error
	mergeClosesAs string
	autoMerge     bool
	forceState    map[int64]string
	snapshots     int

	failAfterWrite bool
	readsFail      bool
	onWrite        func()
}

type g7aFix struct {
	*writeFix
	cat   *fakeCatalog
	works *g7aWorks
	user  *g7aUser
}

// g7aWorks lets a test play the search index a day behind: the display gate on
// the query is dropped, the way a stale index would ignore it.
type g7aWorks struct {
	*fakeCatalog
	lag bool
}

func (w *g7aWorks) CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError) {
	if w.lag {
		q = g7aClone(q)
		q.Del("content_limit")
	}
	return w.fakeCatalog.CatalogWorksSearch(ctx, q)
}

func g7aClone(q url.Values) url.Values {
	out := url.Values{}
	for k, v := range q {
		out[k] = slices.Clone(v)
	}
	return out
}

func newG7aFix(t *testing.T) *g7aFix {
	return newG7aFixWith(t, nil)
}

func newG7aFixWith(t *testing.T, checker gate.Checker) *g7aFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	g6bindToken(t, f, "sess-alice", w3UserAlice, g7aTokAlice)
	g6bindToken(t, f, "sess-bob", w3UserBob, g7aTokBob)
	g6bindToken(t, f, "sess-staff", w3UserStaff, g7aTokStaff, "user", "moderator")
	g6bindToken(t, f, "sess-other", w3UserOther, "g7-other")
	g6bindToken(t, f, "sess-noscope", w3UserAlice, g7aTokNoscope)
	g6bindToken(t, f, "sess-429", w3UserAlice, g7aTok429)
	g6bindToken(t, f, "sess-trusted", w3UserAlice, g7aTokTrusted)
	g6bindToken(t, f, "sess-banned", w3UserBanned, g7aTokAlice)

	cat := &fakeCatalog{
		rows:       map[int]client.CatalogWorkListItem{},
		details:    map[int]*client.CatalogWorkDetail{},
		movedWorks: map[int]int64{},
	}
	works := &g7aWorks{fakeCatalog: cat}
	user := &g7aUser{
		g6User: newG6User(), cat: cat,
		claims: map[int64]*g7aClaim{}, nextWork: g7aFirstMint, nextEvent: 700,
		clock:      time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		forceState: map[int64]string{},
	}
	user.seed(t)
	for _, w := range []struct {
		id     int
		name   string
		rating string
	}{{g7aWizNSFW, "WizardNsfw", "r18"}, {g7aWizSFW, "WizardSafe", "all_ages"}} {
		var row client.CatalogWorkListItem
		decodeInto(t, g7aRowJSON(w.id, w.name, "", "", w.rating), &row)
		cat.rows[w.id] = row
	}
	cat.works = []geWork{
		{id: g7aWizNSFW, name: "WizardNsfw"}, {id: g7aWizSFW, name: "WizardSafe"},
		{id: g7aWizDraft, name: "WizardDraft"}, {id: g7aWizDecl, name: "WizardDeclined"},
		{id: g7aTwin, name: "TwinTitle"},
	}

	repo := galgameRepo.NewGalgameCollectionRepository(f.db)
	f.GalgameV1 = galgameapiv1.New(works, nil, f.UserClient, f.rdb, geCDN).
		WithWork(f.db, user, nil, f.recordAward).
		WithUserPlane(f.TrustCheck, nil, repo)
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	gf := &g7aFix{writeFix: f, cat: cat, works: works, user: user}
	gf.cleanupG7(t)
	t.Cleanup(func() { gf.cleanupG7(t) })
	gf.seedG7(t)
	return gf
}

func g7aRowJSON(id int, name, state, limit, rating string) string {
	claim := "null"
	if state != "" {
		claim = fmt.Sprintf(`{"site":"kungal","site_work_id":%d,"state":%q,"content_limit":%q}`, id, state, limit)
	}
	return fmt.Sprintf(`{"id":%d,"display_name":%q,"latin":%q,"localized":{},
		"content_rating":%q,"release_date":null,"claim":%s,
		"cover_slots":{"portrait":{"url":%q,"width":256,"height":361,"thumbhash":"pUgK"}}}`,
		id, name, strings.ToLower(name), rating, claim, geImageURL(id))
}

func g7aRating(code int) string {
	switch code {
	case 1:
		return "sensitive"
	case 2:
		return "r18"
	}
	return "all_ages"
}

func (u *g7aUser) syncRow(t *testing.T, c *g7aClaim) {
	limit := "sfw"
	if c.nsfw {
		limit = "nsfw"
	}
	var row client.CatalogWorkListItem
	decodeInto(t, g7aRowJSON(int(c.id), c.name, c.state, limit, g7aRating(c.rating)), &row)
	u.cat.rows[int(c.id)] = row
}

func g7aStr(s string) *string { return &s }

func (u *g7aUser) seed(t *testing.T) {
	t.Helper()
	add := func(id int64, name, state string, owner int64, nsfw bool, history ...g7aEvent) {
		c := &g7aClaim{id: id, name: name, state: state, owner: owner, nsfw: nsfw, version: 1}
		for _, e := range history {
			u.nextEvent++
			e.id = u.nextEvent
			u.clock = u.clock.Add(time.Hour)
			e.at = u.clock
			c.events = append(c.events, e)
		}
		u.claims[id] = c
		u.syncRow(t, c)
	}
	alice, bob, staff := int64(w3UserAlice), int64(w3UserBob), int64(w3UserStaff)
	add(g7aDraft, "DraftG7", "draft", alice, false, g7aEvent{to: "draft", actor: alice})
	add(g7aPending, "PendingG7", "pending", alice, true, g7aEvent{to: "pending", actor: alice})
	add(g7aDeclined, "DeclinedG7", "declined", alice, false,
		g7aEvent{to: "pending", actor: alice},
		g7aEvent{from: g7aStr("pending"), to: "declined", reason: g7aStr("titles are wrong"), actor: staff})
	add(g7aHiddenLive, "HiddenLiveG7", "hidden", alice, false,
		g7aEvent{to: "pending", actor: alice},
		g7aEvent{from: g7aStr("pending"), to: "live", actor: staff},
		g7aEvent{from: g7aStr("live"), to: "hidden", actor: staff})
	add(g7aHiddenDraf, "HiddenDraftG7", "hidden", alice, false,
		g7aEvent{to: "draft", actor: alice},
		g7aEvent{from: g7aStr("draft"), to: "hidden", actor: staff})
	add(g7aHiddenBare, "HiddenBareG7", "hidden", bob, false)
	add(g7aBobPending, "BobPendingG7", "pending", bob, false, g7aEvent{to: "pending", actor: bob})
	add(g7aLive, "LiveG7", "live", alice, false,
		g7aEvent{to: "pending", actor: alice}, g7aEvent{from: g7aStr("pending"), to: "live", actor: staff})
	add(g7aDraftRes, "DraftResG7", "draft", alice, false, g7aEvent{to: "draft", actor: alice})
	add(g7aWizDraft, "WizardDraft", "draft", bob, false, g7aEvent{to: "draft", actor: bob})
	add(g7aWizDecl, "WizardDeclined", "declined", bob, false, g7aEvent{to: "declined", actor: staff})
	add(g7aTwin, "TwinTitle", "live", bob, false, g7aEvent{to: "live", actor: bob})
	add(g7aStaffOwn, "StaffOwnG7", "pending", staff, false, g7aEvent{to: "pending", actor: staff})
	add(g7aGhostOwner, "GhostOwnerG7", "pending", bob, false, g7aEvent{to: "pending", actor: bob})
}

func (f *g7aFix) cleanupG7(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM galgame_resource WHERE id = 948500001`,
		`DELETE FROM galgame WHERE id BETWEEN 948000001 AND 948000999`,
	} {
		_ = f.db.Exec(q).Error
	}
}

func (f *g7aFix) seedG7(t *testing.T) {
	t.Helper()
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("g7 seed: %v\n%s", err, q)
		}
	}
	for _, id := range []int{g7aDraft, g7aPending, g7aDeclined, g7aHiddenLive, g7aLive, g7aDraftRes} {
		run(`INSERT INTO galgame (id, view, like_count, favorite_count, created, updated, published, creator_user_id, resource_update_time, resource_publish_banned)
			VALUES (?, 0, 0, 0, ?, ?, false, ?, ?, false)`, id, base, base, w3UserAlice, base)
	}
	run(`INSERT INTO galgame (id, view, like_count, favorite_count, created, updated, published, creator_user_id, resource_update_time, resource_publish_banned)
		VALUES (?, 0, 0, 0, ?, ?, false, ?, ?, false), (?, 0, 0, 0, ?, ?, false, ?, ?, false), (?, 0, 0, 0, ?, ?, false, ?, ?, false)`,
		g7aBobPending, base, base, w3UserBob, base, g7aHiddenBare, base, base, w3UserBanned, base,
		g7aStaffOwn, base, base, w3UserStaff, base)
	run(`INSERT INTO galgame (id, view, like_count, favorite_count, created, updated, published, creator_user_id, resource_update_time, resource_publish_banned)
		VALUES (?, 0, 0, 0, ?, ?, false, ?, ?, false)`, g7aGhostOwner, base, base, g7aGhostUser, base)
	run(`INSERT INTO galgame_resource (id, type, language, platform, title, version_label, languages, platforms, runtimes, size, code, password, note, status, work_id, user_id, like_count, comment_count, view, download, provider_name, created, updated)
		VALUES (?, 'game', 'zh-cn', 'windows', '', '', '["zh-cn"]'::jsonb, '["win"]'::jsonb, '["native-win"]'::jsonb, '1 GB', '', '', 'kept', 0, ?, ?, 0, 0, 0, 0, '[]'::jsonb, ?, ?)`,
		g7aResource, g7aDraftRes, w3UserAlice, base, base)
}

func (u *g7aUser) uid(token string) int64 {
	switch token {
	case g7aTokAlice, g7aTokNoscope, g7aTok429, g7aTokTrusted:
		return w3UserAlice
	case g7aTokBob, "bob-token":
		return w3UserBob
	case g7aTokStaff, "staff-token":
		return w3UserStaff
	case "g7-other":
		return w3UserOther
	}
	return 0
}

func (u *g7aUser) gate(token string) error {
	switch token {
	case g7aTokNoscope:
		return catalogclient.ErrInsufficientScope
	case g7aTok429:
		return &catalogclient.UserAPIError{Status: http.StatusTooManyRequests, ProblemCode: "QUOTA_EXCEEDED", Message: "slow down", RetryAfter: "30"}
	}
	return nil
}

func (u *g7aUser) reviewer(token string) bool {
	return token == g7aTokStaff || token == "staff-token"
}

func g7aProblem(status int, code, msg string) *catalogclient.UserAPIError {
	return &catalogclient.UserAPIError{Status: status, ProblemCode: code, Message: msg}
}

func g7aETag(c *g7aClaim) string { return fmt.Sprintf(`"c%d.%d"`, c.id, c.version) }

func (u *g7aUser) checkMatch(c *g7aClaim, ifMatch string) error {
	switch ifMatch {
	case "":
		return g7aProblem(http.StatusPreconditionRequired, "PRECONDITION_REQUIRED", "If-Match required")
	case "*", g7aETag(c):
		return nil
	}
	return g7aProblem(http.StatusPreconditionFailed, "PRECONDITION_FAILED", "stale")
}

func (u *g7aUser) move(c *g7aClaim, to string, actor int64, reason *string) {
	from := c.state
	u.nextEvent++
	u.clock = u.clock.Add(time.Hour)
	c.events = append(c.events, g7aEvent{id: u.nextEvent, from: &from, to: to, reason: reason, actor: actor, at: u.clock})
	c.state = to
	c.version++
}

func (u *g7aUser) item(c *g7aClaim, viewer int64, queueRow bool) catalogclient.UserClaimItem {
	it := catalogclient.UserClaimItem{WorkID: c.id, DisplayName: c.name, Site: "kungal", ClaimState: c.state}
	if queueRow {
		return it
	}
	if n := len(c.events); n > 0 {
		last := c.events[n-1]
		it.LastEventID, it.LastFromState, it.LastToState = last.id, last.from, last.to
		it.LastReason, it.LastActorUID, it.LastEventAt = last.reason, last.actor, last.at
	}
	for _, e := range c.events {
		if e.actor != viewer {
			continue
		}
		if it.FirstActedAt.IsZero() {
			it.FirstActedAt = e.at
		}
		it.ActedCount++
	}
	return it
}

func (u *g7aUser) wrote() {
	if !u.failAfterWrite {
		return
	}
	u.readsFail = true
	if u.onWrite != nil {
		u.onWrite()
	}
}

func (u *g7aUser) readGate(token string) error {
	if u.readsFail {
		return catalogclient.ErrUpstream
	}
	return u.gate(token)
}

func (u *g7aUser) own(token string, id int64) (*g7aClaim, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	c := u.claims[id]
	if c == nil {
		return nil, g7aProblem(http.StatusNotFound, "NOT_FOUND", "No claim with this id.")
	}
	if c.owner != u.uid(token) {
		return nil, g7aProblem(http.StatusForbidden, "CLAIM_NOT_OWNED", "this claim belongs to another user; only its owner may move it.")
	}
	return c, nil
}

func (u *g7aUser) SubmitWorkUser(_ context.Context, token string, req catalogclient.UserWorkSubmitRequest) (*catalogclient.WorkSubmitResult, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.lock.Lock()
	defer u.lock.Unlock()
	u.mints = append(u.mints, req)
	names := []string{}
	if d, ok := req.Fields["catalog.work.display_name"].(string); ok {
		names = append(names, d)
	}
	titles, _ := req.Fields["catalog.work.titles"].([]any)
	for _, t := range titles {
		if m, ok := t.(map[string]any); ok {
			names = append(names, fmt.Sprint(m["title"]))
		}
	}
	for _, n := range names {
		switch n {
		case "ExistsTitle":
			return nil, g7aProblem(http.StatusConflict, "ALREADY_EXISTS", "refs already resolve to a work")
		case "InflightTitle":
			return nil, g7aProblem(http.StatusConflict, "IDEMPOTENCY_REQUEST_IN_PROGRESS", "in flight")
		}
	}
	if !req.ConfirmDuplicates {
		for _, n := range names {
			if strings.EqualFold(n, "TwinTitle") {
				return nil, &catalogclient.UserAPIError{
					Status: http.StatusConflict, ProblemCode: "DUPLICATE_SUSPECTS",
					Message:  "The mint's titles match live works of the same medium; re-send with confirm_duplicates=true to mint anyway.",
					Suspects: []catalogclient.DuplicateSuspect{{ID: strconv.Itoa(g7aTwin), DisplayName: "TwinTitle"}},
				}
			}
		}
	}
	uid := u.uid(token)
	state := "pending"
	if token == g7aTokTrusted {
		state = "live"
	}
	nsfw, _ := req.Fields["catalog.work.display_nsfw"].(bool)
	rating, _ := req.Fields["catalog.work.content_rating"].(int)
	c := &g7aClaim{id: u.nextWork, name: names[0], state: state, owner: uid, nsfw: nsfw, rating: rating, version: 1}
	u.nextWork++
	u.nextEvent++
	u.clock = u.clock.Add(time.Hour)
	c.events = append(c.events, g7aEvent{id: u.nextEvent, to: state, actor: uid, at: u.clock})
	u.claims[c.id] = c
	var row client.CatalogWorkListItem
	limit := "sfw"
	if nsfw {
		limit = "nsfw"
	}
	if err := json.Unmarshal([]byte(g7aRowJSON(int(c.id), c.name, c.state, limit, g7aRating(rating))), &row); err != nil {
		return nil, err
	}
	u.cat.rows[int(c.id)] = row
	u.wrote()
	return &catalogclient.WorkSubmitResult{WorkID: c.id, ProductWorkID: c.id, ClaimState: state}, nil
}

func (u *g7aUser) GetMyClaim(_ context.Context, token string, workID int64) (*catalogclient.UserClaimItem, string, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	if err := u.readGate(token); err != nil {
		return nil, "", err
	}
	c := u.claims[workID]
	if c == nil || c.owner != u.uid(token) {
		return nil, "", g7aProblem(http.StatusNotFound, "NOT_FOUND", "No claim with this id.")
	}
	it := u.item(c, u.uid(token), false)
	return &it, g7aETag(c), nil
}

func (u *g7aUser) GetModerationClaim(_ context.Context, token string, workID int64) (*catalogclient.UserClaimItem, string, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.modCalls++
	if err := u.readGate(token); err != nil {
		return nil, "", err
	}
	if !u.reviewer(token) {
		return nil, "", g7aProblem(http.StatusForbidden, "PERMISSION_REQUIRED", "needs catalog.claim.review")
	}
	c := u.claims[workID]
	if c == nil {
		return nil, "", g7aProblem(http.StatusNotFound, "NOT_FOUND", "No claim with this id.")
	}
	it := u.item(c, u.uid(token), false)
	return &it, g7aETag(c), nil
}

func (u *g7aUser) PatchMyClaim(_ context.Context, token string, workID int64, state, ifMatch string) (*catalogclient.UserClaimItem, string, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.patches = append(u.patches, g7aMutation{workID: workID, state: state, ifMatch: ifMatch, token: token})
	c, err := u.own(token, workID)
	if err != nil {
		return nil, "", err
	}
	if err := u.checkMatch(c, ifMatch); err != nil {
		return nil, "", err
	}
	ok := false
	switch state {
	case "pending":
		ok = c.state == "draft" || c.state == "declined"
	case "withdrawn", "draft":
		ok = c.state == "pending" || c.state == "live"
		state = "draft"
	case "live":
		ok = c.state == "draft"
	}
	if !ok {
		return nil, "", g7aProblem(http.StatusConflict, "INVALID_STATE_TRANSITION", fmt.Sprintf("cannot submit a claim in state %q", c.state))
	}
	u.move(c, state, u.uid(token), nil)
	u.wrote()
	it := u.item(c, 0, false)
	return &it, g7aETag(c), nil
}

func (u *g7aUser) DecideClaim(_ context.Context, token string, workID int64, decision, note, ifMatch string) (*catalogclient.ClaimDecision, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.modCalls++
	u.decisions = append(u.decisions, g7aMutation{workID: workID, state: decision, ifMatch: ifMatch, token: token})
	if err := u.gate(token); err != nil {
		return nil, err
	}
	if !u.reviewer(token) {
		return nil, g7aProblem(http.StatusForbidden, "PERMISSION_REQUIRED", "needs catalog.claim.review")
	}
	c := u.claims[workID]
	if c == nil {
		return nil, g7aProblem(http.StatusNotFound, "NOT_FOUND", "No claim with this id.")
	}
	if err := u.checkMatch(c, ifMatch); err != nil {
		return nil, err
	}
	var to string
	switch decision {
	case "approve":
		if c.state == "pending" {
			to = "live"
		}
	case "decline":
		if strings.TrimSpace(note) == "" {
			return nil, &catalogclient.UserAPIError{Status: http.StatusUnprocessableEntity, ProblemCode: "VALIDATION_FAILED",
				Message: "a decline needs a reason", FieldErrors: []catalogclient.ProblemFieldError{{Pointer: "/", Reason: "REQUIRED"}}}
		}
		if c.state == "pending" {
			to = "declined"
		}
	case "ban":
		if c.state != "hidden" {
			to = "hidden"
		}
	case "unban":
		if c.state == "hidden" {
			to = "live"
			for i := len(c.events) - 1; i >= 0; i-- {
				if e := c.events[i]; e.to == "hidden" {
					if e.from != nil && *e.from != "hidden" && *e.from != "none" {
						to = *e.from
					}
					break
				}
			}
		}
	}
	if to == "" {
		return nil, g7aProblem(http.StatusConflict, "INVALID_STATE_TRANSITION", "cannot "+decision+" a claim in state "+c.state)
	}
	var reason *string
	if note != "" {
		reason = &note
	}
	from := c.state
	u.move(c, to, u.uid(token), reason)
	d := &catalogclient.ClaimDecision{EventID: c.events[len(c.events)-1].id, FromState: &from, ToState: to}
	if forced := u.forceState[workID]; forced != "" {
		c.state = forced
	}
	u.wrote()
	return d, nil
}

func (u *g7aUser) DeleteMyClaim(_ context.Context, token string, workID int64, ifMatch string) error {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.deletes = append(u.deletes, g7aMutation{workID: workID, ifMatch: ifMatch, token: token})
	c, err := u.own(token, workID)
	if err != nil {
		return err
	}
	if err := u.checkMatch(c, ifMatch); err != nil {
		return err
	}
	if c.state != "draft" {
		return g7aProblem(http.StatusConflict, "INVALID_STATE_TRANSITION", "only a draft can be deleted")
	}
	delete(u.claims, workID)
	delete(u.cat.rows, int(workID))
	return nil
}

func (u *g7aUser) listClaims(token string, f catalogclient.UserClaimFilter, keep func(*g7aClaim) bool, queue bool) (*catalogclient.UserClaimPage, error) {
	var picked []*g7aClaim
	for _, c := range u.claims {
		if keep(c) && (len(f.ClaimStates) == 0 || slices.Contains(f.ClaimStates, c.state)) {
			picked = append(picked, c)
		}
	}
	last := func(c *g7aClaim) int64 {
		if n := len(c.events); n > 0 {
			return c.events[n-1].id / 10
		}
		return 0
	}
	sort.Slice(picked, func(i, j int) bool {
		if a, b := last(picked[i]), last(picked[j]); a != b {
			return a > b
		}
		return picked[i].id > picked[j].id
	})
	offset := 0
	if f.Cursor != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(f.Cursor, "cur_"))
		if err != nil {
			return nil, g7aProblem(http.StatusBadRequest, "INVALID_CURSOR", "bad cursor")
		}
		offset = n
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	page := &catalogclient.UserClaimPage{Total: int64(len(picked))}
	end := min(offset+limit, len(picked))
	for _, c := range picked[min(offset, len(picked)):end] {
		page.Items = append(page.Items, u.item(c, u.uid(token), queue))
	}
	if end < len(picked) {
		page.NextCursor = "cur_" + strconv.Itoa(end)
	}
	return page, nil
}

func (u *g7aUser) MyClaims(_ context.Context, token string, f catalogclient.UserClaimFilter) (*catalogclient.UserClaimPage, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	if err := u.gate(token); err != nil {
		return nil, err
	}
	uid := u.uid(token)
	return u.listClaims(token, f, func(c *g7aClaim) bool {
		if f.Kind == catalogclient.ClaimKindAudited {
			if c.owner == uid {
				return false
			}
			for _, e := range c.events {
				if e.actor == uid {
					return true
				}
			}
			return false
		}
		return c.owner == uid
	}, false)
}

func (u *g7aUser) ListModerationClaims(_ context.Context, token string, f catalogclient.UserClaimFilter) (*catalogclient.UserClaimPage, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.modCalls++
	if err := u.gate(token); err != nil {
		return nil, err
	}
	if !u.reviewer(token) {
		return nil, g7aProblem(http.StatusForbidden, "PERMISSION_REQUIRED", "needs catalog.claim.review")
	}
	if len(f.ClaimStates) == 0 {
		f.ClaimStates = []string{"pending"}
	}
	return u.listClaims(token, f, func(*g7aClaim) bool { return true }, true)
}

func (u *g7aUser) CreateMyProposal(_ context.Context, _ string, req catalogclient.UserEditCreateRequest, _ string) (*catalogclient.EditProposal, string, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.proposals = append(u.proposals, req)
	if u.proposeErr != nil {
		return nil, "", u.proposeErr
	}
	status := "open"
	if u.autoMerge {
		status = "merged"
	}
	return &catalogclient.EditProposal{ID: int64(8000 + len(u.proposals)), Status: status}, `"p1"`, nil
}

func (u *g7aUser) DecideProposal(_ context.Context, token string, id int64, decision, _, ifMatch string) (string, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.merges = append(u.merges, g7aMutation{workID: id, state: decision, ifMatch: ifMatch, token: token})
	if u.mergeErr != nil {
		return "", u.mergeErr
	}
	if u.mergeClosesAs != "" {
		return u.mergeClosesAs, nil
	}
	return "merged", nil
}

func (u *g7aUser) EditSnapshotUser(_ context.Context, token, _ string, id int64) (map[string]any, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.snapshots++
	if err := u.gate(token); err != nil {
		return nil, err
	}
	c := u.claims[id]
	if c == nil {
		return nil, catalogclient.ErrNotFound
	}
	return map[string]any{"catalog.work.display_nsfw": c.nsfw, "catalog.work.content_rating": float64(c.rating)}, nil
}

func (f *g7aFix) call(t *testing.T, method, rawURL, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.ts(t, method, rawURL, spec, session, idem, payload)
}

func (f *g7aFix) callWith(t *testing.T, method, rawURL, spec, session string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, method, rawURL, session, spec, "", hdr, payload)
	if len(raw) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, raw)
}

func g7aSub(id int) string { return "/api/v1/work-submissions/" + idStr(id) }

func g7aMintBody(title string, extra map[string]any) map[string]any {
	body := map[string]any{
		"titles":            []map[string]any{{"locale": "ja", "title": title}},
		"original_language": "ja",
		"content_rating":    "r18",
		"is_nsfw":           true,
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}
