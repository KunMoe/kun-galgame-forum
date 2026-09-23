package problem

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

type richBody struct {
	Title string   `json:"title" minLength:"2" maxLength:"8"`
	Age   int      `json:"age" minimum:"1" maximum:"99"`
	Score float64  `json:"score" exclusiveMinimum:"0" exclusiveMaximum:"1"`
	Count int      `json:"count" multipleOf:"2"`
	Tags  []string `json:"tags" minItems:"2" maxItems:"3" uniqueItems:"true"`
	Kind  string   `json:"kind" enum:"alpha,beta"`
	State string   `json:"state" const:"open"`
	Email string   `json:"email" format:"email"`
	Slug  string   `json:"slug" pattern:"^[a-z]+$"`
	Flag  bool     `json:"flag"`
	When  string   `json:"when" format:"date"`
}

type nestedBody struct {
	Items []struct {
		Name string `json:"name" minLength:"2"`
	} `json:"items"`
}

type anyOfDoc struct{}

func (anyOfDoc) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		AnyOf: []*huma.Schema{{Type: huma.TypeString}, {Type: huma.TypeInteger}},
	}
}

type oneOfDoc struct{}

func (oneOfDoc) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		OneOf: []*huma.Schema{{Type: huma.TypeString}, {Type: huma.TypeInteger}},
	}
}

type oneOfOverlapDoc struct{}

func (oneOfOverlapDoc) Schema(huma.Registry) *huma.Schema {
	zero, ten := 0, 10
	return &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeString, MinLength: &zero},
			{Type: huma.TypeString, MaxLength: &ten},
		},
	}
}

func registerBody(api huma.API) {
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/body"},
		func(context.Context, *struct{ Body richBody }) (*struct{}, error) {
			return nil, nil
		})
}

func registerNested(api huma.API) {
	huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/nested"},
		func(context.Context, *struct{ Body nestedBody }) (*struct{}, error) {
			return nil, nil
		})
}

func registerQuery(api huma.API) {
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/query"},
		func(context.Context, *struct {
			Q      int    `query:"q" required:"true"`
			Sort   string `query:"sort" enum:"new,old"`
			Color  string `query:"color" enum:"red,blue"`
			Name   string `query:"name" minLength:"2" maxLength:"8"`
			Limit  int    `query:"limit" minimum:"1" maximum:"100"`
			Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512"`
		}) (*struct{}, error) {
			return nil, nil
		})
}

func registerHeader(api huma.API) {
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/hdr"},
		func(context.Context, *struct {
			Need string `header:"X-Need" required:"true"`
		}) (*struct{}, error) {
			return nil, nil
		})
}

func validBody() string {
	return `{
		"title":"abcd","age":10,"score":0.5,"count":4,
		"tags":["a","b"],"kind":"alpha","state":"open",
		"email":"a@b.co","slug":"hello","flag":true,"when":"2026-01-02"
	}`
}

