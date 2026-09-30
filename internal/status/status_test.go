package status

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
)

func loadShipped(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	if err := os.Chdir(filepath.Join(filepath.Dir(thisFile), "..", "..")); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	mudlog.SetupLogger(nil, "", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
}

func holder(t *testing.T) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Health, c.HealthMax.Value = 20, 20
	return c
}

func TestShippedStatusBuffsLoadWithKnownFlags(t *testing.T) {
	loadShipped(t)
	for _, id := range Ids() {
		spec := buffs.GetBuffSpec(id)
		if !assert.NotNil(t, spec, "buff %d", id) {
			continue
		}
		assert.Equal(t, Get(id).Word != "", true)
		assert.Contains(t, spec.Flags, FlagCombatStatus, "buff %d", id)
		for _, f := range spec.Flags {
			assert.True(t, buffs.IsValidFlag(f), "buff %d flag %q", id, f)
		}
	}
	assert.Equal(t, 3, buffs.GetBuffSpec(Bleeding).MaxStacks)
	// Their counts are combat rounds: game rounds must never trigger them.
	for _, id := range Ids() {
		assert.GreaterOrEqual(t, buffs.GetBuffSpec(id).RoundInterval, 100000, "buff %d", id)
		assert.True(t, buffs.GetBuffSpec(id).CombatRounds, "buff %d counts combat rounds", id)
	}
}

func TestCritEffectBySubtype(t *testing.T) {
	first := func(int) int { return 0 }
	cases := []struct {
		sub  items.ItemSubType
		want []int
	}{
		{items.Slashing, []int{Bleeding}},
		{items.Claws, []int{Bleeding}},
		{items.Stabbing, []int{Bleeding, Bleeding}},
		{items.Bludgeoning, []int{Staggered}},
		{items.Shooting, []int{Exposed}},
		{items.Whipping, []int{Hobbled}},
		{items.Generic, nil},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, CritEffect(c.sub, nil, first), string(c.sub))
	}
	// A weapon's own crit buffs win.
	assert.Equal(t, []int{Burning}, CritEffect(items.Slashing, []int{Burning}, first))
}

func TestCritEffectCleavingWeights(t *testing.T) {
	want := map[int][]int{0: {KnockedDown}, 3: {KnockedDown}, 4: {ArmorBroken}, 7: {ArmorBroken}, 8: {Stunned}, 9: {Stunned}}
	for roll, w := range want {
		assert.Equal(t, w, CritEffect(items.Cleaving, nil, func(int) int { return roll }), "roll %d", roll)
	}
}

func TestBleedingStacksTicksAndExpires(t *testing.T) {
	loadShipped(t)
	c := holder(t)
	assert.NoError(t, c.AddBuff(Bleeding, false))
	assert.NoError(t, c.AddBuff(Bleeding, false))

	c.Health = 20 // adding a buff revalidates the character
	got := Tick(c)
	assert.Len(t, got, 1)
	assert.Equal(t, 2, got[0].Damage, "two stacks: 2 damage")
	assert.Equal(t, 18, c.Health)
	assert.False(t, got[0].Expired)

	assert.NoError(t, c.AddBuff(Bleeding, false)) // third stack, count refreshed
	assert.NoError(t, c.AddBuff(Bleeding, false)) // capped at 3
	assert.Equal(t, 3, Tick(c)[0].Damage)

	Tick(c)
	last := Tick(c)
	assert.True(t, last[0].Expired, "three ticks after the refresh it ends")
	assert.False(t, Has(c))
}

func TestLostActionRules(t *testing.T) {
	loadShipped(t)

	stag := holder(t)
	stag.AddBuff(Staggered, false)
	_, lost := LostAction(stag)
	assert.False(t, lost, "not before its first tick")
	Tick(stag)
	s, lost := LostAction(stag)
	assert.True(t, lost)
	assert.Equal(t, Staggered, s.Id)
	Tick(stag)
	_, lost = LostAction(stag)
	assert.False(t, lost, "staggered costs one action")

	down := holder(t)
	down.AddBuff(KnockedDown, false)
	Tick(down)
	_, lost = LostAction(down)
	assert.True(t, lost)
	assert.True(t, Grounded(down), "down, round one")
	Tick(down)
	_, lost = LostAction(down)
	assert.False(t, lost, "knocked down loses only its next action, though it stays down")
	assert.True(t, Grounded(down), "down, round two")
	Tick(down)
	assert.False(t, Grounded(down), "knocked down lasts 2 rounds (owner, 2026-09-30)")

	stun := holder(t)
	stun.AddBuff(Stunned, false)
	assert.True(t, stun.HasBuffFlag(FlagNoDodge) && stun.HasBuffFlag(FlagNoBlock), "no dodging or blocking while stunned")
	Tick(stun)
	_, first := LostAction(stun)
	assert.True(t, Grounded(stun))
	Tick(stun)
	_, second := LostAction(stun)
	assert.True(t, Grounded(stun))
	Tick(stun)
	_, third := LostAction(stun)
	assert.True(t, first && second && !third, "stunned costs two actions")
	assert.False(t, Grounded(stun), "stunned lasts 2 rounds (owner, 2026-09-30)")
}

func TestClearEndsEveryStatus(t *testing.T) {
	loadShipped(t)
	c := holder(t)
	c.AddBuff(Bleeding, false)
	c.AddBuff(ArmorBroken, false)
	assert.Equal(t, 2, Clear(c))
	assert.False(t, Has(c))
	assert.False(t, c.HasBuffFlag(FlagArmorBroken))
}

func TestWordsNamesStatusesOnce(t *testing.T) {
	assert.Equal(t, []string{"bleeding"}, Words([]int{Bleeding, Bleeding, 7}))
	assert.Nil(t, Words([]int{7}))
}

// Phase 30a review: a knockdown's speed penalty ends with it, in the round
// it ends, not at the next prune; the fighter is down for two rounds after
// the one it loses.
func TestKnockdownSpeedEndsWithIt(t *testing.T) {
	loadShipped(t)
	c := holder(t)
	c.Stats.Speed.Base = 20
	c.RecalculateStats()
	base := c.Stats.Speed.ValueAdj
	c.AddBuff(KnockedDown, false)
	c.RecalculateStats()
	down := c.Stats.Speed.ValueAdj
	assert.Less(t, down, base)

	Tick(c) // loses this round
	_, lost := LostAction(c)
	assert.True(t, lost)
	Tick(c) // down one more round (2 in all)
	_, lost = LostAction(c)
	assert.False(t, lost)
	assert.Equal(t, down, c.Stats.Speed.ValueAdj, "still down")
	got := Tick(c)
	assert.True(t, got[0].Expired)
	assert.Equal(t, base, c.Stats.Speed.ValueAdj, "back up the round it ends")
}

func TestGrounded(t *testing.T) {
	loadShipped(t)
	for id, want := range map[int]bool{KnockedDown: true, Stunned: true, Staggered: false, Bleeding: false} {
		c := holder(t)
		if Grounded(c) {
			t.Fatal("nothing on")
		}
		if err := c.AddBuff(id, false); err != nil {
			t.Fatal(err)
		}
		if got := Grounded(c); got != want {
			t.Errorf("%s: Grounded = %v", Word(id), got)
		}
		Clear(c)
		if Grounded(c) {
			t.Errorf("%s: cleared", Word(id))
		}
	}
}
