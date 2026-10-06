package company

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Phase 39e: a Beast Tamer's bonded beast. Like a doll it is a mob its
// Tamer's record makes when a battle opens (internal/beasts) and takes away
// when it ends, not a member of the company's record: no roster, food,
// morale, loyalty or experience rule touches it and it takes no company slot.
// It stands in a cell of the battle's formation, and combat finds it through
// LeaderAndKeyForInstance as it finds a companion. Unlike a doll it takes its
// own turn. The live ones are remembered here (runtime only, never saved).

// BeastKeyPrefix starts every beast's member key.
const BeastKeyPrefix = "beast:"

// BeastMemberKey is the member key of a Tamer's beast: the Tamer's own key.
func BeastMemberKey(owner MemberKey) MemberKey {
	return MemberKey(fmt.Sprintf("%s%s", BeastKeyPrefix, owner))
}

// IsBeastKey reports whether key names a beast.
func IsBeastKey(key MemberKey) bool { return strings.HasPrefix(string(key), BeastKeyPrefix) }

// BeastEntry is one live beast.
type BeastEntry struct {
	Instance int
	Leader   int
	Key      MemberKey
	Owner    MemberKey // its Tamer's key
}

var (
	beastMu sync.RWMutex
	beasts  = map[int]BeastEntry{}
)

// RegisterBeast records a live beast's mob instance as fighting for a
// leader's company, standing by its Tamer.
func RegisterBeast(instanceID, leader int, owner MemberKey) MemberKey {
	beastMu.Lock()
	defer beastMu.Unlock()
	key := BeastMemberKey(owner)
	beasts[instanceID] = BeastEntry{Instance: instanceID, Leader: leader, Key: key, Owner: owner}
	return key
}

// BeastOf is the entry of a live beast instance.
func BeastOf(instanceID int) (BeastEntry, bool) {
	beastMu.RLock()
	defer beastMu.RUnlock()
	e, ok := beasts[instanceID]
	return e, ok
}

// ForgetBeast drops a beast from the registry.
func ForgetBeast(instanceID int) {
	beastMu.Lock()
	defer beastMu.Unlock()
	delete(beasts, instanceID)
}

// BeastInstances lists every live beast's instance id.
func BeastInstances() []int {
	beastMu.RLock()
	defer beastMu.RUnlock()
	out := make([]int, 0, len(beasts))
	for id := range beasts {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// BeastsOf lists a leader's live beasts, by owner key.
func BeastsOf(leader int) []BeastEntry {
	beastMu.RLock()
	defer beastMu.RUnlock()
	var out []BeastEntry
	for _, e := range beasts {
		if e.Leader == leader {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Owner < out[j].Owner })
	return out
}

// BeastInstanceFor is the live instance of a beast by its key.
func BeastInstanceFor(leader int, key MemberKey) (int, bool) {
	for _, e := range BeastsOf(leader) {
		if e.Key == key {
			return e.Instance, true
		}
	}
	return 0, false
}

// ResetBeastsForTest forgets every beast.
func ResetBeastsForTest() {
	beastMu.Lock()
	defer beastMu.Unlock()
	clear(beasts)
}

// overlayBeasts puts a leader's live beasts into a copy of its formation:
// each stands in the free cell nearest in front of its Tamer (see dollCell).
// A leader with no company record gets the solo formation. A beast with no
// free cell is left out, and fails open as a member outside any formation
// does.
func overlayBeasts(leader int, f Formation, ok bool) (Formation, bool) {
	entries := BeastsOf(leader)
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
