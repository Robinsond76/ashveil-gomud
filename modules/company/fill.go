package company

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 40a: `company fill` tops up every refillable water container the
// company carries (your pack, each companion's pack, and the cargo) at a
// water source. It takes no game time and spends nothing.

func (m *CompanyModule) fillView(user *users.UserRecord, room *rooms.Room) string {
	if usercommands.InBattle(user) {
		return usercommands.BattleUnderWay
	}
	if !usercommands.HasWater(room) {
		return "There is no fresh water here to fill from."
	}
	filled := 0
	// Your own pack.
	for i := range user.Character.Items {
		if full, ok := usercommands.RefillableUses(user.Character.Items[i].GetSpec()); ok && user.Character.Items[i].Uses < full {
			user.Character.Items[i] = user.Character.Items[i].Refilled(full)
			filled++
		}
	}
	// Each companion's own pack, walking with the leader.
	for _, member := range m.presentNeeds(user.UserId, survival.CompanyNeeds(user.UserId)) {
		id, isCompanion := companionIDOf(member.Key)
		if !isCompanion {
			continue
		}
		carried, ok := m.carriedBy(user.UserId, id)
		if !ok {
			continue
		}
		for _, itm := range carried {
			if full, ok := usercommands.RefillableUses(itm.GetSpec()); ok && itm.Uses < full {
				if m.setCompanionItemUses(user.UserId, id, itm, full) {
					filled++
				}
			}
		}
	}
	// The cargo: partly used containers come out and go back full.
	if !user.Character.CompanyCargo {
		for _, stack := range encumbrance.CargoContents(user.UserId) {
			spec := items.GetItemSpec(stack.ItemId)
			if spec == nil {
				continue
			}
			// Phase 43a: an empty container in the cargo comes out and goes
			// back as its full item.
			if spec.FilledItemId > 0 {
				if err := encumbrance.WithdrawCargo(user.UserId, stack.ItemId, stack.Count); err != nil {
					mudlog.Warn("company: fill cargo", "leader", user.UserId, "error", err)
					continue
				}
				deposit := []encumbrance.CargoStack{{ItemId: spec.FilledItemId, Count: stack.Count}}
				if err := encumbrance.DepositCargo(user.UserId, "", deposit); err != nil {
					mudlog.Error("company: fill cargo deposit", "leader", user.UserId, "error", err)
					continue
				}
				filled += stack.Count
				continue
			}
			if stack.Uses <= 0 {
				continue
			}
			if full, ok := usercommands.RefillableUses(*spec); !ok || stack.Uses >= full {
				continue
			}
			if err := encumbrance.WithdrawCargo(user.UserId, stack.ItemId, stack.Count); err != nil {
				mudlog.Warn("company: fill cargo", "leader", user.UserId, "error", err)
				continue
			}
			deposit := []encumbrance.CargoStack{{ItemId: stack.ItemId, Count: stack.Count}}
			if err := encumbrance.DepositCargo(user.UserId, "", deposit); err != nil {
				mudlog.Error("company: fill cargo deposit", "leader", user.UserId, "error", err)
				continue
			}
			filled += stack.Count
		}
	}
	if filled == 0 {
		return "Every water container your company carries is already full, or there is none to fill."
	}
	if room != nil {
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s company fills its water containers.`, user.Character.Name), user.UserId)
	}
	noun := "containers"
	if filled == 1 {
		noun = "container"
	}
	return strings.TrimSpace(fmt.Sprintf("Your company fills %d water %s from the water here.", filled, noun))
}

// setCompanionItemUses sets the uses of a companion's own item: on the live
// mob when it is out, else on its record, saved.
func (m *CompanyModule) setCompanionItemUses(leaderUserID, companionID int, itm items.Item, uses int) bool {
	if instanceID, tracked := m.instance(leaderUserID, companionID); tracked && m.runtime.IsLive(instanceID) {
		if !m.runtime.SetItemUses(instanceID, itm, uses) {
			return false
		}
		m.refreshSnapshot(leaderUserID, companionID)
		return true
	}
	record, ok := m.registry.Get(leaderUserID)
	if !ok {
		return false
	}
	for _, c := range record.Companions {
		if c.ID != companionID || c.State == nil {
			continue
		}
		before := c.State.Clone()
		state := c.State.Clone()
		for i := range state.Items {
			if !state.Items[i].Equals(itm) {
				continue
			}
			state.Items[i] = state.Items[i].Refilled(uses)
			if m.registry.SetState(leaderUserID, companionID, state) != nil {
				return false
			}
			if err := m.save(); err != nil {
				_ = m.registry.SetState(leaderUserID, companionID, before)
				mudlog.Error("company: fill", "error", err)
				return false
			}
			return true
		}
	}
	return false
}
