// Package hexes is the Witch's rules (Phase 38a): which status each hex
// puts on a foe, how many foes a hex reaches at a level, whether a foe
// resists, how long a hex lasts, and the short immunity a landed hex leaves
// so no one is held in a loop. It is GoMud-free: callers supply levels,
// stat edges and rolls, and apply the status themselves.
//
// Statuses are counted in combat rounds, never game rounds (internal/status).
package hexes

import "fmt"

// Stat is the stat a target resists a hex with.
type Stat string

const (
	Mysticism Stat = "mysticism"
	Vitality  Stat = "vitality"
)

// Hex is one hexcraft spell and what it does.
type Hex struct {
	Spell string
	// Level is the character level the Witch learns it at (the archetype
	// config grants it; this is for the help and the tests).
	Level int
	// Buff is the status buff it puts on the foe; 0 for a hex that only
	// shakes morale (Dread Whisper).
	Buff int
	// Rounds is how many combat rounds the status lasts. 0 keeps the
	// buff's own length (a knockdown, a hobble, exposure, poison).
	Rounds int
	// Rounds20 replaces Rounds from level 20 (Binding Hex), when above 0.
	Rounds20 int
	// Resist is the stat the foe's Mysticism-or-Vitality is read from.
	Resist Stat
	// Row is a hex that covers only the primary target's row (Miasma).
	Row bool
	// NeedsHealer is a hex cast only at a foe that heals (Blight).
	NeedsHealer bool
	// Morale is a hex that triggers a morale check instead of a status.
	Morale bool
}

// Buff ids the table names, from internal/status and the shipped buffs.
const (
	buffPoisoned    = 13
	buffKnockedDown = 1102
	buffExposed     = 1104
	buffHobbled     = 1108
	buffAsleep      = 1109
	buffParalyzed   = 1110
	buffBlighted    = 1111
)

// All lists every hex in the order the Witch learns them.
var All = []Hex{
	{Spell: "slumber", Level: 1, Buff: buffAsleep, Rounds: 2, Resist: Mysticism},
	{Spell: "earthbind", Level: 3, Buff: buffKnockedDown, Resist: Mysticism},
	{Spell: "leaden", Level: 5, Buff: buffHobbled, Resist: Mysticism},
	{Spell: "miasma", Level: 7, Buff: buffPoisoned, Resist: Mysticism, Row: true},
	{Spell: "binding", Level: 10, Buff: buffParalyzed, Rounds: 1, Rounds20: 2, Resist: Vitality},
	{Spell: "frailty", Level: 12, Buff: buffExposed, Resist: Mysticism},
	{Spell: "dread", Level: 15, Resist: Mysticism, Morale: true},
	{Spell: "blight", Level: 18, Buff: buffBlighted, Rounds: 3, Resist: Mysticism, NeedsHealer: true},
}

// For is the hex a spell id names.
func For(spell string) (Hex, bool) {
	for _, h := range All {
		if h.Spell == spell {
			return h, true
		}
	}
	return Hex{}, false
}

// Reach is how many foes a hex affects at a character level: 1, then 2 at
// level 8, 3 at 16, 4 at 24, and the whole group (ReachGroup) from 30.
func Reach(level int) int {
	switch {
	case level >= 30:
		return ReachGroup
	case level >= 24:
		return 4
	case level >= 16:
		return 3
	case level >= 8:
		return 2
	}
	return 1
}

// ReachGroup is Reach's "the whole enemy group".
const ReachGroup = 1000

// Land chance bounds and base: even stats land 65 in 100; each full point of
// stat edge (-1..1) moves it 40; held to 25..90. A boss resists 25 more.
const (
	LandBase   = 65
	LandSwing  = 40
	LandMin    = 25
	LandMax    = 90
	BossResist = 25
)

// LandChance is the chance in 100 that a hex lands, from the stat edge of
// the Witch against the foe's resisting stat (internal/combat StatEdge).
func LandChance(edge float64, boss bool) int {
	c := LandBase + int(LandSwing*edge)
	if boss {
		c -= BossResist
	}
	return max(LandMin, min(LandMax, c))
}

// RoundsAt is how many combat rounds the hex's status lasts for a Witch of
// level, 0 to keep the buff's own length. A boss halves it, rounding down
// to at least 1 round; a hex of the buff's own length is halved too, in
// which case the buff's own rounds (ownRounds) are the length to halve.
func (h Hex) RoundsAt(level int, boss bool, ownRounds int) int {
	n := h.Rounds
	if h.Rounds20 > 0 && level >= 20 {
		n = h.Rounds20
	}
	if n == 0 {
		if !boss {
			return 0
		}
		n = ownRounds
	}
	if boss {
		n = max(1, n/2)
	}
	return n
}

// Triggers is the buff trigger count for a status of rounds: a status
// costs nothing before its first tick, so it counts one more than it lasts.
func Triggers(rounds int) int { return rounds + 1 }

// Immunity is the combat rounds a target stays immune to a hex's status
// after it ends (the "no lock loops" rule): with sleep and paralysis each
// lasting at most 2 rounds, no foe is held for more than half the rounds.
const Immunity = 2

// Ledger remembers who is immune to which status, in combat rounds. It is
// runtime only (a fight's state), never saved. Game loop only.
type Ledger struct {
	round int
	until map[string]int
}

// NewLedger is an empty ledger.
func NewLedger() *Ledger { return &Ledger{until: map[string]int{}} }

// key names an immunity. The statuses that take a foe's actions share one
// key, so a Slumber then a Binding Hex then an Earthbind can't chain a foe
// past the half-of-the-fight bound.
func key(holder string, buff int) string {
	switch buff {
	case buffAsleep, buffParalyzed, buffKnockedDown:
		buff = buffAsleep
	}
	return fmt.Sprintf("%s/%d", holder, buff)
}

// Immune reports whether holder is immune to buff now.
func (l *Ledger) Immune(holder string, buff int) bool {
	return l.until[key(holder, buff)] > l.round
}

// Land records a hex's status landing on holder for rounds (the status's
// own length): it is immune until that length and Immunity more rounds
// have passed.
func (l *Ledger) Land(holder string, buff, rounds int) {
	l.until[key(holder, buff)] = l.round + rounds + Immunity + 1
}

// Tick advances one combat round and forgets finished immunities.
func (l *Ledger) Tick() {
	l.round++
	for k, until := range l.until {
		if until <= l.round {
			delete(l.until, k)
		}
	}
}

// Default is the one ledger the game loop uses.
var Default = NewLedger()

// Reset forgets everything (a restart, or a test).
func (l *Ledger) Reset() {
	l.round = 0
	clear(l.until)
}
