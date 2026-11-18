package platformsettings

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientFollowsTheReadFace(t *testing.T) {
	var calls, notModified atomic.Int32
	enabled := `true`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/v1/settings" || r.Header.Get("Authorization") != "Basic eDp5" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("If-None-Match") == `"e1"` {
			notModified.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"e1"`)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"site_id":2,"etag":"\"e1\"","settings":{"agent.enabled":` + enabled + `}}}`))
	}))
	defer srv.Close()

	var nilClient *Client
	if nilClient.Bool(AgentEnabled) {
		t.Fatal("a nil client reads true")
	}
	c := New(srv.URL+"/api/v1/", "x", "y")
	if c.Bool(AgentEnabled) {
		t.Fatal("true before any read")
	}
	stop := c.Start()
	defer stop()
	if !c.Bool(AgentEnabled) {
		t.Fatal("agent.enabled not read on start")
	}
	if err := c.refresh(t.Context()); err != nil || notModified.Load() != 1 || !c.Bool(AgentEnabled) {
		t.Fatalf("304 handling: err=%v notModified=%d", err, notModified.Load())
	}

	c.url = srv.URL + "/wrong"
	if err := c.refresh(t.Context()); err == nil || !c.Bool(AgentEnabled) {
		t.Fatalf("a failed poll must keep the last snapshot: err=%v", err)
	}
}
