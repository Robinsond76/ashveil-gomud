package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39a wiring: the Halberdier's Sweep, Brace and Hook through the
// real strategy pass, ability pass and combat round (shipped config,
// DoCombat), in the brawl world. Tamsin is the Halberdier: a glaive in
// hand, every other member's abilities off, every blow landing.

// halberdBrawl is a brawl whose first companion is a level-level Halberdier
// with the shipped glaive; the fight has begun on the captain.
func halberdBrawl(t *testing.T, level int, before ...string) (*brawl, *[]combatstream.Event) {
	t.Helper()
	b, stream := abilityBrawl(t, map[int]string{1: "halberdier", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.HPArchetype = "halberdier"          // the company runtime sets it when a companion spawns
	tamsin.Character.Equipment.Weapon = items.New(10151) // a militia glaive
	tamsin.Character.Equipment.Offhand = items.Item{}
	for _, c := range before {
		b.cmd("strategy", c)
	}
	b.start()
	// The bandits' formation is ranked by toughness and settles once they
	// are hardened: aim Tamsin at the front row's first foe.
	b.aimTamsin(0)
	return b, stream
}

// aimTamsin points Tamsin at the first foe of an enemy formation row.
func (b *brawl) aimTamsin(row int) {
	b.t.Helper()
	tamsin := b.companion(1)
	var any *mobs.Mob
	for _, m := range b.livingBandits() {
		any = m
		break
	}
	party, ok := enemyparty.PartyOf(b.road, any.InstanceId)
	require.True(b.t, ok)
	for col := 0; col < domain.FormationCols; col++ {
		if id, ok := mobparty.InstanceIdFromMemberKey(party.Formation.At(row, col)); ok {
			tamsin.Character.SetAggro(0, id, characters.DefaultAttack)
			return
		}
	}
	b.t.Fatalf("no foe in row %d", row)
}

// rowOf is the foes standing in a foe's row of the enemy formation.
func (b *brawl) rowOf(foe *mobs.Mob) []int {
	b.t.Helper()
	party, ok := enemyparty.PartyOf(b.road, foe.InstanceId)
	require.True(b.t, ok)
	row, _, found := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	require.True(b.t, found)
	var out []int
	for col := 0; col < domain.FormationCols; col++ {
		if id, ok := mobparty.InstanceIdFromMemberKey(party.Formation.At(row, col)); ok {
			if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 {
				out = append(out, id)
			}
		}
	}
	return out
}

// halberdTargets are the targets of a member's weapon blows in a round's events.
func halberdTargets(stream []combatstream.Event, source string) []int {
	var out []int
	for _, e := range swingsBy(stream, source) {
		out = append(out, e.Target.MobInstanceId)
	}
	return out
}

func TestHalberdierDefaultsToTheCrowdedRowAndSweeps(t *testing.T) {
	b, stream := halberdBrawl(t, 1)
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)
	row := b.rowOf(foe)
	require.GreaterOrEqual(t, len(row), 2, "its aim stands in a crowded row")

	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	require.Len(t, abilityEvents(round, "Tamsin Reed"), 1)
	assert.Equal(t, "Sweep", abilityEvents(round, "Tamsin Reed")[0].Status)
	assert.Regexp(t, `Tamsin Reed sweeps a weapon across the row\. \(sweep\)`, out)
	hit := halberdTargets(round, "Tamsin Reed")
	assert.Len(t, hit, 2, "level 1: the foe and one beside it")
	assert.Equal(t, foe.InstanceId, hit[0], "the aim is struck first")
	assert.NotEqual(t, hit[0], hit[1])
	assert.Subset(t, row, hit, "both stand in the aim's row")

	// The sweep rests three rounds; she swings (or braces) meanwhile.
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hardenBandits()
		n = len(*stream)
		b.fight()
		for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
			assert.NotEqual(t, "Sweep", e.Status, "cooldown, round %d", i+2)
		}
	}
	b.toughen()
	b.hardenBandits()
	n = len(*stream)
	b.fight()
	var sweeps int
	for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
		if e.Status == "Sweep" {
			sweeps++
		}
	}
	assert.Equal(t, 1, sweeps, "ready again on the fourth round")
}

