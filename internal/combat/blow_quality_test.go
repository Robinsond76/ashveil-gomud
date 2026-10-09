package combat

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const qualityClubID = 90360

// qualityOn restores the shipped blow-quality odds over a fixture that turned
// them off, with to-hit certain and no crit or active defense.
func qualityOn(t *testing.T) {
	t.Helper()
	edgeSpecs(t)
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.GlanceEven, cfg.Combat.GlanceFull, cfg.Combat.GlanceLeast = 25, 50, 5
	cfg.Combat.TellingEven, cfg.Combat.TellingFull, cfg.Combat.TellingLeast = 20, 50, 5
	cfg.Combat.GlanceFactor, cfg.Combat.TellingFactor = 0.5, 1.4
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceMax = 0, 0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	items.SetTestItemSpec(&items.ItemSpec{ItemId: qualityClubID, Name: "test club", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "1d1+9", Attacks: 1, DiceCount: 1, SideCount: 1, BonusDamage: 9}})
	t.Cleanup(func() { items.RemoveTestItemSpec(qualityClubID) })
}

func TestBlowQualityOdds(t *testing.T) {
	qualityOn(t)
	for _, tc := range []struct {
		edge              float64
		glancing, telling float64
	}{
		{0, 25, 20},
		{1, 5, 50},
		{-1, 50, 5},
		{0.5, 15, 35},
	} {
		g, tl := qualityShares(tc.edge)
		assert.InDelta(t, tc.glancing, g, 1e-9, "glancing at %v", tc.edge)
		assert.InDelta(t, tc.telling, tl, 1e-9, "telling at %v", tc.edge)
	}

	// The roll boundaries at an even edge: 0-24 glancing, 80-99 telling.
	for roll, want := range map[int]string{0: QualityGlancing, 24: QualityGlancing, 25: QualitySolid, 79: QualitySolid, 80: QualityTelling, 99: QualityTelling} {
		factor, word := blowQuality(0, roll)
		assert.Equal(t, want, word, "roll %d", roll)
		assert.Equal(t, map[string]float64{QualityGlancing: 0.5, QualitySolid: 1, QualityTelling: 1.4}[want], factor)
	}
	assert.InDelta(t, 0.955, expectedQualityFactor(0), 1e-9, "about 0.95x at even")
	assert.Greater(t, expectedQualityFactor(1), expectedQualityFactor(0))
	assert.Less(t, expectedQualityFactor(-1), expectedQualityFactor(0))
}

// A real pass: the quality of each landed blow scales its damage and is
// named on the hit line, and the mean matches what expectedDPS assumes.
func TestBlowQualityThroughARealPass(t *testing.T) {
	qualityOn(t)
	counts := map[string]int{}
	total, rounds := 0, 6000
	for i := 0; i < rounds; i++ {
		src, target := edgeFighter(90231), edgeFighter(90231)
		src.Equipment.Weapon = items.New(qualityClubID)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		require.True(t, res.Hit)
		require.Len(t, res.Qualities, 1)
		q := res.Qualities[0]
		counts[q]++
		total += res.DamageToTarget
		want := map[string]int{QualityGlancing: 5, QualitySolid: 10, QualityTelling: 14}[q]
		assert.Equal(t, want, res.DamageToTarget, q)
		line := res.MessagesToSource[0]
		if q == QualitySolid {
			assert.NotContains(t, line, "glancing")
			assert.NotContains(t, line, "telling")
		} else {
			assert.True(t, strings.Contains(line, "("+q+", "), "%q names a %s blow", line, q)
		}
	}
	assert.InDelta(t, 0.25, float64(counts[QualityGlancing])/float64(rounds), 0.03)
	assert.InDelta(t, 0.20, float64(counts[QualityTelling])/float64(rounds), 0.03)
	mean := float64(total) / float64(rounds)
	assert.InDelta(t, 10*expectedQualityFactor(0), mean, 0.25)

	// expectedDPS reads the same factor: against a certain hit it is the
	// mean damage times the weapon's tempo.
	src, target := edgeFighter(90231), edgeFighter(90231)
	src.Equipment.Weapon = items.New(qualityClubID)
	assert.InDelta(t, 10*expectedQualityFactor(0)*Tempo(src), expectedDPS(*src, *target), 0.01)
}

