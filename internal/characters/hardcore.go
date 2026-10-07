package characters

// Ashveil Phase 77: the Iron (Hardcore) option and the account blessings a
// character was given. Both are fixed when the character is created.

// IronLevelsLost is how many levels a defeat costs an Iron character. A
// normal defeat costs one. PeakLevel is kept, so re-levelling grants
// nothing twice.
const IronLevelsLost = 2

// IsIron reports whether the character took the Iron option at creation.
func (c *Character) IsIron() bool { return c != nil && c.Hardcore }

// SetIron records the Iron choice. It is only for character creation: an
// Iron character stays Iron, and a standard one never becomes Iron later.
func (c *Character) SetIron(on bool) { c.Hardcore = on }

// HasBlessing reports whether the character was given the blessing.
func (c *Character) HasBlessing(id string) bool {
	for _, b := range c.Blessings {
		if b == id {
			return true
		}
	}
	return false
}

// LoseLevels takes n levels (one at a time, as LoseLevel) and reports the
// level before and after.
func (c *Character) LoseLevels(n int) (from, to int) {
	from = max(c.Level, 1)
	to = from
	for i := 0; i < n; i++ {
		_, to = c.LoseLevel()
	}
	return from, to
}

// ironOfferedKey marks a character whose creation has put (and had
// answered) the Iron question.
const ironOfferedKey = `iron-offered`

// IronOffered reports whether the Iron question was answered at creation.
func (c *Character) IronOffered() bool {
	v, _ := c.GetMiscData(ironOfferedKey).(bool)
	return v
}

// MarkIronOffered records that the Iron question was answered.
func (c *Character) MarkIronOffered() { c.SetMiscData(ironOfferedKey, true) }
