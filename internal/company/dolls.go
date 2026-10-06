package company

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Phase 39d: a Doll Master's dolls. A doll is a mob its Master's record
// makes when a battle opens (internal/dolls) and takes away when it ends.
// It is not a member of the company's record, so no roster, food, morale,
// loyalty or experience rule touches it, and it takes no company slot; but it
// stands in a cell of the battle's formation and combat finds it through
// LeaderAndKeyForInstance, as it finds a companion. The live ones are
// remembered here (runtime only, never saved) and answered as members with a
// key of their own.

// DollKeyPrefix starts every doll's member key.
const DollKeyPrefix = "doll:"

// DollMemberKey is the member key of a Master's doll: the Master's own key and
// the doll's place in its list.
func DollMemberKey(owner MemberKey, index int) MemberKey {
	return MemberKey(fmt.Sprintf("%s%s:%d", DollKeyPrefix, owner, index))
}

// IsDollKey reports whether key names a doll.
func IsDollKey(key MemberKey) bool { return strings.HasPrefix(string(key), DollKeyPrefix) }

// DollEntry is one live doll.
type DollEntry struct {
	Instance int
	Leader   int
	Key      MemberKey
	Owner    MemberKey // its Master's key
	Index    int
}

var (
	dollMu sync.RWMutex
	dolls  = map[int]DollEntry{}
)

// RegisterDoll records a live doll's mob instance as fighting for a leader's
// company, standing by its Master.
func RegisterDoll(instanceID, leader int, owner MemberKey, index int) MemberKey {
	dollMu.Lock()
	defer dollMu.Unlock()
	key := DollMemberKey(owner, index)
	dolls[instanceID] = DollEntry{Instance: instanceID, Leader: leader, Key: key, Owner: owner, Index: index}
	return key
}

// DollOf is the entry of a live doll instance.
func DollOf(instanceID int) (DollEntry, bool) {
	dollMu.RLock()
	defer dollMu.RUnlock()
	e, ok := dolls[instanceID]
	return e, ok
}

// ForgetDoll drops a doll from the registry.
func ForgetDoll(instanceID int) {
	dollMu.Lock()
	defer dollMu.Unlock()
	delete(dolls, instanceID)
}

// DollInstances lists every live doll's instance id.
func DollInstances() []int {
	dollMu.RLock()
	defer dollMu.RUnlock()
	out := make([]int, 0, len(dolls))
	for id := range dolls {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// DollsOf lists a leader's live dolls, by owner key then place.
func DollsOf(leader int) []DollEntry {
	dollMu.RLock()
	defer dollMu.RUnlock()
	var out []DollEntry
	for _, e := range dolls {
		if e.Leader == leader {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		return out[i].Index < out[j].Index
	})
	return out
}

// DollInstanceFor is the live instance of a doll by its key.
func DollInstanceFor(leader int, key MemberKey) (int, bool) {
	for _, e := range DollsOf(leader) {
		if e.Key == key {
			return e.Instance, true
		}
	}
	return 0, false
}

// InstanceForKey is the live mob of any company member key: a companion's
// instance, or a live doll's.
func InstanceForKey(leader int, key MemberKey) (int, bool) {
	if IsDollKey(key) {
		return DollInstanceFor(leader, key)
	}
	cid, ok := CompanionIDFromMemberKey(key)
	if !ok {
		return 0, false
	}
	return InstanceFor(leader, cid)
}

// ResetDollsForTest forgets every doll.
func ResetDollsForTest() {
	dollMu.Lock()
	defer dollMu.Unlock()
	clear(dolls)
}

// overlayDolls puts a leader's live dolls into a copy of its formation: each
// stands in the free cell nearest in front of its Master (see dollCell). A
// leader with no company record gets the solo formation (the leader in the
// middle) so its doll has somewhere to stand. A doll with no free cell is left
// out, and fails open as a member outside any formation does.
func overlayDolls(leader int, f Formation, ok bool) (Formation, bool) {
	entries := DollsOf(leader)
	if len(entries) == 0 {
		return f, ok
	}
	if !ok {
		f = Formation{}
		f[1][1] = LeaderMemberKey
	}
	for _, e := range entries {
		if r, c, placed := f.dollCell(e.Owner); placed {
			f[r][c] = e.Key
		}
	}
	return f, true
}

// dollCell is the cell a Master's doll stands in: the nearest free cell in
// front of the Master in its own column, else the nearest free cell in the
// front-most row that has one, by column distance from the Master. A Master
// outside the formation stands in the middle.
func (f *Formation) dollCell(owner MemberKey) (row, col int, ok bool) {
	or, oc, placed := f.Find(owner)
	if !placed {
		or, oc = 1, 1
	}
	for r := or - 1; r >= 0; r-- {
		if f[r][oc] == "" {
			return r, oc, true
		}
	}
	for r := 0; r < FormationRows; r++ {
		best, bestDist := -1, FormationCols
		for c := 0; c < FormationCols; c++ {
			if f[r][c] != "" {
				continue
			}
			if d := abs(c - oc); d < bestDist {
				best, bestDist = c, d
			}
		}
		if best >= 0 {
			return r, best, true
		}
	}
	return 0, 0, false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
