package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupedMobDisplaySoloMobRendersItsOwnLine(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, rawName: "goblin", display: "a rusty goblin"},
	})

	assert.Equal(t, []string{"a rusty goblin"}, lines)
}

func TestGroupedMobDisplayGroupsSharedTagIntoOneAggregateLine(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a rusty goblin"},
		{instanceId: 2, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a scarred goblin"},
		{instanceId: 3, groups: []string{"goblin-raiders"}, rawName: "goblin", display: "a young goblin"},
	})

	assert.Equal(t, []string{"a pack of 3 goblins"}, lines)
}

func TestGroupedMobDisplayMixedNamesUseGenericLabel(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, groups: []string{"mixed-pack"}, rawName: "goblin", display: "a rusty goblin"},
		{instanceId: 2, groups: []string{"mixed-pack"}, rawName: "wolf", display: "a gray wolf"},
	})

	assert.Equal(t, []string{"a pack of 2 creatures"}, lines)
}

func TestGroupedMobDisplayKeepsUngroupedMobsSeparate(t *testing.T) {
	lines := groupedMobDisplay([]hostileMobDisplay{
		{instanceId: 1, rawName: "bear", display: "a hungry bear"},
		{instanceId: 2, rawName: "bear", display: "a sleepy bear"},
	})

	// No shared Groups tag: two solo parties, not one aggregate line.
	assert.ElementsMatch(t, []string{"a hungry bear", "a sleepy bear"}, lines)
}

func TestPluralize(t *testing.T) {
	assert.Equal(t, "goblins", pluralize("goblin"))
	assert.Equal(t, "foxes", pluralize("fox"))
	assert.Equal(t, "", pluralize(""))
}
