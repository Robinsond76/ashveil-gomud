package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCombatFeelHelp (Phase 35d): the pages the phase changed render with
// their config values filled in, and the new aliases reach them.
func TestCombatFeelHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	pages := map[string][]string{
		"attack":     {"glancing", "telling", "(glancing, "},
		"interrupts": {"one-round", "critical"},
		"tactics":    {"patch", "guard", "weakest"},
		"patch":      {"80", "company tactics patch"},
	}
	for topic, wants := range pages {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		text = tagPattern.ReplaceAllString(text, "")
		assert.NotContains(t, text, "<no value>", topic)
		assert.NotContains(t, text, "{{", topic)
		for _, want := range wants {
			assert.Contains(t, text, want, "help %s mentions %q", topic, want)
		}
	}

	for alias, topic := range map[string]string{
		"patch threshold": "patch", "default focus": "tactics", "level defaults": "tactics",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	combat, err := GetHelpContents("combat")
	require.NoError(t, err)
	assert.Contains(t, tagPattern.ReplaceAllString(combat, ""), "help patch")
}
