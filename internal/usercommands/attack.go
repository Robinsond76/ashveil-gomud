package usercommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/battle"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Attack(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	attackPlayerId := 0
	attackMobInstanceId := 0

	// Ashveil Phase 29b2: in a battle, a bare "attack" takes a foe from it.
	if rest == `` {
		if b, inBattle := battle.Current(user.UserId); inBattle {
			for _, mId := range room.GetMobs() {
				if m := mobs.GetInstance(mId); m != nil && b.Has(mId) && m.Character.Health > 0 {
					attackMobInstanceId = mId
					break
				}
			}
		}
	}

	if rest == `` && attackMobInstanceId == 0 {
		partyInfo := parties.Get(user.UserId)

		// If no argument supplied, attack whoever is attacking the player currently.
		for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
			m := mobs.GetInstance(mId)
			if m.Character.Aggro == nil {
				continue
			}

			if m.Character.Aggro.UserId == user.UserId {
				attackMobInstanceId = m.InstanceId
				break
			}

			if partyInfo != nil {
				if partyInfo.IsMember(m.Character.Aggro.UserId) {
					attackMobInstanceId = m.InstanceId
					break
				}
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingPlayer) {
				u := users.GetByUserId(uId)
				if u == nil || u.Character.Aggro == nil {
					continue
				}

				if u.Character.Aggro.UserId == user.UserId {
					attackPlayerId = u.UserId
					break
				}

				if partyInfo != nil {
					if partyInfo.IsMember(u.Character.Aggro.UserId) {
						attackPlayerId = u.UserId
						break
					}
				}
			}
		}

		// Finally, if still no targets, check if any party members are aggroed and just glom onto that
		if attackMobInstanceId == 0 && attackPlayerId == 0 {
			if partyInfo != nil {
				for uId := range partyInfo.GetMembers() {
					if partyUser := users.GetByUserId(uId); partyUser != nil {
						if partyUser.Character.Aggro == nil {
							continue
						}

						if partyUser.Character.Aggro.MobInstanceId > 0 {
							attackMobInstanceId = partyUser.Character.Aggro.MobInstanceId
							break
						}

						if partyUser.Character.Aggro.UserId > 0 {
							attackPlayerId = partyUser.Character.Aggro.UserId
							break
						}

					}
				}
			}
		}

	} else if rest[0] == '*' { // choose a target at random. Friend or foe.

		if rest == `*` { // * ANYONE

			allMobs := room.GetMobs()
			allPlayers := []int{}
			for _, userId := range room.GetPlayers() {
				if userId == user.UserId {
					continue
				}
				allPlayers = append(allPlayers, userId)
			}

			totalTargets := len(allMobs) + len(allPlayers)
			if totalTargets == 0 {
				return true, nil
			}

			randomSelection := util.Rand(totalTargets)

			if randomSelection < len(allMobs) {
				attackMobInstanceId = allMobs[randomSelection]
			} else {
				randomSelection -= len(allMobs)
				attackPlayerId = allPlayers[randomSelection]
			}

		} else if rest == `*mob` { // *mob ANY MOB

			if allMobs := room.GetMobs(); len(allMobs) > 0 {
				attackMobInstanceId = allMobs[util.Rand(len(allMobs))]
			}

		} else { // *user etc. ANY PLAYER

			allPlayers := []int{}
			for _, userId := range room.GetPlayers() {
				if userId == user.UserId {
					continue
				}
				allPlayers = append(allPlayers, userId)
			}

			if len(allPlayers) > 0 {
				attackPlayerId = allPlayers[util.Rand(len(allPlayers))]
			}

		}

	} else {
		attackPlayerId, attackMobInstanceId = room.FindByName(rest)
	}

	if attackPlayerId == user.UserId { // Can't attack self!
		attackPlayerId = 0
	}

	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText("You attack the darkness!")
		return true, nil
	}

	isSneaking := user.Character.HasBuffFlag("hidden")

	/*
		combatAddlWaitRounds := user.Character.Equipment.Weapon.GetSpec().WaitRounds + user.Character.Equipment.Weapon.GetSpec().WaitRounds
		attkType := characters.DefaultAttack
		if user.Character.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			attkType = characters.Shooting
		}
	*/

	if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {
			if m.Character.IsCharmed(user.UserId) {
				user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is your friend!`, m.Character.Name))
				return true, nil
			}

			// Ashveil Phase 29b2: one battle at a time. A group waiting its
			// turn can't be attacked until this fight is over.
			if b, inBattle := battle.Current(user.UserId); inBattle && !b.Has(m.InstanceId) {
				user.SendText(fmt.Sprintf(`You're fighting %s. Finish that fight first.`, battleFoeName(b, room)))
				return true, nil
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {

								partyUser.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId)) // # denotes a specific mob instanceId

							}
						}

					}
				}
			}

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})

			// Phase 29c: no "prepares to fight"; the fight's opener speaks
			// for the room.
			user.SendText(goForText(user.Character, util.Article(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, m.Character.Name))))

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						m.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId)) // # denotes a specific mob instanceId

					}
				}
			}

		}

	} else if attackPlayerId > 0 {

		if p := users.GetByUserId(attackPlayerId); p != nil {

			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(pvpErr.Error())
				return true, nil
			}

			if partyInfo := parties.Get(user.UserId); partyInfo != nil {
				if partyInfo.IsMember(attackPlayerId) {
					user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
					return true, nil
				}
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {
								partyUser.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // # denotes a specific mob instanceId
							}
						}
					}
				}
			}

			user.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})

			user.SendText(goForText(user.Character, fmt.Sprintf(`<ansi fg="username">%s</ansi>`, p.Character.Name)))

			if !isSneaking {
				p.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> comes for you.`, user.Character.Name))
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						m.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // @ denotes a specific user id

					}
				}
			}

		}

	}

	return true, nil
}

// battleFoeName names a player's battle for the refusal: its first living
// foe in the room, and "and the others" when it has more.
func battleFoeName(b battle.Battle, room *rooms.Room) string {
	var names []string
	for _, mId := range room.GetMobs() {
		if m := mobs.GetInstance(mId); m != nil && b.Has(mId) && m.Character.Health > 0 {
			names = append(names, m.Character.Name)
		}
	}
	switch len(names) {
	case 0:
		return `another foe`
	case 1:
		return fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, names[0])
	}
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi> and the others`, names[0])
}

// goForText is the attacker's own line as a fight begins (Phase 29c):
// "You draw your broadsword and go for the bandit captain."
func goForText(c *characters.Character, target string) string {
	if c.Equipment.Weapon.ItemId > 0 {
		return fmt.Sprintf(`You draw your <ansi fg="item">%s</ansi> and go for %s.`, c.Equipment.Weapon.DisplayName(), target)
	}
	return fmt.Sprintf(`You go for %s.`, target)
}
