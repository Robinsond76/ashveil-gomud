package hexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReachGrowsWithLevel(t *testing.T) {
	for level, want := range map[int]int{1: 1, 7: 1, 8: 2, 15: 2, 16: 3, 23: 3, 24: 4, 29: 4, 30: ReachGroup, 40: ReachGroup} {
		if got := Reach(level); got != want {
			t.Errorf("Reach(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestLandChanceIsHeldBetweenBounds(t *testing.T) {
	cases := []struct {
		edge float64
		boss bool
		want int
	}{
		{0, false, 65}, {1, false, 90}, {-1, false, 25}, {0.5, false, 85},
		{0, true, 40}, {1, true, 80}, {-1, true, 25},
	}
	for _, c := range cases {
		if got := LandChance(c.edge, c.boss); got != c.want {
			t.Errorf("LandChance(%v, %v) = %d, want %d", c.edge, c.boss, got, c.want)
		}
	}
}

func TestRoundsAtAndBossHalving(t *testing.T) {
	slumber, _ := For("slumber")
	binding, _ := For("binding")
	earthbind, _ := For("earthbind")
	cases := []struct {
		name string
		h    Hex
		lvl  int
		boss bool
		own  int
		want int
	}{
		{"slumber", slumber, 1, false, 0, 2},
		{"slumber boss", slumber, 1, true, 0, 1},
		{"binding 10", binding, 10, false, 0, 1},
		{"binding 20", binding, 20, false, 0, 2},
		{"binding 20 boss", binding, 20, true, 0, 1},
		{"binding 10 boss keeps at least 1", binding, 10, true, 0, 1},
		{"earthbind keeps the buff's length", earthbind, 3, false, 2, 0},
		{"earthbind boss halves the buff's length", earthbind, 3, true, 2, 1},
	}
	for _, c := range cases {
		if got := c.h.RoundsAt(c.lvl, c.boss, c.own); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
	if Triggers(2) != 3 {
		t.Error("a status counts one trigger more than it lasts")
	}
}

func TestEveryHexNamesAStatusOrMorale(t *testing.T) {
	last := 0
	for _, h := range All {
		if h.Level < last {
			t.Errorf("%s: hexes are listed in the order they are learned", h.Spell)
		}
		last = h.Level
		if h.Buff == 0 && !h.Morale {
			t.Errorf("%s does nothing", h.Spell)
		}
	}
	if _, ok := For("nosuch"); ok {
		t.Error("For finds a hex that isn't")
	}
}

func TestImmunityKeepsAHoldBelowHalfOfTheRounds(t *testing.T) {
	l := NewLedger()
	// A 2-round sleep landed at round 0 holds rounds 1-2; it is then
	// immune through round 4. Sleep and paralysis never hold a foe more
	// than half the rounds.
	l.Land("m1", 1109, 2)
	var immune []int
	for round := 0; round < 8; round++ {
		if l.Immune("m1", 1109) {
			immune = append(immune, round)
		}
		l.Tick()
	}
	want := []int{0, 1, 2, 3, 4}
	if len(immune) != len(want) {
		t.Fatalf("immune rounds %v, want %v", immune, want)
	}
	for i := range want {
		if immune[i] != want[i] {
			t.Fatalf("immune rounds %v, want %v", immune, want)
		}
	}
	if l.Immune("m1", 1110) || l.Immune("m2", 1109) {
		t.Error("immunity is per target and per status")
	}
	l.Reset()
	if l.Immune("m1", 1109) {
		t.Error("Reset forgets")
	}
}

// Phase 38c3: a bonus raises a hex's chance but never past the cap; a boss's
// resist can be halved; a long hold is followed by as long an immunity.
func TestLandChanceWithBonusAndResist(t *testing.T) {
	assert.Equal(t, LandChance(0, false), LandChanceWith(0, 0, 0))
	assert.Equal(t, LandChance(0, true), LandChanceWith(0, BossResist, 0))
	assert.Equal(t, LandChanceWith(0, 0, 0)+10, LandChanceWith(0, 0, 10))
	assert.Equal(t, LandMax, LandChanceWith(5, 0, 40), "never above 90")
	assert.Equal(t, LandMin, LandChanceWith(-5, BossResist, 0), "never below 25")
	assert.Greater(t, LandChanceWith(0, BossResist/2, 0), LandChanceWith(0, BossResist, 0), "Breaking the boss")
}

func TestALongHoldIsFollowedByAsLongAnImmunity(t *testing.T) {
	l := NewLedger()
	l.Land("m1", 1109, 5)
	immune := 0
	for round := 0; round < 14; round++ {
		if l.Immune("m1", 1109) {
			immune++
		}
		l.Tick()
	}
	assert.Equal(t, 10, immune, "a five-round hold and five rounds of immunity: never more than half")
	assert.Equal(t, 14, l.Round())
}