func patchBody(t *testing.T, override map[string]any) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(validBody()), &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range override {
		if v == nil {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type captured struct {
	ctx    huma.Context
	status int
	msg    string
	errs   []error
	called bool
}

var humaHookMu sync.Mutex

func capture(t *testing.T, register func(huma.API), method, path string, args ...any) captured {
	t.Helper()
	humaHookMu.Lock()
	defer humaHookMu.Unlock()

	orig := huma.NewErrorWithContext
	defer func() { huma.NewErrorWithContext = orig }()

	var cap captured
	huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		cap.ctx = ctx
		cap.status = status
		cap.msg = msg
		cap.errs = append([]error(nil), errs...)
		cap.called = true
		return orig(ctx, status, msg, errs...)
	}

	_, api := humatest.New(t)
	register(api)
	_ = api.Do(method, path, args...)
	if !cap.called {
		t.Fatal("huma.NewErrorWithContext was not called")
	}
	return cap
}

func fromCaptured(c captured) *Problem {
	return FromHuma(c.ctx, c.status, c.msg, c.errs...)
}

func jsonMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

type bridgeWant struct {
	code   string
	status int
	loc    string
	name   string
	reason string
	params map[string]any
}

func checkBridge(t *testing.T, p *Problem, w bridgeWant) {
	t.Helper()
	if p.Code != w.code || p.Status != w.status {
		t.Fatalf("code/status = %s %d, want %s %d (detail %q)", p.Code, p.Status, w.code, w.status, p.Detail)
	}
	if w.reason == "" {
		return
	}
	if len(p.Errors) == 0 {
		t.Fatalf("no field errors, want %s %s=%s", w.reason, w.loc, w.name)
	}
	fe := p.Errors[0]
	if fe.Reason != w.reason {
		t.Fatalf("reason %s want %s (detail %q)", fe.Reason, w.reason, fe.Detail)
	}
	item := jsonMap(t, fe)
	if _, has := item["code"]; has {
		t.Fatalf("field error has code: %v", item)
	}
	locs := 0
	for _, k := range []string{"pointer", "parameter", "header"} {
		if _, has := item[k]; has {
			locs++
		}
	}
	if locs != 1 {
		t.Fatalf("location members = %d: %v", locs, item)
	}
	gotLoc, _ := item[w.loc].(string)
	if gotLoc != w.name {
		t.Fatalf("%s = %q want %q (detail %q)", w.loc, gotLoc, w.name, fe.Detail)
	}
	params, _ := item["params"].(map[string]any)
	for k := range params {
		if !reasonAllowsParam(fe.Reason, k) {
			t.Fatalf("%s emitted undeclared params key %s", fe.Reason, k)
		}
	}
	if len(w.params) == 0 {
		if params != nil {
			t.Fatalf("unexpected params %v", params)
		}
		return
	}
	if params == nil {
		t.Fatalf("params missing, want %v", w.params)
	}
	for k, want := range w.params {
		got, ok := params[k]
		if !ok {
			t.Fatalf("params missing %s: %v", k, params)
		}
		if !paramEq(got, want) {
			t.Fatalf("params[%s] = %#v want %#v", k, got, want)
		}
	}
}

func paramEq(got, want any) bool {
	switch w := want.(type) {
	case int:
		n, ok := got.(float64)
		return ok && n == float64(w)
	case float64:
		n, ok := got.(float64)
		return ok && n == w
	case []string:
		arr, ok := got.([]any)
		if !ok || len(arr) != len(w) {
			return false
		}
		for i, s := range w {
			if arr[i] != s {
				return false
			}
		}
		return true
	default:
		return got == want
	}
}

func TestFromHumaBridge(t *testing.T) {
	seenParamKeys := map[string]bool{}
	record := func(p *Problem) {
		for _, fe := range p.Errors {
			item := jsonMap(t, fe)
			params, _ := item["params"].(map[string]any)
			for k := range params {
				seenParamKeys[fe.Reason+"."+k] = true
			}
		}
	}

	t.Run("body required property", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"title": nil})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/title", ReasonRequired, nil})
		record(p)
	})
	t.Run("body too long", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"title": "abcdefghi"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/title", ReasonTooLong, map[string]any{"max_length": 8}})
		record(p)
	})
	t.Run("body too short", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"title": "a"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/title", ReasonTooShort, map[string]any{"min_length": 2}})
		record(p)
	})
	t.Run("body minimum", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"age": 0})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/age", ReasonOutOfRange, map[string]any{"minimum": 1}})
		record(p)
	})
	t.Run("body maximum", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"age": 100})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/age", ReasonOutOfRange, map[string]any{"maximum": 99}})
		record(p)
	})
	t.Run("body exclusive minimum", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"score": 0})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/score", ReasonOutOfRange, map[string]any{"minimum": 0}})
		record(p)
	})
	t.Run("body exclusive maximum", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"score": 1})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/score", ReasonOutOfRange, map[string]any{"maximum": 1}})
		record(p)
	})
	t.Run("body too many items", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"tags": []string{"a", "b", "c", "d"}})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/tags", ReasonTooManyItems, map[string]any{"max_items": 3}})
		record(p)
	})
	t.Run("body too few items", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"tags": []string{"a"}})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/tags", ReasonTooFewItems, map[string]any{"min_items": 2}})
		record(p)
	})
	t.Run("body duplicate item", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"tags": []string{"a", "a"}})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/tags", ReasonDuplicateItem, nil})
		record(p)
	})
	t.Run("body unknown enum", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"kind": "gamma"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/kind", ReasonUnknownValue, map[string]any{"allowed": []string{"alpha", "beta"}}})
		record(p)
	})
	t.Run("body const", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"state": "closed"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/state", ReasonUnknownValue, map[string]any{"allowed": []string{"open"}}})
		record(p)
	})
	t.Run("body expected string", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"title": 12})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/title", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body expected boolean", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"flag": "yes"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/flag", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body expected integer", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"age": 1.5})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/age", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body format email", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"email": "not-an-email"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/email", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body format date", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"when": "2026/01/02"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/when", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body pattern", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"slug": "Hello"})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/slug", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body multipleOf", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(patchBody(t, map[string]any{"count": 3})))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/count", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body nested pointer", func(t *testing.T) {
		c := capture(t, registerNested, http.MethodPost, "/nested",
			"Content-Type: application/json", strings.NewReader(`{"items":[{"name":"ab"},{"name":"cd"},{"name":"x"}]}`))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "/items/2/name", ReasonTooShort, map[string]any{"min_length": 2}})
		record(p)
	})
	t.Run("body malformed json", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: application/json", strings.NewReader(`{`))
		p := fromCaptured(c)
		if p.Code != CodeMalformedBody || p.Status != 400 {
			t.Fatalf("malformed = %s %d", p.Code, p.Status)
		}
		if len(p.Errors) > 0 && p.Errors[0].Pointer != nil && *p.Errors[0].Pointer != "" {
			t.Fatalf("malformed pointer %q, want empty", *p.Errors[0].Pointer)
		}
		record(p)
	})
	t.Run("body anyOf", func(t *testing.T) {
		c := capture(t, func(api huma.API) {
			huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/anyof"},
				func(context.Context, *struct{ Body anyOfDoc }) (*struct{}, error) { return nil, nil })
		}, http.MethodPost, "/anyof", "Content-Type: application/json", strings.NewReader(`true`))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body oneOf none", func(t *testing.T) {
		c := capture(t, func(api huma.API) {
			huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/oneof"},
				func(context.Context, *struct{ Body oneOfDoc }) (*struct{}, error) { return nil, nil })
		}, http.MethodPost, "/oneof", "Content-Type: application/json", strings.NewReader(`true`))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("body oneOf multiple", func(t *testing.T) {
		c := capture(t, func(api huma.API) {
			huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/oneof2"},
				func(context.Context, *struct{ Body oneOfOverlapDoc }) (*struct{}, error) { return nil, nil })
		}, http.MethodPost, "/oneof2", "Content-Type: application/json", strings.NewReader(`"hi"`))
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeValidationFailed, 422, "pointer", "", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("query required", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidParameter, 400, "parameter", "q", ReasonRequired, nil})
		record(p)
	})
	t.Run("query invalid integer", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=bad")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidParameter, 400, "parameter", "q", ReasonInvalidFormat, nil})
		if p.Errors[0].Detail != "invalid integer" {
			t.Fatalf("detail %q", p.Errors[0].Detail)
		}
		record(p)
	})
	t.Run("query unknown enum", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&color=green")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeUnknownEnumValue, 400, "parameter", "color", ReasonUnknownValue, map[string]any{"allowed": []string{"red", "blue"}}})
		record(p)
	})
	t.Run("query unknown sort", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&sort=weird")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeUnknownSort, 400, "parameter", "sort", ReasonUnknownValue, map[string]any{"allowed": []string{"new", "old"}}})
		record(p)
	})
	t.Run("query limit too large", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&limit=101")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeLimitTooLarge, 400, "parameter", "limit", ReasonOutOfRange, map[string]any{"maximum": 100}})
		record(p)
	})
	t.Run("query cursor malformed", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&cursor=nope")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidCursor, 400, "parameter", "cursor", ReasonInvalidFormat, nil})
		record(p)
	})
	t.Run("query limit below one", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&limit=0")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidParameter, 400, "parameter", "limit", ReasonOutOfRange, map[string]any{"minimum": 1}})
		record(p)
	})
	t.Run("query too long", func(t *testing.T) {
		c := capture(t, registerQuery, http.MethodGet, "/query?q=1&name=abcdefghi")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidParameter, 400, "parameter", "name", ReasonTooLong, map[string]any{"max_length": 8}})
		record(p)
	})
	t.Run("header required", func(t *testing.T) {
		c := capture(t, registerHeader, http.MethodGet, "/hdr")
		p := fromCaptured(c)
		checkBridge(t, p, bridgeWant{CodeInvalidParameter, 400, "header", "X-Need", ReasonRequired, nil})
		record(p)
	})
	t.Run("unsupported media type", func(t *testing.T) {
		c := capture(t, registerBody, http.MethodPost, "/body",
			"Content-Type: text/plain", strings.NewReader("hello"))
		p := fromCaptured(c)
		if p.Code != CodeUnsupportedMediaType || p.Status != 415 {
			t.Fatalf("415 = %s %d", p.Code, p.Status)
		}
		record(p)
	})
	t.Run("internal from handler error", func(t *testing.T) {
		c := capture(t, func(api huma.API) {
			huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/boom"},
				func(context.Context, *struct{}) (*struct{}, error) {
					return nil, errors.New("sql: secret dsn leaked")
				})
		}, http.MethodGet, "/boom")
		p := fromCaptured(c)
		if p.Code != CodeInternalError || p.Status != 500 {
			t.Fatalf("500 = %s %d", p.Code, p.Status)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(p.Detail, "sql:") || strings.Contains(p.Detail, "secret") || strings.Contains(string(raw), "secret") {
			t.Fatalf("INTERNAL_ERROR leaked cause: %s", raw)
		}
		if len(p.Errors) != 0 {
			t.Fatalf("INTERNAL_ERROR errors = %#v", p.Errors)
		}
		record(p)
	})
	t.Run("not found status", func(t *testing.T) {
		p := FromHuma(nil, http.StatusNotFound, "Not Found")
		if p.Code != CodeNotFound || p.Status != 404 {
			t.Fatalf("404 = %s %d", p.Code, p.Status)
		}
	})
	t.Run("method not allowed status", func(t *testing.T) {
		p := FromHuma(nil, http.StatusMethodNotAllowed, "Method Not Allowed")
		if p.Code != CodeMethodNotAllowed || p.Status != 405 {
			t.Fatalf("405 = %s %d", p.Code, p.Status)
		}
	})
	t.Run("service unavailable status", func(t *testing.T) {
		p := FromHuma(nil, http.StatusServiceUnavailable, "down")
		if p.Code != CodeServiceUnavailable || p.Status != 503 {
			t.Fatalf("503 = %s %d", p.Code, p.Status)
		}
	})

	wantKeys := []string{
		ReasonTooLong + "." + ParamMaxLength,
		ReasonTooShort + "." + ParamMinLength,
		ReasonOutOfRange + "." + ParamMinimum,
		ReasonOutOfRange + "." + ParamMaximum,
		ReasonTooManyItems + "." + ParamMaxItems,
		ReasonTooFewItems + "." + ParamMinItems,
		ReasonUnknownValue + "." + ParamAllowed,
	}
	for _, k := range wantKeys {
		if !seenParamKeys[k] {
			t.Errorf("declared params key %s was never emitted by a bridge case", k)
		}
	}
}

