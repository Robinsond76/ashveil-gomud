// Package company contains the durable company and companion domain model.
package company

import (
	"errors"
	"strings"
)

// MaxCompanions is the hard upper bound on companions for one company: a
// leader plus this many companions, for a five-character maximum.
const MaxCompanions = 4

func clampCompanionLimit(limit int) int {
	if limit < 1 {
		return 1
	}
	if limit > MaxCompanions {
		return MaxCompanions
	}
	return limit
}

var (
	ErrInvalidLeader      = errors.New("invalid leader user ID")
	ErrInvalidTemplate    = errors.New("invalid mob template ID")
	ErrInvalidCompanionID = errors.New("invalid companion ID")
	ErrTemplateNotAllowed = errors.New("mob template is not allowed")
	ErrCompanyFull        = errors.New("company is full")
	// ErrArchetypeAlreadySet is returned when a companion's archetype is
	// already chosen; the choice is permanent (Phase 17).
	ErrArchetypeAlreadySet = errors.New("companion archetype is already set")
	ErrInvalidArchetype    = errors.New("invalid archetype")
)

type Companion struct {
	ID            int `yaml:"id"`
	MobTemplateID int `yaml:"mob_template_id"`
	// Archetype is the companion's Phase 17 archetype id. Empty for
	// companions recruited before archetypes existed, until the leader sets
	// one. It is set at most once.
	Archetype string `yaml:"archetype,omitempty"`
	// Disposition is the companion's Phase 21a alignment and loyalty. Nil
	// for a companion saved before Phase 21a, until the module seeds it.
	Disposition *Disposition `yaml:"disposition,omitempty"`
	// State is the companion's Phase 22b level and gear. Nil for a
	// companion saved before Phase 22b, until the module initializes it
	// from the mob template.
	State *MemberState `yaml:"state,omitempty"`
	// Death is set while the companion is dead (Phase 25b).
	Death *CompanionDeath `yaml:"death,omitempty"`
}

type Record struct {
	LeaderUserID    int         `yaml:"leader_user_id"`
	Companions      []Companion `yaml:"companions"`
	Formation       Formation   `yaml:"formation"`
	NextCompanionID int         `yaml:"next_companion_id,omitempty"`
	// Claimed lists the mob template IDs of free tutorial recruits this
	// leader has claimed (Phase 22c). A claim is permanent: it outlives
	// dismissal, desertion, and death.
	Claimed []int `yaml:"claimed,omitempty"`
	// Service is each member's Phase 24 time with the band (chemistry).
	Service []Service `yaml:"service,omitempty"`
	// Lost are the companions whose rescue allowance ran out (Phase 25b).
	Lost []LostCompanion `yaml:"lost,omitempty"`
}

// HasClaimed reports whether the tutorial recruit of this template has
// been claimed.
func (r Record) HasClaimed(mobTemplateID int) bool {
	for _, id := range r.Claimed {
		if id == mobTemplateID {
			return true
		}
	}
	return false
}

type Registry struct {
	Companies map[int]Record `yaml:"companies"`
	// DriftIn is the number of rounds until the next alignment drift tick
	// (Phase 21a). 0 or out of range means a full interval.
	DriftIn int `yaml:"drift_in,omitempty"`
}

func NewRegistry() *Registry {
	return &Registry{Companies: make(map[int]Record)}
}

func (r *Registry) Get(leaderUserID int) (Record, bool) {
	if r == nil {
		return Record{}, false
	}
	record, ok := r.Companies[leaderUserID]
	if !ok {
		return Record{}, false
	}
	record.Companions = append([]Companion(nil), record.Companions...)
	if record.Claimed != nil {
		record.Claimed = append([]int(nil), record.Claimed...)
	}
	if record.Service != nil {
		record.Service = append([]Service(nil), record.Service...)
	}
	if record.Lost != nil {
		record.Lost = append([]LostCompanion(nil), record.Lost...)
	}
	for i, c := range record.Companions {
		if c.Disposition != nil {
			d := *c.Disposition
			record.Companions[i].Disposition = &d
		}
		if c.State != nil {
			s := c.State.Clone()
			record.Companions[i].State = &s
		}
		if c.Death != nil {
			d := *c.Death
			record.Companions[i].Death = &d
		}
	}
	return record, true
}

// Put stores a record after pruning stale formation cells and normalizing the
// next companion ID. A record with no companions and an empty formation is
// removed entirely unless it carries a companion-ID high-water mark above 1,
// which must survive dismissal so IDs are never reused, a tutorial claim, or
// a lost companion.
func (r *Registry) Put(record Record) {
	if r.Companies == nil {
		r.Companies = make(map[int]Record)
	}
	record.NextCompanionID = normalizeNextCompanionID(record)
	valid := validMemberKeys(record)
	record.Formation.Prune(valid)
	record.Service = pruneService(record.Service, valid)
	if len(record.Companions) == 0 && record.Formation.empty() && record.NextCompanionID <= 1 && len(record.Claimed) == 0 && len(record.Lost) == 0 {
		delete(r.Companies, record.LeaderUserID)
		return
	}
	r.Companies[record.LeaderUserID] = record
}

// ReserveNextCompanionID raises the durable companion-ID high-water mark for
// a leader without creating a companion. It never lowers an existing mark.
func (r *Registry) ReserveNextCompanionID(leaderUserID, nextID int) error {
	if leaderUserID <= 0 {
		return ErrInvalidLeader
	}
	if nextID < 1 {
		return ErrInvalidCompanionID
	}
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	if record.NextCompanionID < nextID {
		record.NextCompanionID = nextID
	}
	r.Put(record)
	return nil
}

