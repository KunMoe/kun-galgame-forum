package app

import (
	"net/http"
	"slices"
	"testing"

	"kun-galgame-api/pkg/perm"
)

func TestV1PermMine(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{session: "sess-pm-mod"}, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "permission_set" {
		t.Fatalf("mine %d %+v", resp.StatusCode, body)
	}
	want := make([]string, 0)
	for _, p := range perm.Catalog() {
		if slices.Contains(perm.Bundles["moderator"], p) {
			want = append(want, string(p))
		}
	}
	if !slices.Equal(pmStrings(body["permissions"]), want) {
		t.Errorf("moderator holds %v, want the bundle in catalog order %v", body["permissions"], want)
	}

	_, body = f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{session: "sess-pm-plain"}, nil)
	if got := pmStrings(body["permissions"]); len(got) != 0 {
		t.Errorf("a plain user holds %v", got)
	}

	resp, body = f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{}, nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	resp, body = f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{bearer: "nope"}, nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIAL")
}

func TestV1PermMineBearerHoldsNothing(t *testing.T) {
	f := newPermFix(t)
	f.seedOverride(t, pmGrantee, perm.TopicHide, perm.EffectGrant)

	_, body := f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{session: "sess-pm-grantee"}, nil)
	if got := pmStrings(body["permissions"]); !slices.Equal(got, []string{string(perm.TopicHide)}) {
		t.Fatalf("the session sees %v, want the personal grant", got)
	}
	resp, body := f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{bearer: "pm-grantee-token"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer %d %+v", resp.StatusCode, body)
	}
	if got := pmStrings(body["permissions"]); len(got) != 0 {
		t.Errorf("a Bearer request holds %v; staff powers must never reach the App channel", got)
	}
}

func TestV1PermMineSeesAWriteAtOnce(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-admin", pmChange("moderator", pmOverride(perm.TopicHide, perm.EffectRevoke)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	_, body = f.call(t, http.MethodGet, pmMinePath, "/me/permissions", pmAuth{session: "sess-pm-mod"}, nil)
	if pmHas(body["permissions"], perm.TopicHide) {
		t.Error("the moderator still holds topic.hide right after it was revoked")
	}
}