func TestHalberdierSweepReachesTheWholeRowFromLevelEight(t *testing.T) {
	b, stream := halberdBrawl(t, 8)
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)
	row := b.rowOf(foe)
	require.GreaterOrEqual(t, len(row), 3, "the bandits' front row holds three")
	b.hardenBandits()
	n := len(*stream)
	b.fight()
	hit := halberdTargets(since(*stream, n), "Tamsin Reed")
	assert.ElementsMatch(t, row, hit, "every foe in the row is struck once")
}

func TestASweepDefendsSeparatelyAndDoesNotMissTheRow(t *testing.T) {
	// A foe that is down cannot be swept off-turn: only standing foes count.
	b, stream := halberdBrawl(t, 8)
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	row := b.rowOf(foe)
	require.GreaterOrEqual(t, len(row), 3)
	// One of the row falls before the round.
	last := mobs.GetInstance(row[len(row)-1])
	if last.InstanceId == foe.InstanceId {
		last = mobs.GetInstance(row[0])
	}
	last.Character.Health = 0
	b.hardenBandits()
	last.Character.Health = 0
	n := len(*stream)
	b.fight()
	hit := halberdTargets(since(*stream, n), "Tamsin Reed")
	assert.NotContains(t, hit, last.InstanceId, "the fallen are not struck")
	assert.GreaterOrEqual(t, len(hit), 2)
}

func TestAHalberdierBracesAndAnswersTheFirstBlow(t *testing.T) {
	b, stream := halberdBrawl(t, 3)
	tamsin := b.companion(1)
	b.hardenBandits()
	b.fight() // round one: the sweep
	// Every bandit now strikes her.
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	}
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	var braces int
	for _, e := range abilityEvents(round, "Tamsin Reed") {
		if e.Status == "Brace" {
			braces++
		}
	}
	require.Equal(t, 1, braces, "the sweep rests and a foe is striking her: she braces")
	assert.Contains(t, out, "Tamsin Reed sets a weapon and braces for the first blow. (brace)")
	assert.Contains(t, out, "braced weapon meets")
	// The held blow is her only blow of the round, struck during the
	// enemy's turn.
	assert.Len(t, swingsBy(round, "Tamsin Reed"), 1)
	assert.False(t, tamsin.Character.RT.Brace, "the held blow is spent")
}

func TestABraceIsLostWhenNoFoeStrikesIt(t *testing.T) {
	b, stream := halberdBrawl(t, 3)
	tamsin := b.companion(1)
	b.hardenBandits()
	b.fight() // the sweep
	// The foes strike someone else.
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(3).InstanceId, characters.DefaultAttack)
	}
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	b.fight()
	for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
		assert.NotEqual(t, "Brace", e.Status, "no foe strikes her column: nothing to brace for")
	}
	assert.NotNil(t, tamsin.Character)
}

func TestAGlaiveHooksALeapingFoe(t *testing.T) {
	b, _ := halberdBrawl(t, 6, "tamsin abilities off")
	t.Cleanup(hooks.UseHookRollForTest(func(int) int { return 0 }))
	tamsin := b.companion(1)
	require.NotNil(t, tamsin)
	for _, m := range b.livingBandits() { // wolves, all
		m.Leap = true
	}
	b.hardenBandits()
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed hooks")
	assert.Contains(t, out, "(hook, knocked down)")
	down := 0
	for _, m := range b.livingBandits() {
		if status.Live(&m.Character, status.KnockedDown) {
			down++
		}
	}
	assert.Equal(t, 1, down, "only the foe she struck")
}

