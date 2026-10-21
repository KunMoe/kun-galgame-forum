package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/ratingapiv1"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	grAlice   = 962000001
	grBob     = 962000002
	grBanned  = 962000003
	grCreator = 962000004
	grStaff   = 962000005
	grGone    = 962000006
	grRater   = 962000010
	grLiker   = 962000030

	grWorkSFW    = 962000101
	grWorkSFW2   = 962000102
	grWorkNSFW   = 962000103
	grWorkHidden = 962000104
	grWorkFresh  = 962000105
	grWorkNone   = 962000106

	grRatingAlice  = 962000201
	grRatingBanned = 962000202
	grRatingHidden = 962000203
	grRatingGone   = 962000204
	grRatingRaters = 962000210

	grUserMax   = 962000099
	grWorkMax   = 962000199
	grRatingMax = 962000299
)

type grRating struct {
	id, workID, userID  int
	overall, view       int
	created             time.Time
	spoiler, playStatus string
	gameTypes           []string
	scores              [8]int
}

// Ties on purpose: overall, view and created repeat, so only the id tiebreak
// makes the order total.
func grRatings() []grRating {
	t0 := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	works := []int{grWorkSFW, grWorkSFW2, grWorkNSFW}
	overalls := []int{7, 7, 9, 7, 5, 9, 7, 3, 9, 7, 5, 7}
	views := []int{3, 1, 3, 0, 3, 1, 3, 1, 0, 3, 1, 3}
	spoilers := []string{"none", "portion", "serious"}
	statuses := []string{"done_all", "doing", "wish", "done_main"}
	types := [][]string{{"plot"}, {"moe", "daily"}, {"ba_saku"}, {"plot", "moe"}}
	out := []grRating{
		{id: grRatingAlice, workID: grWorkSFW, userID: grAlice, overall: 8, view: 2, created: t0, spoiler: "none", playStatus: "done_all",
			gameTypes: []string{"plot"}, scores: [8]int{9, 0, 7, 0, 0, 0, 10, 1}},
		{id: grRatingBanned, workID: grWorkSFW, userID: grBanned, overall: 7, view: 3, created: t0, spoiler: "none", playStatus: "doing", gameTypes: []string{"plot"}},
		{id: grRatingHidden, workID: grWorkHidden, userID: grAlice, overall: 7, view: 3, created: t0, spoiler: "none", playStatus: "doing", gameTypes: []string{"plot"}},
		{id: grRatingGone, workID: grWorkSFW2, userID: grGone, overall: 7, view: 1, created: t0.Add(time.Hour), spoiler: "portion", playStatus: "wish", gameTypes: []string{"moe"}},
	}
	for i := range 12 {
		out = append(out, grRating{
			id: grRatingRaters + i, workID: works[i%3], userID: grRater + i,
			overall: overalls[i], view: views[i], created: t0.Add(time.Duration(i%4) * time.Hour),
			spoiler: spoilers[i%3], playStatus: statuses[i%4], gameTypes: types[i%4],
		})
	}
	return out
}

type grChecker struct{ calls atomic.Int32 }

func (c *grChecker) Check(_ context.Context, req trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	c.calls.Add(1)
	if strings.Contains(req.Text, "forbidden") {
		return &trustclient.CheckResult{Decision: gate.DecisionDeny, Matched: []string{"forbidden"}}, nil
	}
	return &trustclient.CheckResult{Decision: gate.DecisionAllow}, nil
}

type grScanner struct{ got chan trustclient.ScanRequest }

func (s *grScanner) Scan(_ context.Context, req trustclient.ScanRequest) (*trustclient.ScanResult, error) {
	s.got <- req
	return &trustclient.ScanResult{ScanID: 1}, nil
}

type grVerifier struct{}

