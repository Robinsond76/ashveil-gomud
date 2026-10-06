package gmcp

// Phase 40e: "Company.Battle.Event", the web client's feed of structured
// combat happenings for the battle screen (40f, 40g). Each combat event
// (internal/combatstream) of the player's fight becomes one entry: who did
// what to whom, the result, the damage the narration shows, and statuses.
//
// Entries are queued as events.CombatData as the happening is emitted, so
// they are dispatched in order with the round's narration and released by
// the player's combat pace with it (hooks.CombatData_Hold): with the next
// text line that follows them, or when the round's last line goes out. A
// batch released together is one message. With pacing off, or for what the
// player typed, they go out at once. Nothing is stored; a copyover ends the
// fight, and the client resyncs from the next Company.Battle snapshot.
//
// What is never sent: enemy health or maximums, an enemy the player can't
// make out (a ref of "?"), a secret status, another company's anything.
// The feed goes to the leader of the fight only, and only to a client that
// asked for it: the web client, or one that supports the module by name.

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const battleEventModule = "Company.Battle.Event"

// battleUnseen is the ref of a fighter the player can't make out; the
// client draws a silhouette.
const battleUnseen = "?"

// battleEvent is one combat happening. Fields a kind doesn't use are
// omitted.
type battleEvent struct {
	Seq      uint64   `json:"seq"`
	Kind     string   `json:"kind"`
	Src      string   `json:"src,omitempty"`
	Tgt      string   `json:"tgt,omitempty"`
	Prev     string   `json:"prev,omitempty"` // target-change: the target before
	Outcome  string   `json:"outcome,omitempty"`
	Damage   int      `json:"damage,omitempty"`
	Crit     bool     `json:"crit,omitempty"`
	Quality  string   `json:"quality,omitempty"` // glancing or telling
	Weapon   string   `json:"weapon,omitempty"`
	Spell    string   `json:"spell,omitempty"`
	Amount   int      `json:"amount,omitempty"`
	HeldBack int      `json:"held_back,omitempty"`
	Status   string   `json:"status,omitempty"`
	Rule     string   `json:"rule,omitempty"`
	Defenses []string `json:"defenses,omitempty"`
	// fight-start: who is in the fight (enemies only when seen).
	Company []string `json:"company,omitempty"`
	Enemies []string `json:"enemies,omitempty"`
}

// battleEventItem is what rides the pacer: one entry and the fight and
// round it belongs to.
type battleEventItem struct {
	fight uint64
	round uint64
	event battleEvent
}

type battleEventPayload struct {
	Fight  uint64        `json:"fight"`
	Round  uint64        `json:"round"`
	Events []battleEvent `json:"events"`
}

// wantsBattleEvents reports whether a connection takes the feed: a web
// client, or a GMCP client that asked for the module by name.
func wantsBattleEvents(connectionId uint64) bool {
	if connectionId == 0 {
		return false
	}
	settings, ok := gmcpModule.cache.Get(connectionId)
	if !ok || !settings.GMCPAccepted {
		return false
	}
	return settings.Client.Name == `WebClient` || settings.EnabledModules[battleEventModule] > 0
}

// battleViewer is what the ID mapping needs of the player whose feed it is.
type battleViewer struct {
	userId  int
	unseen  func(instanceId int) bool // an enemy the player can't make out
	enemies map[int]bool              // the fight's enemies, by instance
}

// refID names a combatant as Company.Battle and Company do: "me", a member
// key, "m:<instance>" for an enemy or any other mob, "u:<id>" for another
// player, "?" for an enemy not made out.
func (v battleViewer) refID(r combatstream.Ref) string {
	switch {
	case r.Zero():
		return ""
	case r.UserId == v.userId:
		return "me"
	case r.UserId > 0:
		return "u:" + strconv.Itoa(r.UserId)
	case r.LeaderUserId == v.userId && r.MemberKey != "":
		return r.MemberKey
	case v.unseen != nil && v.unseen(r.MobInstanceId):
		return battleUnseen
	}
	return mobID(r.MobInstanceId)
}

// masked reports whether an event ref is an enemy masked to "?".
func (v battleViewer) masked(r combatstream.Ref) bool { return v.refID(r) == battleUnseen }

