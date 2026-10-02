package gmcp

import (
	"encoding/json"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConditionDurationsAndMechanicalMeaning(t *testing.T) {
	specs := []buffs.BuffSpec{
		{BuffId: 989901, Name: "Timed", Description: "Slows movement", RoundInterval: 5, TriggerCount: 3, StatMods: map[string]int{"speed": -2}},
		{BuffId: 989902, Name: "Stunned", Description: "Loses an action", CombatRounds: true, TriggerCount: 2},
		{BuffId: 989903, Name: "Secret", Description: "private", Secret: true, StatMods: map[string]int{"speed": -9}},
	}
	for i := range specs {
		buffs.SetTestBuffSpec(&specs[i])
		id := specs[i].BuffId
		t.Cleanup(func() { buffs.RemoveTestBuffSpec(id) })
	}
	v := company.MemberConditions{State: "live", Buffs: []buffs.Buff{
		{BuffId: 989901, TriggersLeft: 2, TriggersInitial: 3, RoundCounter: 2},
		{BuffId: 989902, TriggersLeft: 1},
		{BuffId: 989901, TriggersLeft: 1, PermaBuff: true},
		{BuffId: 989903, TriggersLeft: 1},
		{BuffId: 989901, TriggersLeft: 0},
	}, Wounds: []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "ribs", Points: 2, Light: true}}}
	out := conditionsOf(v)
	require.Len(t, out.Effects, 3)
	require.Len(t, out.Bonuses, 1)
	assert.Equal(t, "Persistent", out.Bonuses[0].Duration)
	assert.Equal(t, "Mysterious Affliction", out.Effects[0].Name)
	assert.Empty(t, out.Effects[0].Mods)
	assert.Equal(t, "1 combat rounds remaining", out.Effects[1].Duration)
	assert.Equal(t, fmt.Sprintf("%d seconds remaining", configs.GetTimingConfig().RoundsToSeconds(8)), out.Effects[2].Duration)
	assert.Equal(t, -2, out.Effects[2].Mods["speed"])
	assert.Contains(t, out.Wounds[0].Description, "3 health")
	assert.Equal(t, "Until treated or rested away", out.Wounds[0].Duration)
	assert.Equal(t, "Until fight ends", out.Wounds[1].Duration)
}

// A timed effect's countdown ticks every round, but the client runs it
// itself: the feed resends Company.Conditions only when an effect starts,
// is refreshed or ends, not each round (review of Phase 34d).
func TestConditionsExtraIgnoresTickingCountdown(t *testing.T) {
	spec := buffs.BuffSpec{BuffId: 989904, Name: "Slowed", Description: "Moves slowly.", RoundInterval: 5, TriggerCount: 3, StatMods: map[string]int{"speed": -2}}
	buffs.SetTestBuffSpec(&spec)
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(spec.BuffId) })
	round := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(round) })

	f, out := testFeed()
	f.extras = []companyExtra{conditionsExtra()}
	u := users.NewUserRecord(7, 1)
	slowed := &buffs.Buff{BuffId: spec.BuffId, TriggersLeft: 3, TriggersInitial: 3}
	u.Character.Buffs.List = []*buffs.Buff{slowed}

	f.updateExtras(u)
	require.Len(t, *out, 1)
	var first map[string]memberConditions
	raw, err := json.Marshal((*out)[0].body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &first))
	effect := first["leader"].Effects[0]
	seconds := configs.GetTimingConfig().RoundsToSeconds(15)
	assert.Equal(t, seconds, effect.SecondsLeft)
	assert.Equal(t, seconds, effect.SecondsTotal)
	assert.True(t, effect.Harmful)
	assert.False(t, effect.Helpful)
	assert.Zero(t, effect.ExpiresRound, "the change key's expiry is not sent")

	for i := 0; i < 3; i++ {
		util.SetRoundCount(util.GetRoundCount() + 1)
		u.Character.Buffs.Trigger()
		f.updateExtras(u)
	}
	require.Len(t, *out, 1, "a passing second resends nothing")

	// A command's refresh runs after that round's buffs tick, the round's
	// own refresh before them: the end round they compute differs by one.
	for i := 0; i < 3; i++ {
		util.SetRoundCount(util.GetRoundCount() + 1)
		f.updateExtras(u) // the round refresh, before the tick
		u.Character.Buffs.Trigger()
		f.updateExtras(u) // a command's refresh, after it
	}
	require.Len(t, *out, 1, "refreshes on either side of the tick resend nothing")

	slowed.RoundCounter, slowed.TriggersLeft = 0, 3
	f.updateExtras(u)
	require.Len(t, *out, 2, "a refreshed effect resends its new countdown")

	u.Character.Buffs.List = nil
	f.updateExtras(u)
	require.Len(t, *out, 3, "an ended effect resends")
	raw, err = json.Marshal((*out)[2].body)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "Slowed")
}
