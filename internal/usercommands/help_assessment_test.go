package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssessmentHelpPages (Phase 33i1): help assessment renders and is
// indexed under combat with its aliases; consider is the company's page
// now (a template, not GoMud's one-on-one odds); scout, combat, and
// webclient point to the assessment.
func TestAssessmentHelpPages(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" {
			combat = append(combat, topic.Command)
		}
	}
	assert.Contains(t, combat, "assessment", "help index lists assessment under combat")
	assert.Contains(t, combat, "consider")

	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return tagPattern.ReplaceAllString(text, "")
	}

	a := page("assessment")
	for _, want := range []string{"Help for assessment", "scout [group]", "consider [enemy]",
		"an easy fight", "a fair fight", "a hard fight", "a grave risk", "hopeless",
		"could go either way", "never numbers", "Pets", "allied companies", "Outlook"} {
		assert.Contains(t, a, want)
	}
	for _, alias := range []string{"assess", "odds", "chances", "risk", "outlook"} {
		assert.Equal(t, a, page(alias), "help %s is help assessment", alias)
	}

	c := page("consider")
	assert.Contains(t, c, "Help for consider")
	assert.Contains(t, c, "consider [enemy]")
	assert.Contains(t, c, "whole group")
	assert.NotContains(t, c, "YOU WILL DIE", "the one-on-one odds are retired")

	for _, topic := range []string{"scout", "combat", "webclient"} {
		assert.Contains(t, page(topic), "help assessment", topic)
	}
	assert.Contains(t, page("combat"), "consider [enemy]")
}
