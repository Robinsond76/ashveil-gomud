package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreationHelp (Phase 72a): help appearance and help lifestory render,
// answer to their aliases, are indexed, and are linked from help adventure.
func TestCreationHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	pages := map[string]struct {
		contains []string
		aliases  []string
	}{
		"appearance": {
			contains: []string{"Help for appearance", "appearance edit", "at an inn", "free", "160 characters"},
			aliases:  []string{"looks", "description", "describe", "mirror"},
		},
		"lifestory": {
			contains: []string{"Help for lifestory", "lifestory choose", "+2", "at most +3 in all", "keepsake", "background"},
			aliases:  []string{"backstory", "background", "backgrounds", "homeland", "upbringing", "trade"},
		},
	}
	for topic, want := range pages {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		plain := tagPattern.ReplaceAllString(text, "")
		for _, s := range want.contains {
			assert.Contains(t, plain, s, topic)
		}
		for _, alias := range want.aliases {
			got, err := GetHelpContents(alias)
			require.NoError(t, err, alias)
			assert.Equal(t, text, got, alias)
		}
		var listed bool
		for _, info := range keywords.GetAllHelpTopicInfo() {
			if info.Command == topic {
				listed = true
			}
		}
		assert.True(t, listed, topic+" is in the help index")
	}

	hub, err := GetHelpContents("adventure")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(hub, "")
	assert.Contains(t, plain, "help appearance")
	assert.Contains(t, plain, "help lifestory")
}
