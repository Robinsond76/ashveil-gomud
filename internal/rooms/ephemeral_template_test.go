package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCreateEphemeralRoomIdsWithoutTemplate: a room with no template on disk
// (only in memory) can't be copied; the call reports it instead of panicking,
// and leaves the chunk free.
func TestCreateEphemeralRoomIdsWithoutTemplate(t *testing.T) {
	room := &Room{RoomId: 9403, Zone: "ephemeraltest", Title: "Memory only"}
	SetTestRoom(room)
	t.Cleanup(func() { RemoveTestRoom(room.RoomId) })
	chunks := GetChunkCount()

	assert.NotPanics(t, func() {
		copies, err := CreateEphemeralRoomIds(room.RoomId)
		assert.Error(t, err)
		assert.Empty(t, copies)
	})
	assert.Equal(t, chunks, GetChunkCount())
}
