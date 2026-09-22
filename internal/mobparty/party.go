// Package mobparty groups hostile mobs present in a room into stable
// "parties" — mirroring the player's company — and auto-assigns each
// party's 3x3 company.Formation by a simple EHP/DPS role heuristic.
//
// A Party is never persisted: it is cheap to recompute and is assembled
// fresh every time a caller needs it, the same way characters.Aggro is
// never persisted today (see the Phase 11 design doc's prior-art check).
package mobparty

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/company"
)

// MaxPartySize mirrors the company cap (leader + MaxCompanions): a party's
// Formation has only 9 cells and the company side is capped at 5, so the
// enemy side uses the same cap rather than inventing a different one.
const MaxPartySize = company.MaxCompanions + 1

// MobSummary is the minimal, GoMud-free view of a mob this package needs.
// Callers (module/engine-layer code) adapt a live *mobs.Mob into this.
type MobSummary struct {
	InstanceId int
	Groups     []string
	EHP        float64
	DPS        float64
}

// Party is a stable-for-this-listing group of mobs with an auto-assigned
// formation. It is not persisted.
type Party struct {
	ID        string
	Members   []int
	Formation company.Formation
}

// Assemble groups mobs sharing a Groups tag (their first tag; a mob with no
// Groups tag is always its own solo party) into parties of up to
// MaxPartySize, then assigns each party's Formation front-to-back by
// descending EHP. Parties are returned in first-occurrence order of their
// grouping key, matching the input order of mobs.
func Assemble(mobs []MobSummary) []Party {
	type bucket struct {
		key     string
		members []MobSummary
	}

	var buckets []*bucket
	byKey := make(map[string]*bucket)

	for _, m := range mobs {
		key := groupKey(m)
		if key == "" {
			// Untagged: always a fresh solo party, never merged with
			// another untagged mob.
			buckets = append(buckets, &bucket{
				key:     fmt.Sprintf("solo:%d", m.InstanceId),
				members: []MobSummary{m},
			})
			continue
		}

		b, ok := byKey[key]
		if !ok {
			b = &bucket{key: key}
			byKey[key] = b
			buckets = append(buckets, b)
		}
		b.members = append(b.members, m)
	}

	parties := make([]Party, 0, len(buckets))
	for _, b := range buckets {
		for i, chunkMembers := range chunk(b.members, MaxPartySize) {
			id := b.key
			if len(b.members) > MaxPartySize {
				id = fmt.Sprintf("%s#%d", b.key, i+1)
			}
			parties = append(parties, buildParty(id, chunkMembers))
		}
	}

	return parties
}

// groupKey returns the grouping key for a mob: its first Groups tag,
// prefixed to avoid colliding with the "solo:<id>" key space, or "" if the
// mob has no Groups tag (always solo).
func groupKey(m MobSummary) string {
	if len(m.Groups) == 0 || m.Groups[0] == "" {
		return ""
	}
	return "group:" + m.Groups[0]
}

// chunk splits members into groups of at most size, preserving order —
// this is the "split by spawn order" rule for parties over the cap.
func chunk(members []MobSummary, size int) [][]MobSummary {
	var chunks [][]MobSummary
	for len(members) > 0 {
		n := size
		if n > len(members) {
			n = len(members)
		}
		chunks = append(chunks, members[:n])
		members = members[n:]
	}
	return chunks
}

// buildParty ranks members by descending EHP and fills the Formation
// front-row-first (up to 3), then mid, then back. A solo party occupies
// the front-row center cell rather than front-row-left, matching the
// design's explicit "even a single mob is a unit" placement.
func buildParty(id string, members []MobSummary) Party {
	ranked := make([]MobSummary, len(members))
	copy(ranked, members)
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].EHP > ranked[j].EHP
	})

	var f company.Formation
	memberIds := make([]int, len(ranked))
	for i, m := range ranked {
		memberIds[i] = m.InstanceId
		row, col := slotFor(i, len(ranked))
		// Every (row, col) here is in-bounds by construction (MaxPartySize
		// caps len(ranked) at 5, and slotFor never exceeds the 3x3 grid for
		// i < 5), and each memberKey is unique and unplaced, so Place
		// cannot fail; the error is intentionally discarded.
		_ = f.Place(memberKey(m.InstanceId), row, col)
	}

	return Party{ID: id, Members: memberIds, Formation: f}
}

// slotFor returns the formation cell for the i-th ranked (0-indexed, by
// descending EHP) member of a party of the given size.
func slotFor(i, size int) (row, col int) {
	if size == 1 {
		return 0, 1 // front-row center for a solo party
	}
	return i / company.FormationCols, i % company.FormationCols
}

func memberKey(instanceId int) company.MemberKey {
	return company.MemberKey(fmt.Sprintf("mob:%d", instanceId))
}
