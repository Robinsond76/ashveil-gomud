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
		"5% to 20%", "blocks a melee blow", "1 to 4 damage", "one counter a round",
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

// Phase 30d2: help interrupts tells of wind-ups (the ogre's Crushing
// Blow): the telegraph, the numbers, what breaks one and what doesn't
// (never a shield bash), the cooldown, and the tactics; its aliases lead
// there; the pages that touch it say so.
func TestInterruptsHelpWindUps(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("interrupts")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{
		"Wind-ups",
		"forest ogre",
		"(winding up: Crushing Blow, 1 round)",
		"The forest ogre brings his great club down with all his weight.",
		"(Crushing Blow, 18 damage, knocked down)",
		"double damage",
		"(Crushing Blow interrupted)",
		"An ordinary blow never breaks a wind-up",
		"two of its turns",
		"A bash is a counter strike only",
		// review fixes: the fallback, the wasted blow, statuses, spells
		"(Crushing Blow wasted)",
		"whoever it is aiming at",
		"stuns or knocks it down",
		"A spell's damage never breaks one",
	} {
		assert.Contains(t, plain, want)
	}
	for _, alias := range []string{"wind-up", "windup", "windups", "wind-ups", "telegraph", "telegraphs", "crushing-blow", "ogre"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help interrupts", alias)
	}
	for topic, want := range map[string]string{
		"combat":         "wind",
		"statuses":       "Crushing Blow",
		"guardian":       "Crushing Blow",
		"battle-summary": "wind-up",
		"formation":      "ogre",
	} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(page, ""), want, topic)
	}
}
