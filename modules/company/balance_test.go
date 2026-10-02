package company

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/status"
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

// Enemy modes: spread makes each new aim a random foe (an aim sticks until
// its target falls or can't be reached); default keeps the shipped human
// personality (the weakest, 10% random).
const (
	enemySpread  = "spread"
	enemyDefault = "default"
	// Phase 33i2: the default personality, with the hedge priest a
	// healer (Minor Heal) and the hired blade a guardian, at a tier
	// (enemyRoles + the tier's number: "roles1" to "roles4").
	enemyRoles = "roles"
)

// balanceMirror is the enemy side: the company's kits on five humans.
var balanceMirror = []struct {
	id        int
	name      string
	equipment string
}{
	{9201, "hired blade", "    weapon:\n      itemid: 10015\n    offhand:\n      itemid: 20004\n"},                                                            // Tamsin
	{9202, "hedge priest", "    weapon:\n      itemid: 10015\n    body:\n      itemid: 20008\n"},                                                              // Oswin
	{9203, "sellsword", "    weapon:\n      itemid: 10002\n    head:\n      itemid: 20020\n    legs:\n      itemid: 20029\n    feet:\n      itemid: 20003\n"}, // Garrick
	{9204, "poacher", "    weapon:\n      itemid: 10014\n    neck:\n      itemid: 20024\n    feet:\n      itemid: 20003\n"},                                   // Ysolde
	{9205, "brawler", ""}, // Aria, bare-handed
}

func mirrorMob(id int, name, equipment string) string {
	out := fmt.Sprintf("mobid: %d\nzone: brawl\nhostile: false\nmaxwander: 0\nactivitylevel: 0\nitemdropchance: 0\ngroups: [mirror]\ncharacter:\n  name: %s\n  raceid: 1\n  level: 1\n", id, name)
	if equipment != "" {
		out += "  equipment:\n" + equipment
	}
	return out
}

// balanceTally folds one fight's combat events by side. A turn is one
// attacker's round of blows (one Attack event); a hit is a turn whose blow
// passed the to-hit roll and wasn't dodged (armor may still take all of
// it). Damage includes overkill. Shield bashes (30d1) are counters, not
// turns; status ticks count for the side that didn't take them. Blocks,
// parries, and dodges (30g2) count for the side that made them, one per
// strike stopped.
type balanceTally struct {
	Turns, Hits, Crits, Misses [2]int
	Blocks, Parries, Dodges    [2]int
	Counters                   [2]int
	Damage, TickDamage         [2]int
	Healing                    [2]int
}

// shieldBash is the weapon type a counter's Attack event carries.
const shieldBash = "shield-bash"

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
		if e.WeaponType == shieldBash {
			t.Counters[s]++
			t.Damage[s] += e.Damage
			return
		}
		t.Turns[s]++
		for _, d := range e.Defenses {
			switch d {
			case combat.DefenseBlocked:
				t.Blocks[1-s]++
			case combat.DefenseParried:
				t.Parries[1-s]++
			case combat.DefenseDodged:
				t.Dodges[1-s]++
			}
		}
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
	case combatstream.StatusTick:
		// A tick has no source: it counts for the other side.
		other := 1 - sideOf(e.Target)
		t.Damage[other] += e.Damage
		t.TickDamage[other] += e.Damage
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
	// 30a's statuses, which the brawl world leaves out (review: no bleed
	// or stagger ever landed, so tick damage read 0).
	loadStatusBuffs(t)
	// And the listener that puts a blow's statuses on its target, as the
	// game registers it (hooks.RegisterListeners).
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
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
		switch {
		case enemyMode == enemySpread:
			mob.Targeting, mob.TargetingNoise = string(strategy.Nearest), 100
		case enemyMode == enemyDefault:
		case strings.HasPrefix(enemyMode, enemyRoles):
			tier, err := strconv.Atoi(strings.TrimPrefix(enemyMode, enemyRoles))
			require.NoError(t, err, enemyMode)
			mob.Coordination = tier
			switch m.id {
			case 9201:
				mob.Role = "guardian"
			case 9202:
				mob.Role = "healer"
				mob.Character.SpellBook["heal"] = 250
			}
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
			require.Contains(t, b.cmd("strategy", s), "will go for", s)
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
// automatically, then full health and mana. Experience is the start of the
// level (0 at level 1), so a template's experience can't level anyone on
// their first kill (review: Garrick and Ysolde did at level 1).
func levelTo(c *characters.Character, level int) {
	c.Level = level
	c.PeakLevel = level
	c.Experience = 0
	if level > 1 {
		c.Experience = c.XPTL(level - 1)
	}
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
// round is two game rounds (29f), so regeneration's pass runs for both,
// as the game does (it skips anyone in a battle, and tops up companions'
// mana every third round).
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
		f.step()
	}
	company, enemy := f.standing()
	res.Won = enemy == 0 && company > 0
	res.Stalled = company > 0 && enemy > 0
	res.Fallen = [2]int{5 - company, len(f.enemies) - enemy}
	res.Tally = *f.tally
	return res
}

