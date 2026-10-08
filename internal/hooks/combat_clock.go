package hooks

import (
	"sync/atomic"
	"time"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/combatpace"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ashveil Phase 82c: the battle clock.
//
// While a player is in a fight, combat rounds leave the fixed
// CombatEveryRounds cadence: BattleClock, on every turn, resolves the next
// round once the last one's lines have all gone out (plus the pace's tail),
// and never sooner than MinRoundMs after the last round began. Each round
// still resolves at once under the game lock, exactly as DoCombat always
// has (the turn order of 82b is the sequence the lines play back in), so
// the fight is never faster than one action per beat and no round is
// squeezed into a window. Fights with no player in them (mob against mob)
// keep the cadence while no player fights; while the clock runs it resolves
// every fight, since DoCombat is one pass over the world.
//
// There is one clock for the world, not one per battle cluster (an
// amendment to decision 4 of the design, recorded in the plan): DoCombat's
// thirty-odd passes and its round-keyed state (marks, cries, openings, the
// fight stream) share one round number, so per-cluster rounds would mean
// per-cluster numbering through all of them. Two companies fighting apart
// at the same time share the clock: each round waits for the slower
// playback of the two.
//
// Rounds are numbered by a counter of combat rounds, not by the game round
// (which the cadence used to pass): a fight's rounds count 1, 2, 3 on the
// battle screen, and the clock's rounds, which fall between game rounds,
// have a number of their own. The counter is game-loop state and starts
// over with the process, as the battles themselves do.

type battleClock struct {
	active  bool
	pending bool // a round resolved; its due time is set on the next turn, once its lines are held
	roundAt time.Time
	due     time.Time
	spec    combatpace.Spec // the slowest pace of the last round's players
}

var (
	clock              battleClock
	combatRoundCounter atomic.Uint64
)

// nextCombatRound numbers the next combat round.
func nextCombatRound() uint64 { return combatRoundCounter.Add(1) }

// SetCombatRoundCounterForTest makes n the last combat round's number.
func SetCombatRoundCounterForTest(n uint64) { combatRoundCounter.Store(n) }

// ResetBattleClockForTest forgets the clock and the round counter.
func ResetBattleClockForTest() {
	clock = battleClock{}
	combatRoundCounter.Store(0)
}

// BattleClockDue is when the next clock round resolves, for tests and the
// web client's round timer; ok is false while no player fights.
func BattleClockDue() (due time.Time, ok bool) {
	return clock.due, clock.active && !clock.pending
}

// playerFightLive reports whether any player is in a fight: in a battle, or
// going for someone, or gone for by a mob (them or a companion of theirs).
func playerFightLive() bool {
	if len(battle.Players()) > 0 {
		return true
	}
	for _, id := range users.GetOnlineUserIds() {
		if u := users.GetByUserId(id); u != nil && u.Character != nil && u.Character.Aggro != nil {
			return true
		}
	}
	for _, id := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(id)
		if m == nil || m.Character.Aggro == nil {
			continue
		}
		if m.Character.Aggro.UserId > 0 {
			return true
		}
		if _, _, companion := company.LeaderAndKeyForInstance(m.Character.Aggro.MobInstanceId); companion {
			return true
		}
	}
	return false
}

// BattleClock is a NewTurn listener: while a player fights, it resolves a
// combat round whenever the last one's lines are out and the minimum round
// has passed.
func BattleClock(e events.Event) events.ListenerReturn {
	if _, ok := e.(events.NewTurn); !ok {
		return events.Continue
	}
	now := paceNow()
	if !playerFightLive() {
		clock = battleClock{}
		return events.Continue
	}
	if !clock.active {
		clock = battleClock{active: true, due: now}
	}
	if clock.pending {
		clock.due = clock.dueAfter()
		clock.pending = false
	}
	if now.Before(clock.due) {
		return events.Continue
	}
	clock.spec = slowestPaceNearFights().Beats()
	resolveCombatRound(DoCombat)
	clock.roundAt, clock.pending = now, true
	return events.Continue
}

// dueAfter is when the next round is due: the minimum round after the
// last began, or the slowest player's playback plus the tail, whichever
// is later.
func (c battleClock) dueAfter() time.Time {
	due := c.roundAt.Add(time.Duration(configs.GetCombatConfig().MinRoundMs) * time.Millisecond)
	if end := combatpace.Default().PlaybackEnd(); !end.IsZero() {
		if t := end.Add(c.spec.Tail); t.After(due) {
			due = t
		}
	}
	return due
}

// slowestPaceNearFights is the slowest pace of the players in or beside a
// fight (off counts as normal), whose tail the clock waits after playback.
func slowestPaceNearFights() combatpace.Pace {
	slowest := combatpace.Normal
	for _, id := range users.GetOnlineUserIds() {
		if u := users.GetByUserId(id); u != nil && u.Character != nil && nearAFight(u) {
			slowest = combatpace.Slowest(slowest, paceOf(u))
		}
	}
	return slowest
}
