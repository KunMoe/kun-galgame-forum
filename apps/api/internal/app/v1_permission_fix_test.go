package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	pmUserMin     = 930150001
	pmAdmin       = 930150001
	pmAdminPeer   = 930150002
	pmRen         = 930150003
	pmMod         = 930150004
	pmPlain       = 930150005
	pmRenTarget   = 930150006
	pmModTarget   = 930150007
	pmGrantee     = 930150008
	pmMissing     = 930150009
	pmUserMax     = 930150099
	pmMatrixPath  = "/api/v1/admin/role-permissions"
	pmMinePath    = "/api/v1/me/permissions"
	pmChangesPath = "/api/v1/admin/permission-changes"
	pmUserSpec    = "/admin/user-permissions/{user_id}"
)

var pmAccounts = map[int][]string{
	pmAdmin:     {"user", "admin"},
	pmAdminPeer: {"user", "admin"},
	pmRen:       {"user", "ren"},
	pmMod:       {"user", "moderator"},
	pmPlain:     {"user"},
	pmRenTarget: {"user", "ren"},
	pmModTarget: {"user", "moderator"},
	pmGrantee:   {"user"},
}

type pmVerifier struct{}

func (pmVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "pm-grantee-token":
		return &oauth.AccessClaims{ID: pmGrantee, Name: "grantee", Roles: []string{"user"}, ClientID: "kungal-app"}, nil
	case "pm-admin-token":
		return &oauth.AccessClaims{ID: pmAdmin, Name: "admin", Roles: []string{"user", "admin"}, ClientID: "kungal-app"}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type permFix struct {
	*App
	db     *gorm.DB
	rdb    *redis.Client
	spec   *specConformance
	oaDown atomic.Bool
}

func newPermFix(t *testing.T) *permFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })
	f := &permFix{db: db, rdb: rdb}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.oaDown.Load() || r.URL.Path != "/users/batch" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		users := []map[string]any{}
		missing := []int{}
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			if roles, ok := pmAccounts[id]; ok {
				users = append(users, map[string]any{"id": id, "name": fmt.Sprintf("u%d", id), "status": 0, "roles": roles})
			} else {
				missing = append(missing, id)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": missing}})
	}))
	t.Cleanup(upstream.Close)

	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = "https://image.test.example"
	f.App = &App{
		Fiber:  newFiber(),
		Config: cfg,
		DB:     db,
		Redis:  rdb,
		UserClient: userclient.New(userclient.Config{
			BaseURL: upstream.URL, ClientID: "test-client", ClientSecret: "test-secret",
			ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
		}),
		Authn: middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(pmVerifier{}, rdb, nil)),
	}
	f.setupRoutes()
	f.spec = newSpecConformance(t)

	f.clean(t)
	t.Cleanup(func() { f.clean(t) })
	for token, uid := range map[string]int{
		"sess-pm-admin": pmAdmin, "sess-pm-ren": pmRen, "sess-pm-mod": pmMod,
		"sess-pm-plain": pmPlain, "sess-pm-grantee": pmGrantee,
	} {
		f.session(t, token, uid)
	}
	return f
}

func (f *permFix) clean(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM role_permission_override`,
		`DELETE FROM user_permission_override WHERE user_id BETWEEN ? AND ?`,
		`DELETE FROM permission_audit_log WHERE operator_id BETWEEN ? AND ?`,
	} {
		args := []any{}
		if strings.Contains(q, "?") {
			args = []any{pmUserMin, pmUserMax}
		}
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	perm.SetOverrides(nil)
	perm.SetUserOverrides(nil)
}

func (f *permFix) session(t *testing.T, token string, uid int) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: uid, Name: "n", Roles: pmAccounts[uid]},
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

func (f *permFix) seedOverride(t *testing.T, subject any, p perm.Permission, effect string) {
	t.Helper()
	var err error
	switch s := subject.(type) {
	case string:
		err = f.db.Exec(`INSERT INTO role_permission_override (role, permission, effect) VALUES (?, ?, ?)`, s, string(p), effect).Error
	case int:
		err = f.db.Exec(`INSERT INTO user_permission_override (user_id, permission, effect) VALUES (?, ?, ?)`, s, string(p), effect).Error
	}
	if err != nil {
		t.Fatal(err)
	}
	f.reloadPerm(t)
}

func (f *permFix) reloadPerm(t *testing.T) {
	t.Helper()
	sync := adminService.NewPermissionOverrideSync(adminRepo.NewRolePermissionRepository(f.db), adminRepo.NewUserPermissionRepository(f.db))
	if err := sync.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type pmAuth struct {
	session string
	bearer  string
}

func (f *permFix) call(t *testing.T, method, rawURL, specPath string, who pmAuth, payload any) (*http.Response, map[string]any) {
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
	if who.session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: who.session})
	}
	if who.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+who.bearer)
	}
	resp, err := f.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, method, specPath, resp, body)
	return resp, problemMap(t, body)
}

func (f *permFix) patchMatrix(t *testing.T, session string, changes ...map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodPatch, pmMatrixPath, "/admin/role-permissions", pmAuth{session: session}, map[string]any{"changes": changes})
}

func pmChange(role string, overrides ...map[string]any) map[string]any {
	if overrides == nil {
		overrides = []map[string]any{}
	}
	return map[string]any{"role": role, "overrides": overrides}
}

func pmOverride(p perm.Permission, effect string) map[string]any {
	return map[string]any{"permission": string(p), "effect": effect}
}

func pmStrings(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, len(raw))
	for i, x := range raw {
		out[i], _ = x.(string)
	}
	return out
}

func pmLayer(t *testing.T, body map[string]any, role string) map[string]any {
	t.Helper()
	layers, _ := body["role_permissions"].([]any)
	for _, l := range layers {
		m, _ := l.(map[string]any)
		if m["role"] == role {
			return m
		}
	}
	t.Fatalf("no %s layer in %+v", role, body)
	return nil
}

func pmHas(list any, p perm.Permission) bool {
	for _, s := range pmStrings(list) {
		if s == string(p) {
			return true
		}
	}
	return false
}

func pmRefused(t *testing.T, resp *http.Response, body map[string]any, locKey, loc, reason string) {
	t.Helper()
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	errs, _ := body["errors"].([]any)
	for _, e := range errs {
		m, _ := e.(map[string]any)
		if m[locKey] == loc && m["reason"] == reason {
			return
		}
	}
	t.Fatalf("no %s at %s=%s in %+v", reason, locKey, loc, errs)
}

func (f *permFix) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
