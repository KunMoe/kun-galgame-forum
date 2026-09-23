package dlsite

import (
	_ "embed"
	"strconv"
	"strings"
	"sync"
)

// verified.tsv is the infra-audited work_id → DLsite workno whitelist
// (`work_id \t workno \t score \t matched_field`, header row first). Its keys
// were forum gids until the G0 renumber rewrote them through
// galgame_renumber_2026; the two pages folded into work 4082 carried different
// worknos, so that work now has no whitelist entry.
//
// WHY A VENDORED SNAPSHOT. The canonical source is the catalog's refs.dlsite,
// which arrives on the galgame read faces and updates itself. But refs covers
// 3,701 of kungal's games while the DLsite URLs users already contributed to
// galgame_link cover 5,077 — so ~1,400 games would show no purchase link despite
// having a perfectly good DLsite id sitting in their links.
//
// Those raw links cannot be trusted blindly: they are user/VNDB-contributed, and
// infra's audit found real mismatches — links pointing at a budget re-release
// (まいてつ -pure station- → 普及版), another platform (Kanon → 【Android版】), or a
// multi-title compilation (沙耶の唄 → Nitro The Best! Vol.2). Sending a buyer to
// the wrong product is worse than showing no button, so kungal does NOT parse
// galgame_link itself. It ships only the pairs infra verified by title match
// (4,320 of 5,077); the 757 suspicious ones (429 mirror-coverage gaps, 328 title
// mismatches) are deliberately absent.
//
// This is a BRIDGE, not a permanent source. refs.dlsite always wins where it
// exists; the whitelist only fills gaps, and it does not grow — a game that gets
// a DLsite link after this snapshot is picked up when refs covers it. Once refs
// covers the whitelist, this file and its lookup can be deleted outright.
//
//go:embed verified.tsv
var verifiedTSV string

var (
	verifiedOnce sync.Once
	verifiedMap  map[int]string
)

func VerifiedWorkno(workID int) string {
	verifiedOnce.Do(loadVerified)
	return verifiedMap[workID]
}

func VerifiedCount() int {
	verifiedOnce.Do(loadVerified)
	return len(verifiedMap)
}

var pinnedWorkno = map[int]string{
	4123: "RJ090411",
}

func loadVerified() {
	pairs := make(map[int][]string, 4500)
	for line := range strings.SplitSeq(strings.TrimSpace(verifiedTSV), "\n") {
		cols := strings.Split(strings.TrimSpace(line), "\t")
		if len(cols) < 2 {
			continue
		}
		id, err := strconv.Atoi(cols[0])
		if err != nil || id <= 0 || !ValidWorkno(cols[1]) {
			continue
		}
		pairs[id] = append(pairs[id], cols[1])
	}
	verifiedMap = resolveVerified(pairs)
}

func resolveVerified(pairs map[int][]string) map[int]string {
	out := make(map[int]string, len(pairs))
	for id, worknos := range pairs {
		switch {
		case len(worknos) == 1:
			out[id] = worknos[0]
		default:
			if pinned, ok := pinnedWorkno[id]; ok {
				out[id] = pinned
			}
		}
	}
	return out
}