func (grVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if raw == "gr-staff-token" {
		return &oauth.AccessClaims{ID: grStaff, Name: "staff", Roles: []string{"user", "ren"}, ClientID: "kungal-app"}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type grSync struct {
	workID      int
	token, play string
}

type grFix struct {
	app     *App
	db      *gorm.DB
	rdb     *redis.Client
	spec    *specConformance
	cat     *fakeCatalog
	plane   *grPlane
	checker *grChecker
	scanner *grScanner
	failOA  atomic.Bool
	mu      sync.Mutex
	awards  []awardCall
	syncs   []grSync

	recordWrites atomic.Int32
}

func newGRFix(t *testing.T) *grFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })
	f := &grFix{db: db, rdb: rdb, checker: &grChecker{}, scanner: &grScanner{got: make(chan trustclient.ScanRequest, 16)},
		cat: &fakeCatalog{rows: map[int]client.CatalogWorkListItem{}}, plane: newGRPlane()}

	names := map[int]string{grAlice: "alice", grBob: "bob", grBanned: "banned", grCreator: "creator", grStaff: "staff"}
	for i := range 12 {
		names[grRater+i] = "rater" + strconv.Itoa(i)
	}
	oa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		users := []map[string]any{}
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			name, ok := names[id]
			if !ok {
				continue
			}
			status := 0
			if id == grBanned {
				status = 1
			}
			users = append(users, map[string]any{"id": id, "name": name, "status": status, "roles": []string{"user"}})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": []int{}}})
	}))
	t.Cleanup(oa.Close)
	uc := userclient.New(userclient.Config{
		BaseURL: oa.URL, ClientID: "test-client", ClientSecret: "test-secret",
		ImageCDNBase: geCDN, HTTPTimeout: 2 * time.Second,
	})

	for _, w := range []geWork{
		{id: grWorkSFW, name: "Sakura", release: "2026-01-01", limit: "sfw", rating: "all_ages"},
		{id: grWorkSFW2, name: "Tsubaki", release: "2025-06", limit: "sfw", rating: "all_ages"},
		{id: grWorkNSFW, name: "Yoru", release: "2024", limit: "nsfw", rating: "r18"},
		{id: grWorkFresh, name: "Hajime", release: "2026-02-02", limit: "sfw", rating: "all_ages"},
	} {
		var row client.CatalogWorkListItem
		decodeInto(t, geRowJSON(w), &row)
		if w.id == grWorkSFW {
			decodeInto(t, `{"tags":[
				{"canonical_id":21,"display_name":"Island","kind":"content","spoiler":0,"sexual":false},
				{"canonical_id":22,"display_name":"Twist","kind":"content","spoiler":2,"sexual":false},
				{"canonical_id":23,"display_name":"Adult","kind":"content","spoiler":0,"sexual":true}]}`, &row)
		}
		f.cat.rows[w.id] = row
	}

	helpers := galgameService.InteractionHelpers{}
	svc := ratingapiv1.New(ratingapiv1.Deps{
		Store:  galgameRepo.NewRatingStore(db),
		Rows:   f.cat,
		Works:  workrepr.NewHydrator(f.cat, db, geCDN),
		Users:  uc,
		Check:  gate.NewCheckService(f.checker),
		Scan:   gate.NewScanService(f.scanner),
		Award:  f.recordAward,
		Notify: helpers.CreateRatingLikedMessage,
		Sync: func(_ context.Context, workID int, token, play string) {
			f.plane.setState(token, workID, play)
			f.mu.Lock()
			defer f.mu.Unlock()
			f.syncs = append(f.syncs, grSync{workID, token, play})
		},
		Catalog:       f.plane,
		Mirror:        galgameService.NewRatingMirror(nil, nil, nil, []string{grForumClient}).Row,
		RecordWritten: func(context.Context, int) { f.recordWrites.Add(1) },
		CDN:           geCDN,
	})
	f.app = &App{
		Fiber:           newFiber(),
		Config:          testConfig(),
		DB:              db,
		Redis:           rdb,
		Authn:           middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(grVerifier{}, rdb, nil)),
		GalgameRatingV1: svc,
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	for token, uid := range map[string]int{
		"gr-alice": grAlice, "gr-bob": grBob, "gr-banned": grBanned, "gr-creator": grCreator,
	} {
		f.putSession(t, token, uid, "user")
	}
	f.putSession(t, "gr-staff", grStaff, "user", "ren")
	return f
}

