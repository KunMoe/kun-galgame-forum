package perm

import "testing"

func resetUserOverrides(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		SetOverrides(nil)
		SetUserOverrides(nil)
	})
}

func TestCanUserGrantToRolelessUser(t *testing.T) {
	resetUserOverrides(t)
	SetUserOverrides(map[int][]Override{
		42: {{Permission: TopicHide, Effect: EffectGrant}},
	})

	if !CanUser(42, nil, TopicHide) {
		t.Error("granted user 42 topic.hide but CanUser = false")
	}
	if CanUser(42, nil, TopicEditAny) {
		t.Error("user 42 should only hold the single granted key")
	}
	if CanUser(43, nil, TopicHide) {
		t.Error("the grant must not leak to a different user")
	}
	if Can(nil, TopicHide) {
		t.Error("a personal grant must not change the role-level Can")
	}
}

func TestCanUserRevokeBeatsRoleGrant(t *testing.T) {
	resetUserOverrides(t)
	if !CanUser(7, []string{"moderator"}, TopicHide) {
		t.Fatal("precondition: moderator should hold topic.hide")
	}
	SetUserOverrides(map[int][]Override{
		7: {{Permission: TopicHide, Effect: EffectRevoke}},
	})
	if CanUser(7, []string{"moderator"}, TopicHide) {
		t.Error("personal revoke must beat the moderator role grant")
	}
	if !CanUser(7, []string{"moderator"}, TopicEditAny) {
		t.Error("revoking one key must not drop the role's other baseline keys")
	}
}

func TestCanUserRenImmunity(t *testing.T) {
	resetUserOverrides(t)
	renRevokeAll := make([]Override, 0, len(allPerms))
	for _, p := range allPerms {
		renRevokeAll = append(renRevokeAll, Override{Permission: p, Effect: EffectRevoke})
	}
	SetUserOverrides(map[int][]Override{9: renRevokeAll})

	for _, p := range allPerms {
		if !CanUser(9, []string{"ren"}, p) {
			t.Errorf("ren-holder lost %q to a personal override — ren must be immune", p)
		}
	}
	if got := len(effectiveForUser(9, []string{"ren"})); got != totalPerms {
		t.Errorf("effectiveForUser(ren-holder) has %d keys, want %d", got, totalPerms)
	}
}

func TestCanUserUIDZeroPassthrough(t *testing.T) {
	resetUserOverrides(t)
	SetUserOverrides(map[int][]Override{
		0: {{Permission: TopicHide, Effect: EffectGrant}},
	})
	if CanUser(0, nil, TopicHide) {
		t.Error("uid 0 must fall through to the role decision (no personal layer)")
	}
	if !CanUser(0, []string{"moderator"}, TopicHide) {
		t.Error("uid 0 with a moderator role should hold topic.hide via Can")
	}
}

func TestCanUserUnknownKeyFiltered(t *testing.T) {
	resetUserOverrides(t)
	SetUserOverrides(map[int][]Override{
		5: {{Permission: Permission("does.not.exist"), Effect: EffectGrant}},
	})
	if CanUser(5, nil, Permission("does.not.exist")) {
		t.Error("unknown permission was granted through a personal override")
	}
	if got := len(effectiveForUser(5, nil)); got != 0 {
		t.Errorf("roleless user 5 effective has %d keys, want 0 (unknown filtered out)", got)
	}
}

func TestCanUserReset(t *testing.T) {
	resetUserOverrides(t)
	SetUserOverrides(map[int][]Override{
		42: {{Permission: TopicHide, Effect: EffectGrant}},
		7:  {{Permission: TopicHide, Effect: EffectRevoke}},
	})
	SetUserOverrides(nil)

	if CanUser(42, nil, TopicHide) {
		t.Error("after reset the personal grant should be gone")
	}
	if !CanUser(7, []string{"moderator"}, TopicHide) {
		t.Error("after reset the personal revoke should be gone (role grant restored)")
	}
}

func TestEffectiveForUserComposition(t *testing.T) {
	resetUserOverrides(t)
	SetUserOverrides(map[int][]Override{
		7: {
			{Permission: TopicHide, Effect: EffectRevoke},
			{Permission: AdminDashboard, Effect: EffectGrant},
		},
	})
	eff := effectiveForUser(7, []string{"moderator"})
	if len(eff) != modPerms {
		t.Errorf("effective has %d keys, want %d", len(eff), modPerms)
	}
	set := make(map[Permission]bool, len(eff))
	for _, p := range eff {
		set[p] = true
	}
	if set[TopicHide] {
		t.Error("topic.hide should be revoked from the effective set")
	}
	if !set[AdminDashboard] {
		t.Error("admin.dashboard should be granted into the effective set")
	}
	assertCatalogOrder(t, eff)
}

func effectiveForUser(uid int, roles []string) []Permission {
	var out []Permission
	for _, p := range catalog {
		if CanUser(uid, roles, p) {
			out = append(out, p)
		}
	}
	return out
}
