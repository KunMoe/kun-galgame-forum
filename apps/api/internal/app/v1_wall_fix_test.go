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

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/user/oauth"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	wallRepo "kun-galgame-api/internal/wall/repository"
	websiteapiv1 "kun-galgame-api/internal/website/apiv1"
	websiteRepo "kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	rcAlice   = 950000001
	rcBob     = 950000002
	rcStaff   = 950000003
	rcBanned  = 950000004
	rcOther   = 950000005
	rcGalMod  = 950000006
	rcCarol   = 950000007
	rcCreator = 950000008

	rcGalgame        = 950000101
	rcGalgameMissing = 950000102
	rcRating         = 950000201
	rcRatingBanned   = 950000202
	rcResource       = 950000301
	rcToolsetA       = 950000401
	rcToolsetB       = 950000402
	rcQuizOpen       = 950000501
	rcQuizSpoiler    = 950000502
	rcWebsite        = 950000601
	rcCategory       = 950000701

	rcWebsiteSlug = "rc-wall.example"
)

type fakePost struct {
	communityclient.PostView
	anchorKind int32
	anchorID   string
}

type fakeThread struct {
	id      int64
	kind    int32
	anchor  string
	highest int32
}

type fakeFlag struct {
	postID, flagger int64
	reason          int32
	note            string
}

// fakeCommunity is the community service as far as the walls use it: threads
// keyed by anchor, posts numbered per thread, like reactions, flags.
type fakeCommunity struct {
	mu          sync.Mutex
	threads     map[string]*fakeThread
	posts       map[int64]*fakePost
	reactions   map[[2]int64]bool
	flags       []fakeFlag
	follows     map[int64]bool
	nextPost    int64
	nextThr     int64
	comments    atomic.Int32
	toggles     atomic.Int32
	authorPosts atomic.Int32
	resolves    atomic.Int32
	down        atomic.Bool
	limited     atomic.Bool
	closed      atomic.Bool
	bogus       atomic.Bool
	lastReq     communityclient.CommentRequest

	anchorSubs  map[int64]map[string]int32
	threadSubs  map[[2]int64]int32
	reads       [][2]int64
	levelWrites []levelWrite

	userFollows map[int64]map[int64]string
	followCap   atomic.Bool
	followPuts  atomic.Int32
	followLists atomic.Int32
}

type levelWrite struct {
	scope string
	user  int64
	key   string
	level int32
}

func newFakeCommunity() *fakeCommunity {
	return &fakeCommunity{
		threads: map[string]*fakeThread{}, posts: map[int64]*fakePost{},
		reactions: map[[2]int64]bool{}, follows: map[int64]bool{},
		nextPost: 7_000_000, nextThr: 800_000,
		anchorSubs: map[int64]map[string]int32{}, threadSubs: map[[2]int64]int32{},
		userFollows: map[int64]map[int64]string{},
	}
}

func anchorKey(kind int32, id string) string { return strconv.Itoa(int(kind)) + "|" + id }

func (c *fakeCommunity) thread(kind int32, anchor string) *fakeThread {
	key := anchorKey(kind, anchor)
	th, ok := c.threads[key]
	if !ok {
		c.nextThr++
		th = &fakeThread{id: c.nextThr, kind: kind, anchor: anchor}
		c.threads[key] = th
	}
	return th
}

// seed writes a post straight into the fake, as an import or another client would.
func (c *fakeCommunity) seed(kind int32, anchor string, author int64, body string, status int32, replyTo int64, created time.Time) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.insert(kind, anchor, communityclient.CommentRequest{AuthorID: author, Body: body, ReplyToPostID: replyTo}, status, created)
}

func (c *fakeCommunity) insert(kind int32, anchor string, req communityclient.CommentRequest, status int32, created time.Time) int64 {
	th := c.thread(kind, anchor)
	th.highest++
	c.nextPost++
	p := &fakePost{anchorKind: kind, anchorID: anchor}
	p.ID, p.ThreadID, p.PostNumber = c.nextPost, th.id, th.highest
	p.AuthorID, p.ContentRaw, p.Status = req.AuthorID, req.Body, status
	p.CreatedAt = created.UTC().Format(time.RFC3339)
	p.TargetUserID = req.TargetUserID
	if req.ReplyToPostID != 0 {
		parent := c.posts[req.ReplyToPostID]
		p.ReplyToPostID = parent.ID
		p.RootPostID = parent.ID
		if parent.RootPostID != 0 {
			p.RootPostID = parent.RootPostID
		}
		p.TargetUserID = parent.AuthorID
	}
	c.posts[p.ID] = p
	return p.ID
}

