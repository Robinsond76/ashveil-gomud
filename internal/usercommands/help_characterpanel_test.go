package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCharacterPanelHelp: the web client's Character panel (Phase 57) is
// described by help webclient and help skills, and neither promises the
// Jobs section the panel no longer shows.
func TestCharacterPanelHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	web, err := GetHelpContents("webclient")
	require.NoError(t, err)
	web = tagPattern.ReplaceAllString(web, "")
	for _, want := range []string{"Overview", "Gear", "Skills", "Quests", "Effects", "what it does", "Escape closes it and puts you back"} {
		assert.Contains(t, web, want, "help webclient names the Character panel's %s", want)
	}
	assert.NotContains(t, web, "with your jobs", "the Skills tab no longer shows jobs")

	skills, err := GetHelpContents("skills")
	require.NoError(t, err)
	skills = tagPattern.ReplaceAllString(skills, "")
	assert.Contains(t, skills, "what it does")
	assert.Contains(t, skills, "does not show the jobs")

	jobs, err := GetHelpContents("jobs")
	require.NoError(t, err)
	assert.Contains(t, jobs, "web client does not show jobs")
}
