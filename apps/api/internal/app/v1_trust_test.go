package app

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/trustclient"
)

func reportBody(overrides map[string]any) map[string]any {
	body := map[string]any{
		"subject_kind": "forum_topic",
		"subject_id":   "4121",
		"reason_key":   "spam",
		"note":         "it is spam",
		"snapshot":     "buy now",
		"subject_url":  "https://www.kungal.com/topic/4121",
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

func TestV1ReportReasons(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/report-reasons", "/report-reasons", "", nil, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items %+v", items)
	}
	first, _ := items[0].(map[string]any)
	if first["object"] != "report_reason" || first["key"] != "abuse" || first["display_name"] != "辱骂/骚扰" {
		t.Errorf("first reason %+v", first)
	}
	if _, has := first["severity"]; has {
		t.Error("severity is not part of the reason")
	}
}

func TestV1ReportReasonsUpstreamDown(t *testing.T) {
	f := newTrustFix(t)
	f.trust.failReasons.Store(true)
	resp, body := f.call(t, http.MethodGet, "/api/v1/report-reasons", "/report-reasons", "", nil, nil)
	wantProblem(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1CreateReport(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(nil))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	got := f.trust.submissions()
	if len(got) != 1 {
		t.Fatalf("submissions %+v", got)
	}
	s := got[0]
	if s.ReporterID != tsUserPlain || s.SubjectKind != "forum_topic" || s.SubjectID != "4121" || s.ReasonKey != "spam" ||
		s.Note != "it is spam" || s.Snapshot != "buy now" || s.SubjectURL != "https://www.kungal.com/topic/4121" {
		t.Errorf("upstream got %+v", s)
	}

	resp, body = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil,
		reportBody(map[string]any{"note": nil, "snapshot": nil, "subject_url": nil}))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("nulls: %d %+v", resp.StatusCode, body)
	}
	if last := f.trust.submissions()[1]; last.Note != "" || last.SubjectURL != "" {
		t.Errorf("nulls forwarded as %+v", last)
	}
}

func TestV1CreateReportRejects(t *testing.T) {
	f := newTrustFix(t)
	cases := []struct {
		name    string
		session string
		body    map[string]any
		status  int
		code    string
		pointer string
		reason  string
	}{
		{"anonymous", "", reportBody(nil), 401, "MISSING_CREDENTIAL", "", ""},
		{"unregistered kind", "sess-plain", reportBody(map[string]any{"subject_kind": "community_post"}), 422, "VALIDATION_FAILED", "/subject_kind", "UNKNOWN_VALUE"},
		{"unknown reason", "sess-plain", reportBody(map[string]any{"reason_key": "rude"}), 422, "VALIDATION_FAILED", "/reason_key", "UNKNOWN_VALUE"},
		{"foreign link", "sess-plain", reportBody(map[string]any{"subject_url": "https://evil.example/topic/1"}), 422, "VALIDATION_FAILED", "/subject_url", "NOT_ALLOWED_VALUE"},
		{"look-alike host", "sess-plain", reportBody(map[string]any{"subject_url": "https://www.kungal.com.evil.example/topic/1"}), 422, "VALIDATION_FAILED", "/subject_url", "NOT_ALLOWED_VALUE"},
		{"note too long", "sess-plain", reportBody(map[string]any{"note": strings.Repeat("x", 1001)}), 422, "VALIDATION_FAILED", "/note", "TOO_LONG"},
		{"subject id not decimal", "sess-plain", reportBody(map[string]any{"subject_id": "abc"}), 422, "VALIDATION_FAILED", "/subject_id", "INVALID_FORMAT"},
		{"missing reason", "sess-plain", reportBody(map[string]any{"reason_key": nil}), 422, "VALIDATION_FAILED", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", c.session, nil, c.body)
			wantProblem(t, resp, body, c.status, c.code)
			if c.pointer != "" {
				wantField(t, body, c.pointer, c.reason)
			}
		})
	}
	if got := f.trust.submissions(); len(got) != 0 {
		t.Errorf("rejected reports reached the trust service: %+v", got)
	}
}

func TestV1CreateReportBanned(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-banned", nil, reportBody(nil))
	wantProblem(t, resp, body, http.StatusForbidden, "ACCOUNT_BANNED")

	f.failOA.Store(true)
	resp, body = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(nil))
	wantProblem(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	if got := f.trust.submissions(); len(got) != 0 {
		t.Errorf("reports reached the trust service: %+v", got)
	}
}

