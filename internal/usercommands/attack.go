package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/engagement"

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

	// Ashveil Phase 32c (the owner's rule 5): once a battle has started it
	// plays out on its own; nothing typed changes it. A player who used
	// `break` may rejoin their battle with a bare `attack`.
	if b, inBattle := battle.Current(user.UserId); inBattle || fightingMob(user) {
		if !inBattle || rest != `` || user.Character.Aggro != nil || !engagement.StoodDown(user.UserId) {
			user.SendText(BattleUnderWay)
			return true, nil
		}
		if g, ok := battleGroup(b, room); ok {
			attackMobInstanceId, _ = enemyparty.FirstAim(g, user.UserId, user.Character)
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

	} else if rest == `` {
		// rejoining a battle after a break (above)
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

	} else if rest[0] == '#' {
		// An exact mob (how companions and party members join a fight):
		// its whole group, below.
		_, attackMobInstanceId = room.FindByName(rest)
	} else if g, ok := enemyparty.FindGroup(room, rest); ok {
		// Ashveil Phase 32c: a fight is started by naming a group.
		attackMobInstanceId = g.Party.Members[0]
	} else {
		attackPlayerId, attackMobInstanceId = room.FindByName(rest)
		if attackMobInstanceId > 0 {
			// A member of a group is not a way to start a fight (the
			// owner's rule 3): say what to type instead.
			if g, ok := enemyparty.GroupOf(room, attackMobInstanceId); ok && !g.Solo() {
				m := mobs.GetInstance(attackMobInstanceId)
				user.SendText(fmt.Sprintf(`The <ansi fg="mobname">%s</ansi> fights with <ansi fg="mobname">%s</ansi>. Type <ansi fg="command">attack %s</ansi>.`,
					m.Character.Name, g.Name, GroupKeyword(room, g)))
				return true, nil
			}
		}
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

			// Ashveil Phase 32c: the fight is with the mob's whole group, and
			// the first member struck is chosen as the player's strategy
			// would (enemyparty.FirstAim).
			foeName := m.Character.Name
			if g, ok := enemyparty.GroupOf(room, m.InstanceId); ok {
				if !g.Solo() {
					foeName = g.Name
				}
				if aim, ok := enemyparty.FirstAim(g, user.UserId, user.Character); ok {
					attackMobInstanceId = aim
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

								partyUser.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId)) // # denotes a specific mob instanceId

							}
						}

					}
				}
			}

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})

			user.SendText(fmt.Sprintf(`You prepare to fight <ansi fg="mobname">%s</ansi>!`, foeName))

			if !isSneaking {
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>.`, user.Character.Name, foeName),
					user.UserId,
				)
			}

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

			user.SendText(fmt.Sprintf(`You prepare to fight <ansi fg="mobname">%s</ansi>!`, p.Character.Name))

			if !isSneaking {

				p.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight you!`, user.Character.Name),
				)

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="username">%s</ansi>.`, user.Character.Name, p.Character.Name),
					user.UserId, attackPlayerId)
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

// BattleUnderWay is the answer to anything typed at a battle once it has
// started (Ashveil Phase 32c, the owner's rule 5).
const BattleUnderWay = `The battle is under way: it plays out as you set it up.`

// fightingMob reports whether the player is already aimed at a mob, a
// battle about to begin.
func fightingMob(user *users.UserRecord) bool {
	a := user.Character.Aggro
	return a != nil && a.MobInstanceId > 0
}

// battleGroup is the player's battle's group in the room.
func battleGroup(b battle.Battle, room *rooms.Room) (enemyparty.Group, bool) {
	for _, g := range enemyparty.Groups(room) {
		for _, id := range g.Party.Members {
			if b.Has(id) {
				return g, true
			}
		}
	}
	return enemyparty.Group{}, false
}

// GroupKeyword is what a player types to name g: its keyword, with "#2"
// for the second group of that name in the room.
func GroupKeyword(room *rooms.Room, g enemyparty.Group) string {
	n := 0
	for _, other := range enemyparty.Groups(room) {
		if !other.Solo() && other.Naming.Name == g.Naming.Name {
			n++
		}
		if other.Party.ID == g.Party.ID {
			break
		}
	}
	if n > 1 {
		return fmt.Sprintf(`%s#%d`, g.Naming.Keyword, n)
	}
	return g.Naming.Keyword
}

// NotAnOpener refuses a move that isn't how a fight starts (Ashveil Phase
// 32c, the owner's rule 5): "A backstab doesn't start a fight. Type attack
// ruffians to fight a band of ruffians."
func NotAnOpener(room *rooms.Room, mobInstanceId int, what string) string {
	if room != nil {
		if g, ok := enemyparty.GroupOf(room, mobInstanceId); ok {
			return fmt.Sprintf(`%s doesn't start a fight. Type <ansi fg="command">attack %s</ansi> to fight <ansi fg="mobname">%s</ansi>.`, what, GroupKeyword(room, g), g.Name)
		}
	}
	return fmt.Sprintf(`%s doesn't start a fight. Start one with <ansi fg="command">attack</ansi>.`, what)
}
