package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30d1 (and 30d1b's chance): help interrupts renders line for line, answers to its
// aliases, sits under combat, and the pages it changed point to it.
func TestInterruptsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Command == "interrupts" {
			listed = topic.Category == "combat"
		}
	}
	assert.True(t, listed, "help index lists interrupts under combat")

	text, err := GetHelpContents("interrupts")
	require.NoError(t, err)
	plainText := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Help for interrupts",
		"A weapon blow that draws blood\nmay break it",
		"40%", "90%", "a quarter of their health",
		"critical hit", "staggers", "knocks down", "stuns", "maximum health",
		"always breaks",
		"Brother Oswin flinches, but the chant holds. (Minor Heal, chant held)",
		"You flinch, but your chant holds. (Minor Heal, chant held)",
		"The goblin hexer's chant breaks off under the blow. (Withering Hex interrupted)",
		"(Minor Heal interrupted, 1 mana back)",
		"half its mana comes back",
		"from the first word",
		"back row", "casters",
		"(shield bash, 3 damage, stunned)",
		"50%", "1 to 4 damage", "one counter a round",
	} {
		assert.Contains(t, plainText, want)
	}
	for _, alias := range []string{"interrupt", "interrupted", "chant", "chants", "concentration", "shield-bash", "shieldbash", "counter", "counters", "hexer"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help interrupts", alias)
	}

	for _, topic := range []string{"combat", "cast", "strategy", "tactics", "statuses", "guardian", "battle-summary", "formation"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "help interrupts", topic)
		// 30d1b: a blow breaks a chant only by chance now.
		for _, stale := range []string{"draws blood breaks", "draws blood on the caster\nbreaks", "draws blood on the caster breaks", "draws blood\nbreaks"} {
			assert.NotContains(t, plain, stale, topic)
		}
	}
}
