package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/formationcombat"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39i wiring: the neutral elites (Halberdier, Samurai, Shaman and Doll
// Master lineages) through the real strategy pass, ability pass, spell
// scripts and combat round (shipped config, DoCombat), in the brawl world.
// Each test fires a signature or capstone in a real round, and a neighbour
// without the rank to show it is the rank that did it.

// fresh runs f as a subtest of its own: a brawl keeps its world and its
// listeners until its test ends, so each one a comparison builds gets a test.
func fresh(t *testing.T, name string, f func(t *testing.T)) {
	t.Helper()
	t.Run(name, f)
}

// placeRows stands the company in the formation: the leader and the first
// two companions in the front row, the other two behind them.
func (b *brawl) placeRows() {
	b.t.Helper()
	rec, _ := module.registry.Get(7)
	f := domain.Formation{}
	for i, key := range []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(1), domain.CompanionMemberKey(2), domain.CompanionMemberKey(3), domain.CompanionMemberKey(4)} {
		row, col := 0, i
		if i >= 3 {
			row, col = 1, i-3
		}
		require.NoError(b.t, f.Place(key, row, col))
	}
	rec.Formation = f
	module.registry.Put(rec)
}

// noAbilities keeps the company-mates from using abilities of their own.
func (b *brawl) noAbilities() {
	b.t.Helper()
	for _, who := range []string{"tamsin", "oswin", "garrick", "ysolde"} {
		b.cmd("strategy", who+" abilities off")
	}
}

// foesInRow are the living foes standing in a row of the bandits' formation.
func (b *brawl) foesInRow(row int) []int {
	b.t.Helper()
	var any *mobs.Mob
	for _, m := range b.livingBandits() {
		any = m
		break
	}
	party, ok := enemyparty.PartyOf(b.road, any.InstanceId)
	require.True(b.t, ok)
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

// ----- Halberdier -----

func TestReaperSweepAlsoStrikesTheRowBehind(t *testing.T) {
	var struck, front, behind []int
	fresh(t, "reaper", func(t *testing.T) {
		b, stream := halberdBrawl(t, 30)
		b.companion(1).Character.SetClassState("reaper", nil)
		require.Equal(t, 50, b.companion(1).Character.ClassEffects().Int(classes.SweepBehind))
		b.hardenBandits() // the bandits' formation settles once they are hardened
		b.aimTamsin(0)
		front, behind = b.foesInRow(0), b.foesInRow(1)
		require.GreaterOrEqual(t, len(front), 2)
		require.NotEmpty(t, behind, "the bandits have a second row")
		n := len(*stream)
		b.fight()
		struck = halberdTargets(since(*stream, n), "Tamsin Reed")
	})
	assert.ElementsMatch(t, append(append([]int{}, front...), behind...), struck, "the aim's row and the row behind, each once")

	fresh(t, "sweeper", func(t *testing.T) {
		b, stream := halberdBrawl(t, 30)
		b.companion(1).Character.SetClassState("sweeper", nil)
		b.hardenBandits()
		b.aimTamsin(0)
		front = b.foesInRow(0)
		n := len(*stream)
		b.fight()
		assert.ElementsMatch(t, front, halberdTargets(since(*stream, n), "Tamsin Reed"), "no reach behind without the rank")
	})
}

func TestReaperHarvestSweepsEveryOtherRound(t *testing.T) {
	b, stream := halberdBrawl(t, 60)
	b.companion(1).Character.SetClassState("reaper", nil)
	sweeps := 0
	for i := 0; i < 6; i++ {
		b.toughen()
		b.hardenBandits()
		n := len(*stream)
		b.fight()
		for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
			if e.Status == "Sweep" {
				sweeps++
			}
		}
	}
	assert.GreaterOrEqual(t, sweeps, 3, "Harvest: ready every other round, six rounds give three sweeps")
}

