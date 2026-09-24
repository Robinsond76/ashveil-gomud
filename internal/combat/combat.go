package combat

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type SourceTarget string

const (
	User SourceTarget = "user"
	Mob  SourceTarget = "mob"
)

// Performs a combat round from a player to a mob
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {

	penalty := darknessPenalty(rooms.LoadRoom(user.Character.RoomId), &mob.Character, func(r *rooms.Room) int { return r.VisibilityForUser(user) })
	attackResult := calculateCombat(*user.Character, mob.Character, User, Mob, penalty, company.ChemistryBonusForUser(user.UserId))
	spendEdges(user.Character, attackResult.EdgeSpent)

	if attackResult.DamageToSource != 0 {
		user.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
		user.WimpyCheck()
	}

	mob.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)

	// Remember who has hit him
	mob.Character.TrackPlayerDamage(user.UserId, attackResult.DamageToTarget)

	if attackResult.Hit {
		user.PlaySound(`hit-other`, `combat`)
	} else {
		user.PlaySound(`miss`, `combat`)
	}

	return attackResult
}

// Performs a combat round from a player to a player
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {

	penalty := darknessPenalty(rooms.LoadRoom(userAtk.Character.RoomId), userDef.Character, func(r *rooms.Room) int { return r.VisibilityForUser(userAtk) })
	attackResult := calculateCombat(*userAtk.Character, *userDef.Character, User, User, penalty, company.ChemistryBonusForUser(userAtk.UserId))
	spendEdges(userAtk.Character, attackResult.EdgeSpent)

	if attackResult.DamageToSource != 0 {
		userAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
		userAtk.WimpyCheck()
	}

	if attackResult.DamageToTarget != 0 {
		userDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
		userDef.WimpyCheck()
	}

	if attackResult.Hit {
		userAtk.PlaySound(`hit-other`, `combat`)
		userDef.PlaySound(`hit-self`, `combat`)
	} else {
		userAtk.PlaySound(`miss`, `combat`)
	}

	return attackResult
}

// Performs a combat round from a mob to a player
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {

	penalty := darknessPenalty(rooms.LoadRoom(mob.Character.RoomId), user.Character, func(r *rooms.Room) int { return r.VisibilityForMob(mob) })
	attackResult := calculateCombat(mob.Character, *user.Character, Mob, User, penalty, company.ChemistryBonusForInstance(mob.InstanceId))
	spendEdges(&mob.Character, attackResult.EdgeSpent)

	mob.Character.ApplyHealthChange(attackResult.DamageToSource * -1)

	if attackResult.DamageToTarget != 0 {
		user.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
		user.WimpyCheck()
	}

	if attackResult.Hit {
		user.PlaySound(`hit-self`, `combat`)
	}

	return attackResult
}

// Performs a combat round from a mob to a mob
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {

	penalty := darknessPenalty(rooms.LoadRoom(mobAtk.Character.RoomId), &mobDef.Character, func(r *rooms.Room) int { return r.VisibilityForMob(mobAtk) })
	attackResult := calculateCombat(mobAtk.Character, mobDef.Character, Mob, Mob, penalty, company.ChemistryBonusForInstance(mobAtk.InstanceId))
	spendEdges(&mobAtk.Character, attackResult.EdgeSpent)

	mobAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
	mobDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)

	// If attacking mob was player charmed, attribute damage done to that player
	if charmedUserId := mobAtk.Character.GetCharmedUserId(); charmedUserId > 0 {
		// Remember who has hit him
		mobDef.Character.TrackPlayerDamage(charmedUserId, attackResult.DamageToTarget)
	}

	return attackResult
}

