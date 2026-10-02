package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
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

	// Ashveil Phase 32d/33h2: companions regain mana and health as players do.
	regenCompanionVitals(company.LeaderAndKeyForInstance)
	// Ashveil Phase 33i2: an enemy that lived through a fight recovers.
	regenEnemyVitals(company.LeaderAndKeyForInstance, battle.Engaged)

	onlineIds := users.GetOnlineUserIds()
	for _, userId := range onlineIds {
		user := users.GetByUserId(userId)

		// Only heal if not in combat. Ashveil Phase 32d: nor in a battle,
		// between blows (a spell just ended, a target just fell).
		if user.Character.Aggro != nil {
			continue
		}
		if _, inBattle := battle.Current(userId); inBattle {
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

// regenCompanionVitals gives each living companion out of combat its mana
// (Phase 32d) and health (Phase 33h2) per round, on the same every-third-
// round beat as players, while its leader is online. Mobs otherwise never
// recover: a companion's vitals are saved (33h2), so without this one hurt
// in a battle would stay hurt. Health stops at the wound limit (Heal).
func regenCompanionVitals(leaderOf func(instanceId int) (int, company.MemberKey, bool)) {
	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Aggro != nil || mob.Character.Health < 1 {
			continue
		}
		leaderId, _, ok := leaderOf(instanceId)
		if !ok || users.GetByUserId(leaderId) == nil {
			continue
		}
		if _, inBattle := battle.Current(leaderId); inBattle {
			continue // between blows in a battle is still the battle
		}
		hp, mana := 0, 0
		if mob.Character.Health < mob.Character.HealthLimit() {
			hp = mob.Character.HealthPerRound()
		}
		if mob.Character.Mana < mob.Character.ManaMax.Value {
			mana = mob.Character.ManaPerRound()
		}
		if hp > 0 || mana > 0 {
			mob.Character.Heal(hp, mana)
		}
	}
}

// EnemyRecoveryRounds is the rounds an enemy takes to recover from nothing
// to full health and mana out of a fight (Phase 33i2): 5 game hours at
// the shipped 900-round day, about 12½ real minutes.
const EnemyRecoveryRounds = 188

// enemyRecovery is what an enemy of max health or mana recovers on one
// pass: a 1/EnemyRecoveryRounds share a round, rounded up, for the three
// rounds between passes.
func enemyRecovery(max int) int {
	if max < 1 {
		return 0
	}
	return 3 * ((max + EnemyRecoveryRounds - 1) / EnemyRecoveryRounds)
}

// regenEnemyVitals gives each living mob outside a company (an enemy, or
// any creature a fight left hurt) its share of health and mana while it is
// out of every battle and not fighting (Phase 33i2). Before, a mob that
// survived a fight stayed hurt until it died. It runs on rounds, so world
// time never advances for it; a charmed pet is left to its owner's rules.
// Health stops at the wound limit (Heal), though an enemy's light wounds
// have closed by the time its fight is over.
func regenEnemyVitals(leaderOf func(instanceId int) (int, company.MemberKey, bool), engaged func(instanceId int) bool) {
	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instanceId)
		if mob == nil || mob.Character.Aggro != nil || mob.Character.Health < 1 || mob.Character.IsCharmed() {
			continue
		}
		if _, _, companion := leaderOf(instanceId); companion || engaged(instanceId) {
			continue
		}
		hp, mana := 0, 0
		if mob.Character.Health < mob.Character.HealthLimit() {
			hp = enemyRecovery(mob.Character.HealthMax.Value)
		}
		if mob.Character.Mana < mob.Character.ManaMax.Value {
			mana = enemyRecovery(mob.Character.ManaMax.Value)
		}
		if hp > 0 || mana > 0 {
			mob.Character.Heal(hp, mana)
		}
	}
}
