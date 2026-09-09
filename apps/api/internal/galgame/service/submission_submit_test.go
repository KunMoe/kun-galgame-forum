package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

type submitRecorder struct {
	mu             sync.Mutex
	body           map[string]any
	submitPath     string
	submitAuth     string
	editBody       map[string]any
	editPath       string
	editAuth       string
	mergePath      string
	mergeAuth      string
	mergedOnCreate bool
	mergeStatus    int
	claimState     string
	claimStatus    int
}

func (r *submitRecorder) service(t *testing.T) *SubmissionService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)

		r.mu.Lock()
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v2/me/claims":
			r.body = body
			r.submitPath = req.URL.Path
			r.submitAuth = req.Header.Get("Authorization")
		case req.Method == http.MethodPost && req.URL.Path == "/v2/me/proposals":
			r.editBody = body
			r.editPath = req.URL.Path
			r.editAuth = req.Header.Get("Authorization")
		case strings.HasSuffix(req.URL.Path, "/decisions"):
			r.mergePath = req.URL.Path
			r.mergeAuth = req.Header.Get("Authorization")
		}
		r.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/v2/me/claims":
			if r.claimStatus != 0 {
				w.WriteHeader(r.claimStatus)
				_, _ = w.Write([]byte(`{"code":"DUPLICATE_SUSPECTS","detail":` +
					`"1 live work(s) share a title with this submission; ` +
					`re-send with confirm_duplicates=true to mint anyway"}`))
				return
			}
			state := r.claimState
			if state == "" {
				state = "pending"
			}
			_, _ = w.Write([]byte(`{"object":"claim","id":"90210","state":"` + state +
				`","display_name":"白恋サクラ"}`))
		case strings.HasSuffix(req.URL.Path, "/catalog/lookup/batch"):
			var body struct {
				Items []struct {
					ExternalID string `json:"external_id"`
				} `json:"items"`
			}
			_ = json.Unmarshal(raw, &body)
			out := make([]string, 0, len(body.Items))
			for _, it := range body.Items {
				out = append(out, `{"external_id":"`+it.ExternalID+`","work":null}`)
			}
			_, _ = w.Write([]byte(`{"object":"list","items":[` +
				strings.Join(out, ",") + `],"missing":[]}`))
		case strings.HasSuffix(req.URL.Path, "/catalog/works"):
			_, _ = w.Write([]byte(`{"object":"list","items":[` +
				`{"object":"work","id":"90210","claim":{"site":"kungal","site_work_id":"90210","state":"pending"}}` +
				`],"next_cursor":null}`))
		case req.Method == http.MethodDelete && strings.HasPrefix(req.URL.Path, "/v2/me/claims/"):
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"INVALID_STATE_TRANSITION","detail":"claim is live"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/v2/me/proposals":
			state := "open"
			if r.mergedOnCreate {
				state = "merged"
			}
			_, _ = w.Write([]byte(`{"object":"proposal","id":"1","state":"` + state + `","status":"` + state + `"}`))
		case strings.HasSuffix(req.URL.Path, "/decisions"):
			if r.mergeStatus != 0 {
				w.WriteHeader(r.mergeStatus)
				_, _ = w.Write([]byte(`{"code":"FORBIDDEN","detail":"not yours"}`))
				return
			}
			_, _ = w.Write([]byte(`{"object":"revision","id":"9","seq":2,"action":"merged"}`))
		default:
			_, _ = w.Write([]byte(`{"object":"proposal","id":"1","state":"open"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return NewSubmissionService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL}),
		nil,
	)
}

func TestSubmitAdoptsTheRegistryIssuedID(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "r18", ContentLimit: "nsfw"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if _, present := rec.body["product_work_id"]; present {
		t.Errorf("request carried product_work_id %v — kungal must name no id", rec.body["product_work_id"])
	}
	if rec.submitPath != "/v2/me/claims" {
		t.Errorf("mint hit %q, want POST /v2/me/claims", rec.submitPath)
	}
	if rec.submitAuth != "Bearer user-jwt" {
		t.Errorf("mint auth = %q, want the submitter's bearer and no Basic credential", rec.submitAuth)
	}
	if _, ok := rec.body["site"]; ok {
		t.Errorf("the mint must assert no site: %v", rec.body)
	}
	if _, ok := rec.body["actor"]; ok {
		t.Errorf("the mint must assert no actor: %v", rec.body)
	}
	if res.GID != 90210 {
		t.Errorf("gid = %d, want the registry-issued 90210", res.GID)
	}
	if res.WorkID != 90210 || res.ClaimState != "pending" {
		t.Errorf("result = %+v, want the minted work in pending", res)
	}
}

// The mint body is the whole submission. Asserting only the path let the v2
// rewrite ship a request carrying nothing but display_name, which catalog
// refuses outright — so pin the content the submitter typed, not the envelope.
func TestSubmitSendsTheEditingEngineFieldMap(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt", 0, &SubmissionForm{
		NameJaJP:         "白恋サクラ",
		NameZhCN:         "白恋樱",
		IntroZhCN:        "一个测试简介",
		Aliases:          []string{"SHIROKOI"},
		AgeLimit:         "r18",
		ContentLimit:     "nsfw",
		OriginalLanguage: "ja-jp",
	}); appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.body["display_name"] != "白恋サクラ" {
		t.Errorf("display_name = %v, want the first non-empty title", rec.body["display_name"])
	}
	fields, ok := rec.body["field_values"].(map[string]any)
	if !ok {
		t.Fatalf("no field_values in the mint body: %v", rec.body)
	}
	if fields["catalog.work.display_name"] != "白恋サクラ" ||
		fields["catalog.work.olang"] != "ja" ||
		fields["catalog.work.display_nsfw"] != true ||
		fields["catalog.work.content_rating"] != float64(contentRatingR18) {
		t.Errorf("scalar fields did not survive: %v", fields)
	}
	titles, _ := fields["catalog.work.titles"].([]any)
	if len(titles) != 3 {
		t.Fatalf("titles = %v, want both names and the alias", titles)
	}
	alias, _ := titles[2].(map[string]any)
	if alias["title"] != "SHIROKOI" || alias["kind"] != float64(titleKindAlias) {
		t.Errorf("alias did not survive as an alias title: %v", titles[2])
	}
	intros, _ := fields["catalog.work.intros"].([]any)
	if len(intros) != 1 {
		t.Fatalf("intros = %v, want the one that was filled in", intros)
	}
	if first, _ := intros[0].(map[string]any); first["intro"] != "一个测试简介" || first["lang"] != "zh-Hans" {
		t.Errorf("intro did not survive: %v", intros[0])
	}
}

// A submitter catalog trusts mints straight to live, and the wizard routes on
// this state: announcing "等待审核" for a live entry sends its author to a
// review list that will never contain it.
func TestSubmitReportsATrustedMintAsLive(t *testing.T) {
	rec := &submitRecorder{claimState: "live"}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "all", ContentLimit: "sfw"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}
	if res.ClaimState != "live" {
		t.Errorf("claim_state = %q, want live for a trusted mint", res.ClaimState)
	}
}

// Catalog's English refusal names confirm_duplicates, a field the wizard has no
// control for. It has to reach the client as its own code so the client can ask
// and resend instead of dead-ending the submitter.
func TestSubmitSurfacesDuplicateSuspectsAsItsOwnCode(t *testing.T) {
	rec := &submitRecorder{claimStatus: http.StatusConflict}
	svc := rec.service(t)

	_, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "all", ContentLimit: "sfw"})
	if appErr == nil {
		t.Fatal("a duplicate-suspect mint must not report success")
	}
	if appErr.Code != errors.CodeDuplicateSuspects {
		t.Errorf("code = %d, want %d so the client can offer the resend",
			appErr.Code, errors.CodeDuplicateSuspects)
	}
	if strings.Contains(appErr.Message, "confirm_duplicates") {
		t.Errorf("message leaks the upstream field name: %q", appErr.Message)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if _, sent := rec.body["confirm_duplicates"]; sent {
		t.Errorf("the first attempt must not confirm anything: %v", rec.body)
	}
}

func TestSubmitConfirmsDuplicatesOnlyWhenAsked(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt", 0, &SubmissionForm{
		NameJaJP: "白恋サクラ", AgeLimit: "all", ContentLimit: "sfw",
		ConfirmDuplicates: true,
	}); appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.body["confirm_duplicates"] != true {
		t.Errorf("confirm_duplicates did not reach catalog: %v", rec.body)
	}
}

func TestSubmitAttachesTheBannerAsAFollowUpEdit(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "x", AgeLimit: "all", ContentLimit: "sfw", BannerHash: "abc123"}); appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.editBody == nil {
		t.Fatal("no follow-up edit filed for the submitted banner")
	}
	if rec.editBody["entity_type"] != "catalog.work" {
		t.Errorf("entity_type = %v, want catalog.work", rec.editBody["entity_type"])
	}
	if rec.editBody["entity_id"] != "90210" {
		t.Errorf("entity_id = %v, want the registry work id 90210", rec.editBody["entity_id"])
	}
	if rec.editPath != "/v2/me/proposals" {
		t.Errorf("banner edit hit %q, want the user plane", rec.editPath)
	}
	if rec.editAuth != "Bearer user-jwt" {
		t.Errorf("banner edit auth = %q, want the submitter's bearer", rec.editAuth)
	}
	if _, ok := rec.editBody["actor"]; ok {
		t.Errorf("the banner edit must assert no actor: %v", rec.editBody)
	}
	if _, ok := rec.editBody["site"]; ok {
		t.Errorf("the banner edit must assert no site: %v", rec.editBody)
	}
	if rec.mergePath != "/v2/moderation/proposals/1/decisions" {
		t.Errorf("open banner proposal decided at %q, want the v2 decision face — "+
			"approving the claim never touches an open proposal, so an unmerged "+
			"banner is a banner nobody ever sees", rec.mergePath)
	}
	if rec.mergeAuth != "Bearer user-jwt" {
		t.Errorf("banner merge auth = %q, want the submitter's bearer", rec.mergeAuth)
	}
}

func TestSubmitLeavesAnAlreadyMergedBannerAlone(t *testing.T) {
	rec := &submitRecorder{mergedOnCreate: true}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "x", AgeLimit: "all", ContentLimit: "sfw", BannerHash: "abc123"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}
	if !res.BannerAttached {
		t.Error("an auto-merged banner is attached")
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.mergePath != "" {
		t.Errorf("decided an already-merged proposal at %q", rec.mergePath)
	}
}

func TestSubmitReportsABannerItCouldNotMerge(t *testing.T) {
	rec := &submitRecorder{mergeStatus: http.StatusForbidden}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "x", AgeLimit: "all", ContentLimit: "sfw", BannerHash: "abc123"})
	if appErr != nil {
		t.Fatalf("Submit: %v — a lost banner does not undo the submission", appErr)
	}
	if res.BannerAttached {
		t.Error("banner_attached = true after a refused merge; silence here is what " +
			"hid a broken cover patch for the whole life of the feature")
	}
}

func TestSubmittedEntryIsReachableByItsOwnID(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "all", ContentLimit: "sfw"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	workID, appErr := svc.workIDOf(t.Context(), res.GID)
	if appErr != nil {
		t.Fatalf("resolving the freshly issued gid %d: %v", res.GID, appErr)
	}
	if workID != res.WorkID {
		t.Errorf("gid %d resolved to work %d, want %d", res.GID, workID, res.WorkID)
	}
}

func TestSubmitRefusesATitlelessForm(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt", 0,
		&SubmissionForm{AgeLimit: "all"}); appErr == nil {
		t.Fatal("want a refusal for a form with no title")
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.body != nil {
		t.Error("local validation must not reach the registry")
	}
}

// The local delete cascades (galgame_resource) and can be refused outright
// (galgame_rating is ON DELETE RESTRICT), so it must never run before catalog —
// the authority on owner and draft-only — has agreed. A nil repo here is the
// assertion: reaching it at all panics.
func TestDeleteDraftStopsAtAnUpstreamRefusal(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	appErr := svc.DeleteDraft(t.Context(), "user-jwt", 90210)
	if appErr == nil {
		t.Fatal("want the upstream refusal surfaced; the stub answers no claim delete")
	}
}
