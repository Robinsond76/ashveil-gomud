// Package company contains the durable company and companion domain model.
package company

import (
	"errors"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"slices"
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
	PackGranted bool `yaml:"pack_granted,omitempty"`
	// PendingReturn marks temporary morale flight; gear remains in State.
	PendingReturn bool `yaml:"pending_return,omitempty"`
	MoraleDesert  bool `yaml:"morale_desert,omitempty"`
	ReturnHP      int  `yaml:"return_hp,omitempty"`
	ReturnMana    int  `yaml:"return_mana,omitempty"`
	ID            int  `yaml:"id"`
	MobTemplateID int  `yaml:"mob_template_id"`
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
	// Name and Description are a generated recruit's own (Phase 32a2).
	// Blank for an authored companion, which uses its template's.
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	// GrowthFocus is the stat the leader chose to favour in the
	// companion's growth (Phase 33h1); empty grows by archetype alone.
	GrowthFocus string `yaml:"growth_focus,omitempty"`
	// Separation is set while the companion is separated from its leader
	// (Phase 33h3): off the map, its State the last snapshot.
	Separation *Separation `yaml:"separation,omitempty"`
	// Skills are its ranks in optional skills (Phase 35c), trained with
	// "company train" or brought by a recruit. GrantedSkills are the ranks
	// it arrived with, which cost no points. Both live on the record, not
	// in State, so no gear snapshot can drop them; nil means none.
	Skills        map[string]int `yaml:"skills,omitempty"`
	GrantedSkills map[string]int `yaml:"granted_skills,omitempty"`
	// Class and Talents are the advanced or elite class it promoted into
	// and the talents it picked (Phase 38b); everything they give is
	// derived from them and its level. Blank for an unpromoted companion,
	// so old saves read as unpromoted.
	Class   string   `yaml:"class,omitempty"`
	Talents []string `yaml:"talents,omitempty"`
	// Personality is the temperament its banter is drawn by (Phase 49),
	// rolled when it joins. Blank on a companion saved before then, which
	// banter derives from its ID until one is set.
	Personality string `yaml:"personality,omitempty"`
	// Opinions is what it remembers saying about the leader's choices
	// (Phase 64). Nil until it has said anything.
	Opinions *OpinionMemory `yaml:"opinions,omitempty"`
}

// Identity is what a companion's live mob is called and looks like, over
// its template's (Phase 32a2). Blank fields keep the template's.
type Identity struct {
	Archetype   string // Derived from the durable companion record, not separately saved.
	Name        string
	Description string
	// Skills are its optional-skill ranks (Phase 35c), from the record,
	// written onto the live mob at every spawn.
	Skills map[string]int
	// Class and Talents are its Phase 38b class state, written onto the
	// live mob at every spawn.
	Class   string
	Talents []string
}

// Identity is the companion's own name and description.
func (c Companion) Identity() Identity {
	return Identity{Name: c.Name, Description: c.Description, Archetype: c.Archetype, Skills: cloneRanks(c.Skills), Class: c.Class, Talents: slices.Clone(c.Talents)}
}

// AssetOperation is a write-ahead record. Company gear is saved with this
// resulting leader state before the user file is changed. The user stores ID
// with its assets, making replay idempotent even if clearing this record fails.
type AssetOperation struct {
	ID        string          `yaml:"id"`
	Items     []items.Item    `yaml:"items"`
	Equipment characters.Worn `yaml:"equipment"`
	Gold      int             `yaml:"gold"`
}

type Record struct {
	LeaderPackGranted bool            `yaml:"leader_pack_granted,omitempty"`
	FormationVersion  int             `yaml:"formation_version,omitempty"`
	AssetOperation    *AssetOperation `yaml:"asset_operation,omitempty"`

	MercyPending    []MercyEffect `yaml:"mercy_pending,omitempty"`
	LeaderUserID    int           `yaml:"leader_user_id"`
	Companions      []Companion   `yaml:"companions"`
	Formation       Formation     `yaml:"formation"`
	NextCompanionID int           `yaml:"next_companion_id,omitempty"`
	// Claimed lists the mob template IDs of free tutorial recruits this
	// leader has claimed (Phase 22c). A claim is permanent: it outlives
	// dismissal, desertion, and death.
	Claimed []int `yaml:"claimed,omitempty"`
	// Service is each member's Phase 24 time with the band (chemistry).
	Service []Service `yaml:"service,omitempty"`
	// Lost are the companions whose rescue allowance ran out (Phase 25b).
	Lost []LostCompanion `yaml:"lost,omitempty"`
	// Rosters are the leader's generated recruit candidates, one per
	// recruiter room (Phase 32a2).
	Rosters []Roster `yaml:"rosters,omitempty"`
	// AppliedOps (Phase 33f3) are the most recent operations already
	// applied to this record (a camp's vigil), so a retry never repeats.
	AppliedOps []string `yaml:"applied_ops,omitempty"`
}

// MaxAppliedOps bounds Record.AppliedOps.
const MaxAppliedOps = 32

// HasApplied reports whether an operation was already applied.
func (r Record) HasApplied(op string) bool {
	for _, a := range r.AppliedOps {
		if a == op {
			return true
		}
	}
	return false
}

