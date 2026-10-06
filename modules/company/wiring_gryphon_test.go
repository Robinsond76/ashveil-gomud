package company

import (
	"fmt"
	"slices"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39f wiring: the Gryphon Rider's Dive through the real strategy pass,
// ability pass and combat round, in the brawl world. Tamsin is the rider: a
// short spear in hand, every other member's abilities off, every blow landing.

// gryphonBrawl is a brawl whose first companion is a level-level Gryphon
// Rider with the shipped short spear; the fight has begun on the captain.
func gryphonBrawl(t *testing.T, level int, before ...string) (*brawl, *[]combatstream.Event) {
	t.Helper()
	b, stream := abilityBrawl(t, map[int]string{1: "gryphon-rider", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.HPArchetype = "gryphon-rider"
	tamsin.Character.Equipment.Weapon = items.New(10131) // an iron short spear
	tamsin.Character.Equipment.Offhand = items.Item{}
	for _, c := range before {
		b.cmd("strategy", c)
	}
	// The rider's own rule decides its aim, not the company's saved focus.
	b.cmd("company", "tactics focus none")
	b.start()
	return b, stream
}

// place puts the company in a formation, Tamsin (the rider) in the front row
// at col, so the dive's lateral range is real.
func (b *brawl) place(col int) {
	b.t.Helper()
	rec, _ := module.registry.Get(7)
	f := domain.Formation{}
	require.NoError(b.t, f.Place(domain.CompanionMemberKey(1), 0, col))
	rest := []domain.MemberKey{domain.LeaderMemberKey, domain.CompanionMemberKey(2), domain.CompanionMemberKey(3), domain.CompanionMemberKey(4)}
	slot := 0
	for r := 1; r < domain.FormationRows && slot < len(rest); r++ {
		for c := 0; c < domain.FormationCols && slot < len(rest); c++ {
			require.NoError(b.t, f.Place(rest[slot], r, c))
			slot++
		}
	}
	rec.Formation = f
	module.registry.Put(rec)
}

// deepFoe is a foe standing behind a standing front-row foe of its column.
func (b *brawl) deepFoe(cols ...int) *mobs.Mob {
	b.t.Helper()
	var any *mobs.Mob
	for _, m := range b.livingBandits() {
		any = m
		break
	}
	party, ok := enemyparty.PartyOf(b.road, any.InstanceId)
	require.True(b.t, ok)
	for row := 1; row < domain.FormationRows; row++ {
		for col := 0; col < domain.FormationCols; col++ {
			if party.Formation.At(0, col) == "" || len(cols) > 0 && !slices.Contains(cols, col) {
				continue
			}
			if id, ok := mobparty.InstanceIdFromMemberKey(party.Formation.At(row, col)); ok {
				if m := mobs.GetInstance(id); m != nil && m.Character.Health > 0 {
					return m
				}
			}
		}
	}
	b.t.Fatal("no foe stands behind the front row")
	return nil
}

func (b *brawl) aimTamsinAt(foe *mobs.Mob) {
	b.companion(1).Character.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
}

func diveCount(stream []combatstream.Event, source string) int {
	n := 0
	for _, e := range abilityEvents(stream, source) {
		if e.Status == "Dive" {
			n++
		}
	}
	return n
}

func TestAGryphonRiderDivesPastTheFrontRow(t *testing.T) {
	b, stream := gryphonBrawl(t, 1)
	foe := b.deepFoe()
	b.aimTamsinAt(foe)
	b.hardenBandits()
	out := b.fight()
	round := since(*stream, 0)
	assert.Equal(t, 1, diveCount(round, "Tamsin Reed"))
	assert.Regexp(t, `Tamsin Reed stoops from the sky on .*\(dive\)`, out)
	hit := halberdTargets(round, "Tamsin Reed")
	require.Len(t, hit, 1, "the dive is her whole turn: one blow")
	assert.Equal(t, foe.InstanceId, hit[0], "the blow lands on the foe behind the front row")
	// Level 1: no talons yet.
	assert.False(t, status.Live(&foe.Character, status.Bleeding))
	// The rider paid 10 Evasion for the rest of the round.
	assert.Equal(t, -10, b.companion(1).Character.Aura.Evasion)
}

func TestADiveRestsThreeRoundsAndTheCostEnds(t *testing.T) {
	b, stream := gryphonBrawl(t, 1)
	foe := b.deepFoe()
	b.aimTamsinAt(foe)
	b.hardenBandits()
	b.fight()
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hardenBandits()
		n := len(*stream)
		b.fight()
		assert.Zero(t, diveCount(since(*stream, n), "Tamsin Reed"), "cooldown, round %d", i+2)
		assert.Zero(t, b.companion(1).Character.Aura.Evasion, "the cost ends with the next round's auras")
	}
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	b.fight()
	assert.Equal(t, 1, diveCount(since(*stream, n), "Tamsin Reed"), "ready again on the fourth round")
}

func TestTalonsBleedFromLevelThree(t *testing.T) {
	for _, tc := range []struct {
		level int
		bleed bool
	}{{2, false}, {3, true}} {
		t.Run(fmt.Sprintf("level %d", tc.level), func(t *testing.T) {
			b, stream := gryphonBrawl(t, tc.level)
			foe := b.deepFoe()
			b.aimTamsinAt(foe)
			b.hardenBandits()
			out := b.fight()
			require.Equal(t, 1, diveCount(since(*stream, 0), "Tamsin Reed"))
			assert.Equal(t, tc.bleed, status.Live(&foe.Character, status.Bleeding))
			if tc.bleed {
				assert.Contains(t, out, "talons rake")
			}
		})
	}
}

func TestADiveCannotFlyIndoorsOrOnNarrowGround(t *testing.T) {
	for _, tag := range []string{"narrow", "indoor"} {
		t.Run(tag, func(t *testing.T) {
			b, stream := gryphonBrawl(t, 1)
			was := append([]string(nil), b.road.Tags...)
			t.Cleanup(func() { b.road.Tags = was })
			foe := b.deepFoe()
			b.road.Tags = append(b.road.Tags, tag)
			b.aimTamsinAt(foe)
			b.hardenBandits()
			b.fight()
			assert.Zero(t, diveCount(since(*stream, 0), "Tamsin Reed"))
		})
	}
}

func TestADiveStrikesTheGuardianThatStepsIn(t *testing.T) {
	b, stream := gryphonBrawl(t, 1)
	foe := b.deepFoe()
	party, ok := enemyparty.PartyOf(b.road, foe.InstanceId)
	require.True(t, ok)
	_, col, _ := party.Formation.Find(mobparty.MemberKeyFor(foe.InstanceId))
	guardId, ok := mobparty.InstanceIdFromMemberKey(party.Formation.At(0, col))
	require.True(t, ok)
	guard := mobs.GetInstance(guardId)
	guard.Role = "guardian"
	guard.Coordination = 3 // a drilled group: its guardians may step in
	b.aimTamsinAt(foe)
	b.hardenBandits()
	foe.Character.Health = 100 // the most hurt of its group: the one a guardian guards
	b.fight()
	round := since(*stream, 0)
	require.Equal(t, 1, diveCount(round, "Tamsin Reed"))
	hit := halberdTargets(round, "Tamsin Reed")
	require.Len(t, hit, 1)
	assert.Equal(t, guard.InstanceId, hit[0], "a guardian can still step in for the foe the dive was aimed at")
}

func TestARiderWithAbilitiesOffDoesNotDive(t *testing.T) {
	b, stream := gryphonBrawl(t, 1, "tamsin abilities off")
	foe := b.deepFoe()
	b.aimTamsinAt(foe)
	b.hardenBandits()
	b.fight()
	assert.Zero(t, diveCount(since(*stream, 0), "Tamsin Reed"))
}

func TestAnUnarmedRiderCannotDive(t *testing.T) {
	b, stream := gryphonBrawl(t, 1)
	b.companion(1).Character.Equipment.Weapon = items.Item{}
	foe := b.deepFoe()
	b.aimTamsinAt(foe)
	b.hardenBandits()
	b.fight()
	assert.Zero(t, diveCount(since(*stream, 0), "Tamsin Reed"))
}

func TestAPlayerGryphonRiderDives(t *testing.T) {
	b, stream := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	b.withArchetypesFor("gryphon-rider", nil)
	for _, who := range []string{"tamsin", "garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	aria := b.aria.Character
	aria.SetSkill("skirmish", 1)
	aria.Equipment.Weapon = items.New(10131)
	aria.Equipment.Offhand = items.Item{}
	b.start()
	foe := b.deepFoe()
	aria.SetAggro(0, foe.InstanceId, characters.DefaultAttack)
	b.hardenBandits()
	out := b.fight()
	assert.Regexp(t, `You fold your wings and stoop on .*\(dive\)`, out)
	require.Equal(t, 1, diveCount(since(*stream, 0), "Aria"))
	hit := halberdTargets(since(*stream, 0), "Aria")
	require.Len(t, hit, 1)
	assert.Equal(t, foe.InstanceId, hit[0])
}

func TestAGryphonRidersDefaultAimIsTheHealersThenCasters(t *testing.T) {
	b, _ := gryphonBrawl(t, 1)
	list := b.cmd("strategy", "")
	assert.Regexp(t, `Tamsin Reed\s+gryphon-rider\s+fighter\s+healers`, list)
}

func TestADiveReachesOnlyAColumnOfItsOwnOrTheNext(t *testing.T) {
	// Tamsin stands in column 2: the dive reaches columns 1 and 2, not 0.
	for _, tc := range []struct {
		name   string
		col    int
		struck bool
	}{{"column 1", 1, true}, {"column 0", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			b, stream := gryphonBrawl(t, 1)
			b.place(2)
			foe := b.deepFoe(tc.col)
			b.aimTamsinAt(foe)
			b.hardenBandits()
			b.fight()
			hit := halberdTargets(since(*stream, 0), "Tamsin Reed")
			if tc.struck {
				assert.Contains(t, hit, foe.InstanceId)
			} else {
				assert.NotContains(t, hit, foe.InstanceId, "out of the dive's range, and behind the front")
			}
		})
	}
}

// A placed rider's healers rule aims past the front row at a healer in
// lateral range, and not at one out of it (the aim, before any swing).
func TestAPlacedRidersAimReachesTheHealerBehindTheFront(t *testing.T) {
	for _, tc := range []struct {
		name string
		col  int
		aims bool
	}{{"in range", 1, true}, {"out of range", 0, false}} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := gryphonBrawl(t, 1)
			b.place(2)
			healer := b.deepFoe(tc.col)
			healer.Role = "healer"
			b.hardenBandits()
			g, ok := enemyparty.GroupOf(b.road, healer.InstanceId)
			require.True(t, ok)
			id, ok := enemyparty.Aim(g, enemyparty.CompanionAttacker(7, domain.CompanionMemberKey(1), b.companion(1), 0))
			require.True(t, ok)
			if tc.aims {
				assert.Equal(t, healer.InstanceId, id)
			} else {
				assert.NotEqual(t, healer.InstanceId, id, "a healer outside its columns is behind the front")
			}
		})
	}
}

