// Package sky holds the GoMud-free, pure sky model: moon phases derived
// from the shared round counter, moonlight strength, and cloud-cover names.
// Nothing here is stored or scheduled; every value is recomputed from the
// current round, so restart and copyover reproduce the same sky and the
// world clock is never advanced.
package sky

import "sync/atomic"

// MoonPhase is one of eight lunar phases, New (0) through WaningCrescent (7).
type MoonPhase int

const (
	New MoonPhase = iota
	WaxingCrescent
	FirstQuarter
	WaxingGibbous
	Full
	WaningGibbous
	LastQuarter
	WaningCrescent

	phaseCount = 8
)

// DefaultCycleDays is the default number of game days in one lunar cycle.
const DefaultCycleDays = 8

// MaxCloudCover is the highest cloud-cover value: a fully overcast sky.
const MaxCloudCover = 3

var phaseNames = [phaseCount]string{
	"new moon",
	"waxing crescent",
	"first quarter",
	"waxing gibbous",
	"full moon",
	"waning gibbous",
	"last quarter",
	"waning crescent",
}

// phaseBrightness is the moonlight each phase gives under a clear sky, on
// the same 0-2 scale as rooms.Room.GetVisibility.
var phaseBrightness = [phaseCount]int{0, 1, 1, 2, 2, 2, 1, 1}

var cloudCoverNames = [MaxCloudCover + 1]string{
	"clear",
	"scattered clouds",
	"broken clouds",
	"overcast",
}

var cycleDays atomic.Int64

func init() {
	cycleDays.Store(DefaultCycleDays)
}

// SetCycleDays sets the configured lunar cycle length. A non-positive value
// restores DefaultCycleDays.
func SetCycleDays(days int) {
	if days < 1 {
		days = DefaultCycleDays
	}
	cycleDays.Store(int64(days))
}

// CycleDays reports the configured lunar cycle length.
func CycleDays() int {
	return int(cycleDays.Load())
}

// Name is the phase's display name, or "" for an invalid phase.
func (p MoonPhase) Name() string {
	if p < 0 || p >= phaseCount {
		return ""
	}
	return phaseNames[p]
}

// String implements fmt.Stringer.
func (p MoonPhase) String() string { return p.Name() }

// AbsoluteDay is the number of whole game days elapsed at round.
func AbsoluteDay(round uint64, roundsPerDay int) uint64 {
	if roundsPerDay < 1 {
		return 0
	}
	return round / uint64(roundsPerDay)
}

// PhaseForDay returns the moon phase on an absolute game day for a cycle of
// cycleDays days. A non-positive cycle uses DefaultCycleDays.
func PhaseForDay(day uint64, cycle int) MoonPhase {
	if cycle < 1 {
		cycle = DefaultCycleDays
	}
	dayOfCycle := day % uint64(cycle)
	return MoonPhase(dayOfCycle * phaseCount / uint64(cycle))
}

// MoonVisible reports whether the moon can be seen through cloudCover.
func MoonVisible(cloudCover int) bool {
	return cloudCover < MaxCloudCover
}

// Moonlight is the light (0-2) the moon gives at phase under cloudCover.
// Broken cloud dims it by one step; overcast hides it entirely.
func Moonlight(phase MoonPhase, cloudCover int) int {
	if phase < 0 || phase >= phaseCount || !MoonVisible(cloudCover) {
		return 0
	}
	light := phaseBrightness[phase]
	if cloudCover >= 2 {
		light--
	}
	if light < 0 {
		light = 0
	}
	return light
}

// CloudCoverName is the display name for a cloud-cover value, or "" when out
// of range.
func CloudCoverName(cover int) string {
	if cover < 0 || cover > MaxCloudCover {
		return ""
	}
	return cloudCoverNames[cover]
}
