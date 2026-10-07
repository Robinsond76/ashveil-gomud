package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 75: `help inn` names the three rooms and their prices, and the
// room words reach the page.
func TestInnHelpDescribesTheRooms(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	want, err := GetHelpContents("inn")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(want, "")
	for _, phrase := range []string{"inn rest private", "inn rest suite", "Common    5 gold", "Private  15 gold", "Suite    40 gold",
		"Well Rested for 2 hours", "never cuts a longer one short"} {
		assert.True(t, strings.Contains(plain, phrase), "help says %q", phrase)
	}
	for _, alias := range []string{"inns", "inn-rooms", "suite", "private-room"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.True(t, want == got, alias)
	}
}