func TestNamedReasonsDoNotFallThrough(t *testing.T) {
	msgs := []struct {
		msg    string
		reason string
	}{
		{"expected required property title to be present", ReasonRequired},
		{"required query parameter is missing", ReasonRequired},
		{"expected length >= 2", ReasonTooShort},
		{"expected length <= 8", ReasonTooLong},
		{"expected number >= 1", ReasonOutOfRange},
		{"expected number <= 99", ReasonOutOfRange},
		{"expected number > 0", ReasonOutOfRange},
		{"expected number < 1", ReasonOutOfRange},
		{"expected array length <= 3", ReasonTooManyItems},
		{"expected array length >= 2", ReasonTooFewItems},
		{"expected array items to be unique", ReasonDuplicateItem},
		{`expected value to be one of "alpha, beta"`, ReasonUnknownValue},
		{"expected value to be open", ReasonUnknownValue},
	}
	for _, tc := range msgs {
		reason, _, _ := mapReason(tc.msg)
		if reason != tc.reason {
			t.Errorf("%q → %s, want %s (must not fall through to INVALID_FORMAT)", tc.msg, reason, tc.reason)
		}
		if reason == ReasonInvalidFormat {
			t.Errorf("%q fell through to INVALID_FORMAT", tc.msg)
		}
	}
}

