package app

import (
	"net/http"
	"strconv"
	"testing"
)

func TestV1UserListsOwnerMissingIs404(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.userList(t, "", "topics", strconv.Itoa(u3aGone), "relation=authored")
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1UserListsOwnerBannedIs404(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.userList(t, "", "replies", strconv.Itoa(w3UserBanned), "relation=authored")
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1UserListsOwnerStatus2Is404(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(u3aStatus2, "u3a-gone", 2, nil)
	resp, body := f.userList(t, "", "comments", strconv.Itoa(u3aStatus2), "relation=authored")
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1UserListsOwnerUpstreamIs503(t *testing.T) {
	f := newU3aFix(t)
	f.failOA.Store(true)
	f.UserClient.Invalidate(u3aOwner)
	resp, body := f.userList(t, "", "topics", strconv.Itoa(u3aOwner), "relation=authored")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserListsUnknownRelationIs400(t *testing.T) {
	f := newU3aFix(t)
	owner := strconv.Itoa(u3aOwner)
	cases := []struct{ collection, relation string }{
		{"topics", "topic"},
		{"replies", "hidden"},
		{"comments", "nope"},
	}
	for _, c := range cases {
		resp, body := f.userList(t, "", c.collection, owner, "relation="+c.relation)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
			t.Errorf("%s relation=%s: %d %+v", c.collection, c.relation, resp.StatusCode, body)
		}
	}
}

func TestV1UserListsLimitTooLarge(t *testing.T) {
	f := newU3aFix(t)
	owner := strconv.Itoa(u3aOwner)
	for _, collection := range []string{"topics", "replies", "comments"} {
		resp, body := f.userList(t, "", collection, owner, "relation=authored&limit=101")
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
			t.Errorf("%s limit=101: %d %+v", collection, resp.StatusCode, body)
		}
	}
}

func TestV1UserListsDepthBeyondCap(t *testing.T) {
	f := newU3aFix(t)
	owner := strconv.Itoa(u3aOwner)
	resp, body := f.userList(t, "", "topics", owner, "relation=authored&page=101&limit=100")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("too deep %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	e, _ := errs[0].(map[string]any)
	params, _ := e["params"].(map[string]any)
	if e["parameter"] != "page" || e["reason"] != "OUT_OF_RANGE" || asInt(params["maximum"]) != 100 {
		t.Errorf("depth error %+v", e)
	}
	if resp, body = f.userList(t, "", "topics", owner, "relation=authored&page=100&limit=100"); resp.StatusCode != http.StatusOK {
		t.Errorf("exactly at the cap %d %+v", resp.StatusCode, body)
	}
}
