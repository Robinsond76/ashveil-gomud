package company

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30g1: the balance harness. An evenly matched 5v5 (the company
// against a mirror group with its kits) is fought through the real combat
// round with real dice, many times per cell, and the table of outcomes is
// the baseline every later 30g slice is measured against. TestBalance5v5
// runs only with ASHVEIL_BALANCE=1; TestBalanceHarnessRunsAFight keeps the
// harness honest in the ordinary suite.

// balanceMaxRounds is where a fight counts as a stall.
const balanceMaxRounds = 200

// balanceGroup is the mirror group's spawn group (non-hostile mobs group
// only by a spawn group, Phase 32d).
const balanceGroup = "brawl:mirror"

// Company modes (decision 14). Spread gives each member a different rule,
// so blows land across the enemy; default keeps the shipped strategies
// (each member aims at the weakest); focus is `tactics` weakest.
const (
	companySpread  = "spread"
	companyDefault = "default"
	companyFocus   = "focus"
)

// Enemy modes: spread re-aims at a random foe every time; default keeps
// the shipped human personality (the weakest, 10% random).
const (
	enemySpread  = "spread"
	enemyDefault = "default"
)

// balanceMirror is the enemy side: the company's kits on five humans.
var balanceMirror = []struct {
	id        int
	name      string
	equipment string
}{
	{9201, "hired blade", "    weapon:\n      itemid: 10015\n    offhand:\n      itemid: 20004\n"},                                                                // Tamsin
	{9202, "hedge priest", "    weapon:\n      itemid: 10015\n    body:\n      itemid: 20008\n"},                                                                  // Oswin
	{9203, "sellsword", "    weapon:\n      itemid: 10002\n    head:\n      itemid: 20020\n    legs:\n      itemid: 20029\n    feet:\n      itemid: 20003\n"}, // Garrick
	{9204, "poacher", "    weapon:\n      itemid: 10014\n    neck:\n      itemid: 20024\n    feet:\n      itemid: 20003\n"},                                     // Ysolde
	{9205, "brawler", ""}, // Aria, bare-handed
}

func mirrorMob(id int, name, equipment string) string {
	out := fmt.Sprintf("mobid: %d\nzone: brawl\nhostile: false\nmaxwander: 0\nactivitylevel: 0\nitemdropchance: 0\ngroups: [mirror]\ncharacter:\n  name: %s\n  raceid: 1\n  level: 1\n", id, name)
	if equipment != "" {
		out += "  equipment:\n" + equipment
	}
	return out
}

// balanceTally folds one fight's combat events by side.
type balanceTally struct {
	Turns, Hits, Crits, Misses [2]int
	Damage, Healing            [2]int
}

const (
	sideCompany = 0
	sideEnemy   = 1
)

// sideOf is the side a combatant fights on: the company is Aria (user 7)
// and anyone she leads; everyone else is the enemy.
func sideOf(r combatstream.Ref) int {
	if r.UserId == 7 || r.LeaderUserId == 7 {
		return sideCompany
	}
	return sideEnemy
}

func (t *balanceTally) add(e combatstream.Event) {
	s := sideOf(e.Source)
	switch e.Kind {
	case combatstream.Attack:
		t.Turns[s]++
		switch e.Outcome {
		case combatstream.OutcomeCrit:
			t.Hits[s]++
			t.Crits[s]++
		case combatstream.OutcomeHit:
			t.Hits[s]++
		default:
			t.Misses[s]++
		}
		t.Damage[s] += e.Damage
	case combatstream.SpellHit:
		t.Damage[s] += e.Damage
	case combatstream.Heal:
		t.Healing[s] += e.Amount
	}
}

// percentile is the nearest-rank p-th percentile (0–100) of values, 0 for
// none. values is sorted in place.
func percentile(values []int, p int) int {
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	rank := (p*len(values) + 99) / 100 // ceil(p/100 × n)
	if rank < 1 {
		rank = 1
	}
	return values[rank-1]
}

// balanceResult is one fight's outcome.
type balanceResult struct {
	Rounds       int
	Won, Stalled bool
	Fallen       [2]int
	// FighterRounds is, per side, the sum over rounds of the fighters
	// standing at the round's start: turns per fighter per round divides
	// by it.
	FighterRounds [2]int
	Tally         balanceTally
}

// balanceFight is a brawl set up for one balance fight.
type balanceFight struct {
	*brawl
	enemies []int
	tally   *balanceTally
}

// newBalanceFight is a brawl world with the bandits gone, the mirror group
// in the road, both sides at level, the modes set, real dice, and the
// fight begun.
func newBalanceFight(t *testing.T, level int, companyMode, enemyMode string) *balanceFight {
	t.Helper()
	b := newBrawl(t)
	b.withArchetypes("")
	// Real dice, not the brawl tests' pinned ones.
	t.Cleanup(hooks.UseAimRollForTest(util.Rand))
	t.Cleanup(hooks.UseCounterRollForTest(util.Rand))
	t.Cleanup(hooks.UseBreakRollForTest(util.Rand))

	// No bandits: the mirror group is the only foe.
	for _, ids := range b.bandits {
		for _, id := range ids {
			b.road.RemoveMob(id)
			mobs.DestroyInstance(id)
		}
	}
	b.bandits = map[string][]int{}

	dataDir := configs.GetFilePathsConfig().DataFiles.String()
	for _, m := range balanceMirror {
		path := filepath.Join(dataDir, "mobs", "brawl", fmt.Sprintf("%d-%s.yaml", m.id, strings.ReplaceAll(m.name, " ", "_")))
		require.NoError(t, os.WriteFile(path, []byte(mirrorMob(m.id, m.name, m.equipment)), 0600))
	}
	mobs.LoadDataFiles()

	f := &balanceFight{brawl: b, tally: &balanceTally{}}
	for _, m := range balanceMirror {
		mob := mobs.NewMobById(mobs.MobId(m.id), b.road.RoomId)
		require.NotNil(t, mob, m.name)
		mob.SpawnGroup = balanceGroup
		b.road.AddMob(mob.InstanceId)
		f.enemies = append(f.enemies, mob.InstanceId)
		levelTo(&mob.Character, level)
		switch enemyMode {
		case enemySpread:
			mob.Targeting, mob.TargetingNoise = string(strategy.Nearest), 100
		case enemyDefault:
		default:
			t.Fatalf("enemy mode %q", enemyMode)
		}
	}
	levelTo(b.aria.Character, level)
	for id := 1; id <= 4; id++ {
		levelTo(&b.companion(id).Character, level)
	}

	switch companyMode {
	case companySpread:
		for _, s := range []string{"tamsin nearest", "garrick strongest", "ysolde furthest", "oswin wounded"} {
			b.cmd("strategy", s)
		}
	case companyDefault:
	case companyFocus:
		b.saveTactics(strategy.Tactics{Focus: strategy.Weakest})
	default:
		t.Fatalf("company mode %q", companyMode)
	}

	s := combatstream.New()
	s.Subscribe(func(e combatstream.Event) { f.tally.add(e) })
	t.Cleanup(combatstream.UseForTest(s))

	b.cmd("attack", fmt.Sprintf("#%d", f.enemies[0]))
	return f
}

// levelTo sets a character to level the way a spawning mob is set
// (mobs.NewMobById): training cleared, the level's stat points trained
// automatically, then full health and mana.
func levelTo(c *characters.Character, level int) {
	c.Level = level
	for _, s := range []*int{&c.Stats.Strength.Training, &c.Stats.Speed.Training, &c.Stats.Smarts.Training,
		&c.Stats.Vitality.Training, &c.Stats.Mysticism.Training, &c.Stats.Perception.Training} {
		*s = 0
	}
	cfg := configs.GetProgressionConfig()
	c.StatPoints = 0
	for lvl := 1; lvl <= level; lvl++ {
		if int(cfg.StatPointsEveryNLevels) <= 1 || lvl%int(cfg.StatPointsEveryNLevels) == 0 {
			c.StatPoints += int(cfg.StatPointsPerLevel)
		}
	}
	c.AutoTrain()
	c.Health = c.HealthMax.Value
	c.Mana = c.ManaMax.Value
}

// standing counts each side's fighters still up.
func (f *balanceFight) standing() (company, enemy int) {
	if f.aria.Character.Health >= 1 {
		company++
	}
	for id := 1; id <= 4; id++ {
		if instance, ok := module.instance(7, id); ok {
			if m := mobs.GetInstance(instance); m != nil && m.Character.Health >= 1 {
				company++
			}
		}
	}
	for _, id := range f.enemies {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health >= 1 && m.Character.RoomId == f.road.RoomId {
			enemy++
		}
	}
	return company, enemy
}

// run fights until one side is down or the fight stalls. Each combat
// round is two game rounds (29f), so regeneration's pass runs for both.
// Aria is held at 0 once she falls: fallen and healable, never dead (a
// player's death would cost her a level and move her to a church).
func (f *balanceFight) run() balanceResult {
	var res balanceResult
	for res.Rounds < balanceMaxRounds {
		company, enemy := f.standing()
		if company == 0 || enemy == 0 {
			break
		}
		res.FighterRounds[sideCompany] += company
		res.FighterRounds[sideEnemy] += enemy
		res.Rounds++
		f.round++
		*f.messages = nil
		hooks.DoCombat(events.NewRound{RoundNumber: f.round})
		events.ProcessEvents()
		hooks.IdleMobs(events.NewRound{RoundNumber: f.round})
		events.ProcessEvents()
		for _, game := range []uint64{2*f.round - 1, 2 * f.round} {
			hooks.AutoHeal(events.NewRound{RoundNumber: game})
			events.ProcessEvents()
		}
		if f.aria.Character.Health < 0 {
			f.aria.Character.Health = 0
		}
	}
	company, enemy := f.standing()
	res.Won = enemy == 0 && company > 0
	res.Stalled = company > 0 && enemy > 0
	res.Fallen = [2]int{5 - company, len(f.enemies) - enemy}
	res.Tally = *f.tally
	return res
}

func TestBalancePercentile(t *testing.T) {
	assert.Equal(t, 0, percentile(nil, 50))
	assert.Equal(t, 7, percentile([]int{7}, 10))
	assert.Equal(t, 3, percentile([]int{5, 1, 3}, 50), "odd: the middle")
	assert.Equal(t, 2, percentile([]int{4, 1, 3, 2}, 50), "even: the lower middle")
	assert.Equal(t, 1, percentile([]int{4, 1, 3, 2}, 10))
	assert.Equal(t, 4, percentile([]int{4, 1, 3, 2}, 90))
}

func TestBalanceTally(t *testing.T) {
	aria := combatstream.Ref{UserId: 7, LeaderUserId: 7}
	tamsin := combatstream.Ref{MobInstanceId: 11, LeaderUserId: 7}
	blade := combatstream.Ref{MobInstanceId: 21}
	var tl balanceTally
	for _, e := range []combatstream.Event{
		{Kind: combatstream.Attack, Source: aria, Target: blade, Outcome: combatstream.OutcomeHit, Damage: 4},
		{Kind: combatstream.Attack, Source: tamsin, Target: blade, Outcome: combatstream.OutcomeCrit, Damage: 9},
		{Kind: combatstream.Attack, Source: blade, Target: aria, Outcome: combatstream.OutcomeMiss},
		{Kind: combatstream.SpellHit, Source: blade, Target: tamsin, Damage: 3},
		{Kind: combatstream.Heal, Source: tamsin, Target: aria, Amount: 5},
		{Kind: combatstream.TargetChange, Source: blade, Target: tamsin},
	} {
		tl.add(e)
	}
	assert.Equal(t, [2]int{2, 1}, tl.Turns)
	assert.Equal(t, [2]int{2, 0}, tl.Hits)
	assert.Equal(t, [2]int{1, 0}, tl.Crits)
	assert.Equal(t, [2]int{0, 1}, tl.Misses)
	assert.Equal(t, [2]int{13, 3}, tl.Damage)
	assert.Equal(t, [2]int{5, 0}, tl.Healing)
}

// TestBalanceHarnessRunsAFight keeps the harness working in the ordinary
// suite: one level-1 fight through the real round ends, both sides took
// turns, and the leader was held at no less than 0.
func TestBalanceHarnessRunsAFight(t *testing.T) {
	f := newBalanceFight(t, 1, companyDefault, enemySpread)
	turn, round := util.GetTurnCount(), util.GetRoundCount()
	res := f.run()
	assert.Positive(t, res.Rounds)
	assert.Positive(t, res.Tally.Turns[sideCompany], "the company fought")
	assert.Positive(t, res.Tally.Turns[sideEnemy], "the mirror group fought")
	assert.GreaterOrEqual(t, f.aria.Character.Health, 0, "the leader never dies")
	assert.True(t, res.Won || res.Fallen[sideCompany] == 5 || res.Stalled)
	assert.Equal(t, turn, util.GetTurnCount(), "the world clock never moves")
	assert.Equal(t, round, util.GetRoundCount())
}

// TestBalance5v5 is the table: every level and pair of modes, many fights
// each. ASHVEIL_BALANCE=1 runs it; ASHVEIL_BALANCE_FIGHTS sets the fights
// per cell (default 50).
func TestBalance5v5(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the balance table")
	}
	fights := 50
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	var rows []string
	for _, level := range []int{1, 5, 10} {
		for _, cm := range []string{companySpread, companyDefault, companyFocus} {
			for _, em := range []string{enemySpread, enemyDefault} {
				var results []balanceResult
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("L%d-%s-%s-%d", level, cm, em, i), func(t *testing.T) {
						results = append(results, newBalanceFight(t, level, cm, em).run())
					})
				}
				rows = append(rows, balanceRow(level, cm, em, results))
			}
		}
	}
	t.Logf("\n%s\n%s", balanceHeader, strings.Join(rows, "\n"))
}

