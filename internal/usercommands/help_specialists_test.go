package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpecialistsHelp (33f2): the expedition-skill pages render through
// help, by name and alias, with the numbers that matter, and the pages
// they change point to them.
func TestSpecialistsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return text
	}
	specialists := page("specialists")
	for _, want := range []string{"company specialists", "1, 10, 20, and 30", "help trail", "help keeneye", "help pathfinder", "help forecast", "help haggle", "while you are in a battle"} {
		assert.Contains(t, specialists, want)
	}
	assert.Equal(t, specialists, page("company-specialists"))

	trail := page("trail")
	assert.Contains(t, trail, "20% chance per")
	assert.Contains(t, trail, "two rooms away")
	assert.Equal(t, trail, page("track"), "help track opens Read the Trail")

	keen := page("keeneye")
	assert.Contains(t, keen, "(level x 20) + (Perception / 4)")
	assert.Equal(t, keen, page("search"), "help search opens Keen Eye")
	assert.Equal(t, keen, page("secret-exits"))

	assert.Contains(t, page("pathfinder"), "5% less strain per level")
	assert.Contains(t, page("forecast"), "never wrong")
	assert.Equal(t, page("forecast"), page("weather-sense"))
	assert.Contains(t, page("haggle"), "2% better per level")

	assert.Contains(t, page("company"), "company specialists")
	assert.Contains(t, page("weather"), "help forecast")
	assert.Contains(t, page("market"), "help haggle")
	assert.Contains(t, page("strain"), "help pathfinder")
	assert.Contains(t, page("autoskill"), "keeneye")
	assert.Contains(t, page("travel"), "help trail")
	assert.NotContains(t, page("stash"), "search", "nothing searches stashes now")
}

// TestCampSpecialistsHelp (33f3): the camp-specialist pages render with
// their numbers, by name and alias, and the pages they change point to
// them.
func TestCampSpecialistsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return text
	}
	watch := page("campwatch")
	assert.Contains(t, watch, "15% chance")
	assert.Contains(t, watch, "25% chance per")
	assert.Equal(t, watch, page("raiders"))
	assert.Contains(t, page("fieldsmith"), "20 +\n5 x level")
	assert.Contains(t, page("vigil"), "up to a loyalty of 60")
	assert.Contains(t, page("forage"), "1 find, plus 1 more for every\n2 levels")
	assert.Contains(t, page("camp"), "camp cook")
	assert.NotContains(t, page("camp"), "can't cook at camp")
	assert.Contains(t, page("cooking"), "Cooking at camp")
	assert.Contains(t, page("sharpen"), "help fieldsmith")
	assert.Contains(t, page("specialists"), "help campwatch")
}
