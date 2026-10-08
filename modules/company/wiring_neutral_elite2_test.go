package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/classes"
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

// Phase 39i2 wiring: the Beast Tamer, Gryphon Rider, Alchemist and Arbalist
// elites through the real strategy pass, ability pass, spell scripts and
// combat round (shipped config, DoCombat), in the brawl world. Each test fires
// a signature or capstone in a real round, and a neighbour without the rank to
// show it is the rank that did it.

// ----- Beast Tamer -----

func TestPacklordHoundExposesAFoeItHasHobbled(t *testing.T) {
	// own: the hobble is the hound's own bite's (review: another's hobble,
	// an Arbalist's Crippling Bolt, must not count).
	exposes := func(t *testing.T, route string, own bool) (exposed bool, out string) {
		b, _ := beastBrawl(t, 30, route)
		b.hardenBandits()
		b.fight()
		for i := 0; i < 5 && !exposed; i++ {
			for _, m := range b.livingBandits() {
				require.NoError(t, m.Character.AddBuff(status.Hobbled, true))
				if own {
					m.Character.RTState().HobbledBy = b.beast().Character.RTState()
				} else {
					m.Character.RTState().HobbledBy = nil
				}
			}
			b.toughen()
			b.hardenBandits()
			out += b.fight()
			for _, m := range b.livingBandits() {
				exposed = exposed || status.Live(&m.Character, status.Exposed)
			}
		}
		return
	}
	fresh(t, "packlord", func(t *testing.T) {
		exposed, out := exposes(t, "packlord", true)
		assert.True(t, exposed, "a bite on a hobbled foe leaves it exposed\n%s", out)
		assert.Contains(t, out, "(exposed)")
	})
	fresh(t, "another's hobble", func(t *testing.T) {
		exposed, out := exposes(t, "packlord", false)
		assert.False(t, exposed, "a hobble the hound did not bite is no pack hunt\n%s", out)
		assert.NotContains(t, out, "lame leg")
	})
	fresh(t, "houndmaster", func(t *testing.T) {
		exposed, out := exposes(t, "houndmaster", true)
		assert.False(t, exposed, "no pack hunt without the rank\n%s", out)
		assert.NotContains(t, out, "lame leg")
	})
}

func TestPacklordHoundHobblesFoesBelowThreeQuartersAtRankFortyFive(t *testing.T) {
	hobbled := func(t *testing.T, level int) bool {
		b, _ := beastBrawl(t, level, "packlord")
		noCrits(t)
		b.hardenBandits()
		b.fight()
		for i := 0; i < 5; i++ {
			for _, m := range b.livingBandits() {
				// 70%: above half, below three quarters. The company's own
				// blows land before the hound's bite and wear the foe down, so
				// a foe set just over half could be bitten under it (the
				// level 44 case); 70% leaves 200 points of room.
				m.Character.Health = m.Character.HealthMax.Value * 7 / 10
			}
			b.toughen()
			if strings.Contains(b.fight(), "(hobbled)") {
				return true
			}
		}
		return false
	}
	fresh(t, "level 45", func(t *testing.T) { assert.True(t, hobbled(t, 45), "wider hobble") })
	fresh(t, "level 44", func(t *testing.T) { assert.False(t, hobbled(t, 44), "a foe at 70% is above half") })
}

func TestPacklordHoundStartsWithMeterToSpare(t *testing.T) {
	bites := func(t *testing.T, level int) int {
		b, stream := beastBrawlWith(t, level, "packlord", func(*brawl) {
			t.Cleanup(hooks.UseTempoForTest(func(*characters.Character) float64 { return 0.7 })) // the hound's own cadence, for all
		})
		b.hardenBandits()
		n := len(*stream)
		for i := 0; i < 2; i++ {
			b.toughen()
			b.hardenBandits()
			b.fight()
		}
		return len(swingsBy(since(*stream, n), "Fang"))
	}
	// Every fighter has a turn in the first round; a hound at 70% tempo has
	// none in the second, unless it started with meter to spare.
	fresh(t, "first strike", func(t *testing.T) {
		assert.Equal(t, 2, bites(t, 60), "50 points of meter: a turn in each of the first two rounds")
	})
	fresh(t, "without it", func(t *testing.T) { assert.Equal(t, 1, bites(t, 55), "a hound at 70% tempo has no turn in round two") })
}

