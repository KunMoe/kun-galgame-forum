package app

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/pkg/problem"
)

func (f *g7bFix) messages(t *testing.T, kind string, receiver int) []string {
	t.Helper()
	var out []string
	if err := f.db.Raw(`SELECT content FROM message WHERE type = ? AND receiver_id = ? AND link LIKE '/galgame/957000%' ORDER BY id`,
		kind, receiver).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}

func TestV1CreateWorkEditProposal(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-proposals"
	path := g7bWorkPath(g7bWork) + "/edit-proposals"
	payload := map[string]any{"patch": map[string]any{"catalog.work.display_name": "New Name"}, "note": "typo"}
	before := f.up.proposalCount()

	resp, body := f.call(t, http.MethodPost, path, spec, g7bSessAlice, "", payload)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
	if n := len(f.up.callsTo(http.MethodPost, "/v2/")); n != 0 {
		t.Fatalf("a keyless create reached catalog %d times", n)
	}

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessAlice, keyUUID(7001), payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if resp.Header.Get("Location") != "/api/v1/edit-proposals/"+id {
		t.Errorf("Location %q", resp.Header.Get("Location"))
	}
	if body["state"] != "open" || body["revision"] != nil || strID(body["work_id"]) != strconv.Itoa(g7bWork) {
		t.Errorf("created %+v", body)
	}
	if n := body["note"]; n != "typo" {
		t.Errorf("note %+v", n)
	}
	posts := f.up.callsTo(http.MethodPost, "/v2/me/proposals")
	if len(posts) != 1 || posts[0].idemKey != keyUUID(7001) || posts[0].auth != "g7b-alice" {
		t.Errorf("upstream create %+v", posts)
	}
	if n := f.scalar(t, `SELECT count(*) FROM galgame_activity WHERE wiki_pr_id = ? AND work_id = ? AND user_id = ? AND type = 'GALGAME_PR_CREATION'`,
		id, g7bWork, w3UserAlice); n != 1 {
		t.Errorf("activity rows %d", n)
	}
	if got := f.messages(t, "requested", w3UserBob); len(got) != 1 || got[0] != "EditLive（中）" {
		t.Errorf("owner notification %v", got)
	}

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessAlice, keyUUID(7001), payload)
	if resp.StatusCode != http.StatusCreated || strID(body["id"]) != id {
		t.Errorf("replay %d %+v", resp.StatusCode, body)
	}
	if got := f.up.proposalCount(); got != before+1 {
		t.Errorf("a replayed create filed again: %d → %d", before, got)
	}

	for i, c := range []struct {
		name, session, pointer, code string
		status                       int
		body                         map[string]any
	}{
		{"empty patch", g7bSessAlice, "/patch", problem.CodeValidationFailed, 422, map[string]any{"patch": map[string]any{}}},
		{"foreign key", g7bSessAlice, "/patch/catalog.tag.intros", problem.CodeValidationFailed, 422,
			map[string]any{"patch": map[string]any{"catalog.tag.intros": []any{}}}},
		{"catalog refuses", g7bSessAlice, "/patch/catalog.work.titles", problem.CodeValidationFailed, 422,
			map[string]any{"patch": map[string]any{"catalog.work.titles": "not a list"}}},
		{"scope", g7bSessNoscope, "", problem.CodeScopeRequired, 403, payload},
		{"quota", g7bSessQuota, "", problem.CodeServiceUnavailable, 503, payload},
	} {
		resp, body := f.call(t, http.MethodPost, path, spec, c.session, keyUUID(7100+i), c.body)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %v %+v", c.name, resp.StatusCode, body["code"], body)
			continue
		}
		if c.pointer != "" {
			if e := firstError(t, body); e["pointer"] != c.pointer {
				t.Errorf("%s pointer %+v", c.name, e)
			}
		}
		if strings.Contains(strID(body["detail"]), "must be a list") {
			t.Errorf("%s: catalog's own sentence reached detail: %v", c.name, body["detail"])
		}
	}

	for _, c := range []struct {
		id   int
		code string
	}{{g7bWorkHidden, problem.CodeNotFound}, {g7bWorkMerged, problem.CodeEntityMerged}} {
		resp, body := f.call(t, http.MethodPost, g7bWorkPath(c.id)+"/edit-proposals", spec, g7bSessAlice, keyUUID(7200+c.id%10), payload)
		wantCode(t, resp, body, http.StatusNotFound, c.code)
	}
}