// detectChance is the chance in 100 that the company spots an ambush now,
// found by spawning ambushes against rising rolls: a roll below the chance
// spots it.
func detectChance(t *testing.T, b *brawl) int {
	t.Helper()
	spotted := func(roll int) bool {
		restore := enemyparty.UseAmbushRollForTest(func(int) int { return roll })
		defer restore()
		first, err := enemyparty.SpawnAmbush(b.road.RoomId, 9106, 7)
		require.NoError(t, err)
		p, ok := enemyparty.PartyOf(b.road, first)
		require.True(t, ok)
		spot := mobs.GetInstance(p.Members[0]).AmbushAdvantage >= 0
		for _, id := range p.Members { // clear the spawn away
			m := mobs.GetInstance(id)
			m.Character.Health = 0
			m.AmbushOwner = 0
			b.road.RemoveMob(id)
		}
		return spot
	}
	lo, hi := 0, 100 // spotted at lo, not spotted at hi
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if spotted(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi
}

// A Skyscout's Eagle eye adds Perception to the company's look for an
// ambush in the open, and nothing indoors.
func TestASkyscoutsEagleEyeRaisesAmbushDetectionOutdoorsOnly(t *testing.T) {
	b, _ := gryphonBrawl(t, 10)
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Stats.Perception.ValueAdj = 0
	}
	b.aria.Character.Stats.Perception.ValueAdj = 0
	tamsin := b.companion(1)
	base := detectChance(t, b)
	require.Greater(t, base, 5)
	require.Less(t, base, 85, "the roll can move")

	tamsin.Character.HPClass = "skyscout"
	assert.Equal(t, base+6, detectChance(t, b), "Eagle eye: +6 in the open")

	tamsin.Character.Level = 25
	assert.Equal(t, base+12, detectChance(t, b), "Far sight: +12")

	was := append([]string(nil), b.road.Tags...)
	t.Cleanup(func() { b.road.Tags = was })
	b.road.Tags = append(b.road.Tags, "indoor")
	withEye := detectChance(t, b)
	tamsin.Character.HPClass = ""
	assert.Equal(t, detectChance(t, b), withEye, "no sky indoors: the Skyscout sees no better than anyone")
}

