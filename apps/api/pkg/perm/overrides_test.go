package perm

import "testing"

func resetOverrides(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { SetOverrides(nil) })
}

func TestOverridesApply(t *testing.T) {
	resetOverrides(t)
	SetOverrides(map[string][]Override{
		"creator":   {{Permission: TopicHide, Effect: EffectGrant}},
		"moderator": {{Permission: TopicHide, Effect: EffectRevoke}},
	})

	if !Can([]string{"creator"}, TopicHide) {
		t.Error("granted creator topic.hide but Can = false")
	}
	if Can([]string{"moderator"}, TopicHide) {
		t.Error("revoked moderator topic.hide but Can = true")
	}
	if Can([]string{"creator"}, TopicEditAny) {
		t.Error("creator should only hold the single granted key")
	}
	if !Can([]string{"moderator"}, TopicEditAny) {
		t.Error("revoking one key must not drop moderator's other baseline keys")
	}
}

func TestRenImmunity(t *testing.T) {
	resetOverrides(t)
	renRevokeAll := make([]Override, 0, len(allPerms))
	for _, p := range allPerms {
		renRevokeAll = append(renRevokeAll, Override{Permission: p, Effect: EffectRevoke})
	}
	SetOverrides(map[string][]Override{"ren": renRevokeAll})

	for _, p := range allPerms {
		if !Can([]string{"ren"}, p) {
			t.Errorf("ren lost %q to an override — ren must be immune", p)
		}
	}
	if got := len(effectiveRole("ren")); got != totalPerms {
		t.Errorf("effective ren has %d keys, want %d (pinned to full catalog)", got, totalPerms)
	}
}

func TestUnknownKeyFiltered(t *testing.T) {
	resetOverrides(t)
	SetOverrides(map[string][]Override{
		"creator": {{Permission: Permission("does.not.exist"), Effect: EffectGrant}},
	})
	if Can([]string{"creator"}, Permission("does.not.exist")) {
		t.Error("unknown permission was granted through an override")
	}
	if got := len(effectiveRole("creator")); got != 0 {
		t.Errorf("creator effective has %d keys, want 0 (unknown filtered out)", got)
	}
}

func TestNilResetRestoresBaseline(t *testing.T) {
	resetOverrides(t)
	SetOverrides(map[string][]Override{
		"creator":   {{Permission: TopicHide, Effect: EffectGrant}},
		"moderator": {{Permission: TopicHide, Effect: EffectRevoke}},
	})
	SetOverrides(nil)

	if Can([]string{"creator"}, TopicHide) {
		t.Error("after reset creator should hold nothing")
	}
	if !Can([]string{"moderator"}, TopicHide) {
		t.Error("after reset moderator should hold its baseline topic.hide again")
	}
	if got := len(effectiveRole("moderator")); got != modPerms {
		t.Errorf("after reset moderator has %d keys, want %d", got, modPerms)
	}
}

func TestEffectiveRoleShape(t *testing.T) {
	resetOverrides(t)
	SetOverrides(nil)
	sizes := map[string]int{"creator": 0, "moderator": modPerms, "admin": totalPerms, "ren": totalPerms}
	for role, want := range sizes {
		if got := effectiveRole(role); len(got) != want {
			t.Errorf("effective %q has %d keys, want %d", role, len(got), want)
		}
	}
	assertCatalogOrder(t, effectiveRole("admin"))
}

func TestCatalogAndBaseline(t *testing.T) {
	if got := len(Catalog()); got != totalPerms {
		t.Errorf("Catalog() has %d keys, want %d", got, totalPerms)
	}
	cases := map[string]int{"creator": 0, "moderator": modPerms, "admin": totalPerms, "ren": totalPerms, "user": 0, "unknown": 0}
	for role, want := range cases {
		if got := len(Baseline(role)); got != want {
			t.Errorf("Baseline(%q) has %d keys, want %d", role, got, want)
		}
	}
	if !BaselineHas("moderator", TopicHide) {
		t.Error("BaselineHas(moderator, topic.hide) = false, want true")
	}
	if BaselineHas("moderator", AdminDashboard) {
		t.Error("BaselineHas(moderator, admin.dashboard) = true, want false (admin-only)")
	}
	if BaselineHas("creator", TopicHide) {
		t.Error("BaselineHas(creator, topic.hide) = true, want false")
	}
}

func TestEffectiveSetPure(t *testing.T) {
	if got := len(EffectiveSet("creator", []Override{{Permission: TopicHide, Effect: EffectGrant}})); got != 1 {
		t.Errorf("EffectiveSet(creator, +topic.hide) has %d keys, want 1", got)
	}
	if got := len(EffectiveSet("moderator", []Override{{Permission: TopicHide, Effect: EffectRevoke}})); got != modPerms-1 {
		t.Errorf("EffectiveSet(moderator, -topic.hide) has %d keys, want %d", got, modPerms-1)
	}
	if got := len(EffectiveSet("ren", []Override{{Permission: TopicHide, Effect: EffectRevoke}})); got != totalPerms {
		t.Errorf("EffectiveSet(ren, ...) has %d keys, want %d (pinned)", got, totalPerms)
	}
	if !Can([]string{"moderator"}, TopicHide) {
		t.Error("EffectiveSet must not mutate the installed resolver")
	}
}

func assertCatalogOrder(t *testing.T, perms []Permission) {
	t.Helper()
	last := -1
	for _, p := range perms {
		idx := catalogIndex[p]
		if idx <= last {
			t.Errorf("permissions out of catalog order at %q (index %d after %d)", p, idx, last)
		}
		last = idx
	}
}

func effectiveRole(role string) []Permission {
	var out []Permission
	for _, p := range catalog {
		if Can([]string{role}, p) {
			out = append(out, p)
		}
	}
	return out
}