func TestBeastlordBearSwipesTheFoeBesideItsTarget(t *testing.T) {
	swipes := func(t *testing.T, route string, level int) (out string, hurt int) {
		b, _ := beastBrawl(t, level, route)
		b.hardenBandits()
		b.fight()
		for i := 0; i < 4; i++ {
			b.toughen()
			b.hardenBandits()
			out += b.fight()
		}
		for _, m := range b.livingBandits() {
			if m.Character.Health < 1000 {
				hurt++
			}
		}
		return
	}
	fresh(t, "beastlord", func(t *testing.T) {
		out, hurt := swipes(t, "beastlord", 30)
		assert.Contains(t, out, "swipe rakes", out)
		assert.Contains(t, out, "(swipe,")
		assert.GreaterOrEqual(t, hurt, 2, "the bear's target and the foe beside it are hurt")
	})
	fresh(t, "bearward", func(t *testing.T) {
		out, _ := swipes(t, "bearward", 30)
		assert.NotContains(t, out, "swipe", "no swipe without the rank")
	})
}

func TestBeastlordBearStandsBackUpOnceABattle(t *testing.T) {
	fall := func(t *testing.T, route string) (out string, standing bool, wounded bool) {
		b, _ := beastBrawl(t, 60, route)
		b.hardenBandits()
		b.fight()
		bear := b.beast()
		bear.Character.Health = 0
		hooks.ResetAbilitiesForTest()
		b.toughen()
		b.hardenBandits()
		out = b.fight()
		tamer, _ := domainBeastTamer(b)
		_, standing = domain.BeastOf(bear.InstanceId)
		return out, standing, tamer.Character.Beast != nil && tamer.Character.Beast.Wounded
	}
	fresh(t, "beastlord", func(t *testing.T) {
		out, standing, wounded := fall(t, "beastlord")
		assert.Contains(t, out, "stands again", out)
		assert.True(t, standing, "the bear is on its feet")
		assert.False(t, wounded)
	})
	fresh(t, "bearward", func(t *testing.T) {
		out, standing, wounded := fall(t, "bearward")
		assert.Contains(t, out, "goes down, wounded")
		assert.False(t, standing)
		assert.True(t, wounded)
	})
}

func TestBeastlordBearStandsOnlyOnce(t *testing.T) {
	b, _ := beastBrawl(t, 60, "beastlord")
	b.hardenBandits()
	b.fight()
	bear := b.beast()
	var out string
	for i := 0; i < 2; i++ {
		bear.Character.Health = 0
		hooks.ResetAbilitiesForTest()
		b.toughen()
		b.hardenBandits()
		out += b.fight()
	}
	assert.Equal(t, 1, strings.Count(out, "shakes itself"), out)
	assert.Contains(t, out, "goes down, wounded", "the second fall is a wound")
}

func TestDragonLordBreathBurnsFoesAndReachesFour(t *testing.T) {
	run := func(t *testing.T, route string, level int) (out string, burning, hurt int) {
		b, _ := beastBrawl(t, level, route)
		b.hardenBandits()
		for i := 0; i < 4; i++ {
			b.toughen()
			b.hardenBandits()
			out += b.fight()
			for _, m := range b.livingBandits() {
				if status.Live(&m.Character, status.Burning) {
					burning++
				}
			}
		}
		for _, m := range b.livingBandits() {
			if m.Character.Health < 1000 {
				hurt++
			}
		}
		return
	}
	fresh(t, "dragon lord", func(t *testing.T) {
		out, burning, _ := run(t, "dragon-lord", 30)
		assert.Contains(t, out, "(breath")
		assert.Contains(t, out, ", burning)")
		assert.Positive(t, burning, "foes the Breath struck are alight")
	})
	fresh(t, "dragon tamer", func(t *testing.T) {
		out, burning, _ := run(t, "dragon-tamer", 30)
		assert.Contains(t, out, "(breath")
		assert.Zero(t, burning)
	})
	fresh(t, "wide breath", func(t *testing.T) {
		out, _, _ := run(t, "dragon-lord", 45)
		assert.Regexp(t, `\(breath, [4-9] foes`, out, "Breath reaches four foes at rank 45")
	})
}

