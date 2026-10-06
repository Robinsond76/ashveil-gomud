package camping

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 52 tents. Which tent a camp pitches is the leader's choice among
// the tents the company carries (`camp tent`); the pitched tent is locked on
// each rest when it starts, with its effects:
//   - raiders and thieves: the chance of each is scaled when the rest begins
//     (planRaidLocked, planTheftLocked);
//   - the weather's penalty to the rest's fatigue recovery (campRecovery);
//   - the Rested buff a finished rest grants: a large tent's is Well Rested,
//     a camouflaged tent's lasts half as long (grantPendingTiers). The tent
//     is saved with the pending grant (restedTents), so breaking camp or
//     resting again before the grant neither drops nor swaps it.
// Tents are bought, never sold back (market SupplyOnly), weigh on the
// company's load like any item, and thieves leave them alone (campGearItem).

// pendingTent is the tent a pending Rested grant was slept in.
func (m *CampingModule) pendingTent(leaderUserID int) (camping.Tent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	kind, ok := m.restedTents[leaderUserID]
	if !ok || kind == "" {
		return camping.Tent{}, false
	}
	return camping.TentOf(kind), true
}

// tentRows is the tents carried, for the Camp tab's picker.
func (m *CampingModule) tentRows(leaderUserID int, pitched camping.TentKind, tents []camping.TentKind) []camping.TentChoice {
	out := make([]camping.TentChoice, 0, len(tents))
	for _, k := range tents {
		t := camping.TentOf(k)
		out = append(out, camping.TentChoice{Kind: k, Name: t.Name, Effect: t.Effect, Pitched: k == pitched})
	}
	return out
}

func tentUsage() string {
	return "Usage: camp tent | camp tent [" + strings.Join(tentWords(), "|") + "] | camp tent clear"
}

func tentWords() []string {
	out := make([]string, 0, len(camping.Tents))
	for _, t := range camping.Tents {
		out = append(out, t.Short)
	}
	return out
}

// tentCommand shows the tents carried, or chooses one. The choice outlives
// the camp (52 review), so it can be made before a camp is pitched; at a
// camp it re-pitches the tent there, between rests.
func (m *CampingModule) tentCommand(user *users.UserRecord, room *rooms.Room, args []string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	carried := m.tentsCarried(user.UserId) // before m.mu: it calls the company module
	m.mu.Lock()
	camp, hasCamp := m.camps[user.UserId]
	chosen := m.tentChoices[user.UserId]
	m.mu.Unlock()
	pitched, _ := camping.PickTent(carried, chosen)
	if len(args) == 0 {
		return tentView(carried, pitched, hasCamp && camp.Tent)
	}
	if hasCamp && camp.Rest != nil && camp.Rest.State == camping.Resting {
		return "The company is resting: the tent was fixed when the rest began. Choose another for the next rest once it is over."
	}
	if hasCamp && (room == nil || camp.RoomID != room.RoomId) {
		return "Your camp is not here: pitch its tent at the camp."
	}
	word := strings.Join(args, " ")
	var choice camping.TentKind
	if strings.EqualFold(word, "clear") || strings.EqualFold(word, "reset") {
		choice = ""
	} else {
		kind, valid := camping.ParseTent(word)
		if !valid {
			return fmt.Sprintf("There is no tent called %q. %s", word, tentUsage())
		}
		if !hasTent(carried, kind) {
			return fmt.Sprintf("You carry no %s. Tents are sold at markets (help camp gear).", camping.TentOf(kind).Name)
		}
		choice = kind
	}
	next, has := camping.PickTent(carried, choice)
	m.mu.Lock()
	current, stillCamp := m.camps[user.UserId]
	if stillCamp != hasCamp || (stillCamp && current.Rest != nil && current.Rest.State == camping.Resting) {
		m.mu.Unlock()
		return "You can't change the tent now."
	}
	prevChoice, hadChoice := m.tentChoices[user.UserId]
	if choice == "" {
		delete(m.tentChoices, user.UserId)
	} else {
		m.tentChoices[user.UserId] = choice
	}
	if stillCamp {
		updated := current
		updated.Tent, updated.TentKind = has, next
		m.camps[user.UserId] = updated
	}
	defer m.refreshLitRoomsLocked()
	if err := m.saveLocked(); err != nil {
		if stillCamp {
			m.camps[user.UserId] = current
		}
		if hadChoice {
			m.tentChoices[user.UserId] = prevChoice
		} else {
			delete(m.tentChoices, user.UserId)
		}
		m.mu.Unlock()
		return err.Error()
	}
	m.mu.Unlock()
	if !has {
		return "You carry no tent to pitch."
	}
	t := camping.TentOf(next)
	if !stillCamp {
		return fmt.Sprintf("Your next camp will pitch the %s: %s.", t.Name, t.Effect)
	}
	return fmt.Sprintf("You pitch the %s: %s.", t.Name, t.Effect)
}

func hasTent(kinds []camping.TentKind, kind camping.TentKind) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// tentView is `camp tent` with no argument: every tent carried with its
// effect, and the one pitched (or, with no camp up, the one a camp would
// pitch).
func tentView(carried []camping.TentKind, pitched camping.TentKind, up bool) string {
	if len(carried) == 0 {
		return "You carry no tent. A market sells them (help camp gear)."
	}
	lines := []string{"Tents you carry:"}
	for _, k := range carried {
		t := camping.TentOf(k)
		mark := "  "
		if k == pitched {
			mark = "* "
		}
		lines = append(lines, fmt.Sprintf("%s%s (%s): %s.", mark, util.CapitalizeFirst(t.Name), t.Short, t.Effect))
	}
	legend := "* is pitched."
	if !up {
		legend = "* is the one your next camp pitches."
	}
	lines = append(lines, legend+" Choose with: camp tent ["+strings.Join(tentWords(), "|")+"]. The tent is fixed when a rest begins.")
	return strings.Join(lines, "\n")
}
