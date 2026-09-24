package perm

import (
	"slices"
	"sync/atomic"
)

type userPerms struct {
	grants  map[Permission]struct{}
	revokes map[Permission]struct{}
}

type userOverrideTable struct {
	users map[int]*userPerms
}

func (t *userOverrideTable) forUID(uid int) *userPerms {
	if t == nil {
		return nil
	}
	return t.users[uid]
}

var userCurrent atomic.Pointer[userOverrideTable]

func init() {
	userCurrent.Store(&userOverrideTable{users: map[int]*userPerms{}})
}

func SetUserOverrides(overrides map[int][]Override) {
	tbl := &userOverrideTable{users: make(map[int]*userPerms, len(overrides))}
	for uid, ovs := range overrides {
		if uid <= 0 {
			continue
		}
		up := &userPerms{grants: map[Permission]struct{}{}, revokes: map[Permission]struct{}{}}
		for _, ov := range ovs {
			if _, known := catalogIndex[ov.Permission]; !known {
				continue
			}
			switch ov.Effect {
			case EffectGrant:
				up.grants[ov.Permission] = struct{}{}
			case EffectRevoke:
				up.revokes[ov.Permission] = struct{}{}
			}
		}
		if len(up.grants) == 0 && len(up.revokes) == 0 {
			continue
		}
		tbl.users[uid] = up
	}
	userCurrent.Store(tbl)
}

func CanUser(uid int, roles []string, p Permission) bool {
	if uid <= 0 {
		return Can(roles, p)
	}
	if slices.Contains(roles, roleRen) {
		return IsKnownPermission(p)
	}
	if up := userCurrent.Load().forUID(uid); up != nil {
		if _, revoked := up.revokes[p]; revoked {
			return false
		}
		if _, granted := up.grants[p]; granted {
			return true
		}
	}
	return Can(roles, p)
}
