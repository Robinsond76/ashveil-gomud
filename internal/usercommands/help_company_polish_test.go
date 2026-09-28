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
	// Phase 32a2: rosters of your own that change.
	assert.Contains(t, company, "Your own recruits")
	assert.Contains(t, company, "Your list is yours")
	assert.Contains(t, company, "then moves on and someone new takes their place")
	assert.Contains(t, company, "company recruit hild")

	recruit := page("recruit")
	assert.Contains(t, recruit, "Your own recruits", "help recruit reaches the same page")

	camp := page("camp")
	assert.Contains(t, camp, "A camp is part of the room")
	assert.Contains(t, camp, "crackling campfire")

	drink := page("drink")
	assert.Contains(t, drink, "Thirst: Hydrated.")
}

// Phase 32e: help company explains company experience, and help experience
// mentions the company listing.
func TestCompanyExperienceHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return tagPattern.ReplaceAllString(text, "")
	}
	company := page("company")
	assert.Contains(t, company, "earns the same amount")
	assert.Contains(t, company, "nothing is split")
	assert.Contains(t, page("experience"), "it lists your company")
}
