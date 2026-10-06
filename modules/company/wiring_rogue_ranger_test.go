package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38c2 wiring: the rogue and ranger elites through the real ability
// pass, gates and combat round (shipped config, DoCombat), in the brawl
// world. Tamsin is the rogue (a dagger in hand), Ysolde the ranger (her
// sling); the others use no abilities, and every blow lands.

// eliteBrawl is an ability brawl with Tamsin a rogue of class and level and
// Ysolde a ranger of class and level (either "" for the base archetype).
func eliteBrawl(t *testing.T, rogue string, rogueLevel int, ranger string, rangerLevel int) (*brawl, *[]combatstream.Event) {
	t.Helper()
	b, stream := abilityBrawl(t, map[int]string{1: "rogue", 2: "cleric", 3: "warrior", 4: "ranger"})
	hooks.ResetEliteForTest()
	t.Cleanup(hooks.ResetEliteForTest)
	for _, who := range []string{"garrick", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = rogueLevel
	tamsin.Character.HPArchetype = "rogue"
	tamsin.Character.Equipment.Weapon = items.New(10004) // a dagger
	tamsin.Character.Equipment.Offhand = items.Item{}
	if rogue != "" {
		tamsin.Character.SetClassState(rogue, nil)
	}
	ysolde := b.companion(4)
	ysolde.Character.Level = rangerLevel
	ysolde.Character.HPArchetype = "ranger"
	if ranger != "" {
		ysolde.Character.SetClassState(ranger, nil)
	}
	return b, stream
}

func statusesOf(stream []combatstream.Event, source string) []string {
	var out []string
	for _, e := range abilityEvents(stream, source) {
		out = append(out, e.Status)
	}
	return out
}

// afterAmbush plays the battle's two ambush rounds, then makes the aimed foe
// one that has not acted yet (a latecomer), as the Eye reads it.
func afterAmbush(b *brawl, stream *[]combatstream.Event) *mobs.Mob {
	for i := 0; i < 2; i++ {
		b.fight()
		b.toughen()
		b.hardenBandits()
	}
	foe := mobs.GetInstance(aimOf(&b.companion(1).Character))
	if foe != nil && foe.Character.RT != nil {
		foe.Character.RT.ActedRound = 0
	}
	return foe
}

func TestPathfindersEyeOpensAFoeThatHasNotActedOnce(t *testing.T) {
	b, stream := eliteBrawl(t, "pathfinder", 30, "", 1)
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	require.NotNil(t, afterAmbush(b, stream))
	n := len(*stream)
	out := b.fight()
	assert.Equal(t, []string{"Opening Strike"}, statusesOf(since(*stream, n), "Tamsin Reed"), "an opening on a foe that has not acted")
	assert.Regexp(t, `Tamsin Reed sees an opening on the .*\. \(opening strike\)`, out)
	assert.Equal(t, 1, b.companion(1).Character.RT.OpensUsed)

	// Once a battle: the foe has acted now and nothing else opens it.
	b.toughen()
	b.hardenBandits()
	n = len(*stream)
	b.fight()
	assert.Empty(t, statusesOf(since(*stream, n), "Tamsin Reed"))
}

func TestAScoutHasNoEyeForAFoeThatHasNotActed(t *testing.T) {
	b, stream := eliteBrawl(t, "scout", 29, "", 1)
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	require.NotNil(t, afterAmbush(b, stream))
	n := len(*stream)
	b.fight()
	assert.Empty(t, statusesOf(since(*stream, n), "Tamsin Reed"), "a scout needs a weakened foe once the ambush is over")
}

func TestNightbladeMarksItsAimAndTheMarkPassesWhenItFalls(t *testing.T) {
	b, _ := eliteBrawl(t, "nightblade", 35, "", 1)
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	tamsin := b.companion(1)
	aim := aimOf(&tamsin.Character)
	require.NotZero(t, aim)
	b.fight()
	rt := tamsin.Character.RT
	require.NotNil(t, rt)
	require.NotNil(t, rt.DeathMark, "the aim is marked at the battle's start")
	foe := mobs.GetInstance(aim)
	require.NotNil(t, foe)
	assert.Same(t, foe.Character.RT, rt.DeathMark)

	foe.Character.Health = 0 // it falls
	b.toughen()
	b.fight()
	if living := b.livingBandits(); len(living) > 0 {
		require.NotNil(t, rt.DeathMark, "the mark passes to another foe")
		assert.NotSame(t, foe.Character.RT, rt.DeathMark)
	}
}

func TestEliteRogueAndRangerRanksDoNotApplyBelowTheirLevel(t *testing.T) {
	b, _ := eliteBrawl(t, "nightblade", 29, "ravager", 29)
	b.cmd("strategy", "ysolde abilities off")
	b.start()
	b.fight()
	if rt := b.companion(1).Character.RT; rt != nil {
		assert.Nil(t, rt.DeathMark)
	}
}

func TestSentinelHoldsOverwatchAndShootsTheFirstFoeThatStrikesTheBackRow(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "sentinel", 30)
	for _, mv := range []string{"move #1 1 1", "move me 1 2", "move #3 2 1", "move #2 2 3", "move #4 3 3"} {
		require.Contains(t, b.cmd("formation", mv), "Placed")
	}
	b.cmd("strategy", "tamsin abilities off")
	b.start()
	// Oswin stands in the middle row of a column with no front: exposed.
	ysolde := b.companion(4)
	ysolde.Character.Equipment.Weapon = items.New(10014) // a sling
	sawHold, sawShot := false, false
	for i := 0; i < 6 && !(sawHold && sawShot); i++ {
		b.toughen()
		b.hardenBandits()
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, b.companion(2).InstanceId, characters.DefaultAttack) // every foe goes for Oswin
		}
		n := len(*stream)
		out := b.fight()
		round := since(*stream, n)
		for _, s := range statusesOf(round, "Ysolde") {
			if s == "Overwatch" {
				sawHold = true
			}
		}
		if strings.Contains(out, "overwatch arrow") {
			sawShot = true
		}
	}
	assert.True(t, sawHold, "the Sentinel holds a turn on overwatch")
	assert.True(t, sawShot, "and an overwatch arrow flies at a foe striking the line")
}

