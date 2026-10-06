package rooms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Phase 40a: room resources.

func TestNormalizeResourcesDropsUnknownAndOrders(t *testing.T) {
	kept, dropped := NormalizeResources([]string{"Shelter", "water", "lava", "water", " Forage "})
	assert.Equal(t, []string{"water", "forage", "shelter"}, kept)
	assert.Equal(t, []string{"lava"}, dropped)
}

func TestRoomLoadsResourcesFromYAMLAndDropsUnknown(t *testing.T) {
	mudlog.SetupLogger(nil, "low", "", false)
	var room Room
	require.NoError(t, yaml.Unmarshal([]byte("roomid: 1\nresources:\n- shelter\n- bogus\n- water\n"), &room))
	room.normalizeResources()
	assert.Equal(t, []string{"water", "shelter"}, room.Resources)
	assert.True(t, room.HasResource(ResourceWater))
	assert.False(t, room.HasResource(ResourceForage))
}

func TestShownResourcesHidesReservedOnes(t *testing.T) {
	room := &Room{Resources: []string{"herbs", "forage", "game", "water"}}
	assert.Equal(t, []string{"water", "forage"}, room.ShownResources())
	assert.Equal(t, "Here: fresh water, forage.", room.ResourceLine())
	assert.True(t, room.HasResource(ResourceHerbs), "reserved ones are kept in data")

	only := &Room{Resources: []string{"herbs"}}
	assert.NotNil(t, only.ShownResources())
	assert.Empty(t, only.ShownResources())
	assert.Equal(t, "", only.ResourceLine())
	assert.Equal(t, "", (&Room{}).ResourceLine())
	var none *Room
	assert.False(t, none.HasResource(ResourceWater))
}

// TestShippedWorldResourcesAreValid walks every shipped room: each listed
// resource must be in the vocabulary, so a typo never silently drops.
func TestShippedWorldResourcesAreValid(t *testing.T) {
	root := filepath.Join("..", "..", "_datafiles", "world")
	tagged := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.Contains(path, string(filepath.Separator)+"rooms"+string(filepath.Separator)) || !strings.HasSuffix(path, ".yaml") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var room struct {
			Resources []string `yaml:"resources"`
		}
		if yaml.Unmarshal(data, &room) != nil {
			return nil // zone-config and other shapes
		}
		for _, id := range room.Resources {
			assert.True(t, IsKnownResource(id), "%s: unknown resource %q", path, id)
		}
		if len(room.Resources) > 0 {
			tagged++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Greater(t, tagged, 0, "the default world tags its rooms")
}

// Review fix: forage and shelter only act on a camp rest, so they are
// hidden where no camp can be made; water is shown anywhere.
func TestCampResourcesShowOnlyWhereACampCanBeMade(t *testing.T) {
	SetCampableCheck(func(r *Room) bool { return r.RoomId == 1 })
	t.Cleanup(func() { SetCampableCheck(nil) })

	camp := &Room{RoomId: 1, Resources: []string{"water", "forage", "shelter"}}
	cave := &Room{RoomId: 2, Resources: []string{"water", "forage", "shelter"}}
	assert.Equal(t, []string{"water", "forage", "shelter"}, camp.ShownResources())
	assert.Equal(t, []string{"water"}, cave.ShownResources())
	assert.Equal(t, "Here: fresh water.", cave.ResourceLine())
	assert.True(t, cave.HasResource(ResourceShelter), "the data is kept for when a camp arrives")
}
