package relocation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImportSendsTheRawLinesAndReadsTheReceipts(t *testing.T) {
	var auth string
	var sent struct {
		DryRun bool              `json:"dry_run"`
		Items  []json.RawMessage `json:"items"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"code":0,"message":"OK","data":{"dry_run":false,"results":[
			{"forum_id":11,"resource_id":901,"result":"created","public":true},
			{"forum_id":12,"resource_id":902,"result":"exists","public":false},
			{"forum_id":13,"result":"failed","public":false,"error":"size"}]}}`))
	}))
	defer srv.Close()

	items := []json.RawMessage{json.RawMessage(`{"forum_id":11}`), json.RawMessage(`{"forum_id":12}`), json.RawMessage(`{"forum_id":13}`)}
	receipts, err := NewLetMoe(srv.URL, "k3y").Import(t.Context(), items, false)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer k3y" || sent.DryRun || len(sent.Items) != 3 || string(sent.Items[1]) != `{"forum_id":12}` {
		t.Errorf("request: auth %q, dry_run %v, items %s", auth, sent.DryRun, sent.Items)
	}
	if !receipts[0].Landed() || receipts[0].ResourceID != 901 || !receipts[0].Public {
		t.Errorf("created receipt = %+v", receipts[0])
	}
	if !receipts[1].Landed() || receipts[1].Public {
		t.Errorf("exists receipt = %+v", receipts[1])
	}
	if receipts[2].Landed() || receipts[2].Error != "size" {
		t.Errorf("failed receipt = %+v", receipts[2])
	}
}

func TestImportRefusesAnAnswerThatDoesNotCoverTheBatch(t *testing.T) {
	answers := map[string]struct {
		status int
		body   string
	}{
		"short":    {200, `{"code":0,"data":{"results":[{"forum_id":11,"resource_id":901,"result":"created"}]}}`},
		"refused":  {401, `{"code":40100,"message":"bad key"}`},
		"app code": {200, `{"code":50000,"message":"boom"}`},
		"not json": {200, `<html>`},
	}
	for name, a := range answers {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(a.status)
			_, _ = w.Write([]byte(a.body))
		}))
		_, err := NewLetMoe(srv.URL, "k").Import(t.Context(), []json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{}`)}, true)
		srv.Close()
		if err == nil {
			t.Errorf("%s: no error", name)
		} else if name == "refused" && !strings.Contains(err.Error(), "bad key") {
			t.Errorf("%s: %v, want LetMoe's message", name, err)
		}
	}
}
