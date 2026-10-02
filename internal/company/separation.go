package company

// Phase 33h3: a companion separated from its leader by a move it could not
// share is off the map until it finds its way back.

// Separation reasons, stored on the record.
const (
	// SeparatedByMove: the leader was moved while it stood elsewhere.
	SeparatedByMove = "move"
	// SeparatedStray: it was away from its leader for too long.
	SeparatedStray = "stray"
)

// DefaultSeparationRounds is the catch-up time when config sets none:
// fifteen rounds, about a minute of the leader's online time.
const DefaultSeparationRounds = 15

// Separation is a separated companion's way back.
type Separation struct {
	Reason string `yaml:"reason,omitempty"`
	// RoundsLeft counts down once a round while the leader is online; at
	// zero it rejoins as soon as the leader is free.
	RoundsLeft int `yaml:"rounds_left"`
}

// Separated reports whether the companion is separated from its leader.
func (c Companion) Separated() bool { return c.Separation != nil }
