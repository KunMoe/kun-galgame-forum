package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type fakeUserStore struct {
	rows []model.UserPermissionOverride
}

func (f *fakeUserStore) ListAll(_ context.Context) ([]model.UserPermissionOverride, error) {
	return append([]model.UserPermissionOverride{}, f.rows...), nil
}

func (f *fakeUserStore) ListForUser(_ context.Context, uid int) ([]model.UserPermissionOverride, error) {
	out := make([]model.UserPermissionOverride, 0)
	for _, r := range f.rows {
		if r.UserID == uid {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeUserStore) Replace(ctx context.Context, uid, operatorUID int, plan func([]model.UserPermissionOverride) ([]model.UserPermissionOverride, error)) error {
	current, _ := f.ListForUser(ctx, uid)
	rows, err := plan(current)
	if err != nil {
		return err
	}
	kept := make([]model.UserPermissionOverride, 0, len(f.rows))
	for _, r := range f.rows {
		if r.UserID != uid {
			kept = append(kept, r)
		}
	}
	for _, r := range rows {
		r.UserID = uid
		r.UpdatedBy = operatorUID
		kept = append(kept, r)
	}
	f.rows = kept
	return nil
}

type emptyRoleStore struct{}

func (emptyRoleStore) ListAll(_ context.Context) ([]model.RolePermissionOverride, error) {
	return nil, nil
}

type fakeUserClient struct {
	roles []string
	found bool
	err   error
}

func (f fakeUserClient) User(_ context.Context, id int) (userclient.User, bool, error) {
	if f.err != nil {
		return userclient.User{}, false, f.err
	}
	if !f.found {
		return userclient.User{}, false, nil
	}
	return userclient.User{ID: id, Roles: f.roles}, true, nil
}

func newUserSvc(store *fakeUserStore, client userLookup) *UserPermissionService {
	return NewUserPermissionService(store, client, NewPermissionOverrideSync(emptyRoleStore{}, store))
}

const targetUID = 4242

func replace(svc *UserPermissionService, op Operator, ovs ...perm.Override) (UserLayer, error) {
	return svc.Replace(context.Background(), op, targetUID, ovs)
}

func wantParameter(t *testing.T, err error, reason string) {
	t.Helper()
	vs := violations(t, err)
	if len(vs) != 1 || vs[0].Parameter != "user_id" || vs[0].Reason != reason {
		t.Fatalf("want %s at user_id, got %+v", reason, vs)
	}
}

func TestUserReplaceRejectsRenHolder(t *testing.T) {
	resetPerm(t)
	_, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"ren"}, found: true}), renOp, revoke(perm.TopicHide))
	wantParameter(t, err, problem.ReasonNotAllowedValue)
}

func TestUserReplaceFailClosed(t *testing.T) {
	resetPerm(t)
	if _, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{err: errors.New("oauth down")}), renOp); !errors.Is(err, ErrUserLookup) {
		t.Errorf("a lookup error must fail closed, got %v", err)
	}
	if _, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{found: false}), renOp); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("a nonexistent user must be not found, got %v", err)
	}
}

func TestUserReplaceRejectsUnknownKey(t *testing.T) {
	resetPerm(t)
	_, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{found: true}), renOp, grant("does.not_exist"))
	wantViolation(t, err, "/overrides/0/permission", problem.ReasonUnknownValue)
}

func TestUserReplaceRejectsDuplicate(t *testing.T) {
	resetPerm(t)
	_, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{found: true}), renOp, grant(perm.TopicHide), grant(perm.TopicHide))
	wantViolation(t, err, "/overrides/1/permission", problem.ReasonDuplicateItem)
}

func TestUserReplaceRejectsNoop(t *testing.T) {
	resetPerm(t)
	_, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"moderator"}, found: true}), renOp, grant(perm.TopicHide))
	wantViolation(t, err, "/overrides/0/effect", problem.ReasonNotAllowedValue)
	_, err = replace(newUserSvc(&fakeUserStore{}, fakeUserClient{found: true}), renOp, revoke(perm.TopicHide))
	wantViolation(t, err, "/overrides/0/effect", problem.ReasonNotAllowedValue)
}

func TestUserReplaceGrantToRoleless(t *testing.T) {
	resetPerm(t)
	store := &fakeUserStore{}
	view, err := replace(newUserSvc(store, fakeUserClient{found: true}), opFor(9, "ren"), grant(perm.TopicHide))
	if err != nil {
		t.Fatalf("valid grant failed: %v", err)
	}
	if len(store.rows) != 1 || store.rows[0].UpdatedBy != 9 {
		t.Fatalf("expected 1 row stamped by operator 9, got %+v", store.rows)
	}
	if !perm.CanUser(targetUID, nil, perm.TopicHide) {
		t.Error("roleless user should hold topic.hide immediately after the grant")
	}
	if len(view.Baseline) != 0 || len(view.Overrides) != 1 || !slices.Contains(view.Effective, perm.TopicHide) {
		t.Errorf("view %+v", view)
	}
}

func TestUserReplaceResetRestores(t *testing.T) {
	resetPerm(t)
	store := &fakeUserStore{rows: []model.UserPermissionOverride{
		{UserID: targetUID, Permission: string(perm.TopicHide), Effect: perm.EffectGrant},
	}}
	perm.SetUserOverrides(map[int][]perm.Override{targetUID: {grant(perm.TopicHide)}})
	if _, err := replace(newUserSvc(store, fakeUserClient{found: true}), renOp); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if len(store.rows) != 0 || perm.CanUser(targetUID, nil, perm.TopicHide) {
		t.Errorf("reset left %+v", store.rows)
	}
}

func TestUserReplaceRankAdminCannotEditPeerAdmin(t *testing.T) {
	resetPerm(t)
	_, err := replace(newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"admin"}, found: true}), opFor(100, "admin"), grant(perm.TopicHide))
	wantParameter(t, err, problem.ReasonNotPermitted)
}

func TestUserLayerMirrorsCanUser(t *testing.T) {
	resetPerm(t)
	rows := []model.UserPermissionOverride{
		{UserID: targetUID, Permission: string(perm.TopicHide), Effect: perm.EffectRevoke},
		{UserID: targetUID, Permission: string(perm.UserPurgeContent), Effect: perm.EffectGrant},
	}
	perm.SetUserOverrides(map[int][]perm.Override{targetUID: {revoke(perm.TopicHide), grant(perm.UserPurgeContent)}})
	for _, roles := range [][]string{nil, {"creator"}, {"moderator"}, {"admin"}, {"ren"}} {
		l := buildUserLayer(targetUID, roles, rows)
		for _, p := range perm.Catalog() {
			if slices.Contains(l.Effective, p) != perm.CanUser(targetUID, roles, p) {
				t.Errorf("roles %v key %s: layer and CanUser disagree", roles, p)
			}
		}
	}
}