func (c *fakeCommunity) post(id int64) communityclient.PostView {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.posts[id].PostView
}

func (c *fakeCommunity) threadView(th *fakeThread) communityclient.ThreadView {
	return communityclient.ThreadView{ID: th.id, Site: "kungal", AnchorKind: th.kind, AnchorID: th.anchor, HighestPostNumber: th.highest}
}

func writeEnvelope(w http.ResponseWriter, status int, code int, msg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg, "data": data})
}

func (c *fakeCommunity) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.down.Load() {
		writeEnvelope(w, http.StatusBadGateway, 50000, "upstream down", nil)
		return
	}
	if c.bogus.Load() {
		writeEnvelope(w, http.StatusBadRequest, 40000, "bad request", nil)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && path == "/comments":
		q := r.URL.Query()
		kind, _ := strconv.Atoi(q.Get("anchor_kind"))
		th, ok := c.threads[anchorKey(int32(kind), q.Get("anchor_id"))]
		if !ok {
			writeEnvelope(w, 200, 0, "", communityclient.CommentsPage{Posts: []communityclient.PostView{}})
			return
		}
		after, _ := strconv.Atoi(q.Get("after"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		var rows []communityclient.PostView
		for _, p := range c.posts {
			if p.ThreadID == th.id && int(p.PostNumber) > after {
				rows = append(rows, p.PostView)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].PostNumber < rows[j].PostNumber })
		if len(rows) > limit {
			rows = rows[:limit]
		}
		next := ""
		if len(rows) == limit {
			next = strconv.Itoa(int(rows[len(rows)-1].PostNumber))
		}
		view := c.threadView(th)
		writeEnvelope(w, 200, 0, "", communityclient.CommentsPage{Thread: &view, Posts: rows, NextCursor: next})
	case r.Method == http.MethodPost && path == "/comments":
		var req communityclient.CommentRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if c.limited.Load() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if c.closed.Load() {
			writeEnvelope(w, http.StatusConflict, 40900, "thread is not open", nil)
			return
		}
		if strings.Contains(req.Body, "BLOCKED") {
			writeEnvelope(w, http.StatusUnprocessableEntity, 40001, "content blocked by word list", nil)
			return
		}
		c.comments.Add(1)
		c.lastReq = req
		status := int32(communityclient.PostVisible)
		if strings.Contains(req.Body, "HOLDME") {
			status = communityclient.PostHeld
		}
		id := c.insert(req.AnchorKind, req.AnchorID, req, status, time.Now())
		p := c.posts[id]
		writeEnvelope(w, 200, 0, "", communityclient.ThreadWithPost{
			Thread: c.threadView(c.threads[anchorKey(p.anchorKind, p.anchorID)]), Post: p.PostView,
		})
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/authors/") && strings.HasSuffix(path, "/posts"):
		c.authorPosts.Add(1)
		idStr := strings.TrimSuffix(strings.TrimPrefix(path, "/authors/"), "/posts")
		authorID, _ := strconv.ParseInt(idStr, 10, 64)
		q := r.URL.Query()
		after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 {
			limit = 20
		}
		limit = min(limit, 100)
		var kind *int32
		if k, err := strconv.Atoi(q.Get("anchor_kind")); err == nil && k >= 0 {
			v := int32(k)
			kind = &v
		}
		rows := []communityclient.AuthorPostView{}
		for _, p := range c.posts {
			if p.AuthorID != authorID || p.Status != communityclient.PostVisible {
				continue
			}
			if after > 0 && p.ID >= after {
				continue
			}
			if kind != nil && p.anchorKind != *kind {
				continue
			}
			rows = append(rows, communityclient.AuthorPostView{Post: p.PostView, Thread: communityclient.PostThreadContext{
				ThreadID: p.ThreadID, AnchorKind: p.anchorKind, AnchorID: p.anchorID,
			}})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Post.ID > rows[j].Post.ID })
		if len(rows) > limit {
			rows = rows[:limit]
		}
		next := ""
		if len(rows) == limit {
			next = strconv.FormatInt(rows[len(rows)-1].Post.ID, 10)
		}
		writeEnvelope(w, 200, 0, "", communityclient.AuthorPostsResponse{Posts: rows, NextCursor: next})
	case r.Method == http.MethodPost && path == "/posts/resolve":
		c.resolves.Add(1)
		var req communityclient.PostsResolveRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		out := []communityclient.AuthorPostView{}
		for _, id := range req.IDs {
			if p, ok := c.posts[id]; ok {
				out = append(out, communityclient.AuthorPostView{Post: p.PostView, Thread: communityclient.PostThreadContext{
					ThreadID: p.ThreadID, AnchorKind: p.anchorKind, AnchorID: p.anchorID,
				}})
			}
		}
		writeEnvelope(w, 200, 0, "", communityclient.PostsResolveResponse{Posts: out})
	case strings.HasPrefix(path, "/posts/"):
		c.postOp(w, r)
	case path == "/threads/states":
		var req communityclient.ThreadStatesRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		states := []communityclient.ThreadUserView{}
		for _, tid := range req.ThreadIDs {
			if level, ok := c.threadSubs[[2]int64{tid, req.UserID}]; ok {
				states = append(states, communityclient.ThreadUserView{ThreadID: tid, UserID: req.UserID, NotificationLevel: level})
			}
		}
		if len(states) == 0 && c.follows[req.UserID] {
			states = append(states, communityclient.ThreadUserView{ThreadID: req.ThreadIDs[0], UserID: req.UserID})
		}
		writeEnvelope(w, 200, 0, "", communityclient.ThreadStatesResponse{States: states})
	case path == "/anchors/states":
		var req communityclient.AnchorStatesRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		states := []communityclient.AnchorSubscriptionView{}
		for _, a := range req.Anchors {
			if level, ok := c.anchorSubs[req.UserID][anchorKey(a.AnchorKind, a.AnchorID)]; ok {
				states = append(states, communityclient.AnchorSubscriptionView{UserID: req.UserID, AnchorKind: a.AnchorKind, AnchorID: a.AnchorID, NotificationLevel: level})
			}
		}
		writeEnvelope(w, 200, 0, "", communityclient.AnchorStatesResponse{States: states})
	case r.Method == http.MethodPost && path == "/anchors/notification":
		var req communityclient.AnchorNotificationRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		key := anchorKey(req.AnchorKind, req.AnchorID)
		if c.anchorSubs[req.UserID] == nil {
			c.anchorSubs[req.UserID] = map[string]int32{}
		}
		c.anchorSubs[req.UserID][key] = req.Level
		c.levelWrites = append(c.levelWrites, levelWrite{"anchor", req.UserID, key, req.Level})
		writeEnvelope(w, 200, 0, "", communityclient.AnchorSubscriptionView{UserID: req.UserID, AnchorKind: req.AnchorKind, AnchorID: req.AnchorID, NotificationLevel: req.Level})
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/threads/"):
		idStr, action, _ := strings.Cut(strings.TrimPrefix(path, "/threads/"), "/")
		tid, _ := strconv.ParseInt(idStr, 10, 64)
		var highest int32
		for _, th := range c.threads {
			if th.id == tid {
				highest = th.highest
			}
		}
		switch action {
		case "notification":
			var req communityclient.ThreadNotificationRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			c.threadSubs[[2]int64{tid, req.UserID}] = req.Level
			c.levelWrites = append(c.levelWrites, levelWrite{"thread", req.UserID, idStr, req.Level})
			writeEnvelope(w, 200, 0, "", communityclient.ThreadUserView{ThreadID: tid, UserID: req.UserID, NotificationLevel: req.Level, HighestPostNumber: highest})
		case "read":
			var req communityclient.ThreadReadRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			c.reads = append(c.reads, [2]int64{tid, req.UserID})
			level, ok := c.threadSubs[[2]int64{tid, req.UserID}]
			if !ok {
				level = communityclient.NotificationNormal
				c.threadSubs[[2]int64{tid, req.UserID}] = level
			}
			writeEnvelope(w, 200, 0, "", communityclient.ThreadUserView{ThreadID: tid, UserID: req.UserID, NotificationLevel: level,
				LastReadPostNumber: min(req.LastReadPostNumber, highest), HighestPostNumber: highest})
		default:
			writeEnvelope(w, http.StatusNotFound, 40400, "no route "+path, nil)
		}
	case c.handleUserFollow(w, r):
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/users/") && strings.HasSuffix(path, "/anchor-subscriptions"):
		uid, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(path, "/users/"), "/anchor-subscriptions"), 10, 64)
		keys := make([]string, 0, len(c.anchorSubs[uid]))
		for k := range c.anchorSubs[uid] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		start, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		end := min(start+limit, len(keys))
		subs := []communityclient.AnchorSubscriptionView{}
		for _, k := range keys[min(start, len(keys)):end] {
			kindStr, anchorID, _ := strings.Cut(k, "|")
			kind, _ := strconv.Atoi(kindStr)
			subs = append(subs, communityclient.AnchorSubscriptionView{UserID: uid, AnchorKind: int32(kind), AnchorID: anchorID, NotificationLevel: c.anchorSubs[uid][k]})
		}
		next := ""
		if end < len(keys) {
			next = strconv.Itoa(end)
		}
		writeEnvelope(w, 200, 0, "", communityclient.AnchorSubscriptionListResponse{Subscriptions: subs, NextCursor: next})
	default:
		writeEnvelope(w, http.StatusNotFound, 40400, "no route "+path, nil)
	}
}

