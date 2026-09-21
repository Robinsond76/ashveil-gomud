package exit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRoomExitTravelProfileYAMLCompatibility(t *testing.T) {
	var unmarked RoomExit
	require.NoError(t, yaml.Unmarshal([]byte("roomid: 5\n"), &unmarked))
	assert.Equal(t, 5, unmarked.RoomId)
	assert.Equal(t, "", unmarked.TravelProfile)

	out, err := yaml.Marshal(unmarked)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "travel_profile", "unmarked exits must stay byte-compatible")

	var marked RoomExit
	require.NoError(t, yaml.Unmarshal([]byte("roomid: 5\ntravel_profile: oak-road\n"), &marked))
	assert.Equal(t, 5, marked.RoomId)
	assert.Equal(t, "oak-road", marked.TravelProfile)

	out, err = yaml.Marshal(marked)
	require.NoError(t, err)
	assert.Contains(t, string(out), "travel_profile: oak-road")
}
