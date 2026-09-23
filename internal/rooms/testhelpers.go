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

// SetTestZoneRoom stores a room like SetTestRoom and also indexes it under
// its zone, so GetAllZoneNames and GetAllZoneRoomsIds report it. For
// testing only.
func SetTestZoneRoom(r *Room) {
	SetTestRoom(r)
	zc, ok := roomManager.zones[r.Zone]
	if !ok {
		zc = &ZoneConfig{Name: r.Zone}
		roomManager.zones[r.Zone] = zc
	}
	if zc.RoomIds == nil {
		zc.RoomIds = map[int]struct{}{}
	}
	zc.RoomIds[r.RoomId] = struct{}{}
}

// RemoveTestZoneRoom undoes SetTestZoneRoom, dropping the zone once it has
// no rooms left. For testing only.
func RemoveTestZoneRoom(r *Room) {
	RemoveTestRoom(r.RoomId)
	if zc, ok := roomManager.zones[r.Zone]; ok {
		delete(zc.RoomIds, r.RoomId)
		if len(zc.RoomIds) == 0 {
			delete(roomManager.zones, r.Zone)
		}
	}
}
