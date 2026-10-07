package camping

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidStay = errors.New("camping: invalid inn stay")

// InnTier is the kind of room a stay was bought in (Phase 75). The empty
// value is the common room, so stays saved before the tiers existed read as
// they always did.
type InnTier string

const (
	InnCommon  InnTier = "common"
	InnPrivate InnTier = "private"
	InnSuite   InnTier = "suite"
)

// InnTiers lists the tiers cheapest first.
var InnTiers = []InnTier{InnCommon, InnPrivate, InnSuite}

// Normalize maps the empty tier to the common room.
func (t InnTier) Normalize() InnTier {
	if t == "" {
		return InnCommon
	}
	return t
}

// Valid reports a tier a stay can hold (empty counts as common).
func (t InnTier) Valid() bool {
	switch t.Normalize() {
	case InnCommon, InnPrivate, InnSuite:
		return true
	}
	return false
}

// Label is the room's name for players ("a private room").
func (t InnTier) Label() string {
	switch t.Normalize() {
	case InnPrivate:
		return "a private room"
	case InnSuite:
		return "the suite"
	}
	return "a common room"
}

// ParseInnTier reads a player's word for a room tier. No word is the
// common room.
func ParseInnTier(word string) (InnTier, bool) {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "", "common", "room", "bed", "shared":
		return InnCommon, true
	case "private", "single":
		return InnPrivate, true
	case "suite":
		return InnSuite, true
	}
	return "", false
}

// InnStay is the durable record of a company paying for a real-time rest at
// an inn (Phase 16). Progress is derived from Rest.StartedAtUTC against real
// UTC time over the Duration locked when the stay began, so a stay survives
// restart and copyover exactly as a camp rest does.
type InnStay struct {
	LeaderUserID int           `yaml:"leader_user_id"`
	RoomID       int           `yaml:"room_id"`
	Paid         int           `yaml:"paid"`
	Duration     time.Duration `yaml:"duration"`
	// Tier is the room bought (Phase 75); empty reads as the common room.
	Tier InnTier     `yaml:"tier,omitempty"`
	Rest RestSession `yaml:"rest"`
}

// Validate rejects a stay that could not have come from StartInnStay.
func (s InnStay) Validate() error {
	if s.LeaderUserID <= 0 || s.RoomID <= 0 || s.Paid < 0 || s.Duration <= 0 || !s.Tier.Valid() {
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