func TestV1CreateWorkEditProposalAutoMerge(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-proposals"
	path := g7bWorkPath(g7bWork) + "/edit-proposals"
	payload := map[string]any{"patch": map[string]any{"catalog.work.display_name": "Trusted"}}
	resp, body := f.call(t, http.MethodPost, path, spec, g7bSessTrusted, keyUUID(7301), payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("automerge %d %+v", resp.StatusCode, body)
	}
	rev, _ := body["revision"].(map[string]any)
	if body["state"] != "merged" || rev == nil || rev["proposal_id"] != strID(body["id"]) || rev["revision_action"] != "merged" {
		t.Errorf("automerged %+v", body)
	}
	if n := f.scalar(t, `SELECT count(*) FROM galgame_activity WHERE work_id = ?`, g7bWork); n != 0 {
		t.Errorf("an automerged proposal is not a pending request, activity %d", n)
	}
	if got := f.messages(t, "requested", w3UserBob); len(got) != 0 {
		t.Errorf("owner told about an automerge %v", got)
	}

	f.up.failRevisions = true
	before := f.up.proposalCount()
	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessTrusted, keyUUID(7302), payload)
	if resp.StatusCode != http.StatusCreated || body["state"] != "merged" || body["revision"] != nil {
		t.Errorf("revision lookup failure must still be 201 with revision null: %d %+v", resp.StatusCode, body)
	}
	if got := f.up.proposalCount(); got != before+1 {
		t.Errorf("proposals filed %d", got-before)
	}
}

func TestV1CreateWorkEditRevert(t *testing.T) {
	f := newG7bFix(t)
	spec := "/works/{work_id}/edit-reverts"
	path := g7bWorkPath(g7bWork) + "/edit-reverts"

	resp, body := f.call(t, http.MethodPost, path, spec, g7bSessStaff, "", map[string]any{"to_seq": 1})
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessStaff, keyUUID(7401), map[string]any{"to_seq": 1, "note": "vandalism"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("revert %d %+v", resp.StatusCode, body)
	}
	prop, _ := body["proposal"].(map[string]any)
	rev, _ := body["revision"].(map[string]any)
	if prop["state"] != "merged" || rev == nil || rev["revision_action"] != "reverted" || asInt(rev["seq"]) != 4 {
		t.Errorf("reviewer revert %+v", body)
	}
	if resp.Header.Get("Location") != "/api/v1/edit-proposals/"+strID(prop["id"]) {
		t.Errorf("Location %q", resp.Header.Get("Location"))
	}
	calls := f.up.callsTo(http.MethodPost, "/v2/moderation/reverts")
	if len(calls) != 1 || !strings.Contains(calls[0].body, `"revision_id":"`+g7bID(f.rev1)+`"`) ||
		!strings.Contains(calls[0].body, `"reason":"vandalism"`) || calls[0].idemKey != keyUUID(7401) {
		t.Errorf("upstream revert %+v", calls)
	}

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessAlice, keyUUID(7402), map[string]any{"to_seq": 2})
	prop, _ = body["proposal"].(map[string]any)
	if resp.StatusCode != http.StatusCreated || prop["state"] != "open" || body["revision"] != nil {
		t.Errorf("contributor revert waits for review %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessAlice, keyUUID(7403), map[string]any{"to_seq": 99})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessAlice, keyUUID(7404), map[string]any{"to_seq": 0})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1CreateEditProposalAmendment(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}/amendments"
	path := g7bProposalPath(f.pOpenAlice) + "/amendments"
	payload := map[string]any{"set": map[string]any{"catalog.work.display_name": "Fixed"}, "note": "caps"}

	resp, body := f.call(t, http.MethodPost, path, spec, g7bSessStaff, "", payload)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)

	resp, body = f.call(t, http.MethodPost, path, spec, g7bSessStaff, keyUUID(7501), payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("amend %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "edit_amendment" || asInt(body["seq"]) != 1 || body["note"] != "caps" {
		t.Errorf("amendment %+v", body)
	}
	if a, _ := body["amender"].(map[string]any); a["id"] != strconv.Itoa(w3UserStaff) {
		t.Errorf("amender %+v", body["amender"])
	}
	for _, k := range []string{"set", "unset"} {
		if _, ok := body[k]; ok {
			t.Errorf("receipt echoes %s", k)
		}
	}
	if strID(body["id"]) == g7bID(f.pOpenAlice) {
		t.Error("amendment id is the proposal id: the proposal record was decoded as an amendment")
	}
	if resp.Header.Get("Location") != g7bProposalPath(f.pOpenAlice) || resp.Header.Get("ETag") != f.up.etagOf(f.pOpenAlice) {
		t.Errorf("headers %q %q", resp.Header.Get("Location"), resp.Header.Get("ETag"))
	}
	calls := f.up.callsTo(http.MethodPost, "/v2/me/proposals/")
	if len(calls) != 1 || calls[0].ifMatch != "*" {
		t.Errorf("absent If-Match must reach catalog as * %+v", calls)
	}

	stale := f.up.etagOf(f.pOpenAlice)
	resp, body = f.callHdr(t, http.MethodPost, path, spec, g7bSessStaff, keyUUID(7502), nil, map[string]any{"unset": []string{"catalog.work.display_name"}})
	if resp.StatusCode != http.StatusCreated || asInt(body["seq"]) != 2 {
		t.Fatalf("second amend %d %+v", resp.StatusCode, body)
	}
	f.up.resetCalls()
	resp, body = f.callHdr(t, http.MethodPost, path, spec, g7bSessStaff, keyUUID(7503), http.Header{"If-Match": {stale}}, payload)
	wantCode(t, resp, body, http.StatusPreconditionFailed, problem.CodePreconditionFailed)
	if calls := f.up.callsTo(http.MethodPost, "/v2/me/proposals/"); len(calls) != 1 || calls[0].ifMatch != stale {
		t.Errorf("If-Match must be forwarded as sent %+v", calls)
	}
	current := f.up.etagOf(f.pOpenAlice)
	resp, body = f.callHdr(t, http.MethodPost, path, spec, g7bSessStaff, keyUUID(7504), http.Header{"If-Match": {current}}, payload)
	if resp.StatusCode != http.StatusCreated || asInt(body["seq"]) != 3 {
		t.Errorf("current If-Match %d %+v", resp.StatusCode, body)
	}

	for i, c := range []struct {
		name, session, code string
		status              int
		body                map[string]any
	}{
		{"empty", g7bSessAlice, problem.CodeValidationFailed, 422, map[string]any{"note": "x"}},
		{"foreign key", g7bSessAlice, problem.CodeValidationFailed, 422, map[string]any{"unset": []string{"catalog.tag.intros"}}},
		{"stranger", g7bSessOther, problem.CodePermissionRequired, 403, payload},
	} {
		resp, body := f.call(t, http.MethodPost, path, spec, c.session, keyUUID(7600+i), c.body)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %v %+v", c.name, resp.StatusCode, body["code"], body)
		}
	}
	resp, body = f.call(t, http.MethodPost, g7bProposalPath(f.pMergedBob)+"/amendments", spec, g7bSessStaff, keyUUID(7700), payload)
	wantCode(t, resp, body, http.StatusConflict, problem.CodeInvalidStateTransition)
}

