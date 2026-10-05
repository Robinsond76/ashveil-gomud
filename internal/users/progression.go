package users

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// migrateStatPointRhythm preserves spent points and pays the peak-level
// difference once. The marker and award reach disk in the same atomic save.
func migrateStatPointRhythm(u *UserRecord) error {
	c := u.Character
	if c.StatPointRhythm >= 2 {
		return nil
	}
	cfg := configs.GetProgressionConfig()
	old := cfg
	old.StatPointsEveryNLevels = 5
	peak := max(c.PeakLevel, c.Level)
	before := c.StatPoints
	c.StatPoints = stats.SaturatingSum(c.StatPoints, max(0, cfg.StatPointsAt(peak)-old.StatPointsAt(peak)))
	marker := c.StatPointRhythm
	c.StatPointRhythm = 2
	if err := SaveUserAtomic(*u); err != nil {
		c.StatPoints = before
		c.StatPointRhythm = marker
		return err
	}
	return nil
}
