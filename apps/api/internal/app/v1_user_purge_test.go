package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/communityclient"
)

const (
	u3cTarget    = 930450001
	u3cStaffUser = 930450002
	u3cBanned    = 930450003
	u3cGone      = 930450004
	u3cTopic     = 930450101
	u3cReply     = 930450201
	u3cLottery   = 930450301
)

type purgeFix struct {
	*writeFix
	communityDown  atomic.Bool
	purgeFails     atomic.Bool
	catalogStatus  atomic.Int32
	communityPurge atomic.Int32
	folderPurge    atomic.Int32
	folderToken    atomic.Value
}

func newPurgeFix(t *testing.T) *purgeFix {
	t.Helper()
	f := &purgeFix{writeFix: newWriteFix(t, nil)}
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserStaff, "user", "admin")
	f.addOAuthUser(u3cTarget, "u3c-target", 0, nil)
	f.addOAuthUser(u3cStaffUser, "u3c-staff", 0, map[string]any{"roles": []string{"moderator"}})
	f.addOAuthUser(u3cBanned, "u3c-banned", 1, nil)

	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		write := func(data any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/authors/stats":
			if f.communityDown.Load() {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			stats := []map[string]any{}
			for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
				if id, _ := strconv.Atoi(raw); id == u3cTarget {
					stats = append(stats, map[string]any{"author_id": id, "visible_posts": 4})
				}
			}
			write(map[string]any{"stats": stats})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/authors/") && strings.HasSuffix(r.URL.Path, "/purge"):
			if f.purgeFails.Load() {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			f.communityPurge.Add(1)
			write(map[string]any{"posts_purged": 4, "reactions_deleted": 1})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(community.Close)
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !strings.HasPrefix(r.URL.Path, "/v2/moderation/users/") {
			http.NotFound(w, r)
			return
		}
		if status := int(f.catalogStatus.Load()); status != 0 {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"code": "PERMISSION_REQUIRED", "status": ` + strconv.Itoa(status) + `}`))
			return
		}
		f.folderPurge.Add(1)
		f.folderToken.Store(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"folders_deleted": 1, "items_deleted": 2}`))
	}))
	t.Cleanup(catalog.Close)

	f.AdminPurge = adminService.NewPurgeService(adminRepo.NewPurgeRepository(f.db), f.UserClient,
		communityclient.New(communityclient.Config{BaseURL: community.URL, ClientID: "c", ClientSecret: "s"}),
		catalogclient.New(catalogclient.Config{BaseURL: catalog.URL, AppKey: "k"}))
	f.Fiber = newFiber()
	f.setupRoutes()
	f.spec = newSpecConformance(t)

	f.cleanupPurge(t)
	t.Cleanup(func() { f.cleanupPurge(t) })
	f.seedPurge(t)
	return f
}

