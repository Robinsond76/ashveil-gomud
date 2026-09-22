// Package camping contains the GoMud-free durable campsite and real-time rest
// state machine.
package camping

import (
	"errors"
	"time"
)

const (
	RestDuration    = 60 * time.Second
	FatigueRecovery = 20
)

var (
	ErrInvalidCamp          = errors.New("camping: invalid camp")
	ErrInvalidTransition    = errors.New("camping: invalid transition")
	ErrFireAlreadyLit       = errors.New("camping: fire is already lit")
	ErrFireNotLit           = errors.New("camping: fire is not lit")
	ErrRestAlreadyStarted   = errors.New("camping: rest is already started")
	ErrRestAlreadyCompleted = errors.New("camping: rest is already completed")
	ErrRestInProgress       = errors.New("camping: rest is in progress")
	ErrNoRest               = errors.New("camping: no rest session")
	ErrRestNotDue           = errors.New("camping: rest is not due")
)

// RestState is the durable lifecycle state of a rest session.
type RestState uint8

const (
	Resting RestState = iota
	Completed
)

func (s RestState) Valid() bool { return s == Resting || s == Completed }

// RestSession records only stable rest identity and completion state. Progress
// is derived from StartedAtUTC and real UTC time, so it survives restarts.
type RestSession struct {
	StartedAtUTC time.Time `yaml:"started_at_utc"`
	State        RestState `yaml:"state"`
}

func (s RestSession) Validate() error {
	if s.StartedAtUTC.IsZero() || !s.State.Valid() {
		return ErrInvalidCamp
	}
	return nil
}

// Camp is the durable leader-owned campsite record.
type Camp struct {
	LeaderUserID int          `yaml:"leader_user_id"`
	RoomID       int          `yaml:"room_id"`
	FireLit      bool         `yaml:"fire_lit"`
	Rest         *RestSession `yaml:"rest,omitempty"`
}

func (c Camp) Validate() error {
	if c.LeaderUserID <= 0 || c.RoomID <= 0 {
		return ErrInvalidCamp
	}
	if c.Rest != nil {
		if err := c.Rest.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Established creates a camp after validating both stable IDs.
func Established(leaderUserID, roomID int) (Camp, error) {
	c := Camp{LeaderUserID: leaderUserID, RoomID: roomID}
	if err := c.Validate(); err != nil {
		return Camp{}, err
	}
	return c, nil
}

// LightFire returns a copy with its fire lit.
func (c Camp) LightFire() (Camp, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if c.FireLit {
		return c, ErrFireAlreadyLit
	}
	c.FireLit = true
	return c, nil
}

// StartRest returns a copy with a new resting session. A camp must have a lit
// fire, and only one session may ever be created for a camp.
func (c Camp) StartRest(startedAtUTC time.Time) (Camp, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if !c.FireLit {
		return c, ErrFireNotLit
	}
	if c.Rest != nil {
		if c.Rest.State == Completed {
			return c, ErrRestAlreadyCompleted
		}
		return c, ErrRestAlreadyStarted
	}
	if startedAtUTC.IsZero() {
		return c, ErrInvalidCamp
	}
	c.Rest = &RestSession{StartedAtUTC: startedAtUTC.UTC(), State: Resting}
	return c, nil
}

// ProgressAt returns rest progress clamped to [0,1].
func (c Camp) ProgressAt(now time.Time) float64 {
	if c.Rest == nil {
		return 0
	}
	if c.Rest.State == Completed {
		return 1
	}
	elapsed := now.UTC().Sub(c.Rest.StartedAtUTC)
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= RestDuration {
		return 1
	}
	return float64(elapsed) / float64(RestDuration)
}

func (c Camp) RestDue(now time.Time) bool {
	return c.Rest != nil && c.Rest.State == Resting && c.ProgressAt(now) >= 1
}

// CompleteRest marks a due rest completed. The optional time makes the
// transition deterministic for callers while retaining a convenient current
// time form for command code.
func (c Camp) CompleteRest(now ...time.Time) (Camp, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if c.Rest == nil {
		return c, ErrNoRest
	}
	if c.Rest.State == Completed {
		return c, ErrRestAlreadyCompleted
	}
	when := time.Now().UTC()
	if len(now) > 0 {
		when = now[0]
	}
	if !c.RestDue(when) {
		return c, ErrRestNotDue
	}
	rest := *c.Rest
	rest.State = Completed
	c.Rest = &rest
	return c, nil
}

// Break removes an idle camp. Resting camps cannot be broken; a completed
// session is idle and can be removed.
func (c Camp) Break() (Camp, error) {
	if err := c.Validate(); err != nil {
		return c, err
	}
	if c.Rest != nil && c.Rest.State == Resting {
		return c, ErrRestInProgress
	}
	return Camp{}, nil
}