func TestLinebreakerShieldsItsColumn(t *testing.T) {
	fresh(t, "linebreaker", func(t *testing.T) {
		b, _ := halberdBrawl(t, 30, "tamsin abilities off")
		b.place(1) // Tamsin front row, middle column; Ysolde (2) stands in the middle column behind her
		tamsin := b.companion(1)
		tamsin.Character.SetClassState("linebreaker", nil)
		b.hardenBandits()
		b.fight()
		assert.Equal(t, 10, b.companion(2).Character.Aura.Resolve, "behind the Linebreaker in its column: 10% less damage")
		assert.Equal(t, 10, tamsin.Character.Aura.Resolve, "the Linebreaker itself")
		assert.Zero(t, b.companion(3).Character.Aura.Resolve, "another column")
		assert.Zero(t, b.aria.Character.Aura.Resolve, "another column")
	})
	fresh(t, "vanguard", func(t *testing.T) {
		b, _ := halberdBrawl(t, 30, "tamsin abilities off")
		b.place(1)
		b.companion(1).Character.SetClassState("vanguard", nil)
		b.hardenBandits()
		b.fight()
		assert.Zero(t, b.companion(2).Character.Aura.Resolve, "a Vanguard has no column cut")
	})
}

func TestLinebreakerBraceAnswersTwoFoesAtRankSixty(t *testing.T) {
	answers := func(level int) (n int) {
		fresh(t, fmt.Sprint("level ", level), func(t *testing.T) {
			b, _ := halberdBrawl(t, level)
			tamsin := b.companion(1)
			tamsin.Character.SetClassState("linebreaker", nil)
			b.hardenBandits()
			b.fight() // the sweep
			for _, m := range b.livingBandits() {
				m.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
			}
			b.toughen()
			b.hardenBandits()
			n = strings.Count(b.fight(), "braced weapon meets")
		})
		return n
	}
	assert.Equal(t, 1, answers(59), "one held blow before the capstone")
	assert.Equal(t, 2, answers(60), "Twin brace: the first two foes that strike are answered")
}

func TestTempestLancerLightningArcsToTheNextRow(t *testing.T) {
	fresh(t, "tempest lancer", func(t *testing.T) {
		b, stream := halberdBrawl(t, 30)
		tamsin := b.companion(1)
		tamsin.Character.SetClassState("tempest-lancer", nil)
		tamsin.Character.ManaMax.Value, tamsin.Character.Mana = 80, 60
		require.Equal(t, 50, tamsin.Character.ClassEffects().Int(classes.ChargedArc))
		b.hardenBandits()
		n := len(*stream)
		out := b.fight()
		struck := halberdTargets(since(*stream, n), "Tamsin Reed")
		assert.Contains(t, out, "(charged sweep)")
		assert.Regexp(t, `The lightning from Tamsin Reed's weapon arcs on into .*\. \(lightning arc, \d+ damage\)`, out)
		arcs := strings.Count(out, "(lightning arc,")
		assert.Positive(t, arcs)
		assert.LessOrEqual(t, arcs, len(struck), "at most one arc for each bolt")
	})
	fresh(t, "valkyrie", func(t *testing.T) {
		b, _ := halberdBrawl(t, 30)
		b.companion(1).Character.SetClassState("valkyrie", nil)
		b.companion(1).Character.ManaMax.Value, b.companion(1).Character.Mana = 80, 60
		b.hardenBandits()
		assert.NotContains(t, b.fight(), "(lightning arc,", "a Valkyrie of the same level has no arc")
	})
}

func TestStormstruckParalyzesAndBossesResist(t *testing.T) {
	run := func(name string, boss bool, roll int) (paralyzed int) {
		fresh(t, name, func(t *testing.T) {
			b, _ := halberdBrawl(t, 60)
			tamsin := b.companion(1)
			tamsin.Character.SetClassState("tempest-lancer", nil)
			tamsin.Character.ManaMax.Value, tamsin.Character.Mana = 300, 300
			loadStatusBuffs(t)
			freshEvents(t)
			l := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
			t.Cleanup(func() { events.UnregisterListener(events.Buff{}, l) })
			t.Cleanup(hooks.UseStormRollForTest(func(int) int { return roll }))
			for _, m := range b.livingBandits() {
				m.Boss = boss
			}
			b.hardenBandits()
			b.fight()
			events.ProcessEvents()
			for _, m := range b.livingBandits() {
				if status.Live(&m.Character, status.Paralyzed) {
					paralyzed++
				}
			}
		})
		return paralyzed
	}
	assert.Positive(t, run("lands", false, 0), "a bolt that strikes can lock the foe rigid")
	assert.Zero(t, run("misses", false, 99), "or not")
	assert.Zero(t, run("boss resists", true, 20), "a boss resists: 10% where others take 35%")
	assert.Positive(t, run("boss unlucky", true, 5), "but not always")
}