// buildBattleEvent turns one stream event into its entry for a viewer. ok
// is false for an event the viewer is not shown.
func buildBattleEvent(v battleViewer, e combatstream.Event, fi combatstream.FightInfo, haveFight bool) (battleEvent, bool) {
	switch e.Kind {
	case combatstream.StatusApplied, combatstream.StatusExpired, combatstream.StatusTick:
		// A status on a fighter the player can't make out would give it away.
		if v.masked(e.Target) {
			return battleEvent{}, false
		}
		if e.BuffId > 0 {
			if spec := buffs.GetBuffSpec(e.BuffId); spec != nil && spec.Secret {
				return battleEvent{}, false
			}
		}
	}
	be := battleEvent{
		Seq:      e.Seq,
		Kind:     string(e.Kind),
		Src:      v.refID(e.Source),
		Tgt:      v.refID(e.Target),
		Prev:     v.refID(e.Previous),
		Outcome:  e.Outcome,
		Damage:   e.Damage,
		Crit:     e.Crit,
		Quality:  e.Quality,
		Weapon:   e.WeaponType,
		Spell:    e.SpellId,
		Amount:   e.Amount,
		HeldBack: e.HeldBack,
		Status:   e.Status,
		Rule:     e.Rule,
		Defenses: e.Defenses,
	}
	if e.Kind == combatstream.FightEnd {
		// The leader named in Source is the receiver; nothing else rides it.
		be.Src = ""
	}
	if e.Kind == combatstream.FightStart && haveFight {
		for _, r := range fi.Company {
			if id := v.refID(r); id != "" {
				be.Company = append(be.Company, id)
			}
		}
		for _, r := range fi.Enemies {
			if id := v.refID(r); id != "" && id != battleUnseen {
				be.Enemies = append(be.Enemies, id)
			}
		}
		be.Src = ""
	}
	return be, true
}

// viewerFor builds the viewer of a fight's leader: which enemies they can't
// make out (all of them in the dark, and any that is hidden or was when
// last seen).
func viewerFor(user *users.UserRecord, roomId int, fightId uint64) battleViewer {
	v := battleViewer{userId: user.UserId}
	dark := false
	if room := rooms.LoadRoom(roomId); room != nil && user.Character != nil {
		dark = room.VisibilityForUser(user) < 1 && !user.Character.HasBuffFlag("nightvision")
	}
	var key battleSeenKey
	if b, ok := battle.Current(user.UserId); ok {
		key = battleSeenKey{start: b.StartRound, fight: b.FightID, party: b.PartyID}
	}
	v.unseen = func(instanceId int) bool {
		if instanceId <= 0 {
			return false
		}
		if dark {
			return true
		}
		if m := mobs.GetInstance(instanceId); m != nil && m.Character.HasBuffFlag("hidden") {
			return true
		}
		return battleSeen.hiddenLast(user.UserId, key, instanceId)
	}
	return v
}

// onCombatEvent is the combat stream's sink: it routes an event to its
// fight's leader and queues it behind the round's narration.
func onCombatEvent(e combatstream.Event) {
	if e.FightID == 0 {
		return
	}
	var leaderId int
	var fi combatstream.FightInfo
	haveFight := false
	if f, ok := combatstream.Default().Fight(e.FightID); ok {
		fi, haveFight, leaderId = f, true, f.LeaderUserId
	} else if e.Kind == combatstream.FightEnd {
		leaderId = e.Source.UserId // the fight is already closed
	}
	if leaderId <= 0 {
		return
	}
	user := users.GetByUserId(leaderId)
	if user == nil || user.Character == nil || !wantsBattleEvents(user.ConnectionId()) {
		return
	}
	be, ok := buildBattleEvent(viewerFor(user, e.RoomId, e.FightID), e, fi, haveFight)
	if !ok {
		return
	}
	events.AddToQueue(events.CombatData{UserId: leaderId, Data: battleEventItem{fight: e.FightID, round: e.Round, event: be}})
}

// sendBattleEvents sends a released batch as messages, one per fight and
// round, in order.
func sendBattleEvents(userId int, batch []any) {
	user := users.GetByUserId(userId)
	if user == nil || !wantsBattleEvents(user.ConnectionId()) {
		return
	}
	var cur *battleEventPayload
	flush := func() {
		if cur != nil {
			events.AddToQueue(GMCPOut{UserId: userId, Module: battleEventModule, Payload: *cur})
			cur = nil
		}
	}
	for _, d := range batch {
		it, ok := d.(battleEventItem)
		if !ok {
			continue
		}
		if cur == nil || cur.Fight != it.fight || cur.Round != it.round {
			flush()
			cur = &battleEventPayload{Fight: it.fight, Round: it.round}
		}
		cur.Events = append(cur.Events, it.event)
	}
	flush()
}

func init() {
	combatstream.Default().Subscribe(onCombatEvent)
	hooks.SetCombatDataSender(sendBattleEvents)
}

// SubscribeBattleEventsForTest feeds a test's combat stream to the module,
// and returns a func ending it.
func SubscribeBattleEventsForTest(s *combatstream.Stream) (unsubscribe func()) {
	return s.Subscribe(onCombatEvent)
}
