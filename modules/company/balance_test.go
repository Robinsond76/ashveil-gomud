package company

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/coordination"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/sigils"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/strategy"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 30g1: the balance harness. An evenly matched 5v5 (the company
// against a mirror group with its kits) is fought through the real combat
// round with real dice, many times per cell, and the table of outcomes is
// the baseline every later 30g slice is measured against. TestBalance5v5
// runs only with ASHVEIL_BALANCE=1; TestBalanceHarnessRunsAFight keeps the
// harness honest in the ordinary suite.

// balanceMaxRounds is where a fight counts as a stall.
const balanceMaxRounds = 200

// balanceBossHPBonus is the boss's extra health over an ordinary foe of its
// level, the game's own number (35d; 37b retuned it).
const balanceBossHPBonus = encounters.BossHPBonus

// balanceGroup is the mirror group's spawn group (non-hostile mobs group
// only by a spawn group, Phase 32d).
const balanceGroup = "brawl:mirror"

// Company modes (decision 14). Spread gives each member a different rule,
// so blows land across the enemy; default keeps the shipped strategies
// (each member aims at the weakest); focus is `tactics` weakest.
//
// Spread and focus are the true mirror (30g6 amendment): the mirror group
// has no class abilities and no healer, so the company fights with its
// abilities off and its cleric without mana (still a healer by role, the
// caster enemies aim at), and the tactic is the only difference. Kit is spread with the company's shipped
// abilities and healing, measured against that mirror; default keeps them
// too.
const (
	companySpread  = "spread"
	companyDefault = "default"
	companyFocus   = "focus"
	companyKit     = "kit"
	// companyTactics (Phase 35b) is the default company with the player's
	// answer to a coordinated group: a focus tactic on the weakest foe, and
	// its warrior guarding its healer.
	companyTactics = "tactics"
)

// balanceMirrored reports whether a company mode fights as the exact mirror.
func balanceMirrored(companyMode string) bool {
	return companyMode == companySpread || companyMode == companyFocus
}

