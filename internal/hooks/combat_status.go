package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 30a: the combat round's side of status effects (internal/status).
// Statuses are ticked here, once per combat round, never by game rounds, so
// the world clock is untouched and a status lasts the same number of combat
// rounds at any combat cadence.

// statusHolder is a character a status sits on, with what its lines and
// events need.
type statusHolder struct {
	char   *characters.Character
	ref    combatstream.Ref
	user   *users.UserRecord // nil for a mob
	mob    *mobs.Mob         // nil for a player
	roomId int
}

func userHolder(u *users.UserRecord) statusHolder {
	return statusHolder{char: u.Character, ref: userRef(u), user: u, roomId: u.Character.RoomId}
}

func mobHolder(m *mobs.Mob) statusHolder {
	return statusHolder{char: &m.Character, ref: mobRef(m), mob: m, roomId: m.Character.RoomId}
}

// tag is the holder's name as narration prints it: a player's as typed, a
// mob's with its article and fight ordinal.
func (h statusHolder) tag() string {
	if h.user != nil {
		return userTag(h.user.Character.Name)
	}
	return named(mobTag(mobName(h.mob.InstanceId)))
}

// say tells the holder (second person) and the room (third) one status line;
// a mob has no one to tell but the room.
func (h statusHolder) say(you, other, suffix string) {
	room := rooms.LoadRoom(h.roomId)
	if h.user != nil {
		h.user.SendText(you + suffix)
		if room != nil {
			room.SendText(util.CapitalizeFirst(fmt.Sprintf(other, h.tag()))+suffix, h.user.UserId)
		}
		return
	}
	if room != nil {
		room.SendText(util.CapitalizeFirst(fmt.Sprintf(other, h.tag())) + suffix)
	}
}

// inFight reports whether the holder is still in a fight: one is open in
// its room, or it has an aim (a player-versus-player fight opens none).
func (h statusHolder) inFight(fightRooms map[int]bool) bool {
	return fightRooms[h.roomId] || h.char.Aggro != nil
}

// tickStatuses moves the holder's statuses on one combat round, with the
// lines and events each one earns.
func tickStatuses(h statusHolder) {
	for _, ch := range status.Tick(h.char) {
		if ch.Damage > 0 {
			h.say(ch.Spec.TickYou, ch.Spec.TickOther, fmt.Sprintf(" (%d damage, %s)", ch.Damage, ch.Spec.Word))
			emitCombat(combatstream.Event{Kind: combatstream.StatusTick, RoomId: h.roomId, Target: h.ref, Damage: ch.Damage, BuffId: ch.Spec.Id, Status: ch.Spec.Word})
		}
		if ch.Expired {
			h.say(ch.Spec.EndYou, ch.Spec.EndOther, ``)
			emitCombat(combatstream.Event{Kind: combatstream.StatusExpired, RoomId: h.roomId, Target: h.ref, BuffId: ch.Spec.Id, Status: ch.Spec.Word})
		}
	}
}

// statusPass runs at the top of a combat round, before any blow: statuses
// on a holder with no fight left are cleared without a word (a leftover of
// a restart, or of a flight); the rest tick. Anyone a tick brought down is
// resolved at once, in the round it happens.
func statusPass() {
	fightRooms := map[int]bool{}
	for _, fi := range combatstream.Default().OpenFights() {
		fightRooms[fi.RoomId] = true
	}

	var downPlayers, downMobs []int
	// felled reports whether a tick took the holder down or out.
	felled := func(before, after int, player bool) bool {
		if player {
			return (before >= 1 && after < 1) || (before > -10 && after <= -10)
		}
		return before >= 1 && after < 1
	}

	for _, userId := range users.GetOnlineUserIds() {
		u := users.GetByUserId(userId)
		if u == nil || u.Character == nil || !status.Has(u.Character) {
			continue
		}
		h := userHolder(u)
		if !h.inFight(fightRooms) {
			status.Clear(u.Character)
			continue
		}
		before := u.Character.Health
		tickStatuses(h)
		if felled(before, u.Character.Health, true) {
			downPlayers = append(downPlayers, userId)
		}
	}

	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instanceId)
		if m == nil || !status.Has(&m.Character) {
			continue
		}
		h := mobHolder(m)
		if !h.inFight(fightRooms) {
			status.Clear(&m.Character)
			continue
		}
		before := m.Character.Health
		tickStatuses(h)
		if felled(before, m.Character.Health, false) {
			downMobs = append(downMobs, instanceId)
		}
	}

	if len(downPlayers)+len(downMobs) > 0 {
		handleAffected(downPlayers, downMobs)
	}
}

// statusCostsAction reports whether a status costs the holder this round's
// action, telling the room and the stream when it does.
func statusCostsAction(h statusHolder) bool {
	spec, lost := status.LostAction(h.char)
	if !lost {
		return false
	}
	h.say(spec.LoseYou, spec.LoseOther, fmt.Sprintf(" (%s)", spec.Word))
	emitCombat(combatstream.Event{Kind: combatstream.StatusTick, RoomId: h.roomId, Target: h.ref, BuffId: spec.Id, Status: spec.Word, Outcome: combatstream.OutcomeLostAction})
	return true
}

// clearFightStatuses ends the statuses of everyone in a fight that is ending.
func clearFightStatuses(fi combatstream.FightInfo) {
	for _, r := range append(append([]combatstream.Ref{}, fi.Company...), fi.Enemies...) {
		switch {
		case r.UserId > 0:
			if u := users.GetByUserId(r.UserId); u != nil && u.Character != nil {
				status.Clear(u.Character)
			}
		case r.MobInstanceId > 0:
			if m := mobs.GetInstance(r.MobInstanceId); m != nil {
				status.Clear(&m.Character)
			}
		}
	}
}
