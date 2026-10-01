package usercommands

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Party(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	parties.ClearError()
	handled, err := partyCommand(rest, user, room, flags)
	if storageErr := parties.LastError(); storageErr != nil {
		user.SendText(storageErr.Error())
		return true, storageErr
	}
	return handled, err
}

func partyCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	partyCommand := `list`
	if len(args) > 0 {
		partyCommand = strings.ToLower(args[0])
		rest, _ = strings.CutPrefix(rest, args[0])
		rest = strings.TrimSpace(rest)
	}

	currentParty := parties.Get(user.UserId)

	if partyCommand == `create` || partyCommand == `new` || partyCommand == `start` {

		// check if they are already part of a party
		if currentParty != nil {
			if currentParty.Invited(user.UserId) {
				user.SendText(`You already have a pending party invite. Try <ansi fg="command">party accept/decline</ansi> first`)
			} else if currentParty.IsLeader(user.UserId) {
				user.SendText(`You already own a party Type <ansi fg="command">party list</ansi> for more info.`)
			} else {
				user.SendText(`You are already party of a party.`)
			}
			return true, nil
		}

		if currentParty = parties.New(user.UserId); currentParty != nil {
			user.EventLog.Add(`party`, `Started a new party`)
			user.SendText(`You started a new party!`)

			//
			// User started a new party
			//
			events.AddToQueue(events.PartyUpdated{
				Action:  `created`,
				UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
			})

		} else {
			user.SendText(`Something went wrong.`)
		}

		return true, nil
	}
	// Done with create

	//
	// Everything after this point requires a party or an invitation to a party
	//

	if partyCommand == `invite` {

		if rest == `` {
			user.SendText(`Invite who?`)
			return true, nil
		}

		// Not in a party? Create one.
		if currentParty == nil {
			currentParty = parties.New(user.UserId)
			if currentParty == nil {
				return true, parties.LastError()
			}
		}

		if !currentParty.IsLeader(user.UserId) {
			user.SendText(`You are not the leader of your party.`)
			return true, nil
		}

		partyCfg := configs.GetGamePlayConfig().Party

		if maxCount := int(partyCfg.MaxPlayerCount); maxCount > 0 {
			if len(currentParty.UserIds)+len(currentParty.InviteUserIds) >= maxCount {
				user.SendText(fmt.Sprintf(`Your party is full (%d/%d).`, len(currentParty.UserIds), maxCount))
				return true, nil
			}
		}

		var invitePlayerId int
		if bool(partyCfg.SameRoomOnly) {
			invitePlayerId, _ = room.FindByName(rest)
		} else {
			if u := users.GetByCharacterName(rest); u != nil {
				invitePlayerId = u.UserId
			}
		}

		if invitePlayerId == 0 {
			user.SendText(fmt.Sprintf(`%s not found.`, rest))
			return true, nil
		}

		if invitedParty := parties.Get(invitePlayerId); invitedParty != nil {
			user.SendText(`That player is already in a party.`)
			return true, nil
		}

		invitedUser := users.GetByUserId(invitePlayerId)

		if invitedUser != nil && currentParty.InvitePlayer(invitePlayerId) {
			user.SendText(fmt.Sprintf(`You invited <ansi fg="username">%s</ansi> to your party.`, invitedUser.Character.Name))
			invitedUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> invited you to their party. Type <ansi fg="command">party accept</ansi> or <ansi fg="command">party decline</ansi> to respond.`, user.Character.Name))
		} else {
			user.SendText(`Something went wrong.`)
		}

		//
		// A new user was invited to the party
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `invited`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})

		return true, nil
	}

	//
	// what follows doesn't mamke sense unless they are in a party
	//

	if currentParty == nil {
		user.SendText(`You are not attached to a party.`)
		return true, nil
	}

	if partyCommand == `accept` || partyCommand == `join` {

		if !currentParty.Invited(user.UserId) {
			user.SendText(`You haven't accepted an invitation to the party.`)
			return true, nil
		}

		if bool(configs.GetGamePlayConfig().Party.SameRoomOnly) {
			leader := users.GetByUserId(currentParty.LeaderUserId)
			if leader == nil || leader.Character.RoomId != user.Character.RoomId {
				user.SendText(`You must be in the same room as the party leader to join.`)
				return true, nil
			}
		}

		if currentParty.AcceptInvite(user.UserId) {

			user.EventLog.Add(`party`, `Joined a party`)
			user.SendText(`You joined the party!`)
			for _, uid := range currentParty.UserIds {
				if uid == user.UserId {
					continue
				}
				if u := users.GetByUserId(uid); u != nil {
					u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> joined the party!`, user.Character.Name))
				}
			}

			//
			// A user joined the party
			//
			events.AddToQueue(events.PartyUpdated{
				Action:  `joined`,
				UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
			})

		} else {
			user.SendText(`Something went wrong.`)
		}
		return true, nil
	}

	if partyCommand == `decline` {

		//
		// User declined invitation
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `declined`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})

		if currentParty.DeclineInvite(user.UserId) {

			if u := users.GetByUserId(currentParty.LeaderUserId); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> declined the invitation.`, user.Character.Name))
			}
			user.SendText(`You decline the invitation.`)

		} else {
			user.SendText(`Something went wrong.`)
		}
		return true, nil
	}

	if partyCommand == `list` {

		//headers := []string{"Name", "Status", "Lvl", "Health", "%", "Location", "Position"}
		headers := []string{"Name", "Status", "Lvl", "Health", "Location", "Position"}
		formatting := [][]string{}

		rows := [][]string{}

		if currentParty != nil {
			isInvited := currentParty.Invited(user.UserId)
			leaderId := currentParty.LeaderUserId

			charmedMobInstanceIds := []int{}

			for _, uid := range currentParty.UserIds {
				uStatus := "In Party"
				if leaderId == uid {
					uStatus = "Leader"
				}

				u := users.GetByUserId(uid)
				if u == nil {
					name := fmt.Sprintf("Player #%d", uid)
					if saved, err := users.LoadUserFile(uid); err == nil && saved != nil {
						name = saved.Character.Name
					}
					rows = append(rows, []string{name, uStatus + " (offline)", "-", "-", "-", "-"})
					formatting = append(formatting, []string{"%s", "%s", "%s", "%s", "%s", "%s"})
					continue
				}
				uLevel := fmt.Sprintf(`%d`, u.Character.Level)
				uRoom := rooms.LoadRoom(u.Character.RoomId)
				uLoc := `-`
				if uRoom != nil {
					uLoc = uRoom.Title
				}
				uHealthPct := int(math.Floor((float64(u.Character.Health) / float64(u.Character.HealthMax.Value)) * 100))
				uHealthPctStr := fmt.Sprintf(`%d%%`, uHealthPct)
				rank := currentParty.GetRank(u.UserId)
				healthClass := util.HealthClass(u.Character.Health, u.Character.HealthMax.Value)

				if isInvited {
					uLevel = `-`
					//uHealth = `-`
					uLoc = `-`
					uHealthPctStr = `-`
					rank = `-`
					healthClass = `black-bold`
				}

				rows = append(rows, []string{
					u.Character.Name,
					uStatus,
					uLevel,
					//uHealth,
					uHealthPctStr,
					uLoc,
					rank,
				})

				rowFormat := []string{`<ansi fg="username">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`,
					`<ansi fg="yellow">%s</ansi>`,
					//`<ansi fg="cyan-bold">%s</ansi>`,
					`<ansi fg="` + healthClass + `">%s</ansi>`,
					`<ansi fg="magenta-bold">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`}

				formatting = append(formatting, rowFormat)

				charmedMobInstanceIds = append(charmedMobInstanceIds, u.Character.GetCharmIds()...)
			}

			for _, mobInstanceId := range charmedMobInstanceIds {
				m := mobs.GetInstance(mobInstanceId)
				if m == nil {
					continue
				}
				mRoom := rooms.LoadRoom(m.Character.RoomId)
				mLoc := `-`
				if mRoom != nil {
					mLoc = mRoom.Title
				}
				mHealthPct := int(math.Floor((float64(m.Character.Health) / float64(m.Character.HealthMax.Value)) * 100))
				// A company member is the leader's own, not a ♥friend (Phase 32a).
				mStatus := `♥friend`
				if m.Character.IsCompanion() {
					mStatus = `Company`
				}
				rows = append(rows, []string{
					m.Character.Name,
					mStatus,
					fmt.Sprintf(`%d`, m.Character.Level),
					//fmt.Sprintf(`%d/%d`, m.Character.Health, m.Character.HealthMax.Value),
					fmt.Sprintf(`%d%%`, mHealthPct),
					mLoc,
					`-`,
				})

				rowFormat := []string{`<ansi fg="username">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`,
					`<ansi fg="yellow">%s</ansi>`,
					//`<ansi fg="cyan-bold">%s</ansi>`,
					`<ansi fg="` + util.HealthClass(m.Character.Health, m.Character.HealthMax.Value) + `">%s</ansi>`,
					`<ansi fg="magenta-bold">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`}

				formatting = append(formatting, rowFormat)
			}

			for _, uid := range currentParty.InviteUserIds {
				u := users.GetByUserId(uid)
				if u == nil {
					continue
				}
				rows = append(rows, []string{
					u.Character.Name,
					`Invited`,
					`-`,
					`-`,
					//`-`,
					`-`,
					`-`,
				})

				rowFormat := []string{`<ansi fg="username">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`,
					`<ansi fg="yellow">%s</ansi>`,
					//`<ansi fg="cyan-bold">%s</ansi>`,
					`<ansi fg="black-bold">%s</ansi>`,
					`<ansi fg="magenta-bold">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`}

				formatting = append(formatting, rowFormat)

			}

			partyTableData := templates.GetTable(`Party Members`, headers, rows, formatting...)
			partyTxt, _ := templates.Process("tables/generic", partyTableData, user.UserId)
			user.SendText(partyTxt)

			if !isInvited {
				user.SendText(fmt.Sprintf(`Your consent: follow=%t, support=%t, autoattack=%t. Each owner commands their own company.`, currentParty.Follows(user.UserId), currentParty.Supports(user.UserId), slices.Contains(currentParty.GetAutoAttackUserIds(), user.UserId)))
			}
			if isInvited {
				user.SendText(`Type <ansi fg="command">party accept/decline</ansi> to finalize your party membership.`)
			}
		}
	}

	if currentParty.Invited(user.UserId) {
		user.SendText(`You haven't accepted an invitation to the party.`)
		return true, nil
	}

	//
	// Everything after this point you must be in a party
	//
	if partyCommand == `follow` || partyCommand == `support` {
		if rest != `on` && rest != `off` {
			user.SendText(fmt.Sprintf(`Usage: party %s [on/off]`, partyCommand))
			return true, nil
		}
		if partyCommand == `follow` {
			currentParty.SetFollow(user.UserId, rest == `on`)
		} else {
			currentParty.SetSupport(user.UserId, rest == `on`)
		}
		user.SendText(fmt.Sprintf(`Party %s is %s for your company.`, partyCommand, rest))
		events.AddToQueue(events.PartyUpdated{Action: `behavior`, UserIds: currentParty.GetMembers()})
		return true, nil
	}

	if partyCommand == `autoattack` {
		autoAttackOn := false
		if rest == `on` {
			autoAttackOn = true
		} else if rest == `off` {
			autoAttackOn = false
		} else {
			user.SendText(`Usage: <ansi fg="command">party autoattack [on/off]</ansi>`)
			return true, nil
		}

		wasOnBefore := currentParty.SetAutoAttack(user.UserId, autoAttackOn)

		if autoAttackOn {
			if wasOnBefore {
				user.SendText(`You already have auto-attack enabled.`)
			} else {
				user.SendText(`You are now auto-attacking with your party.`)
			}
		} else {
			if wasOnBefore {
				user.SendText(`You are no longer auto-attacking with your party.`)
			} else {
				user.SendText(`You already have auto-attacking disabled.`)
			}
		}

		//
		// User party behavior changed
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `behavior`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})
	}

	if partyCommand == `leave` || partyCommand == `quit` {

		if currentParty.IsLeader(user.UserId) {

			if len(currentParty.UserIds) <= 1 {

				affected := append(currentParty.GetMembers(), currentParty.GetInvited()...)
				if !currentParty.TryDisband() {
					return true, parties.LastError()
				}
				//
				// Party is disbanded
				//
				events.AddToQueue(events.PartyUpdated{
					Action:  `disbanded`,
					UserIds: affected,
				})

				user.EventLog.Add(`party`, `Disbanded your party`)
				user.SendText(`You disbanded the party.`)

				return true, nil
			}

			if !currentParty.Leave(user.UserId) {
				return true, parties.LastError()
			}
			for _, uid := range currentParty.GetMembers() {
				if u := users.GetByUserId(uid); u != nil {
					if currentParty.IsLeader(uid) {
						u.EventLog.Add(`party`, `Promoted to party leader`)
						u.SendText(`You are now the leader of the party. Following is off for everyone.`)
					} else {
						u.SendText(fmt.Sprintf(`Player #%d is now the party leader. Following is off; use party follow on to consent.`, currentParty.LeaderUserId))
					}
				}
			}
			events.AddToQueue(events.PartyUpdated{
				Action:  `promotion`,
				UserIds: append(append(currentParty.GetMembers(), currentParty.GetInvited()...), user.UserId),
			})

			user.EventLog.Add(`party`, `Left the party`)
			user.SendText(`You left the party.`)

			return true, nil
		}

		//
		// User is leaving the party
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `left`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})

		if !currentParty.Leave(user.UserId) {
			return true, parties.LastError()
		}
		user.EventLog.Add(`party`, `Left the party`)
		user.SendText(`You left the party.`)

		for _, uid := range currentParty.UserIds {
			if uid == user.UserId {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> left the party.`, user.Character.Name))
			}
		}

	}

	if partyCommand == `disband` || partyCommand == `stop` {

		if !currentParty.IsLeader(user.UserId) {
			user.SendText(`You are not the leader of your party.`)
			return true, nil
		}

		members, invited := currentParty.GetMembers(), currentParty.GetInvited()
		if !currentParty.TryDisband() {
			return true, parties.LastError()
		}
		for _, uid := range members {
			if uid == user.UserId {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
			}
		}
		for _, uid := range invited {
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
			}
		}

		//
		// The party is being disbanded
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `disbanded`,
			UserIds: append(members, invited...),
		})

		user.EventLog.Add(`party`, `Disbanded the party`)
		user.SendText(`You disbanded the party.`)

		return true, nil
	}

	if partyCommand == `kick` {

		if !currentParty.IsLeader(user.UserId) {
			user.SendText(`You are not the leader of your party.`)
			return true, nil
		}

		allMembers := []string{}
		memberIds := map[string]int{}
		for _, uid := range currentParty.GetMembers() {
			u := users.GetByUserId(uid)
			if u == nil {
				u, _ = users.LoadUserFile(uid)
			}
			if u == nil {
				allMembers = append(allMembers, fmt.Sprintf("@%d", uid))
				memberIds[fmt.Sprintf("@%d", uid)] = uid
				continue
			}
			allMembers = append(allMembers, u.Character.Name)
			memberIds[u.Character.Name] = uid
		}

		matchUser, closeMatchUser := util.FindMatchIn(rest, allMembers...)
		if matchUser == `` {
			matchUser = closeMatchUser
		}

		if matchUser == `` {
			user.SendText(fmt.Sprintf(`%s not found.`, rest))
			return true, nil
		}

		//
		// The user was kicked from the party
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `left`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})

		kickUserId := memberIds[matchUser]

		if !currentParty.Leave(kickUserId) {
			return true, parties.LastError()
		}

		if u := users.GetByUserId(kickUserId); u != nil {
			u.SendText(`You were kicked from the party.`)
		}

		for _, uid := range currentParty.UserIds {
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> was kicked from the party.`, matchUser))
			}
		}
	}

	if partyCommand == `promote` {

		if !currentParty.IsLeader(user.UserId) {
			user.SendText(`You are not the leader of your party.`)
			return true, nil
		}

		allMembers := []string{}
		memberIds := map[string]int{}
		for _, uid := range currentParty.GetMembers() {
			u := users.GetByUserId(uid)
			if u == nil {
				u, _ = users.LoadUserFile(uid)
			}
			if u == nil {
				allMembers = append(allMembers, fmt.Sprintf("@%d", uid))
				memberIds[fmt.Sprintf("@%d", uid)] = uid
				continue
			}
			allMembers = append(allMembers, u.Character.Name)
			memberIds[u.Character.Name] = uid
		}

		matchUser, closeMatchUser := util.FindMatchIn(rest, allMembers...)
		if matchUser == `` {
			matchUser = closeMatchUser
		}

		if matchUser == `` {
			user.SendText(fmt.Sprintf(`%s not found.`, rest))
			return true, nil
		}

		//
		// User was promoted to leader
		//
		events.AddToQueue(events.PartyUpdated{
			Action:  `promotion`,
			UserIds: append(currentParty.GetMembers(), currentParty.GetInvited()...),
		})

		promoteUserId := memberIds[matchUser]

		if !currentParty.Promote(promoteUserId) {
			return true, parties.LastError()
		}

		if u := users.GetByUserId(promoteUserId); u != nil {
			u.EventLog.Add(`party`, `Promoted to party leader`)
			u.SendText(`You have been promoted to party leader. Following consent has been cleared.`)
		}

		for _, uid := range currentParty.UserIds {
			if uid != promoteUserId {
				if u := users.GetByUserId(uid); u != nil {
					u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the party leader. Following is off; use party follow on to consent.`, matchUser))
				}
			}
		}

	}

	if partyCommand == `chat` || partyCommand == `say` {

		if len(rest) == 0 {
			user.SendText(`What do you want to say?`)
			return true, nil
		}

		for _, uId := range currentParty.GetMembers() {
			if uId == user.UserId {
				continue
			}
			if u := users.GetByUserId(uId); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> <ansi fg="username">%s</ansi> says, "<ansi fg="yellow">%s</ansi>`, user.Character.Name, rest))
			}
		}

		user.SendText(fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> You say, "<ansi fg="yellow">%s</ansi>"`, rest))

		events.AddToQueue(events.Communication{
			SourceUserId: user.UserId,
			CommType:     `party`,
			Name:         user.Character.Name,
			Message:      rest,
		})
	}

	return true, nil
}
