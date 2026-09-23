package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type fakeNotifier struct {
	mu    sync.Mutex
	specs []msgService.Spec
}

func (f *fakeNotifier) Emit(_ *gorm.DB, spec msgService.Spec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.specs = append(f.specs, spec)
	return nil
}

func (f *fakeNotifier) EmitMany(tx *gorm.DB, specs []msgService.Spec) error {
	for _, s := range specs {
		_ = f.Emit(tx, s)
	}
	return nil
}

func fakeGalgame(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/catalog/works":
			_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"work","id":"1000",` +
				`"claim":{"site":"kungal","site_work_id":"1000","state":"live"},` +
				`"localized":{"zh-Hans":{"value":"测试游戏","is_machine":false}}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"title":"Not found","status":404,"code":"NOT_FOUND"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

type fakeOwners map[int]int

func (f fakeOwners) OwnerOf(workID int) int { return f[workID] }

type fakeEditFace struct {
	mu             sync.Mutex
	requests       []recordedRequest
	userStatus     int
	userBody       string
	userReviewable bool
}

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
	Face   string
	Auth   string
}

func s2sFace(path string) bool { return strings.HasPrefix(path, "/api/v1/catalog/edit/") }

func appFace(path string) bool {
	return strings.HasPrefix(path, "/v2/catalog/revisions") ||
		strings.HasPrefix(path, "/v2/catalog/proposals")
}

func userFace(path string) bool {
	return strings.HasPrefix(path, "/api/v1/user/catalog/") ||
		strings.HasPrefix(path, "/v2/me/") ||
		strings.HasPrefix(path, "/v2/moderation/") ||
		strings.HasPrefix(path, "/v2/catalog/schemas/")
}

