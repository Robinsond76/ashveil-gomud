package usercommands

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 50: help survival explains the battle condition and help cooking the
// meal buffs, with the rules' own numbers, and the pages that touch them
// point at both.
func TestBattleConditionAndMealBuffHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	surv, err := GetHelpContents("survival")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(surv, "")
	for _, phrase := range []string{"In battle", "Going in:", fmt.Sprintf("-%d%% damage dealt", survival.ConditionSmallPct),
		fmt.Sprintf("-%d%% damage dealt", survival.ConditionClearPct), fmt.Sprintf("+%d%% damage taken", survival.ConditionClearPct),
		fmt.Sprintf("-%d hit chance", survival.FatigueHitPenalty(25)), fmt.Sprintf("-%d Collapsed", survival.FatigueHitPenalty(0))} {
		assert.Contains(t, plain, phrase)
	}
	assert.NotContains(t, plain, "only change\nthe label", "the stale line is gone")

	cook, err := GetHelpContents("cooking")
	require.NoError(t, err)
	plain = tagPattern.ReplaceAllString(cook, "")
	assert.Contains(t, plain, "Meal buffs")
	for _, kind := range survival.MealKinds() {
		m, _ := survival.MealFor(kind)
		assert.Contains(t, plain, m.Name, kind)
		assert.Contains(t, plain, m.Effect(), kind)
	}

	for alias, page := range map[string]string{"battle condition": surv, "battle-condition": surv, "meal buff": cook, "meal-buffs": cook} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, alias)
	}
	for _, topic := range []string{"combat", "conditions", "company meal", "eat"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help cooking", "help %s points at help cooking", topic)
	}
}
