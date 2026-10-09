package classes

// SetPlannedForTest marks a class as planned (not open yet), or open again,
// until the returned function restores it. Every elite is open, so a test of
// the planned-route paths borrows one.
func SetPlannedForTest(id string, planned bool) (restore func()) {
	id = normalize(id)
	old := byID[id]
	c := old
	c.Planned = planned
	byID[id] = c
	return func() { byID[id] = old }
}
