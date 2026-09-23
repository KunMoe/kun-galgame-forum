package app

import (
	"net/http"
	"strconv"
	"testing"
)

func TestV1UserU3bOwnerMissingIs404(t *testing.T) {
	f := newU3bFix(t)
	owner := strconv.Itoa(u3bGone)
	for _, collection := range []string{"works", "galgame-resources"} {
		resp, body := f.rs(t, http.MethodGet, "/api/v1/users/"+owner+"/"+collection+"?relation=published",
			"/users/{user_id}/"+collection, "", "", nil)
		mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	}
}

func TestV1UserU3bOwnerBannedIs404(t *testing.T) {
	f := newU3bFix(t)
	owner := strconv.Itoa(w3UserBanned)
	for _, c := range []struct{ collection, relation string }{
		{"works", "published"},
		{"galgame-resources", "published"},
	} {
		resp, body := f.rs(t, http.MethodGet, "/api/v1/users/"+owner+"/"+c.collection+"?relation="+c.relation,
			"/users/{user_id}/"+c.collection, "", "", nil)
		mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	}
}

func TestV1UserU3bOwnerStatus2Is404(t *testing.T) {
	f := newU3bFix(t)
	owner := strconv.Itoa(u3bStatus2)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/users/"+owner+"/works?relation=published",
		"/users/{user_id}/works", "", "", nil)
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1UserU3bOwnerUpstreamIs503(t *testing.T) {
	f := newU3bFix(t)
	f.failOA.Store(true)
	f.UserClient.Invalidate(u3bOwner)
	for _, collection := range []string{"works", "galgame-resources"} {
		resp, body := f.u3bList(t, "", collection, "relation=published")
		mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	}
}

func TestV1UserWorksCatalogUpstreamIs503(t *testing.T) {
	f := newU3bFix(t)
	f.cat.fail.Store(true)
	resp, body := f.u3bList(t, "", "works", "relation=published")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserWorksContributedUpstreamIs503(t *testing.T) {
	f := newU3bFix(t)
	f.contribFail.Store(true)
	resp, body := f.u3bList(t, "", "works", "relation=contributed")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserResourcesCatalogUpstreamIs503(t *testing.T) {
	f := newU3bFix(t)
	f.cat.fail.Store(true)
	resp, body := f.u3bList(t, "", "galgame-resources", "relation=published")
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserU3bUnknownRelationIs400(t *testing.T) {
	f := newU3bFix(t)
	cases := []struct{ collection, relation string }{
		{"works", "bogus"},
		{"galgame-resources", "valid"},
	}
	for _, c := range cases {
		resp, body := f.u3bList(t, "", c.collection, "relation="+c.relation)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
			t.Errorf("%s relation=%s: %d %+v", c.collection, c.relation, resp.StatusCode, body)
		}
	}
}

func TestV1UserResourcesUnknownStateIs400(t *testing.T) {
	f := newU3bFix(t)
	resp, body := f.u3bList(t, "", "galgame-resources", "relation=published&state=alive")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
		t.Fatalf("state=alive %d %+v", resp.StatusCode, body)
	}
}

func TestV1UserU3bLimitTooLarge(t *testing.T) {
	f := newU3bFix(t)
	for _, collection := range []string{"works", "galgame-resources"} {
		resp, body := f.u3bList(t, "", collection, "relation=published&limit=101")
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
			t.Errorf("%s limit=101: %d %+v", collection, resp.StatusCode, body)
		}
	}
}

func TestV1UserU3bDepthBeyondCap(t *testing.T) {
	f := newU3bFix(t)
	resp, body := f.u3bList(t, "", "works", "relation=published&page=101&limit=100")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("too deep %d %+v", resp.StatusCode, body)
	}
	if resp, body = f.u3bList(t, "", "galgame-resources", "relation=published&page=101&limit=100"); resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("resources too deep %d %+v", resp.StatusCode, body)
	}
	if resp, body = f.u3bList(t, "", "works", "relation=published&page=100&limit=100"); resp.StatusCode != http.StatusOK {
		t.Errorf("exactly at the cap %d %+v", resp.StatusCode, body)
	}
}
