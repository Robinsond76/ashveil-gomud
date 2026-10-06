package death

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func shippedWorld() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "_datafiles", "world", "default")
}

func shippedRoom(t *testing.T, path string) *rooms.Room {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(shippedWorld(), "rooms", path))
	require.NoError(t, err)
	room := &rooms.Room{}
	require.NoError(t, yaml.Unmarshal(data, room))
	return room
}

// shippedSettings parses the shipped config overlay.
func shippedSettings(t *testing.T) settings {
	t.Helper()
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	values := map[string]any{}
	require.NoError(t, yaml.Unmarshal(data, &values))
	return parseSettings(configMap(values))
}

// TestShippedChurches pins the Phase 25a content: Frostfang's Sanctuary and
// Dunmar's new chapel are churches of their cities, the chapel is reachable
// from the market square, its keeper can't be farmed, and the shipped
// config names both with the Sanctuary as the fallback.
func TestShippedChurches(t *testing.T) {
	sanctuary := shippedRoom(t, "frostfang/18.yaml")
	chapel := shippedRoom(t, "dunmar/2007.yaml")
	square := shippedRoom(t, "dunmar/2004.yaml")

	assert.Equal(t, "Frostfang", sanctuary.Zone)
	assert.True(t, sanctuary.HasTag(domain.ChurchTag))
	assert.Equal(t, 2007, chapel.RoomId)
	assert.Equal(t, "Dunmar", chapel.Zone)
	assert.True(t, chapel.HasTag(domain.ChurchTag))
	assert.Equal(t, 2007, square.Exits["east"].RoomId)
	assert.Equal(t, 2004, chapel.Exits["west"].RoomId)
	require.Len(t, chapel.SpawnInfo, 1)
	assert.Equal(t, 65, chapel.SpawnInfo[0].MobId)

	data, err := os.ReadFile(filepath.Join(shippedWorld(), "mobs", "dunmar", "65-sister_maren.yaml"))
	require.NoError(t, err)
	keeper := mobs.Mob{}
	require.NoError(t, yaml.Unmarshal(data, &keeper))
	assert.Equal(t, mobs.MobId(65), keeper.MobId)
	assert.Equal(t, "Sister Maren", keeper.Character.Name)
	assert.False(t, keeper.Hostile)
	assert.Zero(t, keeper.ItemDropChance, "drops nothing")
	assert.Empty(t, keeper.Character.Items)
	assert.Zero(t, keeper.Character.Gold)
	assert.Zero(t, keeper.Character.Equipment.Weapon.ItemId, "and wields nothing")

	s := shippedSettings(t)
	church, ok := s.registry.ChurchFor("Frostfang")
	assert.True(t, ok)
	assert.Equal(t, 18, church)
	church, ok = s.registry.ChurchFor("Dunmar")
	assert.True(t, ok)
	assert.Equal(t, 2007, church)
	assert.Equal(t, 18, s.fallbackID)
	assert.Equal(t, 50, s.vitalsPct)
}

