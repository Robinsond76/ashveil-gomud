package characters

// ColdDelay is sampled at action start only. Existing exposure stat penalties
// are unchanged; losing/gaining cold during a chant doesn't reset its timer.
func (c *Character) ColdDelay() int {
	if c == nil {
		return 0
	}
	// 1011-1013 are the cold exposure buffs; 1113 is a Shaman's Chill Wind
	// (Phase 39c, status.Windchilled), which slows chants and sling shots alike.
	if len(c.GetBuffs(1011, 1012, 1013, 1113)) > 0 {
		return 1
	}
	return 0
}
