package perm

import (
	"slices"
	"sync/atomic"
)

const (
	EffectGrant  = "grant"
	EffectRevoke = "revoke"
)

const roleRen = "ren"

type Override struct {
	Permission Permission
	Effect     string
}

var catalog = append([]Permission{}, adminPerms...)

var catalogIndex = func() map[Permission]int {
	m := make(map[Permission]int, len(catalog))
	for i, p := range catalog {
		m[p] = i
	}
	return m
}()

var current atomic.Pointer[resolverT]

func init() {
	current.Store(buildResolver(nil))
}

func SetOverrides(overrides map[string][]Override) {
	current.Store(buildResolver(overrides))
}

func buildResolver(overrides map[string][]Override) *resolverT {
	roles := map[string]struct{}{roleRen: {}}
	for r := range Bundles {
		roles[r] = struct{}{}
	}
	for r := range overrides {
		roles[r] = struct{}{}
	}
	grants := make(map[string]map[Permission]struct{}, len(roles))
	for r := range roles {
		grants[r] = applyOverrides(r, overrides[r])
	}
	return &resolverT{grants: grants}
}

func applyOverrides(role string, overrides []Override) map[Permission]struct{} {
	if role == roleRen {
		return fullCatalogSet()
	}
	baseline := Bundles[role]
	set := make(map[Permission]struct{}, len(baseline)+len(overrides))
	for _, p := range baseline {
		set[p] = struct{}{}
	}
	for _, ov := range overrides {
		if _, known := catalogIndex[ov.Permission]; !known {
			continue
		}
		switch ov.Effect {
		case EffectGrant:
			set[ov.Permission] = struct{}{}
		case EffectRevoke:
			delete(set, ov.Permission)
		}
	}
	return set
}

func fullCatalogSet() map[Permission]struct{} {
	set := make(map[Permission]struct{}, len(catalog))
	for _, p := range catalog {
		set[p] = struct{}{}
	}
	return set
}

func orderedFrom(set map[Permission]struct{}) []Permission {
	out := make([]Permission, 0, len(set))
	for _, p := range catalog {
		if _, ok := set[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func Catalog() []Permission {
	return append([]Permission{}, catalog...)
}

func IsKnownPermission(p Permission) bool {
	_, ok := catalogIndex[p]
	return ok
}

func Baseline(role string) []Permission {
	set := make(map[Permission]struct{}, len(Bundles[role]))
	for _, p := range Bundles[role] {
		set[p] = struct{}{}
	}
	return orderedFrom(set)
}

func BaselineHas(role string, p Permission) bool {
	return slices.Contains(Bundles[role], p)
}

func EffectiveSet(role string, overrides []Override) []Permission {
	return orderedFrom(applyOverrides(role, overrides))
}
