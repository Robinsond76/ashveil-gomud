package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPhase34dPlayerHelpRenders(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	for topic, expected := range map[string]string{
		"cooking": "manual camp capability",
		"skills":  "Automatic combat abilities", "conditions": "Light wounds last until fight end",
		"company": "active effects and wounds", "specialists": "field and camp eligibility", "webclient": "persistent bonuses",
	} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, expected, topic)
		assert.Contains(t, text, "Help for", topic)
	}
}
