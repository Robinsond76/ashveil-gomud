package formationcombat

import (
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDetectionBoundariesAndWatch(t *testing.T) {
	for _, tc := range []struct {
		perception, stealth, light, cover, roll int
		watch                                   bool
		chance, advantage                       int
	}{
		{50, 50, 2, 0, 50, false, 50, -1}, {50, 50, 2, 0, 49, false, 50, 0},
		{50, 50, 2, 0, 20, false, 50, 1}, {50, 50, 2, 0, 21, false, 50, 0},
		{50, 50, 0, 10, 99, true, 15, 0}, {500, 0, 2, 0, 94, false, 95, 0},
		{0, 500, 2, 0, 5, false, 5, -1}, {50, 50, 1, 0, 40, false, 40, -1},
	} {
		chance, advantage := Detection(tc.perception, tc.stealth, tc.light, tc.cover, tc.roll, tc.watch)
		assert.Equal(t, tc.chance, chance)
		assert.Equal(t, tc.advantage, advantage)
	}
}

func TestFoldPreservesMembersReservesAndRecovery(t *testing.T) {
	var f company.Formation
	alive := map[company.MemberKey]bool{}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			key := company.CompanionMemberKey(r*3 + c + 1)
			f[r][c] = key
			alive[key] = true
		}
	}
	saved := f
	folded := Fold(f, alive)
	seen := map[company.MemberKey]bool{}
	for _, row := range folded {
		for _, key := range row {
			require.NotEmpty(t, key)
			require.False(t, seen[key])
			seen[key] = true
		}
	}
	assert.Len(t, seen, 9)
	assert.Equal(t, saved, f)
	reserve := folded[0][2]
	assert.False(t, LegalGround(0, folded, reserve, alive, alive, ReachNone, true))
	assert.True(t, LegalGround(0, folded, reserve, alive, alive, ReachAny, true))
	assert.False(t, LegalGround(2, folded, folded[0][0], alive, alive, ReachNone, true))
	delete(alive, folded[0][0])
	delete(alive, folded[0][1])
	delete(alive, folded[1][0])
	recovered := Fold(f, alive)
	assert.Empty(t, recovered[0][2])
	assert.Empty(t, recovered[1][2])
	assert.Empty(t, recovered[2][2])
}

func TestClusterIsOrthogonalAndNeverChains(t *testing.T) {
	var f company.Formation
	live := map[company.MemberKey]bool{}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			key := company.CompanionMemberKey(r*3 + c + 1)
			f[r][c] = key
			live[key] = true
		}
	}
	assert.ElementsMatch(t, []company.MemberKey{f[1][1], f[0][1], f[2][1], f[1][0], f[1][2]}, Cluster(f, f[1][1], live, false))
	assert.Len(t, Cluster(f, f[0][0], live, false), 3)
	assert.NotContains(t, Cluster(f, f[1][1], live, true), f[1][2])
	delete(live, f[0][1])
	assert.Len(t, Cluster(f, f[1][1], live, false), 4)
}

func TestBrokenAdjacentColumnsExposeRear(t *testing.T) {
	f := company.Formation{{"front-a", "front-b", ""}, {"", "mid", ""}, {"", "rear", ""}}
	live := map[company.MemberKey]bool{"front-a": true, "front-b": true, "mid": true, "rear": true}
	standing := map[company.MemberKey]bool{"front-a": true, "front-b": true, "mid": true, "rear": true}
	assert.False(t, LegalGround(1, f, "rear", live, standing, ReachNone, false))
	standing["front-b"] = false
	assert.True(t, Flanked(f, "rear", standing, false))
	assert.True(t, LegalGround(1, f, "rear", live, standing, ReachNone, false))
	assert.False(t, Flanked(f, "rear", standing, true), "in two columns the other front still stands")
}

func TestFatigueBoundaries(t *testing.T) {
	for _, tc := range []struct{ rest, want int }{{100, 0}, {51, 0}, {50, 5}, {26, 5}, {25, 10}, {1, 10}, {0, 20}} {
		assert.Equal(t, tc.want, FatiguePenalty(tc.rest))
	}
}