// step runs one combat round: the round, the idle-mob pass, and
// regeneration for its two game rounds; then Aria is held at 0.
func (f *balanceFight) step() {
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
		{Kind: combatstream.Attack, Source: blade, Target: aria, Outcome: combatstream.OutcomeMiss, Defenses: []string{combat.DefenseParried}},
		{Kind: combatstream.Attack, Source: aria, Target: blade, Outcome: combatstream.OutcomeMiss, Defenses: []string{combat.DefenseBlocked, combat.DefenseDodged}},
		{Kind: combatstream.SpellHit, Source: blade, Target: tamsin, Damage: 3},
		{Kind: combatstream.Heal, Source: tamsin, Target: aria, Amount: 5},
		{Kind: combatstream.TargetChange, Source: blade, Target: tamsin},
		{Kind: combatstream.Attack, Source: tamsin, Target: blade, Outcome: combatstream.OutcomeHit, Damage: 2, WeaponType: shieldBash},
		{Kind: combatstream.StatusTick, Target: blade, Damage: 1},
		{Kind: combatstream.StatusTick, Target: aria, Outcome: combatstream.OutcomeLostAction},
	} {
		tl.add(e)
	}
	assert.Equal(t, [2]int{3, 1}, tl.Turns, "a bash is no turn")
	assert.Equal(t, [2]int{0, 1}, tl.Blocks, "the blade blocked")
	assert.Equal(t, [2]int{0, 1}, tl.Dodges)
	assert.Equal(t, [2]int{1, 0}, tl.Parries, "Aria parried")
	assert.Equal(t, [2]int{1, 0}, tl.Counters)
	assert.Equal(t, [2]int{1, 0}, tl.TickDamage, "the blade's bleeding counts for the company")
	assert.Equal(t, [2]int{2, 0}, tl.Hits)
	assert.Equal(t, [2]int{1, 0}, tl.Crits)
	assert.Equal(t, [2]int{1, 1}, tl.Misses)
	assert.Equal(t, [2]int{16, 3}, tl.Damage, "blows, the bash, and the tick")
	assert.Equal(t, [2]int{5, 0}, tl.Healing)
}

