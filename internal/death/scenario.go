package death

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 53: defeat scenarios. A new death may end in a scenario instead of
// the church: the company is rescued, captured, left for dead, or robbed.
// Scenarios are data (the module's Scenarios config), keyed by what beat the
// company and where, so a replacement world can write its own. This file is
// the pure part: the table, the roll, and the pack split. It never touches
// the world clock.

// ScenarioKind is what happens to a defeated company.
type ScenarioKind string

const (
	// Rescued: a traveller carries the company to the nearest settlement;
	// members wake Hungry and Exhausted.
	Rescued ScenarioKind = "rescued"
	// Captured: the company wakes in the scenario's capture room, bound for
	// a moment, its pack held in a chest the captors guard.
	Captured ScenarioKind = "captured"
	// LeftForDead: the company wakes where it fell, the foes gone, each
	// member with a lasting wound.
	LeftForDead ScenarioKind = "left-for-dead"
	// Robbed: the company wakes where it fell, the foes gone, a share of
	// its gold and loose goods taken for good.
	Robbed ScenarioKind = "robbed"
)

// Character MiscData keys for a defeat. They are saved in the user file
// with the items they concern.
const (
	// ScenarioKey holds the ID of the scenario a death was claimed by,
	// from the claim until the company has woken. It is what lets a
	// restart mid-defeat resume the same scenario instead of rolling again.
	ScenarioKey = "defeat-scenario"
	// SeizedGoldKey is the gold the captors hold with the pack.
	SeizedGoldKey = "seized-gold"
	// SeizedRoomKey is the capture room holding the pack.
	SeizedRoomKey = "seized-room"
	// SeizedByKey is the ID of the capture scenario holding the pack, so
	// its guards can be posted again.
	SeizedByKey = "seized-by"
	// SeizedGuardsKey counts the capture's guards not yet defeated. Guards
	// are not saved with the room, so a restart or an unloaded room posts
	// this many again; the chest opens only when it reaches zero.
	SeizedGuardsKey = "seized-guards"
)

// Killer is what landed a defeated leader's killing blow, as the engine's
// death path knows it. MobID 0 means unknown (a bleed, poison, hunger).
type Killer struct {
	MobID      int
	InstanceID int
	// Protected is true when the engine spares this death its penalties
	// (the death protection levels, or perma-gear): no scenario may then
	// take goods or gold, so captures and robberies are never rolled.
	Protected bool
}

// TakesGoods reports whether a scenario kind takes or holds goods or gold.
func (k ScenarioKind) TakesGoods() bool { return k == Captured || k == Robbed }

// Scenario is one row of the defeat table.
type Scenario struct {
	ID   string
	Kind ScenarioKind
	// Weight is its share among the scenarios that fit; below 1 counts as 1.
	Weight int
	// Foes limits it to defeats by these kinds of foe: a race name or a mob
	// group, lower case. Empty fits any foe, including an unknown killer.
	Foes []string
	// Zones limits it to defeats in these zones. Empty fits any zone.
	Zones []string
	// Text is what the leader reads on waking; empty uses the kind's own.
	Text string

	// Captured: the capture room and the guards posted in it.
	Room       int
	GuardMob   int
	GuardCount int
	GuardLevel int

	// GoldLossPct is the share of gold gone for good (Robbed) or never
	// seen again by the captors (Captured), in percent.
	GoldLossPct int
	// ItemLossPct is the share of loose goods a robbery takes, in percent,
	// at most ItemLossMax items.
	ItemLossPct int
	ItemLossMax int
	// WoundPct is a left-for-dead wound's size, in percent of the member's
	// maximum health.
	WoundPct int
	// Hunger and Fatigue are the values a rescued member wakes at or
	// below (0..100): Hungry is 26..50, Exhausted 1..25.
	Hunger  int
	Fatigue int
}

// Valid reports whether the scenario can be used, and why not.
func (s Scenario) Valid() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("death: scenario has no id")
	}
	switch s.Kind {
	case Rescued, LeftForDead, Robbed:
	case Captured:
		if s.Room < 1 {
			return fmt.Errorf("death: captured scenario %q needs a Room", s.ID)
		}
	default:
		return fmt.Errorf("death: scenario %q has unknown kind %q", s.ID, s.Kind)
	}
	if s.GoldLossPct < 0 || s.GoldLossPct > 100 || s.ItemLossPct < 0 || s.ItemLossPct > 100 || s.WoundPct < 0 || s.WoundPct > 50 {
		return fmt.Errorf("death: scenario %q has a percentage out of range", s.ID)
	}
	return nil
}