func TestV1CreateReportUpstream(t *testing.T) {
	f := newTrustFix(t)
	f.trust.failSubmit.Store(http.StatusTooManyRequests)
	resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(nil))
	wantProblem(t, resp, body, http.StatusTooManyRequests, "RATE_LIMITED")

	f.trust.failSubmit.Store(http.StatusUnprocessableEntity)
	resp, body = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(nil))
	wantProblem(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")

	f2 := newTrustFix(t)
	f2.trust.failReasons.Store(true)
	resp, body = f2.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", nil, reportBody(nil))
	wantProblem(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1CreateReportIdempotency(t *testing.T) {
	f := newTrustFix(t)
	key := http.Header{"Idempotency-Key": {"0192f3a4-5b6c-7d8e-9f00-112233445566"}}
	resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", key, reportBody(nil))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	resp, _ = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", key, reportBody(nil))
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay %d %v", resp.StatusCode, resp.Header)
	}
	if got := f.trust.submissions(); len(got) != 1 {
		t.Errorf("a replay reached the trust service: %d submissions", len(got))
	}
	resp, body = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", key, reportBody(map[string]any{"subject_id": "4122"}))
	wantProblem(t, resp, body, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")

	resp, body = f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain", http.Header{"Idempotency-Key": {"nope"}}, reportBody(nil))
	wantProblem(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
}

func TestV1CreateReportMediaType(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.call(t, http.MethodPost, "/api/v1/reports", "/reports", "sess-plain",
		http.Header{"Content-Type": {"text/plain"}}, reportBody(nil))
	wantProblem(t, resp, body, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
}

func (f *trustFix) inbox(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, "/api/v1/admin/review-items?"+q.Encode(), "/admin/review-items", session, nil, nil)
}

func (f *trustFix) siteIDs(state int16) []string {
	f.trust.mu.Lock()
	defer f.trust.mu.Unlock()
	var items []trustclient.ReviewItem
	for _, it := range f.trust.items {
		if it.Site == tsSite && (state < 0 || it.Status == state) {
			items = append(items, it)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].ID > items[j].ID
	})
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = fmt.Sprint(it.ID)
	}
	return ids
}

