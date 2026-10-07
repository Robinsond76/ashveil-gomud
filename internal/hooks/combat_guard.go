package hooks

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/bonds"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 30c2: guardians. When an enemy's weapon blow is about to land on a
// member of a player's company in that player's battle (after 11c's
// front-row interception and legality have settled who), a guardian whose
// ward that member is steps in and takes the blow instead: one guard
// spent, a line, and a guard-used event. The blow is then resolved against
// the guardian by 11c's own interception paths, so the attacker's aim is
// never touched. Guard counts live on the battle (internal/battle); this
// file only reads and spends them, on the game loop.

// guardMember is one living member of a player's company here.
type guardMember struct {
	key  company.MemberKey
	char *characters.Character
	user *users.UserRecord // the player; nil for a companion
	mob  *mobs.Mob         // a companion; nil for the player
}

func (m guardMember) ref() combatstream.Ref {
	if m.user != nil {
		return userRef(m.user)
	}
	return mobRef(m.mob)
}

// tag is the member's name as narration prints it.
func (m guardMember) tag() string {
	if m.user != nil {
		return userTag(m.user.Character.Name)
	}
	return named(mobTag(mobName(m.mob.InstanceId)))
}

// guardMembers lists the living members of leader's company in the
// leader's room, in formation order (row by row, left to right), the
// unplaced after, the player first among them.
func guardMembers(leader *users.UserRecord, f company.Formation) []guardMember {
	var out []guardMember
	if leader.Character.Health > 0 {
		out = append(out, guardMember{key: company.LeaderMemberKey, char: leader.Character, user: leader})
	}
	room := rooms.LoadRoom(leader.Character.RoomId)
	if room != nil {
		var companions []guardMember
		for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
			leaderId, key, ok := company.LeaderAndKeyForInstance(instanceId)
			m := mobs.GetInstance(instanceId)
			if !ok || leaderId != leader.UserId || m == nil || m.Character.Health < 1 {
				continue
			}
			companions = append(companions, guardMember{key: key, char: &m.Character, mob: m})
		}
		sort.Slice(companions, func(i, j int) bool { return companions[i].mob.InstanceId < companions[j].mob.InstanceId })
		out = append(out, companions...)
	}
	order := func(key company.MemberKey) int {
		if row, col, ok := f.Find(key); ok {
			return row*company.FormationCols + col
		}
		return company.FormationRows * company.FormationCols // unplaced after
	}
	sort.SliceStable(out, func(i, j int) bool { return order(out[i].key) < order(out[j].key) })
	return out
}

// ableToGuard reports whether a member can step in now: standing (alive,
// not no-combat) and not knocked down or stunned (the owner's decision
// 12).
func ableToGuard(m guardMember) bool {
	if m.char.Health < 1 || m.char.HasBuffFlag("no-combat") {
		return false
	}
	return !status.Grounded(m.char)
}

