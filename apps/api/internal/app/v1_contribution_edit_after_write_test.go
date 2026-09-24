package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1EditDecisionSurvivesAFailedReadBack(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	f.up.failReadBack = true
	resp, body := f.call(t, http.MethodPatch, g7bProposalPath(f.pOpenAlice), spec, g7bSessStaff, "", map[string]any{"state": "merged"})
	if resp.StatusCode != http.StatusOK || body["state"] != "merged" {
		t.Fatalf("a merge catalog accepted must answer 200 even when the read-back fails: %d %+v", resp.StatusCode, body)
	}
	if d, _ := body["decider"].(map[string]any); d["id"] != "930000003" {
		t.Errorf("decider %+v", body["decider"])
	}
	if got := f.up.proposal(f.pOpenAlice).state; got != "merged" {
		t.Errorf("upstream state %s", got)
	}
	if n := len(f.snapshotAwards()); n != 1 {
		t.Errorf("the award follows the write, not the read-back: %d", n)
	}
	if got := f.messages(t, "merged", w3UserAlice); len(got) != 1 {
		t.Errorf("merged notification %v", got)
	}

	resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pOpenOther), spec, g7bSessStaff, "", map[string]any{"state": "declined", "note": "dup"})
	if resp.StatusCode != http.StatusOK || body["state"] != "declined" {
		t.Fatalf("decline %d %+v", resp.StatusCode, body)
	}
	if got := f.messages(t, "declined", w3UserOther); len(got) != 1 {
		t.Errorf("declined notification %v", got)
	}

	resp, body = f.call(t, http.MethodPatch, g7bProposalPath(f.pPlainAlice), spec, g7bSessAlice, "", map[string]any{"state": "withdrawn"})
	if resp.StatusCode != http.StatusOK || body["state"] != "withdrawn" {
		t.Errorf("withdraw %d %+v", resp.StatusCode, body)
	}
}

// The fake /users/batch answers every account it knows and the user client
// caches them all, so the proposer here is one it does not know: the lookup
// after the write is a real miss, and it fails.
func TestV1EditWritesSurviveAFailedUserLookupAfterTheWrite(t *testing.T) {
	f := newG7bFix(t)
	const fresh = 930000041
	pid := f.up.addProposal(&g7bProposal{workID: g7bWork, proposer: fresh,
		patch: map[string]any{"catalog.work.display_name": "Fresh"}}).id
	f.up.afterWrite = func() { f.failOA.Store(true) }
	resp, body := f.call(t, http.MethodPatch, g7bProposalPath(pid), "/edit-proposals/{proposal_id}", g7bSessStaff, "", map[string]any{"state": "merged"})
	if resp.StatusCode != http.StatusOK || body["state"] != "merged" {
		t.Fatalf("merge %d %+v", resp.StatusCode, body)
	}
	if prop, _ := body["proposer"].(map[string]any); prop["name"] != nil || prop["id"] != "930000041" {
		t.Errorf("an unreadable proposer falls back to a deleted ref %+v", body["proposer"])
	}
	f.failOA.Store(false)

	resp, body = f.call(t, http.MethodPost, g7bProposalPath(f.pOpenOther)+"/amendments", "/edit-proposals/{proposal_id}/amendments",
		g7bSessStaff, keyUUID(9002), map[string]any{"set": map[string]any{"catalog.work.display_name": "Z"}})
	if resp.StatusCode != http.StatusCreated || asInt(body["seq"]) != 1 {
		t.Errorf("amend %d %+v", resp.StatusCode, body)
	}
}

func TestV1BearerDecisionRefusedBeforeAnyRead(t *testing.T) {
	f := newG7bFix(t)
	spec := "/edit-proposals/{proposal_id}"
	for _, id := range []int64{f.pMergedBob, f.pOtherSite, 957199999} {
		resp, body := f.bearer(t, http.MethodPatch, g7bProposalPath(id), spec, "staff-token", "", map[string]any{"state": "merged"})
		wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	}
	if n := len(f.up.callsTo(http.MethodGet, "/v2/catalog/proposals")); n != 0 {
		t.Errorf("a Bearer decision is refused before catalog is asked, %d reads", n)
	}
}