// TestBalanceHarnessRunsAFight keeps the harness working in the ordinary
// suite: one level-1 fight through the real round ends, both sides took
// turns, and the leader was held at no less than 0.
func TestBalanceHarnessRunsAFight(t *testing.T) {
	f := newBalanceFight(t, 1, companyDefault, enemySpread)
	require.NotNil(t, buffs.GetBuffSpec(status.Bleeding), "30a's statuses are loaded")
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

// TestBalanceHarnessRunsACoordinatedFight (Phase 33i2): a level-5 fight
// against a band with a healer and a guardian runs through the real round
// in the ordinary suite.
func TestBalanceHarnessRunsACoordinatedFight(t *testing.T) {
	f := newBalanceFight(t, 5, companyDefault, enemyRoles+"2")
	res := f.run()
	assert.Positive(t, res.Rounds)
	assert.Positive(t, res.Tally.Turns[sideEnemy], "the band fought")
	assert.True(t, res.Won || res.Fallen[sideCompany] == 5 || res.Stalled)
}

// TestBalanceCoordinated (Phase 33i2) measures coordinated groups against
// the original no-focus baseline (default company, default enemy): at each
// level, a group with a healer and a guardian at tiers 1 to 3 may lengthen
// the median battle by at most a quarter, and must not flip a clear
// winner (one winning 60% or more keeps winning at least half the time).
// ASHVEIL_BALANCE=1 runs it; ASHVEIL_BALANCE_FIGHTS sets the fights per
// cell (default 50).
func TestBalanceCoordinated(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the balance table")
	}
	fights := 50
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	var rows []string
	for _, level := range []int{1, 5, 10} {
		cell := func(em string) []balanceResult {
			var results []balanceResult
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("L%d-%s-%d", level, em, i), func(t *testing.T) {
					results = append(results, newBalanceFight(t, level, companyDefault, em).run())
				})
			}
			rows = append(rows, balanceRow(level, companyDefault, em, results))
			return results
		}
		base := cell(enemyDefault)
		baseMedian, baseWins := balanceMedianAndWins(base)
		for tier := 1; tier <= 3; tier++ {
			median, wins := balanceMedianAndWins(cell(fmt.Sprintf("%s%d", enemyRoles, tier)))
			assert.LessOrEqual(t, median*4, baseMedian*5, "L%d tier %d: median %d against %d", level, tier, median, baseMedian)
			if baseWins >= 60 {
				assert.GreaterOrEqual(t, wins, 50, "L%d tier %d: the company kept winning", level, tier)
			}
			if baseWins <= 40 {
				assert.LessOrEqual(t, wins, 50, "L%d tier %d: the enemy kept winning", level, tier)
			}
		}
	}
	t.Logf("\n%s\n%s", balanceHeader, strings.Join(rows, "\n"))
}

// balanceMedianAndWins is a cell's median rounds and company win percent.
func balanceMedianAndWins(results []balanceResult) (median, winPct int) {
	var rounds []int
	wins := 0
	for _, r := range results {
		rounds = append(rounds, r.Rounds)
		if r.Won {
			wins++
		}
	}
	if len(results) == 0 {
		return 0, 0
	}
	return percentile(rounds, 50), 100 * wins / len(results)
}

const balanceHeader = "| level | company | enemy | fights | company wins | rounds p10/median/p90 | stalls | fallen company/enemy | damage company/enemy | healing company | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"

// balanceRow is one cell's line of the table: averages are per fight.
func balanceRow(level int, cm, em string, results []balanceResult) string {
	n := len(results)
	if n == 0 {
		return fmt.Sprintf("| %d | %s | %s | 0 | | | | | | | | | | | | |", level, cm, em)
	}
	var rounds []int
	var wins, stalls int
	var fallen, damage, turns, hits, crits, fighterRounds, counters, ticks, blocks, parries, dodges [2]int
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
			counters[s] += r.Tally.Counters[s]
			blocks[s] += r.Tally.Blocks[s]
			parries[s] += r.Tally.Parries[s]
			dodges[s] += r.Tally.Dodges[s]
			ticks[s] += r.Tally.TickDamage[s]
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
	return fmt.Sprintf("| %d | %s | %s | %d | %.0f%% | %d/%d/%d | %d | %.1f/%.1f | %.0f/%.0f | %.0f | %.2f/%.2f | %.0f/%.0f | %.0f/%.0f | %.1f/%.1f/%.1f · %.1f/%.1f/%.1f | %.1f/%.1f | %.1f/%.1f |",
		level, cm, em, n, 100*ratio(wins, n), p10, p50, p90, stalls,
		per(fallen[0]), per(fallen[1]), per(damage[0]), per(damage[1]), per(healing),
		ratio(turns[0], fighterRounds[0]), ratio(turns[1], fighterRounds[1]),
		100*ratio(hits[0], turns[0]), 100*ratio(hits[1], turns[1]),
		100*ratio(crits[0], hits[0]), 100*ratio(crits[1], hits[1]),
		per(blocks[0]), per(parries[0]), per(dodges[0]), per(blocks[1]), per(parries[1]), per(dodges[1]),
		per(counters[0]), per(counters[1]), per(ticks[0]), per(ticks[1]))
}