func TestDragonLordFirstBreathComesInTheFirstRound(t *testing.T) {
	breaths := func(t *testing.T, level int) int {
		b, stream := beastBrawl(t, level, "dragon-lord")
		b.hardenBandits()
		n := len(*stream)
		b.toughen()
		b.fight()
		drake := b.beast()
		cnt := 0
		for _, e := range abilityEvents(since(*stream, n), drake.Character.Name) {
			if e.Status == "Breath" {
				cnt++
			}
		}
		return cnt
	}
	fresh(t, "furnace heart", func(t *testing.T) { assert.Equal(t, 1, breaths(t, 60)) })
	fresh(t, "without it", func(t *testing.T) { assert.Zero(t, breaths(t, 55), "the first Breath waits its full interval") })
}

// ----- Gryphon Rider -----

// eliteRider is a Gryphon Rider brawl whose rider has taken the class (at
// its level) and wields a weapon.
func eliteRider(t *testing.T, class string, level, weapon int) (*brawl, *[]combatstream.Event, *mobs.Mob) {
	t.Helper()
	b, stream := gryphonBrawl(t, level)
	loadShippedBuff(t, "13-poisoned.yaml")
	rider := b.companion(1)
	rider.Character.SetClassState(class, nil)
	rider.Character.Equipment.Weapon = items.New(weapon)
	foe := b.deepFoe()
	b.aimTamsinAt(foe)
	b.hardenBandits()
	return b, stream, foe
}

func TestGryphonLordDiveIsReadyEveryOtherRound(t *testing.T) {
	dives := func(t *testing.T, class string) int {
		b, stream, foe := eliteRider(t, class, 30, 10131)
		n := 0
		for i := 0; i < 6; i++ {
			b.toughen()
			b.hardenBandits()
			b.aimTamsinAt(foe)
			from := len(*stream)
			b.fight()
			n += diveCount(since(*stream, from), "Tamsin Reed")
		}
		return n
	}
	fresh(t, "gryphon lord", func(t *testing.T) { assert.Equal(t, 3, dives(t, "gryphon-lord"), "rounds 1, 3 and 5") })
	fresh(t, "gryphon knight", func(t *testing.T) { assert.Equal(t, 2, dives(t, "gryphon-knight"), "rounds 1 and 4") })
}

func TestGryphonLordThunderLandingKnocksDownTheFoesBesideTheTarget(t *testing.T) {
	down := func(t *testing.T, class string, level int) (n int, out string) {
		b, stream, _ := eliteRider(t, class, level, 10141) // a war spear: a lance
		out = b.fight()
		// The battle caption names it (review).
		out += fmt.Sprintf("\n[%d Thunder landing events]", boltCount(since(*stream, 0), "Thunder landing"))
		for _, m := range b.livingBandits() {
			if status.Live(&m.Character, status.KnockedDown) {
				n++
			}
		}
		return
	}
	fresh(t, "thunder landing", func(t *testing.T) {
		n, out := down(t, "gryphon-lord", 60)
		assert.GreaterOrEqual(t, n, 2, "the target and a foe beside it\n%s", out)
		assert.Contains(t, out, "The landing throws")
		assert.NotContains(t, out, "[0 Thunder landing events]")
	})
	fresh(t, "without it", func(t *testing.T) {
		n, out := down(t, "gryphon-lord", 55)
		assert.Equal(t, 1, n, "only the target\n%s", out)
		assert.NotContains(t, out, "The landing throws")
	})
}

func TestFalconMarshalMarksTheFoeItDivesOn(t *testing.T) {
	mark := func(t *testing.T, class string, level int) (int, string) {
		b, _, foe := eliteRider(t, class, level, 10131)
		out := b.fight()
		if foe.Character.RT == nil {
			return 0, out
		}
		return foe.Character.RT.Mark, out
	}
	fresh(t, "marshal 30", func(t *testing.T) {
		n, out := mark(t, "falcon-marshal", 30)
		assert.Equal(t, 5, n, out)
		assert.Contains(t, out, "(marked: +5 Attack")
	})
	fresh(t, "marshal 60", func(t *testing.T) {
		n, _ := mark(t, "falcon-marshal", 60)
		assert.Equal(t, 12, n)
	})
	fresh(t, "skyscout", func(t *testing.T) {
		n, out := mark(t, "skyscout", 30)
		assert.Zero(t, n)
		assert.NotContains(t, out, "marked")
	})
}

