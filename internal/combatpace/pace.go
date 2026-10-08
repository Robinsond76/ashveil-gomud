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

// Spec is a pace's timing. It paces one of two ways:
//
//   - by line (Beats false; the fixed-cadence fights of Phase 29f): Gap
//     before an ordinary line, Dramatic before a critical hit's pain or
//     death line, Quick before an indented follow-up, and the round's gaps
//     squeezed to fit Window;
//   - by action (Beats true; Phase 82c, a battle on its own clock): every
//     fighter's turn is a beat. Beat waits before the first line of a new
//     turn, Follow before each further line of the same turn, Extra is
//     added before a pain or death line, and nothing is squeezed: the round
//     lasts as long as its actions need. Tail is the pause the clock adds
//     after the last line before the next round.
type Spec struct {
	Gap      time.Duration // by line: before an ordinary line
	Dramatic time.Duration // by line: before a critical hit's pain or death line
	Quick    time.Duration // by line: before an indented follow-up line
	Window   time.Duration // by line: a round's lines never take longer than this
	Beats    bool          // pace by action
	Beat     time.Duration // by action: before a turn's first line
	Follow   time.Duration // by action: before a turn's later lines
	Extra    time.Duration // by action: added before a pain or death line
	Tail     time.Duration // by action: after the round's last line
}

var specs = map[Pace]Spec{
	Fast:   {Gap: 400 * time.Millisecond, Dramatic: 800 * time.Millisecond, Quick: 150 * time.Millisecond, Window: 3 * time.Second, Beat: 600 * time.Millisecond, Follow: 150 * time.Millisecond, Extra: 300 * time.Millisecond, Tail: 400 * time.Millisecond},
	Normal: {Gap: 800 * time.Millisecond, Dramatic: 1400 * time.Millisecond, Quick: 250 * time.Millisecond, Window: 6 * time.Second, Beat: 1000 * time.Millisecond, Follow: 250 * time.Millisecond, Extra: 500 * time.Millisecond, Tail: 600 * time.Millisecond},
	Slow:   {Gap: 1000 * time.Millisecond, Dramatic: 1800 * time.Millisecond, Quick: 300 * time.Millisecond, Window: 7500 * time.Millisecond, Beat: 1500 * time.Millisecond, Follow: 300 * time.Millisecond, Extra: 700 * time.Millisecond, Tail: 800 * time.Millisecond},
}

// Spec is the pace's timing. Off, or an unknown pace, is all zero.
func (p Pace) Spec() Spec { return specs[p] }

// ForRound is the pace's timing by line with its window clamped to 90% of a
// combat round, so a round's lines never run into the next round.
func (p Pace) ForRound(combatRound time.Duration) Spec {
	s := p.Spec()
	if limit := combatRound * 9 / 10; limit > 0 && s.Window > limit {
		s.Window = limit
	}
	return s
}

// Beats is the pace's timing by action, for a battle on its own clock: no
// window, since the next round waits for this one's lines.
func (p Pace) Beats() Spec {
	s := p.Spec()
	s.Beats, s.Window = true, 0
	return s
}

// Slowest is the slower of two paces for a shared clock; Off counts as
// Normal, since its players read the round at once but the fight must
// still be followable.
func Slowest(a, b Pace) Pace {
	rank := func(p Pace) int {
		switch p {
		case Fast:
			return 1
		case Slow:
			return 3
		}
		return 2 // Normal, Off, unknown
	}
	if rank(b) > rank(a) {
		return b
	}
	if a == Off {
		return Normal
	}
	return a
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