func TestAHalberdierBelowLevelSixDoesNotHook(t *testing.T) {
	b, _ := halberdBrawl(t, 5, "tamsin abilities off")
	t.Cleanup(hooks.UseHookRollForTest(func(int) int { return 0 }))
	for _, m := range b.livingBandits() {
		m.Leap = true
	}
	b.hardenBandits()
	assert.NotContains(t, b.fight(), "hooks")
}

// A player Halberdier sweeps by the polearm skill, through the player's
// own blow path.
func TestAPlayerHalberdierSweeps(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.withArchetypesFor("halberdier", nil)
	for _, who := range []string{"tamsin", "garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	aria := b.aria.Character
	aria.SetSkill("polearm", 1)
	aria.Equipment.Weapon = items.New(10151)
	aria.Equipment.Offhand = items.Item{}
	b.start()
	b.hardenBandits()
	out := b.fight()
	round := since(*stream, 0)
	assert.Regexp(t, `You sweep your weapon across the .* and the foes beside it\. \(sweep\)`, out)
	require.NotEmpty(t, abilityEvents(round, "Aria"))
	assert.Equal(t, "Sweep", abilityEvents(round, "Aria")[0].Status)
	assert.GreaterOrEqual(t, len(swingsBy(round, "Aria")), 2, "the sweep's blows are Aria's turn")
}

// A Valkyrie's Charged Sweep spends 8 mana for lightning on every foe the
// sweep strikes; without the mana it is an ordinary sweep.
func TestAValkyriesChargedSweepSpendsManaForLightning(t *testing.T) {
	b, stream := halberdBrawl(t, 10)
	tamsin := b.companion(1)
	tamsin.Character.SetClassState("valkyrie", nil)
	tamsin.Character.ManaMax.Value, tamsin.Character.Mana = 40, 20
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	struck := halberdTargets(since(*stream, n), "Tamsin Reed")
	assert.Equal(t, 12, tamsin.Character.Mana, "8 mana spent")
	assert.Contains(t, out, "(charged sweep)")
	assert.Regexp(t, `Lightning leaps from .* into the .*\. \(lightning, \d+ damage\)`, out)
	assert.Equal(t, len(struck), strings.Count(out, "(lightning,"), "lightning on every foe struck, every blow landing")

	// Later, with too little mana, the sweep is plain.
	tamsin.Character.Mana = 5
	out = ""
	for i := 0; i < 4; i++ {
		b.toughen()
		b.hardenBandits()
		out += b.fight()
	}
	assert.Contains(t, out, "(sweep)")
	assert.NotContains(t, out, "(charged sweep)")
	assert.Equal(t, 5, tamsin.Character.Mana)
}

// A Vanguard's held blow knocks the foe it answers down.
func TestAVanguardsBraceKnocksTheFoeDown(t *testing.T) {
	b, stream := halberdBrawl(t, 10)
	tamsin := b.companion(1)
	tamsin.Character.SetClassState("vanguard", nil)
	b.hardenBandits()
	b.fight() // the sweep
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
	}
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	require.Contains(t, out, "braced weapon meets")
	assert.Contains(t, out, "The held blow knocks")
	down := 0
	for _, m := range b.livingBandits() {
		if status.Live(&m.Character, status.KnockedDown) {
			down++
		}
	}
	assert.Equal(t, 1, down, "the foe that struck first is down")
	assert.Len(t, halberdTargets(since(*stream, n), "Tamsin Reed"), 1)
}

// Brace is the whole turn and Sweep is tried first: a halberdier whose
// abilities are off swings as anyone does.
func TestAHalberdierWithAbilitiesOffSwingsPlainly(t *testing.T) {
	b, stream := halberdBrawl(t, 8, "tamsin abilities off")
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	assert.Empty(t, abilityEvents(round, "Tamsin Reed"))
	assert.Len(t, halberdTargets(round, "Tamsin Reed"), 1)
	assert.NotContains(t, out, "sweeps a weapon")
}
