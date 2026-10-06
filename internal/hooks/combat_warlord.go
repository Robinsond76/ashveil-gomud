package hooks

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 38c1: the Warlord (Mercenary elite) and the elite talent Second
// Wind. Everything here is runtime only, on the game loop: marks, stand-up
// watches and per-battle uses live on the battle's class state and are
// never saved.

// marked are the foes a Warlord has marked; expireMarks lifts each once the
// round after its mark is over (a mark lasts 2 rounds: the round it lands
// in and the next).
var marked []*characters.ClassRT

func expireMarks(round uint64) {
	keep := marked[:0]
	for _, rt := range marked {
		if rt.Mark > 0 && round <= rt.MarkRound+1 {
			keep = append(keep, rt)
			continue
		}
		rt.Mark = 0
	}
	clear(marked[len(keep):])
	marked = keep
}

// downFoe is a foe a Warlord knocked down, watched until it stands up
// (Relentless). seen is set once its knockdown is live; it stands up when
// that ends.
type downFoe struct {
	id    int
	round int
	seen  bool
}

// warlordDown are the watched foes by the Warlord who downed them.
var warlordDown = map[caster][]downFoe{}

// ResetWarlordForTest forgets every mark and stand-up watch.
func ResetWarlordForTest() {
	clear(marked)
	marked = nil
	clear(warlordDown)
}

// warlordTackle is what a landed Tackle adds for a Warlord: the mark, the
// broken armor, and the watch for the foe standing up again.
func warlordTackle(a actor, foe *mobs.Mob, target statusHolder) {
	fx := a.char.ClassEffects()
	if n := fx.Int(classes.MarkRuin); n > 0 {
		rt := foe.Character.RTState()
		rt.Mark, rt.MarkRound = n, combatRound.Load()
		if !slices.Contains(marked, rt) {
			marked = append(marked, rt)
		}
		a.holder.say(fmt.Sprintf("You mark %s for ruin.", target.tag()),
			"%s marks "+verbatim(target.tag())+" for ruin.", fmt.Sprintf(" (marked for ruin: +%d Attack for your allies, 2 rounds)", n))
		emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: foe.Character.RoomId, Source: a.ref, Target: target.ref, Status: "Marked for Ruin", Outcome: combatstream.OutcomeSucceeded})
	}
	if fx.Has(classes.Sunder) {
		events.AddToQueue(events.Buff{MobInstanceId: foe.InstanceId, BuffId: status.ArmorBroken, Source: `combat`, Triggers: 2})
		a.holder.say(fmt.Sprintf("You split %s's armor.", target.tag()),
			"%s splits "+verbatim(target.tag())+"'s armor.", " (sunder: armor broken, 2 rounds)")
	}
	if fx.Has(classes.Relentless) {
		warlordDown[a.who] = append(warlordDown[a.who], downFoe{id: foe.InstanceId, round: abilityRounds})
	}
}

// pushMeter adds action meter points to a fighter's meter: it speeds its
// next turn and never grants one outright.
func pushMeter(who caster, points int) bool {
	st := tempoMeters[who]
	if st == nil || points <= 0 {
		return false
	}
	st.meter.Points += float64(points)
	return true
}

// warlordPass runs after the round's statuses tick and before the meters
// fill: a foe a Warlord knocked down that has stood up speeds the Warlord
// (Relentless), and the first fall in the company speeds everyone standing
// (Warlord's Command). Nothing here takes or adds a turn.
func warlordPass() {
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || u.Character.RoomId != b.RoomId {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		side := sideActors(u, room)
		standing := 0
		for _, a := range side {
			if a.char.Health >= 1 && summonInfo(a.char) == nil {
				standing++
			}
		}
		for _, a := range side {
			fx := a.char.ClassEffects()
			if fx == nil || a.char.Health < 1 {
				continue
			}
			if fx.Has(classes.Relentless) {
				relentless(a)
			}
			rt := a.char.RTState()
			if fx.Has(classes.WarCommand) {
				if !rt.CmdUsed && rt.Standing > standing {
					rt.CmdUsed = true
					warlordsCommand(a, side, fx.Int(classes.WarCommand))
				}
			}
			rt.Standing = standing
		}
	}
}

