package usercommands

import (
	"errors"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/effecttargets"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flasks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Cast Skill
Level 1 - You can cast spells
Level 2 - Become proficient in a spell at 125% rate
Level 3 - Become proficient in a spell at 175% rate
Level 4 - Become proficient in a spell at 250% rate
*/
// OtherBattlePatient refuses help for someone fighting another's battle.
const OtherBattlePatient = `That one is caught up in another battle; you can't help either side.`

func Cast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(`cast`)

	if skillLevel == 0 {
		user.SendText("You don't know how to cast spells yet.")
		return true, errors.New(`you don't know how to cast spells yet`)
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		user.SendText("Cast What? At Whom?")
		return true, nil
	}

	spellName := args[0]
	args = args[1:]

	// Ashveil Phase 54: `cast sigil of [kind]` lays a sigil before a fight.
	if spellName == `sigil` {
		castSigil(strings.Join(args, ` `), user, room)
		return true, nil
	}

	if len(args) > 1 {
		if args[0] == `on` {
			args = args[1:]
		}
	}
	spellArg := strings.Join(args, ` `)

	spellInfo := spells.GetSpell(spellName)

	if spellInfo == nil || !user.Character.KnowsSpell(spellName) {
		user.SendText(fmt.Sprintf(`You don't know a spell called <ansi fg="spellname">%s</ansi>.`, spellName))
		return true, nil
	}

	// Ashveil Phase 32c (the owner's rule 5): in a battle, characters cast
	// on their own; nothing is cast by hand.
	if _, inBattle := battle.Current(user.UserId); inBattle || fightingMob(user) {
		user.SendText(BattleUnderWay)
		return true, nil
	}

	if user.Character.Mana < user.Character.SpellCost(spellInfo) {
		user.SendText(fmt.Sprintf(`You don't have enough mana to cast <ansi fg="spellname">%s</ansi>.`, spellName))
		return true, nil
	}

	// Ashveil Phase 39g: an Alchemist's spells cost a flask, not mana.
	if spellInfo.Flask > 0 && flasks.Remaining(user.Character) < spellInfo.Flask {
		user.SendText(fmt.Sprintf(`Your satchel is empty, so you can't throw <ansi fg="spellname">%s</ansi>. Brew more flasks with <ansi fg="command">brew</ansi> (<ansi fg="command">help brew</ansi>).`, spellName))
		return true, nil
	}

	targetPlayerId := 0
	targetMobInstanceId := 0

	if spellArg != `` {
		targetPlayerId, targetMobInstanceId = room.FindByName(spellArg)
	}

	spellAggro := characters.SpellAggroInfo{
		SpellId:              spellInfo.SpellId,
		SpellRest:            ``,
		TargetUserIds:        make([]int, 0),
		TargetMobInstanceIds: make([]int, 0),
	}

	if spellInfo.Type == spells.Neutral {

		spellAggro.SpellRest = spellArg

	} else if spellInfo.Type == spells.HelpSingle {

		if spellArg == `` {

			// No target specified? Default to self
			spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, user.UserId)

		} else {

			if targetPlayerId > 0 {
				spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, targetPlayerId)
			} else if targetMobInstanceId > 0 {
				spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, targetMobInstanceId)
			}

		}

	} else if spellInfo.Type == spells.HarmSingle {

		if targetPlayerId > 0 {
			if u := users.GetByUserId(targetPlayerId); u != nil {
				if pvpErr := room.CanPvp(user, u); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}
		}

		if spellArg == `` {

			if user.Character.Aggro != nil {
				// No target specified? Default to aggro target
				if user.Character.Aggro.UserId > 0 {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, user.Character.Aggro.UserId)
				} else if user.Character.Aggro.MobInstanceId > 0 {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, user.Character.Aggro.MobInstanceId)
				}
			} else {

				fightingMobs := room.GetMobs(rooms.FindFightingPlayer)
				if len(fightingMobs) > 0 {

					for _, mobInstId := range fightingMobs {

						if mob := mobs.GetInstance(mobInstId); mob != nil {
							if mob.Character.IsAggro(user.UserId, 0) {
								spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
								break
							}
						}

					}

				}

				// If no mobs found, try finding an aggro player
				if len(spellAggro.TargetMobInstanceIds) < 1 {
					fightingPlayers := room.GetPlayers(rooms.FindFightingPlayer)
					if len(fightingPlayers) > 0 {

						for _, fUserId := range fightingPlayers {

							if u := users.GetByUserId(fUserId); u != nil {
								if u.Character.IsAggro(user.UserId, 0) {
									spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, fUserId)
									break
								}
							}

						}

					}
				}

			}

		} else {

			if targetPlayerId > 0 {
				spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, targetPlayerId)
			} else if targetMobInstanceId > 0 {
				spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, targetMobInstanceId)
			}

		}

	} else if spellInfo.Type == spells.HelpMulti {
		// The shared resolver below selects the present company.

	} else if spellInfo.Type == spells.HarmMulti {

		// Targets all mobs aggro towards player
		// Targets all players aggro towards player and their parties

		// If not currently aggro, only targets all mobs in the room

		if targetMobInstanceId > 0 {

			// target all mobs
			for _, mobInstId := range room.GetMobs() {
				if m := mobs.GetInstance(mobInstId); m != nil {
					spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
				}
			}

		} else if targetPlayerId > 0 {

			// make sure they can Pvp the player being targetted
			if u := users.GetByUserId(targetPlayerId); u != nil {
				if pvpErr := room.CanPvp(user, u); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}

			for _, uId := range room.GetPlayers() {
				if uId == user.UserId {
					continue
				}
				if u := users.GetByUserId(uId); u != nil {
					if pvpErr := room.CanPvp(user, u); pvpErr != nil {
						spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
					}
				}
			}

		} else {

			fightingMobs := room.GetMobs(rooms.FindFightingPlayer)
			for _, mobInstId := range fightingMobs {
				if m := mobs.GetInstance(mobInstId); m != nil {
					if m.Character.IsAggro(user.UserId, 0) || m.HatesRace(user.Character.Race()) {
						spellAggro.TargetMobInstanceIds = append(spellAggro.TargetMobInstanceIds, mobInstId)
					}
				}
			}

			fightingPlayers := room.GetPlayers(rooms.FindFightingPlayer)
			for _, uId := range fightingPlayers {
				if u := users.GetByUserId(uId); u != nil {
					if u.Character.IsAggro(user.UserId, 0) {
						spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
					}
				}
			}

		}

		if len(spellAggro.TargetUserIds) < 1 && len(spellAggro.TargetMobInstanceIds) < 1 {
			// No targets found, default to all mobs in the room
			spellAggro.TargetMobInstanceIds = room.GetMobs(rooms.FindFightingPlayer)
		}

	} else if spellInfo.Type == spells.HelpArea {

		spellAggro.TargetUserIds = room.GetPlayers()
		spellAggro.TargetMobInstanceIds = room.GetMobs()

	} else if spellInfo.Type == spells.HarmArea {

		// make sure they can Pvp the player being hit
		for _, uId := range room.GetPlayers() {
			if u := users.GetByUserId(uId); u != nil {
				if err := room.CanPvp(user, u); err == nil {
					spellAggro.TargetUserIds = append(spellAggro.TargetUserIds, uId)
				}

			}
		}

		spellAggro.TargetMobInstanceIds = room.GetMobs()

	}

	// Ashveil Phase 32c (the owner's rule 5): a harmful spell doesn't start
	// a fight with an enemy; fights start with attack and a group's name.
	if spellInfo.Type == spells.HarmSingle || spellInfo.Type == spells.HarmMulti || spellInfo.Type == spells.HarmArea {
		for _, id := range spellAggro.TargetMobInstanceIds {
			if m := mobs.GetInstance(id); m != nil && !m.Character.IsCharmed() {
				user.SendText(NotAnOpener(room, id, `A harmful spell`))
				return true, nil
			}
		}
	}

	// Phase 33b review: a bystander helps neither side of someone else's
	// battle (help friendly-effects).
	// Group and area help is narrowed to the caster's company by Resolve;
	// only an explicitly named patient is refused here.
	if effecttargets.Helpful(spellInfo) && spellInfo.FriendlyScope() == spells.ScopeMember {
		for _, id := range spellAggro.TargetUserIds {
			if effecttargets.OtherBattle(user.UserId, 0, id, 0) {
				user.SendText(OtherBattlePatient)
				return true, nil
			}
		}
		for _, id := range spellAggro.TargetMobInstanceIds {
			if effecttargets.OtherBattle(user.UserId, 0, 0, id) {
				user.SendText(OtherBattlePatient)
				return true, nil
			}
		}
	}

	spellAggro = effecttargets.Resolve(user.UserId, 0, spellAggro)

	if len(spellAggro.TargetUserIds) > 0 || len(spellAggro.TargetMobInstanceIds) > 0 || len(spellAggro.SpellRest) > 0 {

		continueCasting := true
		if handled, err := scripting.TrySpellScriptEvent(`onCast`, user.UserId, 0, spellAggro); err == nil {
			continueCasting = handled
		}

		if continueCasting {

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: `cast`, Details: spellInfo.SpellId})

			user.Character.Mana -= user.Character.SpellCost(spellInfo)
			user.Character.FlasksSpent += spellInfo.Flask // Phase 39g: a thrown flask is used up
			events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
			user.Character.SetCast(spellInfo.WaitRounds, spellAggro)
		}

	} else {

		user.SendText(`Couldn't find a target for the spell.`)

	}

	return true, nil
}
