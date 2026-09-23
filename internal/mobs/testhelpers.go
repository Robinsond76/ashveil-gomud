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
