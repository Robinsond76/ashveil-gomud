package usercommands

import (
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkillOverHPHelp (Phase 35a2): the new evasion and shields pages and
// the pages the phase changed render through help with the shipped numbers,
// by topic and by alias.
func TestSkillOverHPHelp(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	old := configs.GetGamePlayConfig()
	t.Cleanup(configs.SetTestGamePlayConfig(old))
	require.NoError(t, configs.ReloadConfig())
	useWorld(t, "default")
	keywords.LoadAliases()

	pages := map[string][]string{
		"evasion": {"Help for evasion", "Attack", "Evasion", "Attack 12 -> 13   Evasion 11 -> 12",
			"1/14 of a full edge", "To hit        75%       10%     95%", "Dodge         8%       3%      40%",
			"Block         28% + shield  8%      55%", "hits 89% of the time", "28% of the time and meets a 30% dodge", "costs 10 Attack"},
		"shields": {"Help for shields", "Buckler", "Tower", "bucklers only", "Cleric, Rogue, Wizard",
			"staffs, rods and maces", "may carry a holy", "an iron shield blocks 38% and a buckler 31%", "between 8% and 55%"},
		"armor": {"(5 damage, 2 absorbed)", "8% fewer turns, 20% less dodge", "20% fewer turns, 50% less dodge",
			"Warrior                    heavy", "Ranger                     medium", "lose 10 Attack and 10 Evasion",
			"1 more round to chant", "help shields", "Bulk is not weight", "(a breastplate, a tower shield)"},
		"health":            {"base of 48", "Warrior     10           1               59        88", "help evasion"},
		"tempo":             {"Medium armor then takes", "8% of it and heavy armor 20%", "help armor"},
		"attack":            {"Attack against your target's Evasion", "help evasion"},
		"perception":        {"8% at an even match", "help evasion"},
		"strength":          {"weapon's own dice matter most", "help shields"},
		"abilities":         {"Attack against the foe's Evasion", "1 more for every 6 levels"},
		"combat":            {"help evasion", "help shields", "help armor", "armor bulk"},
		"equip":             {"rangers only bucklers", "staffs, rods and maces", "warning"},
		"company-inventory": {"company equip", "help shields"},
		"warrior":           {"any shield", "heavy armor"},
		"ranger":            {"buckler", "medium armor"},
		"archetype":         {"How each class fights", "staffs, rods and maces; a holy symbol"},
		"heal":              {"10 to 16 health", "holy symbol"},
		"experience":        {"Attack and Evasion"},
		"stat-edge": {"Dodge        Perception against Perception: 8% even, 3% to 40%", "Block        Strength against Strength: 28% + shield even",
			"is added to the stat edge of", "every chance here except critical hits", "help evasion"},
		"interrupts": {"more skilled the bearer is than the attacker"},
	}
	for topic, wants := range pages {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, want := range wants {
			assert.Contains(t, plain, want, topic)
		}
		assert.NotContains(t, plain, "{{", topic+": every config number rendered")
		assert.NotContains(t, plain, "Phase", topic+": no phase numbers in player help")
	}
	for _, topic := range []string{"evasion", "shields"} {
		assert.Contains(t, keywords.GetAllHelpTopics(), topic)
	}

	for page, aliases := range map[string][]string{
		"evasion": {"attack skill", "combat skill"},
		"shields": {"shield", "buckler", "block"},
		"armor":   {"bulk", "heavy armor"},
	} {
		want, err := GetHelpContents(page)
		require.NoError(t, err)
		for _, alias := range aliases {
			got, err := GetHelpContents(alias)
			require.NoError(t, err, alias)
			assert.Equal(t, want, got, "help %s is help %s", alias, page)
		}
	}
}
