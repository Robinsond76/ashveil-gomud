package combat

import (
	"fmt"
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlignmentChange(t *testing.T) {
	tests := []struct {
		killerAlignment int8
		killedAlignment int8
		expectedChange  int
	}{

		{0, 0, 0},
		{0, 5, 0},
		{5, 0, 0},

		{0, 15, 0},
		{0, -15, 0},

		{15, -15, 0},
		{-15, 15, 0},

		{15, 25, 0},
		{25, 15, 0},

		{-20, -25, 2},
		{-25, -20, 2},

		{50, -10, -1},
		{-50, 10, -1},

		{50, -50, 2},
		{-50, 50, -2},

		{90, 0, -2},
		{-90, 0, -2},

		{100, 20, -4},
		{-100, -20, 4},

		{90, -90, 4},
		{-90, 90, -4},
	}

	for _, test := range tests {
		desc := fmt.Sprintf(`%s kills %s`, characters.AlignmentToString(test.killerAlignment), characters.AlignmentToString(test.killedAlignment))
		delta := int(math.Abs(math.Max(float64(test.killerAlignment), float64(test.killedAlignment))-math.Min(float64(test.killerAlignment), float64(test.killedAlignment))) * 0.5)
		result := AlignmentChange(test.killerAlignment, test.killedAlignment)
		if result != test.expectedChange {
			t.Errorf("%s [Delta: %d]: AlignmentChange(%d, %d) = %d; want %d",
				desc, delta, test.killerAlignment, test.killedAlignment, result, test.expectedChange)
		}
	}
}

// edgeConfig pins the stat-edge formulas' knobs to their shipped defaults.
func edgeConfig(t *testing.T) {
	t.Helper()
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.StatEdgeSpan = 10
	cfg.Combat.ToHitMin, cfg.Combat.ToHitEven, cfg.Combat.ToHitMax = 25, 50, 100
	cfg.Combat.CritChanceMin, cfg.Combat.CritChanceEven, cfg.Combat.CritChanceMax = 5, 15, 30
	cfg.Combat.CritMultMin, cfg.Combat.CritMultMax = 1.5, 3.0
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceEven, cfg.Combat.DodgeChanceMax = 5, 12, 30
	cfg.Combat.DamageBonusMin, cfg.Combat.DamageBonusMax, cfg.Combat.DamagePerStrength = 0, 10, 0
	cfg.Combat.DamageEdgeMax = 10
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
}

// TestStatEdge: the difference over the span, held to −1..1.
func TestStatEdge(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct {
		atk, def int
		want     float64
	}{{0, 0, 0}, {3, 2, 0.1}, {2, 3, -0.1}, {7, 2, 0.5}, {12, 2, 1}, {40, 0, 1}, {0, 40, -1}, {math.MaxInt, math.MinInt, 1}} {
		assert.InDelta(t, tt.want, StatEdge(tt.atk, tt.def), 1e-9, "%d against %d", tt.atk, tt.def)
	}
	assert.Zero(t, statAdvantage(2, 3), "a deficit is no advantage")
	cfg := configs.GetGamePlayConfig()
	cfg.Combat.StatEdgeSpan = 5
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))
	assert.InDelta(t, 0.2, StatEdge(3, 2), 1e-9, "a smaller span makes each point count more")
}

// TestDamageBonus: with no absolute growth, the Strength advantage alone
// adds up to DamageEdgeMax (here the whole 0–10 range) over one span.
func TestDamageBonus(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct{ atk, def, want int }{
		{0, 0, 0}, {5, 0, 5}, {10, 0, 10}, {100, 0, 10}, {0, 100, 0}, {3, 2, 1},
	} {
		assert.Equal(t, tt.want, damageBonus(tt.atk, tt.def), "%d against %d", tt.atk, tt.def)
	}
}