func GetWaitMessages(stepType items.Intensity, sourceChar *characters.Character, targetChar *characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	msgs := items.GetPreAttackMessage(sourceChar.Equipment.Weapon.GetSpec().Subtype, stepType)

	// zero means randomly selected, otherwise use the ItemId to consistently choose a message
	msgSeed := 0
	if configs.GetCombatConfig().ConsistentAttackMessages {
		msgSeed = sourceChar.Equipment.Weapon.ItemId
	}

	weaponName := races.GetRace(sourceChar.GetRaceId()).UnarmedName
	if sourceChar.Equipment.Weapon.ItemId > 0 {
		weaponName = sourceChar.Equipment.Weapon.DisplayName()
	}

	toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg := buildCombatMessages(
		sourceChar, targetChar, sourceType, targetType,
		weaponName, `[Invalid]`, msgSeed,
		msgs.Together.ToAttacker, msgs.Together.ToDefender, msgs.Together.ToRoom, items.MessageOptions(nil),
		msgs.Separate.ToAttacker, msgs.Separate.ToDefender, msgs.Separate.ToAttackerRoom, msgs.Separate.ToDefenderRoom,
	)

	if string(toAttackerMsg) != `` {
		attackResult.SendToSource(string(toAttackerMsg))
	}

	if !sourceChar.HasBuffFlag("hidden") {

		if string(toDefenderMsg) != `` {
			attackResult.SendToTarget(string(toDefenderMsg))
		}

		if string(toAttackerRoomMsg) != `` {
			attackResult.SendToSourceRoom(string(toAttackerRoomMsg))
		}

		if sourceChar.RoomId != targetChar.RoomId {
			if string(toDefenderRoomMsg) != `` {
				attackResult.SendToTargetRoom(string(toDefenderRoomMsg))
			}
		}

	}

	return attackResult
}

// buildCombatMessages resolves token replacements and selects the correct
// together/separate message variants for a single combat message event.
// It returns the four populated message strings ready to send.
// Pass nil (or empty MessageOptions) for togetherToDefenderRoom to suppress that message slot.
func buildCombatMessages(
	sourceChar *characters.Character, targetChar *characters.Character,
	sourceType SourceTarget, targetType SourceTarget,
	weaponName string, damageStr string, msgSeed int,
	togetherToAttacker, togetherToDefender, togetherToRoom, togetherToDefenderRoom items.MessageOptions,
	separateToAttacker, separateToDefender, separateToAttackerRoom, separateToDefenderRoom items.MessageOptions,
) (toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage) {

	tokenReplacements := map[items.TokenName]string{
		items.TokenItemName:     weaponName,
		items.TokenSource:       sourceChar.Name,
		items.TokenSourceType:   string(sourceType) + `name`,
		items.TokenTarget:       targetChar.Name,
		items.TokenTargetType:   string(targetType) + `name`,
		items.TokenUsesLeft:     `[Invalid]`,
		items.TokenDamage:       damageStr,
		items.TokenEntranceName: `unknown`,
		items.TokenExitName:     `unknown`,
	}

	if sourceType == Mob {
		tokenReplacements[items.TokenSource] = sourceChar.GetMobName(0).String()
	}

	if targetType == Mob {
		tokenReplacements[items.TokenTarget] = targetChar.GetMobName(0).String()
	}

	if sourceChar.RoomId == targetChar.RoomId {

		toAttackerMsg = togetherToAttacker.Get(msgSeed)
		toDefenderMsg = togetherToDefender.Get(msgSeed)
		toAttackerRoomMsg = togetherToRoom.Get(msgSeed)
		toDefenderRoomMsg = togetherToDefenderRoom.Get(msgSeed)

	} else {

		toAttackerMsg = separateToAttacker.Get(msgSeed)
		toDefenderMsg = separateToDefender.Get(msgSeed)
		toAttackerRoomMsg = separateToAttackerRoom.Get(msgSeed)
		toDefenderRoomMsg = separateToDefenderRoom.Get(msgSeed)

		// Find the exit that leads to the target from the source (if any)
		if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
			for exitName, exit := range atkRoom.Exits {
				if exit.RoomId == targetChar.RoomId {
					tokenReplacements[items.TokenExitName] = exitName
					break
				}
			}
		}
		// Find the exit that leads to the source from the target (if any)
		if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
			for exitName, exit := range defRoom.Exits {
				if exit.RoomId == sourceChar.RoomId {
					tokenReplacements[items.TokenEntranceName] = exitName
					break
				}
			}
		}
	}

	for tokenName, tokenValue := range tokenReplacements {
		toAttackerMsg = toAttackerMsg.SetTokenValue(tokenName, tokenValue)
		toDefenderMsg = toDefenderMsg.SetTokenValue(tokenName, tokenValue)
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(tokenName, tokenValue)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = toDefenderRoomMsg.SetTokenValue(tokenName, tokenValue)
		}
	}

	return toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg
}

