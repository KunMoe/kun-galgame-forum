package app

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/pkg/perm"
)

func TestV1PermMatrixGate(t *testing.T) {
	f := newPermFix(t)
	for name, who := range map[string]pmAuth{
		"moderator": {session: "sess-pm-mod"},
		"bearer":    {bearer: "pm-admin-token"},
	} {
		resp, body := f.call(t, http.MethodGet, pmMatrixPath, "/admin/role-permissions", who, nil)
		if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
			t.Errorf("%s read the matrix: %d %v", name, resp.StatusCode, body["code"])
		}
		resp, body = f.call(t, http.MethodPatch, pmMatrixPath, "/admin/role-permissions", who,
			map[string]any{"changes": []any{pmChange("creator")}})
		if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
			t.Errorf("%s wrote the matrix: %d %v", name, resp.StatusCode, body["code"])
		}
	}
	resp, body := f.call(t, http.MethodGet, pmMatrixPath, "/admin/role-permissions", pmAuth{}, nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func TestV1PermMatrixRead(t *testing.T) {
	f := newPermFix(t)
	f.seedOverride(t, "moderator", perm.DocEdit, perm.EffectRevoke)
	resp, body := f.call(t, http.MethodGet, pmMatrixPath, "/admin/role-permissions", pmAuth{session: "sess-pm-admin"}, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "role_permission_matrix" {
		t.Fatalf("matrix %d %+v", resp.StatusCode, body)
	}
	if len(pmStrings(body["catalog"])) != len(perm.Catalog()) {
		t.Errorf("catalog %v", body["catalog"])
	}
	layers, _ := body["role_permissions"].([]any)
	order := make([]string, 0, len(layers))
	for _, l := range layers {
		m, _ := l.(map[string]any)
		order = append(order, m["role"].(string))
	}
	if !slices.Equal(order, []string{"creator", "moderator", "admin", "ren"}) {
		t.Errorf("layer order %v", order)
	}
	mod := pmLayer(t, body, "moderator")
	if pmHas(mod["effective"], perm.DocEdit) || !pmHas(mod["baseline"], perm.DocEdit) {
		t.Errorf("moderator layer %+v", mod)
	}
	ovs, _ := mod["overrides"].([]any)
	if len(ovs) != 1 {
		t.Errorf("moderator overrides %+v", ovs)
	}
	canEdit := func(body map[string]any, role string) bool {
		v, _ := pmLayer(t, body, role)["viewer"].(map[string]any)
		return v["can_edit"] == true
	}
	for role, want := range map[string]bool{"creator": true, "moderator": true, "admin": false, "ren": false} {
		if canEdit(body, role) != want {
			t.Errorf("admin can_edit %s = %v, want %v", role, !want, want)
		}
	}
	if pmLayer(t, body, "ren")["is_locked"] != true || pmLayer(t, body, "admin")["is_locked"] != false {
		t.Error("only ren is locked")
	}
	_, body = f.call(t, http.MethodGet, pmMatrixPath, "/admin/role-permissions", pmAuth{session: "sess-pm-ren"}, nil)
	if !canEdit(body, "admin") || canEdit(body, "ren") {
		t.Error("ren edits admin but not ren")
	}
}

func TestV1PermMatrixWrite(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-admin",
		pmChange("creator", pmOverride(perm.TopicHide, perm.EffectGrant)),
		pmChange("moderator", pmOverride(perm.DocEdit, perm.EffectRevoke)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	if !pmHas(pmLayer(t, body, "creator")["effective"], perm.TopicHide) || pmHas(pmLayer(t, body, "moderator")["effective"], perm.DocEdit) {
		t.Errorf("matrix after write %+v", body)
	}
	if n := f.count(t, `SELECT count(*) FROM role_permission_override WHERE updated_by = ?`, pmAdmin); n != 2 {
		t.Errorf("%d rows stamped by the operator, want 2", n)
	}
	if n := f.count(t, `SELECT count(*) FROM permission_audit_log WHERE operator_id = ? AND subject_kind = 'role'`, pmAdmin); n != 2 {
		t.Errorf("%d audit rows, want one per role", n)
	}
	if !perm.Can([]string{"creator"}, perm.TopicHide) {
		t.Error("the grant is not live in this process")
	}

	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator"))
	if resp.StatusCode != http.StatusOK || pmHas(pmLayer(t, body, "creator")["effective"], perm.TopicHide) {
		t.Fatalf("reset %d %+v", resp.StatusCode, body)
	}
	if n := f.count(t, `SELECT count(*) FROM permission_audit_log WHERE operator_id = ? AND subject = 'creator' AND action = 'reset'`, pmAdmin); n != 1 {
		t.Errorf("%d reset audit rows", n)
	}
}

func TestV1PermMatrixRank(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-admin", pmChange("admin", pmOverride(perm.UserPurgeContent, perm.EffectRevoke)))
	pmRefused(t, resp, body, "pointer", "/changes/0/role", "NOT_PERMITTED")
	resp, body = f.patchMatrix(t, "sess-pm-ren", pmChange("admin", pmOverride(perm.AdminDashboard, perm.EffectRevoke)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ren edits admin: %d %+v", resp.StatusCode, body)
	}
}

func TestV1PermMatrixRen(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-ren", pmChange("ren", pmOverride(perm.TopicHide, perm.EffectRevoke)))
	pmRefused(t, resp, body, "pointer", "/changes/0/role", "NOT_ALLOWED_VALUE")
}

func TestV1PermMatrixPossession(t *testing.T) {
	f := newPermFix(t)
	f.seedOverride(t, pmAdmin, perm.DocEdit, perm.EffectRevoke)
	resp, body := f.patchMatrix(t, "sess-pm-admin", pmChange("moderator", pmOverride(perm.DocEdit, perm.EffectRevoke)))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides/0/permission", "NOT_PERMITTED")

	f.seedOverride(t, "creator", perm.DocEdit, perm.EffectGrant)
	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator"))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides", "NOT_PERMITTED")
}

func TestV1PermMatrixCombinedState(t *testing.T) {
	f := newPermFix(t)
	f.seedOverride(t, "admin", perm.DocEdit, perm.EffectRevoke)
	f.seedOverride(t, "moderator", perm.DocEdit, perm.EffectRevoke)
	resp, body := f.patchMatrix(t, "sess-pm-ren", pmChange("moderator"), pmChange("admin"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restoring a key to both roles at once: %d %+v", resp.StatusCode, body)
	}
	if !pmHas(pmLayer(t, body, "moderator")["effective"], perm.DocEdit) || !pmHas(pmLayer(t, body, "admin")["effective"], perm.DocEdit) {
		t.Errorf("both roles should hold doc.edit again: %+v", body)
	}
	if n := f.count(t, `SELECT count(*) FROM role_permission_override`); n != 0 {
		t.Errorf("%d override rows left", n)
	}
}

func TestV1PermMatrixAtomic(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-ren",
		pmChange("creator", pmOverride(perm.TopicHide, perm.EffectGrant)),
		pmChange("moderator", pmOverride(perm.TopicHide, perm.EffectGrant)))
	pmRefused(t, resp, body, "pointer", "/changes/1/overrides/0/effect", "NOT_ALLOWED_VALUE")
	if n := f.count(t, `SELECT count(*) FROM role_permission_override`); n != 0 {
		t.Errorf("a refused patch wrote %d override rows", n)
	}
	if n := f.count(t, `SELECT count(*) FROM permission_audit_log WHERE operator_id = ?`, pmRen); n != 0 {
		t.Errorf("a refused patch wrote %d audit rows", n)
	}
}

func TestV1PermMatrixNoop(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-admin", pmChange("moderator", pmOverride(perm.TopicHide, perm.EffectGrant)))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides/0/effect", "NOT_ALLOWED_VALUE")
	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator", pmOverride(perm.TopicHide, perm.EffectRevoke)))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides/0/effect", "NOT_ALLOWED_VALUE")
}

func TestV1PermMatrixHierarchy(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-ren", pmChange("admin", pmOverride(perm.DocEdit, perm.EffectRevoke)))
	pmRefused(t, resp, body, "pointer", "/changes", "INCONSISTENT_WITH")
	errs, _ := body["errors"].([]any)
	e, _ := errs[0].(map[string]any)
	if d, _ := e["detail"].(string); !strings.Contains(d, string(perm.DocEdit)) {
		t.Errorf("the detail should name the key: %q", d)
	}
}

func TestV1PermMatrixShapeErrors(t *testing.T) {
	f := newPermFix(t)
	resp, body := f.patchMatrix(t, "sess-pm-admin", pmChange("creator"), pmChange("creator"))
	pmRefused(t, resp, body, "pointer", "/changes/1/role", "DUPLICATE_ITEM")
	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator", pmOverride("no.such_key", perm.EffectGrant)))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides/0/permission", "UNKNOWN_VALUE")
	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator",
		pmOverride(perm.TopicHide, perm.EffectGrant), pmOverride(perm.TopicHide, perm.EffectGrant)))
	pmRefused(t, resp, body, "pointer", "/changes/0/overrides/1/permission", "DUPLICATE_ITEM")
	resp, body = f.patchMatrix(t, "sess-pm-admin", pmChange("creator", pmOverride(perm.TopicHide, "toggle")))
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	resp, body = f.call(t, http.MethodPatch, pmMatrixPath, "/admin/role-permissions", pmAuth{session: "sess-pm-admin"}, map[string]any{"changes": []any{}})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	resp, body = f.call(t, http.MethodPatch, pmMatrixPath, "/admin/role-permissions", pmAuth{session: "sess-pm-admin"}, "not an object")
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("a string body: %d %+v", resp.StatusCode, body)
	}
	if n := f.count(t, `SELECT count(*) FROM permission_audit_log WHERE operator_id = ?`, pmAdmin); n != 0 {
		t.Errorf("refused patches wrote %d audit rows", n)
	}
	if slices.Contains(perm.Baseline("creator"), perm.TopicHide) {
		t.Fatal("fixture assumption: creator has no topic.hide baseline")
	}
}