// The poisoned-foe bonus is a percentage on dice the round rolls elsewhere
// too, so no sum of damage separates it from noise (40 seeds still read the
// Lord under the Rider); the number is pinned in the class table test, and
// here the Lord's dive is shown to land through the real round on a foe a
// poisoned status is on, and to strike a clean foe as well.
func TestWyvernLordDivesOnPoisonedAndCleanFoes(t *testing.T) {
	// The Dive's share of a blow on the foe (review: proven by number).
	dive := func(t *testing.T, poisoned bool) int {
		b, stream, foe := eliteRider(t, "wyvern-lord", 30, 10131)
		if poisoned {
			require.NoError(t, foe.Character.AddBuff(13, true))
		}
		seen, restore := hooks.RecordExtraBlowsForTest()
		t.Cleanup(restore)
		b.fight()
		require.Contains(t, halberdTargets(since(*stream, 0), "Tamsin Reed"), foe.InstanceId)
		require.NotEmpty(t, (*seen)[foe.InstanceId])
		return (*seen)[foe.InstanceId][0]
	}
	var clean, poisoned int
	fresh(t, "clean", func(t *testing.T) { clean = dive(t, false) })
	fresh(t, "poisoned", func(t *testing.T) { poisoned = dive(t, true) })
	assert.Equal(t, 25, poisoned-clean, "Rotting venom: +25%% on a poisoned foe")
}

func TestWyvernLordTailLashesAndPoisonsTheFoeBeside(t *testing.T) {
	struck := func(t *testing.T, class string, level int) (ids []int, out string, poisoned int) {
		b, stream, _ := eliteRider(t, class, level, 10131)
		out = b.fight()
		ids = halberdTargets(since(*stream, 0), "Tamsin Reed")
		out += fmt.Sprintf("\n[%d Lashing tail events]", boltCount(since(*stream, 0), "Lashing tail"))
		for _, id := range ids {
			if m := mobs.GetInstance(id); m != nil && m.Character.HasBuff(13) {
				poisoned++
			}
		}
		return
	}
	fresh(t, "lashing tail", func(t *testing.T) {
		seen, restore := hooks.RecordExtraBlowsForTest()
		t.Cleanup(restore)
		ids, out, poisoned := struck(t, "wyvern-lord", 60)
		require.Len(t, ids, 2)
		dive, tail := (*seen)[ids[0]], (*seen)[ids[1]]
		require.NotEmpty(t, dive)
		require.NotEmpty(t, tail)
		assert.Equal(t, dive[0]/2, tail[0], "half the Dive's damage, not half a blow")
		assert.Len(t, ids, 2, "the dive and the tail lash\n%s", out)
		assert.Contains(t, out, "tail lashes")
		assert.Contains(t, out, "[1 Lashing tail events]", "the battle caption names it")
		assert.Equal(t, 2, poisoned)
	})
	fresh(t, "wyvern rider", func(t *testing.T) {
		ids, out, _ := struck(t, "wyvern-rider", 25)
		assert.Len(t, ids, 1)
		assert.NotContains(t, out, "tail lashes")
	})
}

// ----- Alchemist -----