// TestHitChance: ToHitEven when even, toward 100 when faster and 25
// when slower; one point is a tenth of the way.
func TestHitChance(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct{ atk, def, want int }{
		{0, 0, 50}, {50, 50, 50}, {3, 2, 55}, {2, 3, 47}, {7, 2, 75}, {12, 2, 100}, {2, 12, 25}, {100, 0, 100}, {0, 100, 25},
	} {
		assert.Equal(t, tt.want, hitChanceForEdge(StatEdge(tt.atk, tt.def)), "%d against %d", tt.atk, tt.def)
	}
}

// TestCritChance: CritChanceEven at equal Smarts, then the buff flags.
func TestCritChance(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct {
		atk, def        int
		accuracy, blink bool
		want            int
	}{
		{0, 0, false, false, 15},
		{10, 0, false, false, 30},
		{0, 10, false, false, 5},
		{5, 0, false, false, 22},
		{10, 0, true, false, 60},   // accuracy doubles
		{10, 0, false, true, 15},   // blink halves
		{10, 0, true, true, 30},    // both
		{0, 0, false, true, 7},     // 15/2, above the minimum
		{0, 10, false, true, 5},    // the minimum holds
		{100, 0, false, false, 30}, // capped
	} {
		assert.Equal(t, tt.want, critChance(tt.atk, tt.def, tt.accuracy, tt.blink), "%+v", tt)
	}
}

// TestCritMultiplier: the minimum, grown by the Perception advantage.
func TestCritMultiplier(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct {
		atk, def int
		want     float64
	}{{0, 0, 1.5}, {10, 0, 3.0}, {5, 0, 2.25}, {0, 10, 1.5}, {100, 0, 3.0}} {
		assert.InDelta(t, tt.want, critMultiplier(tt.atk, tt.def), 1e-9, "%d against %d", tt.atk, tt.def)
	}
}

