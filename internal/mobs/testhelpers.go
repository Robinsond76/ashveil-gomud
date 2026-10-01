package mobs

// Test helpers for cross-package testing.
// Since this is under internal/, it cannot be used outside the module.

// SetTestInstance registers a mob instance directly, so GetInstance resolves
// it without spawning from a template. For testing only.
func SetTestInstance(m *Mob) {
	mobInstances[m.InstanceId] = m
}

// RemoveTestInstance removes a mob registered with SetTestInstance.
// For testing only.
func RemoveTestInstance(instanceId int) {
	delete(mobInstances, instanceId)
}

// ResetHostility forgets every group's grudge against every player, so a
// test's fight doesn't leave the next test's mobs hostile (MakeHostile's
// grudges outlast a test by many rounds). For testing only.
func ResetHostility() {
	mobsHatePlayers = map[string]map[int]int{}
}

// SetTestSpec registers a mob template directly, so GetMobSpec resolves it
// without loading data files. For testing only.
func SetTestSpec(m *Mob) {
	mobs[int(m.MobId)] = m
}

// RemoveTestSpec removes a template registered with SetTestSpec. For
// testing only.
func RemoveTestSpec(id MobId) {
	delete(mobs, int(id))
}
