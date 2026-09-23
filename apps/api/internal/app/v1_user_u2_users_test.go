package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestV1ListUsersQIDsNeitherIs422(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users", "/users", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "parameter", "q", "REQUIRED")
}

func TestV1ListUsersQIDsBothAre422(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.listUsersQuery(t, "sess-alice", "q=alice&ids="+strconv.Itoa(w3UserBob))
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "parameter", "ids", "INCONSISTENT_WITH")
}

func TestV1ListUsersQModeMissingIsEmpty(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.listUsersQuery(t, "sess-alice", "q=alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if fmt.Sprint(body["missing"]) != "[]" {
		t.Fatalf("q mode missing %v, want []", body["missing"])
	}
}

func TestV1ListUsersIDsOrderDedupeAndOneBatch(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(u2UserStatus2, "deleted", 2, nil)
	bob := strconv.Itoa(w3UserBob)
	alice := strconv.Itoa(w3UserAlice)
	banned := strconv.Itoa(w3UserBanned)
	status2 := strconv.Itoa(u2UserStatus2)
	gone := strconv.Itoa(u2UserGone)
	f.UserClient.Invalidate(w3UserBob, w3UserAlice, w3UserBanned, u2UserStatus2, u2UserGone)
	before := f.nBatch.Load()
	resp, body := f.listUsersQuery(t, "sess-alice", "ids="+strings.Join([]string{bob, alice, banned, status2, gone, bob}, ","))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if f.nBatch.Load() != before+1 {
		t.Fatalf("batch calls %d → %d, want one upstream fetch", before, f.nBatch.Load())
	}
	if got := itemIDs(t, body); fmt.Sprint(got) != fmt.Sprint([]string{bob, alice}) {
		t.Fatalf("items %v, want [%s %s] in request order", got, bob, alice)
	}
	if got := missingIDs(body); fmt.Sprint(got) != fmt.Sprint([]string{banned, status2, gone}) {
		t.Fatalf("missing %v, want [%s %s %s]", got, banned, status2, gone)
	}
}

func TestV1ListUsersIDsCap(t *testing.T) {
	f := newMeFix(t)
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(930001001 + i)
	}
	resp, body := f.listUsersQuery(t, "sess-alice", "ids="+strings.Join(ids, ","))
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("101 ids %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors %v", body["errors"])
	}
	e, _ := errs[0].(map[string]any)
	if e["reason"] != "TOO_MANY_ITEMS" || e["parameter"] != "ids" {
		t.Fatalf("field error %+v", e)
	}
	params, _ := e["params"].(map[string]any)
	if asInt(params["max_items"]) != 100 {
		t.Errorf("max_items %v", params)
	}

	resp, body = f.listUsersQuery(t, "sess-alice", "ids="+strings.Join(ids[:100], ","))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("100 ids %d %+v", resp.StatusCode, body)
	}
	if len(missingIDs(body)) != 100 {
		t.Errorf("none of those extra ids exist, so all 100 are missing: %v", body["missing"])
	}
}

func TestV1ListUsersIDsOAuthDownIs503(t *testing.T) {
	f := newMeFix(t)
	f.failOA.Store(true)
	f.UserClient.Invalidate(w3UserAlice)
	resp, body := f.listUsersQuery(t, "sess-alice", "ids="+strconv.Itoa(w3UserAlice))
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}
