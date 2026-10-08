package hooks

import (
	"sort"
	"sync/atomic"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Phase 82b: speed-ordered turns. A combat round no longer runs every player
// and then every mob; it lines every fighter up by tempo and runs their
// turns one at a time, fastest first. A fighter with n turns this round
// (the action meter, combat.Meter) gets a slot at k / tempo for k = 1..n,
// so a fast fighter's second turn comes after most first turns and before
// the slowest fighters' first. In a fight's first round the opening meter
// a fighter starts with (a Samurai's Iaijutsu, Scouted ground, a Packlord's
// hound) moves its first slot earlier: (1 - bonus/100) / tempo. Ties break
// by raw Speed, then Perception, then a per-round roll (UseTurnOrderRollForTest
// pins it).
//
// Retreats are resolved before the order (as the leader's loop ran first
// before this phase), so a leader's tempo never changes when a retreat
// lands. Everything else a turn does is unchanged: actPlayer and actMob are
// the old loop bodies.

// turnSlot is one fighter's place in the round.
type turnSlot struct {
	who   caster
	k     int     // which of the fighter's turns this round: 1 or 2
	at    float64 // k / tempo, the sort key
	speed int     // tie-breakers: raw Speed, then Perception, then a roll
	perc  int
	roll  float64
}

// TurnRef names a fighter in the round's order for the web client's battle
// data (Company.Battle.order): a player, or a mob instance (a companion or a
// foe), and the slot index from 1.
type TurnRef struct {
	UserId        int
	MobInstanceId int
	Slot          int
}

var (
	// turnOrderRoll breaks ties a per-round roll; tests replace it.
	turnOrderRoll = func() float64 { return float64(util.Rand(1000)) / 1000 }
	// roundOrder is this round's order, kept for RoundTurnOrder.
	roundOrder []turnSlot
	// currentSlot is the slot now resolving (0 outside the order: upkeep,
	// retreats, the round's end); emitCombat stamps it on every event.
	currentSlot atomic.Int32
)

// UseTurnOrderRollForTest pins the tie-breaking roll and returns a restore func.
func UseTurnOrderRollForTest(roll func() float64) (restore func()) {
	prev := turnOrderRoll
	turnOrderRoll = roll
	return func() { turnOrderRoll = prev }
}

// CurrentTurnSlot is the slot of the fighter now acting, 0 between turns.
func CurrentTurnSlot() int { return int(currentSlot.Load()) }

// turnTempo is a fighter's sort tempo: the real rate, as the meter fills.
func turnTempo(c *characters.Character) float64 {
	t := tempoRate(c)
	if t <= 0 {
		t = 0.01
	}
	return t
}

// buildTurnOrder lists every fighter in a battle this round (players with
// an aim, mobs with an aim: companions and foes alike) with its slots, and
// sorts them. Fighters who have fallen keep a slot: their turn only reports
// them for the round's end, as the old loops did.
func buildTurnOrder(round uint64) []turnSlot {
	var slots []turnSlot
	add := func(who caster, c *characters.Character) {
		turns := max(1, tempoTurns[who])
		tempo := turnTempo(c)
		bonus := 0.0
		if st := tempoMeters[who]; st != nil && st.opened == round {
			bonus = st.meter.Bonus
		}
		for k := 1; k <= turns; k++ {
			at := float64(k) / tempo
			if k == 1 && bonus > 0 {
				at = max(0, 1-bonus/100) / tempo
			}
			slots = append(slots, turnSlot{who: who, k: k, at: at,
				speed: c.Stats.Speed.ValueAdj, perc: c.Stats.Perception.ValueAdj, roll: turnOrderRoll()})
		}
	}
	// Fighters are visited in id order, so the tie-breaking rolls are drawn
	// the same way from one round to the next (the maps' order is not).
	userIds := users.GetOnlineUserIds()
	sort.Ints(userIds)
	for _, userId := range userIds {
		user := users.GetByUserId(userId)
		if user == nil || user.Character == nil || user.Character.Aggro == nil {
			continue
		}
		if user.Character.Aggro.Type == characters.Retreat {
			continue // resolved before the order (retreatPass)
		}
		add(caster{userId: userId}, user.Character)
	}
	mobIds := mobs.GetAllMobInstanceIds()
	sort.Ints(mobIds)
	for _, mobId := range mobIds {
		mob := mobs.GetInstance(mobId)
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}
		add(caster{mobId: mobId}, &mob.Character)
	}
	sort.SliceStable(slots, func(i, j int) bool {
		a, b := slots[i], slots[j]
		if a.at != b.at {
			return a.at < b.at
		}
		if a.speed != b.speed {
			return a.speed > b.speed
		}
		if a.perc != b.perc {
			return a.perc > b.perc
		}
		return a.roll < b.roll
	})
	return slots
}

