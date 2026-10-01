package encumbrance

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// UnifyCargo migrates old stacks into the instance-preserving company cargo.
// The marker is written with the instances first. Old stacks can then be removed
// or retried after a crash without importing them twice.
func (m *EncumbranceModule) UnifyCargo(id int) error {
	u := m.sharedUser(id)
	if u == nil {
		return nil
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if u.Character.CargoMigrated {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	beforeItems := append([]items.Item(nil), u.Character.Items...)
	beforeApplied := u.Character.CargoApplied
	oldMode := u.Character.CompanyCargo
	legacy := m.cargo[id]
	if len(legacy.Stacks) > 0 {
		if err := legacy.Validate(); err != nil {
			return err
		}
		for _, s := range legacy.Stacks {
			if _, ok := m.itemSpec(s.ItemId); !ok {
				return fmt.Errorf("cargo: unknown legacy item %d; migration stopped", s.ItemId)
			}
		}
		for _, s := range legacy.Stacks {
			for range s.Count {
				itm := m.newCargoItem(s.ItemId)
				if s.Uses > 0 {
					itm.Uses = s.Uses
				}
				u.Character.Items = append(u.Character.Items, itm)
			}
		}
	}
	u.Character.CargoMigrated, u.Character.CompanyCargo = true, true
	u.Character.CargoApplied = append([]string(nil), legacy.Applied...)
	if err := m.writeShared(u); err != nil {
		u.Character.Items = beforeItems
		u.Character.CargoMigrated, u.Character.CompanyCargo = false, oldMode
		u.Character.CargoApplied = beforeApplied
		return err
	}
	// Keeping an old file after a failed cleanup is safe: the user's durable
	// marker prevents any subsequent migration and readers use shared instances.
	delete(m.cargo, id)
	if err := m.saveLocked(); err != nil {
		m.cargo[id] = legacy
	}
	return nil
}

func (m *EncumbranceModule) sharedUser(id int) *users.UserRecord {
	if m.userLookup == nil {
		return nil
	}
	u := m.userLookup(id)
	if u == nil || u.Character == nil || u.UserId != id {
		return nil
	}
	return u
}

func (m *EncumbranceModule) writeShared(u *users.UserRecord) error {
	var err error
	if m.saveUser != nil {
		err = m.saveUser(u)
	} else {
		err = users.SaveUserAtomic(*u)
	}
	if err == nil {
		events.AddToQueue(events.CompanyAssetsChanged{UserId: u.UserId})
	}
	return err
}

func (m *EncumbranceModule) sharedStacks(u *users.UserRecord) []domain.CargoStack {
	c := domain.Cargo{LeaderUserID: u.UserId}
	for _, itm := range u.Character.Items {
		if itm.ItemId > 0 {
			c, _ = c.DepositUses(itm.ItemId, m.partialUses(itm), 1)
		}
	}
	return c.Stacks
}

func (m *EncumbranceModule) sharedConsume(u *users.UserRecord, id, count int, oneUse bool) error {
	if err := company.PrepareAssets(u.UserId); err != nil {
		return err
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	cargo := append([]items.Item(nil), u.Character.Items...)
	for range count {
		idx := -1
		for i, itm := range cargo {
			if itm.ItemId == id && (idx < 0 || (itm.Uses > 0 && (cargo[idx].Uses == 0 || itm.Uses < cargo[idx].Uses))) {
				idx = i
			}
		}
		if idx < 0 {
			return domain.ErrInsufficientCargo
		}
		itm := cargo[idx]
		if oneUse && itm.Uses > 1 {
			cargo[idx].Uses--
		} else {
			cargo = append(cargo[:idx], cargo[idx+1:]...)
		}
	}
	before := u.Character.Items
	u.Character.Items = cargo
	if err := m.writeShared(u); err != nil {
		u.Character.Items = before
		return err
	}
	return nil
}

func (m *EncumbranceModule) sharedDeposit(u *users.UserRecord, op string, stacks []domain.CargoStack) error {
	if err := company.PrepareAssets(u.UserId); err != nil {
		return err
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if op != "" && slices.Contains(u.Character.CargoApplied, op) {
		return nil
	}
	cargo := append([]items.Item(nil), u.Character.Items...)
	for _, s := range stacks {
		if s.Count < 1 || s.ItemId < 1 || s.Uses < 0 {
			return domain.ErrInvalidAmount
		}
		if _, ok := m.itemSpec(s.ItemId); !ok {
			return domain.ErrInvalidCargo
		}
		for range s.Count {
			itm := m.newCargoItem(s.ItemId)
			if s.Uses > 0 {
				itm.Uses = s.Uses
			}
			cargo = append(cargo, itm)
		}
	}
	oldItems, oldOps := u.Character.Items, u.Character.CargoApplied
	u.Character.Items = cargo
	if op != "" {
		u.Character.CargoApplied = append(append([]string(nil), oldOps...), op)
		if len(u.Character.CargoApplied) > domain.MaxAppliedOps {
			u.Character.CargoApplied = u.Character.CargoApplied[len(u.Character.CargoApplied)-domain.MaxAppliedOps:]
		}
	}
	if err := m.writeShared(u); err != nil {
		u.Character.Items, u.Character.CargoApplied = oldItems, oldOps
		return err
	}
	return nil
}

// sharedPackBonus assigns the largest available packs to living members;
// a physical pack contributes only once, regardless of the item owner field.
func sharedPackBonus(carried []items.Item, members int) int {
	bonuses := []int{}
	for _, itm := range carried {
		if b := itm.CarryBonusGrams(); b > 0 {
			bonuses = append(bonuses, b)
		}
	}
	slices.Sort(bonuses)
	total := 0
	for i := len(bonuses) - 1; i >= 0 && members > 0; i-- {
		total += bonuses[i]
		members--
	}
	return total
}

func (m *EncumbranceModule) StrengthCapacityGrams() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.strengthGrams
}

func (m *EncumbranceModule) ConsumeCargoItemUse(id int, itm items.Item) error {
	if err := company.PrepareAssets(id); err != nil {
		return err
	}
	u := m.sharedUser(id)
	if u == nil || !u.Character.CompanyCargo {
		return domain.ErrNoCargo
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	before := append([]items.Item(nil), u.Character.Items...)
	for _, current := range u.Character.Items {
		if current.Equals(itm) {
			u.Character.UseItem(current)
			if err := m.writeShared(u); err != nil {
				u.Character.Items = before
				return err
			}
			return nil
		}
	}
	return domain.ErrInsufficientCargo
}

func (m *EncumbranceModule) CargoAddedGrams(id int, itm items.Item) (int, bool) {
	return m.CargoExchangeGrams(id, nil, []items.Item{itm})
}

func (m *EncumbranceModule) CargoExchangeGrams(id int, removed, added []items.Item) (int, bool) {
	u := m.sharedUser(id)
	if u == nil || !u.Character.CompanyCargo {
		return 0, false
	}
	members := 1
	if m.companionCarry != nil {
		members += len(m.companionCarry(id))
	}
	before := sharedPackBonus(u.Character.Items, members)
	next := append([]items.Item(nil), u.Character.Items...)
	grams := 0
	for _, itm := range removed {
		for i, old := range next {
			if old.Equals(itm) {
				next = append(next[:i], next[i+1:]...)
				grams -= old.Weight()
				break
			}
		}
	}
	for _, itm := range added {
		next = append(next, itm)
		grams += itm.Weight()
	}
	after := sharedPackBonus(next, members)
	return grams - (after - before), true
}

func (m *EncumbranceModule) TransformCargo(id int, inputs, outputs []items.Item) error {
	if err := company.PrepareAssets(id); err != nil {
		return err
	}
	u := m.sharedUser(id)
	if u == nil || !u.Character.CompanyCargo {
		return domain.ErrNoCargo
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	next := append([]items.Item(nil), u.Character.Items...)
	for _, input := range inputs {
		found := false
		for i, itm := range next {
			if itm.Equals(input) {
				next = append(next[:i], next[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return domain.ErrInsufficientCargo
		}
	}
	next = append(next, outputs...)
	before := u.Character.Items
	u.Character.Items = next
	if err := m.writeShared(u); err != nil {
		u.Character.Items = before
		return err
	}
	return nil
}