// Claim records a claimed tutorial recruit. It is idempotent.
func (r *Registry) Claim(leaderUserID, mobTemplateID int) error {
	if leaderUserID <= 0 {
		return ErrInvalidLeader
	}
	if mobTemplateID <= 0 {
		return ErrInvalidTemplate
	}
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	if !record.HasClaimed(mobTemplateID) {
		record.Claimed = append(record.Claimed, mobTemplateID)
	}
	r.Put(record)
	return nil
}

// Summon adds a companion and returns it with its assigned ID.
func (r *Registry) Summon(leaderUserID, mobTemplateID int, allowed map[int]struct{}, maxCompanions int) (Companion, error) {
	if leaderUserID <= 0 {
		return Companion{}, ErrInvalidLeader
	}
	if mobTemplateID <= 0 {
		return Companion{}, ErrInvalidTemplate
	}
	if _, ok := allowed[mobTemplateID]; !ok {
		return Companion{}, ErrTemplateNotAllowed
	}
	maxCompanions = clampCompanionLimit(maxCompanions)
	record, _ := r.Get(leaderUserID)
	record.LeaderUserID = leaderUserID
	if len(record.Companions) >= maxCompanions {
		return Companion{}, ErrCompanyFull
	}
	record.NextCompanionID = normalizeNextCompanionID(record)
	companion := Companion{ID: record.NextCompanionID, MobTemplateID: mobTemplateID}
	record.NextCompanionID++
	record.Companions = append(record.Companions, companion)
	r.Put(record)
	return companion, nil
}

// SetCompanionArchetype records a companion's archetype. It is set at most
// once: a companion that already has one returns ErrArchetypeAlreadySet.
// Whether the archetype exists is the caller's concern.
func (r *Registry) SetCompanionArchetype(leaderUserID, companionID int, archetype string) error {
	archetype = strings.ToLower(strings.TrimSpace(archetype))
	if archetype == "" {
		return ErrInvalidArchetype
	}
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID != companionID {
			continue
		}
		if c.Archetype != "" {
			return ErrArchetypeAlreadySet
		}
		record.Companions[i].Archetype = archetype
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}

// Dismiss removes one companion and its formation cells. It is idempotent.
func (r *Registry) Dismiss(leaderUserID, companionID int) bool {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return false
	}
	idx := -1
	for i, c := range record.Companions {
		if c.ID == companionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	if companionID >= record.NextCompanionID {
		record.NextCompanionID = companionID + 1
	}
	record.Companions = append(record.Companions[:idx], record.Companions[idx+1:]...)
	if len(record.Companions) == 0 {
		record.Companions = nil
	}
	record.Formation.Clear(CompanionMemberKey(companionID))
	r.Put(record)
	return true
}

// DismissAll removes every companion and returns how many were removed. The
// leader's own formation placement is preserved.
func (r *Registry) DismissAll(leaderUserID int) int {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return 0
	}
	count := len(record.Companions)
	for _, c := range record.Companions {
		if c.ID >= record.NextCompanionID {
			record.NextCompanionID = c.ID + 1
		}
		record.Formation.Clear(CompanionMemberKey(c.ID))
	}
	record.Companions = nil
	r.Put(record)
	return count
}

// PlaceMember places a member in the formation, creating a leader-only record
// when the leader is placed before any companion exists.
func (r *Registry) PlaceMember(leaderUserID int, key MemberKey, row, col int) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		if key != LeaderMemberKey {
			return ErrUnknownMember
		}
		record = Record{LeaderUserID: leaderUserID}
	}
	if !validMemberKeys(record)[key] {
		return ErrUnknownMember
	}
	if record.isDead(key) {
		return ErrMemberDead
	}
	if err := record.Formation.Place(key, row, col); err != nil {
		return err
	}
	r.Put(record)
	return nil
}

// SwapMembers exchanges two members' formation cells.
func (r *Registry) SwapMembers(leaderUserID int, a, b MemberKey) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	valid := validMemberKeys(record)
	if !valid[a] || !valid[b] {
		return ErrUnknownMember
	}
	if record.isDead(a) || record.isDead(b) {
		return ErrMemberDead
	}
	if err := record.Formation.Swap(a, b); err != nil {
		return err
	}
	r.Put(record)
	return nil
}

// ClearMember removes a member from the formation.
func (r *Registry) ClearMember(leaderUserID int, key MemberKey) error {
	record, ok := r.Get(leaderUserID)
	if !ok || !validMemberKeys(record)[key] {
		return ErrUnknownMember
	}
	if _, _, found := record.Formation.Find(key); !found {
		return ErrUnknownMember
	}
	record.Formation.Clear(key)
	r.Put(record)
	return nil
}

// normalizeNextCompanionID returns the durable next companion ID: at least 1
// and strictly greater than every companion currently in the record. The
// stored high-water mark is never lowered.
func normalizeNextCompanionID(record Record) int {
	next := record.NextCompanionID
	if next < 1 {
		next = 1
	}
	for _, c := range record.Companions {
		if c.ID >= next {
			next = c.ID + 1
		}
	}
	return next
}

func validMemberKeys(record Record) map[MemberKey]bool {
	valid := map[MemberKey]bool{LeaderMemberKey: true}
	for _, c := range record.Companions {
		valid[CompanionMemberKey(c.ID)] = true
	}
	return valid
}

// Clone returns a deep copy of the registry.
func (r *Registry) Clone() Registry {
	out := Registry{Companies: make(map[int]Record, len(r.Companies)), DriftIn: r.DriftIn}
	for leaderUserID := range r.Companies {
		record, _ := r.Get(leaderUserID) // Formation is an array, copied by value
		out.Companies[leaderUserID] = record
	}
	return out
}