func (f *fakeEditFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		face := "other"
		switch {
		case s2sFace(r.URL.Path):
			face = "s2s"
		case userFace(r.URL.Path):
			face = "user"
		}
		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body,
			Face: face, Auth: r.Header.Get("Authorization"),
		})
		f.mu.Unlock()
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if face == "user" && f.userStatus != 0 {
			w.WriteHeader(f.userStatus)
			_, _ = w.Write([]byte(f.userBody))
			return
		}
		if appFace(r.URL.Path) {
			face = "app"
		}
		schemaBody := func() string {
			review := "false"
			if f.userReviewable {
				review = "true"
			}
			return `{"object":"schema","entity_type":"catalog.work","fields":[` +
				`{"key":"catalog.work.name_zh_cn","field_type":"text","diff_hint":"inline","deprecated":false,"can_propose":true,"can_review":` + review + `},` +
				`{"key":"catalog.work.vndb_id","field_type":"text","diff_hint":"inline","deprecated":false,"can_propose":false,"can_review":false},` +
				`{"key":"catalog.work.legacy_alias","field_type":"text","diff_hint":"inline","deprecated":true,"can_propose":false,"can_review":false}` +
				`]}`
		}
		switch {
		case r.Method == "POST" && r.URL.Path == "/v2/me/proposals":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"7","entity_type":"catalog.work","entity_id":"1000","state":"open","proposer_uid":"9","patch":{}}`))
		case r.Method == "PATCH" && r.URL.Path == "/v2/me/proposals/7":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"7","entity_type":"catalog.work","entity_id":"1000","state":"withdrawn","patch":{}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/catalog/schemas/work":
			_, _ = w.Write([]byte(schemaBody()))
		case r.Method == "POST" && r.URL.Path == "/v2/me/proposals/7/amendments":
			_, _ = w.Write([]byte(`{"object":"amendment","id":"1","seq":1,"amender_uid":7,"set":{"catalog.work.name_zh_cn":"修正"}}`))
		case r.Method == "POST" && r.URL.Path == "/v2/moderation/proposals/7/decisions":
			if body["decision"] == "decline" {
				_, _ = w.Write([]byte(`{"object":"proposal","id":"7","entity_type":"catalog.work","entity_id":"1000","state":"declined","proposer_uid":"9","patch":{}}`))
				break
			}
			_, _ = w.Write([]byte(`{"object":"revision","id":"100","seq":2,"action":"merged","actor_uid":9,"amender_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}`))
		case r.Method == "POST" && r.URL.Path == "/v2/moderation/reverts":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"8","entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"merged","patch":{}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/catalog/revisions":
			_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"revision","id":"100","seq":1,"action":"created","actor_uid":"7","entity_id":"1000","target_object":"work","changed_fields":["catalog.work.name_zh_cn"]},{"object":"revision","id":"101","seq":3,"action":"merged","actor_uid":"7","entity_id":"1000","target_object":"work","changed_fields":["catalog.work.name_zh_cn"]}]}`))
		case strings.HasPrefix(r.URL.Path, "/v2/catalog/revisions/"):
			_, _ = w.Write([]byte(`{"object":"revision","id":"101","seq":3,"action":"merged","actor_uid":"7","entity_id":"1000","target_object":"work","diff":[{"key":"catalog.work.name_zh_cn","from":"旧","to":"新"}]}`))
		case r.Method == "GET" && r.URL.Path == "/v2/catalog/proposals":
			_, _ = w.Write([]byte(`{"object":"list","items":[],"total":0}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/proposals/7":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"7","entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"open","proposer_uid":"9","patch":{"catalog.work.name_zh_cn":"新标题"}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/proposals/8":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"8","entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"open","proposer_uid":"9","patch":{}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/proposals/9":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"9","entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"open","proposer_uid":"9",` +
				`"patch":{"catalog.work.name_zh_cn":"新标题","catalog.work.vndb_id":"v1"},` +
				`"effective_patch":{"catalog.work.name_zh_cn":"新标题"}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/proposals/55":
			_, _ = w.Write([]byte(`{"object":"proposal","id":"55","entity_type":"catalog.work","entity_id":"1000","site":"letmoe","state":"open","patch":{}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/snapshots/work/1000":
			_, _ = w.Write([]byte(`{"object":"snapshot","entity_type":"catalog.work","entity_id":"1000","field_values":{"catalog.work.name_zh_cn":"现值"}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/schema/catalog.work":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"catalog.work","fields":[{"key":"catalog.work.name_zh_cn","kind":"text","diff_hint":"inline","locked":false,"can_propose":true,"can_review":false,"would_automerge":false}]}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/revisions":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":100,"seq":1,"action":"created","actor_uid":7,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}]}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[]}}`))
		case r.Method == "GET" && r.URL.Path == "/v2/me/proposals":
			_, _ = w.Write([]byte(`{"object":"list","items":[],"total":0}`))
		case r.Method == "GET" && r.URL.Path == "/v2/moderation/proposals":
			_, _ = w.Write([]byte(`{"object":"list","items":[],"total":0}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":233,"message":"not found"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func editTestApp(t *testing.T, catalogURL string, user *middleware.UserInfo) *fiber.App {
	return editTestAppFull(t, catalogURL, fakeGalgame(t).URL, user, nil)
}

func editTestAppFull(t *testing.T, catalogURL, galgameURL string, user *middleware.UserInfo, notifier msgService.Notifier) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, AppKey: "nmk_test"})
	var galgameClient *client.GalgameClient
	if galgameURL != "" {
		galgameClient = client.New(galgameURL, "nm_test", "")
	}
	h := NewEditHandler(cc, galgameClient, nil, notifier, nil).
		WithOwners(fakeOwners{1000: 7})

	app := fiber.New()
	authStub := func(c fiber.Ctx) error {
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"code": 205, "message": "用户登录失效"})
		}
		c.Locals(string(middleware.UserInfoKey), user)
		c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		return c.Next()
	}
	optAuthStub := func(c fiber.Ctx) error {
		if user != nil {
			c.Locals(string(middleware.UserInfoKey), user)
			c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		}
		return c.Next()
	}
	api := app.Group("/api")
	api.Get("/galgame/:id/edit/revisions", optAuthStub, h.Revisions)
	api.Get("/galgame/:id/edit/diff", h.Diff)
	api.Get("/galgame/:id/edit/proposals", h.GameProposals)
	authed := api.Group("", authStub)
	authed.Get("/galgame/:id/edit/bootstrap", h.Bootstrap)
	authed.Post("/galgame/:id/edit/proposals", h.Submit)
	authed.Get("/galgame-edit/mine", h.Mine)
	authed.Post("/galgame-edit/proposals/:id/withdraw", h.Withdraw)
	authed.Get("/galgame-edit/queue", middleware.RequireModerator(), h.Queue)
	authed.Get("/galgame-edit/proposals/:id", h.ProposalDetail)
	authed.Post("/galgame-edit/proposals/:id/amend", h.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", h.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", h.Decline)
	authed.Post("/galgame/:id/edit/revert", h.Revert)
	return app
}

func doJSON(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

var moderatorUser = &middleware.UserInfo{ID: 42, Name: "mod", Roles: []string{"moderator"}}
var adminUser = &middleware.UserInfo{ID: 42, Name: "admin", Roles: []string{"admin"}}
var plainUser = &middleware.UserInfo{ID: 7, Name: "user", Roles: nil}

func TestEditDegradesWhenUnconfigured(t *testing.T) {
	app := editTestApp(t, "", moderatorUser)
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/galgame/1000/edit/bootstrap", ""},
		{"POST", "/api/galgame/1000/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"x"}}`},
		{"GET", "/api/galgame/1000/edit/revisions", ""},
		{"GET", "/api/galgame-edit/queue", ""},
		{"GET", "/api/galgame-edit/mine", ""},
	} {
		status, raw := doJSON(t, app, tc.method, tc.path, tc.body)
		if status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: status = %d body %s, want 503", tc.method, tc.path, status, raw)
		}
	}
}

func TestEditNoWriteAssertsAnActor(t *testing.T) {
	owner := &middleware.UserInfo{ID: 7, Name: "mod-owner", Roles: []string{"moderator"}}
	for _, user := range []*middleware.UserInfo{owner, adminUser, bystander} {
		fake := &fakeEditFace{}
		nm := fakeGalgame(t)
		app := editTestAppFull(t, fake.server(t).URL, nm.URL, user, nil)
		for _, tc := range []struct{ method, path, body string }{
			{"POST", "/api/galgame/1000/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"新标题"},"note":"typo"}`},
			{"POST", "/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"修正"}}`},
			{"POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`},
			{"POST", "/api/galgame-edit/proposals/7/decline", `{"note":"理由"}`},
			{"POST", "/api/galgame/1000/edit/revert", `{"to_seq":3}`},
		} {
			if status, raw := doJSON(t, app, tc.method, tc.path, tc.body); status != http.StatusOK {
				t.Fatalf("%s as %s: status = %d body %s", tc.path, user.Name, status, raw)
			}
		}
		for _, r := range fake.requests {
			if r.Method != "POST" {
				continue
			}
			if r.Face != "user" {
				t.Fatalf("%s as %s took the %s plane", r.Path, user.Name, r.Face)
			}
			if r.Auth != "Bearer user-jwt" {
				t.Fatalf("%s as %s used auth %q", r.Path, user.Name, r.Auth)
			}
			if _, ok := r.Body["actor"]; ok {
				t.Fatalf("%s as %s asserted an actor: %v", r.Path, user.Name, r.Body)
			}
			if _, ok := r.Body["site"]; ok {
				t.Fatalf("%s as %s asserted a site: %v", r.Path, user.Name, r.Body)
			}
		}
	}
}

func TestEditSubmitPassesThePatchThrough(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	if status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals",
		`{"patch":{"catalog.work.name_zh_cn":"新标题"},"note":"typo"}`); status != http.StatusOK {
		t.Fatalf("submit: status = %d body %s", status, raw)
	}
	if len(fake.requests) != 1 || fake.requests[0].Path != "/v2/me/proposals" {
		t.Fatalf("submit must be a single user-plane call, got %+v", fake.requests)
	}
	body := fake.requests[0].Body
	if body["entity_type"] != "catalog.work" || body["entity_id"] != "1000" {
		t.Fatalf("entity assertion wrong: %v", body)
	}
	patch := body["patch"].(map[string]any)
	if patch["catalog.work.name_zh_cn"] != "新标题" || body["note"] != "typo" {
		t.Fatalf("patch/note not passed through verbatim: %v", body)
	}
}

func TestEditRevertRidesTheUserToken(t *testing.T) {
	fake := &fakeEditFace{}
	nm := fakeGalgame(t)
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, plainUser, nil)
	status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/revert", `{"to_seq":3,"note":"回滚测试"}`)
	if status != http.StatusOK {
		t.Fatalf("revert: status = %d body %s", status, raw)
	}
	req := fake.callTo("/v2/moderation/reverts")
	if req == nil {
		t.Fatalf("revert must ride the user plane, got %+v", fake.requests)
	}
	if req.Body["revision_id"] != "101" || req.Body["reason"] != "回滚测试" {
		t.Fatalf("revert body: %v", req.Body)
	}
}

func TestEditRevertDeniedByInfra(t *testing.T) {
	fake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"field review denied"}`}
	nm := fakeGalgame(t)
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, bystander, nil)
	status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/revert", `{"to_seq":3}`)
	if status != http.StatusForbidden {
		t.Fatalf("denied revert: status = %d body %s, want 403", status, raw)
	}
	if code := envelopeCode(t, raw); code != 233 {
		t.Fatalf("code = %d, want a plain permission denial", code)
	}
}

func TestEditSubmitLocalValidation(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	for _, body := range []string{
		`{"patch":{}}`,
		`{"patch":{"galgame.game.name_zh_cn":"x"}}`,
		`{"patch":{"gid":1}}`,
	} {
		status, _ := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals", body)
		if status != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want 400", body, status)
		}
	}
	if len(fake.requests) != 0 {
		t.Fatalf("local validation must not reach the S2S face, got %d calls", len(fake.requests))
	}
}

func TestEditViewGates(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, _ := doJSON(t, app, "GET", "/api/galgame-edit/queue", "")
	if status != http.StatusForbidden {
		t.Fatalf("queue as plain user: status = %d, want 403", status)
	}
	if len(fake.requests) != 0 {
		t.Fatalf("the queue gate must not reach the catalog, got %d calls", len(fake.requests))
	}

	viewFake := &fakeEditFace{}
	nm := fakeGalgame(t)
	app = editTestAppFull(t, viewFake.server(t).URL, nm.URL, bystander, nil)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/7", "")
	if status != http.StatusForbidden {
		t.Fatalf("workbench read as bystander: status = %d body %s, want 403", status, raw)
	}
	for _, r := range viewFake.requests {
		if r.Method != "GET" || strings.Contains(r.Path, "schema") {
			t.Fatalf("a refused workbench read must stop at the proposal read, got %s %s", r.Method, r.Path)
		}
	}

	writeFake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"review denied"}`}
	app = editTestAppFull(t, writeFake.server(t).URL, nm.URL, bystander, nil)
	for _, tc := range []struct{ path, body string }{
		{"/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"y"}}`},
		{"/api/galgame-edit/proposals/7/merge", `{"note":""}`},
		{"/api/galgame-edit/proposals/7/decline", `{"note":"x"}`},
	} {
		status, raw := doJSON(t, app, "POST", tc.path, tc.body)
		if status != http.StatusForbidden {
			t.Fatalf("%s as bystander: status = %d body %s, want infra's 403", tc.path, status, raw)
		}
	}
	if writeFake.callTo("/v2/me/proposals/7/amendments") == nil {
		t.Fatalf("the amend must actually reach infra: %+v", writeFake.requests)
	}

	anon := editTestApp(t, fake.server(t).URL, nil)
	status, _ = doJSON(t, anon, "GET", "/api/galgame/1000/edit/bootstrap", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous bootstrap: status = %d, want 401", status)
	}
}

func TestEditProposalDetailCanDecideFromProjection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         string
		reviewable bool
		want       bool
	}{
		{"every key reviewable", "7", true, true},
		{"a key the caller may not review", "7", false, false},
		{"effective patch wins over the original", "9", true, true},
		{"nothing to decide", "8", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userReviewable: tc.reviewable}
			nm := fakeGalgame(t)
			app := editTestAppFull(t, fake.server(t).URL, nm.URL, moderatorUser, nil)
			status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/"+tc.id, "")
			if status != http.StatusOK {
				t.Fatalf("workbench read: status = %d body %s", status, raw)
			}
			var out struct {
				Data struct {
					CanDecide bool `json:"can_decide"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Data.CanDecide != tc.want {
				t.Fatalf("can_decide = %v, want %v (body %s)", out.Data.CanDecide, tc.want, raw)
			}
			if req := fake.callTo("/v2/catalog/schemas/work"); req == nil {
				t.Fatalf("the workbench projection must ride the user plane: %+v", fake.requests)
			}
		})
	}
}

func TestEditRevisionsCanRevertFromProjection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		reviewable bool
		token      bool
		want       bool
	}{
		{"reviews every editable field", true, true, true},
		{"cannot review an editable field", false, true, false},
		{"anonymous reader", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userReviewable: tc.reviewable}
			var user *middleware.UserInfo
			if tc.token {
				user = moderatorUser
			}
			app := editTestApp(t, fake.server(t).URL, user)
			status, raw := doJSON(t, app, "GET", "/api/galgame/1000/edit/revisions", "")
			if status != http.StatusOK {
				t.Fatalf("revisions: status = %d body %s", status, raw)
			}
			var out struct {
				Data struct {
					CanRevert bool `json:"can_revert"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Data.CanRevert != tc.want {
				t.Fatalf("can_revert = %v, want %v (body %s)", out.Data.CanRevert, tc.want, raw)
			}
		})
	}
}