// ----- Samurai -----

func TestSwordSaintTwinDrawKeepsIaijutsuForASecondStrike(t *testing.T) {
	for _, tc := range []struct {
		level   int
		spent   bool // after the first swing
		strikes int
	}{{59, true, 1}, {60, false, 2}} {
		fresh(t, fmt.Sprint("level ", tc.level), func(t *testing.T) {
			b := samuraiRounds(t, tc.level, "sword-saint") // the opening round is her first swing
			rt := b.aria.Character.RT
			require.NotNil(t, rt)
			assert.Equal(t, tc.spent, rt.IaiSpent, "after the first swing")
			b.hold(nil)
			b.fight()
			assert.True(t, rt.IaiSpent, "spent by the second swing at the latest")
			assert.Equal(t, tc.strikes, rt.IaiStrikes)
		})
	}
}

func TestSwordSaintFocusBuildsToTwentyFive(t *testing.T) {
	fresh(t, "sword saint", func(t *testing.T) {
		b := samuraiRounds(t, 30, "sword-saint")
		b.aria.Character.RTState().Quiet = 20
		assert.Equal(t, 25, b.aria.Character.ClassCrit(), "Still mind: +5% a quiet round, up to +25%")
	})
	fresh(t, "kensai", func(t *testing.T) {
		b := samuraiRounds(t, 30, "kensai")
		b.aria.Character.RTState().Quiet = 20
		assert.Equal(t, 20, b.aria.Character.ClassCrit(), "a Kensai stops at +20%")
	})
}

func TestShogunGivesTheWholeCompanyAHeadStartAndItsRowABanner(t *testing.T) {
	shogun := func(t *testing.T, level int, class string) *brawl {
		b := guardBrawl(t, "tamsin fighter")
		b.placeRows()
		tamsin := b.companion(1)
		tamsin.Character.HPArchetype = "samurai"
		tamsin.Character.Level = level
		tamsin.Character.SetClassState(class, nil)
		return b
	}
	fresh(t, "shogun", func(t *testing.T) {
		b := shogun(t, 60, "shogun")
		assert.Equal(t, 15, b.companion(1).Character.ClassEffects().Int(classes.ScoutMeter))
		b.strike(0, false)
		b.fight()
		f, _ := domain.FormationFor(7)
		tRow, _, _ := f.Find(domain.CompanionMemberKey(1))
		banner, other := 0, 0
		for key, c := range map[domain.MemberKey]*characters.Character{
			domain.LeaderMemberKey:       b.aria.Character,
			domain.CompanionMemberKey(2): &b.companion(2).Character,
			domain.CompanionMemberKey(3): &b.companion(3).Character,
			domain.CompanionMemberKey(4): &b.companion(4).Character,
		} {
			row, _, ok := f.Find(key)
			if !ok {
				continue
			}
			if row == tRow {
				banner++
				assert.Equal(t, 5, c.Aura.Attack, "%s fights under the banner of war", key)
			} else {
				other++
				assert.Zero(t, c.Aura.Attack, "%s stands in another row", key)
			}
		}
		assert.Positive(t, banner)
		assert.Positive(t, other)
	})
	fresh(t, "hatamoto", func(t *testing.T) {
		b := shogun(t, 60, "hatamoto")
		b.strike(0, false)
		b.fight()
		assert.Zero(t, b.aria.Character.Aura.Attack, "a Hatamoto has no banner")
	})
}

