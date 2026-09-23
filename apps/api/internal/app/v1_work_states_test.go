package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func TestV1ListMyWorkStatesMissingAndLiked(t *testing.T) {
	f := newWorkFix(t)
	if _, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil); body["code"] != nil {
		t.Fatalf("seed like %+v", body)
	}
	ids := strings.Join([]string{idStr(g4WorkHidden), idStr(g4WorkNoLocal), idStr(g4WorkLive), idStr(g4WorkUnknown)}, ",")
	resp, body := f.wk(t, http.MethodGet, "/api/v1/me/work-states?work_ids="+ids, "/me/work-states", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("states %d %+v", resp.StatusCode, body)
	}
	missing := map[string]bool{}
	for _, raw := range body["missing"].([]any) {
		missing[strID(raw)] = true
	}
	if !missing[idStr(g4WorkHidden)] || !missing[idStr(g4WorkUnknown)] {
		t.Errorf("missing %+v", body["missing"])
	}
	items, _ := body["items"].([]any)
	byID := map[string]map[string]any{}
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		byID[strID(it["work_id"])] = it
	}
	if byID[idStr(g4WorkHidden)] != nil || byID[idStr(g4WorkUnknown)] != nil {
		t.Errorf("hidden/unknown in items %+v", items)
	}
	live := byID[idStr(g4WorkLive)]
	if live == nil || live["has_liked"] != true {
		t.Errorf("live %+v", live)
	}
	ghost := byID[idStr(g4WorkNoLocal)]
	if ghost == nil || ghost["has_liked"] != false {
		t.Errorf("readable unliked %+v", ghost)
	}
}

func TestV1ListMyWorkStatesFolderScope(t *testing.T) {
	f := newWorkFix(t)
	f.user.scopeFolders = true
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.wk(t, http.MethodGet, "/api/v1/me/work-states?work_ids="+idStr(g4WorkLive), "/me/work-states", "sess-alice", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("states %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items %+v", items)
	}
	it, _ := items[0].(map[string]any)
	if it["has_favorited"] != false {
		t.Errorf("has_favorited %+v", it)
	}
	if !strings.Contains(buf.String(), "token lacks folder:read") {
		t.Errorf("log %q", buf.String())
	}
}

func TestV1ListMyWorkStatesHoldingsErrorIsUnavailable(t *testing.T) {
	f := newWorkFix(t)
	f.user.holdingsErr = catalogclient.ErrUnauthorized
	resp, body := f.wk(t, http.MethodGet, "/api/v1/me/work-states?work_ids="+idStr(g4WorkLive), "/me/work-states", "sess-alice", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
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
