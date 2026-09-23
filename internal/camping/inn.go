package camping

import (
	"errors"
	"time"
)

var ErrInvalidStay = errors.New("camping: invalid inn stay")

// InnStay is the durable record of a company paying for a real-time rest at
// an inn (Phase 16). Progress is derived from Rest.StartedAtUTC against real
// UTC time over the Duration locked when the stay began, so a stay survives
// restart and copyover exactly as a camp rest does.
type InnStay struct {
	LeaderUserID int           `yaml:"leader_user_id"`
	RoomID       int           `yaml:"room_id"`
	Paid         int           `yaml:"paid"`
	Duration     time.Duration `yaml:"duration"`
	Rest         RestSession   `yaml:"rest"`
}

// Validate rejects a stay that could not have come from StartInnStay.
func (s InnStay) Validate() error {
	if s.LeaderUserID <= 0 || s.RoomID <= 0 || s.Paid < 0 || s.Duration <= 0 {
		return ErrInvalidStay
	}
	if err := s.Rest.Validate(); err != nil {
		return ErrInvalidStay
	}
	if s.Rest.Recovery <= 0 {
		return ErrInvalidStay
	}
	return nil
}

// StartInnStay creates a resting stay. recovery is the fatigue it will
// restore; it must be positive.
func StartInnStay(leaderUserID, roomID, paid int, startedAtUTC time.Time, duration time.Duration, recovery int) (InnStay, error) {
	if startedAtUTC.IsZero() {
		return InnStay{}, ErrInvalidStay
	}
	s := InnStay{
		LeaderUserID: leaderUserID,
		RoomID:       roomID,
		Paid:         paid,
		Duration:     duration,
		Rest:         RestSession{StartedAtUTC: startedAtUTC.UTC(), State: Resting, Recovery: recovery},
	}
	if err := s.Validate(); err != nil {
		return InnStay{}, err
	}
	return s, nil
}

// Resting reports a stay still in progress.
func (s InnStay) Resting() bool { return s.Rest.State == Resting }

// ProgressAt is the stay's progress, clamped to [0,1].
func (s InnStay) ProgressAt(now time.Time) float64 {
	return s.Rest.ProgressFor(now, s.Duration)
}

// RemainingAt is the time left in the stay.
func (s InnStay) RemainingAt(now time.Time) time.Duration {
	return s.Rest.RemainingFor(now, s.Duration)
}

// Due reports a resting stay whose time is up.
func (s InnStay) Due(now time.Time) bool {
	return s.Resting() && s.ProgressAt(now) >= 1
}

// Complete marks a due stay completed.
func (s InnStay) Complete(now time.Time) (InnStay, error) {
	if err := s.Validate(); err != nil {
		return s, err
	}
	if !s.Resting() {
		return s, ErrRestAlreadyCompleted
	}
	if !s.Due(now) {
		return s, ErrRestNotDue
	}
	s.Rest.State = Completed
	return s, nil
}
