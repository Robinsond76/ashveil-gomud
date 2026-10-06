package classes

import (
	"errors"
	"fmt"
	"slices"
)

// Talent is a small passive chosen at a talent level (5, 15, 25, 35, 45 and
// 55). Add is added to the character's effects for each time it is chosen,
// up to Max.
type Talent struct {
	ID   string
	Name string
	Text string
	Max  int
	Add  Effects
	// Elite marks a talent only an elite character of the lineage may take,
	// from level EliteTalentLevel (Phase 38c1).
	Elite bool
}

// EliteTalentLevel is the first talent level an elite talent is offered at.
const EliteTalentLevel = 35

// MaxPicks is how many times one talent may be taken unless it says
// otherwise: six talent levels over five choices a lineage.
const MaxPicks = 2

var (
	talentByID = map[string]Talent{}
	talentsFor = map[string][]string{} // lineage -> talent ids in list order
)

func defineTalent(t Talent) Talent {
	if t.Max <= 0 {
		t.Max = MaxPicks
	}
	if _, dup := talentByID[t.ID]; dup {
		panic("classes: duplicate talent " + t.ID)
	}
	talentByID[t.ID] = t
	return t
}

func offer(lineageID string, ts ...Talent) {
	for _, t := range ts {
		talentsFor[lineageID] = append(talentsFor[lineageID], t.ID)
	}
}

// defineEliteTalent defines a talent taken once, offered only to a
// lineage's elites from level 35. 38c2 and 38c3 add their lineages' with
// offerElite.
func defineEliteTalent(t Talent) Talent {
	t.Elite, t.Max = true, 1
	return defineTalent(t)
}

func offerElite(lineageID string, ts ...Talent) { offer(lineageID, ts...) }

// TalentByID returns a talent by id.
func TalentByID(id string) (Talent, bool) {
	t, ok := talentByID[normalize(id)]
	return t, ok
}

// TalentsFor lists the talents any character of a lineage may choose, in
// list order (the elite talents are MenuFor's).
func TalentsFor(lineageID string) []Talent {
	var out []Talent
	for _, id := range talentsFor[normalize(lineageID)] {
		if t := talentByID[id]; !t.Elite {
			out = append(out, t)
		}
	}
	return out
}

// EliteTalentsFor lists a lineage's elite talents.
func EliteTalentsFor(lineageID string) []Talent {
	var out []Talent
	for _, id := range talentsFor[normalize(lineageID)] {
		if t := talentByID[id]; t.Elite {
			out = append(out, t)
		}
	}
	return out
}

// eliteOpen reports whether an elite talent may be taken: the character's
// class is an elite one and the level is the elite talent level or more.
func eliteOpen(classID string, level int) bool {
	c, ok := Get(classID)
	return ok && c.Tier == TierElite && level >= EliteTalentLevel
}

// MenuFor is every talent a character may be offered: its lineage's, and
// its lineage's elite talents once an elite of level 35 or more.
func MenuFor(lineageID, classID string, level int) []Talent {
	out := TalentsFor(lineageID)
	if eliteOpen(classID, level) {
		out = append(out, EliteTalentsFor(lineageID)...)
	}
	return out
}

// TalentSlots is how many talents a character at a level has earned.
func TalentSlots(level int) int {
	n := 0
	for _, l := range TalentLevels {
		if level >= l {
			n++
		}
	}
	return n
}

// NextTalentLevel is the level the next talent comes at, if any is left.
func NextTalentLevel(level int) (int, bool) {
	for _, l := range TalentLevels {
		if level < l {
			return l, true
		}
	}
	return 0, false
}

// ActiveTalents are the talents in force at a level: the first picks, as
// many as the level has earned. A level lost to death switches the newest
// off until it is regained; a pick is never lost.
func ActiveTalents(level int, picked []string) []string {
	n := TalentSlots(level)
	if n >= len(picked) {
		return picked
	}
	return picked[:n]
}

// TalentsOwed is how many talents a character at a level has yet to
// choose.
func TalentsOwed(level int, picked []string) int {
	return max(0, TalentSlots(level)-len(picked))
}

// Count is how many times a talent was picked.
func Count(picked []string, id string) int {
	id = normalize(id)
	n := 0
	for _, p := range picked {
		if p == id {
			n++
		}
	}
	return n
}