// retreatPass resolves every retreating player before the order: a retreat
// is set any time and lands at the round's start, whatever the leader's
// tempo (the design's upkeep rule).
func retreatPass(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil || user.Character == nil || user.Character.Aggro == nil || user.Character.Aggro.Type != characters.Retreat {
			continue
		}
		p, m := actPlayer(evt, userId, false)
		affectedPlayerIds = append(affectedPlayerIds, p...)
		affectedMobInstanceIds = append(affectedMobInstanceIds, m...)
	}
	return affectedPlayerIds, affectedMobInstanceIds
}

// runTurnOrder resolves the round's turns one fighter at a time, in order.
// After a fighter's first turn its readied strike, shadowstep and leap end
// (they were ended between the old first and second passes); the ranger's
// held Overwatch shots loose after the last first turn, as they did after
// the first pass.
func runTurnOrder(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {
	p, m := retreatPass(evt)
	affectedPlayerIds = append(affectedPlayerIds, p...)
	affectedMobInstanceIds = append(affectedMobInstanceIds, m...)

	roundOrder = buildTurnOrder(evt.RoundNumber)
	lastFirst := -1
	for i, s := range roundOrder {
		if s.k == 1 {
			lastFirst = i
		}
	}
	defer func() { currentSlot.Store(0); events.SetSlot(0) }()
	for i, s := range roundOrder {
		currentSlot.Store(int32(i + 1))
		// Phase 30d2: the last mob's wind-up blow, if its swing never
		// happened, is told as wasted. Turns of both sides interleave
		// now, so it is closed before the next turn whoever takes it,
		// and the wasted line still follows the blow it belongs to.
		finishLanding()
		events.SetSlot(i + 1) // Phase 82c: the turn's lines are paced as one beat
		extra := s.k > 1
		if s.who.userId > 0 {
			p, m = actPlayer(evt, s.who.userId, extra)
		} else {
			p, m = actMob(evt, s.who.mobId, extra)
		}
		affectedPlayerIds = append(affectedPlayerIds, p...)
		affectedMobInstanceIds = append(affectedMobInstanceIds, m...)
		if !extra {
			endAbilityStrike(s.who)
			endShadowstep(s.who)
			if s.who.mobId > 0 {
				delete(battlefieldPowers, s.who.mobId)
			}
		}
		if i == lastFirst {
			looseHeldShots() // Phase 38c2 review: a quiet Overwatch hold still shoots
		}
	}
	currentSlot.Store(0)
	events.SetSlot(0)
	finishLanding() // Phase 30d2
	if lastFirst < 0 {
		looseHeldShots()
	}
	endAbilityStrikes()
	endShadowsteps()
	clear(battlefieldPowers)
	return affectedPlayerIds, affectedMobInstanceIds
}

// RoundTurnOrder is the latest round's order as the web client's battle
// data lists it: every fighter in the same fights as the player, in the
// order they acted. Empty outside a round or when the player is not in one.
func RoundTurnOrder(userId int) []TurnRef {
	fights := tempoMembership(caster{userId: userId})
	if len(fights) == 0 {
		return nil
	}
	var out []TurnRef
	for i, s := range roundOrder {
		if !sharesTempoFight(fights, tempoMembership(s.who)) {
			continue
		}
		out = append(out, TurnRef{UserId: s.who.userId, MobInstanceId: s.who.mobId, Slot: i + 1})
	}
	return out
}
