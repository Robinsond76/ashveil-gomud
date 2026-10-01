package usercommands

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/company"
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
	alreadyFighting := user.Character.Aggro != nil // Phase 29c: "turn toward", not "draw"

	// Ashveil Phase 32c (the owner's rule 5): once a battle has started it
	// plays out on its own; nothing typed changes it. A player who used
	// `break` may rejoin their battle with a bare `attack`.
	if b, inBattle := battle.Current(user.UserId); inBattle || fightingMob(user) {
		if !inBattle || rest != `` || user.Character.Aggro != nil || !engagement.StoodDown(user.UserId) {
			user.SendText(BattleUnderWay)
			return true, nil
		}
		if g, ok := battleGroup(b, room); ok {
			attackMobInstanceId, _ = enemyparty.Aim(g, enemyparty.PlayerAttacker(user))
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
				user.SendText(util.CapitalizeFirst(fmt.Sprintf(`%s fights with <ansi fg="mobname">%s</ansi>. Type <ansi fg="command">attack %s</ansi>.`,
					util.Article(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, m.Character.Name)), g.Name, GroupKeyword(room, g))))
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
			// the first member struck is chosen by the player's strategy
			// (32d: enemyparty.Aim), as is each companion's.
			foeName := m.Character.Name
			group, grouped := enemyparty.GroupOf(room, m.InstanceId)
			if grouped {
				if !group.Solo() {
					foeName = theGroup(group.Name)
				}
				if aim, ok := enemyparty.Aim(group, enemyparty.PlayerAttacker(user)); ok {
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

								events.AddToQueue(events.Input{UserId: id, InputText: fmt.Sprintf("attack #%d", attackMobInstanceId), ReadyTurn: util.GetTurnCount(), PartyAttack: &events.PartyAttackOrder{LeaderUserId: user.UserId, OriginRoomId: room.RoomId, ConsentToken: party.AutoAttackToken(id), TargetMobInstanceId: attackMobInstanceId}})

							}
						}

					}
				}
			}

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)
			// Back in the fight (a bare attack after break): the game turns
			// them again (32c review: a lone player stayed stood down).
			engagement.Resume(user.UserId)

			events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})

			// Phase 29c: no "prepares to fight"; the fight's opener speaks
			// for the room. Phase 32c: the foe is the whole group.
			user.SendText(goForText(user.Character, util.Article(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, foeName)), alreadyFighting))

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						// Phase 32d: a companion starts on its own strategy's choice.
						aim := attackMobInstanceId
						if leaderId, key, ok := company.LeaderAndKeyForInstance(instId); ok && leaderId == user.UserId && grouped {
							if id, ok := enemyparty.Aim(group, enemyparty.CompanionAttacker(user.UserId, key, m, attackMobInstanceId)); ok {
								aim = id
							}
						}
						m.Command(fmt.Sprintf(`attack #%d`, aim)) // # denotes a specific mob instanceId

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

			user.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			events.AddToQueue(events.AggroChanged{UserId: user.UserId, RoomId: user.Character.RoomId})

			user.SendText(goForText(user.Character, fmt.Sprintf(`<ansi fg="username">%s</ansi>`, p.Character.Name), alreadyFighting))

			if !isSneaking {
				p.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> comes for you.`, user.Character.Name))
				// No battle opens between players, so the room is told here.
				room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> goes for <ansi fg="username">%s</ansi>.`, user.Character.Name, p.Character.Name),
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

// BattleOnlyFlee is the answer to trying to step out of a battle (Ashveil
// Phase 32d): only flee takes a player out.
const BattleOnlyFlee = BattleUnderWay + ` Use <ansi fg="command">retreat</ansi> or <ansi fg="command">flee</ansi> to leave it.`

// InBattle reports whether the player is in a battle (Ashveil Phase 32c
// and 32d): they have one, or are aimed at a mob (a battle about to
// begin). Once one has started, nothing typed changes it; only flee takes
// them out.
func InBattle(user *users.UserRecord) bool { return actionpolicy.InBattle(user) }

// fightingMob reports whether the player is already aimed at a mob, a
// battle about to begin.
func fightingMob(user *users.UserRecord) bool {
	a := user.Character.Aggro
	return a != nil && a.MobInstanceId > 0 && a.ExitName == `` // a shot into the next room is no battle (32d review)
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
	return enemyparty.Keyword(room, g)
}

// NotAnOpener refuses a move that isn't how a fight starts (Ashveil Phase
// 32c, the owner's rule 5): "A backstab doesn't start a fight. Type attack
// ruffians to fight a band of ruffians."
func NotAnOpener(room *rooms.Room, mobInstanceId int, what string) string {
	if room != nil {
		if g, ok := enemyparty.GroupOf(room, mobInstanceId); ok {
			return fmt.Sprintf(`%s doesn't start a fight. Type <ansi fg="command">attack %s</ansi> to fight %s.`, what, GroupKeyword(room, g), util.Article(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, g.Name)))
		}
	}
	return fmt.Sprintf(`%s doesn't start a fight. Start one with <ansi fg="command">attack</ansi>.`, what)
}

// goForText is the attacker's own line as a fight begins (Phase 29c):
// "You draw your broadsword and go for the bandit captain.", or, already
// fighting, "You turn toward the bandit captain."
func goForText(c *characters.Character, target string, alreadyFighting bool) string {
	if alreadyFighting {
		return fmt.Sprintf(`You turn toward %s.`, target)
	}
	if c.Equipment.Weapon.ItemId > 0 {
		return fmt.Sprintf(`You draw your <ansi fg="item">%s</ansi> and go for %s.`, c.Equipment.Weapon.DisplayName(), target)
	}
	return fmt.Sprintf(`You go for %s.`, target)
}

// theGroup turns a group's name into the one being gone for: "a band of
// ruffians" reads "the band of ruffians" (Phase 32c with 29c's voice).
func theGroup(name string) string {
	for _, a := range []string{`a `, `an `} {
		if strings.HasPrefix(name, a) {
			return `the ` + name[len(a):]
		}
	}
	return name
}
