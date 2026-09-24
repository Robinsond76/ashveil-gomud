package plugins

// SnapshotLoadStateForTest records the package state that Load changes for
// the rest of a test binary (registration closing, the plugin-data folder)
// and returns a function that restores it. A test that calls Load should
// register the returned function with t.Cleanup so later tests (and
// -count/-shuffle reruns) can still call New and write plugin data.
func SnapshotLoadStateForTest() func() {
	open, folder := registrationOpen, writeFolderPath
	return func() {
		registrationOpen, writeFolderPath = open, folder
	}
}