func (c *fakeCommunity) postOp(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/posts/")
	idStr, action, _ := strings.Cut(rest, "/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	p, ok := c.posts[id]
	if !ok {
		writeEnvelope(w, http.StatusNotFound, 40400, "post not found", nil)
		return
	}
	switch {
	case r.Method == http.MethodPatch && action == "":
		var req communityclient.EditPostRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !req.AsModerator && req.AuthorID != p.AuthorID {
			writeEnvelope(w, http.StatusForbidden, 40300, "not the post author", nil)
			return
		}
		if strings.Contains(req.Body, "BLOCKED") {
			writeEnvelope(w, http.StatusUnprocessableEntity, 40001, "content blocked by word list", nil)
			return
		}
		p.ContentRaw = req.Body
		p.EditedAt = time.Now().UTC().Format(time.RFC3339)
		p.EditedByModerator = req.AsModerator
		writeEnvelope(w, 200, 0, "", map[string]any{"post": p.PostView})
	case r.Method == http.MethodDelete && action == "":
		author, _ := strconv.ParseInt(r.URL.Query().Get("author_id"), 10, 64)
		if r.URL.Query().Get("as_moderator") != "true" && author != p.AuthorID {
			writeEnvelope(w, http.StatusForbidden, 40300, "not the post author", nil)
			return
		}
		p.Status = communityclient.PostDeleted
		p.ContentRaw = ""
		writeEnvelope(w, 200, 0, "", nil)
	case r.Method == http.MethodPost && action == "reaction":
		var req communityclient.ReactionToggleRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		c.toggles.Add(1)
		key := [2]int64{id, req.UserID}
		added := !c.reactions[key]
		if added {
			c.reactions[key] = true
		} else {
			delete(c.reactions, key)
		}
		writeEnvelope(w, 200, 0, "", communityclient.ReactionToggleResult{Added: added, AuthorID: p.AuthorID, ThreadID: p.ThreadID})
	case r.Method == http.MethodPost && action == "flag":
		var req communityclient.FlagRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		c.flags = append(c.flags, fakeFlag{postID: id, flagger: req.FlaggerID, reason: req.Reason, note: req.Note})
		writeEnvelope(w, 200, 0, "", nil)
	default:
		writeEnvelope(w, http.StatusNotFound, 40400, "no route", nil)
	}
}

type rcVerifier struct{}

func (rcVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if raw == "rc-staff-token" {
		return &oauth.AccessClaims{ID: rcStaff, Name: "staff", Roles: []string{"user"}, SiteRoles: []string{"moderator"}, ClientID: "kungal-app"}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type wallFix struct {
	app    *App
	db     *gorm.DB
	rdb    *redis.Client
	spec   *specConformance
	cm     *fakeCommunity
	awards []awardCall
	mu     sync.Mutex
	failOA atomic.Bool
	failGC atomic.Bool

	resolveCalls atomic.Int32
	existCalls   atomic.Int32
}

func (f *wallFix) workRefs(_ context.Context, ids []int) (map[int]repr.WorkRef, error) {
	if f.failGC.Load() {
		return nil, fmt.Errorf("catalog down")
	}
	out := map[int]repr.WorkRef{}
	for _, id := range ids {
		if id == rcGalgame {
			out[id] = repr.NewWorkRef(id, repr.NewCatalogName("RC Game", "RC Geemu", nil), nil, false)
		}
	}
	return out, nil
}

func newWallFix(t *testing.T) *wallFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	f := &wallFix{db: db, rdb: rdb, cm: newFakeCommunity()}
	cmSrv := httptest.NewServer(f.cm)
	t.Cleanup(cmSrv.Close)

	names := map[int]string{
		rcAlice: "alice", rcBob: "bob", rcStaff: "staff", rcOther: "other",
		rcGalMod: "galmod", rcCarol: "carol", rcCreator: "creator", rcBanned: "banned",
	}
	oa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Only the ids asked for, so the client's cache holds exactly what the
		// code under test looked up.
		users := []map[string]any{}
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			name, ok := names[id]
			if !ok {
				continue
			}
			status := 0
			if id == rcBanned {
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
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	community := communityclient.New(communityclient.Config{BaseURL: cmSrv.URL, ClientID: "c", ClientSecret: "s"})
	convert := &content.Converter{CDNBase: "https://image.test.example", SiteBase: "https://www.kungal.com", Users: uc.Users}
	resolve := func(_ context.Context, workID int) (bool, error) {
		f.resolveCalls.Add(1)
		if f.failGC.Load() {
			return false, fmt.Errorf("catalog down")
		}
		return workID == rcGalgame, nil
	}
	exist := func(_ context.Context, workIDs []int) (map[int]bool, error) {
		f.existCalls.Add(1)
		if f.failGC.Load() {
			return nil, fmt.Errorf("catalog down")
		}
		out := map[int]bool{}
		for _, id := range workIDs {
			if id == rcGalgame {
				out[id] = true
			}
		}
		return out, nil
	}
	f.app = &App{
		Fiber:      newFiber(),
		Config:     testConfig(),
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		Community:  community,
		Authn:      middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(rcVerifier{}, rdb, nil)),
		WallV1: wallapiv1.New(wallRepo.NewStore(db), community, uc, convert, resolve, exist, f.recordAward, "https://image.test.example").
			WithFollowing(msgRepo.NewMessageRepository(db).MarkCommunityThreadRead, f.workRefs,
				websiteapiv1.New(websiteRepo.NewStore(db), uc, nil, "https://image.test.example").SummariesByIDs),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	for token, uid := range map[string]int{
		"rc-alice": rcAlice, "rc-bob": rcBob, "rc-other": rcOther, "rc-galmod": rcGalMod,
		"rc-carol": rcCarol, "rc-creator": rcCreator,
	} {
		f.putSession(t, token, uid, "user")
	}
	f.putSession(t, "rc-staff", rcStaff, "user", "moderator")
	f.putSession(t, "rc-banned", rcBanned, "user")
	return f
}

func (f *wallFix) recordAward(userID, delta int, reason, ref, key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.awards = append(f.awards, awardCall{userID, delta, reason, ref, key})
}

func (f *wallFix) awardsSnapshot() []awardCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]awardCall(nil), f.awards...)
}

