package gmcp

import (
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
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