func (f *purgeFix) cleanupPurge(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM user_purge_archive WHERE target_user_id BETWEEN 930450000 AND 930450999`,
		`DELETE FROM message WHERE sender_id BETWEEN 930450000 AND 930450999 OR receiver_id BETWEEN 930450000 AND 930450999`,
		`DELETE FROM topic_reply WHERE id = 930450201`,
		`DELETE FROM topic WHERE id = 930450101`,
		`DELETE FROM kungal_user_state WHERE user_id BETWEEN 930450000 AND 930450999`,
		`DELETE FROM feed_activity WHERE user_id BETWEEN 930450000 AND 930450999`,
	} {
		if err := f.db.Exec(q).Error; err != nil {
			t.Errorf("u3c cleanup: %v\n%s", err, q)
		}
	}
}

func (f *purgeFix) seedPurge(t *testing.T) {
	t.Helper()
	at := time.Now().AddDate(0, 0, -3)
	f.run(t, `INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, 10, ?, ?)`, u3cTarget, at, at)
	f.run(t, `INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, 'u3c spam', 'body', 0, 0, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		u3cTopic, at, at, at, u3cTarget)
	f.run(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'spam reply', 9, ?, ?, 0, 0, ?, ?)`, u3cReply, u3cTarget, w3TopicPub, at, at)
	f.run(t, `INSERT INTO message (sender_id, receiver_id, type, content, created, updated)
		VALUES (?, ?, 'liked', 'm', ?, ?)`, u3cTarget, w3UserAlice, at, at)
}

func (f *purgeFix) run(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("u3c seed: %v\n%s", err, q)
	}
}

func (f *purgeFix) setOAuthRoles(id int, roles []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.oauthExtra {
		if u["id"] == id {
			u["roles"] = roles
		}
	}
}

func (f *purgeFix) content(t *testing.T, session string, id string, hdr http.Header) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodGet, "/api/v1/admin/user-contents/"+id, session, "/admin/user-contents/{user_id}", "", hdr, nil)
	return resp, problemOrObject(t, body)
}

func (f *purgeFix) purge(t *testing.T, session string, id string, hdr http.Header) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodDelete, "/api/v1/admin/user-contents/"+id, session, "/admin/user-contents/{user_id}", "", hdr, nil)
	if resp.StatusCode == http.StatusNoContent && len(body) != 0 {
		t.Errorf("a 204 carried a body: %s", body)
	}
	return resp, problemOrObject(t, body)
}

func problemOrObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	if len(body) == 0 {
		return nil
	}
	return problemMap(t, body)
}

func (f *purgeFix) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *purgeFix) intact(t *testing.T) {
	t.Helper()
	if f.count(t, `SELECT count(*) FROM topic WHERE id = ?`, u3cTopic) != 1 ||
		f.count(t, `SELECT count(*) FROM topic_reply WHERE id = ?`, u3cReply) != 1 {
		t.Error("the target's content is gone")
	}
	if n := f.count(t, `SELECT count(*) FROM user_purge_archive WHERE target_user_id = ?`, u3cTarget); n != 0 {
		t.Errorf("%d archive rows for a purge that did not happen", n)
	}
	if f.communityPurge.Load() != 0 || f.folderPurge.Load() != 0 {
		t.Error("the remote purges ran")
	}
}

func TestV1UserContentNeedsThePermission(t *testing.T) {
	f := newPurgeFix(t)
	target := strconv.Itoa(u3cTarget)
	for _, c := range []struct {
		name, session string
		hdr           http.Header
		status        int
		code          string
	}{
		{"anonymous", "", nil, http.StatusUnauthorized, "MISSING_CREDENTIAL"},
		{"a bad Bearer", "", http.Header{"Authorization": {"Bearer nope"}}, http.StatusUnauthorized, "INVALID_CREDENTIAL"},
		{"a user", "sess-alice", nil, http.StatusForbidden, "PERMISSION_REQUIRED"},
		{"a moderator, who lacks user.purge_content", "sess-staff", nil, http.StatusForbidden, "PERMISSION_REQUIRED"},
		{"a Bearer moderator", "", http.Header{"Authorization": {"Bearer staff-token"}}, http.StatusForbidden, "PERMISSION_REQUIRED"},
	} {
		if resp, body := f.content(t, c.session, target, c.hdr); resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("preview as %s: %d %v, want %d %s", c.name, resp.StatusCode, body["code"], c.status, c.code)
		}
		if resp, body := f.purge(t, c.session, target, c.hdr); resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("purge as %s: %d %v, want %d %s", c.name, resp.StatusCode, body["code"], c.status, c.code)
		}
	}
	f.intact(t)
}

func TestV1UserContentPreview(t *testing.T) {
	f := newPurgeFix(t)
	resp, body := f.content(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview %d %+v", resp.StatusCode, body)
	}
	want := map[string]any{
		"object": "user_content", "id": strconv.Itoa(u3cTarget),
		"is_protected": false, "is_account_active": true,
		"topic_count": 1.0, "reply_count": 1.0, "message_count": 1.0, "resource_count": 0.0,
		"community_post_count": 4.0, "total_count": 3.0,
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("%s = %v, want %v", k, body[k], v)
		}
	}

	for _, c := range []struct {
		name   string
		id     int
		active bool
	}{{"a banned account", u3cBanned, false}, {"an account the account service does not know", u3cGone, false}} {
		resp, body := f.content(t, "sess-admin", strconv.Itoa(c.id), nil)
		if resp.StatusCode != http.StatusOK || body["is_account_active"] != c.active || body["total_count"] != 0.0 {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, body)
		}
	}

	f.communityDown.Store(true)
	if resp, body := f.content(t, "sess-admin", strconv.Itoa(u3cTarget), nil); resp.StatusCode != http.StatusOK || body["community_post_count"] != nil {
		t.Errorf("community down: %d community_post_count %v, want null", resp.StatusCode, body["community_post_count"])
	}

	if resp, body := f.content(t, "sess-admin", "99999999999", nil); resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("an id past int32: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.purge(t, "sess-admin", "99999999999", nil); resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("purge of an id past int32: %d %+v", resp.StatusCode, body)
	}
}

func TestV1PurgeUserContent(t *testing.T) {
	f := newPurgeFix(t)
	resp, body := f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("purge %d %+v", resp.StatusCode, body)
	}
	for _, q := range []string{
		`SELECT count(*) FROM topic WHERE user_id = 930450001`,
		`SELECT count(*) FROM topic_reply WHERE user_id = 930450001`,
		`SELECT count(*) FROM message WHERE sender_id = 930450001`,
		`SELECT count(*) FROM kungal_user_state WHERE user_id = 930450001`,
	} {
		if n := f.count(t, q); n != 0 {
			t.Errorf("after the purge %d rows: %s", n, q)
		}
	}
	var archived []struct {
		TableName  string
		OperatorID int
	}
	if err := f.db.Raw(`SELECT table_name, operator_id FROM user_purge_archive WHERE target_user_id = ?`, u3cTarget).Scan(&archived).Error; err != nil {
		t.Fatal(err)
	}
	tables := map[string]bool{}
	for _, a := range archived {
		tables[a.TableName] = true
		if a.OperatorID != w3UserStaff {
			t.Errorf("archive row of %s names operator %d, want %d", a.TableName, a.OperatorID, w3UserStaff)
		}
	}
	for _, tbl := range []string{"topic", "topic_reply", "message", "kungal_user_state"} {
		if !tables[tbl] {
			t.Errorf("the purge through the API archived nothing from %s (archived %v)", tbl, tables)
		}
	}
	if f.folderPurge.Load() != 1 || f.communityPurge.Load() != 1 {
		t.Errorf("remote purges: folders %d community %d, want 1 each", f.folderPurge.Load(), f.communityPurge.Load())
	}
	if tok, _ := f.folderToken.Load().(string); tok != "access" {
		t.Errorf("the catalog was asked with token %q, want the operator's own", tok)
	}

	resp, body = f.content(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusOK || body["total_count"] != 0.0 {
		t.Errorf("preview after the purge: %d %+v", resp.StatusCode, body)
	}
	if resp, _ = f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil); resp.StatusCode != http.StatusNoContent {
		t.Errorf("a second purge %d, want 204", resp.StatusCode)
	}
}

func TestV1PurgeRefusesStaff(t *testing.T) {
	f := newPurgeFix(t)
	resp, body := f.content(t, "sess-admin", strconv.Itoa(u3cStaffUser), nil)
	if resp.StatusCode != http.StatusOK || body["is_protected"] != true {
		t.Fatalf("staff preview %d %+v", resp.StatusCode, body)
	}
	f.setOAuthRoles(u3cTarget, []string{"moderator"})
	resp, body = f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusForbidden || body["code"] != "USER_PROTECTED" {
		t.Fatalf("purge of a moderator %d %+v", resp.StatusCode, body)
	}
	f.intact(t)
}

func TestV1PurgeReadsTheTargetsRolesFresh(t *testing.T) {
	f := newPurgeFix(t)
	if resp, body := f.content(t, "sess-admin", strconv.Itoa(u3cTarget), nil); resp.StatusCode != http.StatusOK || body["is_protected"] != false {
		t.Fatalf("preview %d %+v", resp.StatusCode, body)
	}
	f.setOAuthRoles(u3cTarget, []string{"moderator"})
	resp, body := f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusForbidden || body["code"] != "USER_PROTECTED" {
		t.Fatalf("purge right after a promotion: %d %+v, want USER_PROTECTED", resp.StatusCode, body)
	}
	f.intact(t)
}

func TestV1PurgeWhileTheAccountServiceIsDown(t *testing.T) {
	f := newPurgeFix(t)
	f.failOA.Store(true)
	if resp, body := f.content(t, "sess-admin", strconv.Itoa(u3cTarget), nil); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("preview %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("purge %d %+v", resp.StatusCode, body)
	}
	f.intact(t)
}

func TestV1PurgeRemoteFailureIsRetried(t *testing.T) {
	f := newPurgeFix(t)
	target := strconv.Itoa(u3cTarget)
	f.purgeFails.Store(true)
	resp, body := f.purge(t, "sess-admin", target, nil)
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("purge with community down %d %+v", resp.StatusCode, body)
	}
	if f.count(t, `SELECT count(*) FROM topic WHERE id = ?`, u3cTopic) != 0 {
		t.Error("the local part did not run before the community failed")
	}
	if f.count(t, `SELECT count(*) FROM user_purge_archive WHERE target_user_id = ?`, u3cTarget) == 0 {
		t.Error("the local part left no archive")
	}

	f.purgeFails.Store(false)
	f.catalogStatus.Store(http.StatusServiceUnavailable)
	if resp, body = f.purge(t, "sess-admin", target, nil); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("purge with the catalog down %d %+v", resp.StatusCode, body)
	}
	f.catalogStatus.Store(http.StatusForbidden)
	if resp, body = f.purge(t, "sess-admin", target, nil); resp.StatusCode != http.StatusInternalServerError || body["code"] != "INTERNAL_ERROR" {
		t.Fatalf("purge the catalog refuses: %d %+v, want INTERNAL_ERROR since a retry cannot fix it", resp.StatusCode, body)
	}
	f.catalogStatus.Store(0)
	if resp, body = f.purge(t, "sess-admin", target, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("retry %d %+v", resp.StatusCode, body)
	}
	if f.communityPurge.Load() != 1 {
		t.Errorf("community purged %d times, want 1", f.communityPurge.Load())
	}
}

func TestV1PurgeWaitsForADraw(t *testing.T) {
	f := newPurgeFix(t)
	f.run(t, `INSERT INTO topic_lottery (id, topic_id, user_id, title, status, created, updated) VALUES (?, ?, ?, 'u3c', 'drawing', now(), now())`,
		u3cLottery, u3cTopic, u3cTarget)
	resp, body := f.purge(t, "sess-admin", strconv.Itoa(u3cTarget), nil)
	if resp.StatusCode != http.StatusConflict || body["code"] != "LOTTERY_DRAWN" {
		t.Fatalf("purge during a draw %d %+v", resp.StatusCode, body)
	}
	f.intact(t)
}
