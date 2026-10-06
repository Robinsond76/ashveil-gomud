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
}

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

// TalentByID returns a talent by id.
func TalentByID(id string) (Talent, bool) {
	t, ok := talentByID[normalize(id)]
	return t, ok
}

// TalentsFor lists the talents a lineage may choose, in list order.
func TalentsFor(lineageID string) []Talent {
	var out []Talent
	for _, id := range talentsFor[normalize(lineageID)] {
		out = append(out, talentByID[id])
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
)

// CanPick reports whether a character of a lineage, at a level, with the
// talents it has, may take a talent now.
func CanPick(lineageID string, picked []string, level int, id string) error {
	id = normalize(id)
	if !slices.Contains(talentsFor[normalize(lineageID)], id) {
		return fmt.Errorf("%w: %q", ErrUnknownTalent, id)
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

	tPotentHex = defineTalent(Talent{ID: "potent-hex", Name: "Potent Hex", Text: "+5 to a hex's chance to land", Add: Effects{HexLand: 5}})
)

func init() {
	offer("cleric", tMendingHands, tSteadfast, tDeepWell, tSanctuary, tGentleRest)
	offer("warrior", tToughness, tHeavyHands, tKeenEdge, tFootwork, tTackleDril)
	offer("rogue", tKeenEdge, tFootwork, tDeepCuts, tPatientHand, tHeavyHands)
	offer("ranger", tKeenEdge, tFootwork, tSteadyAim, tQuickDraw, tToughness)
	offer("wizard", tDeepWell, tFocus, tSteadfast, tThrift, tSanctuary)
	offer("witch", tDeepWell, tSteadfast, tThrift, tPotentHex, tSanctuary)
}
