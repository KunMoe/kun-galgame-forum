package service

import (
	"fmt"
	"slices"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

// Checks run in three stages, structure, possession, hierarchy, and only the
// first stage that finds anything is reported.
func validateRoleChanges(op Operator, current []model.RolePermissionOverride, changes []RoleChange) []Violation {
	var vs []Violation
	seen := make(map[string]bool, len(changes))
	for i, ch := range changes {
		at := fmt.Sprintf("/changes/%d", i)
		switch {
		case ch.Role == roleRen:
			vs = append(vs, Violation{Pointer: at + "/role", Reason: problem.ReasonNotAllowedValue,
				Detail: "ren always holds every permission and cannot be changed"})
			continue
		case seen[ch.Role]:
			vs = append(vs, Violation{Pointer: at + "/role", Reason: problem.ReasonDuplicateItem,
				Detail: "role " + ch.Role + " appears more than once"})
			continue
		}
		seen[ch.Role] = true
		if op.Rank <= perm.RoleRank(ch.Role) {
			vs = append(vs, Violation{Pointer: at + "/role", Reason: problem.ReasonNotPermitted,
				Detail: "role " + ch.Role + " ranks at or above the caller"})
		}
		role := ch.Role
		vs = append(vs, overrideViolations(at+"/overrides", ch.Overrides, func(p perm.Permission) bool {
			return perm.BaselineHas(role, p)
		})...)
	}
	if len(vs) > 0 {
		return vs
	}

	for i, ch := range changes {
		vs = append(vs, possessionViolations(fmt.Sprintf("/changes/%d/overrides", i),
			roleEffects(current, ch.Role), ch.Overrides, op.Holds)...)
	}
	if len(vs) > 0 {
		return vs
	}

	prospective := roleRowsToOverrideMap(current)
	for _, ch := range changes {
		prospective[ch.Role] = ch.Overrides
	}
	admin := perm.EffectiveSet("admin", prospective["admin"])
	for _, p := range perm.EffectiveSet("moderator", prospective["moderator"]) {
		if !slices.Contains(admin, p) {
			vs = append(vs, Violation{Pointer: "/changes", Reason: problem.ReasonInconsistentWith,
				Detail: "moderator would hold " + string(p) + ", which admin lacks"})
		}
	}
	return vs
}

func validateUserReplace(op Operator, targetRoles []string, current []model.UserPermissionOverride, overrides []perm.Override) []Violation {
	if slices.Contains(targetRoles, roleRen) {
		return []Violation{{Parameter: "user_id", Reason: problem.ReasonNotAllowedValue,
			Detail: "a ren holder always holds every permission and cannot be changed"}}
	}
	if op.Rank <= perm.Rank(targetRoles) {
		return []Violation{{Parameter: "user_id", Reason: problem.ReasonNotPermitted,
			Detail: "the user ranks at or above the caller"}}
	}
	if vs := overrideViolations("/overrides", overrides, func(p perm.Permission) bool {
		return perm.Can(targetRoles, p)
	}); len(vs) > 0 {
		return vs
	}
	stored := make(map[string]string, len(current))
	for _, r := range current {
		stored[r.Permission] = r.Effect
	}
	return possessionViolations("/overrides", stored, overrides, op.Holds)
}

func overrideViolations(at string, overrides []perm.Override, inBaseline func(perm.Permission) bool) []Violation {
	var vs []Violation
	seen := make(map[perm.Permission]bool, len(overrides))
	for j, ov := range overrides {
		pos := fmt.Sprintf("%s/%d", at, j)
		switch {
		case !perm.IsKnownPermission(ov.Permission):
			vs = append(vs, Violation{Pointer: pos + "/permission", Reason: problem.ReasonUnknownValue,
				Detail: "permission " + string(ov.Permission) + " is not in the catalog"})
		case seen[ov.Permission]:
			vs = append(vs, Violation{Pointer: pos + "/permission", Reason: problem.ReasonDuplicateItem,
				Detail: "permission " + string(ov.Permission) + " appears more than once"})
		case (ov.Effect == perm.EffectGrant) == inBaseline(ov.Permission):
			vs = append(vs, Violation{Pointer: pos + "/effect", Reason: problem.ReasonNotAllowedValue,
				Detail: ov.Effect + " of " + string(ov.Permission) + " changes nothing against the baseline"})
		}
		seen[ov.Permission] = true
	}
	return vs
}

// Possession is judged on the delta against the stored rows only, so a row an
// operator could not have written survives an edit that leaves it alone.
func possessionViolations(at string, stored map[string]string, overrides []perm.Override, holds func(perm.Permission) bool) []Violation {
	submitted := make(map[string]string, len(overrides))
	index := make(map[string]int, len(overrides))
	for j, ov := range overrides {
		submitted[string(ov.Permission)] = ov.Effect
		index[string(ov.Permission)] = j
	}
	var vs []Violation
	for _, p := range perm.Catalog() {
		key := string(p)
		if stored[key] == submitted[key] || holds(p) {
			continue
		}
		pos := at
		if j, ok := index[key]; ok {
			pos = fmt.Sprintf("%s/%d/permission", at, j)
		}
		vs = append(vs, Violation{Pointer: pos, Reason: problem.ReasonNotPermitted,
			Detail: "the caller does not hold " + key})
	}
	return vs
}

func roleEffects(rows []model.RolePermissionOverride, role string) map[string]string {
	m := make(map[string]string)
	for _, r := range rows {
		if r.Role == role {
			m[r.Permission] = r.Effect
		}
	}
	return m
}
