package company

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"

	domain "github.com/GoMudEngine/GoMud/internal/company"
)

// theftUnit is one stealable unit: a cargo item or one pack item.
type theftUnit struct {
	itemID int
	owner  int // companion ID, or 0 for the cargo
	item   items.Item
}

// stealable reports whether thieves may take an item: never a quest token,
// a key, or something the caller protects.
func stealable(itemID int, protect func(int) bool) bool {
	if protect != nil && protect(itemID) {
		return false
	}
	spec := items.GetItemSpec(itemID)
	if spec == nil {
		return false
	}
	return spec.QuestToken == "" && spec.Type != items.Key
}

// CampTheft implements company.TheftProvider (Phase 40a4).
func (m *CompanyModule) CampTheft(leaderUserID, sharePct, maxUnits int, pick func(n int) int, protect func(itemID int) bool) []domain.TheftLoss {
	user := users.GetByUserId(leaderUserID)
	if user == nil || user.Character == nil || m.persistenceAvailable() != nil || pick == nil {
		return nil
	}
	var pool []theftUnit
	for _, stack := range encumbrance.CargoContents(leaderUserID) {
		if !stealable(stack.ItemId, protect) {
			continue
		}
		for i := 0; i < stack.Count; i++ {
			pool = append(pool, theftUnit{itemID: stack.ItemId})
		}
	}
	for _, w := range m.woundMembers(user) {
		if w.leader() {
			continue
		}
		carried, ok := m.carriedBy(leaderUserID, w.companionID)
		if !ok {
			continue
		}
		for _, itm := range carried {
			if stealable(itm.ItemId, protect) {
				pool = append(pool, theftUnit{itemID: itm.ItemId, owner: w.companionID, item: itm})
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	take := (len(pool)*sharePct + 99) / 100
	take = max(1, min(take, maxUnits, len(pool)))
	counts := map[int]int{}
	for ; take > 0 && len(pool) > 0; take-- {
		i := pick(len(pool))
		if i < 0 || i >= len(pool) {
			i = 0
		}
		unit := pool[i]
		pool = append(pool[:i], pool[i+1:]...)
		if m.stealUnit(leaderUserID, unit) {
			counts[unit.itemID]++
		}
	}
	ids := make([]int, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]domain.TheftLoss, 0, len(ids))
	for _, id := range ids {
		name := ""
		if spec := items.GetItemSpec(id); spec != nil {
			name = spec.Name
		}
		out = append(out, domain.TheftLoss{ItemID: id, Name: name, Count: counts[id]})
	}
	return out
}

// stealUnit removes one unit, a whole item from a pack however many uses
// it had left.
func (m *CompanyModule) stealUnit(leaderUserID int, u theftUnit) bool {
	if u.owner == 0 {
		if err := encumbrance.WithdrawCargo(leaderUserID, u.itemID, 1); err != nil {
			mudlog.Warn("company: camp theft cargo", "error", err)
			return false
		}
		return true
	}
	took := false
	for n := max(1, u.item.Uses); n > 0; n-- {
		if !m.useCompanionItem(leaderUserID, u.owner, u.item) {
			break
		}
		took = true
	}
	return took
}

var _ domain.TheftProvider = (*CompanyModule)(nil)
