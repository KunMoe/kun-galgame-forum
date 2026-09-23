package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
)

func userPlaneApp(t *testing.T, catalogURL string, user *middleware.UserInfo, token string) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, AppKey: "nmk_test"})
	h := NewEditHandler(cc, client.New(fakeGalgame(t).URL, "nm_test", ""), nil, nil, nil).
		WithOwners(fakeOwners{1000: 7})

	app := fiber.New()
	api := app.Group("/api")
	authed := api.Group("", func(c fiber.Ctx) error {
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"code": 205, "message": "用户登录失效"})
		}
		c.Locals(string(middleware.UserInfoKey), user)
		c.Locals(string(middleware.OAuthAccessTokenKey), token)
		return c.Next()
	})
	authed.Get("/galgame/:id/edit/bootstrap", h.Bootstrap)
	authed.Post("/galgame/:id/edit/proposals", h.Submit)
	authed.Post("/galgame-edit/proposals/:id/withdraw", h.Withdraw)
	authed.Post("/galgame-edit/proposals/:id/amend", h.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", h.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", h.Decline)
	authed.Post("/galgame/:id/edit/revert", h.Revert)
	return app
}

var bystander = &middleware.UserInfo{ID: 8, Name: "bystander", Roles: nil}

func (f *fakeEditFace) callTo(path string) *recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.requests {
		if f.requests[i].Path == path {
			return &f.requests[i]
		}
	}
	return nil
}

func envelopeCode(t *testing.T, raw []byte) int {
	t.Helper()
	var env struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("bad envelope %s", raw)
	}
	return env.Code
}

func TestEditSubmitRidesTheUserToken(t *testing.T) {
	for _, user := range []*middleware.UserInfo{bystander, plainUser, adminUser} {
		fake := &fakeEditFace{}
		app := userPlaneApp(t, fake.server(t).URL, user, "user-jwt")
		status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals",
			`{"patch":{"catalog.work.name_zh_cn":"新标题"}}`)
		if status != http.StatusOK {
			t.Fatalf("submit as %s: status = %d body %s", user.Name, status, raw)
		}
		if len(fake.requests) != 1 || fake.requests[0].Face != "user" {
			t.Fatalf("submit as %s must be a single user-plane call, got %+v", user.Name, fake.requests)
		}
		if fake.requests[0].Auth != "Bearer user-jwt" {
			t.Fatalf("submit as %s used auth %q", user.Name, fake.requests[0].Auth)
		}
		if _, ok := fake.requests[0].Body["actor"]; ok {
			t.Fatalf("submit as %s asserted an actor: %v", user.Name, fake.requests[0].Body)
		}
	}
}