func TestKenshiCannotBeKnockedDownWhenItStandsAlone(t *testing.T) {
	knockedDown := func(class string, alone bool) (down bool) {
		fresh(t, fmt.Sprint(class, " alone=", alone), func(t *testing.T) {
			b := samuraiRounds(t, 30, class)
			b.hold(nil)
			b.fight()
			if alone {
				for i := 1; i <= 4; i++ {
					b.companion(i).Character.Health = 0
				}
			}
			b.hold(nil)
			b.fight()
			loadStatusBuffs(t)
			b.aria.Character.AddBuff(status.KnockedDown, false, 3)
			down = status.Grounded(b.aria.Character)
		})
		return down
	}
	assert.False(t, knockedDown("kenshi", true), "Last stand: the last one standing keeps its feet")
	assert.True(t, knockedDown("kenshi", false), "with allies standing it falls like anyone")
	assert.True(t, knockedDown("ronin", true), "a Ronin has no Last stand")
}

// ----- Shaman -----

func TestTempestLordRainLastsTheWholeBattle(t *testing.T) {
	fresh(t, "tempest lord", func(t *testing.T) {
		b := shamanBrawl(t, "tempest-lord", 30, "rain", "lightning")
		b.startWitchFight()
		require.Equal(t, "rain", b.aria.Character.Aggro.SpellInfo.SpellId)
		out := b.roundsUntil(3, func() bool { return battle.WeatherOf(7).Kind == stormcraft.Rain })
		assert.Equal(t, stormcraft.Rain, battle.WeatherOf(7).Kind)
		assert.Contains(t, out, "(the whole battle, Lightning 50% stronger)")
		assert.Greater(t, battle.WeatherOf(7).Left, 20, "it will not run out in any battle")
		b.roundsUntil(8, func() bool { return false })
		assert.Equal(t, stormcraft.Rain, battle.WeatherOf(7).Kind, "eight rounds on it is still raining")
	})
	fresh(t, "stormcaller", func(t *testing.T) {
		b := shamanBrawl(t, "stormcaller", 30, "rain", "lightning")
		b.startWitchFight()
		b.roundsUntil(3, func() bool { return battle.WeatherOf(7).Kind == stormcraft.Rain })
		assert.LessOrEqual(t, battle.WeatherOf(7).Left, 5, "a Stormcaller's rain counts its rounds")
	})
}

func TestTempestLordLightningChainsThroughTheWholeRowAtRankSixty(t *testing.T) {
	fresh(t, "rank sixty", func(t *testing.T) {
		b := shamanBrawl(t, "tempest-lord", 60, "lightning")
		require.True(t, b.aria.Character.ClassEffects().Has(classes.ChainRow))
		b.startWitchFight()
		agg := b.aria.Character.Aggro
		require.NotNil(t, agg)
		require.Equal(t, "lightning", agg.SpellInfo.SpellId)
		// The foes settle into their formation as the round opens, so the row
		// itself is covered by the hooks test; here the bolt reaches past the
		// two foes a Stormcaller's chain stops at, or at least a second.
		assert.GreaterOrEqual(t, len(agg.SpellInfo.TargetMobInstanceIds), 2)
		out := b.roundsUntil(3, func() bool { return false })
		assert.Contains(t, out, "The bolt leaps on to")
	})
	fresh(t, "rank fifty-nine", func(t *testing.T) {
		b := shamanBrawl(t, "tempest-lord", 59, "lightning")
		b.startWitchFight()
		assert.Len(t, b.aria.Character.Aggro.SpellInfo.TargetMobInstanceIds, 2, "before the capstone: a primary and a second foe")
	})
}

func TestVeilMotherWeatherLastsSevenRounds(t *testing.T) {
	b := shamanBrawl(t, "veil-mother", 30, "callfog")
	require.Equal(t, 4, b.aria.Character.ClassEffects().Int(classes.WeatherLong))
	b.slingTheSlinger()
	b.startWitchFight()
	b.roundsUntil(3, func() bool { return b.marked(status.Fogbound) > 0 })
	assert.Equal(t, 8, b.livingBandits()[0].Character.GetBuffs(status.Fogbound)[0].TriggersInitial, "7 rounds, one more than it lasts")
}