// TestShippedShaman pins the Phase 25b content: Fernhollow is a village
// west of the Fork at the Black Oak, its lodge carries the shaman tag and
// Old Wenna, who can't be farmed; the shipped config names every
// settlement's keeper, and the village is never a checkpoint.
func TestShippedShaman(t *testing.T) {
	fork := shippedRoom(t, "old_kings_road/2002.yaml")
	green := shippedRoom(t, "fernhollow/2008.yaml")
	lodge := shippedRoom(t, "fernhollow/2009.yaml")
	assert.Equal(t, 2008, fork.Exits["west"].RoomId)
	assert.Equal(t, 2002, green.Exits["east"].RoomId)
	assert.Equal(t, 2009, green.Exits["west"].RoomId)
	assert.Equal(t, 2008, lodge.Exits["east"].RoomId)
	for _, room := range []*rooms.Room{green, lodge} {
		assert.Equal(t, "Fernhollow", room.Zone)
	}
	assert.True(t, lodge.HasTag(domain.ShamanTag))
	assert.False(t, lodge.HasTag(domain.ChurchTag))
	require.Len(t, lodge.SpawnInfo, 1)
	assert.Equal(t, 66, lodge.SpawnInfo[0].MobId)

	data, err := os.ReadFile(filepath.Join(shippedWorld(), "mobs", "fernhollow", "66-old_wenna.yaml"))
	require.NoError(t, err)
	keeper := mobs.Mob{}
	require.NoError(t, yaml.Unmarshal(data, &keeper))
	assert.Equal(t, mobs.MobId(66), keeper.MobId)
	assert.Equal(t, "Old Wenna", keeper.Character.Name)
	assert.False(t, keeper.Hostile)
	assert.Zero(t, keeper.ItemDropChance)
	assert.Empty(t, keeper.Character.Items)
	assert.Zero(t, keeper.Character.Gold)
	assert.Zero(t, keeper.Character.Equipment.Weapon.ItemId)

	s := shippedSettings(t)
	village, ok := s.registry.ServiceAt(2009)
	require.True(t, ok)
	assert.Equal(t, domain.Settlement{Zone: "Fernhollow", Kind: domain.Village, ServiceRoomID: 2009, ServiceMobID: 66}, village)
	_, ok = s.registry.ChurchFor("Fernhollow")
	assert.False(t, ok, "a village is never a checkpoint")
	for room, keeper := range map[int]int{18: 4, 2007: 65} {
		church, ok := s.registry.ServiceAt(room)
		require.True(t, ok)
		assert.Equal(t, keeper, church.ServiceMobID, room)
	}
	sanctuary := shippedRoom(t, "frostfang/18.yaml")
	require.NotEmpty(t, sanctuary.SpawnInfo)
	assert.Equal(t, 4, sanctuary.SpawnInfo[0].MobId, "the Sanctuary's priest is its keeper")

	// Every keeper stays in its room, so the rite doesn't depend on where
	// it wandered (review finding 2).
	for _, path := range []string{"frostfang/4-clergyman.yaml", "dunmar/65-sister_maren.yaml", "fernhollow/66-old_wenna.yaml"} {
		data, err := os.ReadFile(filepath.Join(shippedWorld(), "mobs", path))
		require.NoError(t, err)
		mob := mobs.Mob{}
		require.NoError(t, yaml.Unmarshal(data, &mob))
		assert.Zero(t, mob.MaxWander, path)
	}
}

// TestShippedDefeatScenarios pins the Phase 53 test content: the shipped
// table parses with one scenario of each kind, the capture room is a camp
// tent with a way out to the road and back, its guard template is a weak
// human, and the bound buff exists with the no-go flag.
func TestShippedDefeatScenarios(t *testing.T) {
	s := shippedSettings(t)
	kinds := map[domain.ScenarioKind]domain.Scenario{}
	for _, sc := range s.scenarios {
		assert.NoError(t, sc.Valid(), sc.ID)
		kinds[sc.Kind] = sc
	}
	for _, kind := range []domain.ScenarioKind{domain.Rescued, domain.Captured, domain.LeftForDead, domain.Robbed} {
		assert.Contains(t, kinds, kind)
	}

	capture := kinds[domain.Captured]
	tent := shippedRoom(t, "brigand_camp/91001.yaml")
	camp := shippedRoom(t, "brigand_camp/91002.yaml")
	fork := shippedRoom(t, "old_kings_road/2002.yaml")
	assert.Equal(t, capture.Room, tent.RoomId)
	assert.Equal(t, 91002, tent.Exits["south"].RoomId, "the tent opens on the camp")
	assert.Equal(t, 91001, camp.Exits["north"].RoomId)
	assert.Equal(t, 2002, camp.Exits["south"].RoomId, "the camp opens on the road")
	assert.Equal(t, 91002, fork.Exits["north"].RoomId, "and the road leads back, to reclaim")

	data, err := os.ReadFile(filepath.Join(shippedWorld(), "mobs", "old_kings_road", "86-road_brigand.yaml"))
	require.NoError(t, err)
	guard := mobs.Mob{}
	require.NoError(t, yaml.Unmarshal(data, &guard))
	assert.Equal(t, mobs.MobId(capture.GuardMob), guard.MobId)
	assert.LessOrEqual(t, capture.GuardLevel, guard.Character.Level, "guards come from the lowest band")

	buff, err := os.ReadFile(filepath.Join(shippedWorld(), "buffs", "9301-bound.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(buff), "no-go")
	assert.Contains(t, string(buff), "buffid: 9301")
}
