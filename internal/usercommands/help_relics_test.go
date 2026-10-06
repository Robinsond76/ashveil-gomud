package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 36d: the relics help page renders, is indexed with its aliases,
// is linked from the pages it touches, points only at pages that exist, and
// states the numbers the game uses.
func TestRelicsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && topic.Command == "relics" && !topic.AdminOnly {
			indexed = true
		}
	}
	assert.True(t, indexed, "help index lists relics")

	text, err := GetHelpContents("relics")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for relics")
	assert.NotContains(t, plain, "{{")
	for _, m := range regexp.MustCompile(`help ([a-z-]+)`).FindAllStringSubmatch(plain, -1) {
		_, err := GetHelpContents(m[1])
		assert.NoError(t, err, "help relics points at help %s", m[1])
	}

	for _, alias := range []string{"legendary", "legendaries", "signature", "sets", "set bonus", "set items", "bad luck", "relic"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help relics", alias)
	}

	for hub, want := range map[string][]string{
		"rarity":         {"help relics"},
		"loot":           {"help relics"},
		"equipmenttiers": {"help relics"},
		"encounters":     {"help relics"},
		"itemlevel":      {"help relics"},
		"identify":       {"help relics"},
		"salvage":        {"help relics"},
		"sell":           {"help relics"},
		"inventory":      {"help relics", "Relics worn"},
	} {
		page, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		for _, w := range want {
			assert.Contains(t, page, w, "help %s mentions %s", hub, w)
		}
	}

	// The numbers the page states are the game's.
	assert.Contains(t, plain, "the 20th kill")
	assert.Equal(t, 20, loot.BadLuckKills)
	specs := shippedItems(t)
	reaper := specs[50001]
	require.NotNil(t, reaper)
	assert.Equal(t, "Ashen Reaper", reaper.Name)
	for _, sentence := range classes.DescribeGearEffects(map[string]int{classes.Wounded: reaper.Relic.Effects[classes.Wounded]}) {
		assert.Contains(t, strings.Join(strings.Fields(plain), " "), "Reaping ("+sentence+")", "the example signature is the shipped one's")
	}
	circlet := specs[50002]
	require.NotNil(t, circlet)
	for _, sentence := range classes.DescribeGearEffects(map[string]int{classes.Bargain: 1}) {
		assert.Contains(t, strings.Join(strings.Fields(plain), " "), "Grave Pact ("+strings.ReplaceAll(sentence, "it at", "its wearer at")+")", "the second example is the shipped one's")
	}

	// A tier 5 kite shield is the 17 the tier page states.
	tiers, err := GetHelpContents("equipmenttiers")
	require.NoError(t, err)
	assert.Equal(t, 17, specs[20315].DamageReduction)
	assert.Contains(t, tagPattern.ReplaceAllString(tiers, ""), "tier 5 one 17")
}
