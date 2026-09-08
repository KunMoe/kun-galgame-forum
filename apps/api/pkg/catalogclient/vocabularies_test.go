package catalogclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestVocabulariesIndexesByNameAndCachesTheAnswer(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v2/vocabularies" {
			t.Errorf("hit %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"object":"list","items":[` +
			`{"object":"vocabulary","name":"sexual","closed":true,"values":[` +
			`{"value":"safe","display_name":"Safe"},{"value":"explicit"}]},` +
			`{"object":"vocabulary","name":"violence","closed":true,"values":[{"value":"tame"}]}]}`))
	}))
	t.Cleanup(srv.Close)

	c := New(Config{BaseURL: srv.URL, AppKey: "k"})
	for range 3 {
		got, err := c.Vocabularies(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got["sexual"].Values[0].DisplayName != "Safe" {
			t.Fatalf("decoded wrong: %+v", got)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("upstream called %d times, want 1 — every edit page would pay it", n)
	}
}

func TestVocabulariesNeedsTheAppKey(t *testing.T) {
	c := New(Config{BaseURL: "https://example.invalid"})
	if _, err := c.Vocabularies(context.Background()); err == nil {
		t.Fatal("an unconfigured app plane must not silently answer an empty map")
	}
}
