package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 33c owner review: retreat is the one way out of a battle, flee is
// its other name, and nobody is separated.
func TestRetreatHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()
	text, err := GetHelpContents("retreat")
	require.NoError(t, err)
	plain := tagPattern.ReplaceAllString(text, "")
	for _, want := range []string{"retreat [exit]", "flee [exit] works too", "only way out of a battle", "One combat round", "+15", "between 30% and 95%", "Every active foe of your battle pursues", "pins the whole company", "names who is pinned", "set wimpy", "restart or", "copyover"} {
		assert.Contains(t, plain, want)
	}
	// The withdrawal itself never splits the company; only a companion that
	// didn't join the order is separated afterwards (Phase 33h3).
	order := plain[:strings.Index(plain, "Fallen companions stay")]
	for _, gone := range []string{"emergency", "separate", "5 loyalty"} {
		assert.NotContains(t, order, gone)
	}
	assert.NotContains(t, plain, "emergency")
	assert.NotContains(t, plain, "5 loyalty")
	for _, alias := range []string{"flee", "withdrawal", "company-withdrawal", "escape"} {
		other, err := GetHelpContents(alias)
		require.NoError(t, err)
		assert.Equal(t, text, other, "help %s is help retreat", alias)
	}
	for _, topic := range []string{"company", "combat", "morale", "guardian", "targeting", "break", "set-wimpy", "statuses"} {
		hub, err := GetHelpContents(topic)
		require.NoError(t, err)
		assert.Contains(t, tagPattern.ReplaceAllString(hub, ""), "retreat", topic)
		assert.NotContains(t, hub, "emergency", topic)
	}
	// 33a review convention: the hub's retreat note sits above "See also".
	for _, topic := range []string{"morale", "guardian", "company"} {
		hub, _ := GetHelpContents(topic)
		assert.Less(t, strings.Index(hub, "help retreat"), strings.LastIndex(hub, "See also"), topic)
	}
}
