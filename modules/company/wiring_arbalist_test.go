package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39h wiring: the Arbalist's Piercing Bolt and its winding through the
// real strategy pass, ability pass and combat round, in the brawl world.
// Tamsin is the arbalist: the shipped hunting crossbow in hand, every other
// member's abilities off, every blow landing.

const huntingCrossbow = 10181

func arbalistBrawl(t *testing.T, level int, class string, before ...string) (*brawl, *[]combatstream.Event) {
	t.Helper()
	b, stream := abilityBrawl(t, map[int]string{1: "arbalist", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.HPArchetype = "arbalist"
	tamsin.Character.HPClass = class
	tamsin.Character.Equipment.Weapon = items.New(huntingCrossbow)
	tamsin.Character.Equipment.Offhand = items.Item{}
	for _, c := range before {
		b.cmd("strategy", c)
	}
	b.cmd("company", "tactics focus none")
	b.start()
	return b, stream
}

func boltCount(stream []combatstream.Event, status string) int {
	n := 0
	for _, e := range abilityEvents(stream, "Tamsin Reed") {
		if e.Status == status {
			n++
		}
	}
	return n
}

// round runs one round and reports what Tamsin did in it: bolts, windings and
// plain shots.
func (b *brawl) arbalistRound(stream *[]combatstream.Event, foe *mobs.Mob) (bolts, reloads, swings int, out string) {
	b.t.Helper()
	b.toughen()
	b.hardenBandits()
	b.companion(1).Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
	n := len(*stream)
	out = b.fight()
	round := since(*stream, n)
	return boltCount(round, "Piercing Bolt"), boltCount(round, "Reload"), len(halberdTargets(round, "Tamsin Reed")), out
}

func TestAnArbalistLoosesABoltThenWindsTheCrossbow(t *testing.T) {
	b, stream := arbalistBrawl(t, 1, "")
	foe := b.livingBandits()[0]

	bolts, reloads, swings, out := b.arbalistRound(stream, foe)
	assert.Equal(t, 1, bolts)
	assert.Zero(t, reloads)
	assert.Equal(t, 1, swings, "the bolt is the turn's one blow")
	assert.Contains(t, out, "Tamsin Reed looses a heavy bolt at")
	assert.Contains(t, out, "(piercing bolt)")
	assert.True(t, b.companion(1).Character.RT.Reload, "the crossbow is unwound")

	bolts, reloads, swings, out = b.arbalistRound(stream, foe)
	assert.Zero(t, bolts)
	assert.Equal(t, 1, reloads, "the next turn is the winding")
	assert.Zero(t, swings, "no blow while winding")
	assert.Contains(t, out, "Tamsin Reed winds the crossbow for the next bolt. (reloading)")
	assert.False(t, b.companion(1).Character.RT.Reload)

	bolts, reloads, swings, _ = b.arbalistRound(stream, foe)
	assert.Equal(t, 1, bolts, "ready again on the third round")
	assert.Zero(t, reloads)
	assert.Equal(t, 1, swings)
	assert.Zero(t, b.companion(1).Character.RT.BlowPierce, "the piercing was for that blow alone")
}

// A landed bolt hobbles from level 6, and Siegebreaker's bolts wear armor
// down for the battle: 10 a bolt, stopping at 30.
func TestBoltRanksHobbleAndWearArmorDown(t *testing.T) {
	for _, tc := range []struct {
		name    string
		class   string
		level   int
		hobbled bool
		shred   int
	}{
		{"level 5", "", 5, false, 0},
		{"level 6: crippling", "", 6, true, 0},
		{"siegebreaker", "siegebreaker", 10, true, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, stream := arbalistBrawl(t, tc.level, tc.class)
			foe := b.livingBandits()[0]
			bolts, _, _, out := b.arbalistRound(stream, foe)
			require.Equal(t, 1, bolts)
			assert.Equal(t, tc.hobbled, status.Live(&foe.Character, status.Hobbled))
			if tc.hobbled {
				assert.Contains(t, out, "The bolt cripples")
			}
			shred := 0
			if foe.Character.RT != nil {
				shred = foe.Character.RT.Shred
			}
			assert.Equal(t, tc.shred, shred)
			if tc.shred > 0 {
				assert.Contains(t, out, "tears at")
			}
		})
	}
}

func TestSundersStackToTheirCap(t *testing.T) {
	b, stream := arbalistBrawl(t, 10, "siegebreaker")
	foe := b.livingBandits()[0]
	var shred []int
	for round := 0; round < 8; round++ {
		b.arbalistRound(stream, foe)
		shred = append(shred, foe.Character.RT.Shred)
	}
	// A bolt every third round (winding between, cooldown two): 10, 20, 30, 30.
	assert.Equal(t, 30, shred[len(shred)-1], "stops at the cap: %v", shred)
	assert.Equal(t, 10, shred[0])
}

// Practiced loader: the first bolt of a battle needs no winding, so the next
// turn is a plain shot; the second bolt is wound as usual.
func TestAPracticedLoaderFiresTheFirstBoltFree(t *testing.T) {
	b, stream := arbalistBrawl(t, 20, "sharpshooter")
	foe := b.livingBandits()[0]
	bolts, reloads, swings, _ := b.arbalistRound(stream, foe)
	assert.Equal(t, []int{1, 0, 1}, []int{bolts, reloads, swings})
	assert.False(t, b.companion(1).Character.RT.Reload)
	bolts, reloads, swings, _ = b.arbalistRound(stream, foe)
	assert.Equal(t, []int{0, 0, 1}, []int{bolts, reloads, swings}, "no winding: a plain shot while the bolt cools")
	bolts, reloads, swings, _ = b.arbalistRound(stream, foe)
	assert.Equal(t, []int{1, 0, 1}, []int{bolts, reloads, swings})
	assert.True(t, b.companion(1).Character.RT.Reload, "the second bolt is wound")
	bolts, reloads, swings, _ = b.arbalistRound(stream, foe)
	assert.Equal(t, []int{0, 1, 0}, []int{bolts, reloads, swings})
}

func TestAnArbalistThatCannotBoltDoesNotWind(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before []string
		prep   func(b *brawl)
	}{
		{"abilities off", []string{"tamsin abilities off"}, func(b *brawl) {}},
		{"a spear, not a crossbow", nil, func(b *brawl) { b.companion(1).Character.Equipment.Weapon = items.New(10131) }},
		{"unarmed", nil, func(b *brawl) { b.companion(1).Character.Equipment.Weapon = items.Item{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, stream := arbalistBrawl(t, 1, "", tc.before...)
			tc.prep(b)
			foe := b.livingBandits()[0]
			for i := 0; i < 3; i++ {
				bolts, reloads, _, _ := b.arbalistRound(stream, foe)
				assert.Zero(t, bolts+reloads, "round %d: %v", i+1, abilityEvents(*stream, "Tamsin Reed"))
			}
		})
	}
}

// The windup does not outlive the battle: a new fight starts loaded.
func TestTheWindingEndsWithTheFight(t *testing.T) {
	b, stream := arbalistBrawl(t, 1, "")
	foe := b.livingBandits()[0]
	b.arbalistRound(stream, foe)
	require.True(t, b.companion(1).Character.RT.Reload)
	b.companion(1).Character.EndFightRT()
	assert.False(t, b.companion(1).Character.RT.Reload)
}

func TestAPlayerArbalistLoosesABolt(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.withArchetypesFor("arbalist", nil)
	for _, who := range []string{"tamsin", "garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	aria := b.aria.Character
	aria.SetSkill("arbalestry", 1)
	aria.Equipment.Weapon = items.New(huntingCrossbow)
	aria.Equipment.Offhand = items.Item{}
	b.start()
	foe := b.livingBandits()[0]
	aria.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
	b.hardenBandits()
	out := b.fight()
	assert.Regexp(t, `You set a heavy bolt and loose it at .*\(piercing bolt\)`, out)
	assert.Equal(t, 1, len(abilityEvents(since(*stream, 0), "Aria")))
	n := len(*stream)
	out = b.fight()
	assert.Contains(t, out, "You wind the crossbow for the next bolt.")
	assert.Empty(t, halberdTargets(since(*stream, n), "Aria"), "no blow while winding")
}

// The arbalist's default aim is the most armored foe it can reach.
func TestAnArbalistsDefaultAimIsTheMostArmoredFoe(t *testing.T) {
	b, _ := arbalistBrawl(t, 1, "")
	list := b.cmd("strategy", "")
	assert.Regexp(t, `Tamsin Reed\s+arbalist\s+fighter\s+armored`, list)

	bandits := b.livingBandits()
	require.GreaterOrEqual(t, len(bandits), 2)
	for _, m := range bandits {
		m.Character.Equipment = bandits[0].Character.Equipment
		m.Character.Equipment.Body = items.Item{}
		m.Character.Equipment.Head = items.Item{}
		m.Character.Equipment.Legs = items.Item{}
		m.Character.Equipment.Feet = items.Item{}
	}
	heavy := bandits[len(bandits)-1]
	heavy.Character.Equipment.Body = items.New(20163) // a padded jack
	require.Greater(t, heavy.Character.GetDefense(), bandits[0].Character.GetDefense())
	b.hardenBandits()
	g, ok := enemyparty.GroupOf(b.road, heavy.InstanceId)
	require.True(t, ok)
	id, ok := enemyparty.Aim(g, enemyparty.CompanionAttacker(7, domain.CompanionMemberKey(1), b.companion(1), 0))
	require.True(t, ok)
	assert.Equal(t, heavy.InstanceId, id)
}
