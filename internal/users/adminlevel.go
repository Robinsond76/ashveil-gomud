package users

import "fmt"

// MaxAdminLevel caps the level the admin test area sets.
const MaxAdminLevel = 100

// AdminSetLevel sets the character's level outright, for the admin test
// area (modules/testarea). A higher level is earned as experience would earn
// it (points, stats and level spells included); a lower one is lost the way
// a death loses it. Health and mana end full. It reports the level reached.
func (u *UserRecord) AdminSetLevel(level int) (int, error) {
	if level < 1 || level > MaxAdminLevel {
		return u.Character.Level, fmt.Errorf("level must be from 1 to %d", MaxAdminLevel)
	}
	c := u.Character
	for c.Level > level {
		c.LoseLevel()
	}
	if c.Level < level {
		c.Experience = max(c.Experience, c.XPTL(level-1)) + 1
		u.GrantXP(0, "test area")
		for c.Level < level {
			c.Experience = max(c.Experience, c.XPTL(c.Level)) + 1
			u.GrantXP(0, "test area")
		}
	}
	c.RecalculateStats()
	c.Health, c.Mana = c.HealthMax.Value, c.ManaMax.Value
	return c.Level, nil
}