// Enemy modes: spread makes each new aim a random foe (an aim sticks until
// its target falls or can't be reached); default chooses weakest and casters chooses spell users, both at the
// rabble tier (which imposes at least 30% targeting noise).
const (
	enemySpread  = "spread"
	enemyDefault = "default"
	enemyCasters = "casters"
	// enemyHealer (Phase 37): the mirror's priest is a healer in an
	// otherwise ordinary group (Weakest targeting), as an encounter's healer
	// group is.
	enemyHealer = "healer"
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
	{9202, "hedge priest", "    weapon:\n      itemid: 10023\n    body:\n      itemid: 20008\n"},                                                              // Oswin (35a2: a mace)
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
	// Phase 35b: spells begun, cast, fizzled and broken off, by side.
	Casts, Cast, Fizzled, Broken [2]int
	// Phase 35d: swings that ended with no damage, the quality of the blows
	// that landed, and Minor Heal chants begun and finished.
	NoDamage                  [2]int
	Glancing, Telling         [2]int
	HealsBegun, HealsFinished [2]int
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
		if e.Damage == 0 {
			t.NoDamage[s]++
		}
		switch e.Quality {
		case combat.QualityGlancing:
			t.Glancing[s]++
		case combat.QualityTelling:
			t.Telling[s]++
		}
	case combatstream.SpellHit:
		t.Damage[s] += e.Damage
	case combatstream.Heal:
		t.Healing[s] += e.Amount
	case combatstream.CastStart:
		t.Casts[s]++
		if e.SpellId == "heal" {
			t.HealsBegun[s]++
		}
	case combatstream.CastComplete:
		switch e.Outcome {
		case combatstream.OutcomeCast:
			t.Cast[s]++
			if e.SpellId == "heal" {
				t.HealsFinished[s]++
			}
		case combatstream.OutcomeFizzled:
			t.Fizzled[s]++
		case combatstream.OutcomeInterrupted:
			t.Broken[s]++
		}
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
	Lines, PeakLines int // Text lines delivered to the leader, including round aftermath.
	Rounds           int
	Won, Stalled     bool
	HPRemoved        [2]int // Starting minus remaining health, by victim side; excludes overkill.
	StartHP          [2]int // Each side's health when the fight began (35a2's share-lost rows).
	Fallen           [2]int
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

// Balance uses the shipped archetype rates rather than provisional test constants.
type balanceHPArchetypes struct {
	fakeArchetypes
	rates    map[string]float64
	profiles map[string]archetypes.Profile
}

// CombatProfile serves the shipped classes' 35a2 profiles (HPStart,
// Attack and Evasion rates, armor training, gear rules).
func (p balanceHPArchetypes) CombatProfile(id string) (archetypes.Profile, bool) {
	v, ok := p.profiles[id]
	return v, ok
}

func (p balanceHPArchetypes) HealthPerLevel(id string) (float64, bool) {
	v, ok := p.rates[id]
	return v, ok
}
func (p balanceHPArchetypes) HealthArchetypes() map[string]float64 { return p.rates }
func balanceHPProvider(t *testing.T) balanceHPArchetypes {
	t.Helper()
	data, err := os.ReadFile("modules/archetype/files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg struct {
		Archetypes []struct {
			ID            string   `yaml:"ArchetypeId"`
			Name          string   `yaml:"Name"`
			HP            float64  `yaml:"HPPerLevel"`
			HPStart       int      `yaml:"HPStart"`
			AttackRate    float64  `yaml:"AttackRate"`
			EvasionRate   float64  `yaml:"EvasionRate"`
			ArmorTraining string   `yaml:"ArmorTraining"`
			ShieldSizes   []string `yaml:"ShieldSizes"`
			WeaponClasses []string `yaml:"WeaponClasses"`
			ManaBase      int      `yaml:"ManaBase"`
			ManaPerLevel  float64  `yaml:"ManaPerLevel"`
		} `yaml:"Archetypes"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	p := balanceHPArchetypes{rates: map[string]float64{}, profiles: map[string]archetypes.Profile{}}
	for _, c := range cfg.Archetypes {
		p.rates[c.ID] = c.HP
		p.profiles[c.ID] = archetypes.Profile{Name: c.Name, AttackRate: c.AttackRate, EvasionRate: c.EvasionRate, HPStart: c.HPStart, ArmorTraining: c.ArmorTraining, ShieldSizes: c.ShieldSizes, WeaponClasses: c.WeaponClasses, ManaBase: c.ManaBase, ManaPerLevel: c.ManaPerLevel}
	}
	require.NotEmpty(t, p.rates, "shipped archetypes must load")
	for _, id := range []string{"warrior", "cleric", "ranger", "rogue", "wizard"} {
		require.Positive(t, p.rates[id], id+" HP rate must load")
	}
	return p
}

type balanceFightOptions struct {
	EnemyCount   int
	EnemyLevels  []int
	Coordination int // Zero leaves live group-level coordination in force.
	Boss         bool
	LegacyMirror bool // The original five-member mirror remains a stress test.
	// Classes overrides companions' archetypes (Phase 35b: a wizard for
	// the mana run); the rest keep the shipped ones.
	Classes map[int]string
	// Sigil is the sigil the leader laid in the room before the fight
	// (Phase 54); empty for none.
	Sigil sigils.Kind
	// Setup runs once the company and the foes stand ready, just before the
	// attack (Phase 69: gear and stances).
	Setup func(b *brawl)
}

func newBalanceFight(t *testing.T, level int, companyMode, enemyMode string, enemyLevels ...int) *balanceFight {
	return newBalanceFightWithOptions(t, level, companyMode, enemyMode, balanceFightOptions{EnemyCount: 5, EnemyLevels: enemyLevels, Coordination: 1, LegacyMirror: true})
}

func newBalanceFightWithOptions(t *testing.T, level int, companyMode, enemyMode string, opts balanceFightOptions) *balanceFight {
	t.Helper()
	b := newBrawl(t)
	mudlog.SetLogLevel("LOW") // report the table, without a log line for every fixture buff
	t.Cleanup(hooks.UseTempoForTest(nil))
	classes := map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"}
	for id, class := range opts.Classes {
		classes[id] = class
	}
	b.withArchetypesFor("", classes)
	// 30g4: configured classes supply HP on both sides of the even mirror.
	archetypes.SetProvider(balanceHPProvider(t))
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.HPArchetype, _ = module.CompanionArchetype(7, id)
	}
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

	require.GreaterOrEqual(t, opts.EnemyCount, 2)
	require.LessOrEqual(t, opts.EnemyCount, 5)
	roster := append(balanceMirror[:0:0], balanceMirror...)
	if !opts.LegacyMirror {
		roster[4].name = "brute"
		roster[4].equipment = roster[0].equipment
	}
	roster = roster[:opts.EnemyCount]
	dataDir := configs.GetFilePathsConfig().DataFiles.String()
	for _, m := range roster {
		path := filepath.Join(dataDir, "mobs", "brawl", fmt.Sprintf("%d-%s.yaml", m.id, strings.ReplaceAll(m.name, " ", "_")))
		require.NoError(t, os.WriteFile(path, []byte(mirrorMob(m.id, m.name, m.equipment)), 0600))
	}
	mobs.LoadDataFiles()

	f := &balanceFight{brawl: b, tally: &balanceTally{}}
	for i, m := range roster {
		enemyLevel := level
		if len(opts.EnemyLevels) > 0 {
			enemyLevel = opts.EnemyLevels[min(i, len(opts.EnemyLevels)-1)]
		}
		enemyLevel = max(enemyLevel, 1)
		if opts.Boss && i == 0 {
			enemyLevel += encounters.BossLevelBonus
		}
		mob := mobs.NewMobById(mobs.MobId(m.id), b.road.RoomId)
		require.NotNil(t, mob, m.name)
		mob.SpawnGroup = balanceGroup
		// Passive and personality controls do not acquire group focus merely
		// by crossing a level tier. Enhanced coordination has its own cells.
		mob.Coordination = opts.Coordination
		class := map[int]string{9201: "warrior", 9202: "cleric", 9203: "warrior", 9204: "ranger"}[m.id]
		if !opts.LegacyMirror && i == 4 {
			class = "warrior"
		}
		if class != "" {
			mob.Character.HPPerLevel, _ = archetypes.HealthPerLevel(class)
			// 35a2: the mirror fights as its class too (HP head start,
			// Attack and Evasion rates, armor training).
			mob.Character.HPArchetype = class
		} else {
			mob.Character.HPPerLevel = float64(configs.GetProgressionConfig().DefaultHPPerLevel)
		}
		b.road.AddMob(mob.InstanceId)
		f.enemies = append(f.enemies, mob.InstanceId)
		levelTo(&mob.Character, enemyLevel)
		if opts.Boss && i == 0 {
			mob.Character.HealthMax.Training += int(float64(mob.Character.HealthMax.Value) * balanceBossHPBonus)
			mob.Character.RecalculateStats()
			mob.Character.Health = mob.Character.HealthMax.Value
		}
		switch {
		case enemyMode == enemySpread:
			mob.Targeting, mob.TargetingNoise = string(strategy.Nearest), 100
		case enemyMode == enemyDefault:
			mob.Targeting, mob.TargetingNoise = string(strategy.Weakest), 0
		case enemyMode == enemyCasters:
			mob.Targeting, mob.TargetingNoise = string(strategy.Casters), 0
		case enemyMode == enemyHealer:
			mob.Targeting, mob.TargetingNoise = string(strategy.Weakest), 0
			if m.id == 9202 {
				mob.Role = "healer"
				mob.Character.SpellBook["heal"] = 250
			}
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

	if balanceMirrored(companyMode) {
		for _, who := range []string{"me", "tamsin", "oswin", "garrick", "ysolde"} {
			require.Contains(t, b.cmd("strategy", who+" abilities off"), "class abilities", who)
		}
		// Oswin stays a healer, so caster-targeting enemies still see a
		// caster, but with no mana he casts nothing and swings like the
		// mirror's hedge priest (review: a "fighter" Oswin left the casters
		// cell measuring nearest-foe targeting).
		oswin := &b.companion(2).Character
		oswin.Mana, oswin.ManaMax.Value = 0, 0
	}
	// Phase 35d: an untouched company now has the level ladder's defaults (a
	// focus by leader level, a warrior guarding the healer), so the modes
	// that mean something else say so: their warriors fight, and spread and
	// kit keep every member to its own rule.
	if companyMode != companyDefault && companyMode != companyTactics {
		for _, who := range []string{"tamsin", "garrick"} {
			b.cmd("strategy", who+" fighter")
		}
	}
	switch companyMode {
	case companySpread, companyKit:
		b.saveTactics(strategy.Tactics{Focus: strategy.NoFocus})
		for _, s := range []string{"tamsin nearest", "garrick strongest", "ysolde furthest", "oswin wounded"} {
			require.Contains(t, b.cmd("strategy", s), "will go for", s)
		}
	case companyDefault:
	case companyFocus:
		b.saveTactics(strategy.Tactics{Focus: strategy.Weakest})
	case companyTactics:
		b.saveTactics(strategy.Tactics{Focus: strategy.Weakest})
		require.Contains(t, b.cmd("strategy", "tamsin guard oswin"), "guard", "tamsin guards the healer")
	default:
		t.Fatalf("company mode %q", companyMode)
	}

	s := combatstream.New()
	s.Subscribe(func(e combatstream.Event) { f.tally.add(e) })
	t.Cleanup(combatstream.UseForTest(s))

	opening := f.enemies[0]
	if companyMode == companyFocus || companyMode == companyTactics {
		for _, g := range enemyparty.Groups(b.road) {
			if aim, ok := enemyparty.Aim(g, enemyparty.PlayerAttacker(b.aria)); ok {
				opening = aim
				break
			}
		}
	}
	if opts.Sigil != sigils.None {
		b.aria.Character.Sigil = sigils.Lay(opts.Sigil, b.road.RoomId, time.Now())
		t.Cleanup(func() { b.aria.Character.Sigil = sigils.Laid{} })
	}
	if opts.Setup != nil {
		opts.Setup(b)
	}
	b.cmd("attack", fmt.Sprintf("#%d", opening))
	// Spread starts with one distinct opponent per fighter. Strategies keep
	// a reachable aim until it falls; a list of different rules alone does
	// not spread initial blows when all foes begin at full health.
	members := []*characters.Character{b.aria.Character}
	for id := 1; id <= 4; id++ {
		members = append(members, &b.companion(id).Character)
	}
	mirrors := make([]int, len(members))
	for i := range mirrors {
		mirrors[i] = f.enemies[(i+len(f.enemies)-1)%len(f.enemies)]
	}
	for i, c := range members {
		if companyMode == companySpread {
			c.SetAggro(0, mirrors[i], characters.DefaultAttack)
		}
		if enemyMode == enemySpread {
			m := mobs.GetInstance(mirrors[i])
			uid, mid := 0, 0
			if i == 0 {
				uid = 7
			} else {
				mid = b.companion(i).InstanceId
			}
			m.Character.SetAggro(uid, mid, characters.DefaultAttack)
		}
	}
	// Personalities must choose an opening target through upkeep, rather
	// than inherit the attack command's retaliation aim at the leader.
	if enemyMode != enemySpread {
		require.NotNil(t, b.aria.Character.Aggro, "attack command opened combat")
		require.Contains(t, f.enemies, b.aria.Character.Aggro.MobInstanceId)
		for _, id := range f.enemies {
			mobs.GetInstance(id).Character.EndAggro()
		}
	}
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
	c.StatPoints = cfg.StatPointsAt(level)
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

// healthRemaining totals live health on each side. A member who fled or
// was routed keeps its health (it was not removed); a fallen or destroyed
// one counts zero.
func (f *balanceFight) healthRemaining() (hp [2]int) {
	hp[sideCompany] = max(0, f.aria.Character.Health)
	for id := 1; id <= 4; id++ {
		if instance, ok := module.instance(7, id); ok {
			if m := mobs.GetInstance(instance); m != nil {
				hp[sideCompany] += max(0, m.Character.Health)
			}
		}
	}
	for _, id := range f.enemies {
		if m := mobs.GetInstance(id); m != nil {
			hp[sideEnemy] += max(0, m.Character.Health)
		}
	}
	return hp
}

// run fights until one side is down or the fight stalls. Each combat
// round is two game rounds (29f), so regeneration's pass runs for both,
// as the game does (it skips anyone in a battle, and tops up companions'
// mana every third round).
// Aria is held at 0 once she falls: fallen and healable, never dead (a
// player's death would cost her a level and move her to a church).
func (f *balanceFight) run() balanceResult {
	var res balanceResult
	startHP := f.healthRemaining()
	res.StartHP = startHP
	for res.Rounds < balanceMaxRounds {
		company, enemy := f.standing()
		if company == 0 || enemy == 0 {
			break
		}
		res.FighterRounds[sideCompany] += company
		res.FighterRounds[sideEnemy] += enemy
		res.Rounds++
		f.step()
		lines := 0
		for _, msg := range *f.messages {
			if strings.TrimSpace(msg) != "" {
				lines += strings.Count(strings.TrimSpace(msg), "\n") + 1
			}
		}
		res.Lines += lines
		res.PeakLines = max(res.PeakLines, lines)
	}
	company, enemy := f.standing()
	res.Won = enemy == 0 && company > 0
	res.Stalled = company > 0 && enemy > 0
	res.Fallen = [2]int{5 - company, len(f.enemies) - enemy}
	res.Tally = *f.tally
	endHP := f.healthRemaining()
	for side := range startHP {
		res.HPRemoved[side] = startHP[side] - endHP[side]
	}
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
// per cell (default 100).
func TestBalance5v5(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the balance table")
	}
	fights := 100
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	var rows []string
	for _, level := range []int{1, 5, 10, 30, 60, 100} {
		cells := map[string][]balanceResult{}
		for _, cm := range []string{companySpread, companyKit, companyDefault, companyFocus} {
			for _, em := range []string{enemySpread, enemyDefault, enemyCasters} {
				if cm == companyKit && em != enemySpread {
					continue
				}
				var results []balanceResult
				for i := 0; i < fights; i++ {
					t.Run(fmt.Sprintf("L%d-%s-%s-%d", level, cm, em, i), func(t *testing.T) {
						results = append(results, newBalanceFight(t, level, cm, em).run())
					})
				}
				rows = append(rows, balanceRow(level, cm, em, results))
				cells[cm+"/"+em] = results
			}
		}
		if level <= 60 {
			passive := cells[companySpread+"/"+enemySpread]
			baselineMedian, baselineWins := balanceMedianAndWins(passive)
			// The mirror is even: neither side should be clearly favored.
			assert.GreaterOrEqual(t, baselineWins, 35, "L%d mirror parity", level)
			assert.LessOrEqual(t, baselineWins, 65, "L%d mirror parity", level)
			// Class abilities and healing must not be a handicap.
			kitWins := balanceWins(cells[companyKit+"/"+enemySpread])
			assert.Greater(t, balanceWelchZ(kitWins, balanceWins(passive)), -balanceSignificantZ, "L%d kit is no worse than the mirror", level)
			// 35a2 acceptance 4, retuned in 35b: equal fights stay short.
			// 8–12 would need hits of a third of a fighter's health (35a2
			// row 1 holds them to a fifth), so the mirror is held to 16.
			assert.GreaterOrEqual(t, baselineMedian, 8, "L%d spread median", level)
			assert.LessOrEqual(t, baselineMedian, 16, "L%d spread median", level)
			// Company focus is reported, not asserted (owner, 2026-10-04): it is
			// a strategy the player may choose, not a guaranteed advantage.
			// Enemy targeting must still trouble a passive company (decision 14).
			for _, em := range []string{enemyDefault, enemyCasters} {
				lost, base := balanceHPLost(cells[companySpread+"/"+em], sideCompany), balanceHPLost(passive, sideCompany)
				z := balanceWelchZ(lost, base)
				// Phase 35d: the mirror is a stress check, reported not asserted;
				// the default guard and focus now help the passive company too,
				// so enemy targeting no longer shows as significant at 30+.
				t.Logf("L%d %s company health lost %.1f vs %.1f passive (z %.1f)", level, em, balanceMean(lost), balanceMean(base), z)
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
// cell (default 100).
func TestBalanceCoordinated(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run the balance table")
	}
	fights := 100
	if v, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && v > 0 {
		fights = v
	}
	var rows []string
	for _, level := range []int{1, 5, 10, 30} {
		cellFor := func(cm, em string) []balanceResult {
			var results []balanceResult
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("L%d-%s-%s-%d", level, cm, em, i), func(t *testing.T) {
					results = append(results, newBalanceFight(t, level, cm, em).run())
				})
			}
			rows = append(rows, balanceRow(level, cm, em, results))
			return results
		}
		cell := func(em string) []balanceResult { return cellFor(companyDefault, em) }
		base := cell(enemyDefault)
		baseMedian, baseWins := balanceMedianAndWins(base)
		for tier := 1; tier <= 3; tier++ {
			median, wins := balanceMedianAndWins(cell(fmt.Sprintf("%s%d", enemyRoles, tier)))
			assert.LessOrEqual(t, median*4, baseMedian*5, "L%d tier %d: median %d against %d", level, tier, median, baseMedian)
			// Phase 35b retires "a clear winner keeps winning at least half
			// the time" for a company with no tactics: coordination is meant
			// to beat it. The answer in kind is asserted below.
			if baseWins <= 40 {
				assert.LessOrEqual(t, wins, 50, "L%d tier %d: the enemy kept winning", level, tier)
			}
		}
		// Phase 35b (replaces the 35a2 row "tiers 2–3 keep a level-10
		// company's wins at 50%"): a coordinated group beats a company that
		// leaves its targeting to chance, but a company that answers in
		// kind (a focus, and its warrior guarding its healer) wins at least
		// half its fights against the tier its own level meets (a band at
		// level 10, a drilled company at 30), and a third against one a
		// tier above.
		for tier := 2; tier <= 3; tier++ {
			// Phase 35d: the default tactics ladder (decision 4) is what
			// answers in kind; explicit tactics stay a reported cell.
			_, wins := balanceMedianAndWins(cellFor(companyDefault, fmt.Sprintf("%s%d", enemyRoles, tier)))
			cellFor(companyTactics, fmt.Sprintf("%s%d", enemyRoles, tier))
			switch native := coordination.ForLevel(level); {
			case coordination.Tier(tier) == native:
				assert.GreaterOrEqual(t, wins, 50, "L%d tier %d: a coordinated company wins", level, tier)
			case coordination.Tier(tier) == native+1:
				assert.GreaterOrEqual(t, wins, 33, "L%d tier %d: a coordinated company holds its own", level, tier)
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

// balanceSignificantZ is a one-sided 95% threshold. A tactics claim must
// show in the sample, not merely lean its way: real dice make point
// estimates of equal cells differ by chance.
const balanceSignificantZ = 1.645

// balanceWonRounds is the rounds of each fight the company won.
func balanceWonRounds(results []balanceResult) []float64 {
	var out []float64
	for _, r := range results {
		if r.Won {
			out = append(out, float64(r.Rounds))
		}
	}
	return out
}

// balanceWins is each fight's outcome for the company, 1 for a win.
func balanceWins(results []balanceResult) []float64 {
	out := make([]float64, 0, len(results))
	for _, r := range results {
		if r.Won {
			out = append(out, 1)
		} else {
			out = append(out, 0)
		}
	}
	return out
}

// balanceHPLost is each fight's health removed from a side.
func balanceHPLost(results []balanceResult, side int) []float64 {
	out := make([]float64, 0, len(results))
	for _, r := range results {
		out = append(out, float64(r.HPRemoved[side]))
	}
	return out
}

func balanceMean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	total := 0.0
	for _, x := range v {
		total += x
	}
	return total / float64(len(v))
}

// balanceWelchZ is how many standard errors a's mean is above b's
// (Welch's statistic; a z threshold is fine at the harness's sample sizes).
// Fewer than two values on either side, or no spread at all, gives 0
// unless the means already differ.
func balanceWelchZ(a, b []float64) float64 {
	if len(a) < 2 || len(b) < 2 {
		return 0
	}
	variance := func(v []float64) float64 {
		m, total := balanceMean(v), 0.0
		for _, x := range v {
			total += (x - m) * (x - m)
		}
		return total / float64(len(v)-1)
	}
	diff := balanceMean(a) - balanceMean(b)
	se := math.Sqrt(variance(a)/float64(len(a)) + variance(b)/float64(len(b)))
	if se == 0 {
		switch {
		case diff > 0:
			return math.Inf(1)
		case diff < 0:
			return math.Inf(-1)
		}
		return 0
	}
	return diff / se
}

const balanceHeader = "| level | company | enemy | fights | company wins | rounds p10/median/p90 | won-fight rounds mean | stalls | fallen company/enemy | damage company/enemy | net HP lost company/enemy | healing company/enemy | turns per fighter-round company/enemy | hit% company/enemy | crit% company/enemy | blocks/parries/dodges company · enemy | bashes company/enemy | tick damage company/enemy | lines/round mean/peak | casts begun/cast/fizzled/broken company · enemy | seconds median | lines per fight | no-damage swings% of weapon swings company/enemy | glancing/telling% of landed company/enemy | heals finished/begun company · enemy |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"

// balanceRow is one cell's line of the table: averages are per fight.
func balanceRow(level int, cm, em string, results []balanceResult) string {
	n := len(results)
	if n == 0 {
		return fmt.Sprintf("| %d | %s | %s | 0 | | | | | | | | | | | | |", level, cm, em)
	}
	var rounds []int
	var wins, stalls int
	var fallen, damage, hpLost, turns, hits, crits, fighterRounds, counters, ticks, blocks, parries, dodges [2]int
	var casts, cast, fizzled, broken [2]int
	var healing [2]int
	var lines, totalRounds, peakLines int
	for _, r := range results {
		lines += r.Lines
		totalRounds += r.Rounds
		peakLines = max(peakLines, r.PeakLines)
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
			hpLost[s] += r.HPRemoved[s]
			turns[s] += r.Tally.Turns[s]
			hits[s] += r.Tally.Hits[s]
			crits[s] += r.Tally.Crits[s]
			fighterRounds[s] += r.FighterRounds[s]
			counters[s] += r.Tally.Counters[s]
			blocks[s] += r.Tally.Blocks[s]
			parries[s] += r.Tally.Parries[s]
			dodges[s] += r.Tally.Dodges[s]
			ticks[s] += r.Tally.TickDamage[s]
			casts[s] += r.Tally.Casts[s]
			cast[s] += r.Tally.Cast[s]
			fizzled[s] += r.Tally.Fizzled[s]
			broken[s] += r.Tally.Broken[s]
		}
		healing[0] += r.Tally.Healing[sideCompany]
		healing[1] += r.Tally.Healing[sideEnemy]
	}
	per := func(v int) float64 { return float64(v) / float64(n) }
	ratio := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / float64(b)
	}
	p10, p50, p90 := percentile(rounds, 10), percentile(rounds, 50), percentile(rounds, 90)
	var noDamage, swings, glancing, telling, healsBegun, healsDone [2]int
	for _, r := range results {
		for s := 0; s < 2; s++ {
			noDamage[s] += r.Tally.NoDamage[s]
			swings[s] += r.Tally.Hits[s] + r.Tally.Misses[s]
			glancing[s] += r.Tally.Glancing[s]
			telling[s] += r.Tally.Telling[s]
			healsBegun[s] += r.Tally.HealsBegun[s]
			healsDone[s] += r.Tally.HealsFinished[s]
		}
	}
	return fmt.Sprintf("| %d | %s | %s | %d | %.0f%% | %d/%d/%d | %.1f | %d | %.1f/%.1f | %.0f/%.0f | %.0f/%.0f | %.0f/%.0f | %.2f/%.2f | %.0f/%.0f | %.0f/%.0f | %.1f/%.1f/%.1f · %.1f/%.1f/%.1f | %.1f/%.1f | %.1f/%.1f | %.1f/%d | %.1f/%.1f/%.1f/%.1f · %.1f/%.1f/%.1f/%.1f | %d | %.0f | %.0f/%.0f | %.0f/%.0f · %.0f/%.0f | %.1f/%.1f · %.1f/%.1f |",
		level, cm, em, n, 100*ratio(wins, n), p10, p50, p90, balanceMean(balanceWonRounds(results)), stalls,
		per(fallen[0]), per(fallen[1]), per(damage[0]), per(damage[1]), per(hpLost[0]), per(hpLost[1]), per(healing[0]), per(healing[1]),
		ratio(turns[0], fighterRounds[0]), ratio(turns[1], fighterRounds[1]),
		100*ratio(hits[0], turns[0]), 100*ratio(hits[1], turns[1]),
		100*ratio(crits[0], hits[0]), 100*ratio(crits[1], hits[1]),
		per(blocks[0]), per(parries[0]), per(dodges[0]), per(blocks[1]), per(parries[1]), per(dodges[1]),
		per(counters[0]), per(counters[1]), per(ticks[0]), per(ticks[1]), ratio(lines, totalRounds), peakLines,
		per(casts[0]), per(cast[0]), per(fizzled[0]), per(broken[0]), per(casts[1]), per(cast[1]), per(fizzled[1]), per(broken[1]),
		p50*balanceSecondsPerRound(), per(lines),
		100*ratio(noDamage[0], swings[0]), 100*ratio(noDamage[1], swings[1]),
		100*ratio(glancing[0], hits[0]), 100*ratio(telling[0], hits[0]), 100*ratio(glancing[1], hits[1]), 100*ratio(telling[1], hits[1]),
		per(healsDone[0]), per(healsBegun[0]), per(healsDone[1]), per(healsBegun[1]))
}

// balanceSecondsPerRound is a combat round's length in seconds: it resolves
// every second game round (29f), so 2 x RoundSeconds, 8 in the shipped
// config. The 35d design's "rounds x RoundSeconds" read a combat round as
// one 4-second game round; the measurements report the real figure.
func balanceSecondsPerRound() int {
	return 2 * int(configs.GetTimingConfig().RoundSeconds)
}

// TestBalanceSidesStayEven: each member matches its mirror on everything
// that decides a blow, and nobody levels during the fight (review: the
// companions' template experience levelled Garrick and Ysolde on their
// first kill at level 1).
func TestBalanceSidesStayEven(t *testing.T) {
	for _, level := range []int{1, 5, 10, 30, 60, 100} {
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
				// 30g4 removes the player's hidden racial HealthMax growth.
				rate := float64(configs.GetProgressionConfig().DefaultHPPerLevel)
				if i > 0 {
					class, _ := module.CompanionArchetype(7, i)
					var ok bool
					rate, ok = archetypes.HealthPerLevel(class)
					require.True(t, ok, class)
				}
				assert.Equal(t, rate, c.HealthGainPerLevel(), c.Name+" configured class rate")
				assert.Equal(t, rate, m.HealthGainPerLevel(), m.Name+" mirrored class rate")
				assert.Equal(t, configs.GetProgressionConfig().HealthAtLevel(level, c.Stats.Vitality.ValueAdj, rate, c.HPStart())+c.StatMod("healthmax"), c.HealthMax.Value, c.Name+" configured HP")
				assert.Equal(t, m.HealthMax.Value, c.HealthMax.Value, "%s vs %s: health", c.Name, m.Name)
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

func TestBalanceMismatches(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to run mismatches")
	}
	fights := 100
	if n, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && n > 0 {
		fights = n
	}
	for _, level := range []int{15, 30} {
		var results []balanceResult
		for i := 0; i < fights; i++ {
			t.Run(fmt.Sprintf("L%d-v10-%d", level, i), func(t *testing.T) {
				results = append(results, newBalanceFight(t, level, companySpread, enemySpread, 10).run())
			})
		}
		t.Log(balanceRow(level, companySpread, "L10-spread", results))
		_, winPct := balanceMedianAndWins(results)
		if level == 30 {
			assert.GreaterOrEqual(t, winPct, 95, "L30 against L10 wins at least 95%%")
			intact := 0
			for _, r := range results {
				if r.Fallen[sideCompany] == 0 {
					intact++
				}
			}
			assert.Greater(t, intact*2, len(results), "L30 against L10 loses no member in most fights")
		} else {
			// Phase 35a2 (owner, 2026-10-05): five levels of skill decide the
			// fight, so the 30g6 "can lose" row is retired.
			assert.Greater(t, winPct, 50, "L15 against L10 is favored")
		}
	}
}

func TestBalanceWelchZ(t *testing.T) {
	assert.Zero(t, balanceWelchZ([]float64{1}, []float64{1, 2}), "too few values")
	assert.Zero(t, balanceWelchZ([]float64{3, 3}, []float64{3, 3}))
	assert.True(t, math.IsInf(balanceWelchZ([]float64{4, 4}, []float64{3, 3}), 1))
	// Means 12 and 10, each variance 4 over 50: se = sqrt(0.16) = 0.4, z = 5.
	a, b := make([]float64, 50), make([]float64, 50)
	for i := range a {
		a[i], b[i] = 12+2*float64(i%2*2-1), 10+2*float64(i%2*2-1)
	}
	assert.InDelta(t, 2/math.Sqrt(2*4.0816326/50), balanceWelchZ(a, b), 1e-6)
	assert.Less(t, balanceWelchZ(b, a), -balanceSignificantZ)
	won := balanceWonRounds([]balanceResult{{Rounds: 9, Won: true}, {Rounds: 4}, {Rounds: 11, Won: true}})
	assert.Equal(t, []float64{9, 11}, won)
}

func TestBalanceZoneOptions(t *testing.T) {
	for _, n := range []int{2, 3, 4, 5} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := newBalanceFightWithOptions(t, 10, companyDefault, enemyDefault, balanceFightOptions{EnemyCount: n, EnemyLevels: []int{9, 8}, Boss: n == 5})
			require.Len(t, f.enemies, n)
			for i, id := range f.enemies {
				m := mobs.GetInstance(id)
				assert.Zero(t, m.Coordination, "live tier comes from group level")
				want := 8
				if i == 0 {
					want = 9
					if n == 5 {
						want += encounters.BossLevelBonus
					}
				}
				assert.Equal(t, want, m.Character.Level)
				if i == 0 && n == 5 {
					assert.Positive(t, m.Character.HealthMax.Training, "boss adds 1.5 times ordinary HP")
				}
			}
			assert.False(t, f.run().Stalled)
		})
	}
}

// Phase 35d fight-length targets for a band-middle company over its zone's 2
// and 3 foe groups: a median of 6 to 9 combat rounds. (The design also gives
// 25 to 40 seconds, reading a combat round as 4 s; it is 8 s, see
// balanceSecondsPerRound, and the table reports the real seconds.)
const (
	zoneMiddleRoundsMin = 6
	zoneMiddleRoundsMax = 9
)

// TestBalanceZoneBands runs a company at each level of a zone band against
// the zone's groups (35a) and asserts the level impact design's section 4
// table as 35d reshapes it: four foes and a boss's escorts come from the
// band's low level, a boss runs at Rabble with 1.75x health and 2 or 3
// escorts, and the under-levelled company is 5 levels under the band.
func TestBalanceZoneBands(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("set ASHVEIL_BALANCE=1 to measure zone bands")
	}
	fights := 100
	if n, err := strconv.Atoi(os.Getenv("ASHVEIL_BALANCE_FIGHTS")); err == nil && n > 0 {
		fights = n
	}
	type zoneCell struct {
		level int
		cm    string
		row   string // the design row the cell counts toward, or "" to only report it
		opts  balanceFightOptions
	}
	for _, band := range [][2]int{{1, 3}, {8, 10}, {18, 20}, {28, 30}} {
		middle := (band[0] + band[1]) / 2
		var cells []zoneCell
		groups := func(level int, row string) {
			for _, count := range []int{2, 3} {
				for enemy := max(1, band[0]-1); enemy <= band[1]-1; enemy++ {
					cells = append(cells, zoneCell{level, companyDefault, row, balanceFightOptions{EnemyCount: count, EnemyLevels: []int{enemy}}})
				}
			}
		}
		for level := band[0]; level <= band[1]; level++ {
			row := map[int]string{band[0]: "low", middle: "middle"}[level]
			groups(level, row)
		}
		// Four foes at the band's low level, met by a company at the band's
		// middle and at its low end.
		cells = append(cells,
			zoneCell{middle, companyDefault, "", balanceFightOptions{EnemyCount: 4, EnemyLevels: []int{band[0]}}},
			zoneCell{band[0], companyDefault, "four", balanceFightOptions{EnemyCount: 4, EnemyLevels: []int{band[0]}}})
		// A boss (+2 levels, 1.75x health, at Rabble) with 2 or 3 escorts at
		// the band's low level, against a company at the band's top: as it
		// comes (the level ladder's defaults), and with the tactics a player
		// brings, reported.
		for _, count := range []int{3, 4} {
			cells = append(cells,
				zoneCell{band[1], companyDefault, "boss", balanceFightOptions{EnemyCount: count, EnemyLevels: []int{band[0]}, Boss: true, Coordination: 1}},
				zoneCell{band[1], companyTactics, "", balanceFightOptions{EnemyCount: count, EnemyLevels: []int{band[0]}, Boss: true, Coordination: 1}})
		}
		if under := band[0] - 5; under >= 1 {
			groups(under, "under")
		}

		rows := map[string][]balanceResult{}
		lost := map[string]float64{}
		for _, c := range cells {
			label := fmt.Sprintf("band%d-%d/L%d/%dvL%d/boss%t", band[0], band[1], c.level, c.opts.EnemyCount, c.opts.EnemyLevels[0], c.opts.Boss)
			var results []balanceResult
			var hpLostPct float64
			for i := 0; i < fights; i++ {
				t.Run(fmt.Sprintf("%s/%s/%d", label, c.cm, i), func(t *testing.T) {
					f := newBalanceFightWithOptions(t, c.level, c.cm, enemyDefault, c.opts)
					start := f.healthRemaining()[sideCompany]
					res := f.run()
					results = append(results, res)
					hpLostPct += 100 * float64(res.HPRemoved[sideCompany]) / float64(start)
				})
			}
			if len(results) == 0 {
				continue
			}
			t.Log(balanceRow(c.level, c.cm, label, results))
			t.Logf("ZONE %s %s mean HP lost %.1f%%", label, c.cm, hpLostPct/float64(len(results)))
			if c.row != "" {
				rows[c.row] = append(rows[c.row], results...)
				lost[c.row] += hpLostPct
			}
		}

		for _, row := range []string{"middle", "low", "four", "boss", "under"} {
			results := rows[row]
			if len(results) == 0 {
				continue
			}
			median, wins := balanceMedianAndWins(results)
			clean, fallen := 0, 0
			for _, r := range results {
				fallen += r.Fallen[sideCompany]
				if r.Fallen[sideCompany] == 0 {
					clean++
				}
			}
			n := float64(len(results))
			meanFallen, cleanPct, meanLost := float64(fallen)/n, 100*float64(clean)/n, lost[row]/n
			t.Logf("ZONEROW | %d-%d | %s | %d | %d%% | %.0f%% | %.2f | %.1f%% | %d | %ds |", band[0], band[1], row, len(results), wins, cleanPct, meanFallen, meanLost, median, median*balanceSecondsPerRound())
			name := fmt.Sprintf("band %d-%d %s", band[0], band[1], row)
			switch row {
			case "middle":
				assert.GreaterOrEqual(t, wins, 97, name)
				assert.GreaterOrEqual(t, cleanPct, 85.0, name+": nobody fallen")
				assert.LessOrEqual(t, meanLost, 30.0, name)
				assert.GreaterOrEqual(t, median, zoneMiddleRoundsMin, name)
				assert.LessOrEqual(t, median, zoneMiddleRoundsMax, name)
			case "low":
				assert.GreaterOrEqual(t, wins, 85, name)
				assert.LessOrEqual(t, meanFallen, 1.0, name)
				assert.LessOrEqual(t, meanLost, 45.0, name)
			case "four":
				assert.GreaterOrEqual(t, wins, 90, name)
				assert.LessOrEqual(t, meanFallen, 1.0, name)
			case "boss":
				// Timeboxed (35d): the shorter boss wins more than the 70-85%
				// design target; settled and recorded in the measurements doc,
				// so only the floor is asserted. Menace is left to 38b.
				assert.GreaterOrEqual(t, wins, 70, name)
				assert.GreaterOrEqual(t, median, 12, name)
				assert.LessOrEqual(t, median, 18, name)
			}
			// "under" is reported for phase 37 (target 40 to 70% wins).
		}
	}
}

func TestPhase35StartingClassMeasurements(t *testing.T) {
	if os.Getenv("ASHVEIL_BALANCE") != "1" {
		t.Skip("opt-in measurements")
	}
	newBrawl(t)
	archetypes.SetProvider(balanceHPProvider(t))
	now := configs.GetGamePlayConfig()
	old := now
	old.Progression.SmoothStatGrowth = false
	old.Progression.StatPointsEveryNLevels = 5
	old.Progression.HPFullLevels = 10
	old.Progression.HPAfterFull = 1.3
	for _, arch := range []string{"warrior", "rogue", "wizard", "cleric", "ranger"} {
		for _, level := range []int{1, 5, 10, 20, 30} {
			snapshot := func(g configs.GamePlay) (int, []int) {
				undo := configs.SetTestGamePlayConfig(g)
				defer undo()
				c := characters.New()
				c.RaceId = 1
				c.HPArchetype = arch
				c.Level = level
				c.Validate()
				return c.HealthMax.Value, []int{c.Stats.Strength.ValueAdj, c.Stats.Speed.ValueAdj, c.Stats.Smarts.ValueAdj, c.Stats.Vitality.ValueAdj, c.Stats.Mysticism.ValueAdj, c.Stats.Perception.ValueAdj}
			}
			oldHP, oldStats := snapshot(old)
			hp, stats := snapshot(now)
			t.Logf("CLASS | %s | %d | %d -> %d | %v -> %v |", arch, level, oldHP, hp, oldStats, stats)
		}
	}
	c := characters.New()
	c.RaceId, c.HPArchetype = 1, "warrior"
	var previous [6]int
	var changedLevels []int
	for level := 1; level <= 60; level++ {
		c.Level = level
		c.Validate()
		current := [6]int{c.Stats.Strength.ValueAdj, c.Stats.Speed.ValueAdj, c.Stats.Smarts.ValueAdj, c.Stats.Vitality.ValueAdj, c.Stats.Mysticism.ValueAdj, c.Stats.Perception.ValueAdj}
		if level > 1 && current != previous {
			changedLevels = append(changedLevels, level)
		}
		previous = current
	}
	t.Logf("HUMAN automatic adjusted stats change on %d/59 level-ups: %v", len(changedLevels), changedLevels)
}
