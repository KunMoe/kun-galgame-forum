package service

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
)

type wizardRecorder struct {
	mu         sync.Mutex
	catalogQ   url.Values
	claimsQ    url.Values
	claimsPath string
	claimsAuth string
	claimsHit  int
	wikiHits   int
}

func (r *wizardRecorder) service(t *testing.T) *SubmissionService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		body := `{"items":[],"total":0}`
		switch {
		case req.URL.Path == "/v2/catalog/works":
			r.catalogQ = req.URL.Query()
			body = `{"total":2,"items":[
			  {"id":11,"display_name":"A","cover":"https://img/aa/bb/hash1.webp",
			   "claim":{"site":"kungal","site_work_id":"292","state":"live"},
			   "localized":{"zh-Hans":{"value":"白恋樱","kind":"official"}},"refs":[{"source":"vndb","external_id":"v22610"}]},
			  {"id":12,"display_name":"B","cover":"",
			   "claim":{"site":"kungal","site_work_id":"9978","state":"draft"}},
			  {"id":13,"display_name":"withdrawn","cover":"",
			   "claim":{"site":"kungal","site_work_id":"404","state":"hidden"}},
			  {"id":14,"display_name":"unclaimed","cover":"","claim":null},
			  {"id":15,"display_name":"declined","cover":"",
			   "claim":{"site":"kungal","site_work_id":"331","state":"declined"}},
			  {"id":16,"display_name":"pending","cover":"",
			   "claim":{"site":"kungal","site_work_id":"5150","state":"pending"}},
			  {"id":17,"display_name":"gidless","cover":"",
			   "claim":{"site":"kungal","site_work_id":"0","state":"live"}},
			  {"id":18,"display_name":"foreign","cover":"",
			   "claim":{"site":"moyu","site_work_id":"700","state":"live"}}
			]}`
		case strings.Contains(req.URL.Path, "/v2/me/claims") || strings.Contains(req.URL.Path, "/claims"):
			r.claimsQ = req.URL.Query()
			r.claimsPath = req.URL.Path
			r.claimsAuth = req.Header.Get("Authorization")
			r.claimsHit++
			body = `{"items":[
			  {"work_id":64689,"display_name":"曇った瞳に恋してる","site":"kungal",
			   "product_work_id":64689,"claim_state":"pending","last_event_id":9,
			   "last_from_state":"draft","last_to_state":"pending","last_reason":null,
			   "last_actor_uid":7,"last_event_at":"2026-07-31T00:00:00Z",
			   "first_acted_at":"2026-07-31T00:00:00Z","acted_count":1}
			],"next_before":0,"total":1}`
		case strings.HasSuffix(req.URL.Path, "/galgame/search"):
			r.wikiHits++
		}
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewSubmissionService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL}),
		nil,
	)
}

func wizardSearch(t *testing.T, svc *SubmissionService) *WizardSearchPage {
	t.Helper()
	page, appErr := svc.SearchWithPending(t.Context(), "user-jwt",
		url.Values{"q": {"sakura"}, "limit": {"12"}})
	if appErr != nil {
		t.Fatalf("SearchWithPending: %v", appErr)
	}
	return page
}

func TestWizard_ItemsComeFromTheCatalogSearch(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if _, sent := rec.catalogQ["claim_state"]; sent {
		t.Errorf("claim_state = %q, want it absent — the facet is the index's and lags a day; "+
			"CatalogItemWizardEligible gates on the hydrated registry state instead",
			rec.catalogQ.Get("claim_state"))
	}
	if got := rec.catalogQ.Get("claimed"); got != "" {
		t.Errorf("claimed = %q, want it absent — the wizard must also surface unclaimed works", got)
	}
	if got := rec.catalogQ.Get("q"); got != "sakura" {
		t.Errorf("q = %q, want sakura", got)
	}
	if got := rec.catalogQ.Get("limit"); got != "12" {
		t.Errorf("limit = %q, want 12", got)
	}
	if got := rec.catalogQ.Get("nsfw"); got != "true" {
		t.Errorf("nsfw = %q, want true", got)
	}
	if got := rec.catalogQ.Get("content_limit"); got != "" {
		t.Errorf("content_limit = %q, want it absent on the wizard lane", got)
	}
	if !strings.Contains(rec.catalogQ.Get("include"), "refs") {
		t.Errorf("include = %q, want refs (the row prints the VNDB id)", rec.catalogQ.Get("include"))
	}
	if page.Total != 2 {
		t.Errorf("total = %d, want the catalog total 2", page.Total)
	}
}

