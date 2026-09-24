package problem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestWireShapeEmptyErrors(t *testing.T) {
	p := New(CodeNotFound, "gone")
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	v, ok := m["errors"]
	if !ok {
		t.Fatalf("errors missing: %s", raw)
	}
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("errors is %T, want array (never null): %s", v, raw)
	}
	if arr == nil || len(arr) != 0 {
		t.Fatalf("errors = %#v, want []", arr)
	}
	if strings.Contains(string(raw), `"errors":null`) {
		t.Fatalf("marshalled null errors: %s", raw)
	}
}

func TestWireShapeFieldError(t *testing.T) {
	max := 8
	p := New(CodeValidationFailed, "bad", AtPointer("/title", ReasonTooLong, "expected length <= 8", &FieldParams{MaxLength: &max}))
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	items, ok := m["errors"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("errors = %#v", m["errors"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("item %T", items[0])
	}
	if _, has := item["code"]; has {
		t.Fatalf("field error has code: %s", raw)
	}
	locs := 0
	for _, k := range []string{"pointer", "parameter", "header"} {
		if _, has := item[k]; has {
			locs++
		}
	}
	if locs != 1 {
		t.Fatalf("location members = %d: %s", locs, raw)
	}
	if item["pointer"] != "/title" {
		t.Fatalf("pointer = %v", item["pointer"])
	}
	params, ok := item["params"].(map[string]any)
	if !ok {
		t.Fatalf("params missing: %s", raw)
	}
	if params["max_length"] != float64(8) {
		t.Fatalf("max_length = %v", params["max_length"])
	}
}

func TestParamsOmittedWhenEmpty(t *testing.T) {
	p := New(CodeValidationFailed, "bad", AtPointer("/title", ReasonRequired, "missing", nil))
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	item := m["errors"].([]any)[0].(map[string]any)
	if _, has := item["params"]; has {
		t.Fatalf("empty params present: %s", raw)
	}
}

func TestParamsKeysMatchReasonSchema(t *testing.T) {
	max := 3
	min := 1
	lo, hi := 2.0, 9.0
	allowed := []string{"a", "b"}
	cases := []FieldError{
		AtPointer("/a", ReasonTooLong, "x", &FieldParams{MaxLength: &max}),
		AtPointer("/a", ReasonTooShort, "x", &FieldParams{MinLength: &min}),
		AtPointer("/a", ReasonOutOfRange, "x", &FieldParams{Minimum: &lo, Maximum: &hi}),
		AtPointer("/a", ReasonTooManyItems, "x", &FieldParams{MaxItems: &max}),
		AtPointer("/a", ReasonTooFewItems, "x", &FieldParams{MinItems: &min}),
		AtPointer("/a", ReasonUnknownValue, "x", &FieldParams{Allowed: &allowed}),
		AtPointer("/a", ReasonRequired, "x", nil),
	}
	for _, fe := range cases {
		raw, err := json.Marshal(fe)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		params, _ := m["params"].(map[string]any)
		for k := range params {
			if !reasonAllowsParam(fe.Reason, k) {
				t.Errorf("%s emitted undeclared params key %s: %s", fe.Reason, k, raw)
			}
		}
	}
}

func TestExtensionMembers(t *testing.T) {
	const code = "TEST_ONLY_MERGED"
	prev, existed := codeByName[code]
	codeByName[code] = Def{
		Code:        code,
		Domain:      DomainKungal,
		Status:      http.StatusNotFound,
		Title:       "Test only",
		Description: "test-only definition",
		Extensions:  []ExtDef{{Name: "current_id", Type: "string"}},
	}
	t.Cleanup(func() {
		if existed {
			codeByName[code] = prev
		} else {
			delete(codeByName, code)
		}
	})

	p := New(code, "merged")
	p.SetExtension("current_id", "12")
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["current_id"] != "12" {
		t.Fatalf("current_id = %v, body %s", m["current_id"], raw)
	}

	undeclared := New(CodeNotFound, "gone")
	defer func() {
		if recover() == nil {
			t.Fatal("undeclared extension did not panic")
		}
	}()
	undeclared.SetExtension("current_id", "12")
}

func TestInternalDetailHidesCause(t *testing.T) {
	cause := errors.New("sql: connection refused at 10.0.0.1")
	p := Internal(cause)
	if strings.Contains(p.Detail, "sql:") || strings.Contains(p.Detail, "10.0.0.1") || strings.Contains(p.Detail, cause.Error()) {
		t.Fatalf("detail leaked cause: %q", p.Detail)
	}
	if p.Code != CodeInternalError || p.Status != 500 {
		t.Fatalf("code/status = %s %d", p.Code, p.Status)
	}
}

func TestWriteHeadersAndInstance(t *testing.T) {
	app := fiber.New()
	app.Get("/x", func(c fiber.Ctx) error {
		return Write(c, New(CodeNotFound, "missing"))
	})
	req := httptest.NewRequest(http.MethodGet, "/x?q=1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d body %s", resp.StatusCode, raw)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != ContentType && !strings.HasPrefix(ct, ContentType) {
		t.Fatalf("Content-Type %q", ct)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control %q", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("WWW-Authenticate") != "" {
		t.Fatalf("WWW-Authenticate on non-401: %q", resp.Header.Get("WWW-Authenticate"))
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("%s: %s", err, raw)
	}
	if body["instance"] != "/x?q=1" {
		t.Fatalf("instance %v", body["instance"])
	}
	if resp.Header.Get(HeaderRequestID) != body["request_id"] {
		t.Fatalf("header %q body %v", resp.Header.Get(HeaderRequestID), body["request_id"])
	}
	if !ValidRequestID(resp.Header.Get(HeaderRequestID)) {
		t.Fatalf("request id %q", resp.Header.Get(HeaderRequestID))
	}
}

func TestWriteWWWAuthenticateOn401(t *testing.T) {
	app := fiber.New()
	app.Get("/x", func(c fiber.Ctx) error {
		return Write(c, New(CodeMissingCredential, "no token"))
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
	got := resp.Header.Get("WWW-Authenticate")
	if got != `Bearer realm="kungal"` {
		t.Fatalf("WWW-Authenticate %q", got)
	}
}

func TestWriteReusesInboundRequestID(t *testing.T) {
	const id = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if !ValidRequestID(id) {
		t.Fatalf("fixture id rejected: %s", id)
	}
	app := fiber.New()
	app.Get("/x", func(c fiber.Ctx) error {
		return Write(c, New(CodeNotFound, "missing"))
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, id)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get(HeaderRequestID) != id || body["request_id"] != id {
		t.Fatalf("header %q body %v", resp.Header.Get(HeaderRequestID), body["request_id"])
	}
}

func TestMarshalOfALiteralProblemNeverEmitsNullErrors(t *testing.T) {
	raw, err := json.Marshal(&Problem{Code: CodeNotFound, Status: http.StatusNotFound})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"errors":[]`) {
		t.Fatalf("errors not [] on a literal Problem: %s", raw)
	}
}

func TestWriteLogsTheInternalCauseUnderTheRequestID(t *testing.T) {
	var logs strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(prev)

	app := fiber.New()
	app.Get("/x", func(c fiber.Ctx) error {
		return Write(c, Internal(errors.New("pq: relation topic_secret does not exist")))
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(raw), "topic_secret") {
		t.Fatalf("cause reached the body: %s", raw)
	}
	id := resp.Header.Get(HeaderRequestID)
	if !strings.Contains(logs.String(), "topic_secret") || !strings.Contains(logs.String(), id) {
		t.Fatalf("log %q lacks the cause or request id %s", logs.String(), id)
	}
}

type throttledErr struct{ throttled bool }

func (e throttledErr) Error() string   { return "upstream said 429" }
func (e throttledErr) Throttled() bool { return e.throttled }

func TestLogCauseLevelFollowsThrottled(t *testing.T) {
	var logs strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(prev)

	p := Unavailable(fmt.Errorf("wrapped: %w", throttledErr{throttled: true}))
	p.RequestID = "req_throttled"
	LogCause(p)
	if !strings.Contains(logs.String(), "level=WARN") || strings.Contains(logs.String(), "level=ERROR") || !strings.Contains(logs.String(), "req_throttled") {
		t.Fatalf("a throttled cause must log one WARN with the request id: %q", logs.String())
	}

	logs.Reset()
	LogCause(Unavailable(throttledErr{throttled: false}))
	if !strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("an unthrottled cause must still log ERROR: %q", logs.String())
	}
}