func (f *grFix) recordAward(userID, delta int, reason, ref, key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.awards = append(f.awards, awardCall{userID, delta, reason, ref, key})
}

// takeAwards returns the awards since the last call and forgets them.
func (f *grFix) takeAwards() []awardCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.awards
	f.awards = nil
	return out
}

func (f *grFix) nextScan(t *testing.T) trustclient.ScanRequest {
	t.Helper()
	select {
	case req := <-f.scanner.got:
		return req
	case <-time.After(2 * time.Second):
		t.Fatal("no scan fired")
		return trustclient.ScanRequest{}
	}
}

func (f *grFix) takeSyncs() []grSync {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.syncs
	f.syncs = nil
	return out
}

func (f *grFix) putSession(t *testing.T, token string, userID int, roles ...string) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "n", Roles: roles},
		OAuthAccessToken: "access-" + token,
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(token), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *grFix) run(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func (f *grFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })
	now := time.Now()
	f.run(t, `INSERT INTO galgame (id, updated, creator_user_id, content_limit, published) VALUES
		(?, ?, ?, 'sfw', true), (?, ?, NULL, 'sfw', true), (?, ?, NULL, 'nsfw', true), (?, ?, NULL, 'sfw', true)`,
		grWorkSFW, now, grCreator, grWorkSFW2, now, grWorkNSFW, now, grWorkHidden, now)
	for _, r := range grRatings() {
		gt, _ := json.Marshal(r.gameTypes)
		s := r.scores
		state, completion, _ := playstate.ToCatalog(r.playStatus)
		f.run(t, `INSERT INTO galgame_rating (id, work_id, user_id, recommend, overall, view, galgame_type, author_state, author_completion,
			short_summary, spoiler_level, art, story, music, character, route, system, voice, replay_value,
			client_id, site, written_here, created, updated)
			VALUES (?, ?, ?, 'yes', ?, ?, ?::jsonb, ?, ?, 'short', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'kungal', true, ?, ?)`,
			r.id, r.workID, r.userID, r.overall, r.view, string(gt), state, completion, r.spoiler,
			s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7], grForumClient, r.created, r.created)
		f.plane.seed(r)
	}
}

func (f *grFix) cleanup(t *testing.T) {
	t.Helper()
	f.run(t, `DELETE FROM message WHERE sender_id BETWEEN ? AND ? OR receiver_id BETWEEN ? AND ?`, grAlice, grUserMax, grAlice, grUserMax)
	f.run(t, `DELETE FROM galgame_rating WHERE work_id BETWEEN ? AND ? OR id BETWEEN ? AND ?`, grWorkSFW, grWorkMax, grRatingAlice, grRatingMax)
	f.run(t, `DELETE FROM galgame_rating_held WHERE id BETWEEN ? AND ?`, grRatingAlice, grRatingMax)
	f.run(t, `DELETE FROM galgame_rating_gone_retry WHERE rating_id BETWEEN ? AND ?`, grRatingAlice, grRatingMax)
	f.run(t, `DELETE FROM galgame WHERE id BETWEEN ? AND ?`, grWorkSFW, grWorkMax)
}

func (f *grFix) do(t *testing.T, method, rawURL, session, specPath, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	var r io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, rawURL, r)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	switch {
	case strings.HasPrefix(session, "bearer:"):
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(session, "bearer:"))
	case session != "":
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, method, specPath, resp, body)
	var out map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("json: %v\n%s", err, body)
		}
	}
	return resp, out
}

func (f *grFix) list(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodGet, "/api/v1/ratings?"+query, session, "/ratings", "", nil)
}

func (f *grFix) onRating(t *testing.T, method, session string, id int, suffix string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	u := "/api/v1/ratings/" + strconv.Itoa(id) + suffix
	return f.do(t, method, u, session, "/ratings/{rating_id}"+suffix, "", payload)
}