type nestedRequiredBody struct {
	Owner struct {
		Name string `json:"name"`
	} `json:"owner"`
}

func TestFromHumaRequiredPropertyInNestedObject(t *testing.T) {
	c := capture(t, func(api huma.API) {
		huma.Register(api, huma.Operation{Method: http.MethodPost, Path: "/owner"},
			func(context.Context, *struct{ Body nestedRequiredBody }) (*struct{}, error) {
				return nil, nil
			})
	}, http.MethodPost, "/owner", "Content-Type: application/json", strings.NewReader(`{"owner":{}}`))
	checkBridge(t, fromCaptured(c), bridgeWant{CodeValidationFailed, 422, "pointer", "/owner/name", ReasonRequired, nil})
}

func TestFromHumaUnauthorizedSplitsOnAuthorizationHeader(t *testing.T) {
	anonymous := capture(t, registerHeader, http.MethodGet, "/hdr")
	if got := FromHuma(anonymous.ctx, http.StatusUnauthorized, "no").Code; got != CodeMissingCredential {
		t.Fatalf("without Authorization: %s, want %s", got, CodeMissingCredential)
	}
	presented := capture(t, registerHeader, http.MethodGet, "/hdr", "Authorization: Bearer stale")
	if got := FromHuma(presented.ctx, http.StatusUnauthorized, "no").Code; got != CodeInvalidCredential {
		t.Fatalf("with Authorization: %s, want %s", got, CodeInvalidCredential)
	}
}

func TestUnknownEnumTokenLongerThanItsKeysIsStillUnknown(t *testing.T) {
	param := "resource_platforms[0]"
	fields := dropImpliedLengthErrors([]FieldError{
		{Parameter: &param, Reason: ReasonTooLong},
		{Parameter: &param, Reason: ReasonUnknownValue},
	})
	if len(fields) != 1 || fields[0].Reason != ReasonUnknownValue {
		t.Fatalf("fields %+v", fields)
	}
	if code := pickCode(nil, http.StatusBadRequest, "", fields); code != CodeUnknownEnumValue {
		t.Fatalf("code %s, want %s", code, CodeUnknownEnumValue)
	}
	other := "q"
	kept := dropImpliedLengthErrors([]FieldError{{Parameter: &other, Reason: ReasonTooLong}, {Parameter: &param, Reason: ReasonUnknownValue}})
	if len(kept) != 2 {
		t.Fatalf("a length error on another parameter must stay: %+v", kept)
	}
}
