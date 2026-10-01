package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnemyGroupHelpPages: the Phase 32c pages render and say how fights
// start and play out: scout (new, and its aliases), attack by group, and
// the pages that point at attack for fights cast, backstab, and shoot no
// longer start.
func TestEnemyGroupHelpPages(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	var combat []string
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" {
			combat = append(combat, topic.Command)
		}
	}
	assert.Contains(t, combat, "scout", "help index lists scout under combat")

	page := func(topic string) string {
		t.Helper()
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		return tagPattern.ReplaceAllString(text, "")
	}

	scout := page("scout")
	assert.Contains(t, scout, "Help for scout")
	assert.Contains(t, scout, "scout [group]")
	for _, alias := range []string{"scouting", "enemy-groups", "groups"} {
		assert.Equal(t, page("scout"), page(alias), "help %s is help scout", alias)
	}

	attack := page("attack")
	assert.Contains(t, attack, "Help for attack")
	assert.Contains(t, attack, "attack [group]")
	assert.Contains(t, attack, "plays out")
	assert.NotContains(t, attack, "attack goblin", "the old GoMud example is gone")

	assert.Contains(t, page("targeting"), "scout")
	assert.Contains(t, page("targeting"), "attack [group]")
	assert.NotContains(t, page("targeting"), "you choose\neach new target yourself", "a lone player turns by strategy now")
	assert.Contains(t, page("combat"), "help scout")
	assert.Contains(t, page("cast"), "outside a fight")
	assert.Contains(t, page("shoot"), "doesn't start a fight")
	assert.Contains(t, page("battle-summary"), "The fight with a band of ruffians is over")
}
