// Package stormcraft is the Shaman's rules (Phase 39c): the three weathers it
// calls into a battle, how long they last, and the numbers they move. It is
// GoMud-free. A weather belongs to one battle (internal/battle) and never
// touches the world's weather (internal/climate), which every player shares;
// it ends with the battle, and nothing here advances a game round.
package stormcraft

// Kind is one weather.
type Kind string

const (
	None  Kind = ""
	Fog   Kind = "fog"   // dims the foes' ranged attacks and spells
	Chill Kind = "chill" // slows the foes' chants and sling shots
	Rain  Kind = "rain"  // feeds lightning
)

// Kinds are the weathers a Shaman can call.
var Kinds = []Kind{Fog, Chill, Rain}

// Rounds is how many combat rounds a call lasts (a status counts one more).
const Rounds = 3

// Numbers the weathers move.
const (
	// FogHit is the points of accuracy a fogbound foe's ranged attacks lose.
	FogHit = 10
	// FogSpellPct is how much weaker a fogbound foe's spells hit.
	FogSpellPct = 10
	// RainLightningPct is the extra damage Lightning deals in Rain.
	RainLightningPct = 50
	// ChainPct is the share of its damage a chained bolt deals the second foe.
	ChainPct = 50
)

// Spell ids of the lineage, by the weather each calls.
var spells = map[string]Kind{"callfog": Fog, "chillwind": Chill, "rain": Rain}

// KindOf is the weather a spell calls, if it calls one.
func KindOf(spellID string) (Kind, bool) {
	k, ok := spells[spellID]
	return k, ok
}

// Name is the weather as the battle's lines name it.
func (k Kind) Name() string {
	switch k {
	case Fog:
		return "fog"
	case Chill:
		return "chill wind"
	case Rain:
		return "rain"
	}
	return ""
}

// Effect is what the weather does, in a few words, for the battle screen.
func (k Kind) Effect() string {
	switch k {
	case Fog:
		return "foe ranged attacks and spells weaker"
	case Chill:
		return "foe chants and sling shots a round slower"
	case Rain:
		return "Lightning 50% stronger"
	}
	return ""
}

// EndLine is the line that tells a weather has passed.
func (k Kind) EndLine() string {
	switch k {
	case Fog:
		return "The fog thins and lifts."
	case Chill:
		return "The cold wind dies away."
	case Rain:
		return "The rain slackens and passes."
	}
	return ""
}

// LightningDamage is a Lightning bolt's damage in the weather.
func LightningDamage(dmg int, k Kind) int {
	if k == Rain {
		return dmg + dmg*RainLightningPct/100
	}
	return dmg
}

// FoggedSpell is a fogbound caster's spell damage.
func FoggedSpell(dmg int) int { return max(0, dmg*(100-FogSpellPct)/100) }

// Triggers is how many ticks a call counts, in a battle's weather and in the
// status it leaves on a foe: a status costs nothing before its first tick, so
// it counts one more than it lasts.
func Triggers(rounds int) int { return rounds + 1 }
