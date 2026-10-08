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
// Phase 40g2: allied companies. Each allied company fights its own battle
// (33d) over the same enemy, so its happenings ride its own fight. The
// fight's leader's consenting allies, in a battle against some of the same
// mobs in the same room, are sent them too, their members as
// "a:<leader>:<key>" (the ids Company.Battle.allies lists). Only what a
// watcher would see: strikes, casts, deaths and flight; never an ally's
// health, statuses or tactics, and nothing that ends or starts its fight.
// Each payload also carries the receiver's combat pace, so the screen
// paces its animation by it instead of guessing.
//
// What is never sent: enemy health or maximums, an enemy the player can't
// make out (a ref of "?"), a secret status, another company's anything.
// The feed goes to the leader of the fight only, and only to a client that
// asked for it: the web client, or one that supports the module by name.

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const battleEventModule = "Company.Battle.Event"

// battleUnseen is the ref of a fighter the player can't make out; the
// client draws a silhouette.
const battleUnseen = "?"

// battleEvent is one combat happening. Fields a kind doesn't use are
// omitted.
type battleEvent struct {
	Seq     uint64 `json:"seq"`
	Slot    int    `json:"slot,omitempty"` // the round's turn slot acting (Phase 82b); 0 outside a turn
	Kind    string `json:"kind"`
	Src     string `json:"src,omitempty"`
	Tgt     string `json:"tgt,omitempty"`
	Prev    string `json:"prev,omitempty"` // target-change: the target before
	Outcome string `json:"outcome,omitempty"`
	Damage  int    `json:"damage,omitempty"`
	Crit    bool   `json:"crit,omitempty"`
	Quality string `json:"quality,omitempty"` // glancing or telling
	Weapon  string `json:"weapon,omitempty"`
	Spell   string `json:"spell,omitempty"`
	// SpellName is the spell's display name (40g review: the battle
	// screen's last-blow line), as the narration names it.
	SpellName string   `json:"spell_name,omitempty"`
	Amount    int      `json:"amount,omitempty"`
	HeldBack  int      `json:"held_back,omitempty"`
	Status    string   `json:"status,omitempty"`
	Rule      string   `json:"rule,omitempty"`
	Defenses  []string `json:"defenses,omitempty"`
	// Explain is the round's strikes in plain lines (Phase 62), the
	// engine's own numbers: what the hit needed and rolled, the defence it
	// met, armor, named modifiers. Only for the player's own company's
	// rounds against a foe they can make out.
	Explain []string `json:"explain,omitempty"`
	// fight-start: who is in the fight (enemies only when seen).
	Company []string `json:"company,omitempty"`
	Enemies []string `json:"enemies,omitempty"`
}

// battleEventItem is what rides the pacer: one entry and the fight and
// round it belongs to.
type battleEventItem struct {
	fight      uint64
	round      uint64
	fightRound uint64
	event      battleEvent
}

type battleEventPayload struct {
	Fight uint64 `json:"fight"`
	Round uint64 `json:"round"` // the server's round counter
	// FightRound counts the fight's own rounds from 1 (Phase 40g: the
	// battle screen's title); 0 when the fight's start is unknown.
	FightRound uint64 `json:"fight_round,omitempty"`
	// Pace is the receiver's combat pace (fast, normal, slow or off), which
	// the screen's animation budgets follow.
	Pace   string        `json:"pace,omitempty"`
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
	userId int
	unseen func(instanceId int) bool // an enemy the player can't make out
	// allyLeader is set when the event is an allied company's: the leader
	// of the fight it rode, whose members are "a:<leader>:<key>".
	allyLeader int
}