func (f *grFix) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Row().Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// grExpected is the order a listing must produce: the ratings the reader may
// see, sorted by the column with the id breaking ties in the same direction.
func grExpected(includeNSFW bool, column string, desc bool, keep func(grRating) bool) (ids []string, total int) {
	var rows []grRating
	for _, r := range grRatings() {
		if r.workID == grWorkNSFW && !includeNSFW {
			continue
		}
		if keep != nil && !keep(r) {
			continue
		}
		total++
		if r.userID == grBanned || r.workID == grWorkHidden {
			continue
		}
		rows = append(rows, r)
	}
	key := func(r grRating) int64 {
		switch column {
		case "view":
			return int64(r.view)
		case "overall":
			return int64(r.overall)
		}
		return r.created.Unix()
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if key(a) != key(b) {
			return (key(a) > key(b)) == desc
		}
		return (a.id > b.id) == desc
	})
	for _, r := range rows {
		ids = append(ids, strconv.Itoa(r.id))
	}
	return ids, total
}

const grForumClient = "forum-client"

type grKey struct{ user, work int64 }

// grPlane is NextMoe's rating plane as the forum sees it: one rating per user
// and work, its author's play state read live, ids that survive edits.
type grPlane struct {
	mu         sync.Mutex
	next       int64
	ratings    map[grKey]*catalogclient.Rating
	states     map[grKey]catalogclient.WorkStateRecord
	minutes    map[grKey]int
	moderators map[string]bool
	clients    map[string]string
	fail       atomic.Bool
	lastRecord catalogclient.WorkRecordInput
}

func newGRPlane() *grPlane {
	return &grPlane{
		next:       grRatingRaters + 40,
		ratings:    map[grKey]*catalogclient.Rating{},
		states:     map[grKey]catalogclient.WorkStateRecord{},
		minutes:    map[grKey]int{},
		moderators: map[string]bool{"access-gr-staff": true},
		clients:    map[string]string{},
	}
}

var grTokenUsers = map[string]int64{
	"access-gr-alice": grAlice, "access-gr-bob": grBob, "access-gr-banned": grBanned,
	"access-gr-creator": grCreator, "access-gr-staff": grStaff, "gr-staff-token": grStaff,
	"access-gr-fresh": grBob + 900,
}

func (p *grPlane) seed(r grRating) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := grKey{int64(r.userID), int64(r.workID)}
	sc := func(v int) *int {
		if v == 0 {
			return nil
		}
		return &v
	}
	s := r.scores
	p.ratings[k] = &catalogclient.Rating{
		ID: int64(r.id), WorkID: int64(r.workID), UserID: int64(r.userID), Overall: r.overall,
		Aspects: catalogclient.RatingAspects{Art: sc(s[0]), Story: sc(s[1]), Music: sc(s[2]), Character: sc(s[3]),
			Route: sc(s[4]), System: sc(s[5]), Voice: sc(s[6]), ReplayValue: sc(s[7])},
		Recommend: "yes", SpoilerLevel: r.spoiler, ShortSummary: "short", ClientID: grForumClient, Site: "kungal",
		CreatedAt: r.created, UpdatedAt: r.created,
	}
	state, completion, _ := playstate.ToCatalog(r.playStatus)
	p.states[k] = catalogclient.WorkStateRecord{WorkID: int64(r.workID), State: state, Completion: completion}
}

func (p *grPlane) user(token string) (int64, error) {
	if p.fail.Load() {
		return 0, catalogclient.ErrUpstream
	}
	uid, ok := grTokenUsers[token]
	if !ok {
		return 0, catalogclient.ErrUnauthorized
	}
	return uid, nil
}

