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

// SetTestDataFiles points FilePaths.DataFiles at dir. The returned function
// restores the previous overrides. AddOverlayOverrides can't do this: it never
// overwrites a key that is already set, so a later test using it is silently
// ignored and the first test's path leaks into every one after it.
func SetTestDataFiles(dir string) func() {
	set := func(value string) {
		flat := Flatten(GetOverrides())
		flat["FilePaths.DataFiles"] = value
		_ = RestoreOverrides(flat)
	}
	previous := GetFilePathsConfig().DataFiles.String()
	set(dir)
	return func() { set(previous) }
}