func TestV1MergeEditProposal(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	path := g7bProposalPath(f.pOpenAlice)
	resp, body := f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", map[string]any{"state": "merged"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("merge %d %+v", resp.StatusCode, body)
	}
	if body["state"] != "merged" || body["decided_at"] == nil {
		t.Errorf("merged %+v", body)
	}
	if d, _ := body["decider"].(map[string]any); d["id"] != strconv.Itoa(w3UserStaff) {
		t.Errorf("decider %+v", body["decider"])
	}
	if resp.Header.Get("ETag") != f.up.etagOf(f.pOpenAlice) {
		t.Errorf("ETag %q", resp.Header.Get("ETag"))
	}
	decisions := f.up.callsTo(http.MethodPost, "/v2/moderation/proposals/")
	if len(decisions) != 1 || !strings.Contains(decisions[0].body, `"decision":"merge"`) || decisions[0].ifMatch != "*" {
		t.Errorf("upstream decision %+v", decisions)
	}
	awards := f.snapshotAwards()
	want := awardCall{userID: w3UserAlice, delta: 1, reason: "content_approved",
		ref: "galgame_pr:" + strconv.Itoa(g7bWork), key: "kungal:galgame_edit_merged:" + g7bID(f.pOpenAlice)}
	if len(awards) != 1 || awards[0] != want {
		t.Errorf("awards %+v want %+v", awards, want)
	}
	if got := f.messages(t, "merged", w3UserAlice); len(got) != 1 || got[0] != "EditLive（中）" {
		t.Errorf("merged notification %v", got)
	}
	if n := f.scalar(t, `SELECT count(*) FROM galgame WHERE id = ? AND resource_update_time > '2026-09-01 12:00:00+00'`, g7bWork); n != 1 {
		t.Error("resource_update_time not bumped")
	}

	resp, body = f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", map[string]any{"state": "merged"})
	wantCode(t, resp, body, http.StatusConflict, problem.CodeInvalidStateTransition)
	if !strings.Contains(strID(body["detail"]), "merged") {
		t.Errorf("detail names the current state %+v", body["detail"])
	}

	resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pStaffOwn), spec, g7bSessStaff, "", map[string]any{"state": "merged"})
	if resp.StatusCode != http.StatusOK || body["state"] != "merged" {
		t.Fatalf("self merge %d %+v", resp.StatusCode, body)
	}
	if n := len(f.snapshotAwards()); n != 1 {
		t.Errorf("a merger's own proposal earns nothing, awards %d", n)
	}

	resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pOpenOther), spec, g7bSessBob, "", map[string]any{"state": "merged"})
	if resp.StatusCode != http.StatusOK || body["state"] != "merged" {
		t.Errorf("the work's owner may merge %d %+v", resp.StatusCode, body)
	}
}

