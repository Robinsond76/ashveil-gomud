package hooks

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// combatMobCharacter is a presentation-only copy for combat narration. It
// keeps health, equipment, and every combat field from the live mob intact.
func combatMobCharacter(m *mobs.Mob) characters.Character {
	c := m.Character
	c.Name = battle.EnemyDisplayName(m.InstanceId, c.Name)
	return c
}

func DoCombat(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	// Ashveil Phase 29b: every event this round reports is stamped with it.
	combatRound.Store(evt.RoundNumber)
	resetRoundExtras()
	beginBattlefieldRound()

	// Ashveil Phase 30d1: a new round of shield counters; restarts owed by
	// mobs no longer chanting are dropped.
	interruptRound()

	// Ashveil Phase 29b2: each player fights one enemy group at a time.
	// Decide every player's battle first; battles opening and ending open
	// and end their fights on the combat event stream (29b).
	battlePass()

	// Ashveil Phase 30c2: spent guards come back, one per two combat
	// rounds.
	guardPass()

	// Ashveil Phase 30c: a focus ordered since the last round turns the
	// company at this round's upkeep.
	beginRefocus()
	defer endRefocus()

	// Ashveil Phase 29a: keep each player's company and the group they're
	// in battle with fighting as a whole before any blow is struck.
	upkeepEngagements()
	closeIdleBattles()

	// Ashveil Phase 30a: statuses tick once per combat round, before any
	// blow; a fall they cause is resolved at once.
	statusPass()

	// Ashveil Phase 32d: healers and casters cast by their strategies,
	// before any blow.
	beginTempoRound()
	defer func() { tempoActive = false }()
	auraPass()   // Phase 38b: class auras for the round
	summonPass() // Phase 38b: Angels and Demons
	nervePass()
	strategyPass()
	// Ashveil Phase 33i2: enemy healers and casters, by their group's
	// coordination.
	enemyStrategyPass()
	// Ashveil Phase 33e: members about to swing may use a class ability.
	abilityPass()
	defer endAbilityStrikes()
	retreatCover = map[int]bool{}

	//
	// Combat rounds
	//
	affectedPlayers1, affectedMobs1 := handlePlayerCombat(evt, false)

	affectedPlayers2, affectedMobs2 := handleMobCombat(evt, false)

	// Earned second physical turns reuse the same gates and attribution. Round
	// upkeep, chants and waits were already processed in the first pass.
	endAbilityStrikes()
	clear(battlefieldPowers)
	p3, m3 := handlePlayerCombat(evt, true)
	p4, m4 := handleMobCombat(evt, true)
	affectedPlayers1 = append(affectedPlayers1, append(p3, p4...)...)
	affectedMobs1 = append(affectedMobs1, append(m3, m4...)...)

	// Do any resolution or extra checks based on everyone that has been involved in combat this round.
	affectedPlayers := append(append(affectedPlayers1, affectedPlayers2...), roundExtraPlayers...)
	affectedMobs := append(append(affectedMobs1, affectedMobs2...), roundExtraMobs...)
	noteMoraleDeaths()
	handleAffected(affectedPlayers, affectedMobs)
	moralePass()

	// Ashveil Phase 29b2: end each battle one side of which has fallen, and
	// begin the next at once.
	settleBattles()
	snapshotTempoFights()
	pruneTempo()

	return events.Continue
}

