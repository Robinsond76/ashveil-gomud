package configs

// SetTestGamePlayConfig replaces in-memory gameplay settings for integration tests.
// The returned function restores the previous settings.
func SetTestGamePlayConfig(gameplay GamePlay) func() {
	configDataLock.Lock()
	previous := configData.GamePlay
	configData.GamePlay = gameplay
	configDataLock.Unlock()
	return func() {
		configDataLock.Lock()
		configData.GamePlay = previous
		configDataLock.Unlock()
	}
}

// SetTestSpecialRoomsConfig replaces the special rooms for integration
// tests. The returned function restores the previous settings.
func SetTestSpecialRoomsConfig(special SpecialRooms) func() {
	configDataLock.Lock()
	previous := configData.SpecialRooms
	configData.SpecialRooms = special
	configDataLock.Unlock()
	return func() {
		configDataLock.Lock()
		configData.SpecialRooms = previous
		configDataLock.Unlock()
	}
}
