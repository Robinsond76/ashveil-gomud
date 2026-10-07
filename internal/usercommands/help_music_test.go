package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Camp music's help: the three pages render, are indexed and answer to
// their aliases, and the camp pages point at them.
func TestMusicHelpPages(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := map[string]bool{}
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		listed[topic.Command] = true
	}
	for _, topic := range []string{"music", "instruments", "gigs"} {
		assert.True(t, listed[topic], "help index lists %s", topic)
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Help for", topic)
	}
	for alias, topic := range map[string]string{
		"song": "music", "bard": "music", "camp song": "music", "lute": "instruments", "masterwork": "instruments", "busk": "gigs", "gig": "gigs",
	} {
		want, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, want, got, "help %s is help %s", alias, topic)
	}

	music, _ := GetHelpContents("music")
	plain := tagPattern.ReplaceAllString(music, "")
	for _, want := range []string{"30 gold", "10, 25 and 50", "up to 40%", "Ensemble", "not stack", "likelier", "music learn", "camp music"} {
		assert.Contains(t, plain, want, "help music mentions %q", want)
	}
	gigs, _ := GetHelpContents("gigs")
	for _, want := range []string{"19:00 to 21:00", "two families", "inn gig status"} {
		assert.Contains(t, tagPattern.ReplaceAllString(gigs, ""), want, "help gigs mentions %q", want)
	}

	camp, _ := GetHelpContents("camp")
	for _, want := range []string{"help music", "help instruments", "help gigs", "camp music"} {
		assert.Contains(t, tagPattern.ReplaceAllString(camp, ""), want, "help camp links %q", want)
	}
	inn, _ := GetHelpContents("inn")
	assert.Contains(t, tagPattern.ReplaceAllString(inn, ""), "inn gig")
}
