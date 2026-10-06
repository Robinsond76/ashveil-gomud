package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 52: the tents are on `help camp gear`, named by their aliases, and
// the pages they touch say how a tent changes a rest.
func TestCampTentsHelpDescribesEachTent(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	want, err := GetHelpContents("campgear")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, tent := range camping.Tents {
		assert.True(t, strings.Contains(strings.ToLower(plain), tent.Name), "help names the %s", tent.Kind)
	}
	for _, phrase := range []string{"camp tent", "half as often", "half again", "Well Rested", "7 and a half minutes", "never buy gear back, tents included",
		"no penalty at all", "pack\n  horse", "camp tent clear"} {
		assert.True(t, strings.Contains(plain, phrase), "help says %q", phrase)
	}
	for _, alias := range []string{"tents", "camp tent", "fur tent", "camouflaged tent", "large tent", "pavilion tent"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.True(t, want == got, alias)
	}
	for _, topic := range []string{"camp", "campwatch", "inn"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.True(t, strings.Contains(tagPattern.ReplaceAllString(text, ""), "tent"), "help %s mentions the tents", topic)
	}
	camp, err := GetHelpContents("camp")
	require.NoError(t, err)
	assert.True(t, strings.Contains(tagPattern.ReplaceAllString(camp, ""), "camp tent [canvas|fur|camouflaged|large|clear]"))
}
