package app

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func myWorksByID(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, raw := range body["items"].([]any) {
		it, _ := raw.(map[string]any)
		out[strID(it["work_id"])] = it
	}
	return out
}

func TestV1ListMyWorksMissingLikedAndLibrary(t *testing.T) {
	f := newWorkFix(t)
	if _, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil); body["code"] != nil {
		t.Fatalf("seed like %+v", body)
	}
	f.user.containing = map[int64]bool{g4WorkLive: true}
	f.user.play = &catalogclient.PlaytimeSelf{WorkID: g4WorkLive, Minutes: 120}
	main := "main"
	f.user.state = &catalogclient.WorkStateRecord{WorkID: g4WorkLive, State: "done", Completion: &main}
	ids := strings.Join([]string{idStr(g4WorkHidden), idStr(g4WorkNoLocal), idStr(g4WorkLive), idStr(g4WorkUnknown), idStr(g4WorkLive)}, ",")
	resp, body := f.wk(t, http.MethodGet, "/api/v1/me/works?work_ids="+ids, "/me/works", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("my works %d %+v", resp.StatusCode, body)
	}
	missing := map[string]bool{}
	for _, raw := range body["missing"].([]any) {
		missing[strID(raw)] = true
	}
	if !missing[idStr(g4WorkHidden)] || !missing[idStr(g4WorkUnknown)] {
		t.Errorf("missing %+v", body["missing"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 || strID(items[0].(map[string]any)["work_id"]) != idStr(g4WorkNoLocal) {
		t.Fatalf("want the two readable works once each in request order: %+v", items)
	}
	byID := myWorksByID(t, body)
	live := byID[idStr(g4WorkLive)]
	lib, _ := live["library"].(map[string]any)
	if live["has_liked"] != true || lib == nil {
		t.Fatalf("live %+v", live)
	}
	if cols, _ := lib["collection_ids"].([]any); len(cols) != 1 || strID(cols[0]) != "1" {
		t.Errorf("collection_ids %+v", lib["collection_ids"])
	}
	if pt, _ := lib["playtime"].(map[string]any); pt == nil || asInt(pt["minutes"]) != 120 || pt["play_state"] != "done_main" {
		t.Errorf("playtime %+v", lib["playtime"])
	}
	ghost, _ := byID[idStr(g4WorkNoLocal)]["library"].(map[string]any)
	if cols, ok := ghost["collection_ids"].([]any); !ok || len(cols) != 0 || ghost["playtime"] != nil {
		t.Errorf("a work the reader holds nowhere answers [] and no playtime: %+v", ghost)
	}
	if len(f.user.sentWorks) != 1 || fmt.Sprint(f.user.sentWorks[0]) != fmt.Sprint([]int64{g4WorkNoLocal, g4WorkLive}) {
		t.Errorf("catalog saw %v; a hidden or unknown work must never reach /v2/me/works", f.user.sentWorks)
	}
}

// A failure reading catalog is "unknown", never "not collected", and it never
// takes the likes down with it.
func TestV1ListMyWorksLibraryUnknownWhenCatalogFails(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(*workFix)
		bearer  bool
		wantLog string
		quiet   bool
	}{
		{"cookie session without folder:read", func(f *workFix) { f.user.scopeFolders = true }, false, "token lacks folder:read", false},
		{"bearer app token without folder:read", func(f *workFix) { f.user.scopeFolders = true }, true, "", true},
		{"throttled", func(f *workFix) {
			f.user.worksErr = &catalogclient.UserAPIError{Status: http.StatusTooManyRequests, Message: "Short-window rate limit exceeded."}
		}, false, "upstream_status=429", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newWorkFix(t)
			if _, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil); body["code"] != nil {
				t.Fatalf("seed like %+v", body)
			}
			c.setup(f)
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })
			url := "/api/v1/me/works?work_ids=" + idStr(g4WorkLive)
			var (
				resp *http.Response
				body map[string]any
			)
			if c.bearer {
				var raw []byte
				resp, raw = f.doJSON(t, http.MethodGet, url, "", "/me/works", "", http.Header{"Authorization": {"Bearer bob-token"}}, nil)
				body = problemMap(t, raw)
			} else {
				resp, body = f.wk(t, http.MethodGet, url, "/me/works", "sess-bob", nil)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("the batch must answer 200 with likes: %d %+v", resp.StatusCode, body)
			}
			live := myWorksByID(t, body)[idStr(g4WorkLive)]
			if live["has_liked"] != true {
				t.Errorf("likes went with the catalog failure: %+v", live)
			}
			if v, ok := live["library"]; !ok || v != nil {
				t.Errorf("library must be null (unknown): %+v", live)
			}
			logs := buf.String()
			if c.wantLog != "" && (!strings.Contains(logs, "level=WARN") || !strings.Contains(logs, c.wantLog)) {
				t.Errorf("want a WARN with %q, got %q", c.wantLog, logs)
			}
			if c.quiet && (strings.Contains(logs, "level=WARN") || strings.Contains(logs, "level=ERROR")) {
				t.Errorf("a Bearer app token without folder:read is the App's ordinary case, not a WARN: %q", logs)
			}
		})
	}
}