// Errors a talent pick can fail with; each is player-facing.
var (
	ErrNoTalentOwed  = errors.New("no talent is owed yet")
	ErrUnknownTalent = errors.New("no such talent")
	ErrTalentMaxed   = errors.New("that talent is at its limit")
	ErrEliteTalent   = errors.New("that talent is for elite characters from level 35")
)

// CanPick reports whether a character of a lineage and class, at a level,
// with the talents it has, may take a talent now.
func CanPick(lineageID, classID string, picked []string, level int, id string) error {
	id = normalize(id)
	if !slices.Contains(talentsFor[normalize(lineageID)], id) {
		return fmt.Errorf("%w: %q", ErrUnknownTalent, id)
	}
	if talentByID[id].Elite && !eliteOpen(classID, level) {
		return ErrEliteTalent
	}
	if TalentsOwed(level, picked) < 1 {
		return ErrNoTalentOwed
	}
	if t := talentByID[id]; Count(picked, id) >= t.Max {
		return ErrTalentMaxed
	}
	return nil
}

// The talents. Shared ones are defined once and offered to several
// lineages.
var (
	tMendingHands = defineTalent(Talent{ID: "mending-hands", Name: "Mending Hands", Text: "+10% healing", Add: Effects{HealPct: 10}})
	tSteadfast    = defineTalent(Talent{ID: "steadfast-prayer", Name: "Steadfast Prayer", Text: "blows break your chant 25% less often", Add: Effects{ChantBreak: 25}})
	tDeepWell     = defineTalent(Talent{ID: "deep-well", Name: "Deep Well", Text: "+10% maximum mana", Add: Effects{ManaPct: 10}})
	tSanctuary    = defineTalent(Talent{ID: "sanctuary", Name: "Sanctuary", Text: "+3 Evasion while chanting", Add: Effects{ChantEvade: 3}})
	tGentleRest   = defineTalent(Talent{ID: "gentle-rest", Name: "Gentle Rest", Text: "after-battle patching costs 15% less mana", Add: Effects{PatchCost: 15}})

	tToughness  = defineTalent(Talent{ID: "toughness", Name: "Toughness", Text: "+8% maximum health", Add: Effects{HealthPct: 8}})
	tHeavyHands = defineTalent(Talent{ID: "heavy-hands", Name: "Heavy Hands", Text: "+1 damage on every blow", Add: Effects{Damage: 1}})
	tKeenEdge   = defineTalent(Talent{ID: "keen-edge", Name: "Keen Edge", Text: "+2 Attack", Add: Effects{Attack: 2}})
	tFootwork   = defineTalent(Talent{ID: "footwork", Name: "Footwork", Text: "+2 Evasion", Add: Effects{Evasion: 2}})
	tTackleDril = defineTalent(Talent{ID: "tackle-drill", Name: "Tackle Drill", Text: "Tackle is ready a round sooner", Max: 1, Add: Effects{TackleCD: 1}})

	tDeepCuts    = defineTalent(Talent{ID: "deep-cuts", Name: "Deep Cuts", Text: "Opening Strike deals +2 damage", Add: Effects{OpenBonus: 2}})
	tPatientHand = defineTalent(Talent{ID: "patient-hand", Name: "Patient Hand", Text: "Opening Strike is ready a round sooner", Max: 1, Add: Effects{OpenCD: 1}})

	tSteadyAim = defineTalent(Talent{ID: "steady-aim", Name: "Steady Aim", Text: "Aimed Shot deals +2 damage", Add: Effects{AimBonus: 2}})
	tQuickDraw = defineTalent(Talent{ID: "quick-draw", Name: "Quick Draw", Text: "Aimed Shot is ready a round sooner", Max: 1, Add: Effects{AimCD: 1}})

	tFocus  = defineTalent(Talent{ID: "focus", Name: "Focus", Text: "+8% spell damage", Add: Effects{SpellPct: 8}})
	tThrift = defineTalent(Talent{ID: "thrift", Name: "Thrift", Text: "spells cost 10% less mana", Add: Effects{SpellCost: 10}})

	tLongWeather = defineTalent(Talent{ID: "long-weather", Name: "Long Weather", Text: "its weather calls last a round longer", Add: Effects{WeatherLong: 1}})
	tSharpEye    = defineTalent(Talent{ID: "sharp-eye", Name: "Sharp Eye", Text: "+3% critical chance on every blow", Add: Effects{Crit: 3}})

	tPotentHex = defineTalent(Talent{ID: "potent-hex", Name: "Potent Hex", Text: "+5 to a hex's chance to land", Add: Effects{HexLand: 5}})
)

