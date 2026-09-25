package camping

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeaderRest (Phase 26a): a camp, a camp rest, and an inn stay, read
// without side effects.
func TestLeaderRest(t *testing.T) {
	now := baseTime()
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 100)
	_, ok := module.LeaderRest(7)
	assert.False(t, ok, "no camp")

	module.establish(user, eligibleRoom())
	r, ok := module.LeaderRest(7)
	require.True(t, ok)
	assert.False(t, r.Resting)
	assert.False(t, r.Inn)

	module.lightFire(user, eligibleRoom())
	module.startRest(user, eligibleRoom())
	now = now.Add(time.Minute)
	r, ok = module.LeaderRest(7)
	require.True(t, ok)
	assert.True(t, r.Resting)
	assert.Equal(t, camping.RestDuration-time.Minute, r.Remaining)
}

func TestLeaderRestAtInn(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	e.module.innRest(user, innRoom())
	*e.now = baseTime().Add(20 * time.Second)
	r, ok := e.module.LeaderRest(7)
	require.True(t, ok)
	assert.True(t, r.Inn)
	assert.True(t, r.Resting)
	assert.Equal(t, 40*time.Second, r.Remaining)
}

// TestRestTierOf: the best tier a character holds and its real time left.
func TestRestTierOf(t *testing.T) {
	e := newInnEnv(t)
	campUser(t, 7, 2003)
	held := map[int]bool{}
	e.module.hasBuff = func(_ *characters.Character, id int) bool { return held[id] }
	e.module.buffRounds = func(_ *characters.Character, id int) int { return map[int]int{1033: 30, 1030: 90}[id] }
	e.module.roundSeconds = func() int { return 4 }
	_, _, ok := e.module.RestTierOf(7)
	assert.False(t, ok)
	held[1033] = true
	tier, left, ok := e.module.RestTierOf(7)
	require.True(t, ok)
	assert.Equal(t, camping.TierRested, tier)
	assert.Equal(t, 120*time.Second, left)
	held[1030] = true
	tier, left, _ = e.module.RestTierOf(7)
	assert.Equal(t, camping.TierWellRested, tier, "the better tier")
	assert.Equal(t, 360*time.Second, left)
	_, _, ok = e.module.RestTierOf(99)
	assert.False(t, ok, "no such user")
}

func TestRestBuffsRegistered(t *testing.T) {
	e := newInnEnv(t)
	e.module.registerBuffGroupsLocked()
	for _, id := range []int{1030, 1033} {
		g, ok := companyview.GroupOf(id)
		assert.True(t, ok, id)
		assert.Equal(t, companyview.GroupRest, g)
	}
}