func TestV1ListMyWorksDecodesPlayStateStrictly(t *testing.T) {
	odd := "sideways"
	for _, st := range []catalogclient.WorkStateRecord{
		{WorkID: g4WorkLive, State: "finished"},
		{WorkID: g4WorkLive, State: "done", Completion: &odd},
	} {
		f := newWorkFix(t)
		f.user.play = &catalogclient.PlaytimeSelf{WorkID: g4WorkLive, Minutes: 45}
		f.user.state = &st
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		resp, body := f.wk(t, http.MethodGet, "/api/v1/me/works?work_ids="+idStr(g4WorkLive), "/me/works", "sess-bob", nil)
		slog.SetDefault(prev)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %+v", st.State, resp.StatusCode, body)
		}
		lib, _ := myWorksByID(t, body)[idStr(g4WorkLive)]["library"].(map[string]any)
		pt, _ := lib["playtime"].(map[string]any)
		if pt == nil || asInt(pt["minutes"]) != 45 || pt["play_state"] != nil {
			t.Errorf("%s/%v: an unknown state nulls play_state only: %+v", st.State, st.Completion, lib)
		}
		if !strings.Contains(buf.String(), "unknown catalog work state") {
			t.Errorf("%s: no WARN for the unknown state: %q", st.State, buf.String())
		}
	}
}

func TestV1WorkRefJSONKeysAppearOnWork(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodGet, g4WorkPath(g4WorkLive), "/works/{work_id}", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	ref := map[string]any{}
	for _, k := range []string{"object", "id", "display_name", "latin", "localized", "cover", "is_nsfw"} {
		v, ok := body[k]
		if !ok {
			t.Errorf("Work missing WorkRef key %s", k)
			continue
		}
		ref[k] = v
		if asJSONType(body[k]) != asJSONType(v) {
			t.Errorf("%s type %s", k, asJSONType(body[k]))
		}
	}
}

func TestV1AppPlaneThrottleLogsWarnNotError(t *testing.T) {
	f := newWorkFix(t)
	f.cat.throttle.Store(true)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.wk(t, http.MethodGet, "/api/v1/me/works?work_ids="+idStr(g4WorkLive), "/me/works", "sess-bob", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	wantThrottleLoggedOnce(t, buf.String(), resp.Header.Get("X-Request-ID"))
}

// g-plan ②: an upstream 429 is a quota state. It logs one WARN that carries
// the request id, and no ERROR.
func wantThrottleLoggedOnce(t *testing.T, logs, requestID string) {
	t.Helper()
	if requestID == "" {
		t.Fatal("no X-Request-ID on the response")
	}
	var warns int
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		if strings.Contains(line, "level=ERROR") {
			t.Errorf("a throttled upstream logged ERROR: %s", line)
		}
		if strings.Contains(line, "level=WARN") && strings.Contains(line, `msg="problem cause"`) {
			warns++
			if !strings.Contains(line, requestID) {
				t.Errorf("the WARN lacks request_id %s: %s", requestID, line)
			}
		}
	}
	if warns != 1 {
		t.Errorf("want exactly one WARN for the 429, got %d in %q", warns, logs)
	}
}