// refID names a combatant as Company.Battle and Company do: a member key
// for the player's own company (the leader is "leader", the key of their
// cell in Company.Battle.positions), "me" for the player when they lead no
// company, "m:<instance>" for an enemy or any other mob, "u:<id>" for
// another player, "?" for an enemy not made out.
func (v battleViewer) refID(r combatstream.Ref) string {
	switch {
	case r.Zero():
		return ""
	case v.allyLeader > 0 && r.LeaderUserId == v.allyLeader && r.MemberKey != "":
		return allyMemberRef(v.allyLeader, r.MemberKey)
	case r.LeaderUserId == v.userId && r.MemberKey != "":
		return r.MemberKey
	case r.UserId == v.userId:
		return "me"
	case r.UserId > 0:
		return "u:" + strconv.Itoa(r.UserId)
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
		Slot:     e.Slot,
		Kind:     string(e.Kind),
		Src:      v.refID(e.Source),
		Tgt:      v.refID(e.Target),
		Prev:     v.refID(e.Previous),
		Outcome:  e.Outcome,
		Damage:   e.Damage,
		Crit:     e.Crit && (e.Kind != combatstream.Attack || combatstream.CritLanded(e.Strikes, e.Crit)),
		Quality:  e.Quality,
		Weapon:   e.WeaponType,
		Spell:    e.SpellId,
		Amount:   e.Amount,
		HeldBack: e.HeldBack,
		Status:   e.Status,
		Rule:     e.Rule,
		Defenses: e.Defenses,
	}
	if e.Kind == combatstream.Attack && v.allyLeader == 0 && !v.masked(e.Source) && !v.masked(e.Target) && (e.Source.LeaderUserId == v.userId || e.Target.LeaderUserId == v.userId || e.Source.UserId == v.userId || e.Target.UserId == v.userId) {
		be.Explain = combatstream.Breakdown(e.Strikes)
	}
	if e.SpellId != "" {
		if sp := spells.GetSpell(e.SpellId); sp != nil {
			be.SpellName = sp.Name
		}
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

// allyKinds are the happenings an ally's watcher is shown.
var allyKinds = map[combatstream.Kind]bool{
	combatstream.TargetChange: true, combatstream.Attack: true, combatstream.SpellHit: true, combatstream.Heal: true,
	combatstream.CastStart: true, combatstream.CastProgress: true, combatstream.CastComplete: true,
	combatstream.WindUpStart: true, combatstream.WindUpLand: true, combatstream.Interrupt: true,
	combatstream.StatusApplied: true, combatstream.StatusExpired: true, combatstream.StatusTick: true,
	combatstream.Death: true, combatstream.Flee: true, combatstream.Ability: true,
}

// isAllyRef reports whether a ref, in an allied fight's event, names one of
// that company's side: a member ("a:<leader>:<key>") or a player without a
// formation ("u:<id>"), whose numbers are as private as a member's.
func isAllyRef(id string) bool { return strings.HasPrefix(id, "a:") || strings.HasPrefix(id, "u:") }

// scrubAlly removes what another company's watcher is not shown of its
// members: the numbers of what befell them (the narration of their own
// company's fight is theirs, not ours) and the status a blow left on them.
func scrubAlly(be battleEvent) battleEvent {
	if isAllyRef(be.Tgt) {
		be.Damage, be.Amount, be.HeldBack, be.Status, be.Explain = 0, 0, 0, "", nil
	}
	return be
}

// relayToAllies queues an allied company's event to each allied leader
// fighting the same enemy here. The fight must be open: its roster tells
// which mobs it fights.
func relayToAllies(e combatstream.Event, leaderId int, fi combatstream.FightInfo, haveFight bool) {
	if !haveFight || !allyKinds[e.Kind] {
		return
	}
	for _, allyId := range parties.AlliedLeaders(leaderId) {
		user := users.GetByUserId(allyId)
		if user == nil || user.Character == nil || !wantsBattleEvents(user.ConnectionId()) {
			continue
		}
		b, ok := battle.Current(allyId)
		if !ok || b.FightID == 0 || b.FightID == e.FightID || b.RoomId != fi.RoomId || user.Character.RoomId != fi.RoomId {
			continue
		}
		shared := false
		for _, r := range fi.Enemies {
			if b.Enemies[r.MobInstanceId] {
				shared = true
				break
			}
		}
		if !shared {
			continue
		}
		v := viewerFor(user, e.RoomId, b.FightID)
		v.allyLeader = leaderId
		// A status on one of its members is not shown; nor is a status on
		// a fighter the receiver can't make out.
		if be, ok := buildBattleEvent(v, e, fi, true); ok && !(isAllyRef(be.Tgt) && (e.Kind == combatstream.StatusApplied || e.Kind == combatstream.StatusExpired || e.Kind == combatstream.StatusTick)) {
			own, ownOk := combatstream.Default().Fight(b.FightID)
			events.AddToQueue(events.CombatData{UserId: allyId, Data: battleEventItem{fight: b.FightID, round: e.Round, fightRound: fightRound(e, own, ownOk), event: scrubAlly(be)}})
		}
	}
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
	relayToAllies(e, leaderId, fi, haveFight)
	user := users.GetByUserId(leaderId)
	if user == nil || user.Character == nil || !wantsBattleEvents(user.ConnectionId()) {
		return
	}
	be, ok := buildBattleEvent(viewerFor(user, e.RoomId, e.FightID), e, fi, haveFight)
	if !ok {
		return
	}
	events.AddToQueue(events.CombatData{UserId: leaderId, Data: battleEventItem{fight: e.FightID, round: e.Round, fightRound: fightRound(e, fi, haveFight), event: be}})
}

// fightRound is the event's round counted from the fight's first (1), from
// the open fight or, for a fight-end, its summary; 0 when unknown.
func fightRound(e combatstream.Event, fi combatstream.FightInfo, haveFight bool) uint64 {
	start := uint64(0)
	if haveFight {
		start = fi.StartRound
	} else if e.Summary != nil {
		start = e.Summary.StartRound
	}
	if start == 0 || e.Round < start {
		return 0
	}
	return e.Round - start + 1
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
			cur = &battleEventPayload{Fight: it.fight, Round: it.round, FightRound: it.fightRound, Pace: string(hooks.PaceOf(user))}
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