func TestVeilMotherFogHidesTheBackRowFromExtendedReach(t *testing.T) {
	reachInFog := func(class string, row int, reach formationcombat.Reach) (got formationcombat.Reach) {
		fresh(t, fmt.Sprint(class, " row ", row, " reach ", reach), func(t *testing.T) {
			b := shamanBrawl(t, class, 60, "callfog")
			b.place(1) // the leader and two more in row 1, the last in row 2
			b.slingTheSlinger()
			b.startWitchFight()
			b.roundsUntil(3, func() bool { return b.marked(status.Fogbound) > 0 })
			require.Equal(t, stormcraft.Fog, battle.WeatherOf(7).Kind)
			f, _ := domain.FormationFor(7)
			var key domain.MemberKey
			for col := 0; col < domain.FormationCols && key == ""; col++ {
				key = f.At(row, col)
			}
			require.NotEmpty(t, key, "someone stands in row %d", row)
			got = hooks.FogReachForTest(b.aria, b.road, f, key, reach)
		})
		return got
	}
	assert.Equal(t, formationcombat.ReachNone, reachInFog("veil-mother", 2, formationcombat.ReachExtended), "extended reach can't find the back row in the fog")
	assert.Equal(t, formationcombat.ReachAny, reachInFog("veil-mother", 2, formationcombat.ReachAny), "a shooter's reach is untouched")
	assert.Equal(t, formationcombat.ReachExtended, reachInFog("veil-mother", 1, formationcombat.ReachExtended), "the middle row is not hidden")
	assert.Equal(t, formationcombat.ReachExtended, reachInFog("mistweaver", 2, formationcombat.ReachExtended), "a Mistweaver's fog hides no one")
}

func TestMountainSpeakerTremorKnocksTheFrontRowDownWhenStoneskinLands(t *testing.T) {
	run := func(class string, roll int) (down int, out string) {
		fresh(t, fmt.Sprint(class, " roll ", roll), func(t *testing.T) {
			b := shamanBrawl(t, class, 30, "stoneskin")
			b.noAbilities() // a tackle would knock the front row down first
			t.Cleanup(scripting.UseTremorRollForTest(func(int) int { return roll }))
			b.startWitchFight() // a one-round chant: the stone goes on in the opening round
			out = b.roundsUntil(2, func() bool { return false })
			events.ProcessEvents()
			for _, m := range b.livingBandits() {
				if status.Live(&m.Character, status.KnockedDown) {
					down++
				}
			}
		})
		return down, out
	}
	down, out := run("mountain-speaker", 0)
	assert.Positive(t, down, "the earth shook them off their feet")
	assert.Contains(t, out, "(tremor)", "the next round's Stoneskin finds the front row already down") // the opening round's line is not in out
	down, out = run("mountain-speaker", 99)
	assert.Zero(t, down)
	assert.Contains(t, out, "they keep their feet. (tremor)")
	down, out = run("earthspeaker", 0)
	assert.Zero(t, down, "an Earthspeaker has no tremor")
	assert.NotContains(t, out, "tremor")
}

func TestMountainSpeakerStoneCloakCoversARow(t *testing.T) {
	cloaked := func(level int) (n int) {
		fresh(t, fmt.Sprint("level ", level), func(t *testing.T) {
			b := shamanBrawl(t, "mountain-speaker", level, "stoneskin")
			b.placeRows()
			b.startWitchFight()
			for _, c := range []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character} {
				if c.RT != nil && c.RT.Bark > 0 {
					n++
				}
			}
		})
		return n
	}
	assert.Equal(t, 1, cloaked(59), "one ally before the capstone")
	assert.Greater(t, cloaked(60), 1, "Stone cloak: the target's whole row")
}

// ----- Doll Master -----

