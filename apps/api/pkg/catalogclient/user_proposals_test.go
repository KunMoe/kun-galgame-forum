package catalogclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAmendMyProposal_DecodesTheProposalItAnswers(t *testing.T) {
	var ifMatch, key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifMatch, key = r.Header.Get("If-Match"), r.Header.Get("Idempotency-Key")
		w.Header().Set("ETag", `"p3"`)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"proposal","id":"41","state":"open","target_object":"work","entity_type":"catalog.work",` +
			`"entity_id":"1000","note":"n","proposer_uid":"9","site":"kungal","base_revision_seq":2,"decided_by_uid":null,` +
			`"decided_at":null,"created_at":"2026-09-20T00:00:00Z","updated_at":"2026-09-21T00:00:00Z",` +
			`"amendments":[{"id":"1","seq":1,"amender_uid":"9","note":"a","created_at":"2026-09-21T00:00:00Z"}]}`))
	}))
	t.Cleanup(srv.Close)

	prop, etag, err := New(Config{BaseURL: srv.URL}).AmendMyProposal(context.Background(), "user-jwt", 41,
		map[string]any{"catalog.work.display_name": "x"}, nil, "a", `"p2"`, "k-7")
	if err != nil {
		t.Fatalf("AmendMyProposal: %v", err)
	}
	if prop.ID != 41 || prop.Status != "open" || len(prop.Amendments) != 1 {
		t.Fatalf("proposal = %+v", prop)
	}
	if ifMatch != `"p2"` || key != "k-7" || etag != `"p3"` {
		t.Errorf("If-Match %q, Idempotency-Key %q, ETag %q", ifMatch, key, etag)
	}
}

func TestProposalWithoutAStateStaysEmpty(t *testing.T) {
	if got := (v2Proposal{}).proposal().Status; got != "" {
		t.Errorf("status = %q; an absent state must not read as open", got)
	}
}

func TestDecideProposal_SendsTheDecision(t *testing.T) {
	var path, ifMatch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, ifMatch = r.URL.Path, r.Header.Get("If-Match")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"decision","id":"41","decision":"merge","note":"","from_state":"open","to_state":"merged"}`))
	}))
	t.Cleanup(srv.Close)

	if err := New(Config{BaseURL: srv.URL}).DecideProposal(context.Background(), "user-jwt", 41, "merge", "", `"p3"`); err != nil {
		t.Fatalf("DecideProposal: %v", err)
	}
	if path != "/v2/moderation/proposals/41/decisions" || ifMatch != `"p3"` {
		t.Errorf("hit %s with If-Match %q", path, ifMatch)
	}
}
