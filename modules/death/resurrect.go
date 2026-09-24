package death

// Phase 25b: the resurrect command. This module owns the rite: where it can
// be performed (a registered city's church or village's shaman, with its
// keeper present) and what the player is told. modules/company owns what
// it does to the companion, through company.ResurrectCompanion.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// keeperInRoom finds a living mob of the keeper's template in the room.
func keeperInRoom(room *rooms.Room, mobID int) (string, bool) {
	if room == nil || mobID <= 0 {
		return "", false
	}
	for _, instanceID := range room.GetMobs() {
		mob := mobs.GetInstance(instanceID)
		if mob != nil && int(mob.MobId) == mobID && mob.Character.Health >= 1 {
			return mob.Character.Name, true
		}
	}
	return "", false
}

func allowanceText(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", max(seconds, 0))
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, seconds%3600/60)
}

// listing shows the leader's dead companions and the time left to raise
// each.
func (m *DeathModule) listing(userID int) string {
	dead := m.deadCompanions(userID)
	if len(dead) == 0 {
		return "None of your company lies dead."
	}
	lines := []string{"Your fallen, and the time you have left to raise them:"}
	for _, d := range dead {
		lines = append(lines, fmt.Sprintf("  #%d %s, level %d: %s", d.ID, d.Name, d.Level, allowanceText(d.Remaining)))
	}
	lines = append(lines, `Bring your company to a city's church or a village's shaman and <ansi fg="command">resurrect <member></ansi>. It costs them a level.`)
	return strings.Join(lines, "\n")
}

// resurrectCommand is "resurrect" (list the dead) and "resurrect <member>"
// (raise one, at a service room with its keeper present).
func (m *DeathModule) resurrectCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	selector := strings.TrimSpace(rest)
	if selector == "" {
		user.SendText(m.listing(user.UserId))
		return true, nil
	}
	if room == nil {
		room = m.loadRoom(user.Character.RoomId)
	}
	var settlement domain.Settlement
	ok := false
	if room != nil {
		settlement, ok = m.config().registry.ServiceAt(room.RoomId)
		ok = ok && room.Zone == settlement.Zone && room.HasTag(settlement.ServiceTag())
	}
	if !ok {
		user.SendText("There is no one here who can call back the dead. Seek a city's church or a village's shaman.")
		return true, nil
	}
	keeper, present := m.keeper(room, settlement.ServiceMobID)
	if !present {
		user.SendText("There is no one to perform the rite just now.")
		return true, nil
	}
	if user.Character.Aggro != nil {
		user.SendText("Not while you are fighting.")
		return true, nil
	}
	result, err := m.raise(user.UserId, selector, room.RoomId)
	switch {
	case errors.Is(err, company.ErrUnknownMember):
		user.SendText(fmt.Sprintf(`None of your company answers to "%s".`, selector))
		return true, nil
	case errors.Is(err, company.ErrAmbiguousMember):
		user.SendText(fmt.Sprintf(`More than one of your company answers to "%s". Use their number: <ansi fg="command">resurrect</ansi> lists them.`, selector))
		return true, nil
	case errors.Is(err, company.ErrNotDead):
		name := result.Name
		if name == "" {
			name = selector
		}
		user.SendText(fmt.Sprintf("%s is not dead.", name))
		return true, nil
	case errors.Is(err, company.ErrCompanionLost):
		user.SendText("It is too late: they are lost to you, and no rite can call them back.")
		return true, nil
	case err != nil:
		mudlog.Error("death: resurrection failed", "user", user.UserId, "selector", selector, "error", err)
		user.SendText("The rite falters and fails. Try again.")
		return true, nil
	}
	text := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kneels and calls %s back from death. %s draws breath again (now level <ansi fg="yellow">%d</ansi>).`,
		keeper, result.Name, result.Name, result.Level)
	if !result.Spawned {
		text += " They will rejoin you when you next return to the world."
	}
	user.SendText(text)
	room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kneels and calls %s back from death.`, keeper, result.Name), user.UserId)
	return true, nil
}
