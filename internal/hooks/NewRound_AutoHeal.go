package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Watches the rounds go by
// Applies autohealing where appropriate
//

func AutoHeal(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	// Every 3 rounds. Else, pass it along.
	if evt.RoundNumber%3 != 0 {
		return events.Continue
	}

	deathRecoveryRoomId := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)

	// Ashveil Phase 32d: companions regain mana as players do.
	regenCompanionMana(company.LeaderAndKeyForInstance)

	onlineIds := users.GetOnlineUserIds()
	for _, userId := range onlineIds {
		user := users.GetByUserId(userId)

		// Only heal if not in combat
		if user.Character.Aggro != nil {
			continue
		}

		if user.Character.RoomId == deathRecoveryRoomId {
			continue
		}

		healthStart := user.Character.Health
		manaStart := user.Character.Mana

		if user.Character.Health < 1 {

			if user.Character.Health <= -10 {

				user.Command(`suicide`) // suicide drops all money/items and transports to land of the dead.

			} else {
				user.Character.Health--
				user.SendText(`<ansi fg="red">you are bleeding out!</ansi>`)
				if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is <ansi fg="red">bleeding out</ansi>! Somebody needs to provide aid!`, user.Character.Name), user.UserId)
				}
			}

		} else {

			if user.Character.Health > 0 {
				user.Character.Heal(
					user.Character.HealthPerRound(),
					user.Character.ManaPerRound(),
				)
			}
		}

		// If it has changed, send an update
		if user.Character.Health != healthStart || user.Character.Mana != manaStart {

			// Trigger a redraw, but only if the users prompt has changed.
			events.AddToQueue(events.RedrawPrompt{UserId: user.UserId, OnlyIfChanged: true}, 100)

			events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

		}

	}

	return events.Continue
}

// regenCompanionMana gives each living companion out of combat its mana
// per round (Phase 32d), on the same every-third-round beat as players,
// while its leader is online. Mobs otherwise never regain mana: a
// companion that cast in a battle would stay empty. Health is left as it
// is (companions heal by their own rules).
func regenCompanionMana(leaderOf func(instanceId int) (int, company.MemberKey, bool)) {
	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Aggro != nil || mob.Character.Health < 1 {
			continue
		}
		leaderId, _, ok := leaderOf(instanceId)
		if !ok || users.GetByUserId(leaderId) == nil {
			continue
		}
		if mob.Character.Mana < mob.Character.ManaMax.Value {
			mob.Character.Heal(0, mob.Character.ManaPerRound())
		}
	}
}
