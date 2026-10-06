package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/loot"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncounterFightEndsInOneCacheAndTheSpoilsLine (Phase 37): a company
// beats a random encounter's group through the real combat loop; the
// group's cache rolls exactly once, the claimant holds it, and the leader's
// spoils are noted for the battle summary.
func TestEncounterFightEndsInOneCacheAndTheSpoilsLine(t *testing.T) {
	f := newBalanceFightWithOptions(t, 10, companyDefault, enemyDefault,
		balanceFightOptions{EnemyCount: 3, EnemyLevels: []int{7}, Coordination: 1})
	road := f.road
	road.Zone = "brawl"
	t.Cleanup(rooms.SetTestZoneConfig(&rooms.ZoneConfig{Name: "brawl", Encounters: encounters.ZoneConfig{Band: encounters.Band{Low: 7, High: 9}}}))
	for _, id := range f.enemies {
		m := mobs.GetInstance(id)
		m.EncounterOwner, m.EncounterID = 7, "encounter:test"
		m.Character.Gold = 0
	}
	loot.TakeSpoils(7)

	res := f.run()
	require.True(t, res.Won, "a level-10 company beats three level-7 foes")

	cacheGold := 0
	for _, c := range road.Corpses {
		cacheGold += c.Gold
	}
	assert.GreaterOrEqual(t, cacheGold, 7*4, "one cache of gold by the band's level")
	assert.Less(t, cacheGold, 7*4+7*2+1+1, "only one cache, however the foes fell")
	spoils := loot.TakeSpoils(7)
	require.NotEmpty(t, spoils, "the leader's spoils were noted")
	assert.Contains(t, spoils[len(spoils)-1], "gold")
}
