package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobparty"
	"github.com/stretchr/testify/assert"
)

func entry(id int, name, group, display string) hostileMobDisplay {
	return hostileMobDisplay{summary: mobparty.MobSummary{InstanceId: id, Name: name, SpawnGroup: group}, display: display}
}

func TestGroupedMobDisplaySoloMobRendersItsOwnLine(t *testing.T) {
	solo, groups := groupedMobDisplay([]hostileMobDisplay{entry(1, "goblin", "", "a rusty goblin")}, nil)
	assert.Equal(t, []string{"a rusty goblin"}, solo)
	assert.Empty(t, groups)
}

func TestGroupedMobDisplayOneKindNeedsNoList(t *testing.T) {
	solo, groups := groupedMobDisplay([]hostileMobDisplay{
		entry(1, "goblin", "spawn:1:1", "a rusty goblin"),
		entry(2, "goblin", "spawn:1:1", "a scarred goblin"),
		entry(3, "goblin", "spawn:1:1", "a young goblin"),
	}, nil)
	assert.Empty(t, solo)
	assert.Equal(t, []string{`<ansi fg="mobname">A band of goblins</ansi> (3).`}, groups)
}

func TestGroupedMobDisplayMixedGroupListsItsMembers(t *testing.T) {
	_, groups := groupedMobDisplay([]hostileMobDisplay{
		entry(1, "ruffian", "spawn:1:1", "a ruffian"),
		entry(2, "cutpurse", "spawn:1:1", "a cutpurse"),
		entry(3, "ruffian", "spawn:1:1", "a ruffian"),
		entry(4, "rat", "spawn:1:1", "a rat"),
	}, func([]int) string { return "fighting you" })
	assert.Equal(t, []string{`<ansi fg="mobname">A band of ruffians</ansi> (4): two ruffians, a cutpurse, and a rat (fighting you).`}, groups)
}

func TestGroupedMobDisplayRepeatsAreNumbered(t *testing.T) {
	_, groups := groupedMobDisplay([]hostileMobDisplay{
		entry(1, "ruffian", "spawn:1:1", "a ruffian"),
		entry(2, "ruffian", "spawn:1:1", "a ruffian"),
		entry(3, "ruffian", "spawn:1:2", "a ruffian"),
		entry(4, "ruffian", "spawn:1:2", "a ruffian"),
	}, nil)
	assert.Equal(t, []string{
		`<ansi fg="mobname">A band of ruffians</ansi> (2).`,
		`<ansi fg="mobname">A second band of ruffians</ansi> (2).`,
	}, groups)
}

func TestGroupedMobDisplayALastSurvivorKeepsTheName(t *testing.T) {
	e := entry(4, "rat", "spawn:1:1", "a rat")
	e.summary.GroupName = "a band of ruffians"
	solo, groups := groupedMobDisplay([]hostileMobDisplay{e}, nil)
	assert.Empty(t, solo)
	assert.Equal(t, []string{`<ansi fg="mobname">A band of ruffians</ansi> (1): a rat.`}, groups)
}

func TestGroupedMobDisplayKeepsUngroupedMobsSeparate(t *testing.T) {
	solo, groups := groupedMobDisplay([]hostileMobDisplay{
		entry(1, "bear", "", "a hungry bear"),
		entry(2, "bear", "", "a sleepy bear"),
	}, nil)
	// No shared group: two lone mobs, not one group line.
	assert.ElementsMatch(t, []string{"a hungry bear", "a sleepy bear"}, solo)
	assert.Empty(t, groups)
}