// guardianFor finds the guardian who steps in for a blow about to land on
// struck, spends its guard, and says so. ok is false when no one does.
// struckAlready (a sweep's) names members already hit by this action, who
// don't step in again.
func guardianFor(leader *users.UserRecord, f company.Formation, struck company.MemberKey, struckAlready ...map[company.MemberKey]bool) (guardMember, bool) {
	b, inBattle := battle.Current(leader.UserId)
	if !inBattle || leader.Character.RoomId != b.RoomId {
		return guardMember{}, false
	}
	members := guardMembers(leader, f)
	var ward guardMember
	found := false
	for _, m := range members {
		if m.key == struck {
			ward, found = m, true
		}
	}
	if !found {
		return guardMember{}, false
	}
	// Phase 39b: a Hatamoto's Bodyguard steps in for the company leader,
	// whatever strategy it fights by, a few times a battle.
	if struck == company.LeaderMemberKey {
		narrow := enemyparty.Narrow(rooms.LoadRoom(leader.Character.RoomId))
		for _, g := range members {
			if g.key == struck || (len(struckAlready) > 0 && struckAlready[0][g.key]) ||
				bodyguardLeft(g.char) < 1 || !ableToGuard(g) || !formationcombat.GuardGround(f, g.key, struck, narrow) {
				continue
			}
			g.char.RTState().Bodyguards++
			announceGuard(leader, b.FightID, g, ward, bodyguardLeft(g.char))
			return g, true
		}
	}
	for _, g := range members {
		if g.key == struck || (len(struckAlready) > 0 && struckAlready[0][g.key]) {
			continue
		}
		s := enemyparty.MemberStrategy(leader.UserId, g.key)
		// Phase 38b: a Hierarch's Angel guards the most hurt ally, a few
		// times a battle, whatever strategy a summon (which has none) holds.
		angel := angelGuardsLeft(g.char) > 0
		doll := dollGuardsLeft(g.char) > 0  // Phase 39d: Guard String
		bear := beastGuardsLeft(g.char) > 0 // Phase 39e: a war bear guards
		if angel || doll || bear {
			s.Role, s.Ward = strategy.Guardian, ""
		}
		// Phase 61: a guard order makes the member a guardian for its ward
		// this round, whatever its strategy.
		if w, ok := orderedWard(leader.UserId, g.key); ok && !angel && !doll && !bear {
			s.Role, s.Ward = strategy.Guardian, string(w)
		}
		if s.Role == strategy.Guardian && !angel && !doll && !bear {
			battle.CaptureGuards(leader.UserId, string(g.key), g.char.Level) // Phase 35b
		}
		if s.Role != strategy.Guardian || !ableToGuard(g) || !angel && !doll && !bear && battle.GuardsLeft(leader.UserId, string(g.key)) < 1 {
			continue
		}
		// A set ward still in the company but not here (away, fallen) is
		// guarded by no one else: only one gone from the company reads as
		// none, the most hurt (review finding 1).
		if s.Ward != "" && !presentIn(members, s.Ward) && inCompany(leader.UserId, s.Ward) {
			continue
		}
		// Phase 65: a guardian set to guard a rival won't, and says so.
		if s.Ward != "" && s.Ward == string(struck) && !angel && !doll && !bear && bondRefuses(leader, f, b.FightID, g, ward, struck) {
			continue
		}
		guarded := make([]strategy.Guarded, 0, len(members))
		for _, m := range members {
			guarded = append(guarded, strategy.Guarded{
				Key: string(m.key), HP: m.char.Health, MaxHP: m.char.HealthMax.Value,
				// Phase 65: a rival is never the most hurt one it guards.
				InReach: formationcombat.GuardGround(f, g.key, m.key, enemyparty.Narrow(rooms.LoadRoom(leader.Character.RoomId))) &&
					!bonds.IsRival(company.BondValue(leader.UserId, g.key, m.key)),
			})
		}
		if w, ok := strategy.GuardWard(string(g.key), s.Ward, guarded); !ok || w != string(struck) {
			continue
		}
		var left int
		if angel {
			g.char.RT.Summon.GuardsUsed++
			left = angelGuardsLeft(g.char)
		} else if doll {
			master := dollMasterOf(g.char)
			master.RTState().DollGuards++
			left = dollGuardsLeft(g.char)
		} else if bear {
			g.char.RT.Beast.GuardsUsed++
			left = beastGuardsLeft(g.char)
		} else {
			var ok bool
			if left, ok = battle.SpendGuard(leader.UserId, string(g.key)); !ok {
				continue
			}
		}
		announceGuard(leader, b.FightID, g, ward, left)
		company.BondEvent(leader.UserId, g.key, struck, bonds.Rescue) // Phase 65
		return g, true
	}
	// Phase 65: a friend steps in for a friend who is hurt, whatever its role.
	var already map[company.MemberKey]bool
	if len(struckAlready) > 0 {
		already = struckAlready[0]
	}
	return bondFriendGuard(leader, f, b.FightID, members, ward, struck, already)
}

func presentIn(members []guardMember, key string) bool {
	for _, m := range members {
		if string(m.key) == key {
			return true
		}
	}
	return false
}

// inCompany reports whether key names a member of leaderId's company: the
// player, or a companion on the record (alive or dead).
func inCompany(leaderId int, key string) bool {
	if key == string(company.LeaderMemberKey) {
		return true
	}
	views, _ := company.CompanyMembers(leaderId)
	for _, v := range views {
		if string(company.CompanionMemberKey(v.ID)) == key {
			return true
		}
	}
	return false
}

// guardLine is the room's line for a guard: "Tamsin Reed steps in front of
// Aria. (guard, 1 left)". guardian and ward are tagged names, or "You" /
// "you".
func guardLine(guardian, ward string, left int) string {
	count := "none left"
	if left > 0 {
		count = fmt.Sprintf("%d left", left)
	}
	if guardian == `You` {
		return fmt.Sprintf(`You step in front of %s. (guard, %s)`, ward, count)
	}
	return util.CapitalizeFirst(fmt.Sprintf(`%s steps in front of %s. (guard, %s)`, guardian, ward, count))
}

// announceGuard tells the player and the room, and reports the guard (and
// the last one) on the combat event stream.
func announceGuard(leader *users.UserRecord, fightID uint64, g, ward guardMember, left int) {
	roomId := leader.Character.RoomId
	switch {
	case g.user != nil:
		leader.SendText(guardLine(`You`, ward.tag(), left))
	case ward.user != nil:
		leader.SendText(guardLine(g.tag(), `you`, left))
	default:
		leader.SendText(guardLine(g.tag(), ward.tag(), left))
	}
	if room := rooms.LoadRoom(roomId); room != nil {
		room.SendText(guardLine(g.tag(), ward.tag(), left), leader.UserId)
	}
	emitCombat(combatstream.Event{Kind: combatstream.GuardUsed, FightID: fightID, RoomId: roomId, Source: g.ref(), Target: ward.ref()})
	if left == 0 {
		emitCombat(combatstream.Event{Kind: combatstream.GuardExhausted, FightID: fightID, RoomId: roomId, Source: g.ref()})
	}
}

// guardPass counts one combat round toward every spent guard's return.
func guardPass() { battle.TickGuards() }

// angelGuardsLeft is how many guards a summoned Angel has left this battle
// (none for any other character).
func angelGuardsLeft(c *characters.Character) int {
	if c == nil || c.RT == nil || c.RT.Summon == nil || c.RT.Summon.Kind != "angel" {
		return 0
	}
	return max(0, c.RT.Summon.Guards-c.RT.Summon.GuardsUsed)
}
