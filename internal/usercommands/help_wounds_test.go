package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWoundsHelp (Phase 30b): wounds and heal render through help, are
// listed under combat, the aliases reach wounds and take no other topic's
// word, and the hub and the changed pages point to wounds.
func TestWoundsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" {
			listed[topic.Command] = true
		}
	}
	assert.True(t, listed["wounds"], "help index lists wounds under combat")
	assert.True(t, listed["heal"], "help index lists heal under combat")

	page, err := GetHelpContents("wounds")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(page, "")
	for _, want := range []string{"Help for wounds", "wound limit", "Lasting wounds", "Light wounds", "linen bandage", "birch splint", "physician", "camp rest", "inn stay", "(critical hit, 6 damage, bleeding, wounded)"} {
		assert.Contains(t, plain, want)
	}
	for _, alias := range []string{"wound", "wound-limit", "heal-wounds", "bandage", "splint", "physician", "tend", "injuries"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help wounds", alias)
	}

	heal, err := GetHelpContents("heal")
	require.NoError(t, err)
	healPlain := tagPattern.ReplaceAllString(heal, "")
	assert.Contains(t, healPlain, "heal wounds")
	assert.Contains(t, healPlain, "Pay 30 gold? [yes/no]")
	assert.NotContains(t, healPlain, "(skill)", "the stale GoMud skill page is gone")
	assert.NotEqual(t, page, heal)
	health, err := GetHelpContents("health")
	require.NoError(t, err)
	assert.NotEqual(t, heal, health, "help health is still health")

	for _, topic := range []string{"combat", "statuses", "camp", "inn", "death", "health", "heal"} {
		got, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(got, ""), "help wounds", "help %s points to help wounds", topic)
	}
}