// rangerRounds plays rounds with the aimed foe of Ysolde wounded, so a
// Ravager's hunt has a foe to open.
func rangerRounds(b *brawl, stream *[]combatstream.Event, rounds int, wound func()) string {
	var out strings.Builder
	for i := 0; i < rounds; i++ {
		b.toughen()
		b.hardenBandits()
		if wound != nil {
			wound()
		}
		out.WriteString(b.fight())
	}
	return out.String()
}

func TestRavagerHuntsAWoundedFoeIntoBleeding(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "ravager", 30)
	for _, who := range []string{"tamsin"} {
		b.cmd("strategy", who+" abilities off")
	}
	b.start()
	ysolde := b.companion(4)
	ysolde.Character.Equipment.Weapon = items.New(10014)
	out := rangerRounds(b, stream, 4, func() {
		if foe := mobs.GetInstance(aimOf(&ysolde.Character)); foe != nil {
			foe.Character.Health = foe.Character.HealthMax.Value / 3
		}
	})
	assert.Regexp(t, `Ysolde (opens|deepens) the wound on`, out)
	assert.Contains(t, out, "hunt down: bleeding")
}

// isolate leaves Ysolde alone on a one-health foe: everyone else turns on the
// captain, so her shot is the kill.
func isolate(b *brawl) func() {
	ysolde := b.companion(4)
	return func() {
		var victim, captain *mobs.Mob
		for _, m := range b.livingBandits() {
			if strings.Contains(m.Character.Name, "cutthroat") && victim == nil {
				victim = m
			}
			if strings.Contains(m.Character.Name, "captain") {
				captain = m
			}
		}
		if victim == nil || captain == nil {
			return
		}
		victim.Character.Health = 1
		aim := func(c *characters.Character, id int) {
			if aimOf(c) != id { // a re-aim would restart a sling's wait
				c.SetAggro(0, id, characters.DefaultAttack)
			}
		}
		aim(&ysolde.Character, victim.InstanceId)
		aim(b.aria.Character, captain.InstanceId)
		for id := 1; id <= 3; id++ {
			aim(&b.companion(id).Character, captain.InstanceId)
		}
	}
}

func TestRavagerApexShakesTheBandOnAKill(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "ravager", 60)
	b.cmd("strategy", "tamsin abilities off")
	b.start()
	b.companion(4).Character.Equipment.Weapon = items.New(10014)
	out := rangerRounds(b, stream, 5, isolate(b))
	assert.True(t, strings.Contains(out, "apex: morale check"), "the kill shakes the band")
}

func TestMarksmanSpendsItsPerfectShotOncePerBattle(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "marksman", 60)
	b.cmd("strategy", "tamsin abilities off")
	b.start()
	ysolde := b.companion(4)
	ysolde.Character.Equipment.Weapon = items.New(10014)
	out := rangerRounds(b, stream, 6, nil)
	assert.Contains(t, out, "(aimed shot)")
	require.NotNil(t, ysolde.Character.RT)
	assert.True(t, ysolde.Character.RT.ShotUsed, "the first Aimed Shot was the perfect one")
	assert.False(t, ysolde.Character.RT.ShotNow, "and it is cleared after the round")
}

func TestSecondNockLoosesAnotherArrowAfterAKilledAimedShot(t *testing.T) {
	b, stream := eliteBrawl(t, "", 1, "marksman", 55)
	b.cmd("strategy", "tamsin abilities off")
	b.start()
	b.companion(4).Character.Equipment.Weapon = items.New(10014)
	out := rangerRounds(b, stream, 6, isolate(b))
	assert.True(t, strings.Contains(out, "nocks a second arrow"), "a killed Aimed Shot sends another arrow")
}

