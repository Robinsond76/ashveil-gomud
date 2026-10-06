package loot

import "sync"

// The spoils ledger: what a player's company took from a fight's deaths,
// noted as each foe falls and read once when the fight's summary prints
// ("Spoils: ..."). It is in memory, bounded, and never decides a drop.

const maxSpoils = 24

var (
	spoilsMu sync.Mutex
	spoils   = map[int][]string{}
)

// NoteSpoils records spoils lines (an item name, "12 gold") for a player.
func NoteSpoils(userID int, lines ...string) {
	if userID <= 0 || len(lines) == 0 {
		return
	}
	spoilsMu.Lock()
	defer spoilsMu.Unlock()
	got := append(spoils[userID], lines...)
	if len(got) > maxSpoils {
		got = got[len(got)-maxSpoils:]
	}
	spoils[userID] = got
}

// TakeSpoils returns and clears a player's noted spoils.
func TakeSpoils(userID int) []string {
	spoilsMu.Lock()
	defer spoilsMu.Unlock()
	got := spoils[userID]
	delete(spoils, userID)
	return got
}