// chemistryHitText tells the attacker that company chemistry made a hit.
const chemistryHitText = `<ansi fg="cyan">Fighting beside a trusted companion, you find an opening.</ansi>`

// calculateCombat resolves one attack round. darkPenalty is the attacker's
// to-hit penalty for poor visibility (see darknessPenalty), as a positive
// magnitude that is subtracted from the hit chance. chemistryBonus is the
// attacker's Phase 24 company chemistry, in points added to the hit chance
// of its weapon strikes (not its pet's).
func calculateCombat(sourceChar characters.Character, targetChar characters.Character, sourceType SourceTarget, targetType SourceTarget, darkPenalty int, chemistryBonus int) AttackResult {

	attackResult := AttackResult{}
	chemistryShown := false

	atkCount := combatAttackCount(sourceChar, targetChar)

	// Statmods can add a damage bonus plus the stat-driven damage bonus.
	statModDBonus := sourceChar.StatMod(`damage`) + damageBonus(sourceChar.Stats.Strength.ValueAdj, targetChar.Stats.Strength.ValueAdj)

	for i := 0; i < atkCount; i++ {

		mudlog.Debug(`calculateCombat`, `Atk`, fmt.Sprintf(`%d/%d`, i+1, atkCount), `Source`, fmt.Sprintf(`%s (%s)`, sourceChar.Name, sourceType), `Target`, fmt.Sprintf(`%s (%s)`, targetChar.Name, targetType))

		attackWeapons, weaponSlots := resolveAttackWeaponSlots(sourceChar)

		dualWieldLevel := sourceChar.GetSkillLevel(`dual-wield`)

		if len(attackWeapons) > 1 {
			bothClaws := sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws &&
				sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws
			maxWeapons := dualWieldActiveWeaponCount(dualWieldLevel, bothClaws)
			util.LogRoll(`Both Weapons`, maxWeapons, 2)

			for len(attackWeapons) > maxWeapons {
				rnd := util.Rand(len(attackWeapons))
				attackWeapons = append(attackWeapons[:rnd], attackWeapons[rnd+1:]...)
				weaponSlots = append(weaponSlots[:rnd], weaponSlots[rnd+1:]...)
			}
		}

		attackMessagePrefix := ``
		// If they are backstabbing it's a free crit
		if sourceChar.Aggro.Type == characters.BackStab {
			attackResult.Crit = true
			attackMessagePrefix = `<ansi fg="magenta-bold">*[BACKSTAB]*</ansi> `
			// Failover to the default attack
			sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
		}

		for wIdx, weapon := range attackWeapons {

			// Only the offhand weapon (index > 0) incurs a hit penalty for dual-wielding.
			// Hits adds its modifier, so a penalty is passed as a negative
			// value (the same convention as dualWieldHitPenalty).
			penalty := -darkPenalty
			if wIdx > 0 {
				penalty += dualWieldHitPenalty(dualWieldLevel)
			}

			// Set the default weapon info
			raceInfo := races.GetRace(sourceChar.GetRaceId())
			weaponName := raceInfo.UnarmedName
			weaponSubType := items.Generic

			// Get default racial dice rolls
			attacks, dCount, dSides, dBonus, critBuffs := sourceChar.GetDefaultDiceRoll()

			if weapon.ItemId > 0 {

				itemSpec := weapon.GetSpec()

				weaponName = weapon.DisplayName()

				weaponSubType = itemSpec.Subtype
				attacks, dCount, dSides, dBonus, critBuffs = weapon.GetDiceRoll()

				// If there is a bonus vs. a specific race, apply it
				dBonus += weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Race()))
			}

			// Apply damage stat modifier after weapon selection so it is never overwritten.
			dBonus += statModDBonus

			// zero means randomly selected, otherwise use the ItemId to consistently choose a message
			msgSeed := 0
			if configs.GetCombatConfig().ConsistentAttackMessages {
				msgSeed = weapon.ItemId
			}

			mudlog.Debug("DiceRolls", "attacks", attacks, "dCount", dCount, "dSides", dSides, "dBonus", dBonus, "critBuffs", critBuffs)

			// Individual weapons may get multiple attacks
			for j := 0; j < attacks; j++ {

				attackTargetDamage := 0
				attackTargetReduction := 0
				edgeBonus := 0
				isCrit := false

				hit, byChemistry := hitRoll(sourceChar.Stats.Speed.ValueAdj, targetChar.Stats.Speed.ValueAdj, penalty, chemistryBonus)
				if hit {
					// Check dodge before applying damage.
					if Dodges(targetChar.Stats.Perception.ValueAdj, sourceChar.Stats.Perception.ValueAdj) {
						attackResult.SendToSource(fmt.Sprintf(`<ansi fg="cyan">%s dodges your attack!</ansi>`, targetChar.Name))
						attackResult.SendToTarget(`<ansi fg="cyan">You dodge the attack!</ansi>`)
						continue
					}
					// Phase 24: say so, once a round, when only company
					// chemistry made the strike land.
					if byChemistry && !chemistryShown {
						chemistryShown = true
						attackResult.SendToSource(chemistryHitText)
					}
					attackResult.Hit = true
					attackTargetDamage = util.RollDice(dCount, dSides) + dBonus

					// Phase 23b: a sharpened weapon adds its edge to each
					// successful strike until its strikes are spent.
					if slot := weaponSlots[wIdx]; slot != `` && weapon.SharpStrikes-attackResult.EdgeSpent[slot] > 0 && weapon.SharpBonus > 0 {
						edgeBonus = weapon.SharpBonus
						attackTargetDamage += edgeBonus
						if attackResult.EdgeSpent == nil {
							attackResult.EdgeSpent = map[items.ItemType]int{}
						}
						attackResult.EdgeSpent[slot]++
					}

					// Backstab sets attackResult.Crit for the first hit only; subsequent
					// hits use a fresh per-attack roll so crits don't cascade.
					isCrit = attackResult.Crit || Crits(sourceChar, targetChar)
					attackResult.Crit = false // consume the backstab flag after one use
					if isCrit {
						attackResult.Crit = true // record that at least one crit occurred this round
						attackResult.BuffTarget = critBuffs
						attackTargetDamage += critDamageBonus(dCount, dSides, dBonus,
							sourceChar.Stats.Perception.ValueAdj, targetChar.Stats.Perception.ValueAdj)
					}
				}

				attackTargetDamage, attackTargetReduction = applyDefenseReduction(attackTargetDamage, targetChar.GetDefense())

				// An edge raises the strike's ceiling too, so a sharpened
				// top roll isn't described as a critical.
				pct := damagePercentOfMax(attackTargetDamage, dCount, dSides, dBonus+edgeBonus)
				msgs := items.GetAttackMessage(weaponSubType, pct)

				toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg := buildCombatMessages(
					&sourceChar, &targetChar, sourceType, targetType,
					weaponName, strconv.Itoa(attackTargetDamage), msgSeed,
					msgs.Together.ToAttacker, msgs.Together.ToDefender, msgs.Together.ToRoom, items.MessageOptions(nil),
					msgs.Separate.ToAttacker, msgs.Separate.ToDefender, msgs.Separate.ToAttackerRoom, msgs.Separate.ToDefenderRoom,
				)

				if isCrit {
					toAttackerMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toDefenderMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toAttackerRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					}
				}

				if len(attackMessagePrefix) > 0 {
					toAttackerMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerMsg))
					toDefenderMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderMsg))
					toAttackerRoomMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerRoomMsg))
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderRoomMsg))
					}
				}

				attackResult.SendToSource(string(toAttackerMsg))

				// Send to victim
				defenderMsg := string(toDefenderMsg)
				if attackTargetDamage > 0 && attackTargetReduction > 0 {
					defenderMsg += fmt.Sprintf(` <ansi fg="red">[you blocked %d]</ansi>`, attackTargetReduction)
				}

				attackResult.SendToTarget(defenderMsg)

				// Send to room
				attackResult.SendToSourceRoom(
					string(toAttackerRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).
						SetTokenValue(items.TokenTargetType, string(targetType))),
				)

				// Send to defender room if separate
				if len(string(toDefenderRoomMsg)) > 0 {
					attackResult.SendToTargetRoom(
						string(toDefenderRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).SetTokenValue(items.TokenTargetType, string(targetType))),
					)
				}

				attackResult.DamageToTarget += attackTargetDamage
				attackResult.DamageToTargetReduction += attackTargetReduction
			}

		}

		// Pet has a 20% chance per attack round to join the fight (once, regardless of weapon count)
		if sourceChar.Pet.Exists() && !sourceChar.Pet.IsMissing() {
			chance, petDmg := sourceChar.Pet.GetEffectiveDamage()
			if chance > 0 && util.RollDice(1, chance) <= chance {
				if sourceChar.RoomId == targetChar.RoomId {
					if petDmg.DiceRoll != `` {

						pAttacks, pDCount, pDSides, pDBonus, critBuffs := sourceChar.Pet.GetDiceRoll()
						combatMsgs := sourceChar.Pet.GetCombatMessages(string(targetType))

						for p := 0; p < pAttacks; p++ {

							if !Hits(0, targetChar.Stats.Speed.ValueAdj, 0) {
								targetDisplayName := fmt.Sprintf(`<ansi fg="%sname">%s</ansi>`, string(targetType), targetChar.Name)
								toAttackerMsg := combatMsgs.ApplyTokens(combatMsgs.Miss, sourceChar.Pet.DisplayName(), 0, targetDisplayName)
								attackResult.SendToSource(toAttackerMsg)
								continue
							}

							attackTargetDamage := util.RollDice(pDCount, pDSides) + pDBonus

							attackTargetDamage, _ = applyDefenseReduction(attackTargetDamage, targetChar.GetDefense())

							attackResult.DamageToTarget += attackTargetDamage

							targetDisplayName := fmt.Sprintf(`<ansi fg="%sname">%s</ansi>`, string(targetType), targetChar.Name)
							petDisplayName := sourceChar.Pet.DisplayName()

							toAttackerMsg := combatMsgs.ApplyTokens(combatMsgs.ToOwner, petDisplayName, attackTargetDamage, targetDisplayName)
							attackResult.SendToSource(toAttackerMsg)

							toDefenderMsg := combatMsgs.ApplyTokens(combatMsgs.ToTarget, petDisplayName, attackTargetDamage, targetDisplayName)
							attackResult.SendToTarget(toDefenderMsg)

							toAttackerRoomMsg := combatMsgs.ApplyTokens(combatMsgs.ToRoom, petDisplayName, attackTargetDamage, targetDisplayName)
							attackResult.SendToTargetRoom(toAttackerRoomMsg)

							// pets doing max damage are considered "crits" and will always apply any special critBuffs
							if len(critBuffs) > 0 && (attackTargetDamage == (pDCount*pDSides)+pDBonus) {
								attackResult.BuffTarget = critBuffs
							}
						}

					}
				}
			}
		}
	}

	return attackResult

}

// targetCarriesLight reports whether the target bears its own or a party
// light, which makes it visible to an attacker even in darkness.
func targetCarriesLight(target *characters.Character) bool {
	return target.HasBuffFlag(rooms.FlagLightSource) || target.HasBuffFlag(rooms.FlagPartyLight)
}

// darknessPenalty is the attacker's to-hit penalty in room, where
// visibilityOf reports what the attacker can see there. An unknown room
// applies no penalty.
func darknessPenalty(room *rooms.Room, target *characters.Character, visibilityOf func(*rooms.Room) int) int {
	if room == nil {
		return 0
	}
	return rooms.HitPenaltyForVisibility(visibilityOf(room), targetCarriesLight(target))
}