// relentless watches the foes a Warlord knocked down.
func relentless(a actor) {
	watch := warlordDown[a.who]
	if len(watch) == 0 {
		return
	}
	keep := watch[:0]
	for _, w := range watch {
		foe := mobs.GetInstance(w.id)
		if foe == nil || foe.Character.Health < 1 || abilityRounds-w.round > 8 {
			continue
		}
		if status.Live(&foe.Character, status.KnockedDown) {
			w.seen = true
			keep = append(keep, w)
			continue
		}
		if !w.seen {
			keep = append(keep, w)
			continue
		}
		pts := a.char.ClassEffects().Int(classes.Relentless)
		if pushMeter(a.who, pts) {
			target := mobHolder(foe)
			a.holder.say(fmt.Sprintf("%s climbs back up, and you press the advantage.", target.tag()),
				verbatim(target.tag())+" climbs back up, and %s presses the advantage.", fmt.Sprintf(" (relentless: +%d action meter)", pts))
		}
	}
	if len(keep) == 0 {
		delete(warlordDown, a.who)
		return
	}
	warlordDown[a.who] = keep
}

// warlordsCommand is the capstone: when an ally falls, every standing ally
// is quickened, once a battle.
func warlordsCommand(a actor, side []actor, points int) {
	for _, t := range side {
		if t.char.Health >= 1 && summonInfo(t.char) == nil {
			pushMeter(t.who, points)
		}
	}
	a.holder.say("You bark an order, and the company closes up around the fallen.",
		"%s barks an order, and the company closes up around the fallen.", fmt.Sprintf(" (warlord's command: +%d action meter for everyone standing)", points))
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Status: "Warlord's Command", Outcome: combatstream.OutcomeSucceeded})
}

// battleCry is the Attack every ally has from a standing Warlord for the
// battle's first two rounds; it is derived each round, never stored.
func battleCry(side []actor, b battle.Battle, round uint64) (points int, cryer *actor) {
	if round < b.StartRound || round-b.StartRound >= 2 {
		return 0, nil
	}
	for i := range side {
		a := side[i]
		if a.char.Health < 1 || summonInfo(a.char) != nil {
			continue
		}
		if n := a.char.ClassEffects().Int(classes.BattleCry); n > points {
			points, cryer = n, &side[i]
		}
	}
	return points, cryer
}

// announceCry tells the room of the Battle Cry on the battle's first round.
func announceCry(cryer *actor, points int, b battle.Battle, round uint64) {
	if cryer == nil || round != b.StartRound {
		return
	}
	cryer.holder.say("You roar a battle cry, and the company takes it up.",
		"%s roars a battle cry, and the company takes it up.", fmt.Sprintf(" (battle cry: +%d Attack for your allies, 2 rounds)", points))
	emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: cryer.char.RoomId, Source: cryer.ref, Status: "Battle Cry", Outcome: combatstream.OutcomeSucceeded})
}

// secondWindPass is the elite talent Second Wind: once a battle, a fighter
// with the talent that begins its turn below a quarter of its health heals a
// tenth of its maximum. The turn is still taken, so it adds no action.
func secondWindPass() {
	for _, uid := range battle.Players() {
		b, ok := battle.Current(uid)
		u := users.GetByUserId(uid)
		if !ok || u == nil || u.Character == nil || u.Character.RoomId != b.RoomId {
			continue
		}
		room := rooms.LoadRoom(b.RoomId)
		if room == nil {
			continue
		}
		for _, a := range sideActors(u, room) {
			fx := a.char.ClassEffects()
			pct := fx.Int(classes.SecondWind)
			if pct <= 0 || a.char.Health < 1 || surprised(a.who.userId, a.who.mobId) {
				continue
			}
			if tempoActive && tempoTurns[a.who] == 0 {
				continue
			}
			if ag := a.char.Aggro; ag != nil && ag.Type == characters.Retreat {
				continue
			}
			rt := a.char.RTState()
			if rt.WindUsed || a.char.Health*4 >= a.char.HealthMax.Value {
				continue
			}
			rt.WindUsed = true
			healed := a.char.ApplyHealthChange(max(1, a.char.HealthMax.Value*pct/100))
			if a.who.userId > 0 {
				events.AddToQueue(events.CharacterVitalsChanged{UserId: a.who.userId})
			}
			a.holder.say("You catch your second wind.", "%s catches their second wind.", fmt.Sprintf(" (second wind, %d healed)", healed))
			emitCombat(combatstream.Event{Kind: combatstream.Ability, RoomId: a.char.RoomId, Source: a.ref, Target: a.ref, Status: "Second Wind", Outcome: combatstream.OutcomeSucceeded})
		}
	}
}
