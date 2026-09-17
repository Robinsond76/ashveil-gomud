// Package company contains the durable company and companion domain model.
package company

import "errors"

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
	ErrTemplateNotAllowed = errors.New("mob template is not allowed")
	ErrCompanyFull        = errors.New("company is full")
)

type Companion struct {
	ID            int `yaml:"id"`
	MobTemplateID int `yaml:"mob_template_id"`
}

type Record struct {
	LeaderUserID int         `yaml:"leader_user_id"`
	Companions   []Companion `yaml:"companions"`
	Formation    Formation   `yaml:"formation"`
}

type Registry struct {
	Companies map[int]Record `yaml:"companies"`
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
	return record, true
}

// Put stores a record after pruning stale formation cells. A record with no
// companions and an empty formation is removed entirely.
func (r *Registry) Put(record Record) {
	if r.Companies == nil {
		r.Companies = make(map[int]Record)
	}
	record.Formation.Prune(validMemberKeys(record))
	if len(record.Companions) == 0 && record.Formation.empty() {
		delete(r.Companies, record.LeaderUserID)
		return
	}
	r.Companies[record.LeaderUserID] = record
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
	companion := Companion{ID: nextCompanionID(record), MobTemplateID: mobTemplateID}
	record.Companions = append(record.Companions, companion)
	r.Put(record)
	return companion, nil
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
	record.Companions = append(record.Companions[:idx], record.Companions[idx+1:]...)
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

func nextCompanionID(record Record) int {
	maxID := 0
	for _, c := range record.Companions {
		if c.ID > maxID {
			maxID = c.ID
		}
	}
	return maxID + 1
}

func validMemberKeys(record Record) map[MemberKey]bool {
	valid := map[MemberKey]bool{LeaderMemberKey: true}
	for _, c := range record.Companions {
		valid[CompanionMemberKey(c.ID)] = true
	}
	return valid
}
