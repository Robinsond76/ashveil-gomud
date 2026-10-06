package company

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// MemberState is a companion's durable progression and gear (Phase 22b).
// The live mob is rebuilt from it on every restore, so gear given to a
// companion survives logout, restart, and copyover, and template gear is
// minted only once.
type MemberState struct {
	Level      int             `yaml:"level"`
	Experience int             `yaml:"experience,omitempty"`
	Equipment  characters.Worn `yaml:"equipment,omitempty"`
	Items      []items.Item    `yaml:"items,omitempty"`
	Gold       int             `yaml:"gold,omitempty"`
	// Wounds are the companion's wounds (Phase 30b), snapshotted from the
	// live mob with its gear and put back on every spawn.
	Wounds []wounds.Wound `yaml:"wounds,omitempty"`
	// Vitals are its health and mana (Phase 33h2), snapshotted with its
	// gear so a respawn doesn't refill it. Nil (a record from before 33h2)
	// spawns full once.
	Vitals *Vitals `yaml:"vitals,omitempty"`
	// Dolls are a Doll Master companion's dolls (Phase 39d): name, wear,
	// broken flag and gear, snapshotted with the companion.
	Dolls []characters.DollState `yaml:"dolls,omitempty"`
	// Beast is a Beast Tamer companion's bonded beast (Phase 39e).
	Beast *characters.BeastState `yaml:"beast,omitempty"`
	// FlasksSpent is the flasks an Alchemist companion has thrown since it
	// last brewed (Phase 39g); 0 is a full satchel.
	FlasksSpent int `yaml:"flasksspent,omitempty"`
}

// Vitals are a companion's saved health and mana (Phase 33h2). Percent,
// when set, stands for that share of the limits instead (a resurrection),
// resolved at the next spawn so a crash before it can't refill anyone.
type Vitals struct {
	Health  int `yaml:"health"`
	Mana    int `yaml:"mana"`
	Percent int `yaml:"percent,omitempty"`
}

// Resolve is the health and mana a companion spawns with, against its
// current wound limit and mana maximum. Saved points are kept as points,
// never rescaled, so a raised maximum grants nothing; a lowered one clamps.
// A living companion never spawns below 1 health.
func (v *Vitals) Resolve(limit, manaMax int) (health, mana int) {
	limit, manaMax = max(limit, 1), max(manaMax, 0)
	switch {
	case v == nil:
		return limit, manaMax
	case v.Percent > 0:
		pct := min(v.Percent, 100)
		return max(1, limit*pct/100), manaMax * pct / 100
	}
	return min(max(v.Health, 1), limit), min(max(v.Mana, 0), manaMax)
}

func cloneItem(i items.Item) items.Item {
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

// Clone returns a deep copy.
func (s MemberState) Clone() MemberState {
	out := s
	out.Wounds = append([]wounds.Wound(nil), s.Wounds...)
	out.Dolls = characters.CloneDolls(s.Dolls)
	out.Beast = characters.CloneBeast(s.Beast)
	if s.Vitals != nil {
		v := *s.Vitals
		out.Vitals = &v
	}
	for _, slot := range characters.AllSlots() {
		if itm := s.Equipment.Get(slot); itm != nil {
			out.Equipment.Set(slot, cloneItem(*itm))
		}
	}
	if s.Items != nil {
		out.Items = make([]items.Item, len(s.Items))
		for i, itm := range s.Items {
			out.Items[i] = cloneItem(itm)
		}
	}
	return out
}

// ClearGear drops every worn and carried item and the gold, keeping
// progression.
func (s *MemberState) ClearGear() {
	s.Equipment = characters.Worn{}
	s.Items = nil
	s.Gold = 0
}

// SetState records a companion's state. It returns ErrUnknownMember for a
// leader or companion not in the registry.
func (r *Registry) SetState(leaderUserID, companionID int, state MemberState) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID == companionID {
			s := state.Clone()
			record.Companions[i].State = &s
			r.Put(record)
			return nil
		}
	}
	return ErrUnknownMember
}

// BestPackGrams is the largest pack bonus among carried items (Phase 32f):
// a member counts one pack; a second is only weight.
func BestPackGrams(carried []items.Item) int {
	_, grams := BestPack(carried)
	return grams
}

// BestPack is the carried pack that counts, and its bonus; a zero item
// and 0 when there is none.
func BestPack(carried []items.Item) (items.Item, int) {
	best, bestGrams := items.Item{}, 0
	for i := range carried {
		if carried[i].ItemId <= 0 {
			continue
		}
		if bonus := carried[i].CarryBonusGrams(); bonus > bestGrams {
			best, bestGrams = carried[i], bonus
		}
	}
	return best, bestGrams
}
