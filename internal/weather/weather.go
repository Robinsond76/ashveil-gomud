// Package weather contains the GoMud-free, round-driven, zone-scoped weather
// state machine. Unlike internal/expedition and internal/camping, this
// package is intentionally not real-UTC-based: weather tracks the shared
// world round clock, never a per-player or wall-clock timer.
package weather

import "errors"

// PctMin and PctMax bound a Condition's multiplier fields. 100 means
// unchanged; the range is intentionally generous but still sane.
const (
	PctMin = 25
	PctMax = 300
)

var (
	ErrInvalidCondition   = errors.New("weather: invalid condition")
	ErrInvalidZoneWeather = errors.New("weather: invalid zone weather")
	ErrNotDue             = errors.New("weather: not due")
)

// Condition is one biome-tabled weather state.
type Condition struct {
	Name              string
	Description       string
	TravelDurationPct int
	ExertionPct       int
	RestRecoveryPct   int
}

// Validate rejects a malformed condition rather than letting it be guessed
// at, matching expedition.TravelProfile.Validate.
func (c Condition) Validate() error {
	if c.Name == "" {
		return ErrInvalidCondition
	}
	if !pctValid(c.TravelDurationPct) || !pctValid(c.ExertionPct) || !pctValid(c.RestRecoveryPct) {
		return ErrInvalidCondition
	}
	return nil
}

func pctValid(pct int) bool {
	return pct >= PctMin && pct <= PctMax
}

// ZoneWeather is the durable current-condition record for one zone.
type ZoneWeather struct {
	Zone            string
	Current         string
	NextChangeRound uint64
}

// Validate reports whether the record is structurally sound. It does not
// check Current against any particular biome table; callers validate that
// at apply time, where the table is known.
func (w ZoneWeather) Validate() error {
	if w.Zone == "" || w.Current == "" {
		return ErrInvalidZoneWeather
	}
	return nil
}

// Due reports whether this zone's weather should change at currentRound.
func (w ZoneWeather) Due(currentRound uint64) bool {
	if err := w.Validate(); err != nil {
		return false
	}
	return currentRound >= w.NextChangeRound
}

// Advance returns a copy of an already-tracked zone's weather transitioned
// to next, scheduled for nextChangeRound. It is pure and refuses a
// transition that is not yet due or a malformed incoming condition/schedule.
func (w ZoneWeather) Advance(currentRound uint64, next Condition, nextChangeRound uint64) (ZoneWeather, error) {
	if err := w.Validate(); err != nil {
		return ZoneWeather{}, err
	}
	if err := next.Validate(); err != nil {
		return ZoneWeather{}, err
	}
	if nextChangeRound <= currentRound {
		return ZoneWeather{}, ErrInvalidZoneWeather
	}
	if !w.Due(currentRound) {
		return ZoneWeather{}, ErrNotDue
	}
	return ZoneWeather{Zone: w.Zone, Current: next.Name, NextChangeRound: nextChangeRound}, nil
}

// Established returns the initial zone weather record for a zone that has
// not been tracked before, unconditionally (there is nothing to be "due"
// against yet).
func Established(zone string, initial Condition, nextChangeRound uint64, currentRound uint64) (ZoneWeather, error) {
	if zone == "" {
		return ZoneWeather{}, ErrInvalidZoneWeather
	}
	if err := initial.Validate(); err != nil {
		return ZoneWeather{}, err
	}
	if nextChangeRound <= currentRound {
		return ZoneWeather{}, ErrInvalidZoneWeather
	}
	return ZoneWeather{Zone: zone, Current: initial.Name, NextChangeRound: nextChangeRound}, nil
}