// TestCritDamageBonus verifies the crit bonus scales with the multiplier.
func TestCritDamageBonus(t *testing.T) {
	tests := []struct {
		dCount, dSides, dBonus int
		atkPerc, defPerc       int
		wantMin                int
		wantMax                int
	}{
		// base=12, mult=1.5 -> bonus = floor(12*(1.5-1)) = floor(6) = 6
		{2, 6, 0, 0, 0, 6, 6},
		// base=12, mult=3.0 (a full Perception edge) -> floor(12*(3.0-1)) = 24
		{2, 6, 0, 100, 0, 24, 24},
		// base=0, any -> 0
		{0, 0, 0, 100, 0, 0, 0},
		// negative base clamped to 0
		{1, 4, -10, 0, 0, 0, 0},
	}
	for _, tt := range tests {
		got := critDamageBonus(tt.dCount, tt.dSides, tt.dBonus, tt.atkPerc, tt.defPerc)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("critDamageBonus(%d,%d,%d,%d,%d) = %d; want [%d, %d]",
				tt.dCount, tt.dSides, tt.dBonus, tt.atkPerc, tt.defPerc, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestDodgeChance: two-sided since 35a2: DodgeChanceEven when even, moved
// toward a bound by the defender's edge.
func TestDodgeChance(t *testing.T) {
	edgeConfig(t)
	for _, tt := range []struct{ def, atk, want int }{
		{0, 0, 12}, {10, 0, 30}, {0, 10, 5}, {5, 0, 21}, {3, 2, 13}, {2, 3, 11}, {100, 0, 30},
	} {
		assert.Equal(t, tt.want, dodgeChanceForEdge(StatEdge(tt.def, tt.atk)), "%d against %d", tt.def, tt.atk)
	}
}

func TestDualWieldHitPenalty(t *testing.T) {
	tests := []struct {
		dwLevel int
		want    int
	}{
		{0, -35},
		{1, -35},
		{2, -35},
		{3, -35},
		{4, -25},
		{5, -25},
	}
	for _, tt := range tests {
		got := dualWieldHitPenalty(tt.dwLevel)
		if got != tt.want {
			t.Errorf("dualWieldHitPenalty(%d) = %d; want %d", tt.dwLevel, got, tt.want)
		}
	}
}

func TestDualWieldActiveWeaponCount(t *testing.T) {
	// Deterministic cases
	tests := []struct {
		dwLevel   int
		bothClaws bool
		want      int
	}{
		{0, false, 1},
		{1, false, 1},
		{3, false, 2},
		{4, false, 2},
		{0, true, 2},
		{1, true, 2},
	}
	for _, tt := range tests {
		got := dualWieldActiveWeaponCount(tt.dwLevel, tt.bothClaws)
		if got != tt.want {
			t.Errorf("dualWieldActiveWeaponCount(%d, %v) = %d; want %d", tt.dwLevel, tt.bothClaws, got, tt.want)
		}
	}

	// Probabilistic case: dwLevel == 2, bothClaws == false should return 1 or 2
	saw1, saw2 := false, false
	for i := 0; i < 200; i++ {
		v := dualWieldActiveWeaponCount(2, false)
		if v == 1 {
			saw1 = true
		} else if v == 2 {
			saw2 = true
		} else {
			t.Errorf("dualWieldActiveWeaponCount(2, false) returned unexpected value %d", v)
		}
		if saw1 && saw2 {
			break
		}
	}
	if !saw1 || !saw2 {
		t.Errorf("dualWieldActiveWeaponCount(2, false) did not produce both 1 and 2 over 200 iterations")
	}
}

func TestApplyDefenseReduction(t *testing.T) {
	// Zero defense: no reduction ever
	for i := 0; i < 50; i++ {
		final, red := applyDefenseReduction(100, 0)
		if final != 100 || red != 0 {
			t.Errorf("applyDefenseReduction(100, 0) = (%d, %d); want (100, 0)", final, red)
		}
	}

	// Non-zero defense: final + reduction == original damage
	for i := 0; i < 100; i++ {
		final, red := applyDefenseReduction(100, 50)
		if final+red != 100 {
			t.Errorf("applyDefenseReduction(100, 50): final(%d) + reduction(%d) != 100", final, red)
		}
		if final < 0 || red < 0 {
			t.Errorf("applyDefenseReduction(100, 50): negative value final=%d red=%d", final, red)
		}
	}
}

func TestDamagePercentOfMax(t *testing.T) {
	tests := []struct {
		damage, dCount, dSides, dBonus int
		want                           int
	}{
		{0, 2, 6, 0, 0},    // 0% of max
		{12, 2, 6, 0, 100}, // max damage = 12, 100%
		{6, 2, 6, 0, 50},   // half max
		{1, 0, 0, 0, 100},  // maxDmg clamped to 1, 100%
		{5, 2, 6, 0, 42},   // ceil(5/12*100) = 42
	}
	for _, tt := range tests {
		got := damagePercentOfMax(tt.damage, tt.dCount, tt.dSides, tt.dBonus)
		if got != tt.want {
			t.Errorf("damagePercentOfMax(%d, %d, %d, %d) = %d; want %d",
				tt.damage, tt.dCount, tt.dSides, tt.dBonus, got, tt.want)
		}
	}
}

func TestDarknessPenaltyNilRoom(t *testing.T) {
	called := false
	got := darknessPenalty(nil, &characters.Character{}, func(*rooms.Room) int { called = true; return 0 })
	if got != 0 || called {
		t.Fatalf("an unknown room applies no penalty and computes no visibility, got %d", got)
	}
}

// Phase 38a review: a sleeper is easier to hit, not harder. The penalty is
// subtracted from the hit chance, so the sleeper's bonus lowers it.
func TestDarknessPenaltyFavorsASleepingTarget(t *testing.T) {
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1109, Name: "Asleep", TriggerRate: "100000 rounds", TriggerCount: 3, Flags: []string{status.FlagAsleep}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(1109) })
	awake := characters.New()
	asleep := characters.New()
	if err := asleep.AddBuff(1109, false); err != nil {
		t.Fatal(err)
	}
	require.True(t, asleep.HasBuffFlag(status.FlagAsleep))
	assert.Equal(t, -status.AsleepHitBonus, darknessPenalty(nil, asleep, nil))
	lit := func(*rooms.Room) int { return 2 }
	room := &rooms.Room{}
	assert.Less(t, darknessPenalty(room, asleep, lit), darknessPenalty(room, awake, lit))
}