func (f *wallFix) putSession(t *testing.T, token string, userID int, roles ...string) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "n", Roles: roles},
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(token), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *wallFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	now := time.Now()
	run(`INSERT INTO galgame (id, updated, creator_user_id) VALUES (?, ?, ?)`, rcGalgame, now, rcCreator)
	run(`INSERT INTO galgame_rating (id, work_id, user_id, recommend, overall, updated) VALUES (?, ?, ?, 'yes', 8, ?), (?, ?, ?, 'yes', 8, ?)`,
		rcRating, rcGalgame, rcBob, now, rcRatingBanned, rcGalgame, rcBanned, now)
	run(`INSERT INTO galgame_resource (id, work_id, user_id, updated) VALUES (?, ?, ?, ?)`, rcResource, rcGalgame, rcBob, now)
	run(`INSERT INTO galgame_toolset (id, user_id, updated) VALUES (?, ?, ?), (?, ?, ?)`, rcToolsetA, rcBob, now, rcToolsetB, rcOther, now)
	run(`INSERT INTO galgame_quiz (id, user_id, type, question, spoiler_level) VALUES (?, ?, 'single', 'q', 'none'), (?, ?, 'single', 'q', 'serious')`,
		rcQuizOpen, rcBob, rcQuizSpoiler, rcBob)
	run(`INSERT INTO galgame_quiz_answer (quiz_id, user_id) VALUES (?, ?)`, rcQuizSpoiler, rcCarol)
	run(`INSERT INTO galgame_website_category (id, name, updated) VALUES (?, 'rc', ?)`, rcCategory, now)
	run(`INSERT INTO galgame_website (id, name, url, create_time, category_id, age_limit, updated) VALUES (?, 'rc', ?, '2020', ?, 'r18', ?)`,
		rcWebsite, rcWebsiteSlug, rcCategory, now)
}

