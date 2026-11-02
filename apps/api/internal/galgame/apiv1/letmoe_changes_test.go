package apiv1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/letmoeclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRunLetmoeChanges(t *testing.T) {
	var asked []string
	pages := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/letmoe/changes" {
			t.Errorf("path %s", r.URL.Path)
		}
		cursor := r.URL.Query().Get("cursor")
		asked = append(asked, cursor)
		body, ok := pages[cursor]
		if !ok {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"code":"INVALID_CURSOR","detail":"expired"}`)
			return
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	s := New(nil, nil, nil, rdb, "").WithLetmoe(letmoeclient.New(letmoeclient.Config{BaseURL: srv.URL, APIKey: "nmk_test"}))
	ctx := context.Background()
	cursor := func() string { v, _ := mr.Get(letmoeChangesCursorKey); return v }

	pages[""] = `{"object":"list","items":[],"next_cursor":"cur_now","total":null}`
	s.RunLetmoeChanges(ctx)
	if cursor() != "cur_now" {
		t.Fatalf("first read must store the feed's starting cursor, got %q", cursor())
	}

	_ = mr.Set(letmoeCacheKey+"61311", "[]")
	_ = mr.Set(letmoeCacheKey+"7", "[]")
	full := make([]string, letmoeChangesLimit)
	for i := range full {
		full[i] = fmt.Sprintf(`{"object":"work_change","work_id":"%d","changed_at":"2026-11-01T00:00:00Z"}`, 1000+i)
	}
	pages["cur_now"] = `{"object":"list","items":[` + strings.Join(full, ",") + `],"next_cursor":"cur_a","total":null}`
	pages["cur_a"] = `{"object":"list","items":[{"object":"work_change","work_id":"61311","changed_at":"2026-11-01T00:00:00Z"}],"next_cursor":"cur_b","total":null}`
	asked = nil
	s.RunLetmoeChanges(ctx)
	if strings.Join(asked, ",") != "cur_now,cur_a" {
		t.Errorf("a full page must be followed by the next one in the same tick: %v", asked)
	}
	if mr.Exists(letmoeCacheKey + "61311") {
		t.Error("a changed work's cached list must be dropped")
	}
	if !mr.Exists(letmoeCacheKey + "7") {
		t.Error("an unchanged work's cached list must stay")
	}
	if cursor() != "cur_b" {
		t.Errorf("cursor %q", cursor())
	}

	pages["cur_b"] = `{"object":"list","items":[],"next_cursor":"cur_b","total":null}`
	asked = nil
	s.RunLetmoeChanges(ctx)
	if len(asked) != 1 || cursor() != "cur_b" {
		t.Errorf("a quiet feed is one call that keeps the cursor: %v %q", asked, cursor())
	}

	delete(pages, "cur_b")
	_ = mr.Set(letmoeCacheKey+"8", "[]")
	_ = mr.Set("letmoe:other", "x")
	s.RunLetmoeChanges(ctx)
	if mr.Exists(letmoeCacheKey+"7") || mr.Exists(letmoeCacheKey+"8") || mr.Exists(letmoeChangesCursorKey) {
		t.Errorf("INVALID_CURSOR must drop every cached list and the cursor: %v", mr.Keys())
	}
	if !mr.Exists("letmoe:other") {
		t.Error("only letmoe resource lists are dropped")
	}
}
