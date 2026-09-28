package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 32a: help company, camp, and drink render with what 32a changed.
func TestCompanyPolishHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return tagPattern.ReplaceAllString(text, "")
	}

	company := page("company")
	assert.Contains(t, company, "Moving together")
	assert.Contains(t, company, "leads their company towards the east exit.")
	assert.Contains(t, company, "look [candidate]")
	assert.Contains(t, company, "won't join you")

	camp := page("camp")
	assert.Contains(t, camp, "A camp is part of the room")
	assert.Contains(t, camp, "crackling campfire")

	drink := page("drink")
	assert.Contains(t, drink, "Thirst: Hydrated.")
}
