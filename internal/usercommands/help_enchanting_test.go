package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 71: the enchanting help page renders, is indexed with its aliases,
// is linked from the pages it touches, points only at pages that exist, and
// states the numbers and trophies the world ships.
func TestEnchantingHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	indexed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "company" && topic.Command == "enchanting" && !topic.AdminOnly {
			indexed = true
		}
	}
	assert.True(t, indexed, "help index lists enchanting")

	text, err := GetHelpContents("enchanting")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	assert.Contains(t, plain, "Help for enchanting")
	assert.NotContains(t, plain, "{{")
	for _, m := range regexp.MustCompile(`help ([a-z-]+)`).FindAllStringSubmatch(plain, -1) {
		_, err := GetHelpContents(m[1])
		assert.NoError(t, err, "help enchanting points at help %s", m[1])
	}
	for _, alias := range []string{"imbue", "imbuing", "enchanter", "trophy enchanting", "enchant gear"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help enchanting", alias)
	}
	skill, err := GetHelpContents("enchant")
	require.NoError(t, err)
	assert.NotEqual(t, text, skill, "the enchant skill keeps its own page")

	for hub, want := range map[string][]string{
		"relics":   {"help enchanting"},
		"goods":    {"help enchanting"},
		"salvage":  {"help enchanting"},
		"bestiary": {"help enchanting"},
	} {
		page, err := GetHelpContents(hub)
		require.NoError(t, err, hub)
		for _, w := range want {
			assert.Contains(t, page, w, "help %s mentions %s", hub, w)
		}
	}

	// The numbers on the page are the world's.
	flat := strings.Join(strings.Fields(plain), " ")
	assert.Contains(t, flat, "It costs 25 gold for each tier")
	assert.Equal(t, 25, items.EnchantFeePerTier)
	for _, spec := range shippedItems(t) {
		if spec.Trophy == nil {
			continue
		}
		assert.Contains(t, strings.ToLower(flat), spec.Name, "the page lists %s", spec.Name)
		assert.Contains(t, []int{20, 25}, spec.Trophy.Chance, "the page says about 1 in 5 or 1 in 4 for %s", spec.Name)
		fx := strings.Join(classes.DescribeGearEffects(spec.Trophy.Effects), "; ")
		want := strings.ReplaceAll(fx, " damage reduction on top of worn armor", " damage reduction")
		assert.Contains(t, flat, want, "the page says what %s gives", spec.Name)
	}
}