func (f *wallFix) cleanup(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM message WHERE sender_id BETWEEN 950000001 AND 950000099 OR receiver_id BETWEEN 950000001 AND 950000099`,
		`DELETE FROM feed_activity WHERE user_id BETWEEN 950000001 AND 950000099`,
		`DELETE FROM galgame_post_like WHERE user_id BETWEEN 950000001 AND 950000099`,
		`DELETE FROM galgame_quiz_answer WHERE quiz_id BETWEEN 950000501 AND 950000599`,
		`DELETE FROM galgame_quiz WHERE id BETWEEN 950000501 AND 950000599`,
		`DELETE FROM galgame_toolset WHERE id BETWEEN 950000401 AND 950000499`,
		`DELETE FROM galgame_resource WHERE id BETWEEN 950000301 AND 950000399`,
		`DELETE FROM galgame_rating WHERE id BETWEEN 950000201 AND 950000299`,
		`DELETE FROM galgame_website WHERE id BETWEEN 950000601 AND 950000699`,
		`DELETE FROM galgame_website_category WHERE id BETWEEN 950000701 AND 950000799`,
		`DELETE FROM galgame WHERE id BETWEEN 950000101 AND 950000199`,
	} {
		if err := f.db.Exec(q).Error; err != nil {
			t.Fatalf("cleanup: %v\n%s", err, q)
		}
	}
}

// do sends one request and checks the response against the committed spec.
func (f *wallFix) do(t *testing.T, method, rawURL, session, specPath, idem string, payload any) (*http.Response, map[string]any) {
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

func (f *wallFix) list(t *testing.T, session, subjectType string, subjectID int, extra string) (*http.Response, map[string]any) {
	t.Helper()
	u := fmt.Sprintf("/api/v1/wall-comments?subject_type=%s&subject_id=%d%s", subjectType, subjectID, extra)
	return f.do(t, http.MethodGet, u, session, "/wall-comments", "", nil)
}

func (f *wallFix) create(t *testing.T, session, idem string, body map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodPost, "/api/v1/wall-comments", session, "/wall-comments", idem, body)
}

func (f *wallFix) onComment(t *testing.T, method, session string, id int64, suffix string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	u := "/api/v1/wall-comments/" + strconv.FormatInt(id, 10) + suffix
	return f.do(t, method, u, session, "/wall-comments/{wall_comment_id}"+suffix, "", payload)
}

func (f *wallFix) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Row().Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func itemIDs(t *testing.T, out map[string]any) []string {
	t.Helper()
	raw, _ := out["items"].([]any)
	ids := make([]string, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		ids = append(ids, m["id"].(string))
	}
	return ids
}

func wallCode(out map[string]any) string {
	s, _ := out["code"].(string)
	return s
}

func wallViewer(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	v, ok := out["viewer"].(map[string]any)
	if !ok {
		t.Fatalf("viewer %v", out["viewer"])
	}
	return v
}

func pid(id int64) string { return strconv.FormatInt(id, 10) }

const (
	anchorGame = communityclient.AnchorSiteGame
	anchorRes  = communityclient.AnchorSiteResource
)

func resAnchor(prefix string, id int) string { return prefix + ":" + strconv.Itoa(id) }