// The routes shape the dive: a Gryphon Knight's lance charge knocks the foe
// down (with a two-handed reach weapon only), a Skyscout's Hawk's mark leaves
// it exposed, a Wyvern Rider's venom poisons it, and a Steady wings dive
// costs no Evasion.
func TestRoutesShapeTheDive(t *testing.T) {
	for _, tc := range []struct {
		name   string
		class  string
		level  int
		weapon int
		check  func(t *testing.T, foe *mobs.Mob, rider *characters.Character)
	}{
		{"knight with a war spear", "gryphon-knight", 10, 10141, func(t *testing.T, foe *mobs.Mob, _ *characters.Character) {
			assert.True(t, status.Live(&foe.Character, status.KnockedDown))
		}},
		{"knight with a short spear", "gryphon-knight", 10, 10131, func(t *testing.T, foe *mobs.Mob, _ *characters.Character) {
			assert.False(t, status.Live(&foe.Character, status.KnockedDown), "no reach weapon, no lance charge")
		}},
		{"skyscout before Hawk's mark", "skyscout", 19, 10131, func(t *testing.T, foe *mobs.Mob, _ *characters.Character) {
			assert.False(t, status.Live(&foe.Character, status.Exposed))
		}},
		{"skyscout at level 20", "skyscout", 20, 10131, func(t *testing.T, foe *mobs.Mob, _ *characters.Character) {
			assert.True(t, status.Live(&foe.Character, status.Exposed))
		}},
		{"wyvern rider", "wyvern-rider", 10, 10131, func(t *testing.T, foe *mobs.Mob, _ *characters.Character) {
			assert.True(t, foe.Character.HasBuff(13), "poisoned")
		}},
		{"knight at 25", "gryphon-knight", 25, 10131, func(t *testing.T, _ *mobs.Mob, rider *characters.Character) {
			assert.Zero(t, rider.Aura.Evasion, "Steady wings: no Evasion spent")
		}},
		{"knight at 24", "gryphon-knight", 24, 10131, func(t *testing.T, _ *mobs.Mob, rider *characters.Character) {
			assert.Equal(t, -10, rider.Aura.Evasion)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, stream := gryphonBrawl(t, tc.level)
			loadShippedBuff(t, "13-poisoned.yaml")
			rider := b.companion(1)
			rider.Character.HPClass = tc.class
			rider.Character.Equipment.Weapon = items.New(tc.weapon)
			foe := b.deepFoe()
			b.aimTamsinAt(foe)
			b.hardenBandits()
			b.fight()
			require.Equal(t, 1, diveCount(since(*stream, 0), "Tamsin Reed"))
			tc.check(t, foe, &rider.Character)
		})
	}
}