func TestEditOwnerReview(t *testing.T) {
	fake := &fakeEditFace{userReviewable: true}
	nm := fakeGalgame(t)
	sink := &fakeNotifier{}
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, plainUser, sink)

	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/7", "")
	if status != http.StatusOK {
		t.Fatalf("owner workbench read: status = %d body %s", status, raw)
	}
	var detail struct {
		Data struct {
			CanDecide bool `json:"can_decide"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &detail)
	if !detail.Data.CanDecide {
		t.Fatalf("an owner whose projection reviews the patch must get can_decide: %s", raw)
	}
	if fake.callTo("/api/v1/catalog/edit/schema/catalog.work") != nil {
		t.Fatalf("no projection may still be asked for on the S2S face: %+v", fake.requests)
	}

	status, raw = doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`)
	if status != http.StatusOK {
		t.Fatalf("owner merge: status = %d body %s", status, raw)
	}
	req := fake.callTo("/v2/moderation/proposals/7/decisions")
	if req == nil || req.Auth != "Bearer user-jwt" {
		t.Fatalf("the owner's merge must ride their own token: %+v", fake.requests)
	}
	if _, ok := req.Body["actor"]; ok {
		t.Fatalf("the merge must assert no actor: %v", req.Body)
	}

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.specs) != 1 {
		t.Fatalf("want exactly one merged notice, got %d", len(sink.specs))
	}
	n := sink.specs[0]
	if n.Kind != msgService.NotifyMerged || n.ReceiverID != 9 || n.SenderID != 7 || n.WorkID != 1000 {
		t.Fatalf("merged notice shape: %+v", n)
	}
	if !strings.Contains(n.Content, "测试游戏") || !strings.Contains(n.Content, "修正") {
		t.Fatalf("merged notice content must name the game + mark the correction: %q", n.Content)
	}
}

