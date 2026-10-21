package catalogclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	state, err := New(Config{BaseURL: srv.URL}).DecideProposal(context.Background(), "user-jwt", 41, "merge", "", `"p3"`)
	if err != nil || state != "merged" {
		t.Fatalf("DecideProposal: %q %v", state, err)
	}
	if path != "/v2/moderation/proposals/41/decisions" || ifMatch != `"p3"` {
		t.Errorf("hit %s with If-Match %q", path, ifMatch)
	}
}
