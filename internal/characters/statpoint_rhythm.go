package characters

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

const (
	// LegacyStatPointsEveryNLevels is the rhythm shipped before 35a (30g4).
	LegacyStatPointsEveryNLevels = 5
	// StatPointRhythmMigrated marks a character whose points follow the
	// current rhythm.
	StatPointRhythmMigrated = 2
)

// CatchUpStatPoints tops a pre-35a character up to the stat points the
// current rhythm pays at its peak level, once (Phase 35a). Points it already
// holds, spent or unspent, count toward that total, so characters who earned
// on the every-level rhythm before 30g4, or were given points, are not paid
// twice; points are never taken away. While the server still runs the legacy
// rhythm the character stays unmarked and is caught up when the rhythm
// changes, including a live change before its next level-up. It reports
// whether the character changed.
func (c *Character) CatchUpStatPoints() bool {
	if c.StatPointRhythm >= StatPointRhythmMigrated {
		return false
	}
	cfg := configs.GetProgressionConfig()
	if int(cfg.StatPointsEveryNLevels) == LegacyStatPointsEveryNLevels {
		return false
	}
	s := c.Stats
	held := stats.SaturatingSum(max(c.StatPoints, 0), max(s.Strength.Training, 0), max(s.Speed.Training, 0),
		max(s.Smarts.Training, 0), max(s.Vitality.Training, 0), max(s.Mysticism.Training, 0), max(s.Perception.Training, 0))
	peak := max(c.PeakLevel, c.Level)
	c.StatPoints = stats.SaturatingSum(c.StatPoints, max(0, cfg.StatPointsAt(peak)-held))
	c.StatPointRhythm = StatPointRhythmMigrated
	return true
}