func TestGrandPuppeteerThirdDollJoinsAtRankSixty(t *testing.T) {
	count := func(level int) (n int) {
		fresh(t, fmt.Sprint("level ", level), func(t *testing.T) {
			b, _ := dollBrawl(t, level)
			b.companion(1).Character.SetClassState("grand-puppeteer", nil)
			b.hardenBandits()
			b.fight()
			master, ok := dolls.Of(7, domain.CompanionMemberKey(1))
			require.True(t, ok)
			for i := 0; i < 3; i++ {
				if _, standing := dolls.Live(master, i); standing {
					n++
				}
			}
		})
		return n
	}
	assert.Equal(t, 2, count(59), "two dolls before the capstone")
	assert.Equal(t, 3, count(60), "Third doll")
}

func TestGolemLordBlowsKnockFoesDown(t *testing.T) {
	run := func(class string, roll int) (down int) {
		fresh(t, fmt.Sprint(class, " roll ", roll), func(t *testing.T) {
			b, _ := dollBrawl(t, 30)
			b.companion(1).Character.SetClassState(class, nil)
			loadStatusBuffs(t)
			freshEvents(t)
			l := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
			t.Cleanup(func() { events.UnregisterListener(events.Buff{}, l) })
			t.Cleanup(hooks.UseDollKnockRollForTest(func(int) int { return roll }))
			b.hardenBandits()
			out := b.fight()
			events.ProcessEvents()
			if class == "golem-lord" && roll == 0 {
				assert.Contains(t, out, "(knocked down)")
			}
			for _, m := range b.livingBandits() {
				if status.Live(&m.Character, status.KnockedDown) {
					down++
				}
			}
		})
		return down
	}
	assert.Positive(t, run("golem-lord", 0), "Hammering blows")
	assert.Zero(t, run("golem-lord", 99))
	assert.Zero(t, run("golemancer", 0), "a Golemancer's blows only hit")
}

func TestGolemLordGolemStandsUpASecondTime(t *testing.T) {
	b, _ := dollBrawl(t, 60)
	tamsin := b.companion(1)
	tamsin.Character.SetClassState("golem-lord", nil)
	b.hardenBandits()
	b.fight()
	tamsin.Character.RT.Spliced = true // Emergency Splice is already spent
	golem := b.doll(0)
	golem.Character.Health = 0
	b.toughen()
	b.hardenBandits()
	out := b.fight()
	assert.Contains(t, out, "grinds back up from the rubble")
	golem = b.doll(0)
	assert.Greater(t, golem.Character.Health, 0)
	assert.True(t, tamsin.Character.RT.SplicedAgain)
	assert.False(t, tamsin.Character.Dolls[0].Broken)
	golem.Character.Health = 0 // a third fall breaks it
	b.toughen()
	b.hardenBandits()
	b.fight()
	assert.True(t, tamsin.Character.Dolls[0].Broken)
}

func TestStringSovereignCutsATangledFoesNextAttack(t *testing.T) {
	b, _ := dollBrawl(t, 30)
	tamsin := b.companion(1)
	tamsin.Character.SetClassState("string-sovereign", nil)
	require.Equal(t, 15, tamsin.Character.ClassEffects().Int(classes.TangleWeak))
	b.hardenBandits()
	out := b.fight()
	assert.Contains(t, out, "(tangled, -15 Attack on its next attack)")
	// The cut lifts after the foe's next blow, which lands later in the round.
	assert.Equal(t, 2, strings.Count(out, "(tangled, -15 Attack on its next attack)"), "two foes are snagged and cut")
}

func TestTangleSnagsAsManyFoesAsItsRouteSays(t *testing.T) {
	snagged := func(class string, level int) (n int) {
		fresh(t, class, func(t *testing.T) {
			b, _ := dollBrawl(t, level)
			b.companion(1).Character.SetClassState(class, nil)
			b.hardenBandits()
			n = strings.Count(b.fight(), "Strings from Tamsin Reed snag")
		})
		return n
	}
	assert.Equal(t, 2, snagged("marionettist", 20), "Nimble strings: two foes, as its help says")
	assert.Equal(t, 2, snagged("string-sovereign", 59), "two before the capstone")
	assert.Equal(t, 3, snagged("string-sovereign", 60), "Sovereign strings: three foes")
}
