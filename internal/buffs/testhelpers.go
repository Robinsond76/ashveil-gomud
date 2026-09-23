package buffs

// Test helpers for cross-package testing. They let other internal packages
// register buff specs and flags without loading datafiles from disk.
// Since this is under internal/, it cannot be used outside the module.

// SetTestBuffSpec registers a buff spec directly in memory. For testing only.
func SetTestBuffSpec(spec *BuffSpec) {
	buffs[spec.BuffId] = spec
}

// RemoveTestBuffSpec removes a buff spec registered with SetTestBuffSpec.
// For testing only.
func RemoveTestBuffSpec(buffId int) {
	delete(buffs, buffId)
}

// SetTestFlag registers a buff flag as valid. For testing only.
func SetTestFlag(flag string) {
	flagSpecs[flag] = &FlagSpec{Flag: flag, Name: flag}
}
