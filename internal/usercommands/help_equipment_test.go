package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEquipmentTreasuryAndLootHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	expected := map[string][]string{
		"equipment": {"company compare", "company equip", "company remove", "exact reference", "no equipment presets", "battle"},
		"treasury":  {"shares one treasury", "Living companions", "Other players"},
		"loot":      {"2 game hours", "2 more game hours", "autoloot on", "starts off"},
		"gearup":    {"deliberate", "No automatic"},
		"give":      {"giving to them moves nothing", "shared cargo or treasury"},
		"ask":       {"compatibility orders are refused in shared mode", "company equip/remove"},
	}
	for topic, wants := range expected {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		for _, want := range wants {
			assert.Contains(t, text, want, topic)
		}
	}
	for _, topic := range []string{"company-equipment", "company treasury", "autoloot"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.NotEmpty(t, text)
	}
}
