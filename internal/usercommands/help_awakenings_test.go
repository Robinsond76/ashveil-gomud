package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 67: the awakenings help page renders, is indexed with its aliases,
// is linked from the relic pages, points only at pages that exist, and
// states the numbers the game uses.
func TestAwakeningsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && topic.Command == "awakenings" && !topic.AdminOnly {
			indexed = true
		}
	}
	assert.True(t, indexed, "help index lists awakenings")

	text, err := GetHelpContents("awakenings")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for awakenings")
	assert.NotContains(t, plain, "{{")
	for _, m := range regexp.MustCompile(`help ([a-z-]+)`).FindAllStringSubmatch(plain, -1) {
		_, err := GetHelpContents(m[1])
		assert.NoError(t, err, "help awakenings points at help %s", m[1])
	}
	for _, alias := range []string{"awakening", "awaken", "awakened", "relic awakenings", "awakened relic"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help awakenings", alias)
	}
	for hub, want := range map[string][]string{
		"relics":    {"help awakenings"},
		"chronicle": {"help awakenings", "awakenings"},
		"inventory": {"help awakenings"},
	} {
		page, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		for _, w := range want {
			assert.Contains(t, page, w, "help %s mentions %s", hub, w)
		}
	}

	// The page's example is the shipped relic's own.
	flat := strings.Join(strings.Fields(plain), " ")
	ogrebane := shippedItems(t)[50007]
	require.NotNil(t, ogrebane)
	giant := ogrebane.Relic.Awakenings[0]
	assert.Equal(t, "Giant-Slayer", giant.Name)
	assert.Contains(t, flat, "Ogrebane's Giant-Slayer: 15 ogres")
	assert.Equal(t, 15, giant.Need())
	assert.Equal(t, "ogres", giant.Target)
	assert.Contains(t, flat, "Ogrebane woke Giant-Slayer: +1 damage on every landed blow")
	assert.Equal(t, map[string]int{"damage": 1}, giant.Effects)
	assert.Contains(t, flat, "at most a third of what that effect may ever reach")
}
