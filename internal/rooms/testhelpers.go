package rooms

// Test helpers for cross-package testing.
// Since this is under internal/, it cannot be used outside the module.

// SetTestRoom stores a room directly in the room manager so LoadRoom
// resolves it without reading a file. For testing only.
func SetTestRoom(r *Room) {
	roomManager.rooms[r.RoomId] = r
}

// RemoveTestRoom removes a room stored with SetTestRoom. For testing only.
func RemoveTestRoom(roomId int) {
	delete(roomManager.rooms, roomId)
}

// SetTestOccupants sets which users and mob instances are in the room.
// For testing only.
func (r *Room) SetTestOccupants(userIds []int, mobInstanceIds []int) {
	r.players = append([]int(nil), userIds...)
	r.mobs = append([]int(nil), mobInstanceIds...)
}

// SetTestBiome registers a biome directly, replacing any with the same id.
// For testing only.
func SetTestBiome(b *BiomeInfo) {
	biomes[b.Id()] = b
}

// RemoveTestBiome removes a biome registered with SetTestBiome.
// For testing only.
func RemoveTestBiome(biomeId string) {
	delete(biomes, biomeId)
}
