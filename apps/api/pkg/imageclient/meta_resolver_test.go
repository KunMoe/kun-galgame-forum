package imageclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// An absent sexual field means the nightly grader has not reached the image
// yet. Folding that into 0 would render an unreviewed image as a clean one.
func TestMetaBatchTellsUngradedApartFromSafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"metas":{
			"`+strings.Repeat("a", 64)+`":{"width":8,"height":8,"thumbhash":"t","sexual":0},
			"`+strings.Repeat("b", 64)+`":{"width":8,"height":8,"thumbhash":"t","sexual":2},
			"`+strings.Repeat("c", 64)+`":{"width":8,"height":8,"thumbhash":"t"}}}}`)
	}))
	defer srv.Close()

	metas, err := testClient(srv.URL).MetaBatch(t.Context(), []string{
		strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64),
	})
	if err != nil {
		t.Fatalf("meta batch: %v", err)
	}

	if safe := metas[strings.Repeat("a", 64)]; safe.Sexual == nil || *safe.Sexual != 0 {
		t.Errorf("a graded-safe image must keep its explicit 0, got %v", safe.Sexual)
	}
	if metas[strings.Repeat("a", 64)].IsSexuallyExplicit() {
		t.Errorf("grade 0 must not be explicit")
	}
	if !metas[strings.Repeat("b", 64)].IsSexuallyExplicit() {
		t.Errorf("grade 2 must be explicit")
	}
	if ungraded := metas[strings.Repeat("c", 64)]; ungraded.Sexual != nil {
		t.Errorf("an ungraded image must stay nil, got %v", *ungraded.Sexual)
	}
	if metas[strings.Repeat("c", 64)].IsSexuallyExplicit() {
		t.Errorf("an ungraded image must not be treated as explicit")
	}
}

// The grade shows up the night after the upload, so an ungraded answer may not
// be cached for the life of the process the way the dimensions are.
func TestMetaResolverRefetchesOnlyTheUngraded(t *testing.T) {
	graded, ungraded := strings.Repeat("a", 64), strings.Repeat("b", 64)
	var asked [][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hashes []string `json:"hashes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		asked = append(asked, body.Hashes)
		_, _ = io.WriteString(w, `{"data":{"metas":{
			"`+graded+`":{"width":8,"height":8,"thumbhash":"t","sexual":2},
			"`+ungraded+`":{"width":8,"height":8,"thumbhash":"t"}}}}`)
	}))
	defer srv.Close()

	fresh := testClient(srv.URL).NewMetaResolver(0)
	fresh.Resolve([]string{graded, ungraded})
	fresh.Resolve([]string{graded, ungraded})
	if len(asked) != 1 {
		t.Fatalf("within the ttl nothing may be re-fetched, got %d requests", len(asked))
	}

	expired := testClient(srv.URL).NewMetaResolver(0)
	expired.gradeTTL = -time.Second
	expired.Resolve([]string{graded, ungraded})
	expired.Resolve([]string{graded, ungraded})

	if len(asked) != 3 {
		t.Fatalf("want 3 requests, got %d: %v", len(asked), asked)
	}
	if got := asked[2]; len(got) != 1 || got[0] != ungraded {
		t.Errorf("once the ttl lapses only the ungraded image is asked for again, got %v", got)
	}
}

func testClient(baseURL string) *Client {
	return New(Config{BaseURL: baseURL, ClientID: "id", ClientSecret: "secret"})
}
