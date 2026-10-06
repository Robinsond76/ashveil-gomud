package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPoisonsHelp (Phase 43b): the page renders through help, is listed
// under combat with its aliases, is linked from the combat and camp pages,
// and names every poison and command the phase ships.
func TestPoisonsHelp(t *testing.T) {
	useWorld(t, "default")
	keywords.LoadAliases()

	listed := false
	for _, topic := range keywords.GetAllHelpTopicInfo() {
		if topic.Category == "combat" && topic.Command == "poisons" {
			listed = true
		}
	}
	assert.True(t, listed, "help index lists poisons under combat")

	page, err := GetHelpContents("poisons")
	require.NoError(t, err)
	plain := strings.ToLower(tagPattern.ReplaceAllString(page, ""))
	assert.Contains(t, plain, "help for")
	for _, p := range items.Poisons {
		assert.Contains(t, plain, p.ID, "the page names %s", p.ID)
	}
	for _, cmd := range []string{"coat status", "coat clear", "camp poison assign", "camp poison unassign", "camp poison preview", "camp poison apply"} {
		assert.Contains(t, plain, cmd)
	}
	assert.Contains(t, plain, "10 real minutes")
	assert.Contains(t, plain, "40%")

	for _, alias := range []string{"poison", "coat", "coating", "weapon-poison", "bitterleaf", "leechbane", "leadroot", "mirethorn", "vials"} {
		got, err := GetHelpContents(alias)
		require.NoError(t, err, alias)
		assert.Equal(t, page, got, "help %s is help poisons", alias)
	}

	for _, topic := range []string{"combat", "camp", "statuses"} {
		text, err := GetHelpContents(topic)
		require.NoError(t, err, topic)
		assert.Contains(t, text, "help poisons", "%s links to help poisons", topic)
	}
}

// TestShippedVialsMatchTheCatalogue: every poison's vial is a shipped item.
func TestShippedVialsMatchTheCatalogue(t *testing.T) {
	useWorld(t, "default")
	items.LoadDataFiles()
	for _, p := range items.Poisons {
		spec := items.GetItemSpec(p.VialID)
		require.NotNil(t, spec, "vial item %d for %s", p.VialID, p.ID)
		assert.Contains(t, spec.Name, p.ID)
		assert.Equal(t, 1, spec.Uses, "one dose")
	}
}