func (p *grPlane) setState(token string, workID int, flat string) {
	uid, err := p.user(token)
	state, completion, ok := playstate.ToCatalog(flat)
	if err != nil || !ok {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.states[grKey{uid, int64(workID)}] = catalogclient.WorkStateRecord{WorkID: int64(workID), State: state, Completion: completion}
}

// view copies a rating with its author's current state, as NextMoe answers it.
func (p *grPlane) view(k grKey) *catalogclient.Rating {
	r, ok := p.ratings[k]
	if !ok {
		return nil
	}
	out := *r
	out.WorkState = nil
	if st, ok := p.states[k]; ok {
		out.WorkState = &st
	}
	return &out
}

func (p *grPlane) MyRating(_ context.Context, token string, workID int64) (*catalogclient.Rating, error) {
	uid, err := p.user(token)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.view(grKey{uid, workID}), nil
}

func (p *grPlane) put(token string, uid, workID int64, in catalogclient.RatingInput) *catalogclient.Rating {
	k := grKey{uid, workID}
	now := time.Now().UTC().Truncate(time.Millisecond)
	r, ok := p.ratings[k]
	if !ok {
		client := p.clients[token]
		if client == "" {
			client = grForumClient
		}
		p.next++
		r = &catalogclient.Rating{ID: p.next, WorkID: workID, UserID: uid, ClientID: client, Site: "kungal", CreatedAt: now}
		p.ratings[k] = r
	}
	r.Overall, r.Recommend, r.ShortSummary = in.Overall, in.Recommend, in.ShortSummary
	r.SpoilerLevel = in.SpoilerLevel
	if r.SpoilerLevel == "" {
		r.SpoilerLevel = "none"
	}
	r.Aspects = catalogclient.RatingAspects{}
	if in.AspectScores != nil {
		r.Aspects = *in.AspectScores
	}
	r.UpdatedAt = now
	return p.view(k)
}

func (p *grPlane) PutMyRating(_ context.Context, token string, workID int64, in catalogclient.RatingInput) (*catalogclient.Rating, error) {
	uid, err := p.user(token)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.put(token, uid, workID, in), nil
}

func (p *grPlane) DeleteMyRating(_ context.Context, token string, workID int64) error {
	uid, err := p.user(token)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.ratings, grKey{uid, workID})
	return nil
}

func (p *grPlane) ModerationDeleteRating(_ context.Context, token string, ratingID int64) error {
	if _, err := p.user(token); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.moderators[token] {
		return &catalogclient.UserAPIError{Status: http.StatusForbidden, ProblemCode: "PERMISSION_REQUIRED"}
	}
	for k, r := range p.ratings {
		if r.ID == ratingID {
			delete(p.ratings, k)
			return nil
		}
	}
	return catalogclient.ErrNotFound
}

func (p *grPlane) record(k grKey) *catalogclient.WorkRecord {
	rec := &catalogclient.WorkRecord{Rating: p.view(k)}
	if st, ok := p.states[k]; ok {
		rec.WorkState = &st
	}
	if m, ok := p.minutes[k]; ok {
		rec.Playtime = &catalogclient.PlaytimeSelf{WorkID: k.work, Minutes: m}
	}
	return rec
}

func (p *grPlane) WorkRecord(_ context.Context, token string, workID int64) (*catalogclient.WorkRecord, error) {
	uid, err := p.user(token)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.record(grKey{uid, workID}), nil
}

func (p *grPlane) PutWorkRecord(_ context.Context, token string, workID int64, in catalogclient.WorkRecordInput) (*catalogclient.WorkRecord, error) {
	uid, err := p.user(token)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRecord = in
	k := grKey{uid, workID}
	delete(p.states, k)
	if s := in.WorkState; s != nil {
		var completion *string
		if s.Completion != "" {
			c := s.Completion
			completion = &c
		}
		p.states[k] = catalogclient.WorkStateRecord{WorkID: workID, State: s.State, Completion: completion}
	}
	delete(p.minutes, k)
	if pt := in.Playtime; pt != nil {
		p.minutes[k] = pt.Minutes
	}
	if in.Rating == nil {
		delete(p.ratings, k)
	} else {
		p.put(token, uid, workID, *in.Rating)
	}
	return p.record(k), nil
}
