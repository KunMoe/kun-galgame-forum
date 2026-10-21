package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captured struct {
	method string
	path   string
	query  string
	auth   string
	body   map[string]any
}

func recordingServer(t *testing.T, status int, reply string) (*httptest.Server, *captured) {
	t.Helper()
	var got captured
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.method, got.path, got.query = r.Method, r.URL.Path, r.URL.RawQuery
		got.auth = r.Header.Get("Authorization")
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		if status != 0 {
			w.WriteHeader(status)
		}
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func userClient(baseURL string) *Client {
	return New(Config{BaseURL: baseURL})
}

func TestCreateMyProposal_TravelsAsTheUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"object":"proposal","id":"7","state":"open","target_object":"work",`+
		`"entity_type":"catalog.work","entity_id":"1000","site":"kungal","proposer_uid":"9","note":"typo",`+
		`"base_revision_seq":3,"decided_by_uid":null,"decided_at":null,`+
		`"created_at":"2026-09-20T00:00:00Z","updated_at":"2026-09-20T00:00:00Z"}`)

	prop, _, err := userClient(srv.URL).CreateMyProposal(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.name_zh_cn": "新标题"}, Note: "typo"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v2/me/proposals" {
		t.Fatalf("create hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if strings.HasPrefix(got.auth, "Basic ") {
		t.Fatal("the user plane must never attach the client credential")
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", got.body)
	}
	if _, ok := got.body["site"]; ok {
		t.Fatalf("the user plane asserts no site: %v", got.body)
	}
	if got.body["entity_type"] != "catalog.work" || got.body["note"] != "typo" {
		t.Fatalf("create body wrong: %v", got.body)
	}
	patch, _ := got.body["patch"].(map[string]any)
	if patch["catalog.work.name_zh_cn"] != "新标题" {
		t.Fatalf("patch not passed through verbatim: %v", got.body)
	}
	if prop.ID != 7 || prop.Status != "open" {
		t.Fatalf("proposal decoded wrong: %+v", prop)
	}
}

func TestUserEditErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"invalid token"}`, ErrUnauthorized},
		{"missing scope", http.StatusForbidden, `{"code":233,"message":"missing required scope: catalog:edit"}`, ErrInsufficientScope},
		{"unknown entity", http.StatusNotFound, `{"code":233,"message":"entity not found"}`, ErrNotFound},
		{"upstream down", http.StatusBadGateway, `nonsense`, ErrUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := recordingServer(t, tc.status, tc.body)
			_, _, err := userClient(srv.URL).CreateMyProposal(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}}, "")
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUserEditKeepsActionableReplies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"policy denial", http.StatusForbidden, `{"code":233,"message":"field is locked by policy"}`},
		{"validation", http.StatusUnprocessableEntity, `{"code":233,"message":"name_zh_cn 不能为空"}`},
		{"rebase conflict", http.StatusConflict, `{"code":233,"message":"proposal is closed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := recordingServer(t, tc.status, tc.body)
			_, _, err := userClient(srv.URL).CreateMyProposal(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}}, "")
			if errors.Is(err, ErrInsufficientScope) {
				t.Fatalf("%s must not read as a scope problem", tc.name)
			}
			var apiErr *UserAPIError
			if !errors.As(err, &apiErr) || apiErr.Status != tc.status {
				t.Fatalf("err = %v, want a *UserAPIError %d", err, tc.status)
			}
			if !strings.Contains(apiErr.Message, "policy") && !strings.Contains(apiErr.Message, "不能为空") &&
				!strings.Contains(apiErr.Message, "closed") {
				t.Fatalf("upstream wording lost: %q", apiErr.Message)
			}
		})
	}
}

func TestUserEditWithoutTokenNeverCalls(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := userClient(srv.URL)
	if _, _, err := c.CreateMyProposal(context.Background(), "",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000}, ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if _, err := c.EditSnapshotUser(context.Background(), "", "catalog.work", 1000); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if called {
		t.Fatal("a tokenless user-plane call must not reach the catalog")
	}
}

func TestUserEditUnconfigured(t *testing.T) {
	c := New(Config{})
	if _, _, err := c.CreateMyProposal(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1}, ""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if _, err := c.EditSnapshotUser(context.Background(), "user-jwt", "catalog.work", 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestUserEditAdjudicationErrorMapping(t *testing.T) {
	lanes := map[string]func(c *Client) error{
		"merge": func(c *Client) error {
			_, err := c.DecideProposal(context.Background(), "user-jwt", 7, "merge", "", "")
			return err
		},
		"decline": func(c *Client) error {
			_, err := c.DecideProposal(context.Background(), "user-jwt", 7, "decline", "no", "")
			return err
		},
	}
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"invalid token"}`, ErrUnauthorized},
		{"missing scope", http.StatusForbidden, `{"code":233,"message":"missing required scope: catalog:edit"}`, ErrInsufficientScope},
		{"unknown proposal", http.StatusNotFound, `{"code":233,"message":"not found"}`, ErrNotFound},
	}
	for lane, call := range lanes {
		for _, tc := range cases {
			t.Run(lane+"/"+tc.name, func(t *testing.T) {
				srv, _ := recordingServer(t, tc.status, tc.body)
				if err := call(userClient(srv.URL)); !errors.Is(err, tc.want) {
					t.Fatalf("err = %v, want %v", err, tc.want)
				}
			})
		}
		t.Run(lane+"/plain denial", func(t *testing.T) {
			srv, _ := recordingServer(t, http.StatusForbidden, `{"code":233,"message":"field review denied"}`)
			err := call(userClient(srv.URL))
			if errors.Is(err, ErrInsufficientScope) {
				t.Fatalf("a policy denial must not read as a scope problem: %v", err)
			}
			var apiErr *UserAPIError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden ||
				!strings.Contains(apiErr.Message, "review denied") {
				t.Fatalf("err = %v, want the upstream's own 403", err)
			}
		})
		t.Run(lane+"/closed", func(t *testing.T) {
			srv, _ := recordingServer(t, http.StatusConflict, `{"code":233,"message":"proposal is closed"}`)
			var apiErr *UserAPIError
			if err := call(userClient(srv.URL)); !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
				t.Fatalf("err = %v, want a *UserAPIError 409", err)
			}
		})
	}
}