// MarkApplied remembers op on the record, keeping the last MaxAppliedOps.
func (r *Record) MarkApplied(op string) {
	applied := append(append([]string(nil), r.AppliedOps...), op)
	if len(applied) > MaxAppliedOps {
		applied = applied[len(applied)-MaxAppliedOps:]
	}
	r.AppliedOps = applied
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
	if record.AssetOperation != nil {
		op := *record.AssetOperation
		st := MemberState{Items: op.Items, Equipment: op.Equipment}.Clone()
		op.Items, op.Equipment = st.Items, st.Equipment
		record.AssetOperation = &op
	}
	record.Companions = append([]Companion(nil), record.Companions...)
	record.MercyPending = append([]MercyEffect(nil), record.MercyPending...)
	if record.AppliedOps != nil {
		record.AppliedOps = append([]string(nil), record.AppliedOps...)
	}
	if record.Claimed != nil {
		record.Claimed = append([]int(nil), record.Claimed...)
	}
	if record.Service != nil {
		record.Service = append([]Service(nil), record.Service...)
	}
	if record.Lost != nil {
		record.Lost = append([]LostCompanion(nil), record.Lost...)
	}
	if record.Rosters != nil {
		rosters := make([]Roster, len(record.Rosters))
		for i, ros := range record.Rosters {
			rosters[i] = ros.clone()
		}
		record.Rosters = rosters
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
		if c.Separation != nil {
			sep := *c.Separation
			record.Companions[i].Separation = &sep
		}
		if c.Opinions != nil {
			record.Companions[i].Opinions = c.Opinions.Clone()
		}
		record.Companions[i].Skills = cloneRanks(c.Skills)
		record.Companions[i].GrantedSkills = cloneRanks(c.GrantedSkills)
	}
	return record, true
}

// Remove drops a leader's record entirely, claims and lost companions
// included (Phase 32b: a purged user), and reports whether there was one.
func (r *Registry) Remove(leaderUserID int) bool {
	if r == nil {
		return false
	}
	_, ok := r.Companies[leaderUserID]
	delete(r.Companies, leaderUserID)
	return ok
}

// Put stores a record after pruning stale formation cells and normalizing the
// next companion ID. A record with no companions keeps its leader at the
// centre cell (Phase 34a), so Put never drops a record: every leader who has
// played keeps one, which also holds durable markers such as the starter
// pack grant (34b). Remove is the only way to drop a record.
func (r *Registry) Put(record Record) {
	if r.Companies == nil {
		r.Companies = make(map[int]Record)
	}
	record.NextCompanionID = normalizeNextCompanionID(record)
	valid := validMemberKeys(record)
	record.Formation.Prune(valid)
	if len(record.Companions) == 0 {
		_ = record.Formation.Place(LeaderMemberKey, 1, 1)
	}
	record.Service = pruneService(record.Service, valid)
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
	record, exists := r.Get(leaderUserID)
	if !exists && nextID == 1 {
		return nil
	}
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
	record.BackfillFormation()
	record.Formation.PlaceVacant(LeaderMemberKey)
	record.Formation.PlaceVacant(CompanionMemberKey(companion.ID))
	r.Put(record)
	return companion, nil
}

// SetIdentity gives a companion its own name and description.
func (r *Registry) SetIdentity(leaderUserID, companionID int, id Identity) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID == companionID {
			record.Companions[i].Name = strings.TrimSpace(id.Name)
			record.Companions[i].Description = strings.TrimSpace(id.Description)
			r.Put(record)
			return nil
		}
	}
	return ErrUnknownMember
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

// SetCompanionPersonality records a companion's banter personality (Phase
// 49). It is set at most once: a companion that has one keeps it.
func (r *Registry) SetCompanionPersonality(leaderUserID, companionID int, personality string) error {
	record, ok := r.Get(leaderUserID)
	if !ok {
		return ErrUnknownMember
	}
	for i, c := range record.Companions {
		if c.ID != companionID {
			continue
		}
		if c.Personality != "" {
			return nil
		}
		record.Companions[i].Personality = personality
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}

// SetCompanionClass records the class a companion promoted into (Phase 38b).
// The rules (level, alignment, route) are the caller's; this refuses an
// unknown companion and never lowers a class: a repeated promotion to the
// class it already has is a no-op, and a class below its current tier is
// refused.
func (r *Registry) SetCompanionClass(leaderUserID, companionID int, class string) error {
	class = strings.ToLower(strings.TrimSpace(class))
	if class == "" {
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
		record.Companions[i].Class = class
		r.Put(record)
		return nil
	}
	return ErrUnknownMember
}

// AddCompanionTalent appends a picked talent to a companion (Phase 38b).
func (r *Registry) AddCompanionTalent(leaderUserID, companionID int, talent string) error {
	talent = strings.ToLower(strings.TrimSpace(talent))
	if talent == "" {
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
		record.Companions[i].Talents = append(slices.Clone(c.Talents), talent)
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
	if len(record.Companions) == 0 && key == LeaderMemberKey && (row != 1 || col != 1) {
		return ErrSoloFormation
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
	if len(record.Companions) == 0 && key == LeaderMemberKey {
		return ErrSoloFormation
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