// TestBalanceSidesStayEven: each member matches its mirror on everything
// that decides a blow, and nobody levels during the fight (review: the
// companions' template experience levelled Garrick and Ysolde on their
// first kill at level 1).
func TestBalanceSidesStayEven(t *testing.T) {
	for _, level := range []int{1, 5, 10} {
		t.Run(fmt.Sprintf("L%d", level), func(t *testing.T) {
			f := newBalanceFight(t, level, companyDefault, enemySpread)
			members := []*characters.Character{f.aria.Character}
			for id := 1; id <= 4; id++ {
				members = append(members, &f.companion(id).Character)
			}
			// balanceMirror is in companion order, with Aria's mirror last.
			mirrors := []int{f.enemies[4], f.enemies[0], f.enemies[1], f.enemies[2], f.enemies[3]}
			for i, c := range members {
				m := &mobs.GetInstance(mirrors[i]).Character
				assert.Equal(t, level, c.Level, c.Name)
				assert.Equal(t, m.Level, c.Level, "%s vs %s: level", c.Name, m.Name)
				assert.Equal(t, m.Experience, c.Experience, "%s vs %s: experience", c.Name, m.Name)
				for _, s := range [][2]int{
					{c.Stats.Speed.ValueAdj, m.Stats.Speed.ValueAdj},
					{c.Stats.Perception.ValueAdj, m.Stats.Perception.ValueAdj},
					{c.Stats.Strength.ValueAdj, m.Stats.Strength.ValueAdj},
					{c.Stats.Smarts.ValueAdj, m.Stats.Smarts.ValueAdj},
				} {
					assert.Equal(t, s[1], s[0], "%s vs %s: a stat", c.Name, m.Name)
				}
				// A player's HealthMax starts with Base 1 (characters.New), a
				// mob's with 0, so a player gains racial HP growth a mob
				// doesn't: an engine quirk the harness keeps (30g4 replaces
				// the HP formula).
				playerHP := 0
				if c == f.aria.Character {
					playerHP = (&stats.StatInfo{Base: 1}).GainsForLevel(level) - (&stats.StatInfo{}).GainsForLevel(level)
				}
				assert.Equal(t, m.HealthMax.Value+playerHP, c.HealthMax.Value, "%s vs %s: health", c.Name, m.Name)
				assert.Equal(t, m.GetDefense(), c.GetDefense(), "%s vs %s: armor", c.Name, m.Name)
			}
			f.run()
			assert.Equal(t, level, f.aria.Character.Level, "Aria stays at level")
			for id := 1; id <= 4; id++ {
				if instance, ok := module.instance(7, id); ok {
					if m := mobs.GetInstance(instance); m != nil {
						assert.Equal(t, level, m.Character.Level, "%s stays at level", m.Character.Name)
					}
				}
			}
			for _, id := range f.enemies {
				if m := mobs.GetInstance(id); m != nil {
					assert.Equal(t, level, m.Character.Level, "%s stays at level", m.Character.Name)
				}
			}
		})
	}
}

// TestBalanceStatusesLand: a blow's status reaches its target and ticks
// in the harness, as in the game (review: without 30a's buffs and the
// Buff listener, no status ever landed and tick damage read 0).
func TestBalanceStatusesLand(t *testing.T) {
	f := newBalanceFight(t, 10, companyDefault, enemySpread)
	f.step() // the battle begins; a status lands only on someone in a fight
	var target *mobs.Mob
	for _, id := range f.enemies {
		if m := mobs.GetInstance(id); m != nil && m.Character.Health >= 1 {
			target = m
			break
		}
	}
	require.NotNil(t, target, "an enemy still stands after one round")
	target.Character.HealthMax.Value, target.Character.Health = 1000, 1000 // it outlasts its bleed
	events.AddToQueue(events.Buff{MobInstanceId: target.InstanceId, BuffId: status.Bleeding, Source: "test"})
	events.ProcessEvents()
	require.True(t, target.Character.HasBuff(status.Bleeding), "the bleed landed")
	res := f.run()
	assert.Positive(t, res.Tally.TickDamage[sideCompany], "the enemy's bleed ticked for the company")
}
