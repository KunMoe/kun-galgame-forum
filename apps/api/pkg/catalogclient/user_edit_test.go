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

func TestCreateEditProposalUser_TravelsAsTheUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"merged":false,`+
		`"proposal":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal",`+
		`"status":"open","proposer_uid":9,"patch":{"catalog.work.name_zh_cn":"新标题"}}}}`)

	res, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.name_zh_cn": "新标题"}, Note: "typo"})
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
	if res.Merged || res.Proposal.ID != 7 || res.Proposal.Status != "open" {
		t.Fatalf("pending result decoded wrong: %+v", res)
	}
}

func TestCreateEditProposalUser_DecodesAutomerge(t *testing.T) {
	srv, _ := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"merged":true,`+
		`"proposal":{"id":8,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"merged","patch":{}},`+
		`"revision":{"id":100,"seq":4,"action":"merged","actor_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}}`)

	res, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "admin-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Merged || res.Revision == nil || res.Revision.Seq != 4 {
		t.Fatalf("automerge result decoded wrong: %+v", res)
	}
}

func TestWithdrawEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":7,`+
		`"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"withdrawn","patch":{}}}`)

	prop, err := userClient(srv.URL).WithdrawEditProposalUser(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPatch || got.path != "/v2/me/proposals/7" {
		t.Fatalf("withdraw hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("withdraw must assert no actor: %v", got.body)
	}
	if prop.Status != "withdrawn" || prop.ID != 7 {
		t.Fatalf("withdraw result decoded wrong: %+v", prop)
	}
}

func TestGetEditSchemaUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"entity_type":"catalog.work",`+
		`"fields":[{"key":"catalog.work.name_zh_cn","kind":"text","diff_hint":"inline",`+
		`"locked":false,"can_propose":true,"can_review":false,"would_automerge":false}]}}`)

	schema, err := userClient(srv.URL).GetEditSchemaUser(context.Background(), "user-jwt", "catalog.work", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/v2/catalog/schemas/work" {
		t.Fatalf("schema hit %s %s", got.method, got.path)
	}
	if got.query != "" && got.query != "entity_id=1000" {
		t.Fatalf("schema query = %q", got.query)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if len(schema.Fields) != 1 || schema.Fields[0].Key != "catalog.work.name_zh_cn" ||
		!schema.Fields[0].CanPropose {
		t.Fatalf("projection decoded wrong: %+v", schema)
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
			_, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
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
			_, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
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
	if _, err := c.WithdrawEditProposalUser(context.Background(), "", 7); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if _, err := c.GetEditSchemaUser(context.Background(), "", "catalog.work", 1000); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if called {
		t.Fatal("a tokenless user-plane call must not reach the catalog")
	}
}

func TestUserEditUnconfigured(t *testing.T) {
	c := New(Config{})
	if _, err := c.CreateEditProposalUser(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if _, err := c.GetEditSchemaUser(context.Background(), "user-jwt", "catalog.work", 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestAmendEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":1,"seq":1,`+
		`"amender_uid":7,"set":{"catalog.work.name_zh_cn":"修正"},"note":"typo fixed"}}`)

	amendment, err := userClient(srv.URL).AmendEditProposalUser(context.Background(), "user-jwt", 7,
		map[string]any{"catalog.work.name_zh_cn": "修正"}, []string{"catalog.work.vndb_id"}, "typo fixed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v2/me/proposals/7/amendments" {
		t.Fatalf("amend hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" || strings.HasPrefix(got.auth, "Basic ") {
		t.Fatalf("auth = %q, want the user's bearer only", got.auth)
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", got.body)
	}
	set, _ := got.body["set"].(map[string]any)
	unset, _ := got.body["unset"].([]any)
	if set["catalog.work.name_zh_cn"] != "修正" || len(unset) != 1 || unset[0] != "catalog.work.vndb_id" {
		t.Fatalf("amend delta wrong: %v", got.body)
	}
	if got.body["note"] != "typo fixed" {
		t.Fatalf("note lost: %v", got.body)
	}
	if amendment.Seq != 1 || amendment.AmenderUID != 7 {
		t.Fatalf("amendment decoded wrong: %+v", amendment)
	}
}

func TestAmendEditProposalUserOmitsEmptyParts(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":1,"seq":1,"amender_uid":7}}`)
	if _, err := userClient(srv.URL).AmendEditProposalUser(context.Background(), "user-jwt", 7,
		map[string]any{"catalog.work.name_zh_cn": "x"}, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got.body["unset"]; ok {
		t.Fatalf("an absent unset must not travel: %v", got.body)
	}
	if _, ok := got.body["note"]; ok {
		t.Fatalf("an empty note must not travel: %v", got.body)
	}
}

func TestMergeEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":100,"seq":4,`+
		`"action":"merged","actor_uid":9,"amender_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}`)

	rev, err := userClient(srv.URL).MergeEditProposalUser(context.Background(), "user-jwt", 7, "looks right")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v2/moderation/proposals/7/decisions" {
		t.Fatalf("merge hit %s %s", got.method, got.path)
	}
	if got.body["decision"] != "merge" {
		t.Fatalf("merge body wrong: %v", got.body)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", got.body)
	}
	if got.body["note"] != "looks right" {
		t.Fatalf("note lost: %v", got.body)
	}
	if rev.Seq != 4 || rev.AmenderUID == nil || *rev.AmenderUID != 42 {
		t.Fatalf("revision decoded wrong: %+v", rev)
	}
}

func TestDeclineEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":7,`+
		`"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"declined","proposer_uid":9,"patch":{}}}`)

	prop, err := userClient(srv.URL).DeclineEditProposalUser(context.Background(), "user-jwt", 7, "来源不可靠")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v2/moderation/proposals/7/decisions" {
		t.Fatalf("decline hit %s %s", got.method, got.path)
	}
	if got.body["decision"] != "decline" {
		t.Fatalf("decline body wrong: %v", got.body)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", got.body)
	}
	if got.body["note"] != "来源不可靠" {
		t.Fatalf("decline reason lost: %v", got.body)
	}
	if prop.Status != "declined" {
		t.Fatalf("proposal decoded wrong: %+v", prop)
	}
}

func TestRevertEditEntityUser(t *testing.T) {
	srv, got := recordingServer(t, http.StatusCreated, `{"object":"proposal","id":"8","entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"merged","patch":{}}`)

	res, err := userClient(srv.URL).RevertEditEntityUser(context.Background(), "user-jwt", 101, "回滚")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v2/moderation/reverts" {
		t.Fatalf("revert hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if got.body["revision_id"] != "101" || got.body["reason"] != "回滚" {
		t.Fatalf("revert body wrong: %v", got.body)
	}
	if res.Proposal.ID != 8 || res.Proposal.Status != "merged" {
		t.Fatalf("revert result decoded wrong: %+v", res)
	}
}

func TestGetEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"object":"proposal","id":"7",`+
		`"entity_type":"catalog.work","entity_id":"1000","site":"kungal","state":"open","proposer_uid":"9",`+
		`"patch":{"catalog.work.name_zh_cn":"新标题","catalog.work.vndb_id":"v1"},`+
		`"effective_patch":{"catalog.work.name_zh_cn":"修正"},`+
		`"amendments":[{"id":"1","seq":1,"amender_uid":"42","set":{"catalog.work.name_zh_cn":"修正"},`+
		`"unset":["catalog.work.vndb_id"],"note":"来源存疑"}]}`)

	prop, err := userClient(srv.URL).GetEditProposalUser(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/v2/moderation/proposals/7" {
		t.Fatalf("detail hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if !strings.Contains(got.query, "include=") {
		t.Fatalf("review detail must ask for patch+amendments: %q", got.query)
	}
	if len(prop.Amendments) != 1 || prop.Amendments[0].AmenderUID != 42 {
		t.Fatalf("amendments decoded wrong: %+v", prop)
	}
	if prop.EffectivePatch["catalog.work.name_zh_cn"] != "修正" || len(prop.EffectivePatch) != 1 {
		t.Fatalf("effective patch decoded wrong: %+v", prop.EffectivePatch)
	}
}

func TestUserEditAdjudicationErrorMapping(t *testing.T) {
	lanes := map[string]func(c *Client) error{
		"amend": func(c *Client) error {
			_, err := c.AmendEditProposalUser(context.Background(), "user-jwt", 7,
				map[string]any{"catalog.work.name_zh_cn": "x"}, nil, "")
			return err
		},
		"merge": func(c *Client) error {
			_, err := c.MergeEditProposalUser(context.Background(), "user-jwt", 7, "")
			return err
		},
		"decline": func(c *Client) error {
			_, err := c.DeclineEditProposalUser(context.Background(), "user-jwt", 7, "no")
			return err
		},
		"revert": func(c *Client) error {
			_, err := c.RevertEditEntityUser(context.Background(), "user-jwt", 101, "")
			return err
		},
		"detail": func(c *Client) error {
			_, err := c.GetEditProposalUser(context.Background(), "user-jwt", 7)
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
	_, amend := c.AmendEditProposalUser(context.Background(), "", 7, map[string]any{"k": "v"}, nil, "")
	_, merge := c.MergeEditProposalUser(context.Background(), "", 7, "")
	_, decline := c.DeclineEditProposalUser(context.Background(), "", 7, "x")
	_, revert := c.RevertEditEntityUser(context.Background(), "", 2, "")
	_, detail := c.GetEditProposalUser(context.Background(), "", 7)
	for name, err := range map[string]error{
		"amend": amend, "merge": merge, "decline": decline, "revert": revert, "detail": detail,
	} {
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
	if _, err := c.MergeEditProposalUser(context.Background(), "user-jwt", 7, ""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if _, err := c.RevertEditEntityUser(context.Background(), "user-jwt", 2, ""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// The schema face carries the value half — how big a list may be, which
// vocabulary an enum draws from, how that vocabulary is encoded on the wire.
// Dropping it is invisible from the API and fatal in the browser: an enum with
// no options renders read-only, which is how the edit page lost whole tabs.
func TestGetEditSchemaUserKeepsTheValueFace(t *testing.T) {
	srv, _ := recordingServer(t, 0, `{"object":"object_schema","entity_type":"catalog.work","fields":[`+
		`{"key":"catalog.work.olang","field_type":"enum","diff_hint":"inline",`+
		`"vocabulary":"olang","encoding":"token","base":0,"nullable":false},`+
		`{"key":"catalog.work.titles","field_type":"list","diff_hint":"items",`+
		`"max_elements":100,"max_suppressed":200,"element":{"type":"object","members":[`+
		`{"key":"title","type":"text"},{"key":"kind","type":"int","vocabulary":"title_kind"}]}}]}`)

	schema, err := userClient(srv.URL).GetEditSchemaUser(context.Background(), "user-jwt", "catalog.work", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(schema.Fields))
	}
	olang := schema.Fields[0]
	if olang.Vocabulary != "olang" || olang.Encoding != "token" {
		t.Fatalf("olang lost its vocabulary: %+v", olang)
	}
	titles := schema.Fields[1]
	if titles.MaxElements != 100 || titles.MaxSuppressed != 200 {
		t.Fatalf("titles lost its caps: %+v", titles)
	}
	if titles.Element == nil || len(titles.Element.Members) != 2 ||
		titles.Element.Members[1].Vocabulary != "title_kind" {
		t.Fatalf("titles lost its element shape: %+v", titles.Element)
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

	_, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.links": []string{"ftp://x"}}})

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
