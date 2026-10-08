package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The camp and sharpen pages explain the Camp tab's "Before you sleep"
// chores, and the alias leads to the camp page.
func TestCampHelpExplainsChoresBeforeSleep(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	for _, topic := range []string{"camp", "sharpen"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, tagPattern.ReplaceAllString(text, ""), "Before you sleep", topic)
	}
	want, err := GetHelpContents("camp")
	require.NoError(t, err)
	got, err := GetHelpContents("camp-chores")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