func TestPanaceanElixirKeepsAFallingAllyOnItsFeet(t *testing.T) {
	// One bandit strikes Ysolde, who stands at one health of a hundred; the
	// rest keep to Garrick, who is whole.
	strike := func(t *testing.T, class string, level int) (health int, out string) {
		b := alchemistBrawl(t, class, level) // no spells: no draught gets to Ysolde first
		stream := b.listen()
		defer func() {
			// The battle caption names it by the ally it saved (review: an
			// event with no source showed nothing).
			for _, e := range *stream {
				if e.Kind == combatstream.Ability && e.Status == "Elixir" {
					out += "\n[Elixir event from " + e.Source.Name + "]"
				}
			}
		}()
		forceBlows(t, true)
		b.startWitchFight()
		ysolde, garrick := b.companion(4), b.companion(3)
		ysolde.Character.HealthMax.Value, ysolde.Character.Health = 100, 1
		garrick.Character.HealthMax.Value, garrick.Character.Health = 1000, 1000
		b.aria.Character.Health = b.aria.Character.HealthMax.Value
		for _, m := range b.livingBandits() {
			target := garrick
			if m.InstanceId == b.bandits["bandit bruiser"][0] {
				target = ysolde
			}
			m.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
			m.Character.Aggro.RoundsWaiting = 0
		}
		out = b.fight()
		return ysolde.Character.Health, out
	}
	fresh(t, "panacean", func(t *testing.T) {
		health, out := strike(t, "panacean", 30)
		assert.Equal(t, 25, health, "a quarter of its health\n%s", out)
		assert.Contains(t, out, "(elixir,")
		assert.Contains(t, out, "[Elixir event from Ysolde")
	})
	fresh(t, "fine elixir", func(t *testing.T) {
		health, _ := strike(t, "panacean", 50)
		assert.Equal(t, 40, health, "40% of its health at rank 50")
	})
	fresh(t, "apothecary", func(t *testing.T) {
		health, out := strike(t, "apothecary", 30)
		assert.Less(t, health, 1, "no elixir without the rank\n%s", out)
		assert.NotContains(t, out, "elixir")
	})
}

func TestPanaceanElixirWorksOnceUntilTheSecondElixir(t *testing.T) {
	// Two strikes on one ally in one round: the elixir is spent on the first.
	two := func(t *testing.T, level int) int {
		b := alchemistBrawl(t, "panacean", level)
		forceBlows(t, true)
		b.startWitchFight()
		ysolde := b.companion(4)
		ysolde.Character.HealthMax.Value, ysolde.Character.Health = 20, 1 // a quarter is 5: the next blow is lethal again
		b.aria.Character.Health = b.aria.Character.HealthMax.Value
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, ysolde.InstanceId, characters.DefaultAttack)
			m.Character.Aggro.RoundsWaiting = 0
		}
		b.fight()
		return b.aria.Character.RT.ElixirSpent
	}
	fresh(t, "once", func(t *testing.T) { assert.Equal(t, 1, two(t, 59)) })
	fresh(t, "twice", func(t *testing.T) { assert.Equal(t, 2, two(t, 60)) })
}

func TestGrenadierFireFlaskReachesMostOfAnEnemyCompany(t *testing.T) {
	hit := func(t *testing.T, class string, level int) (n int) {
		b := alchemistBrawl(t, class, level, "draught", "fireflask")
		b.startWitchFight()
		b.aria.Character.FlasksSpent = 0
		b.classRounds(1, nil)
		return scorched(b)
	}
	fresh(t, "bombardier", func(t *testing.T) { assert.Equal(t, 3, hit(t, "bombardier", 29), "three foes") })
	fresh(t, "grenadier", func(t *testing.T) { assert.Equal(t, 4, hit(t, "grenadier", 30), "Grenado: four foes") })
	fresh(t, "barrage", func(t *testing.T) { assert.Equal(t, 5, hit(t, "grenadier", 45), "Barrage: the whole company") })
}

func TestTransmuterMutagenHardensMoreOfThePatientsRow(t *testing.T) {
	armored := func(t *testing.T, class string, level int) (n, rowSize int) {
		b := alchemistBrawl(t, class, level, "draught", "tonic")
		b.placeRows()
		b.startWitchFight()
		rows := map[int]int{} // formation row by member, from placeRows: Aria, 1 and 2 in front; 3 and 4 behind
		all := []*characters.Character{b.aria.Character, &b.companion(1).Character, &b.companion(2).Character, &b.companion(3).Character, &b.companion(4).Character}
		for i := range all {
			rows[i] = 0
			if i >= 3 {
				rows[i] = 1
			}
		}
		patientRow := -1
		for i, c := range all {
			if c.RT != nil && c.RT.Bark > 0 {
				n++
				patientRow = rows[i]
			}
		}
		for _, r := range rows {
			if r == patientRow {
				rowSize++
			}
		}
		return n, rowSize
	}
	fresh(t, "mutagenist", func(t *testing.T) {
		n, _ := armored(t, "mutagenist", 30)
		assert.Equal(t, 1, n)
	})
	fresh(t, "twin mutagen", func(t *testing.T) {
		n, _ := armored(t, "transmuter", 30)
		assert.Equal(t, 2, n, "the patient and one ally in its row")
	})
	fresh(t, "row mutagen", func(t *testing.T) {
		n, size := armored(t, "transmuter", 60)
		assert.Equal(t, size, n, "the whole row")
		assert.GreaterOrEqual(t, n, 2)
	})
}

