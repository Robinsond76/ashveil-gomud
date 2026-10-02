package characters

// ColdDelay is sampled at action start only. Existing exposure stat penalties
// are unchanged; losing/gaining cold during a chant doesn't reset its timer.
func (c *Character) ColdDelay() int {
	if c == nil {
		return 0
	}
	if len(c.GetBuffs(1011, 1012, 1013)) > 0 {
		return 1
	}
	return 0
}