func TestTrailwiseAndAmbushMasterAreCompanyGifts(t *testing.T) {
	b, _ := eliteBrawl(t, "pathfinder", 55, "", 1)
	gold, name := enemyparty.CompanyEffect(7, classes.TrailGold)
	assert.Equal(t, 15, gold)
	assert.Equal(t, "Tamsin Reed", name)
	loot, _ := enemyparty.CompanyEffect(7, classes.TrailLoot)
	assert.Equal(t, 10, loot)
	flip, _ := enemyparty.CompanyEffect(7, classes.AmbushFlip)
	assert.Zero(t, flip, "Ambush Master comes at 60")
	b.companion(1).Character.Level = 60
	flip, _ = enemyparty.CompanyEffect(7, classes.AmbushFlip)
	assert.Equal(t, 1, flip)
	b.companion(1).Character.Health = 0
	gold, _ = enemyparty.CompanyEffect(7, classes.TrailGold)
	assert.Zero(t, gold, "a fallen Pathfinder gives nothing")
}

// alwaysParry makes every parry-capable defender parry what it can.
func alwaysParry(t *testing.T) {
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
}

// ripostesIn counts a rogue's ripostes over some rounds in which every foe
// strikes Tamsin and she parries all she can.
func ripostesIn(t *testing.T, class string, level, rounds int) (perRound []int) {
	b, _ := eliteBrawl(t, class, level, "", 1)
	for _, who := range []string{"tamsin", "ysolde"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysParry(t)
	b.start()
	tamsin := b.companion(1)
	tamsin.Character.Equipment.Weapon = items.New(10004)
	for i := 0; i < rounds; i++ {
		b.toughen()
		b.hardenBandits()
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
		}
		out := b.fight()
		perRound = append(perRound, strings.Count(out, "turns the parry into a riposte"))
	}
	return perRound
}

func TestSwordmasterRipostesTwiceARoundFromRankFortyFive(t *testing.T) {
	most := func(counts []int) int {
		m := 0
		for _, c := range counts {
			m = max(m, c)
		}
		return m
	}
	for _, tc := range []struct {
		class string
		level int
		want  int
	}{{"duelist", 25, 1}, {"swordmaster", 44, 1}, {"swordmaster", 45, 2}} {
		t.Run(fmt.Sprint(tc.class, tc.level), func(t *testing.T) {
			assert.Equal(t, tc.want, most(ripostesIn(t, tc.class, tc.level, 4)))
		})
	}
}

func TestStrategyListsASentinelsOverwatchOnlyOnceItHasIt(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
		want  bool
	}{{"sentinel", 30, true}, {"sentinel", 29, false}, {"warden", 30, false}} {
		t.Run(fmt.Sprint(tc.class, tc.level), func(t *testing.T) {
			w, store := classBrawl(t, tc.level, 50)
			w.withArchetypes("ranger")
			store.state.Class = tc.class
			view := w.cmd("strategy", "me")
			assert.Equal(t, tc.want, strings.Contains(view, "Overwatch"), view)
		})
	}
}

// ambushAgainst spawns a real ambush on the company, the first dice
// (Detection's) always an ambush, and the Eye's own coin the given roll.
func ambushAgainst(t *testing.T, class string, level, eyeRoll int) (advantage int, observer string) {
	t.Helper()
	b := battlefieldBrawl(t)
	b.aria.Character.Stats.Perception.ValueAdj = 0
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Stats.Perception.ValueAdj = 0
	}
	scout := b.companion(1)
	scout.Character.Level = level
	scout.Character.SetClassState(class, nil)
	calls := 0
	t.Cleanup(enemyparty.UseAmbushRollForTest(func(int) int {
		calls++
		if calls == 1 {
			return 99
		}
		return eyeRoll
	}))
	first, err := enemyparty.SpawnAmbush(b.road.RoomId, 9106, 7)
	require.NoError(t, err)
	foe := mobs.GetInstance(first)
	require.NotNil(t, foe)
	return foe.AmbushAdvantage, foe.AmbushObserver
}

func TestPathfindersEyeHalvesAmbushesAndAmbushMasterTurnsThemAround(t *testing.T) {
	t.Run("scout", func(t *testing.T) {
		adv, _ := ambushAgainst(t, "scout", 29, 10)
		assert.Equal(t, -1, adv, "a scout's company is ambushed as ever")
	})
	t.Run("eye catches half", func(t *testing.T) {
		adv, who := ambushAgainst(t, "pathfinder", 30, 10)
		assert.Zero(t, adv, "the coin is under 50: no one is surprised")
		assert.Equal(t, "Tamsin Reed", who)
	})
	t.Run("eye misses half", func(t *testing.T) {
		adv, _ := ambushAgainst(t, "pathfinder", 30, 80)
		assert.Equal(t, -1, adv)
	})
	t.Run("ambush master", func(t *testing.T) {
		adv, who := ambushAgainst(t, "pathfinder", 60, 80)
		assert.Equal(t, 1, adv, "the company ambushes instead")
		assert.Equal(t, "Tamsin Reed", who)
	})
}
