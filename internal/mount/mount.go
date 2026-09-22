// Package mount contains the GoMud-free durable mount assignment, matching
// the domain/module split of internal/expedition, internal/camping,
// internal/weather, and internal/encumbrance.
//
// The MVP mount carries no decaying state at all (no fatigue, health, or
// feed) — handoff §35 explicitly defers those to "Later." A mount is just a
// stable type assignment whose configured spec contributes a cargo-capacity
// bonus; travel-duration is computed data, not yet wired into travel.
package mount

import "errors"

// PctMin and PctMax bound a MountSpec's TravelDurationPct, matching
// weather.Condition's and modules/encumbrance's LoadBand convention. 100
// means unchanged.
const (
	PctMin = 25
	PctMax = 300
)

var (
	ErrInvalidSpec  = errors.New("mount: invalid mount spec")
	ErrInvalidMount = errors.New("mount: invalid mount")
)

// MountSpec is one configured mount type.
type MountSpec struct {
	Type                    string
	Description             string
	CargoCapacityBonusGrams int
	TravelDurationPct       int
}

// Validate rejects a malformed spec rather than letting it be guessed at,
// matching weather.Condition.Validate.
func (s MountSpec) Validate() error {
	if s.Type == "" {
		return ErrInvalidSpec
	}
	if s.CargoCapacityBonusGrams < 0 {
		return ErrInvalidSpec
	}
	if s.TravelDurationPct != 0 && (s.TravelDurationPct < PctMin || s.TravelDurationPct > PctMax) {
		return ErrInvalidSpec
	}
	return nil
}

// Mount is the durable leader-owned mount assignment.
type Mount struct {
	LeaderUserID int
	Type         string
}

func (m Mount) Validate() error {
	if m.LeaderUserID <= 0 || m.Type == "" {
		return ErrInvalidMount
	}
	return nil
}

// Established assigns a mount type to a leader.
func Established(leaderUserID int, mountType string) (Mount, error) {
	m := Mount{LeaderUserID: leaderUserID, Type: mountType}
	if err := m.Validate(); err != nil {
		return Mount{}, err
	}
	return m, nil
}
