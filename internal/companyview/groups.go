package companyview

import "sync"

// BuffGroup is a conditions group a module can claim buffs for.
type BuffGroup string

const (
	// GroupRest holds the rest tier buffs (Rested, Well Rested).
	GroupRest BuffGroup = "rest"
	// GroupSurvival holds survival and exposure buffs.
	GroupSurvival BuffGroup = "survival"
)

var (
	groupsMu   sync.RWMutex
	buffGroups = map[int]BuffGroup{}
)

// RegisterBuffGroup files buff IDs under a conditions group. Modules call it
// when they load their config; a later call for the same ID replaces it.
func RegisterBuffGroup(group BuffGroup, buffIDs ...int) {
	groupsMu.Lock()
	defer groupsMu.Unlock()
	for _, id := range buffIDs {
		if id > 0 {
			buffGroups[id] = group
		}
	}
}

// GroupOf is the group a buff was filed under; ok is false for any other
// buff.
func GroupOf(buffID int) (BuffGroup, bool) {
	groupsMu.RLock()
	defer groupsMu.RUnlock()
	g, ok := buffGroups[buffID]
	return g, ok
}
