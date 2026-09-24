package company

import "errors"

// Phase 25b: companion death and resurrection. A dead companion stays on
// the roster, out of the formation, until it is resurrected or its rescue
// allowance runs out and it is lost.

// MaxLost is how many lost companions a record remembers, most recent last.
const MaxLost = 10

var (
	// ErrMemberDead is returned for a dead companion where a living one is
	// needed (placing it in the formation, or killing it twice).
	ErrMemberDead = errors.New("companion is dead")
	// ErrNotDead is returned when resurrecting or charging a living
	// companion.
	ErrNotDead = errors.New("companion is not dead")
	// ErrCompanionLost is returned when a dead companion's allowance has run
	// out; it is lost for good.
	ErrCompanionLost = errors.New("companion is lost")
	// ErrAmbiguousMember is returned when a name matches more than one
	// companion; the caller should ask for the number.
	ErrAmbiguousMember = errors.New("more than one companion matches")
	// ErrNoResurrection is returned when no provider can resurrect.
	ErrNoResurrection = errors.New("resurrection is unavailable")
)

// CompanionDeath is a dead companion's durable death (Phase 25b). The
// companion's level and the gear its body kept stay in its State.
type CompanionDeath struct {
	// OpID identifies this death; resurrection and expiry each end it once.
	OpID string `yaml:"op_id"`
	// Allowance is the rescue time granted at death, in seconds of the
	// leader's online time; Remaining is what is left of it.
	Allowance int `yaml:"allowance"`
	Remaining int `yaml:"remaining"`
	// Warned is set once the leader has been told time is short.
	Warned bool `yaml:"warned,omitempty"`
	// Placed, Row, and Col are the formation cell the companion held.
	Placed bool `yaml:"placed,omitempty"`
	Row    int  `yaml:"row,omitempty"`
	Col    int  `yaml:"col,omitempty"`
}

// LostCompanion is a companion whose allowance ran out, kept for display.
type LostCompanion struct {
	ID            int    `yaml:"id"`
	MobTemplateID int    `yaml:"mob_template_id"`
	Name          string `yaml:"name,omitempty"`
	Level         int    `yaml:"level,omitempty"`
	OpID          string `yaml:"op_id,omitempty"`
}

// Dead reports whether the companion is dead.
func (c Companion) Dead() bool { return c.Death != nil }

func (r Record) companionIndex(companionID int) int {
	for i, c := range r.Companions {
		if c.ID == companionID {
			return i
		}
	}
	return -1
}

// isDead reports whether key names a dead companion of the record.
func (r Record) isDead(key MemberKey) bool {
	id, ok := CompanionIDFromMemberKey(key)
	if !ok {
		return false
	}
	i := r.companionIndex(id)
	return i >= 0 && r.Companions[i].Dead()
}

// MarkDead records a companion's death: it leaves the formation, and the
// cell it held is remembered in the death.
func (r *Registry) MarkDead(leaderUserID, companionID int, death CompanionDeath) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	i := record.companionIndex(companionID)
	if i < 0 {
		return ErrUnknownMember
	}
	if record.Companions[i].Dead() {
		return ErrMemberDead
	}
	key := CompanionMemberKey(companionID)
	death.Row, death.Col, death.Placed = record.Formation.Find(key)
	if !death.Placed {
		death.Row, death.Col = 0, 0
	}
	record.Formation.Clear(key)
	record.Companions[i].Death = &death
	r.Put(record)
	return nil
}

// SetRemaining records a dead companion's remaining allowance (never below
// zero) and whether the leader has been warned.
func (r *Registry) SetRemaining(leaderUserID, companionID, remaining int, warned bool) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	i := record.companionIndex(companionID)
	if i < 0 {
		return ErrUnknownMember
	}
	if !record.Companions[i].Dead() {
		return ErrNotDead
	}
	record.Companions[i].Death.Remaining = max(remaining, 0)
	record.Companions[i].Death.Warned = warned
	r.Put(record)
	return nil
}

// Revive ends a companion's death. It returns to the cell it held when
// that cell is still free.
func (r *Registry) Revive(leaderUserID, companionID int) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	i := record.companionIndex(companionID)
	if i < 0 {
		return ErrUnknownMember
	}
	death := record.Companions[i].Death
	if death == nil {
		return ErrNotDead
	}
	record.Companions[i].Death = nil
	if death.Placed && record.Formation.At(death.Row, death.Col) == "" {
		_ = record.Formation.Place(CompanionMemberKey(companionID), death.Row, death.Col)
	}
	r.Put(record)
	return nil
}

// Lose removes a companion for good and remembers it among the lost (the
// MaxLost most recent). Its ID stays spent. It reports whether the
// companion was in the record.
func (r *Registry) Lose(leaderUserID, companionID int, lost LostCompanion) bool {
	if !r.Dismiss(leaderUserID, companionID) {
		return false
	}
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	record.Lost = append(record.Lost, lost)
	if len(record.Lost) > MaxLost {
		record.Lost = append([]LostCompanion(nil), record.Lost[len(record.Lost)-MaxLost:]...)
	}
	r.Put(record)
	return true
}
