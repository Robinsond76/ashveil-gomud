package characters

import (
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// EliteRT is the battle state of the rogue and ranger elites (Phase 38c2).
// Like the rest of ClassRT it is runtime only and cleared when the fight
// ends. Some of it sits on the foe (a mark, its row, whether it has acted)
// because the blow code sees the defender only as a Character.
type EliteRT struct {
	// On a foe.
	Boss         bool   // the foe is a boss (Coup de Grace deals double instead of felling it)
	BackRow      bool   // the foe stands in its group's back row (a Marksman's Back-line eye)
	ActedRound   uint64 // the combat round it last acted in; 0 when it has not
	HuntedRound  uint64 // the combat round a Ravager last struck it (Hunt Down)
	HuntedBy     *ClassRT
	WoundedBy    *ClassRT // a Ravager that wounded it (Harrow)
	WoundedRound uint64

	// On the holder.
	DeathMark     *ClassRT // the foe a Nightblade has marked (its class state, for the blow code)
	MarkedFoe     int      // that foe's mob instance id
	MarkSet       bool     // it has chosen its first target this battle
	StepRound     uint64   // the combat round its Shadowstep was last used in (0: never)
	Stepping      bool     // a Shadowstep is in force for this round's blow (extended reach)
	SpreeRound    uint64   // the combat round Killing Spree last gave its meter
	OpensUsed     int      // Opening Strikes made on foes that had not acted yet
	Vanished      bool     // Vanish has been spent this battle
	VanishEvade   int      // Evasion Vanish gives now
	VanishRound   uint64   // the combat round Vanish began in
	RiposteRound  uint64   // the combat round of its last riposte
	Ripostes      int      // ripostes made in that round
	PerfectUsed   bool     // Perfect Parry has been spent this battle
	ShotUsed      bool     // Perfect Shot has been spent this battle
	ShotNow       bool     // the Aimed Shot being resolved is the Perfect Shot
	NockRound     uint64   // the combat round Second Nock last fired
	Watching      int      // Overwatch shots held (this turn's hold)
	WatchDebt     bool     // the second Overwatch shot spends the next turn too
	WatchFired    int      // Overwatch shots fired this round
	ChantRound    uint64   // the combat round a foe began a chant in
	ChantAnswered bool     // an overwatch shot already answered that chant
	HuntPenalty   int      // flee-band points a Ravager's hunt took off this foe
	HarrowPts     int      // Attack a harrowed group has gained
	Spoil         int      // percent damage the next blow it strikes keeps (a spoiled blow)
}

// shooting reports whether the character wields a shooting weapon.
func (c *Character) shooting() bool {
	return c.Equipment.Weapon.ItemId > 0 && c.Equipment.Weapon.GetSpec().Subtype == items.Shooting
}

// Shooting reports whether the character's weapon is a shooting one.
func (c *Character) Shooting() bool { return c.shooting() }

// EliteCrit is the critical chance points a rogue or ranger elite adds to a
// blow of its own against a defender: a Marksman's Called Shot with a bow,
// and a Nightblade's hunt of its marked foe.
func (c *Character) EliteCrit(def *Character) int {
	fx := c.ClassEffects()
	if fx == nil {
		return 0
	}
	pts := 0
	if c.shooting() {
		pts += fx.Int(classes.RangedCrit)
	}
	if c.RT != nil && c.RT.DeathMark != nil && def != nil && def.RT == c.RT.DeathMark {
		pts += fx.Int(classes.DeathCrit)
	}
	return pts
}

// ReachOverride reports whether a Shadowstep is in force: the character
// strikes as if it had extended reach for this round's blow.
func (c *Character) ReachOverride() bool { return c != nil && c.RT != nil && c.RT.Stepping }
