package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1EditErrorMatrix(t *testing.T) {
	f := newG7bFix(t)
	w := g7bWorkPath(g7bWork)
	merged := g7bWorkPath(g7bWorkMerged)
	prop := g7bProposalPath(f.pOpenAlice)
	patch := map[string]any{"patch": map[string]any{"catalog.work.display_name": "X"}}
	amend := map[string]any{"set": map[string]any{"catalog.work.display_name": "X"}}
	type call struct {
		name, method, url, spec, session, bearer, idem string
		body                                           any
		status                                         int
		code                                           string
	}
	cases := []call{
		{"create banned", "POST", w + "/edit-proposals", "/works/{work_id}/edit-proposals", g7bSessBanned, "", keyUUID(8001), patch, 403, problem.CodeAccountBanned},
		{"revert banned", "POST", w + "/edit-reverts", "/works/{work_id}/edit-reverts", g7bSessBanned, "", keyUUID(8002), map[string]any{"to_seq": 1}, 403, problem.CodeAccountBanned},
		{"amend banned", "POST", prop + "/amendments", "/edit-proposals/{proposal_id}/amendments", g7bSessBanned, "", keyUUID(8003), amend, 403, problem.CodeAccountBanned},
		{"patch banned", "PATCH", prop, "/edit-proposals/{proposal_id}", g7bSessBanned, "", "", map[string]any{"state": "withdrawn"}, 403, problem.CodeAccountBanned},
		{"get banned", "GET", prop, "/edit-proposals/{proposal_id}", g7bSessBanned, "", "", nil, 403, problem.CodeAccountBanned},
		{"mine banned", "GET", "/api/v1/me/edit-proposals", "/me/edit-proposals", g7bSessBanned, "", "", nil, 403, problem.CodeAccountBanned},
		{"queue banned", "GET", "/api/v1/edit-proposals", "/edit-proposals", g7bSessBanned, "", "", nil, 403, problem.CodeAccountBanned},

		{"revert scope", "POST", w + "/edit-reverts", "/works/{work_id}/edit-reverts", g7bSessNoscope, "", keyUUID(8011), map[string]any{"to_seq": 1}, 403, problem.CodeScopeRequired},
		{"mine scope", "GET", "/api/v1/me/edit-proposals", "/me/edit-proposals", g7bSessNoscope, "", "", nil, 403, problem.CodeScopeRequired},
		{"get scope", "GET", prop, "/edit-proposals/{proposal_id}", g7bSessNoscope, "", "", nil, 403, problem.CodeScopeRequired},
		{"amend scope", "POST", prop + "/amendments", "/edit-proposals/{proposal_id}/amendments", g7bSessNoscope, "", keyUUID(8012), amend, 403, problem.CodeScopeRequired},
		{"withdraw scope", "PATCH", prop, "/edit-proposals/{proposal_id}", g7bSessNoscope, "", "", map[string]any{"state": "withdrawn"}, 403, problem.CodeScopeRequired},

		{"mine quota", "GET", "/api/v1/me/edit-proposals", "/me/edit-proposals", g7bSessQuota, "", "", nil, 503, problem.CodeServiceUnavailable},
		{"get quota", "GET", prop, "/edit-proposals/{proposal_id}", g7bSessQuota, "", "", nil, 503, problem.CodeServiceUnavailable},
		{"amend quota", "POST", prop + "/amendments", "/edit-proposals/{proposal_id}/amendments", g7bSessQuota, "", keyUUID(8021), amend, 503, problem.CodeServiceUnavailable},
		{"revert quota", "POST", w + "/edit-reverts", "/works/{work_id}/edit-reverts", g7bSessQuota, "", keyUUID(8022), map[string]any{"to_seq": 1}, 503, problem.CodeServiceUnavailable},

		{"list bad bearer", "GET", w + "/edit-proposals", "/works/{work_id}/edit-proposals", "", "no-such", "", nil, 401, problem.CodeInvalidCredential},
		{"revisions bad bearer", "GET", w + "/edit-revisions", "/works/{work_id}/edit-revisions", "", "no-such", "", nil, 401, problem.CodeInvalidCredential},
		{"diff bad bearer", "GET", w + "/edit-revisions/diff?from_seq=1&to_seq=2", "/works/{work_id}/edit-revisions/diff", "", "no-such", "", nil, 401, problem.CodeInvalidCredential},

		{"list merged", "GET", merged + "/edit-proposals", "/works/{work_id}/edit-proposals", "", "", "", nil, 404, problem.CodeEntityMerged},
		{"revisions merged", "GET", merged + "/edit-revisions", "/works/{work_id}/edit-revisions", "", "", "", nil, 404, problem.CodeEntityMerged},
		{"diff merged", "GET", merged + "/edit-revisions/diff?from_seq=1&to_seq=2", "/works/{work_id}/edit-revisions/diff", "", "", "", nil, 404, problem.CodeEntityMerged},
		{"revert merged", "POST", merged + "/edit-reverts", "/works/{work_id}/edit-reverts", g7bSessAlice, "", keyUUID(8031), map[string]any{"to_seq": 1}, 404, problem.CodeEntityMerged},

		{"list limit", "GET", w + "/edit-proposals?limit=101", "/works/{work_id}/edit-proposals", "", "", "", nil, 400, problem.CodeLimitTooLarge},
		{"mine limit", "GET", "/api/v1/me/edit-proposals?limit=101", "/me/edit-proposals", g7bSessAlice, "", "", nil, 400, problem.CodeLimitTooLarge},
		{"mine cursor", "GET", "/api/v1/me/edit-proposals?cursor=cur_bm9wZQ", "/me/edit-proposals", g7bSessAlice, "", "", nil, 400, problem.CodeInvalidCursor},
		{"queue state", "GET", "/api/v1/edit-proposals?state=pending", "/edit-proposals", g7bSessStaff, "", "", nil, 400, problem.CodeUnknownEnumValue},
		{"queue limit", "GET", "/api/v1/edit-proposals?limit=101", "/edit-proposals", g7bSessStaff, "", "", nil, 400, problem.CodeLimitTooLarge},
		{"revisions depth", "GET", w + "/edit-revisions?page=101&limit=100", "/works/{work_id}/edit-revisions", "", "", "", nil, 400, problem.CodeInvalidParameter},
		{"amend unknown", "POST", g7bProposalPath(957199999) + "/amendments", "/edit-proposals/{proposal_id}/amendments", g7bSessStaff, "", keyUUID(8041), amend, 404, problem.CodeNotFound},
		{"patch unknown", "PATCH", g7bProposalPath(957199999), "/edit-proposals/{proposal_id}", g7bSessStaff, "", "", map[string]any{"state": "merged"}, 404, problem.CodeNotFound},
		{"mine anonymous", "GET", "/api/v1/me/edit-proposals", "/me/edit-proposals", "", "", "", nil, 401, problem.CodeMissingCredential},
		{"queue no token", "GET", "/api/v1/edit-proposals", "/edit-proposals", g7bSessNoToken, "", "", nil, 403, problem.CodePermissionRequired},
	}
	for _, c := range cases {
		var hdr http.Header
		if c.bearer != "" {
			hdr = http.Header{"Authorization": {"Bearer " + c.bearer}}
		}
		resp, body := f.callHdr(t, c.method, c.url, c.spec, c.session, c.idem, hdr, c.body)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %v, want %d %s", c.name, resp.StatusCode, body["code"], c.status, c.code)
		}
		if c.code == problem.CodeInvalidCredential && resp.Header.Get("WWW-Authenticate") == "" {
			t.Errorf("%s: 401 without WWW-Authenticate", c.name)
		}
	}

	f.up.failApp = true
	for _, c := range []struct{ url, spec, session string }{
		{w + "/edit-proposals", "/works/{work_id}/edit-proposals", ""},
		{w + "/edit-revisions", "/works/{work_id}/edit-revisions", ""},
		{w + "/edit-revisions/diff?from_seq=1&to_seq=2", "/works/{work_id}/edit-revisions/diff", ""},
		{"/api/v1/edit-proposals?state=merged", "/edit-proposals", g7bSessStaff},
		{prop, "/edit-proposals/{proposal_id}", g7bSessAlice},
	} {
		resp, body := f.call(t, http.MethodGet, c.url, c.spec, c.session, "", nil)
		want := http.StatusServiceUnavailable
		if c.spec == "/edit-proposals/{proposal_id}" {
			want = http.StatusOK
		}
		if resp.StatusCode != want {
			t.Errorf("app face down on %s: %d %v", c.url, resp.StatusCode, body["code"])
		}
	}
	resp, body := f.call(t, http.MethodPatch, prop, "/edit-proposals/{proposal_id}", g7bSessStaff, "", map[string]any{"state": "merged"})
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	f.up.failApp = false

	f.up.denyRevert = true
	resp, body = f.call(t, http.MethodPost, w+"/edit-reverts", "/works/{work_id}/edit-reverts", g7bSessAlice, keyUUID(8051), map[string]any{"to_seq": 1})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	f.up.denyRevert = false

	body1 := map[string]any{"patch": map[string]any{"catalog.work.display_name": "One"}}
	body2 := map[string]any{"patch": map[string]any{"catalog.work.display_name": "Two"}}
	resp, _ = f.call(t, http.MethodPost, w+"/edit-proposals", "/works/{work_id}/edit-proposals", g7bSessAlice, keyUUID(8061), body1)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first create %d", resp.StatusCode)
	}
	resp, body = f.call(t, http.MethodPost, w+"/edit-proposals", "/works/{work_id}/edit-proposals", g7bSessAlice, keyUUID(8061), body2)
	wantCode(t, resp, body, http.StatusConflict, problem.CodeIdempotencyKeyReused)
}