const balanceHeader = "| level | company | enemy | fights | company wins | rounds p10/median/p90 | stalls | fallen company/enemy | damage company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|"

// balanceRow is one cell's line of the table: averages are per fight.
func balanceRow(level int, cm, em string, results []balanceResult) string {
	n := len(results)
	if n == 0 {
		return fmt.Sprintf("| %d | %s | %s | 0 | | | | | | | | | |", level, cm, em)
	}
	var rounds []int
	var wins, stalls int
	var fallen, damage, turns, hits, crits, fighterRounds [2]int
	var healing int
	for _, r := range results {
		rounds = append(rounds, r.Rounds)
		if r.Won {
			wins++
		}
		if r.Stalled {
			stalls++
		}
		for s := 0; s < 2; s++ {
			fallen[s] += r.Fallen[s]
			damage[s] += r.Tally.Damage[s]
			turns[s] += r.Tally.Turns[s]
			hits[s] += r.Tally.Hits[s]
			crits[s] += r.Tally.Crits[s]
			fighterRounds[s] += r.FighterRounds[s]
		}
		healing += r.Tally.Healing[sideCompany]
	}
	per := func(v int) float64 { return float64(v) / float64(n) }
	ratio := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / float64(b)
	}
	p10, p50, p90 := percentile(rounds, 10), percentile(rounds, 50), percentile(rounds, 90)
	return fmt.Sprintf("| %d | %s | %s | %d | %.0f%% | %d/%d/%d | %d | %.1f/%.1f | %.0f/%.0f | %.0f | %.2f/%.2f | %.0f/%.0f | %.0f/%.0f |",
		level, cm, em, n, 100*ratio(wins, n), p10, p50, p90, stalls,
		per(fallen[0]), per(fallen[1]), per(damage[0]), per(damage[1]), per(healing),
		ratio(turns[0], fighterRounds[0]), ratio(turns[1], fighterRounds[1]),
		100*ratio(hits[0], turns[0]), 100*ratio(hits[1], turns[1]),
		100*ratio(crits[0], hits[0]), 100*ratio(crits[1], hits[1]))
}