func TestV1EditDecisionsNeedStanding(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	path := g7bProposalPath(f.pOpenAlice)
	for _, state := range []string{"merged", "declined"} {
		resp, body := f.bearer(t, http.MethodPatch, path, spec, "staff-token", "", map[string]any{"state": state, "note": "n"})
		wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
		resp, body = f.call(t, http.MethodPatch, path, spec, g7bSessOther, "", map[string]any{"state": state, "note": "n"})
		wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
		resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pPlainAlice), spec, g7bSessBob, "", map[string]any{"state": state, "note": "n"})
		wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	}
	f.denyEditReview(t)
	resp, body := f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", map[string]any{"state": "merged"})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	if n := len(f.up.callsTo(http.MethodPost, "/v2/moderation/")); n != 0 {
		t.Errorf("refused decisions reached catalog %d times", n)
	}
	if got := f.up.proposal(f.pOpenAlice).state; got != "open" {
		t.Errorf("state %s", got)
	}
	resp, body = f.bearer(t, http.MethodGet, path, spec, "bob-token", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1DeclineEditProposal(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	path := g7bProposalPath(f.pOpenAlice)
	for _, note := range []any{nil, "", "   "} {
		payload := map[string]any{"state": "declined"}
		if note != nil {
			payload["note"] = note
		}
		resp, body := f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", payload)
		wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
		if e := firstError(t, body); e["pointer"] != "/note" || e["reason"] != problem.ReasonRequired {
			t.Errorf("note %q: %+v", note, e)
		}
	}
	if n := len(f.up.callsTo(http.MethodPost, "/v2/moderation/")); n != 0 {
		t.Errorf("a blank decline reached catalog %d times", n)
	}
	resp, body := f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", map[string]any{"state": "declined", "note": "duplicate of #12"})
	if resp.StatusCode != http.StatusOK || body["state"] != "declined" {
		t.Fatalf("decline %d %+v", resp.StatusCode, body)
	}
	if got := f.messages(t, "declined", w3UserAlice); len(got) != 1 || got[0] != "EditLive（中）：duplicate of #12" {
		t.Errorf("the decline reason must reach the proposer %v", got)
	}
	if n := len(f.snapshotAwards()); n != 0 {
		t.Errorf("a decline awards nothing, %d", n)
	}
}

func TestV1WithdrawEditProposal(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	path := g7bProposalPath(f.pOpenAlice)
	resp, body := f.call(t, http.MethodPatch, path, spec, g7bSessStaff, "", map[string]any{"state": "withdrawn"})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)

	stale := f.up.etagOf(f.pOpenAlice)
	f.up.mu.Lock()
	f.up.proposals[f.pOpenAlice].updated = f.up.proposals[f.pOpenAlice].updated.Add(time.Second)
	f.up.mu.Unlock()
	resp, body = f.callHdr(t, http.MethodPatch, path, spec, g7bSessAlice, "", http.Header{"If-Match": {stale}}, map[string]any{"state": "withdrawn"})
	wantCode(t, resp, body, http.StatusPreconditionFailed, problem.CodePreconditionFailed)
	if calls := f.up.callsTo(http.MethodPatch, "/v2/me/proposals/"); len(calls) != 1 || calls[0].ifMatch != stale {
		t.Errorf("forwarded If-Match %+v", calls)
	}

	resp, body = f.call(t, http.MethodPatch, path, spec, g7bSessAlice, "", map[string]any{"state": "withdrawn"})
	if resp.StatusCode != http.StatusOK || body["state"] != "withdrawn" || body["decider"] != nil {
		t.Fatalf("withdraw %d %+v", resp.StatusCode, body)
	}
	if calls := f.up.callsTo(http.MethodPatch, "/v2/me/proposals/"); len(calls) != 2 || calls[1].ifMatch != "*" {
		t.Errorf("absent If-Match %+v", calls)
	}
	if v, _ := body["viewer"].(map[string]any); v["can_withdraw"] != false {
		t.Errorf("viewer after withdraw %+v", v)
	}

	resp, body = f.bearer(t, http.MethodPatch, g7bProposalPath(f.pOpenOther), spec, "bob-token", "", map[string]any{"state": "withdrawn"})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pOtherSite), spec, g7bSessAlice, "", map[string]any{"state": "withdrawn"})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.call(t, http.MethodPatch, path, spec, g7bSessAlice, "", map[string]any{"state": "open"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}