func listHas(list []string, want ...string) bool {
	for _, have := range list {
		have = strings.ToLower(strings.TrimSpace(have))
		for _, w := range want {
			if w != "" && have == strings.ToLower(w) {
				return true
			}
		}
	}
	return false
}

// Fits reports whether the scenario applies to a defeat in zone by a foe of
// the given race and groups. An unknown killer (race "" and no groups) fits
// only scenarios that name no foes.
func (s Scenario) Fits(zone, race string, groups []string) bool {
	if len(s.Zones) > 0 && !listHas(s.Zones, zone) {
		return false
	}
	if len(s.Foes) == 0 {
		return true
	}
	if listHas(s.Foes, race) {
		return true
	}
	return listHas(s.Foes, groups...)
}

// Pick rolls a scenario from those that fit. roll returns a whole number in
// [0, n). ok is false when none fit, and the church stands.
func Pick(table []Scenario, zone, race string, groups []string, roll func(n int) int) (Scenario, bool) {
	var fit []Scenario
	total := 0
	for _, s := range table {
		if s.Fits(zone, race, groups) {
			fit = append(fit, s)
			total += max(1, s.Weight)
		}
	}
	if len(fit) == 0 {
		return Scenario{}, false
	}
	n := 0
	if roll != nil && total > 1 {
		n = roll(total)
	}
	n = max(0, min(n, total-1))
	for _, s := range fit {
		n -= max(1, s.Weight)
		if n < 0 {
			return s, true
		}
	}
	return fit[len(fit)-1], true
}

// ByID finds a scenario in the table.
func ByID(table []Scenario, id string) (Scenario, bool) {
	for _, s := range table {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

// Camp gear is item IDs 45 to 50 (the 40a4 camp plumbing): robbers leave it.
const (
	campGearFirst = 45
	campGearLast  = 50
)

// Robbable reports whether robbers may take an item: never a quest token, a
// key, or the camp gear (the 40a4 theft rules).
func Robbable(itemID int) bool {
	if itemID >= campGearFirst && itemID <= campGearLast {
		return false
	}
	spec := items.GetItemSpec(itemID)
	if spec == nil {
		return false
	}
	return spec.QuestToken == "" && spec.Type != items.Key
}

// GoldShare is pct percent of gold, rounded down, never more than gold.
func GoldShare(gold, pct int) int {
	if gold <= 0 || pct <= 0 {
		return 0
	}
	return min(gold, gold*min(pct, 100)/100)
}

// Rob splits a pack into what stays and what robbers take: pct percent of
// the robbable items, rounded up, at least one when any can be taken, at
// most limit (0 for none). robbable says which items may go (nil: Robbable).
// Nothing is duplicated: kept plus taken is the pack. roll returns a whole
// number in [0, n).
func Rob(pack []items.Item, pct, limit int, roll func(n int) int, robbable func(itemID int) bool) (kept, taken []items.Item) {
	if robbable == nil {
		robbable = Robbable
	}
	var pool []int
	for i, itm := range pack {
		if robbable(itm.ItemId) {
			pool = append(pool, i)
		}
	}
	take := 0
	if len(pool) > 0 && pct > 0 {
		take = (len(pool)*pct + 99) / 100
		take = max(1, min(take, len(pool)))
		if limit > 0 {
			take = min(take, limit)
		}
	}
	gone := map[int]bool{}
	for ; take > 0 && len(pool) > 0; take-- {
		j := 0
		if roll != nil && len(pool) > 1 {
			j = max(0, min(roll(len(pool)), len(pool)-1))
		}
		gone[pool[j]] = true
		pool = append(pool[:j], pool[j+1:]...)
	}
	for i, itm := range pack {
		if gone[i] {
			taken = append(taken, itm)
		} else {
			kept = append(kept, itm)
		}
	}
	return kept, taken
}

// CaptorGroup names the spawn group of a capture room's guards, so the
// guards of one company's capture are told apart from any other.
func CaptorGroup(roomID, userID int) string {
	return fmt.Sprintf("captors:%d:%d", roomID, userID)
}

func miscInt(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// SeizedGold is the gold a capture holds for a character (YAML may bring
// the number back as another integer type).
func SeizedGold(c interface{ GetMiscData(string) any }) int {
	return max(0, miscInt(c.GetMiscData(SeizedGoldKey)))
}

// SeizedGuards is how many of a capture's guards still stand over the
// chest, saved with the character.
func SeizedGuards(c interface{ GetMiscData(string) any }) int {
	return max(0, miscInt(c.GetMiscData(SeizedGuardsKey)))
}

// SeizedRoom is the capture room holding a character's pack; 0 when none.
func SeizedRoom(c interface{ GetMiscData(string) any }) int {
	return max(0, miscInt(c.GetMiscData(SeizedRoomKey)))
}