func TestEditDeclineNotification(t *testing.T) {
	fake := &fakeEditFace{}
	nm := fakeGalgame(t)
	sink := &fakeNotifier{}

	deniedFake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"review denied"}`}
	deniedSink := &fakeNotifier{}
	deniedApp := editTestAppFull(t, deniedFake.server(t).URL, nm.URL, moderatorUser, deniedSink)
	if status, raw := doJSON(t, deniedApp, "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"x"}`); status != http.StatusForbidden {
		t.Fatalf("denied decline: status = %d body %s, want 403", status, raw)
	}
	deniedSink.mu.Lock()
	if len(deniedSink.specs) != 0 {
		t.Fatalf("a refused decline must notify nobody, got %+v", deniedSink.specs)
	}
	deniedSink.mu.Unlock()

	app := editTestAppFull(t, fake.server(t).URL, nm.URL, adminUser, sink)
	status, raw := doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"资料来源不可靠，请补充出处"}`)
	if status != http.StatusOK {
		t.Fatalf("decline: status = %d body %s", status, raw)
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.specs) != 1 {
		t.Fatalf("want exactly one declined notice, got %d", len(sink.specs))
	}
	n := sink.specs[0]
	if n.Kind != msgService.NotifyDeclined || n.ReceiverID != 9 || n.SenderID != 42 || n.WorkID != 1000 {
		t.Fatalf("declined notice shape: %+v", n)
	}
	if !strings.Contains(n.Content, "资料来源不可靠，请补充出处") {
		t.Fatalf("declined notice must carry the reason in full: %q", n.Content)
	}
}

func TestEditGameProposals(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, nil)
	status, raw := doJSON(t, app, "GET", "/api/galgame/1000/edit/proposals", "")
	if status != http.StatusOK {
		t.Fatalf("game proposals: status = %d body %s", status, raw)
	}
	if len(fake.requests) != 1 || !strings.Contains(fake.requests[0].Query, "entity_id=1000") {
		t.Fatalf("list must filter to the game: %+v", fake.requests)
	}
	if !strings.Contains(fake.requests[0].Query, "object=work") {
		t.Fatalf("list must name the family: %q", fake.requests[0].Query)
	}
	if !strings.Contains(fake.requests[0].Query, "state=open") {
		t.Fatalf("list must default to open proposals: %q", fake.requests[0].Query)
	}
}

func TestEditTenantPin(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/55", "")
	if status != http.StatusNotFound {
		t.Fatalf("foreign-tenant proposal: status = %d body %s, want 404", status, raw)
	}
	status, _ = doJSON(t, app, "POST", "/api/galgame-edit/proposals/55/merge", `{"note":""}`)
	if status != http.StatusNotFound {
		t.Fatalf("foreign-tenant merge: status = %d, want 404", status)
	}
}

func TestEditBootstrapShape(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame/1000/edit/bootstrap", "")
	if status != http.StatusOK {
		t.Fatalf("bootstrap: status = %d body %s", status, raw)
	}
	var out struct {
		Data struct {
			WorkID    int64          `json:"gid"`
			Values    map[string]any `json:"values"`
			Fields    []any          `json:"fields"`
			CanReview bool           `json:"can_review"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Data.WorkID != 1000 || out.Data.Values["catalog.work.name_zh_cn"] != "现值" {
		t.Fatalf("bootstrap shape wrong: %s", raw)
	}
	if len(out.Data.Fields) != 3 || out.Data.CanReview {
		t.Fatalf("projection shape wrong: %s", raw)
	}
}