func TestEditSubmitPayloadOnTheUserPlane(t *testing.T) {
	fake := &fakeEditFace{}
	app := userPlaneApp(t, fake.server(t).URL, bystander, "user-jwt")

	status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals",
		`{"patch":{"catalog.work.name_zh_cn":"新标题"},"note":"typo"}`)
	if status != http.StatusOK {
		t.Fatalf("submit: status = %d body %s", status, raw)
	}
	req := fake.callTo("/v2/me/proposals")
	if req == nil || req.Face != "user" {
		t.Fatalf("a non-owner submit must ride the user plane, got %+v", fake.requests)
	}
	if req.Auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the session bearer", req.Auth)
	}
	if _, ok := req.Body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", req.Body)
	}
	if _, ok := req.Body["site"]; ok {
		t.Fatalf("the user plane asserts no site: %v", req.Body)
	}
	if req.Body["entity_id"] != "1000" {
		t.Fatalf("entity id must be the TRANSLATED registry work id: %v", req.Body)
	}
	patch, _ := req.Body["patch"].(map[string]any)
	if patch["catalog.work.name_zh_cn"] != "新标题" {
		t.Fatalf("patch not passed through verbatim: %v", req.Body)
	}
	var out struct {
		Data struct {
			Merged bool `json:"merged"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &out)
	if out.Data.Merged {
		t.Fatalf("a contributor's proposal must not report merged: %s", raw)
	}
}

func TestEditWithdrawAlwaysRidesTheUserToken(t *testing.T) {
	for _, user := range []*middleware.UserInfo{bystander, plainUser, adminUser} {
		fake := &fakeEditFace{}
		app := userPlaneApp(t, fake.server(t).URL, user, "user-jwt")
		status, raw := doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/withdraw", "")
		if status != http.StatusOK {
			t.Fatalf("withdraw as %s: status = %d body %s", user.Name, status, raw)
		}
		if len(fake.requests) != 1 {
			t.Fatalf("withdraw as %s must be exactly one call, got %+v", user.Name, fake.requests)
		}
		req := fake.requests[0]
		if req.Face != "user" || req.Path != "/v2/me/proposals/7" || req.Method != "PATCH" {
			t.Fatalf("withdraw as %s hit %s %s (%s)", user.Name, req.Method, req.Path, req.Face)
		}
		if req.Auth != "Bearer user-jwt" {
			t.Fatalf("withdraw as %s used auth %q", user.Name, req.Auth)
		}
		if _, ok := req.Body["actor"]; ok {
			t.Fatalf("withdraw must assert no actor: %v", req.Body)
		}
	}
}

func TestEditBootstrapProjectionFollowsThePlane(t *testing.T) {
	for _, user := range []*middleware.UserInfo{bystander, plainUser} {
		fake := &fakeEditFace{}
		app := userPlaneApp(t, fake.server(t).URL, user, "user-jwt")
		if status, raw := doJSON(t, app, "GET", "/api/galgame/1000/edit/bootstrap", ""); status != http.StatusOK {
			t.Fatalf("bootstrap as %s: status = %d body %s", user.Name, status, raw)
		}
		req := fake.callTo("/v2/catalog/schemas/work")
		if req == nil {
			t.Fatalf("the projection for %s must ride the user plane, got %+v", user.Name, fake.requests)
		}
		if req.Query != "" {
			t.Fatalf("the user plane takes no actor parameters: %q", req.Query)
		}
		if fake.callTo("/api/v1/catalog/edit/schema/catalog.work") != nil {
			t.Fatalf("no projection may still be asserted on S2S for %s: %+v", user.Name, fake.requests)
		}
		if req := fake.callTo("/v2/moderation/snapshots/work/1000"); req == nil || req.Face != "user" {
			t.Fatalf("the value snapshot must ride the user plane, got %+v", fake.requests)
		}
		if fake.callTo("/api/v1/catalog/edit/snapshot") != nil {
			t.Fatalf("no snapshot may still be read S2S for %s: %+v", user.Name, fake.requests)
		}
	}
}

func TestEditUserPlaneStaleGrantAsksForReauth(t *testing.T) {
	for _, tc := range []struct{ name, method, path, body string }{
		{"submit", "POST", "/api/galgame/1000/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"x"}}`},
		{"withdraw", "POST", "/api/galgame-edit/proposals/7/withdraw", ""},
		{"bootstrap", "GET", "/api/galgame/1000/edit/bootstrap", ""},
		{"amend", "POST", "/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"x"}}`},
		{"merge", "POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`},
		{"decline", "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"理由"}`},
		{"revert", "POST", "/api/galgame/1000/edit/revert", `{"to_seq":3}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userStatus: http.StatusForbidden,
				userBody: `{"code":233,"message":"missing required scope: catalog:edit"}`}
			app := userPlaneApp(t, fake.server(t).URL, bystander, "user-jwt")
			status, raw := doJSON(t, app, tc.method, tc.path, tc.body)
			if status != http.StatusForbidden {
				t.Fatalf("status = %d body %s, want 403", status, raw)
			}
			if code := envelopeCode(t, raw); code != 235 {
				t.Fatalf("code = %d, want 235 (re-login required); body %s", code, raw)
			}
		})
	}
}

func TestEditUserPlaneErrorPassThrough(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantCode   int
	}{
		{"policy denial", http.StatusForbidden, `{"code":233,"message":"field locked"}`, 403, 233},
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"bad token"}`, 401, 205},
		{"unknown entity", http.StatusNotFound, `{"code":233,"message":"no such entity"}`, 404, 233},
		{"validation", http.StatusUnprocessableEntity, `{"code":233,"message":"名称不能为空"}`, 400, 233},
		{"rebase conflict", http.StatusConflict, `{"code":233,"message":"closed"}`, 409, 233},
		{"upstream broken", http.StatusBadGateway, `nonsense`, 503, 233},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userStatus: tc.status, userBody: tc.body}
			app := userPlaneApp(t, fake.server(t).URL, bystander, "user-jwt")
			status, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals",
				`{"patch":{"catalog.work.name_zh_cn":"x"}}`)
			if status != tc.wantStatus {
				t.Fatalf("status = %d body %s, want %d", status, raw, tc.wantStatus)
			}
			if code := envelopeCode(t, raw); code != tc.wantCode {
				t.Fatalf("code = %d, want %d (body %s)", code, tc.wantCode, raw)
			}
		})
	}
	fake := &fakeEditFace{userStatus: http.StatusUnprocessableEntity,
		userBody: `{"code":233,"message":"名称不能为空"}`}
	app := userPlaneApp(t, fake.server(t).URL, bystander, "user-jwt")
	_, raw := doJSON(t, app, "POST", "/api/galgame/1000/edit/proposals",
		`{"patch":{"catalog.work.name_zh_cn":"x"}}`)
	if !strings.Contains(string(raw), "名称不能为空") {
		t.Fatalf("upstream validation wording lost: %s", raw)
	}
}

func TestEditUserPlaneWithoutTokenNeverCalls(t *testing.T) {
	for _, tc := range []struct{ name, method, path, body string }{
		{"submit", "POST", "/api/galgame/1000/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"x"}}`},
		{"withdraw", "POST", "/api/galgame-edit/proposals/7/withdraw", ""},
		{"bootstrap", "GET", "/api/galgame/1000/edit/bootstrap", ""},
		{"amend", "POST", "/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"x"}}`},
		{"merge", "POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`},
		{"decline", "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"理由"}`},
		{"revert", "POST", "/api/galgame/1000/edit/revert", `{"to_seq":3}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{}
			app := userPlaneApp(t, fake.server(t).URL, bystander, "")
			status, raw := doJSON(t, app, tc.method, tc.path, tc.body)
			if status != http.StatusUnauthorized {
				t.Fatalf("status = %d body %s, want 401", status, raw)
			}
			if code := envelopeCode(t, raw); code != 205 {
				t.Fatalf("code = %d, want 205; body %s", code, raw)
			}
			for _, r := range fake.requests {
				if r.Face == "user" {
					t.Fatalf("a tokenless session must not reach the user plane: %+v", r)
				}
			}
		})
	}
}
