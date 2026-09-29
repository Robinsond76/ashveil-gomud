// Package combatpace paces combat text (Ashveil Phase 29f).
//
// A combat round resolves in an instant, but its lines are held per player
// and released one by one, so the fight reads as if it happens in real
// time. This package is pure: it holds lines and says which are due. The
// caller (internal/hooks) owns delivery, the clock, and the game loop.
package combatpace

import (
	"strings"
	"time"
)

// Pace is a player's chosen pacing: the saved value of the "combatpace"
// config option.
type Pace string

const (
	Fast   Pace = "fast"
	Normal Pace = "normal"
	Slow   Pace = "slow"
	Off    Pace = "off"
)

// OptionKey is the player config option holding a Pace.
const OptionKey = "combatpace"

// Paces lists every pace, in the order help and `set` show them.
var Paces = []Pace{Fast, Normal, Slow, Off}

// Spec is a pace's timing.
type Spec struct {
	Gap      time.Duration // before an ordinary line
	Dramatic time.Duration // before a critical hit's pain or death line
	Quick    time.Duration // before an indented follow-up line
	Window   time.Duration // a round's lines never take longer than this
}

var specs = map[Pace]Spec{
	Fast:   {Gap: 400 * time.Millisecond, Dramatic: 800 * time.Millisecond, Quick: 150 * time.Millisecond, Window: 3 * time.Second},
	Normal: {Gap: 800 * time.Millisecond, Dramatic: 1400 * time.Millisecond, Quick: 250 * time.Millisecond, Window: 6 * time.Second},
	Slow:   {Gap: 1000 * time.Millisecond, Dramatic: 1800 * time.Millisecond, Quick: 300 * time.Millisecond, Window: 7500 * time.Millisecond},
}

// Spec is the pace's timing. Off, or an unknown pace, is all zero.
func (p Pace) Spec() Spec { return specs[p] }

// ForRound is the pace's timing with its window clamped to 90% of a combat
// round, so a round's lines never run into the next round.
func (p Pace) ForRound(combatRound time.Duration) Spec {
	s := p.Spec()
	if limit := combatRound * 9 / 10; limit > 0 && s.Window > limit {
		s.Window = limit
	}
	return s
}

// Parse reads a pace a player typed.
func Parse(s string) (Pace, bool) {
	p := Pace(strings.ToLower(strings.TrimSpace(s)))
	for _, known := range Paces {
		if p == known {
			return p, true
		}
	}
	return "", false
}

// DefaultPace is the pace of a player who hasn't chosen one: off for a screen
// reader, normal for everyone else.
func DefaultPace(screenReader bool) Pace {
	if screenReader {
		return Off
	}
	return Normal
}

// For is a player's pace from their saved option (nil when unset).
func For(option any, screenReader bool) Pace {
	if s, ok := option.(string); ok {
		if p, ok := Parse(s); ok {
			return p
		}
	}
	return DefaultPace(screenReader)
}
