package app

import (
	"fmt"
	"net/http"
	"testing"

	"kun-galgame-api/pkg/perm"
)

func pmUserURL(uid int) string { return fmt.Sprintf("/api/v1/admin/user-permissions/%d", uid) }

func (f *permFix) putUser(t *testing.T, session string, uid int, overrides ...map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	if overrides == nil {
		overrides = []map[string]any{}
	}
	return f.call(t, http.MethodPut, pmUserURL(uid), pmUserSpec, pmAuth{session: session}, map[string]any{"overrides": overrides})
}

func TestV1PermUserRead(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.call(t, http.MethodGet, pmUserURL(pmModTarget), pmUserSpec, pmAuth{session: "sess-pm-admin"}, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "user_permissions" || strID(body["id"]) != fmt.Sprint(pmModTarget) {
		t.Fatalf("view %d %+v", resp.StatusCode, body)
	}
	if roles := pmStrings(body["roles"]); len(roles) != 1 || roles[0] != "moderator" {
		t.Errorf("roles %v: only ranked roles", roles)
	}
	if len(pmStrings(body["baseline"])) != len(perm.Bundles["moderator"]) || body["is_locked"] != false {
		t.Errorf("baseline %v", body["baseline"])
	}
	if v, _ := body["viewer"].(map[string]any); v["can_edit"] != true {
		t.Errorf("admin can edit a moderator: %+v", v)
	}

	_, body = f.call(t, http.MethodGet, pmUserURL(pmAdminPeer), pmUserSpec, pmAuth{session: "sess-pm-admin"}, nil)
	if v, _ := body["viewer"].(map[string]any); v["can_edit"] != false {
		t.Errorf("admin cannot edit a peer admin: %+v", v)
	}
	_, body = f.call(t, http.MethodGet, pmUserURL(pmRenTarget), pmUserSpec, pmAuth{session: "sess-pm-ren"}, nil)
	if body["is_locked"] != true || len(pmStrings(body["effective"])) != len(perm.Catalog()) {
		t.Errorf("a ren holder is locked and holds everything: %+v", body)
	}
	if v, _ := body["viewer"].(map[string]any); v["can_edit"] != false {
		t.Errorf("nobody edits a ren holder: %+v", v)
	}

	resp, body = f.call(t, http.MethodGet, pmUserURL(pmMissing), pmUserSpec, pmAuth{session: "sess-pm-admin"}, nil)
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	resp, body = f.call(t, http.MethodGet, pmUserURL(pmModTarget), pmUserSpec, pmAuth{session: "sess-pm-mod"}, nil)
	mustCode(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
}

func TestV1PermUserReplace(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.putUser(t, "sess-pm-admin", pmModTarget,
		pmOverride(perm.TopicHide, perm.EffectRevoke), pmOverride(perm.UserPurgeContent, perm.EffectGrant))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replace %d %+v", resp.StatusCode, body)
	}
	if pmHas(body["effective"], perm.TopicHide) || !pmHas(body["effective"], perm.UserPurgeContent) || !pmHas(body["baseline"], perm.TopicHide) {
		t.Errorf("layer after replace %+v", body)
	}
	if perm.CanUser(pmModTarget, []string{"moderator"}, perm.TopicHide) {
		t.Error("the revoke is not live in this process")
	}
	if n := f.count(t, `SELECT count(*) FROM user_permission_override WHERE user_id = ? AND updated_by = ?`, pmModTarget, pmAdmin); n != 2 {
		t.Errorf("%d rows stamped by the operator", n)
	}
	if n := f.count(t, `SELECT count(*) FROM permission_audit_log WHERE operator_id = ? AND subject_kind = 'user' AND subject = ?`, pmAdmin, fmt.Sprint(pmModTarget)); n != 1 {
		t.Errorf("%d audit rows", n)
	}

	resp, body = f.putUser(t, "sess-pm-admin", pmModTarget)
	if resp.StatusCode != http.StatusOK || len(pmStrings(body["effective"])) != len(perm.Bundles["moderator"]) {
		t.Fatalf("reset %d %+v", resp.StatusCode, body)
	}
}

func TestV1PermUserRefusals(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.putUser(t, "sess-pm-ren", pmRenTarget, pmOverride(perm.TopicHide, perm.EffectRevoke))
	pmRefused(t, resp, body, "parameter", "user_id", "NOT_ALLOWED_VALUE")
	resp, body = f.putUser(t, "sess-pm-admin", pmAdminPeer, pmOverride(perm.TopicHide, perm.EffectRevoke))
	pmRefused(t, resp, body, "parameter", "user_id", "NOT_PERMITTED")
	resp, body = f.putUser(t, "sess-pm-admin", pmModTarget, pmOverride(perm.TopicHide, perm.EffectGrant))
	pmRefused(t, resp, body, "pointer", "/overrides/0/effect", "NOT_ALLOWED_VALUE")
	resp, body = f.putUser(t, "sess-pm-admin", pmModTarget, pmOverride("no.such_key", perm.EffectRevoke))
	pmRefused(t, resp, body, "pointer", "/overrides/0/permission", "UNKNOWN_VALUE")
	resp, body = f.putUser(t, "sess-pm-admin", pmModTarget,
		pmOverride(perm.TopicHide, perm.EffectRevoke), pmOverride(perm.TopicHide, perm.EffectRevoke))
	pmRefused(t, resp, body, "pointer", "/overrides/1/permission", "DUPLICATE_ITEM")

	f.seedOverride(t, pmAdmin, perm.DocEdit, perm.EffectRevoke)
	resp, body = f.putUser(t, "sess-pm-admin", pmModTarget, pmOverride(perm.DocEdit, perm.EffectRevoke))
	pmRefused(t, resp, body, "pointer", "/overrides/0/permission", "NOT_PERMITTED")

	resp, body = f.putUser(t, "sess-pm-admin", pmMissing)
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	if n := f.count(t, `SELECT count(*) FROM user_permission_override WHERE user_id <> ?`, pmAdmin); n != 0 {
		t.Errorf("refused writes stored %d rows", n)
	}
}

func TestV1PermUserFailsClosed(t *testing.T) {
	f := newPermFix(t)
	f.oaDown.Store(true)
	resp, body := f.putUser(t, "sess-pm-admin", pmModTarget, pmOverride(perm.TopicHide, perm.EffectRevoke))
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.call(t, http.MethodGet, pmUserURL(pmModTarget), pmUserSpec, pmAuth{session: "sess-pm-admin"}, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	if n := f.count(t, `SELECT count(*) FROM user_permission_override WHERE user_id = ?`, pmModTarget); n != 0 {
		t.Errorf("a write went through with the account service down: %d rows", n)
	}
}