// ----- Arbalist -----

// frontFoeOf is the foe standing in the front row of the column a deeper foe
// stands in.
func (b *brawl) frontFoeOf(deep *mobs.Mob) *mobs.Mob {
	b.t.Helper()
	party, ok := enemyparty.PartyOf(b.road, deep.InstanceId)
	require.True(b.t, ok)
	_, col, found := party.Formation.Find(mobparty.MemberKeyFor(deep.InstanceId))
	require.True(b.t, found)
	id, ok := mobparty.InstanceIdFromMemberKey(party.Formation.At(0, col))
	require.True(b.t, ok)
	return mobs.GetInstance(id)
}

func TestSiegeMasterBoltPassesThroughToTheFoeBehind(t *testing.T) {
	// The enemy line re-forms between rounds, so only a battle's first bolt
	// has a known foe behind it: the bolts already spent are set beforehand.
	run := func(t *testing.T, class string, level, spent int) (through int, struck []int, deep *mobs.Mob) {
		b, stream := arbalistBrawl(t, level, class)
		b.companion(1).Character.RTState().Through = spent
		bandits := b.livingBandits()
		for _, m := range bandits { // armor sets the line and the arbalist's aim
			m.Character.Equipment = bandits[0].Character.Equipment
			m.Character.Equipment.Body = items.Item{}
			m.Character.Equipment.Head = items.Item{}
			m.Character.Equipment.Legs = items.Item{}
			m.Character.Equipment.Feet = items.Item{}
		}
		b.hardenBandits()
		deep = b.deepFoe()
		front := b.frontFoeOf(deep)
		front.Character.Equipment.Body = items.New(20163)
		seen, restore := hooks.RecordExtraBlowsForTest()
		t.Cleanup(restore)
		b.arbalistRound(stream, front)
		through = boltCount(since(*stream, 0), "Ballista bolt")
		if through > 0 {
			bolt, behind := (*seen)[front.InstanceId], (*seen)[deep.InstanceId]
			require.NotEmpty(t, bolt)
			require.NotEmpty(t, behind)
			assert.Equal(t, bolt[0]*b.companion(1).Character.ClassEffects().Int(classes.BoltThroughPct)/100, behind[0], "a share of the bolt, not of a plain blow")
		}
		return through, halberdTargets(since(*stream, 0), "Tamsin Reed"), deep
	}
	fresh(t, "siege master", func(t *testing.T) {
		through, struck, deep := run(t, "siege-master", 30, 0)
		require.Equal(t, 1, through)
		assert.Len(t, struck, 2, "the bolt and its pass-through")
		assert.Contains(t, struck, deep.InstanceId, "the foe behind takes the bolt on")
	})
	fresh(t, "once a battle", func(t *testing.T) {
		through, _, _ := run(t, "siege-master", 30, 1)
		assert.Zero(t, through, "its one pass-through is spent")
	})
	fresh(t, "crew", func(t *testing.T) {
		through, _, _ := run(t, "siege-master", 45, 1)
		assert.Equal(t, 1, through, "Ballista crew: a second pass-through")
	})
	fresh(t, "crew spent", func(t *testing.T) {
		through, _, _ := run(t, "siege-master", 45, 2)
		assert.Zero(t, through)
	})
	fresh(t, "siegebreaker", func(t *testing.T) {
		through, struck, _ := run(t, "siegebreaker", 30, 0)
		assert.Zero(t, through)
		assert.Len(t, struck, 1)
	})
}

