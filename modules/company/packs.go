package company

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/creatures"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/characters"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Data selects the starter. Test modules can opt into the same migration.
func (m *CompanyModule) starterPackID() int {
	if m.starterPackForTest > 0 {
		return m.starterPackForTest
	}
	if m.plug == nil {
		return 0
	}
	id, _ := strconv.Atoi(fmt.Sprint(m.plug.Config.Get("StarterPackItemId")))
	return id
}

func validStarterPack(id int) error {
	spec := items.GetItemSpec(id)
	if spec == nil || spec.Type != items.Pack || spec.CarryBonus <= 0 {
		return fmt.Errorf("company: starter pack %d unavailable; pack migration awaits recovery", id)
	}
	return nil
}

func (m *CompanyModule) giveRecruitPack(st *domain.MemberState, id int) error {
	if err := validStarterPack(id); err != nil {
		return err
	}
	if st.Equipment.Pack.ItemId == 0 {
		st.Equipment.Pack = items.New(id)
	}
	return nil
}

// migratePacks runs inside the existing asset journal. Grant markers and exact
// instances are saved with the resulting cargo and leader/companion equipment.
// An acknowledged grant remains spent across removal, death and resurrection.
func (m *CompanyModule) migratePacks(rec *domain.Record, cargo *[]items.Item, worn *characters.Worn, starter int) (bool, error) {
	if err := validStarterPack(starter); err != nil {
		return false, err
	}
	assign := func(w *characters.Worn) {
		if w.Pack.ItemId != 0 {
			return
		}
		best := -1
		for i, itm := range *cargo {
			if itm.GetSpec().Type == items.Pack && itm.CarryBonusGrams() > 0 && (best < 0 || itm.CarryBonusGrams() > (*cargo)[best].CarryBonusGrams()) {
				best = i
			}
		}
		if best >= 0 {
			w.Pack = (*cargo)[best]
			*cargo = append((*cargo)[:best], (*cargo)[best+1:]...)
		} else {
			w.Pack = items.New(starter)
		}
	}
	changed := false
	if !rec.LeaderPackGranted {
		assign(worn)
		rec.LeaderPackGranted, changed = true, true
	}
	for i, c := range rec.Companions {
		if c.PackGranted || c.Dead() || c.State == nil || creatures.Is(c.Archetype) { // Phase 38e: a creature carries no pack
			continue
		}
		st := c.State.Clone()
		assign(&st.Equipment)
		rec.Companions[i].State = &st
		rec.Companions[i].PackGranted, changed = true, true
	}
	return changed, nil
}

// equipmentLoad uses final item locations and assigned capacity. It also drives
// comparison so the read model and mutation guard cannot disagree.
func equipmentLoad(id int, actor, proposed *characters.Character, before, after []items.Item) (encumbrance.Load, encumbrance.Load, bool) {
	load, ok := encumbrance.CurrentLoad(id)
	return equipmentLoadFrom(load, ok, actor, proposed, before, after)
}

// equipmentLoadFrom is equipmentLoad from a load already read, so a view
// previewing many changes reads the company's load once.
func equipmentLoadFrom(load encumbrance.Load, ok bool, actor, proposed *characters.Character, before, after []items.Item) (encumbrance.Load, encumbrance.Load, bool) {
	next := load
	for _, itm := range before {
		next.CargoGrams -= itm.Weight()
	}
	for _, itm := range after {
		next.CargoGrams += itm.Weight()
	}
	delta := proposed.Equipment.Pack.CarryBonusGrams() - actor.Equipment.Pack.CarryBonusGrams()
	next.CapacityGrams += delta
	next.MemberCapacityGrams += delta
	return load, next, ok
}
