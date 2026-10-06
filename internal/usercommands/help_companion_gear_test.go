package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanionGearHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	text, err := GetHelpContents("companion-gear")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"Help for companion gear", "company equip [member] [item]", "company remove [member] [slot]",
		"company gear [member]", "When a companion leaves", "comes back to your cargo", "never earns you free gear",
		"doll wield", "Company > Inventory", "drag", "Character > Gear"} {
		assert.Contains(t, plain, want)
	}
	assert.NotContains(t, plain, "{{")
	for _, alias := range []string{"companion gear", "buff companions", "disband gear", "company gear"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, text, got, "help %s is help companion-gear", alias)
	}
	for _, topic := range []string{"equipment", "company"} {
		page, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, page, "help companion-gear", "%s links to help companion-gear", topic)
	}
}
