package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestBalanceHelp(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	old := configs.GetGamePlayConfig()
	t.Cleanup(configs.SetTestGamePlayConfig(old))
	require.NoError(t, configs.ReloadConfig())
	useWorld(t, "default")
	keywords.LoadAliases()
	for topic, wants := range map[string][]string{"strength": {"per effective Strength", "equally strong", "help stat-edge", "+4 at a 40-point lead"}, "progression": {"Fractional gains accumulate", "a warrior more, a wizard less"}, "stat-edge": {"difference", "1/40 of the way", "Speed against Perception"}, "speed": {"help stat-edge", "At equal Speed you hit"}, "smarts": {"help stat-edge"}, "perception": {"help stat-edge"}, "brawling": {"40%"}, "friendly-effects": {"1 per level of the caster", "1 for every 2 levels"}, "tempo": {"before your earned physical turn", "second earned turn"}, "abilities": {"avoid", "tackling that foe again"}, "combat": {"recovery pauses throughout a battle", "help strength", "help progression", "help stat-edge"}} {
		assert.Contains(t, keywords.GetAllHelpTopics(), topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		for _, want := range wants {
			assert.Contains(t, text, want)
		}
		assert.NotContains(t, text, "{{")
	}
}
