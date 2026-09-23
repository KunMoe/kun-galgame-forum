package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type fakeStore struct {
	rows []model.RolePermissionOverride
}

func (f *fakeStore) ListAll(_ context.Context) ([]model.RolePermissionOverride, error) {
	return append([]model.RolePermissionOverride{}, f.rows...), nil
}

func (f *fakeStore) Replace(_ context.Context, operatorUID int, plan func([]model.RolePermissionOverride) ([]model.RoleReplacement, error)) error {
	next, err := plan(append([]model.RolePermissionOverride{}, f.rows...))
	if err != nil {
		return err
	}
	for _, rep := range next {
		kept := make([]model.RolePermissionOverride, 0, len(f.rows))
		for _, r := range f.rows {
			if r.Role != rep.Role {
				kept = append(kept, r)
			}
		}
		for _, r := range rep.Rows {
			r.Role = rep.Role
			r.UpdatedBy = operatorUID
			kept = append(kept, r)
		}
		f.rows = kept
	}
	return nil
}

func resetPerm(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		perm.SetOverrides(nil)
		perm.SetUserOverrides(nil)
	})
}

type emptyUserStore struct{}

func (emptyUserStore) ListAll(_ context.Context) ([]model.UserPermissionOverride, error) {
	return nil, nil
}

func newSvc(store *fakeStore) *RolePermissionService {
	return NewRolePermissionService(store, NewPermissionOverrideSync(store, emptyUserStore{}))
}

func grant(p perm.Permission) perm.Override {
	return perm.Override{Permission: p, Effect: perm.EffectGrant}
}
func revoke(p perm.Permission) perm.Override {
	return perm.Override{Permission: p, Effect: perm.EffectRevoke}
}

func opFor(uid int, roles ...string) Operator {
	return Operator{ID: uid, Rank: perm.Rank(roles), Holds: func(p perm.Permission) bool { return perm.CanUser(uid, roles, p) }}
}

var renOp = opFor(1, "ren")

func violations(t *testing.T, err error) []Violation {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want a ValidationError, got %v", err)
	}
	return ve.Violations
}

func wantViolation(t *testing.T, err error, pointer, reason string) {
	t.Helper()
	for _, v := range violations(t, err) {
		if v.Pointer == pointer && v.Reason == reason {
			return
		}
	}
	t.Fatalf("no %s at %s in %+v", reason, pointer, violations(t, err))
}

func apply(svc *RolePermissionService, op Operator, changes ...RoleChange) (Matrix, error) {
	return svc.Apply(context.Background(), op, changes)
}

func layer(m Matrix, role string) RoleLayer {
	for _, l := range m.Roles {
		if l.Role == role {
			return l
		}
	}
	return RoleLayer{}
}

func TestApplyRejectsRen(t *testing.T) {
	resetPerm(t)
	_, err := apply(newSvc(&fakeStore{}), renOp, RoleChange{Role: "ren", Overrides: []perm.Override{revoke(perm.TopicHide)}})
	wantViolation(t, err, "/changes/0/role", problem.ReasonNotAllowedValue)
}

func TestApplyRejectsDuplicateRole(t *testing.T) {
	resetPerm(t)
	_, err := apply(newSvc(&fakeStore{}), renOp, RoleChange{Role: "creator"}, RoleChange{Role: "creator"})
	wantViolation(t, err, "/changes/1/role", problem.ReasonDuplicateItem)
}

func TestApplyRejectsUnknownKey(t *testing.T) {
	resetPerm(t)
	_, err := apply(newSvc(&fakeStore{}), renOp, RoleChange{Role: "creator", Overrides: []perm.Override{grant("does.not_exist")}})
	wantViolation(t, err, "/changes/0/overrides/0/permission", problem.ReasonUnknownValue)
}

func TestApplyRejectsNoop(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, err := apply(svc, renOp, RoleChange{Role: "moderator", Overrides: []perm.Override{grant(perm.TopicHide)}})
	wantViolation(t, err, "/changes/0/overrides/0/effect", problem.ReasonNotAllowedValue)
	_, err = apply(svc, renOp, RoleChange{Role: "creator", Overrides: []perm.Override{revoke(perm.TopicHide)}})
	wantViolation(t, err, "/changes/0/overrides/0/effect", problem.ReasonNotAllowedValue)
}

func TestApplyRejectsDuplicateKey(t *testing.T) {
	resetPerm(t)
	_, err := apply(newSvc(&fakeStore{}), renOp, RoleChange{Role: "creator", Overrides: []perm.Override{grant(perm.TopicHide), grant(perm.TopicHide)}})
	wantViolation(t, err, "/changes/0/overrides/1/permission", problem.ReasonDuplicateItem)
}

