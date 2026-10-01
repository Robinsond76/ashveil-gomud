package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRetreatHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("retreat")
	require.NoError(t, err)
	for _, want := range []string{"retreat [exit]", "One combat round", "+15", "95%", "5 loyalty", "No-go/no-flee", "restart or copyover"} {
		assert.Contains(t, text, want)
	}
	for _, alias := range []string{"withdrawal", "company-withdrawal", "rout", "separation"} {
		other, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, other)
	}
	for _, topic := range []string{"company", "combat", "morale", "guardian", "flee"} {
		hub, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, hub, "help retreat")
	}
}