func TestV1ReviewInboxWalk(t *testing.T) {
	f := newTrustFix(t)
	want := f.siteIDs(-1)
	for _, limit := range []int{2, 3} {
		var got []string
		for page := 1; page <= 20; page++ {
			resp, body := f.inbox(t, "sess-mod", url.Values{"page": {fmt.Sprint(page)}, "limit": {fmt.Sprint(limit)}})
			if resp.StatusCode != http.StatusOK || body["object"] != "list" {
				t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
			}
			if asInt(body["total"]) != len(want) || body["total_relation"] != "eq" {
				t.Fatalf("total %v %v, want %d eq", body["total"], body["total_relation"], len(want))
			}
			ids := adminItemIDs(body)
			if len(ids) == 0 {
				break
			}
			got = append(got, ids...)
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
	for _, c := range f.trust.callsMatching("GET /api/v1/admin/trust/review-items?") {
		if !strings.Contains(c, "site=kungal") {
			t.Errorf("inbox call without the site: %s", c)
		}
	}
}

func TestV1ReviewInboxShape(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.inbox(t, "sess-mod", url.Values{"limit": {"100"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	byID := map[string]map[string]any{}
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		byID[strID(m["id"])] = m
	}
	pending := byID["9001"]
	if pending["state"] != "pending" || pending["opened_by"] != "reports" || pending["claimant"] != nil ||
		asInt(pending["severity"]) != 2 || pending["subject_kind"] != "forum_topic" || pending["subject_id"] != "9001" {
		t.Errorf("pending %+v", pending)
	}
	if _, has := pending["reports"]; has {
		t.Error("the inbox carries no reports")
	}
	claimed := byID["9009"]
	claimant, _ := claimed["claimant"].(map[string]any)
	if claimed["state"] != "claimed" || claimant["id"] != fmt.Sprint(tsUserMod) || claimant["name"] != "mod" || claimed["decider"] != nil {
		t.Errorf("claimed %+v", claimed)
	}
	if forward := byID["9010"]; forward["state"] != "actioned" || forward["opened_by"] != "community_forward" || forward["context_note"] != "flagged excerpt" || forward["severity"] != nil {
		t.Errorf("forward %+v", forward)
	}
	if odd := byID["9012"]; odd["opened_by"] != "unknown" || odd["state"] != "dismissed" {
		t.Errorf("unknown origin %+v", odd)
	}
	for id := range byID {
		if id == "9013" || id == "9014" {
			t.Errorf("another site's item %s leaked into the inbox", id)
		}
	}
}

func TestV1ReviewInboxStateFilter(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.inbox(t, "sess-mod", url.Values{"state": {"dismissed"}, "limit": {"100"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	want := f.siteIDs(3)
	if fmt.Sprint(adminItemIDs(body)) != fmt.Sprint(want) || asInt(body["total"]) != len(want) {
		t.Errorf("dismissed %v total %v, want %v", adminItemIDs(body), body["total"], want)
	}
	resp, body = f.inbox(t, "sess-mod", url.Values{"state": {"open"}})
	wantProblem(t, resp, body, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE")
}

func TestV1ReviewInboxRejects(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.inbox(t, "", nil)
	wantProblem(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	resp, body = f.inbox(t, "sess-plain", nil)
	wantProblem(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
	resp, body = f.inbox(t, "sess-mod", url.Values{"page": {"101"}, "limit": {"100"}})
	wantProblem(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	wantField(t, body, "page", "OUT_OF_RANGE")
	resp, body = f.inbox(t, "sess-mod", url.Values{"limit": {"101"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("limit 101: %d %+v", resp.StatusCode, body)
	}
	if calls := f.trust.callsMatching("GET /api/v1/admin/trust/"); len(calls) != 0 {
		t.Errorf("refused requests reached the trust service: %v", calls)
	}
}

func TestV1ReviewBearerNeverReviews(t *testing.T) {
	f := newTrustFix(t)
	perm.SetUserOverrides(map[int][]perm.Override{tsUserGrantee: {{Permission: perm.TrustReview, Effect: perm.EffectGrant}}})
	t.Cleanup(func() { perm.SetUserOverrides(nil) })
	if !perm.CanUser(tsUserGrantee, []string{"user"}, perm.TrustReview) {
		t.Fatal("the personal grant did not take")
	}
	for _, token := range []string{"grantee-bearer", "mod-bearer"} {
		hdr := http.Header{"Authorization": {"Bearer " + token}}
		resp, body := f.call(t, http.MethodGet, "/api/v1/admin/review-items", "/admin/review-items", "", hdr, nil)
		wantProblem(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
		resp, body = f.call(t, http.MethodPatch, "/api/v1/admin/review-items/9001", tsReviewPath, "", hdr, map[string]any{"state": "claimed"})
		wantProblem(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
	}
}

func TestV1ReviewInboxUpstream(t *testing.T) {
	for _, c := range []struct {
		upstream int
		status   int
		code     string
	}{
		{http.StatusUnauthorized, 401, "INVALID_CREDENTIAL"},
		{http.StatusForbidden, 403, "PERMISSION_REQUIRED"},
		{http.StatusInternalServerError, 503, "SERVICE_UNAVAILABLE"},
		{http.StatusBadRequest, 503, "SERVICE_UNAVAILABLE"},
	} {
		f := newTrustFix(t)
		f.trust.failAdmin.Store(int32(c.upstream))
		resp, body := f.inbox(t, "sess-mod", nil)
		wantProblem(t, resp, body, c.status, c.code)
	}
	f := newTrustFix(t)
	f.failOA.Store(true)
	resp, body := f.inbox(t, "sess-mod", nil)
	wantProblem(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func (f *trustFix) reviewItem(t *testing.T, session, id string) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, "/api/v1/admin/review-items/"+id, tsReviewPath, session, nil, nil)
}

func TestV1ReviewItemDetail(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.reviewItem(t, "sess-mod", "9001")
	if resp.StatusCode != http.StatusOK || body["object"] != "review_item" || body["id"] != "9001" {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	reports, _ := body["reports"].([]any)
	if len(reports) != 2 {
		t.Fatalf("reports %+v", reports)
	}
	first, _ := reports[0].(map[string]any)
	reporter, _ := first["reporter"].(map[string]any)
	reason, _ := first["report_reason"].(map[string]any)
	if first["object"] != "report" || first["id"] != "71" || reporter["id"] != fmt.Sprint(tsReporterA) || reporter["name"] != "reporter-a" ||
		reason["key"] != "spam" || reason["display_name"] != "垃圾信息" ||
		first["snapshot"] != "the content" || first["subject_url"] != "https://www.kungal.com/topic/9001" {
		t.Errorf("first report %+v", first)
	}
	second, _ := reports[1].(map[string]any)
	gone, _ := second["reporter"].(map[string]any)
	if second["report_reason"] != nil || second["subject_url"] != nil || gone["name"] != nil || gone["id"] != fmt.Sprint(tsReporterB) {
		t.Errorf("second report %+v", second)
	}

	resp, body = f.reviewItem(t, "sess-mod", "9010")
	if got, _ := body["reports"].([]any); resp.StatusCode != http.StatusOK || got == nil || len(got) != 0 {
		t.Errorf("item without reports %d %+v", resp.StatusCode, body)
	}
}

func TestV1ReviewItemMissing(t *testing.T) {
	f := newTrustFix(t)
	for _, id := range []string{"8999", "9013"} {
		resp, body := f.reviewItem(t, "sess-mod", id)
		wantProblem(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	}
	resp, body := f.reviewItem(t, "sess-plain", "9001")
	wantProblem(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
	resp, body = f.reviewItem(t, "sess-mod", "abc")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("malformed id %d %+v", resp.StatusCode, body)
	}
}

func (f *trustFix) patchItem(t *testing.T, session, id string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodPatch, "/api/v1/admin/review-items/"+id, tsReviewPath, session, nil, payload)
}

func TestV1ReviewItemClaim(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.patchItem(t, "sess-mod", "9001", map[string]any{"state": "claimed"})
	claimant, _ := body["claimant"].(map[string]any)
	if resp.StatusCode != http.StatusOK || body["state"] != "claimed" || claimant["id"] != fmt.Sprint(tsUserMod) || body["claimed_at"] == nil {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if _, has := body["reports"]; !has {
		t.Error("the patch answers with the full item")
	}
	resp, body = f.patchItem(t, "sess-mod", "9001", map[string]any{"state": "claimed"})
	wantProblem(t, resp, body, http.StatusConflict, "INVALID_STATE_TRANSITION")
	if !strings.Contains(fmt.Sprint(body["detail"]), "claimed") {
		t.Errorf("detail does not name the current state: %v", body["detail"])
	}
	resp, body = f.patchItem(t, "sess-mod", "9011", map[string]any{"state": "dismissed"})
	wantProblem(t, resp, body, http.StatusConflict, "INVALID_STATE_TRANSITION")
	resp, body = f.patchItem(t, "sess-mod", "8999", map[string]any{"state": "claimed"})
	wantProblem(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1ReviewItemDecide(t *testing.T) {
	f := newTrustFix(t)
	resp, body := f.patchItem(t, "sess-mod", "9002", map[string]any{
		"state": "actioned", "action": "hide", "reason_code": "spam", "statement": "spam",
	})
	if resp.StatusCode != http.StatusOK || body["state"] != "actioned" || body["decider"] == nil {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if it := f.trust.item(9002); it.Status != 2 || it.DecidedBy == nil || *it.DecidedBy != tsUserMod {
		t.Errorf("upstream item %+v", it)
	}
	resp, body = f.patchItem(t, "sess-mod", "9003", map[string]any{"state": "dismissed"})
	if resp.StatusCode != http.StatusOK || body["state"] != "dismissed" {
		t.Fatalf("dismiss %d %+v", resp.StatusCode, body)
	}
}

func TestV1ReviewItemDecideRejects(t *testing.T) {
	f := newTrustFix(t)
	cases := []struct {
		name     string
		body     map[string]any
		status   int
		code     string
		pointers map[string]string
	}{
		{"actioned without action", map[string]any{"state": "actioned", "reason_code": "spam"}, 422, "VALIDATION_FAILED", map[string]string{"/action": "REQUIRED"}},
		{"actioned without reason", map[string]any{"state": "actioned", "action": "hide"}, 422, "VALIDATION_FAILED", map[string]string{"/reason_code": "REQUIRED"}},
		{"dismissed with action", map[string]any{"state": "dismissed", "action": "hide"}, 422, "VALIDATION_FAILED", map[string]string{"/action": "INCONSISTENT_WITH"}},
		{"claimed with extras", map[string]any{"state": "claimed", "reason_code": "spam", "statement": "x"}, 422, "VALIDATION_FAILED", map[string]string{"/reason_code": "INCONSISTENT_WITH", "/statement": "INCONSISTENT_WITH"}},
		{"unknown action", map[string]any{"state": "actioned", "action": "ban", "reason_code": "spam"}, 422, "VALIDATION_FAILED", map[string]string{"/action": "UNKNOWN_VALUE"}},
		{"reopen", map[string]any{"state": "pending"}, 422, "VALIDATION_FAILED", map[string]string{"/state": "UNKNOWN_VALUE"}},
		{"reason code shape", map[string]any{"state": "actioned", "action": "hide", "reason_code": "Spam!"}, 422, "VALIDATION_FAILED", map[string]string{"/reason_code": "INVALID_FORMAT"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := f.patchItem(t, "sess-mod", "9004", c.body)
			wantProblem(t, resp, body, c.status, c.code)
			for p, r := range c.pointers {
				wantField(t, body, p, r)
			}
		})
	}
	if calls := f.trust.callsMatching("POST "); len(calls) != 0 {
		t.Errorf("rejected patches reached the trust service: %v", calls)
	}
	resp, body := f.call(t, http.MethodPatch, "/api/v1/admin/review-items/9004", tsReviewPath, "sess-mod",
		http.Header{"Content-Type": {"text/plain"}}, map[string]any{"state": "claimed"})
	wantProblem(t, resp, body, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
}