func TestWizard_ItemsAreKeyedByWorkIDAndDropWithdrawnRows(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if len(page.Items) != 4 {
		t.Fatalf("items = %d, want 4 — live, draft, pending and the unclaimed row stay; "+
			"hidden, declined, gidless and foreign-claimed rows are not actionable", len(page.Items))
	}
	for _, it := range page.Items {
		if it.ClaimState == "declined" || it.ClaimState == "hidden" {
			t.Errorf("item %d leaked with claim_state=%q — the wizard must drop non-live/draft/pending rows", it.ID, it.ClaimState)
		}
		if it.ID <= 0 {
			t.Errorf("item with claim_state=%q leaked with no work id", it.ClaimState)
		}
	}
	if page.Items[0].ID != 11 || page.Items[1].ID != 12 || page.Items[3].ID != 16 {
		t.Errorf("ids = %d,%d,%d, want the catalog work ids 11,12,16",
			page.Items[0].ID, page.Items[1].ID, page.Items[3].ID)
	}
	if page.Items[2].ID != 14 {
		t.Errorf("unclaimed row = id %d, want catalog work id 14", page.Items[2].ID)
	}
	if page.Items[2].ClaimState != "none" {
		t.Errorf("unclaimed row claim_state = %q, want none — calling it a draft made the wizard "+
			"offer 认领此草稿 over a work nobody had ever touched, and routed it at the claim endpoint",
			page.Items[2].ClaimState)
	}
	if page.Items[0].VndbID != "v22610" {
		t.Errorf("vndb_id = %q, want v22610", page.Items[0].VndbID)
	}
	if page.Items[0].ClaimState != "live" || page.Items[1].ClaimState != "draft" {
		t.Errorf("claim_state = %q,%q, want live,draft — the wizard branches on it",
			page.Items[0].ClaimState, page.Items[1].ClaimState)
	}
	if page.Items[0].EffectiveBannerURL == "" {
		t.Error("effective_banner_url is empty — the wizard row renders the cover from it")
	}
}

func TestWizard_PendingComesFromThePerUserClaimFace(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if rec.claimsHit != 1 {
		t.Fatalf("per-user claim face hits = %d, want exactly 1", rec.claimsHit)
	}
	if rec.wikiHits != 0 {
		t.Errorf("wiki face hits = %d, want 0 — the pending half is terminal now", rec.wikiHits)
	}
	if rec.claimsPath != "/v2/me/claims" {
		t.Errorf("pending half hit %q, want the user plane's own-claims face", rec.claimsPath)
	}
	if rec.claimsAuth != "Bearer user-jwt" {
		t.Errorf("pending half auth = %q, want the caller's bearer", rec.claimsAuth)
	}
	if got := rec.claimsQ.Get("site"); got != "" {
		t.Errorf("site = %q, want it absent — the acting tenant rides the token", got)
	}
	if got := rec.claimsQ.Get("claim_state"); got != "pending,declined" {
		t.Errorf("claim_state = %q, want pending,declined", got)
	}
	if got := rec.claimsQ.Get("kind"); got != "submitted" {
		t.Errorf("kind = %q, want submitted — the wizard must list only the caller's own claims, not their reviews", got)
	}
	if len(page.Pending) != 1 || page.Pending[0].WorkID != 64689 {
		t.Fatalf("pending = %+v, want the caller's own pending claim", page.Pending)
	}
	if page.Pending[0].ClaimState != "pending" || page.Pending[0].LastToState != "pending" {
		t.Errorf("pending row = %+v, want the current state and its last transition", page.Pending[0])
	}
}

func TestWizard_PendingIsAnEmptyArrayWhenTheUserHasNone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":null,"next_before":0,"total":0}`))
	}))
	t.Cleanup(srv.Close)
	svc := NewSubmissionService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL}),
		nil,
	)

	page := wizardSearch(t, svc)
	if page.Pending == nil || len(page.Pending) != 0 {
		t.Errorf("pending = %v, want an empty array", page.Pending)
	}
}
