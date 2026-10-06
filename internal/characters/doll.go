package characters

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Phase 39d: a Doll Master's dolls. A doll is a durable record kept on its
// Master (a player's saved character, a companion's company state). It
// becomes a live mob only for a battle (internal/dolls), and its health and
// gear are written back here as the battle goes, so a restart or copyover
// keeps them.

// DollStarterWeapon is the item a new doll wields: a plain cudgel.
const DollStarterWeapon = 10010

// dollNames are the names new dolls take, in order, so two dolls of one
// Master read apart in the battle lines.
var dollNames = []string{"Pip", "Pim", "Pod", "Pax"}

// DollState is one doll's durable record.
type DollState struct {
	Name string `yaml:"name,omitempty"`
	// Damage is the health the doll has lost, which mending takes back. A
	// doll is whole at 0.
	Damage int `yaml:"damage,omitempty"`
	// Broken is a doll that fell in battle. It stays out until it is mended.
	Broken    bool `yaml:"broken,omitempty"`
	Equipment Worn `yaml:"equipment,omitempty"`
}

// NewDollState is a fresh doll: named, whole, and holding a cudgel.
func NewDollState(index int) DollState {
	d := DollState{Name: dollNames[index%len(dollNames)]}
	d.Equipment.Weapon = items.New(DollStarterWeapon)
	return d
}

// DollInfo is what a live doll is: its Master, its place in the Master's
// list, and the gifts the Master's route gave it when it was made. It lives
// for one battle.
type DollInfo struct {
	OwnerUser, OwnerMob int    // the Master's user id, or its mob instance
	OwnerKey            string // the Master's company member key
	Index               int    // its place among the Master's dolls
	HPPct               int    // percent of a warrior's health at its level
	Damage              int    // damage added to each of its blows
}

// EnsureDolls makes the character's doll records at least n long, adding
// fresh dolls, and returns them. It never removes one.
func (c *Character) EnsureDolls(n int) []DollState {
	for len(c.Dolls) < n {
		c.Dolls = append(c.Dolls, NewDollState(len(c.Dolls)))
	}
	return c.Dolls
}

// HasDoll reports whether the character has any doll records.
func (c *Character) HasDoll() bool { return len(c.Dolls) > 0 }

// CloneDolls copies a list of doll records, items and all.
func CloneDolls(in []DollState) []DollState {
	if in == nil {
		return nil
	}
	out := make([]DollState, len(in))
	for i, d := range in {
		out[i] = d
		out[i].Equipment = Worn{}
		for _, slot := range AllSlots() {
			if itm := d.Equipment.Get(slot); itm != nil && itm.ItemId != 0 {
				out[i].Equipment.Set(slot, cloneWornItem(*itm))
			}
		}
	}
	return out
}

func cloneWornItem(i items.Item) items.Item {
	if i.Adjectives != nil {
		i.Adjectives = append([]string(nil), i.Adjectives...)
	}
	if i.Spec != nil {
		spec := *i.Spec
		i.Spec = &spec
	}
	i.Loot.Affixes = append([]items.RolledAffix(nil), i.Loot.Affixes...)
	return i
}