func TestUserEditAdjudicationWithoutTokenNeverCalls(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := userClient(srv.URL)
	_, merge := c.DecideProposal(context.Background(), "", 7, "merge", "", "")
	_, decline := c.DecideProposal(context.Background(), "", 7, "decline", "x", "")
	for name, err := range map[string]error{"merge": merge, "decline": decline} {
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("%s: err = %v, want ErrUnauthorized", name, err)
		}
	}
	if called {
		t.Fatal("a tokenless user-plane call must not reach the catalog")
	}
}

func TestUserEditAdjudicationUnconfigured(t *testing.T) {
	c := New(Config{})
	if _, err := c.DecideProposal(context.Background(), "user-jwt", 7, "merge", "", ""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// The engine names the field it refused in the problem body's errors[]. Keeping
// only the joined message turns "第 1 行 标题：必填" into a toast the reader has
// to map back onto a form by hand.
func TestUserEditKeepsTheProblemsFieldErrors(t *testing.T) {
	srv, _ := recordingServer(t, http.StatusUnprocessableEntity,
		`{"code":"UNKNOWN_VALUE","title":"Unprocessable Entity",`+
			`"detail":"editing: field \"catalog.work.links\": element 0: must be an http:// or https:// URL",`+
			`"errors":[{"pointer":"/patch/catalog.work.links","reason":"UNKNOWN_VALUE",`+
			`"detail":"element 0: must be an http:// or https:// URL"}]}`)

	_, _, err := userClient(srv.URL).CreateMyProposal(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.links": []string{"ftp://x"}}}, "")

	var apiErr *UserAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want a *UserAPIError", err)
	}
	if apiErr.ProblemCode != "UNKNOWN_VALUE" {
		t.Fatalf("problem code = %q", apiErr.ProblemCode)
	}
	if len(apiErr.FieldErrors) != 1 ||
		apiErr.FieldErrors[0].Pointer != "/patch/catalog.work.links" {
		t.Fatalf("field errors lost: %+v", apiErr.FieldErrors)
	}
}
