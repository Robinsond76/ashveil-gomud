package walking

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
)

// TestPathfinderEasesRoughGroundForTheCompany (33f2): a pathfinder walking
// with the leader cuts the step's strain for every member, by 5% a level,
// asked about the rooms the company walks between.
func TestPathfinderEasesRoughGroundForTheCompany(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.addCompanion(7, 1, "Bran", 1)
	var asked []int
	e.m.pathfinder = func(leader int, roomIDs ...int) (archetypes.Specialist, bool) {
		assert.Equal(t, 7, leader)
		asked = roomIDs
		return archetypes.Specialist{Name: "Bran", Level: 4}, true
	}

	e.steps(7, 1, 2, 1) // forest 50, eased 20% = 40
	assert.Equal(t, []int{2, 1}, asked, "the pathfinder must walk with the company")
	assert.Equal(t, 40, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
	assert.Equal(t, 40, e.m.registry.Carry[7][string(survival.CompanionMemberKey(1))])

	lines := strings.Join(e.m.report(e.users[7], e.rooms[4]), "\n")
	assert.Contains(t, lines, "costs 72 strain per step", "snow 90 eased 20%")
	assert.Contains(t, lines, "Pathfinder: Bran finds easier going, 20% less strain")
}

// TestPathfinderNeverMakesGroundCheaperThanRoad (33f2): easing stops at the
// road's strain, and road or settled ground is never touched.
func TestPathfinderNeverMakesGroundCheaperThanRoad(t *testing.T) {
	e := setup(t)
	e.addUser(t, 7, 1)
	e.m.settings.PathfinderPctPerLevel = 30
	e.m.pathfinder = func(int, ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{IsLeader: true, Level: 4}, true
	}
	e.steps(7, 1, 2, 1) // forest 50 × 10% would be 5; road is 25
	assert.Equal(t, 25, e.m.registry.Carry[7][string(survival.LeaderMemberKey)])
	assert.Contains(t, strings.Join(e.m.report(e.users[7], e.rooms[3]), "\n"), "costs no fatigue", "settled ground stays free")

	e.m.pathfinder = func(int, ...int) (archetypes.Specialist, bool) { return archetypes.Specialist{}, false }
	assert.NotContains(t, strings.Join(e.m.report(e.users[7], e.rooms[2]), "\n"), "Pathfinder", "no pathfinder, no line")
}
