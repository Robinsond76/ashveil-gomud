package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCombatHelpTopics: every topic in the shipped help index's combat
// category renders, and the Ashveil combat pages answer to their aliases
// (help reach, help whetstone, ...).
func TestCombatHelpTopics(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" && !topic.AdminOnly {
			combat = append(combat, topic.Command)
		}
	}
	for _, want := range []string{"combat", "formation", "targeting", "strategy", "chemistry", "sharpen", "light", "battle-summary", "resurrect", "narration", "combatpace"} {
		assert.Contains(t, combat, want, "help index lists %s under combat", want)
	}
	for _, topic := range combat {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for", topic)
	}

	aliases := map[string]string{
		"battle": "combat", "fighting": "combat",
		"reach": "formation", "interception": "formation",
		"target": "targeting", "whetstone": "sharpen", "darkness": "light",
		"battlesummary": "battle-summary", "resurrection": "resurrect",
		"critical": "narration", "crit": "narration", "healed": "narration", "chanting": "narration",
		"pace": "combatpace", "pacing": "combatpace", "combat-pace": "combatpace",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	// Phase 29c: the narration page explains the parentheses, and the
	// pages that quoted combat lines quote the new voice.
	text, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, want := range []string{"(5 damage)", "(critical hit, 9 damage)", "blocked", "healed)", "(chanting: "} {
		assert.Contains(t, text, want)
	}
	for _, topic := range []string{"combat", "targeting", "formation", "attack"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.NotContains(t, text, "turn on the", topic)
		assert.NotContains(t, text, "turns on the", topic)
	}
	for _, topic := range []string{"combat", "damage"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help narration", "%s links to help narration", topic)
	}

	// The death page reflects Ashveil's death rules (Phase 25a).
	text, err = GetHelpContents("death")
	require.NoError(t, err)
	assert.Contains(t, text, "one level")
	assert.Contains(t, text, "church")
}

func TestCombatHelpPronounsAndOrdinals(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, alias := range []string{"pronouns", "ordinals"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, narration, got)
	}
	for _, want := range []string{"he", "she", "they", "it", "second cutthroat", "third cutthroat", "restart", "you", "your"} {
		assert.Contains(t, narration, want)
	}
	for _, topic := range []string{"narration", "combat", "targeting", "battle-summary"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, text, "second", topic)
		assert.NotContains(t, text, "set pronouns", topic)
	}
	targeting, err := GetHelpContents("targeting")
	require.NoError(t, err)
	assert.Contains(t, targeting, "attack [group]")
}

func TestPainReactionHelpExplainsCriticalAndLethalCases(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	for _, want := range []string{"pain reaction", "critical hit", "stays standing", "death line"} {
		assert.Contains(t, narration, want)
	}
	combat, err := GetHelpContents("combat")
	require.NoError(t, err)
	assert.Contains(t, combat, "pain reaction")
	assert.Contains(t, combat, "help narration")
	critical, err := GetHelpContents("critical")
	require.NoError(t, err)
	assert.Equal(t, narration, critical)
	pain, err := GetHelpContents("pain")
	require.NoError(t, err)
	assert.Equal(t, narration, pain)
}

// TestCombatPaceHelp (Phase 29f): the pacing page gives the four paces and
// their timings, the screen-reader default, the catch-up and flush rules,
// and what is never held back; combat, set, and narration point to it.
func TestCombatPaceHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	page, err := GetHelpContents("combatpace")
	require.NoError(t, err)
	page = tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{
		"set combatpace fast", "set combatpace normal", "set combatpace slow", "set combatpace off",
		"0.4 seconds", "done in 6 seconds", "about 7 seconds",
		"screen reader", "the default is off",
		"every 8 seconds", "before the next begins",
		"pain reaction or a death line",
		"Your prompt", "battle view",
		"Nothing is lost",
		"never held back", "tells",
	} {
		assert.Contains(t, page, want)
	}

	combat, err := GetHelpContents("combat")
	require.NoError(t, err)
	combat = tagPattern.ReplaceAllString(combat, "")
	for _, want := range []string{"every 8 seconds", "up to 8 seconds", "help combatpace", "set combatpace"} {
		assert.Contains(t, combat, want)
	}
	assert.NotContains(t, combat, "a round is 4 seconds", "the old round length is gone")

	set, err := GetHelpContents("set")
	require.NoError(t, err)
	assert.Contains(t, set, "combatpace")

	narration, err := GetHelpContents("narration")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(narration, ""), "A short pause comes before a pain reaction")
}
