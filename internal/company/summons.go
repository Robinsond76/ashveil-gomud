package company

import (
	"strings"
	"sync"
)

// Phase 38b: summoned creatures (a Hierarch's Angel, a Demonologist's Demon)
// fight beside a company for one battle. They are not members of its
// record, but the combat code finds its company's fighters through
// LeaderAndKeyForInstance, so the live ones are remembered here (runtime
// only, never saved) and answered as members with a key of their own, which
// no formation holds.

// SummonKeyPrefix starts every summon's member key.
const SummonKeyPrefix = "summon:"

var (
	summonMu sync.RWMutex
	summoned = map[int]summonEntry{}
)

type summonEntry struct {
	leader int
	key    MemberKey
}

// SummonMemberKey is the member key of a leader's summon of a kind.
func SummonMemberKey(kind string) MemberKey { return MemberKey(SummonKeyPrefix + kind) }

// IsSummonKey reports whether key names a summon.
func IsSummonKey(key MemberKey) bool { return strings.HasPrefix(string(key), SummonKeyPrefix) }

// RegisterSummon records a live summon's mob instance as fighting for a
// leader's company.
func RegisterSummon(instanceID, leader int, kind string) MemberKey {
	summonMu.Lock()
	defer summonMu.Unlock()
	key := SummonMemberKey(kind)
	summoned[instanceID] = summonEntry{leader: leader, key: key}
	return key
}

// SummonOf is the leader and key of a live summon instance.
func SummonOf(instanceID int) (int, MemberKey, bool) {
	summonMu.RLock()
	defer summonMu.RUnlock()
	e, ok := summoned[instanceID]
	return e.leader, e.key, ok
}

// ForgetSummon drops a summon from the registry.
func ForgetSummon(instanceID int) {
	summonMu.Lock()
	defer summonMu.Unlock()
	delete(summoned, instanceID)
}

// SummonInstances lists every live summon's instance id.
func SummonInstances() []int {
	summonMu.RLock()
	defer summonMu.RUnlock()
	out := make([]int, 0, len(summoned))
	for id := range summoned {
		out = append(out, id)
	}
	return out
}

// ResetSummonsForTest forgets every summon.
func ResetSummonsForTest() {
	summonMu.Lock()
	defer summonMu.Unlock()
	clear(summoned)
}
