package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/creatures"
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

	// Ashveil Phase 32d/33h2: companions regain health as players do.
	// Phase 35b: neither regains mana, and health only trickles back to
	// half of max (TrickleHeal).
	regenCompanionVitals(company.LeaderAndKeyForInstance)
	// Ashveil Phase 33i2: an enemy that lived through a fight recovers.
	regenEnemyVitals(evt.RoundNumber, company.LeaderAndKeyForInstance, battle.Engaged)

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
				user.SendText(`<ansi fg="red">you are bleeding out.</ansi>`)
				if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is <ansi fg="red">bleeding out</ansi>. Somebody needs to provide aid.`, user.Character.Name), user.UserId)
				}
			}

		} else {

			if user.Character.Health > 0 {
				user.Character.TrickleHeal(user.Character.HealthPerRound())
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

// regenCompanionVitals gives each living companion out of combat its
// health (Phase 33h2) per round, on the same every-third-round beat as
// players, while its leader is online. Mobs otherwise never recover: a
// companion's vitals are saved (33h2), so without this one hurt in a
// battle would stay hurt. Phase 35b: health trickles back only to half of
// max (TrickleHeal), and mana never returns this way: only a rest, an inn
// or a draught refills it.
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
		// Phase 38e: a construct never mends on its own; it is repaired
		// with mortar (company repair).
		if family, creature := creatures.ForArchetype(mob.Character.ArchetypeID()); creature && !family.MendsOnItsOwn() {
			continue
		}
		mob.Character.TrickleHeal(mob.Character.HealthPerRound())
	}
}

// EnemyRecoveryRounds is the rounds an enemy takes to recover from nothing
// to full health and mana out of a fight (Phase 33i2): 5 game hours at
// the shipped 900-round day, about 12½ real minutes.
const EnemyRecoveryRounds = 188

// enemyRecovery is what an enemy of max health or mana recovers on the
// pass at round: its share of the three rounds since the last pass, so
// that any 188 rounds in a row add up to max, however small max is (33i2
// review finding 1: rounding each round's share up healed a 40-health
// enemy in 40 rounds). No state is kept: the share comes from the round.
func enemyRecovery(max int, round uint64) int {
	if max < 1 {
		return 0
	}
	r := round % EnemyRecoveryRounds
	m := uint64(max)
	if r >= 3 {
		return int(m*r/EnemyRecoveryRounds - m*(r-3)/EnemyRecoveryRounds)
	}
	// The window wraps past a multiple of EnemyRecoveryRounds.
	return int(m - m*(r+EnemyRecoveryRounds-3)/EnemyRecoveryRounds + m*r/EnemyRecoveryRounds)
}

// regenEnemyVitals gives each living mob outside a company (an enemy, or
// any creature a fight left hurt) its share of health and mana while it is
// out of every battle and not fighting (Phase 33i2). Before, a mob that
// survived a fight stayed hurt until it died. It runs on rounds, so world
// time never advances for it; a charmed pet is left to its owner's rules.
// Health stops at the wound limit (Heal), though an enemy's light wounds
// have closed by the time its fight is over.
func regenEnemyVitals(round uint64, leaderOf func(instanceId int) (int, company.MemberKey, bool), engaged func(instanceId int) bool) {
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
			hp = enemyRecovery(mob.Character.HealthMax.Value, round)
		}
		if mob.Character.Mana < mob.Character.ManaMax.Value {
			mana = enemyRecovery(mob.Character.ManaMax.Value, round)
		}
		if hp > 0 || mana > 0 {
			mob.Character.Heal(hp, mana)
		}
	}
}