// Elite talents (Phase 38c1: warrior and cleric; 38c2 adds rogue and ranger,
// 38c3 wizard and witch). Taken once each, from level 35, elites only; they
// stack with a base talent of the same kind.
var (
	tIronHide     = defineEliteTalent(Talent{ID: "iron-hide", Name: "Iron Hide", Text: "+5 armor", Add: Effects{Armor: 5}})
	tSecondWind   = defineEliteTalent(Talent{ID: "second-wind", Name: "Second Wind", Text: "once a battle, at the start of its turn below 25% health, heals 10% of its maximum health (the turn is still taken)", Add: Effects{SecondWind: 10}})
	tVeteransEdge = defineEliteTalent(Talent{ID: "veterans-edge", Name: "Veteran's Edge", Text: "+3 Attack", Add: Effects{Attack: 3}})

	tFontOfGrace    = defineEliteTalent(Talent{ID: "font-of-grace", Name: "Font of Grace", Text: "+10% maximum mana", Add: Effects{ManaPct: 10}})
	tRadiantHealing = defineEliteTalent(Talent{ID: "radiant-healing", Name: "Radiant Healing", Text: "+10% healing", Add: Effects{HealPct: 10}})
	tUnshaken       = defineEliteTalent(Talent{ID: "unshaken", Name: "Unshaken", Text: "blows break your chant 25% less often", Add: Effects{ChantBreak: 25}})

	// Phase 38c3: the wizard and witch elite talents. Deep Reserves is
	// offered to both lineages.
	tDeepReserves = defineEliteTalent(Talent{ID: "deep-reserves", Name: "Deep Reserves", Text: "+10% maximum mana", Add: Effects{ManaPct: 10}})
	tFocusedWill  = defineEliteTalent(Talent{ID: "focused-will", Name: "Focused Will", Text: "blows break your chant 25% less often", Add: Effects{ChantBreak: 25}})
	tSpellEdge    = defineEliteTalent(Talent{ID: "spell-edge", Name: "Spell Edge", Text: "+10% spell damage", Add: Effects{SpellPct: 10}})
	tIronWill     = defineEliteTalent(Talent{ID: "iron-will", Name: "Iron Will", Text: "blows break your chant 25% less often", Add: Effects{ChantBreak: 25}})
	tHexReach     = defineEliteTalent(Talent{ID: "hex-reach", Name: "Hex Reach", Text: "+5 to a hex's chance to land (never above 90)", Add: Effects{HexLand: 5}})
)

func init() {
	offerElite("warrior", tIronHide, tSecondWind, tVeteransEdge)
	offerElite("cleric", tFontOfGrace, tRadiantHealing, tUnshaken)
	offerElite("wizard", tDeepReserves, tFocusedWill, tSpellEdge)
	offerElite("witch", tDeepReserves, tIronWill, tHexReach)
	offer("cleric", tMendingHands, tSteadfast, tDeepWell, tSanctuary, tGentleRest)
	offer("warrior", tToughness, tHeavyHands, tKeenEdge, tFootwork, tTackleDril)
	offer("rogue", tKeenEdge, tFootwork, tDeepCuts, tPatientHand, tHeavyHands)
	offer("ranger", tKeenEdge, tFootwork, tSteadyAim, tQuickDraw, tToughness)
	offer("wizard", tDeepWell, tFocus, tSteadfast, tThrift, tSanctuary)
	offer("samurai", tKeenEdge, tFootwork, tHeavyHands, tToughness, tSharpEye)
	offer("shaman", tDeepWell, tFocus, tLongWeather, tThrift, tSanctuary)
	offer("witch", tDeepWell, tSteadfast, tThrift, tPotentHex, tSanctuary)
}
