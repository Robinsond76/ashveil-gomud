package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func conditionsText(t *testing.T, user *users.UserRecord) string {
	t.Helper()
	messages := captureLookMessages(t)
	_, err := Conditions(``, user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return tagPattern.ReplaceAllString(strings.Join(*messages, "\n"), "")
}

// TestConditionsGrouped (Phase 26a): rest, weapon edges, survival (need
// warnings and exposure buffs), company chemistry (lasting, never shown as
// expiring), and other effects, each in order with what is left.
func TestConditionsGrouped(t *testing.T) {
	sharpenedSpecs(t)
	for _, spec := range []buffs.BuffSpec{
		{BuffId: 9801, Name: "Rested", Description: "You slept well.", RoundInterval: 1, TriggerCount: 30},
		{BuffId: 9802, Name: "Chilled", Description: "You are cold.", RoundInterval: 1, TriggerCount: 10},
		{BuffId: 9803, Name: "Blessed", Description: "The gods smile.", RoundInterval: 1, TriggerCount: 15},
	} {
		spec := spec
		buffs.SetTestBuffSpec(&spec)
		t.Cleanup(func() { buffs.RemoveTestBuffSpec(spec.BuffId) })
	}
	companyview.RegisterBuffGroup(companyview.GroupRest, 9801)
	companyview.RegisterBuffGroup(companyview.GroupSurvival, 9802)
	company.SetFormationProvider(fakeChemistryStanding{ok: true, view: company.ChemistryStandingView{Together: 3, Tier: company.TierTrusted, Bonus: 4}})
	t.Cleanup(func() { company.SetFormationProvider(nil) })
	useSummary(t, sampleSummary())

	user := users.NewUserRecord(7, 1)
	for _, id := range []int{9803, 9802, 9801} {
		require.True(t, user.Character.Buffs.AddBuff(id, false))
	}
	user.Character.Equipment.Weapon = items.New(sharpTestSword)
	user.Character.Equipment.Weapon.Sharpen(1, 9)

	text := conditionsText(t, user)
	order := []string{"Rest", "Rested", "Weapon edges", "9 strikes left", "Survival", "Hungry (40)", "Tired (45)", "Chilled",
		"Company", "Trusted band, 3 together: +4% to hit", "Other effects", "Blessed"}
	last := -1
	for _, want := range order {
		at := strings.Index(text, want)
		require.GreaterOrEqual(t, at, 0, want)
		assert.Greater(t, at, last, "%s in order", want)
		last = at
	}
	assert.Contains(t, text, "left)", "rest and effects show time left")
	chem := text[strings.Index(text, "Trusted band"):]
	assert.NotContains(t, strings.SplitN(chem, "\n", 2)[0], "left", "chemistry never expires")
}

// TestConditionsNoneWhenNothing: nothing to report is still "None".
func TestConditionsNoneWhenNothing(t *testing.T) {
	company.SetFormationProvider(nil)
	useSummary(t, companyview.Summary{Alive: 1})
	assert.Contains(t, conditionsText(t, users.NewUserRecord(7, 1)), "None")
}