func handlePlayerCombat(evt events.NewRound, extra bool) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	c := configs.GetConfig()

	tStart := time.Now()

	for _, userId := range users.GetOnlineUserIds() {

		user := users.GetByUserId(userId)

		if user == nil || user.Character == nil || user.Character.Aggro == nil {
			continue
		}
		if user.Character.Health < 1 {
			affectedPlayerIds = append(affectedPlayerIds, userId)
			continue
		}
		who := caster{userId: userId}
		if extra && !extraTempoTurn(who, user.Character) {
			continue
		}
		if !extra {
			fillTempo(who, user.Character)
		}
		// If has a buff that prevents combat, skip the player
		if user.Character.HasBuffFlag("no-combat") {
			continue
		}

		if user == nil || user.Character.Aggro == nil {
			continue
		}

		if surprised(user.UserId, 0) && user.Character.Aggro.Type != characters.Retreat {
			continue
		}

		// Ashveil Phase 30a: a staggered, downed, or stunned fighter loses
		// its action.
		if status.Has(user.Character) && tempoStatusCostsAction(userHolder(user)) {
			continue
		}

		// Disable any buffs that are cancelled by combat
		user.Character.CancelBuffsWithFlag("cancel-on-combat")

		// Ashveil Phase 33e: a tackle took this round's turn.
		if !extra && abilityTurns[caster{userId: userId}] {
			continue
		}

		roomId := user.Character.RoomId

		uRoom := rooms.LoadRoom(roomId)
		if uRoom == nil {
			continue
		}

		if user.Character.Aggro.Type != characters.SpellCast && user.Character.Aggro.Type != characters.Retreat && combat.ResolveReach(user.Character, false) != formationcombat.ReachAny && physicalReserve(userId, nil) {
			continue
		}
		if user.Character.Aggro.Type == characters.Retreat {
			handleRetreat(user, uRoom)
			continue
		}

		// Phase 33c owner review: flee is the retreat order now. A leftover
		// flight (none is ever set) just fights on.
		if user.Character.Aggro.Type == characters.Flee {
			user.Character.SetAggro(user.Character.Aggro.UserId, user.Character.Aggro.MobInstanceId, characters.DefaultAttack)
			continue
		}

		/**************************
		*
		* START HANDLING MAGIC
		*
		**************************/

		if user.Character.Aggro != nil && user.Character.Aggro.Type == characters.SpellCast {
			tempoBlocked[who] = true
			user.Character.Aggro.SpellInfo = effecttargets.Resolve(user.UserId, 0, user.Character.Aggro.SpellInfo)

			if user.Character.Aggro.RoundsWaiting > 0 {
				coldWait(userHolder(user))
				user.Character.Aggro.RoundsWaiting--

				scripting.TrySpellScriptEvent(`onWait`, user.UserId, 0, user.Character.Aggro.SpellInfo)
				emitCast(combatstream.CastProgress, userRef(user), user.Character.Aggro.SpellInfo.SpellId, ``, roomId)

				continue
			}

			// Phase 29b2: a harmful spell keeps to the caster's battle.
			if holdPlayerSpell(user.UserId, &user.Character.Aggro.SpellInfo) {
				user.SendText(`Your spell has no foe in your battle. The others wait their turn.`)
				emitCast(combatstream.CastComplete, userRef(user), user.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeHeld, roomId)
				endCast(user.Character, caster{userId: user.UserId}) // Phase 32d: back to its aim
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			// Phase 33b review: a helpful chant whose patients all left,
			// fell, or changed hands ends without effect, and says so.
			if effecttargets.Unaided(user.Character.Aggro.SpellInfo) {
				what := `spell`
				if user.Character.Aggro.SpellInfo.SpellId == `aidskill` {
					what = `aid`
				}
				user.SendText(fmt.Sprintf(`<ansi fg="spell-text">Your %s finds no one left to help, and fades.</ansi>`, what))
				emitCast(combatstream.CastComplete, userRef(user), user.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeWasted, roomId)
				endCast(user.Character, caster{userId: user.UserId})
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			// Phase 35b: an owned spell never fizzles in battle; elsewhere a
			// 100% chance never fails.
			roll := util.RollDice(1, 100)
			successChance := user.Character.GetBaseCastSuccessChance(user.Character.Aggro.SpellInfo.SpellId)
			if !playerCastsSure(user, user.Character.Aggro.SpellInfo.SpellId) && characters.CastFails(roll, successChance) {

				// fail
				user.SendText(fmt.Sprintf(`<ansi fg="spell-text">The words slip away from you, and your spell <ansi fg="magenta">fizzles</ansi>. (rolled %d against a %d%% chance)</ansi>`, roll, successChance))
				uRoom.SendText(fmt.Sprintf(`<ansi fg="spell-text"><ansi fg="username">%s</ansi> falters, and the spell <ansi fg="magenta">fizzles</ansi>.</ansi>`, user.Character.Name), userId)
				emitCast(combatstream.CastComplete, userRef(user), user.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeFizzled, roomId)
				endCast(user.Character, caster{userId: user.UserId}) // Phase 32d: back to its aim

				continue

			}

			//
			// Need to track before health to calculate if damage was done post-spell
			//
			mobHealthBefore := map[int]int{}
			for _, mInstId := range user.Character.Aggro.SpellInfo.TargetMobInstanceIds {
				if defMob := mobs.GetInstance(mInstId); defMob != nil {

					// Remember who has hit him
					defMob.Character.TrackPlayerDamage(user.UserId, 0)
					mobHealthBefore[mInstId] = defMob.Character.Health

				}
			}

			spellInfo := user.Character.Aggro.SpellInfo
			spellTargetsBefore := snapshotSpellTargets(spellInfo)

			allowRetaliation := true
			if handled, err := scripting.TrySpellScriptEvent(`onMagic`, user.UserId, 0, user.Character.Aggro.SpellInfo); err == nil {
				if handled {
					allowRetaliation = false
				}
			}

			emitCast(combatstream.CastComplete, userRef(user), spellInfo.SpellId, combatstream.OutcomeCast, roomId)
			spellTargetsBefore.emitResults(userRef(user), spellInfo.SpellId, roomId)

			user.Character.TrackSpellCast(user.Character.Aggro.SpellInfo.SpellId)

			if allowRetaliation {
				if spellData := spells.GetSpell(user.Character.Aggro.SpellInfo.SpellId); spellData != nil {

					if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmMulti || spellData.Type == spells.HarmArea {

						for _, mobId := range user.Character.Aggro.SpellInfo.TargetMobInstanceIds {

							affectedMobInstanceIds = append(affectedMobInstanceIds, mobId)

							if defMob := mobs.GetInstance(mobId); defMob != nil {

								// Track damage done
								if hBefore, ok := mobHealthBefore[mobId]; ok {
									hDelta := hBefore - defMob.Character.Health
									if hDelta > 0 {
										defMob.Character.TrackPlayerDamage(user.UserId, hDelta)
									}
								}

								defMob.Character.CancelBuffsWithFlag("cancel-on-combat")

								if defMob.Character.Health <= 0 {
									defMob.Character.EndAggro()
									events.AddToQueue(events.AggroChanged{MobInstanceId: defMob.InstanceId, RoomId: defMob.Character.RoomId})
								} else if defMob.Character.Aggro == nil {
									defMob.PreventIdle = true
									defMob.Command(fmt.Sprintf("attack @%d", user.UserId)) // @ means player
								}

							}
						}

					}
				}
			}

			endCast(user.Character, caster{userId: user.UserId}) // Phase 32d: back to its aim

			continue

		}

		/**************************
		*
		* END HANDLING MAGIC
		*
		**************************/

		/**************************
		*
		* START HANDLING PHYSICAL COMBAT
		*
		**************************/

		if tempoTurns[who] == 0 && user.Character.Aggro.RoundsWaiting == 0 || tempoChanted[who] {
			continue
		}

		// In combat with another player
		if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {

			defUser := users.GetByUserId(user.Character.Aggro.UserId)

			uRoom := rooms.LoadRoom(roomId)

			if uRoom == nil {
				user.Character.Aggro = nil
				continue
			}

			targetFound := true
			if defUser == nil {
				targetFound = false
			} else if defUser.Character.RoomId != user.Character.RoomId {

				if user.Character.Aggro.ExitName == `` {
					targetFound = false
				} else {
					// If the exitId doesn't match the target room id, can't find em
					if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defUser.Character.RoomId {
						targetFound = false
					}
				}

			}

			if !targetFound {
				user.SendText(`Your target can't be found.`)
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			defRoom := rooms.LoadRoom(defUser.Character.RoomId)
			if defRoom == nil {
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			defUser.Character.CancelBuffsWithFlag("cancel-on-combat")

			if defUser.Character.Health < 1 {
				user.SendText(`Your rage subsides.`)
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			if user.Character.Aggro.RoundsWaiting > 0 {
				tempoBlocked[who] = true
				mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)

				coldWait(userHolder(user))
				user.Character.Aggro.RoundsWaiting--

				roundResult := combat.GetWaitMessages(items.Wait, user.Character, defUser.Character, combat.User, combat.User)

				for _, msg := range roundResult.MessagesToSource {
					user.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToTarget {
					defUser.SendText(msg)
				}

				if len(roundResult.MessagesToSourceRoom) > 0 {
					for _, msg := range roundResult.MessagesToSourceRoom {
						uRoom.SendText(msg, user.UserId, defUser.UserId)
					}
				}

				if len(roundResult.MessagesToTargetRoom) > 0 {
					for _, msg := range roundResult.MessagesToTargetRoom {
						defRoom.SendText(msg, user.UserId, defUser.UserId)
					}
				}

				continue
			}

			// Can't see them, can't fight them.
			if defUser.Character.HasBuffFlag("hidden") {
				user.SendText("You can't seem to find your target.")
				continue
			}

			affectedPlayerIds = append(affectedPlayerIds, user.Character.Aggro.UserId)

			roundResult := combat.AttackPlayerVsPlayer(user, defUser)
			emitAttack(userRef(user), userRef(defUser), roomId, user.Character, roundResult)

			// If a mob attacks a player, check whether player has a charmed mob helping them, and if so, they will move to attack back
			room := rooms.LoadRoom(roomId)
			for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
				if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
					if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {

						// Set aggro to something to prevent multiple attack triggers on this conditional
						charmedMob.Character.Aggro = &characters.Aggro{
							Type: characters.DefaultAttack,
						}

						charmedMob.Command(fmt.Sprintf("attack @%d", user.UserId))

					}
				}
			}

			for _, buffId := range roundResult.BuffSource {
				user.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defUser.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSource {
				user.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToTarget {
				defUser.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				uRoom.SendText(msg, user.UserId, defUser.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, user.UserId, defUser.UserId)
			}

			// Ashveil Phase 30d1: a broken chant, or a shield's counter.
			afterBlow(userHolder(user), userHolder(defUser), roundResult)

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {

				defUser.Character.TrackPlayerDamage(user.UserId, roundResult.DamageToTarget)

				// For now, only focus on offhand items.
				if defUser.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defUser.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defUser.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						defUser.SendText(shieldBreaksOwnerLine(defUser.Character.Equipment.Offhand.NameSimple()))

						defRoom.SendText(shieldBreaksRoomLine(defUser.Character.Equipment.Offhand.NameSimple(), userTag(defUser.Character.Name)), defUser.UserId)

						events.AddToQueue(events.ItemOwnership{
							UserId: defUser.UserId,
							Item:   defUser.Character.Equipment.Offhand,
							Gained: false,
						})

						defUser.Character.RemoveFromBody(defUser.Character.Equipment.Offhand)

						itm := items.New(20) // Broken item
						if !defUser.Character.StoreItem(itm) {
							room.AddItem(itm, false)

							events.AddToQueue(events.ItemOwnership{
								UserId: defUser.UserId,
								Item:   itm,
								Gained: true,
							})
						}
					}
				}
			}

			if roundResult.DamageToSource != 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
			}
			if roundResult.DamageToTarget != 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: defUser.UserId})
			}

			if user.Character.Health <= 0 || defUser.Character.Health <= 0 {
				defUser.Character.EndAggro()
				user.Character.EndAggro()
			} else {
				user.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
			}

		}

		// In combat with a mob
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {

			affectedMobInstanceIds = append(affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

			defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId)
			// A kill earlier in this round did not spend this fighter's turn.
			// Select a legal replacement before resolving the earned blow.
			if defMob != nil && defMob.Character.Health < 1 {
				previous := user.Character.Aggro
				if reassignPlayerTarget(user, uRoom) {
					// Replacement did not fire the weapon: retain its readiness.
					user.Character.Aggro.RoundsWaiting = previous.RoundsWaiting
					user.Character.Aggro.ColdDelayed = previous.ColdDelayed
					user.Character.Aggro.ColdNotice = previous.ColdNotice
					defMob = mobs.GetInstance(user.Character.Aggro.MobInstanceId)
					affectedMobInstanceIds = append(affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)
				}
			}

			targetFound := true
			if defMob == nil {
				targetFound = false
			} else if defMob.Character.RoomId != user.Character.RoomId {

				if user.Character.Aggro.ExitName == `` {
					targetFound = false
				} else {
					// Make sure the target is still at the exit

					uRoom := rooms.LoadRoom(roomId)
					if uRoom == nil {
						user.Character.Aggro = nil
						continue
					}

					// If the exitId doesn't match the target room id, can't find em
					if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defMob.Character.RoomId {
						targetFound = false
					}

				}

			}

			if !targetFound {
				if reassignPlayerTarget(user, uRoom) {
					continue
				}
				user.SendText("Your target can't be found.")
				user.Character.Aggro = nil
				continue
			}

			defRoom := rooms.LoadRoom(defMob.Character.RoomId)

			defMob.Character.CancelBuffsWithFlag("cancel-on-combat")

			if defMob.Character.Health < 1 {
				if reassignPlayerTarget(user, uRoom) {
					continue
				}
				user.SendText("Your rage subsides.")
				user.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				continue
			}

			if user.Character.Aggro.RoundsWaiting > 0 {
				tempoBlocked[who] = true
				mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)

				coldWait(userHolder(user))
				user.Character.Aggro.RoundsWaiting--

				defCharacter := combatMobCharacter(defMob)
				roundResult := combat.GetWaitMessages(items.Wait, user.Character, &defCharacter, combat.User, combat.Mob)

				for _, msg := range roundResult.MessagesToSource {
					user.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToSourceRoom {
					uRoom.SendText(msg, user.UserId)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg, user.UserId)
				}

				continue
			}

			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag("hidden") {
				user.SendText("You can't seem to find your target.")
				continue
			}

			// Ashveil Phase 29b2: a foe outside the player's battle waits. A
			// plain attack is turned onto the battle (keepOnBattle); any
			// other (a backstab, a shot) is called off.
			if playerHolds(user.UserId, defMob) {
				if user.Character.Aggro.Type != characters.DefaultAttack {
					user.SendText(`That foe is waiting its turn. Finish your battle first.`)
					user.Character.Aggro = nil
					events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				}
				continue
			}

			if gated, gateOk := gateFormationAttack(user, defMob, uRoom); !gateOk {
				user.SendText("You can't reach that target from here.")
				continue
			} else {
				defMob = gated
			}

			affectedPlayerIds = append(affectedPlayerIds, user.Character.Aggro.UserId)

			var roundResult combat.AttackResult

			roundResult = combat.AttackPlayerVsMob(user, defMob)
			emitAttack(userRef(user), mobRef(defMob), roomId, user.Character, roundResult)

			for _, buffId := range roundResult.BuffSource {
				user.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defMob.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSource {
				user.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				uRoom.SendText(msg, user.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, user.UserId)
			}

			// Ashveil Phase 30d1: a broken chant, or a shield's counter.
			afterBlow(userHolder(user), mobHolder(defMob), roundResult)

			// Handle any scripted behavior now.
			if roundResult.Hit {
				scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, user.UserId, `user`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
			}

			//
			// Special mob-only reaction/behavior
			//
			// Hostility default to 5 minutes
			for _, groupName := range defMob.Groups {
				mobs.MakeHostile(groupName, user.UserId, c.Timing.MinutesToRounds(2)-user.Character.Stats.Perception.ValueAdj)
			}

			// Mobs get aggro when attacked
			if defMob.Character.Aggro == nil {
				defMob.PreventIdle = true
				// If not in the same room,
				// find an exit to the room of the player to move to
				if user.Character.RoomId != defMob.Character.RoomId {
					if mobRoom := rooms.LoadRoom(defMob.Character.RoomId); mobRoom != nil {
						for exitName, exitInfo := range mobRoom.Exits {
							if exitInfo.RoomId == user.Character.RoomId {
								defMob.Command(fmt.Sprintf(`go %s`, exitName))
								if actionStr := defMob.GetAngryCommand(); actionStr != `` {
									defMob.Command(actionStr)
								}
								break
							}
						}
					}
				}

				defMob.Command(fmt.Sprintf("attack @%d", user.UserId)) // @ means player
			}

			if roundResult.DamageToSource != 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
			}

			if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
				defMob.Character.EndAggro()
				events.AddToQueue(events.AggroChanged{MobInstanceId: defMob.InstanceId, RoomId: defMob.Character.RoomId})
				if user.Character.Health <= 0 || !reassignPlayerTarget(user, uRoom) {
					user.Character.EndAggro()
					events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
				}
			} else {
				user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
				events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})
			}

		}

		/**************************
		*
		* END HANDLING PHYSICAL COMBAT
		*
		**************************/

	}

	util.TrackTime(`DoCombat::handlePlayerCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

func handleMobCombat(evt events.NewRound, extra bool) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	tStart := time.Now()

	// Handle mob round of combat
	for _, mobId := range mobs.GetAllMobInstanceIds() {

		// Ashveil Phase 30d2: the last mob's wind-up blow, if its swing
		// never happened, is told as wasted.
		finishLanding()

		mob := mobs.GetInstance(mobId)

		// Only handling combat functions here, so ditch out if not in combat
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}

		who := caster{mobId: mobId}
		if mob.Character.Health < 1 {
			affectedMobInstanceIds = append(affectedMobInstanceIds, mobId)
			continue
		}
		if extra && !extraTempoTurn(who, &mob.Character) {
			continue
		}
		if !extra {
			fillTempo(who, &mob.Character)
		}
		// If has a buff that prevents combat, skip the player
		if mob.Character.CombatWithdrawn || retreatCover[mobId] || nerveSkip[mobId] || mob.Character.HasBuffFlag("no-combat") {
			continue
		}

		if surprised(0, mobId) {
			continue
		}

		// Ashveil Phase 30a: a staggered, downed, or stunned fighter loses
		// its action (and Phase 30d2: a wind-up with it).
		if status.Has(&mob.Character) && tempoStatusCostsAction(mobHolder(mob)) {
			windUpLostTurn(mob)
			continue
		}

		mobRoom := rooms.LoadRoom(mob.Character.RoomId)

		if mobRoom == nil {
			mob.Character.Aggro = nil
			events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
			continue
		}

		if mob.Character.Aggro.Type != characters.SpellCast && combat.ResolveReach(&mob.Character, mob.Reach) != formationcombat.ReachAny && physicalReserve(0, mob) {
			continue
		}
		// Disable any buffs that are cancelled by combat
		mob.Character.CancelBuffsWithFlag("cancel-on-combat")

		// Ashveil Phase 33e: a tackle took this round's turn.
		if !extra && abilityTurns[caster{mobId: mobId}] {
			continue
		}

		/**************************
		*
		* START HANDLING MAGIC
		*
		**************************/

		if mob.Character.Aggro != nil && mob.Character.Aggro.Type == characters.SpellCast {
			tempoBlocked[who] = true
			mob.Character.Aggro.SpellInfo = effecttargets.Resolve(0, mob.InstanceId, mob.Character.Aggro.SpellInfo)

			// Ashveil Phase 30d1: a chant a blow broke starts again from
			// the first word, taking this turn.
			if restartChant(mob) {
				continue
			}

			if mob.Character.Aggro.RoundsWaiting > 0 {
				tempoBlocked[who] = true
				coldWait(mobHolder(mob))
				mob.Character.Aggro.RoundsWaiting--

				scripting.TrySpellScriptEvent(`onWait`, 0, mob.InstanceId, mob.Character.Aggro.SpellInfo)
				emitCast(combatstream.CastProgress, mobRef(mob), mob.Character.Aggro.SpellInfo.SpellId, ``, mob.Character.RoomId)

				continue
			}

			// Phase 29b2: a harmful spell waits its turn against a player
			// fighting another group, and the mob keeps its place in line.
			if held, waitOn := holdMobSpell(mob, &mob.Character.Aggro.SpellInfo); held {
				emitCast(combatstream.CastComplete, mobRef(mob), mob.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeHeld, mob.Character.RoomId)
				endCast(&mob.Character, caster{mobId: mob.InstanceId}) // Phase 32d: back to its aim
				if waitOn > 0 {
					mob.Character.SetAggro(waitOn, 0, characters.DefaultAttack)
				}
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			// Phase 33b review: as for players, a helpful chant with no one
			// left to help ends without effect.
			if effecttargets.Unaided(mob.Character.Aggro.SpellInfo) {
				emitCast(combatstream.CastComplete, mobRef(mob), mob.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeWasted, mob.Character.RoomId)
				endCast(&mob.Character, caster{mobId: mob.InstanceId})
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			// Phase 35b: as for players.
			successChance := mob.Character.GetBaseCastSuccessChance(mob.Character.Aggro.SpellInfo.SpellId)
			if !mobCastsSure(mob, mob.Character.Aggro.SpellInfo.SpellId) && characters.CastFails(util.RollDice(1, 100), successChance) {

				// fail
				mobRoom.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s falters, and the spell <ansi fg="magenta">fizzles</ansi>.`, util.Article(mobTag(mobName(mob.InstanceId))))))
				emitCast(combatstream.CastComplete, mobRef(mob), mob.Character.Aggro.SpellInfo.SpellId, combatstream.OutcomeFizzled, mob.Character.RoomId)
				endCast(&mob.Character, caster{mobId: mob.InstanceId}) // Phase 32d: back to its aim

				continue

			}

			spellInfo := mob.Character.Aggro.SpellInfo
			spellTargetsBefore := snapshotSpellTargets(spellInfo)

			allowRetaliation := true
			if handled, err := scripting.TrySpellScriptEvent(`onMagic`, 0, mob.InstanceId, mob.Character.Aggro.SpellInfo); err == nil {
				if handled {
					allowRetaliation = false
				}
			}

			emitCast(combatstream.CastComplete, mobRef(mob), spellInfo.SpellId, combatstream.OutcomeCast, mob.Character.RoomId)
			spellTargetsBefore.emitResults(mobRef(mob), spellInfo.SpellId, mob.Character.RoomId)

			if allowRetaliation {
				if spellData := spells.GetSpell(mob.Character.Aggro.SpellInfo.SpellId); spellData != nil {

					if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmMulti || spellData.Type == spells.HarmArea {

						for _, mobId := range mob.Character.Aggro.SpellInfo.TargetMobInstanceIds {

							affectedMobInstanceIds = append(affectedMobInstanceIds, mobId)

							if defMob := mobs.GetInstance(mobId); defMob != nil {

								defMob.Character.CancelBuffsWithFlag("cancel-on-combat")

								if defMob.Character.Health <= 0 {
									defMob.Character.EndAggro()
								} else if defMob.Character.Aggro == nil {
									defMob.PreventIdle = true
									defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId)) // # means mob
								}

							}
						}

					}
				}
			}

			endCast(&mob.Character, caster{mobId: mob.InstanceId}) // Phase 32d: back to its aim

			continue

		}

		/**************************
		*
		* END HANDLING MAGIC
		*
		**************************/

		/**************************
		*
		* START HANDLING PHYSICAL COMBAT
		*
		**************************/
		c := configs.GetConfig()

		// Ashveil Phase 30d2: a wind-up takes the turn, or makes this
		// turn's swing its blow.
		if !extra && mob.Character.Aggro.Type == characters.DefaultAttack {
			if windUpTurn(mob) {
				tempoBlocked[who] = true
				continue
			}
			if isLanding(mob.InstanceId) {
				tempoBlocked[who] = true
			}
		}
		if tempoChanted[who] || tempoTurns[who] == 0 && mob.Character.Aggro.RoundsWaiting == 0 && !isLanding(mob.InstanceId) {
			continue
		}

		// H2H is the base level combat, can do combat commands then
		// (not while a wind-up's blow lands, Phase 30d2)
		if !extra && mob.Character.Aggro.Type == characters.DefaultAttack && !isLanding(mob.InstanceId) {

			// If they have idle commands, maybe do one of them?
			cmdCt := len(mob.CombatCommands)
			if cmdCt > 0 {

				// Each mob has a 10% chance of doing an idle action.
				if util.Rand(100) < mob.ActivityLevel {

					combatAction := mob.CombatCommands[util.Rand(cmdCt)]

					if combatAction == `` { // blank is a no-op
						continue
					}

					var waitTime float64 = 0.0
					allCmds := strings.Split(combatAction, `;`)
					if len(allCmds) >= c.Timing.TurnsPerRound() {
						mob.Command(`say I have a CombatAction that is too long. Please notify an admin.`)
					} else {
						for _, action := range strings.Split(combatAction, `;`) {
							mob.Command(action, waitTime)
							waitTime += 0.1
						}
					}
					continue
				}
			}

		}
		roomId := mob.Character.RoomId

		affectedMobInstanceIds = append(affectedMobInstanceIds, mob.InstanceId)

		// A kill earlier in this round did not spend this fighter's turn
		// (30g6a): a companion or an enemy group member picks a legal
		// replacement, keeping its weapon's readiness, before its blow.
		if a := mob.Character.Aggro; a != nil && aggroTargetDown(a) {
			if reassignCompanionTarget(mob, mobRoom) || reassignEnemyTarget(mob, mobRoom) {
				mob.Character.Aggro.RoundsWaiting = a.RoundsWaiting
				mob.Character.Aggro.ColdDelayed = a.ColdDelayed
				mob.Character.Aggro.ColdNotice = a.ColdNotice
			}
		}

		// mob attacks player
		if mob.Character.Aggro != nil && mob.Character.Aggro.UserId > 0 {

			defUser := users.GetByUserId(mob.Character.Aggro.UserId)
			if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			defRoom := rooms.LoadRoom(defUser.Character.RoomId)
			if defRoom == nil {
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			defUser.Character.CancelBuffsWithFlag("cancel-on-combat")

			if defUser.Character.Health < 1 {
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}
			if defUser.Character.HasBuffFlag("hidden") {
				continue
			}

			affectedPlayerIds = append(affectedPlayerIds, mob.Character.Aggro.UserId)

			// If no weapon but has stuff in the backpack, look for a weapon
			// Especially useful for when they get disarmed
			if mob.Character.Equipment.Weapon.ItemId == 0 && len(mob.Character.Items) > 0 {

				roll := util.Rand(100)

				util.LogRoll(`Look for weapon`, roll, mob.Character.Stats.Perception.ValueAdj)

				if roll < mob.Character.Stats.Perception.ValueAdj {
					possibleWeapons := []string{}
					for _, itm := range mob.Character.Items {
						iSpec := itm.GetSpec()
						if iSpec.Type == items.Weapon {
							possibleWeapons = append(possibleWeapons, itm.DisplayName())
						}
					}

					if len(possibleWeapons) > 0 {
						mob.Command(fmt.Sprintf("equip %s", possibleWeapons[util.Rand(len(possibleWeapons))]))
					}

				}
			}

			if mob.Character.Aggro.RoundsWaiting > 0 {
				tempoBlocked[who] = true
				mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)

				coldWait(mobHolder(mob))
				mob.Character.Aggro.RoundsWaiting--

				mobCharacter := combatMobCharacter(mob)
				roundResult := combat.GetWaitMessages(items.Wait, &mobCharacter, defUser.Character, combat.Mob, combat.User)

				for _, msg := range roundResult.MessagesToTarget {
					defUser.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToSourceRoom {
					mobRoom.SendText(msg, defUser.UserId)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg, defUser.UserId)
				}

				continue
			}

			// Ashveil Phase 29b2: a group waiting its turn holds back.
			if holdsAgainstPlayer(mob, defUser.UserId) {
				continue
			}

			if trySweep(mob, defUser.UserId, mobRoom) {
				continue
			}
			if handled, gateOk := gateMobVsPlayerAttack(mob, defUser, mobRoom, defRoom); !gateOk {
				continue
			} else if handled {
				continue
			}

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsPlayer(mob, defUser)
			emitAttack(mobRef(mob), userRef(defUser), roomId, &mob.Character, roundResult)

			// If a mob attacks a player, check whether player has a charmed mob helping them, and if so, they will move to attack back
			room := rooms.LoadRoom(roomId)
			for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
				if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
					if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {
						// This is set to prevent it from triggering more than once
						charmedMob.Character.Aggro = &characters.Aggro{
							Type: characters.DefaultAttack,
						}

						charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))

					}
				}
			}

			for _, buffId := range roundResult.BuffSource {
				mob.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defUser.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToTarget {
				defUser.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				mobRoom.SendText(msg, defUser.UserId)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg, defUser.UserId)
			}

			// Ashveil Phase 30d1: a broken chant, or a shield's counter.
			afterBlow(mobHolder(mob), userHolder(defUser), roundResult)

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {

				// For now, only focus on offhand items.
				if defUser.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defUser.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defUser.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						defUser.SendText(shieldBreaksOwnerLine(defUser.Character.Equipment.Offhand.NameSimple()))

						defRoom.SendText(shieldBreaksRoomLine(defUser.Character.Equipment.Offhand.NameSimple(), userTag(defUser.Character.Name)), defUser.UserId)

						events.AddToQueue(events.ItemOwnership{
							UserId: defUser.UserId,
							Item:   defUser.Character.Equipment.Offhand,
							Gained: false,
						})

						defUser.Character.RemoveFromBody(defUser.Character.Equipment.Offhand)

						itm := items.New(20) // Broken item
						if !defUser.Character.StoreItem(itm) {
							room.AddItem(itm, false)

							events.AddToQueue(events.ItemOwnership{
								UserId: defUser.UserId,
								Item:   itm,
								Gained: true,
							})

						}
					}
				}
			}

			if roundResult.DamageToTarget != 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: defUser.UserId})
			}

			if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
				if mob.Character.Health <= 0 || !reassignEnemyTarget(mob, mobRoom) {
					mob.Character.EndAggro()
					events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				}
				defUser.Character.EndAggro()
				events.AddToQueue(events.AggroChanged{UserId: defUser.UserId, RoomId: defUser.Character.RoomId})
			} else {
				mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
			}
		}

		// mob attacks mob
		if mob.Character.Aggro != nil && mob.Character.Aggro.MobInstanceId > 0 {

			affectedMobInstanceIds = append(affectedMobInstanceIds, mob.Character.Aggro.MobInstanceId)

			defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId)

			if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
				if reassignCompanionTarget(mob, mobRoom) {
					continue
				}
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			defRoom := rooms.LoadRoom(defMob.Character.RoomId)

			defMob.Character.CancelBuffsWithFlag("cancel-on-combat")

			if defMob.Character.Health < 1 {
				if reassignCompanionTarget(mob, mobRoom) {
					continue
				}
				mob.Character.Aggro = nil
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				continue
			}

			if mob.Character.Aggro.RoundsWaiting > 0 {
				tempoBlocked[who] = true
				mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)

				coldWait(mobHolder(mob))
				mob.Character.Aggro.RoundsWaiting--

				mobCharacter := combatMobCharacter(mob)
				defCharacter := combatMobCharacter(defMob)
				roundResult := combat.GetWaitMessages(items.Wait, &mobCharacter, &defCharacter, combat.Mob, combat.Mob)

				for _, msg := range roundResult.MessagesToSourceRoom {
					mobRoom.SendText(msg)
				}

				for _, msg := range roundResult.MessagesToTargetRoom {
					defRoom.SendText(msg)
				}

				continue
			}

			// Can't see them, can't fight them.
			if defMob.Character.HasBuffFlag("hidden") {
				continue
			}

			// Ashveil Phase 29b2: a blow outside a player's battle waits.
			if mobHolds(mob, defMob) {
				continue
			}

			if owner, _, ok := company.LeaderAndKeyForInstance(defMob.InstanceId); ok && trySweep(mob, owner, mobRoom) {
				continue
			}
			if gated, handled, gateOk := gateMobVsMobAttack(mob, defMob, mobRoom); !gateOk {
				continue
			} else if handled {
				continue
			} else {
				defMob = gated
			}

			var roundResult combat.AttackResult

			roundResult = combat.AttackMobVsMob(mob, defMob)
			emitAttack(mobRef(mob), mobRef(defMob), mob.Character.RoomId, &mob.Character, roundResult)

			for _, buffId := range roundResult.BuffSource {
				mob.AddBuff(buffId, `combat`)
			}

			for _, buffId := range roundResult.BuffTarget {
				defMob.AddBuff(buffId, `combat`)
			}

			for _, msg := range roundResult.MessagesToSourceRoom {
				mobRoom.SendText(msg)
			}

			for _, msg := range roundResult.MessagesToTargetRoom {
				defRoom.SendText(msg)
			}

			// Ashveil Phase 30d1: a broken chant, or a shield's counter.
			afterBlow(mobHolder(mob), mobHolder(defMob), roundResult)

			// Handle any scripted behavior now.
			if roundResult.Hit {
				scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, mob.InstanceId, `mob`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
			}

			// Mobs get aggro when attacked
			if defMob.Character.Aggro == nil {
				defMob.PreventIdle = true
				defMob.Character.Aggro = &characters.Aggro{
					Type: characters.DefaultAttack,
				}
				defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId)) // # means mob
			}

			// If the attack connected, check for damage to equipment.
			if roundResult.Hit {
				// For now, only focus on offhand items.
				if defMob.Character.Equipment.Offhand.ItemId > 0 {

					modifier := 0
					if roundResult.Crit { // Crits double the chance of breakage for offhand items.
						modifier = int(defMob.Character.Equipment.Offhand.GetSpec().BreakChance)
					}

					if defMob.Character.Equipment.Offhand.BreakTest(modifier) {
						// Send message about the break

						if defRoom := rooms.LoadRoom(defMob.Character.RoomId); defRoom != nil {

							defRoom.SendText(shieldBreaksRoomLine(defMob.Character.Equipment.Offhand.NameSimple(), mobTag(mobName(defMob.InstanceId))))

							events.AddToQueue(events.ItemOwnership{
								MobInstanceId: defMob.InstanceId,
								Item:          defMob.Character.Equipment.Offhand,
								Gained:        false,
							})

							defMob.Character.RemoveFromBody(defMob.Character.Equipment.Offhand)
							itm := items.New(20) // Broken item
							if !defMob.Character.StoreItem(itm) {
								defRoom.AddItem(itm, false)

								events.AddToQueue(events.ItemOwnership{
									MobInstanceId: defMob.InstanceId,
									Item:          itm,
									Gained:        true,
								})
							}
						}
					}
				}
			}

			if mob.Character.Health <= 0 || defMob.Character.Health <= 0 {
				if mob.Character.Health <= 0 || !(reassignCompanionTarget(mob, mobRoom) || reassignEnemyTarget(mob, mobRoom)) {
					mob.Character.EndAggro()
					events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
				}
				defMob.Character.EndAggro()
				events.AddToQueue(events.AggroChanged{MobInstanceId: defMob.InstanceId, RoomId: defMob.Character.RoomId})
			} else {
				mob.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
				events.AddToQueue(events.AggroChanged{MobInstanceId: mob.InstanceId, RoomId: mob.Character.RoomId})
			}

		}

		/**************************
		*
		* END HANDLING PHYSICAL COMBAT
		*
		**************************/

	}

	finishLanding() // Phase 30d2

	util.TrackTime(`World::handleMobCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

func handleAffected(affectedPlayerIds []int, affectedMobInstanceIds []int) {

	playersHandled := map[int]struct{}{}
	for _, userId := range affectedPlayerIds {
		if _, ok := playersHandled[userId]; ok {
			continue
		}
		playersHandled[userId] = struct{}{}

		if user := users.GetByUserId(userId); user != nil {

			if user.Character.Health <= -10 {

				emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: user.Character.RoomId, Target: userRef(user), Outcome: combatstream.OutcomeSlain})

				if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
					user.Character.KillerMobInstanceId = user.Character.Aggro.MobInstanceId
					if killerMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); killerMob != nil {
						user.Character.KillerMobIsElite = killerMob.IsElite
						user.Character.KillerMobName = killerMob.Character.Name
					}
				}
				user.Command(`suicide`) // suicide drops all money/items and transports to land of the dead.

			} else if user.Character.Health < 1 {

				emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: user.Character.RoomId, Target: userRef(user), Outcome: combatstream.OutcomeIncapacitated})

				events.AddToQueue(events.PlayerDrop{UserId: user.UserId, RoomId: user.Character.RoomId})

			}

		}
	}

	mobsHandled := map[int]struct{}{}
	for _, mobId := range affectedMobInstanceIds {
		if _, ok := mobsHandled[mobId]; ok {
			continue
		}
		mobsHandled[mobId] = struct{}{}

		if mob := mobs.GetInstance(mobId); mob != nil {
			if mob.Character.Health < 1 {

				outcome := combatstream.OutcomeSlain
				if mob.Practice {
					outcome = combatstream.OutcomeBeaten // the tutorial's practice foes are beaten, not killed
				}
				emitCombat(combatstream.Event{Kind: combatstream.Death, RoomId: mob.Character.RoomId, Target: mobRef(mob), Outcome: outcome})

				// Phase 29c: the death line now, in the round's order (the
				// fight's closing line follows at once); a mob that will
				// revive keeps suicide's own path.
				if mob.Character.HasBuffFlag("revive-on-death") {
					mob.Command(`suicide`)
				} else {
					mobcommands.CaptureRewardContributors(mob)
					mobDeathNotice(mob)
					mob.Command(`suicide quiet`)
				}

			}
		}

	}

}

// aggroTargetDown reports whether an Aggro's player or mob target is still
// present but has fallen (health below 1).
func aggroTargetDown(a *characters.Aggro) bool {
	if a.UserId > 0 {
		u := users.GetByUserId(a.UserId)
		return u != nil && u.Character.Health < 1
	}
	if a.MobInstanceId > 0 {
		m := mobs.GetInstance(a.MobInstanceId)
		return m != nil && m.Character.Health < 1
	}
	return false
}