func TestApplyContainmentViolation(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "admin", Permission: string(perm.UserPurgeContent), Effect: perm.EffectRevoke},
	}}
	_, err := apply(newSvc(store), renOp, RoleChange{Role: "moderator", Overrides: []perm.Override{grant(perm.UserPurgeContent)}})
	wantViolation(t, err, "/changes", problem.ReasonInconsistentWith)
}

func TestApplyJudgesTheCombinedState(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "admin", Permission: string(perm.DocEdit), Effect: perm.EffectRevoke},
		{Role: "moderator", Permission: string(perm.DocEdit), Effect: perm.EffectRevoke},
	}}
	m, err := apply(newSvc(store), renOp, RoleChange{Role: "moderator"}, RoleChange{Role: "admin"})
	if err != nil {
		t.Fatalf("restoring a key to both roles at once must pass: %v", err)
	}
	if !slices.Contains(layer(m, "moderator").Effective, perm.DocEdit) || !slices.Contains(layer(m, "admin").Effective, perm.DocEdit) {
		t.Fatal("both roles should hold doc.edit again")
	}
	if len(store.rows) != 0 {
		t.Fatalf("both reset, rows %+v", store.rows)
	}
}

func TestApplyHappyPath(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{}
	m, err := apply(newSvc(store), opFor(7, "ren"), RoleChange{Role: "creator", Overrides: []perm.Override{grant(perm.TopicHide)}})
	if err != nil {
		t.Fatalf("valid apply failed: %v", err)
	}
	if len(store.rows) != 1 || store.rows[0].UpdatedBy != 7 {
		t.Fatalf("expected 1 row stamped by operator 7, got %+v", store.rows)
	}
	if !perm.Can([]string{"creator"}, perm.TopicHide) {
		t.Error("creator should hold topic.hide immediately after a valid apply")
	}
	creator := layer(m, "creator")
	if len(creator.Overrides) != 1 || creator.Overrides[0].Permission != perm.TopicHide {
		t.Errorf("creator overrides = %+v", creator.Overrides)
	}
	if !slices.Contains(creator.Effective, perm.TopicHide) || len(creator.Baseline) != 0 {
		t.Errorf("creator layer %+v", creator)
	}
	if !layer(m, "ren").Locked || len(m.Catalog) != len(perm.Catalog()) || len(m.Roles) != 4 {
		t.Errorf("matrix shape %+v", m)
	}
}

func TestApplyResetRestoresBaseline(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(perm.TopicHide), Effect: perm.EffectGrant},
	}}
	if _, err := apply(newSvc(store), renOp, RoleChange{Role: "creator"}); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if len(store.rows) != 0 || perm.Can([]string{"creator"}, perm.TopicHide) {
		t.Errorf("reset left %+v", store.rows)
	}
}

func TestApplyRankAdminCannotEditAdminRole(t *testing.T) {
	resetPerm(t)
	_, err := apply(newSvc(&fakeStore{}), opFor(100, "admin"), RoleChange{Role: "admin", Overrides: []perm.Override{revoke(perm.UserPurgeContent)}})
	wantViolation(t, err, "/changes/0/role", problem.ReasonNotPermitted)
}

func TestApplyRankRenCanEditAdminRole(t *testing.T) {
	resetPerm(t)
	if _, err := apply(newSvc(&fakeStore{}), renOp, RoleChange{Role: "admin", Overrides: []perm.Override{revoke(perm.AdminDashboard)}}); err != nil {
		t.Fatalf("ren editing the admin role must succeed, got %v", err)
	}
}

func TestApplyPossessionAddedRow(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{100: {revoke(perm.TopicHide)}})
	_, err := apply(newSvc(&fakeStore{}), opFor(100, "admin"), RoleChange{Role: "creator", Overrides: []perm.Override{grant(perm.TopicHide)}})
	wantViolation(t, err, "/changes/0/overrides/0/permission", problem.ReasonNotPermitted)
}

func TestApplyPossessionCarriedOverPasses(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{100: {revoke(perm.TopicHide)}})
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(perm.TopicHide), Effect: perm.EffectGrant},
	}}
	m, err := apply(newSvc(store), opFor(100, "admin"), RoleChange{Role: "creator", Overrides: []perm.Override{grant(perm.TopicHide), grant(perm.DocEdit)}})
	if err != nil {
		t.Fatalf("carrying over an unheld row while adding a held one must pass, got %v", err)
	}
	creator := layer(m, "creator")
	if !slices.Contains(creator.Effective, perm.TopicHide) || !slices.Contains(creator.Effective, perm.DocEdit) {
		t.Errorf("creator effective %v", creator.Effective)
	}
}

func TestApplyPossessionRemovalChecked(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{100: {revoke(perm.TopicHide)}})
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(perm.TopicHide), Effect: perm.EffectGrant},
	}}
	_, err := apply(newSvc(store), opFor(100, "admin"), RoleChange{Role: "creator", Overrides: []perm.Override{}})
	wantViolation(t, err, "/changes/0/overrides", problem.ReasonNotPermitted)
}