func TestDeadeyeCriticalBoltNeedsNoWinding(t *testing.T) {
	// Four rounds: the first bolt is loaded already (a Sharpshooter's practiced
	// loader), a plain shot cools, the second bolt and then its winding.
	reloads := func(t *testing.T, class string, crit bool) (n int) {
		b, stream := arbalistBrawl(t, 30, class)
		if crit {
			forceCrits(t)
		}
		foe := b.livingBandits()[0]
		for i := 0; i < 4; i++ {
			_, r, _, _ := b.arbalistRound(stream, foe)
			n += r
		}
		return n
	}
	fresh(t, "deadeye with a critical bolt", func(t *testing.T) { assert.Zero(t, reloads(t, "deadeye", true), "no winding after a critical bolt") })
	fresh(t, "sharpshooter with a critical bolt", func(t *testing.T) {
		assert.Equal(t, 1, reloads(t, "sharpshooter", true), "no hair trigger without the rank")
	})
}

func TestDeadeyeBoltThatFellsItsFoeNeedsNoWinding(t *testing.T) {
	// Every round Tamsin's target is a fresh foe on 1 health, so each bolt
	// fells its foe: with the rank no bolt is ever wound, without it each is.
	run := func(t *testing.T, level int) (bolts, reloads int) {
		b, stream := arbalistBrawl(t, level, "deadeye")
		noCrits(t) // a critical bolt needs no winding (Hair trigger), which would hide the winding counted below
		foes := b.livingBandits()
		for i := 0; i < 5 && i < len(foes); i++ {
			b.toughen()
			b.hardenBandits()
			foes[i].Character.Health = 1
			b.companion(1).Character.SetAggro(0, foes[i].InstanceId, characters.DefaultAttack)
			n := len(*stream)
			b.fight()
			bolts += boltCount(since(*stream, n), "Piercing Bolt")
			reloads += boltCount(since(*stream, n), "Winding the crossbow")
		}
		return bolts, reloads
	}
	fresh(t, "ready again", func(t *testing.T) {
		bolts, reloads := run(t, 60)
		require.GreaterOrEqual(t, bolts, 2)
		assert.Zero(t, reloads)
	})
	fresh(t, "without it", func(t *testing.T) {
		bolts, reloads := run(t, 55)
		require.GreaterOrEqual(t, bolts, 2)
		assert.Positive(t, reloads, "a felling bolt is wound at rank 55")
	})
}

func TestBastionAnswersFoesThatStrikeItsColumn(t *testing.T) {
	// Every bandit strikes Tamsin each round. A loaded crossbow answers the
	// first blow of a round and is unloaded by it; her next turn winds it.
	answers := func(t *testing.T, class string, level int) (n int, out string) {
		b, stream := arbalistBrawl(t, level, class, "tamsin abilities off")
		tamsin := b.companion(1)
		for i := 0; i < 5; i++ {
			b.toughen()
			b.hardenBandits()
			for _, m := range b.livingBandits() {
				m.Character.SetAggro(0, tamsin.InstanceId, characters.DefaultAttack)
			}
			out += b.fight()
		}
		return boltCount(since(*stream, 0), "Covering shot"), out
	}
	fresh(t, "bastion", func(t *testing.T) {
		n, out := answers(t, "bastion", 30)
		assert.Equal(t, 2, n, "twice a battle\n%s", out)
		assert.Contains(t, out, "held bolt answers")
		assert.Contains(t, out, "winds the crossbow", "each answer unloads it")
	})
	fresh(t, "third shot", func(t *testing.T) {
		n, _ := answers(t, "bastion", 45)
		assert.Equal(t, 3, n)
	})
	fresh(t, "warden", func(t *testing.T) {
		n, out := answers(t, "warden-of-the-wall", 30)
		assert.Zero(t, n)
		assert.NotContains(t, out, "held bolt")
	})
}

// domainBeastTamer is Tamsin's mob (the Tamer) in a beast brawl.
func domainBeastTamer(b *brawl) (*mobs.Mob, bool) { return b.companion(1), true }

var _ = []any{characters.DefaultAttack, classes.BeastHunt, combatstream.Ability, items.New, hooks.ResetBeastsForTest}
