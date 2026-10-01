package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMemberOrderHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	for _, topic := range []string{"ask", "company", "combat", "equip", "remove", "eat", "drink", "use", "tame", "protection"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "Help for", topic)
		assert.Contains(t, text, "battle", topic)
	}
	ask, err := GetHelpContents("ask")
	require.NoError(t, err)
	assert.Contains(t, ask, "[member]")
	assert.Contains(t, ask, "attack, throw and cast orders are refused")
	assert.Contains(t, ask, "Temporary charmed")
}
