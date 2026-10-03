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
	for topic, wants := range map[string][]string{"strength": {"1.75", "+20", "equally strong"}, "progression": {"level 5", "0.7", "Fractional gains accumulate"}, "tempo": {"before your earned physical turn", "second earned turn"}, "abilities": {"avoid", "tackling that foe again"}, "combat": {"help strength", "help progression", "recovery pauses throughout a battle"}} {
		assert.Contains(t, keywords.GetAllHelpTopics(), topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err)
		for _, want := range wants {
			assert.Contains(t, text, want)
		}
		assert.NotContains(t, text, "{{")
	}
}