// A full edge for the attacker shifts the odds to telling blows and a full
// edge against it to glancing ones, through a real pass.
func TestBlowQualityFollowsTheEdge(t *testing.T) {
	qualityOn(t)
	share := func(attackRating, evasionRating int) (glancing, telling float64) {
		n := 3000
		var g, tl int
		for i := 0; i < n; i++ {
			src, target := edgeFighter(90231), edgeFighter(90231)
			src.Equipment.Weapon = items.New(qualityClubID)
			src.AttackOffset, target.EvasionOffset = attackRating, evasionRating
			res := calculateCombat(*src, *target, User, Mob, 0, 0)
			switch res.Qualities[0] {
			case QualityGlancing:
				g++
			case QualityTelling:
				tl++
			}
		}
		return float64(g) / float64(n), float64(tl) / float64(n)
	}
	_, tellingFor := share(50, 0)
	glancingAgainst, tellingAgainst := share(0, 50)
	assert.GreaterOrEqual(t, tellingFor, 0.45)
	assert.LessOrEqual(t, tellingAgainst, 0.08)
	assert.InDelta(t, 0.5, glancingAgainst, 0.04)
}

// A blow that armor absorbs whole reports no glancing or telling quality.
func TestAbsorbedBlowReportsNoQuality(t *testing.T) {
	qualityOn(t)
	const plateID = 90361
	items.SetTestItemSpec(&items.ItemSpec{ItemId: plateID, Name: "test plate", Type: items.Body, DamageReduction: 100000})
	t.Cleanup(func() { items.RemoveTestItemSpec(plateID) })
	for i := 0; i < 400; i++ {
		src, target := edgeFighter(90231), edgeFighter(90231)
		src.Equipment.Weapon = items.New(qualityClubID)
		target.Equipment.Body = items.New(plateID)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		if res.DamageToTarget != 0 {
			continue
		}
		require.Len(t, res.Qualities, 1)
		assert.Equal(t, QualitySolid, res.Qualities[0])
		return
	}
	t.Fatal("no fully absorbed blow")
}

// intensityOf names the intensity whose bludgeoning lines a hit line came
// from, by each line's longest stretch of plain words.
func intensityOf(t *testing.T, line string) string {
	t.Helper()
	strip := regexp.MustCompile(`<[^>]*>|\{[^}]*\}`)
	found := ""
	for name, pct := range map[string]int{"weak": 10, "normal": 50, "heavy": 100} {
		for _, tmpl := range items.GetAttackMessage(items.Bludgeoning, pct, false).Together.ToAttacker {
			longest := ""
			for _, piece := range strip.Split(string(tmpl), -1) {
				if len(piece) > len(longest) {
					longest = piece
				}
			}
			if strings.Contains(line, longest) {
				found = name
			}
		}
	}
	return found
}

// 89 review: a glancing blow reads as a weak hit, never one that "punches
// through the guard", and a telling blow never as a nick, through a real
// pass. The test club's blows are 5 (glancing), 10 or 14 of a top of 10,
// so without the rule a glancing blow took the normal lines.
func TestABlowsLinesMatchItsQuality(t *testing.T) {
	qualityOn(t)
	seen := map[string]int{}
	for i := 0; i < 600; i++ {
		src, target := edgeFighter(90231), edgeFighter(90231)
		src.Equipment.Weapon = items.New(qualityClubID)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		require.Len(t, res.Qualities, 1)
		q, line := res.Qualities[0], res.MessagesToSource[0]
		got := intensityOf(t, line)
		require.NotEmpty(t, got, "%q comes from a bludgeoning line", line)
		switch q {
		case QualityGlancing:
			assert.Equal(t, "weak", got, line)
		case QualityTelling, QualitySolid:
			assert.Equal(t, "heavy", got, line)
		}
		seen[q]++
	}
	assert.Positive(t, seen[QualityGlancing])
	assert.Positive(t, seen[QualityTelling])
	assert.Equal(t, 0, blowProsePct(0, QualityTelling, false), "a blow that did nothing still reads as a miss")
	assert.Equal(t, items.NormalAttackPct, blowProsePct(12, QualityTelling, false), "a light telling blow reads as solid")
	assert.Equal(t, 100, blowProsePct(100, QualityGlancing, true), "a crit reads as a crit")
}
