package usercommands

import (
	"regexp"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36a: the loot item model's help pages render, are indexed under the
// company category with their aliases, are linked from their hubs, and
// point only at pages that exist.
func TestLootItemModelHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && !topic.AdminOnly {
			indexed[topic.Command] = true
		}
	}
	topics := []string{"rarity", "quality", "itemlevel", "affixes", "identify", "scribe"}
	pointer := regexp.MustCompile(`help ([a-z-]+)`)
	for _, topic := range topics {
		assert.True(t, indexed[topic], "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		assert.Contains(t, plain, "Help for "+topic)
		assert.NotContains(t, plain, "{{", topic)
		for _, m := range pointer.FindAllStringSubmatch(plain, -1) {
			_, err := GetHelpContents(m[1])
			assert.NoError(t, err, "help %s points at help %s", topic, m[1])
		}
	}

	aliases := map[string]string{
		"rare": "rarity", "epic": "rarity", "unidentified": "rarity", "level requirement": "rarity",
		"fine": "quality", "exquisite": "quality", "ilvl": "itemlevel", "item level": "itemlevel",
		"prefix": "affixes", "suffix": "affixes", "identification": "identify",
	}
	for alias, topic := range aliases {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	// The hubs and the pages this phase changed link to the new pages.
	for hub, want := range map[string][]string{
		"equipment":     {"help rarity", "help identify", "help scribe"},
		"loot":          {"help rarity", "help scribe"},
		"skills":        {"help scribe", "Scribe"},
		"company-train": {"help scribe", "Scribe"},
		"autoskill":     {"scribe", "help scribe"},
		"camp":          {"help identify"},
	} {
		text, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		for _, w := range want {
			assert.Contains(t, text, w, "help %s mentions %s", hub, w)
		}
	}

	// The numbers the pages state are the ones the game uses.
	text, err := GetHelpContents("scribe")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"8 for a Rare", "15 for an Epic", "25 for a Legendary or Set", "Magic Academy", "company train [member] scribe"} {
		assert.Contains(t, plain, want)
	}
	text, err = GetHelpContents("quality")
	require.NoError(t, err)
	plain = tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"crude         -20%", "exquisite     +30%", "x4"} {
		assert.Contains(t, plain, want)
	}
}
