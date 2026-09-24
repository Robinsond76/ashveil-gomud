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
	for _, settlement := range s.registry.Settlements() {
		assert.Equal(t, domain.City, settlement.Kind, "25a ships no village")
	}
}
