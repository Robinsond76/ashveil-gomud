package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
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

// fightMembers is everyone in an open fight, by Ref key.
func fightMembers() map[string]bool {
	members := map[string]bool{}
	for _, fi := range combatstream.Default().OpenFights() {
		for _, r := range fi.Company {
			members[r.Key()] = true
		}
		for _, r := range fi.Enemies {
			members[r.Key()] = true
		}
	}
	return members
}

// inFight reports whether the holder is still in a fight: it is a member of
// an open one, or it is fighting another player (which opens none). Being
// in the same room as someone else's fight is not enough.
func (h statusHolder) inFight(members map[string]bool) bool {
	if members[h.ref.Key()] {
		return true
	}
	return h.user != nil && h.char.Aggro != nil && h.char.Aggro.UserId > 0
}

// statusBuffLands reports whether a status buff about to be applied (a
// queued Buff event) may land on its holder: only while the holder is still
// in a fight. A blow struck in a fight's last round is applied after the
// fight has ended and cleared its statuses; it lands on nobody.
func statusBuffLands(buffId int, char *characters.Character, user *users.UserRecord, mob *mobs.Mob) bool {
	if char.CombatWithdrawn {
		return false
	}
	if status.Get(buffId) == nil {
		return true
	}
	h := statusHolder{char: char, user: user, mob: mob}
	switch {
	case user != nil:
		h.ref = userRef(user)
	case mob != nil:
		h.ref = mobRef(mob)
	}
	return h.inFight(fightMembers())
}

// enemy reports whether the holder is a mob outside every company (an
// enemy, or a charmed pet): not a player or a company companion.
func (h statusHolder) enemy() bool {
	if h.user != nil || h.mob == nil {
		return false
	}
	_, _, companion := company.LeaderAndKeyForInstance(h.mob.InstanceId)
	return !companion
}

// woundable reports whether the holder can be wounded (Phase 30b): a
// player, a company companion, or (Phase 33i2) an enemy whose template
// allows it. A bleed's wound is light for everyone.
func (h statusHolder) woundable() bool {
	if h.user != nil {
		return true
	}
	ok, _ := combat.MobWounds(h.mob)
	return ok
}

// tickStatuses moves the holder's statuses on one combat round, with the
// lines and events each one earns.
func tickStatuses(h statusHolder) {
	hurt := false
	defer func() {
		// Phase 38a review: a bleed's or a burn's damage wakes a sleeper,
		// as a blow's does.
		if hurt && status.Wake(h.char) {
			spec := status.Get(status.Asleep)
			h.say(spec.EndYou, spec.EndOther, ``)
			emitCombat(combatstream.Event{Kind: combatstream.StatusExpired, RoomId: h.roomId, Target: h.ref, BuffId: spec.Id, Status: spec.Word})
		}
	}()
	for _, ch := range status.Tick(h.char) {
		hurt = hurt || ch.Damage > 0
		// Phase 30b: a bleed that runs its course leaves a light wound.
		if ch.Expired && ch.Spec.Id == status.Bleeding && h.woundable() {
			h.char.AddWound(wounds.Bled(ch.Stacks, util.Rand))
		}
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
	// Phase 38a: the hex immunity ledger counts combat rounds.
	hexes.Default.Tick()
	members := fightMembers()

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
		if u == nil || u.Character == nil {
			continue
		}
		hasStatus := status.Has(u.Character)
		if !hasStatus && !wounds.HasLight(u.Character.Wounds) {
			continue
		}
		h := userHolder(u)
		if !h.inFight(members) {
			status.Clear(u.Character)
			u.Character.Wounds = wounds.CloseLight(u.Character.Wounds)
			continue
		}
		if !hasStatus {
			continue
		}
		before := u.Character.Health
		tickStatuses(h)
		if felled(before, u.Character.Health, true) {
			downPlayers = append(downPlayers, userId)
			// A fall stops the fighting at once, as a killing blow does.
			u.Character.EndAggro()
			events.AddToQueue(events.AggroChanged{UserId: u.UserId, RoomId: u.Character.RoomId})
		}
	}

	for _, instanceId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instanceId)
		if m == nil {
			continue
		}
		hasStatus := status.Has(&m.Character)
		if !hasStatus && !wounds.HasLight(m.Character.Wounds) {
			continue
		}
		h := mobHolder(m)
		if !h.inFight(members) {
			status.Clear(&m.Character)
			m.Character.Wounds = wounds.CloseLight(m.Character.Wounds)
			continue
		}
		if !hasStatus {
			continue
		}
		before := m.Character.Health
		tickStatuses(h)
		if felled(before, m.Character.Health, false) {
			downMobs = append(downMobs, instanceId)
			m.Character.EndAggro()
			events.AddToQueue(events.AggroChanged{MobInstanceId: m.InstanceId, RoomId: m.Character.RoomId})
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

// clearFightStatuses ends the statuses of everyone in a fight that is
// ending, and closes their light wounds (Phase 30b).
func clearFightStatuses(fi combatstream.FightInfo) {
	for _, r := range append(append([]combatstream.Ref{}, fi.Company...), fi.Enemies...) {
		switch {
		case r.UserId > 0:
			if u := users.GetByUserId(r.UserId); u != nil && u.Character != nil {
				status.Clear(u.Character)
				u.Character.EndFightRT()
				u.Character.Wounds = wounds.CloseLight(u.Character.Wounds)
			}
		case r.MobInstanceId > 0:
			if m := mobs.GetInstance(r.MobInstanceId); m != nil {
				status.Clear(&m.Character)
				m.Character.EndFightRT()
				m.Character.Wounds = wounds.CloseLight(m.Character.Wounds)
			}
		}
	}
}
