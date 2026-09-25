// Package survival defines the durable company survival domain: per-member
// hunger, thirst, and fatigue values, their threshold bands, and the explicit
// service operations that mutate them.
//
// The package has no dependency on GoMud users, mobs, rooms, commands,
// plugins, clocks, or runtime instances. State changes only through explicit
// calls; there is no idle drain, timer, or global-time mutation. Company
// identity is expressed with the stable leader/companion member keys defined by
// internal/company, never a runtime mob InstanceId.
package survival

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MemberKey identifies a company member for survival state.
type MemberKey = domain.MemberKey

// LeaderMemberKey is the survival key for the company leader.
const LeaderMemberKey = domain.LeaderMemberKey

// CompanionMemberKey returns the survival key for a companion ID.
func CompanionMemberKey(id int) MemberKey { return domain.CompanionMemberKey(id) }

// Needs are the individual, normalized survival values for one member. 100
// means fully supplied or rested; 0 means depleted or exhausted.
type Needs struct {
	Hunger  int `yaml:"hunger"`
	Thirst  int `yaml:"thirst"`
	Fatigue int `yaml:"fatigue"`
}

// FullNeeds is the default state for a new or missing member record.
func FullNeeds() Needs {
	return Needs{Hunger: 100, Thirst: 100, Fatigue: 100}
}

// Band is the qualitative threshold a need value falls into.
type Band uint8

const (
	BandDepleted Band = iota // 0
	BandCritical             // 1..25
	BandLow                  // 26..50
	BandSteady               // 51..75
	BandFull                 // 76..100
)

// Change reports the band before and after a mutation so callers can announce
// crossings without duplicating threshold logic.
type Change struct {
	Before Band `yaml:"before"`
	After  Band `yaml:"after"`
}

// Crossed reports whether the need changed band.
func (c Change) Crossed() bool { return c.Before != c.After }

// Exertion is an explicit cost applied by later travel and effort systems.
type Exertion struct {
	Hunger  int
	Thirst  int
	Fatigue int
}

// Benefit is the supply a consumable or effect offers to a member.
type Benefit struct {
	Nutrition int
	Hydration int
	Fatigue   int
}

// ProvisionResult describes the outcome of applying a Benefit to a member.
type ProvisionResult struct {
	Member  MemberKey
	Name    string
	Needs   Needs
	Hunger  Change
	Thirst  Change
	Fatigue Change
}

// Crossed reports whether any need changed band.
func (r ProvisionResult) Crossed() bool {
	return r.Hunger.Crossed() || r.Thirst.Crossed() || r.Fatigue.Crossed()
}

// ExertionResult describes one company member's response to an exertion cost.
type ExertionResult struct {
	Member  MemberKey
	Name    string
	Needs   Needs
	Hunger  Change
	Thirst  Change
	Fatigue Change
}

// Crossed reports whether any need changed band.
func (r ExertionResult) Crossed() bool {
	return r.Hunger.Crossed() || r.Thirst.Crossed() || r.Fatigue.Crossed()
}

// MemberNeeds is a member's current survival values for read-only rendering.
type MemberNeeds struct {
	Key   MemberKey
	Name  string
	Needs Needs
}

// Errors returned by the survival domain and provider seam.
var (
	ErrInvalidMember          = errors.New("survival: invalid company member")
	ErrUnknownMember          = errors.New("survival: unknown company member")
	ErrAmbiguousMember        = errors.New("survival: ambiguous company member")
	ErrDeadMember             = errors.New("survival: that companion is dead")
	ErrInvalidAmount          = errors.New("survival: amount must be positive")
	ErrPersistenceUnavailable = errors.New("survival: persistence unavailable")
	ErrProvisionUnavailable   = errors.New("survival: provisioning unavailable")
	ErrExertionUnavailable    = errors.New("survival: exertion unavailable")
	ErrExertionConflict       = errors.New("survival: exertion operation conflicts with prior cost")
	ErrRestUnavailable        = errors.New("survival: rest recovery unavailable")
	ErrRestConflict           = errors.New("survival: rest operation conflicts with prior amount")
)

func clamp(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// Normalize clamps every need into 0..100.
func Normalize(needs Needs) Needs {
	return Needs{
		Hunger:  clamp(needs.Hunger),
		Thirst:  clamp(needs.Thirst),
		Fatigue: clamp(needs.Fatigue),
	}
}

// BandFor maps a need value to its threshold band.
func BandFor(value int) Band {
	switch value = clamp(value); {
	case value == 0:
		return BandDepleted
	case value <= 25:
		return BandCritical
	case value <= 50:
		return BandLow
	case value <= 75:
		return BandSteady
	default:
		return BandFull
	}
}

// HungerLabel renders the hunger-specific label for a value.
func HungerLabel(value int) string {
	switch BandFor(value) {
	case BandDepleted:
		return "Starving"
	case BandCritical:
		return "Starving"
	case BandLow:
		return "Hungry"
	case BandSteady:
		return "Sated"
	default:
		return "Well fed"
	}
}

// ThirstLabel renders the thirst-specific label for a value.
func ThirstLabel(value int) string {
	switch BandFor(value) {
	case BandDepleted:
		return "Dehydrated"
	case BandCritical:
		return "Parched"
	case BandLow:
		return "Thirsty"
	case BandSteady:
		return "Comfortable"
	default:
		return "Hydrated"
	}
}

// FatigueLabel renders the fatigue-specific label for a value.
func FatigueLabel(value int) string {
	switch BandFor(value) {
	case BandDepleted:
		return "Collapsed"
	case BandCritical:
		return "Exhausted"
	case BandLow:
		return "Tired"
	case BandSteady:
		return "Ready"
	default:
		return "Rested"
	}
}

// ValidMemberKey reports whether a key names the leader or a positive companion.
func ValidMemberKey(key MemberKey) bool {
	if key == LeaderMemberKey {
		return true
	}
	raw := string(key)
	if !strings.HasPrefix(raw, "companion:") {
		return false
	}
	id, err := strconv.Atoi(strings.TrimPrefix(raw, "companion:"))
	return err == nil && id > 0
}

func validateMember(leaderUserID int, key MemberKey) error {
	if leaderUserID <= 0 {
		return ErrInvalidMember
	}
	if !ValidMemberKey(key) {
		return ErrInvalidMember
	}
	return nil
}

// Registry holds normalized needs keyed by leader user ID and stable member key.
type Registry struct {
	Leaders                  map[int]map[MemberKey]Needs `yaml:"leaders"`
	ReservedNextCompanionIDs map[int]int                 `yaml:"reserved_next_companion_ids,omitempty"`
	AppliedExertion          map[int]map[string]Exertion `yaml:"applied_exertion,omitempty"`
	AppliedRestOperation     map[int]map[string]int      `yaml:"applied_rest_operation,omitempty"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		Leaders:                  map[int]map[MemberKey]Needs{},
		ReservedNextCompanionIDs: map[int]int{},
		AppliedExertion:          map[int]map[string]Exertion{},
		AppliedRestOperation:     map[int]map[string]int{},
	}
}

func (r *Registry) ensureMaps() {
	if r.Leaders == nil {
		r.Leaders = map[int]map[MemberKey]Needs{}
	}
	if r.ReservedNextCompanionIDs == nil {
		r.ReservedNextCompanionIDs = map[int]int{}
	}
	if r.AppliedExertion == nil {
		r.AppliedExertion = map[int]map[string]Exertion{}
	}
	if r.AppliedRestOperation == nil {
		r.AppliedRestOperation = map[int]map[string]int{}
	}
}

// ReserveNextCompanionID records a durable lower bound for a leader's next
// company ID. Survival writes this with the companion state so a failed
// company-file write cannot make an already-issued identity reusable.
func (r *Registry) ReserveNextCompanionID(leaderUserID, nextID int) error {
	if leaderUserID <= 0 || nextID < 1 {
		return ErrInvalidMember
	}
	r.ensureMaps()
	if r.ReservedNextCompanionIDs[leaderUserID] < nextID {
		r.ReservedNextCompanionIDs[leaderUserID] = nextID
	}
	return nil
}

// NextReservedCompanionID returns the durable lower bound for a leader's next
// companion ID, defaulting to one when no survival reservation exists.
func (r *Registry) NextReservedCompanionID(leaderUserID int) (int, error) {
	if leaderUserID <= 0 {
		return 0, ErrInvalidMember
	}
	if r == nil || r.ReservedNextCompanionIDs == nil || r.ReservedNextCompanionIDs[leaderUserID] < 1 {
		return 1, nil
	}
	return r.ReservedNextCompanionIDs[leaderUserID], nil
}

// Ensure creates a fully supplied default record for a member if absent.
func (r *Registry) Ensure(leaderUserID int, key MemberKey) error {
	if err := validateMember(leaderUserID, key); err != nil {
		return err
	}
	r.ensureMaps()
	if r.Leaders[leaderUserID] == nil {
		r.Leaders[leaderUserID] = map[MemberKey]Needs{}
	}
	if _, ok := r.Leaders[leaderUserID][key]; !ok {
		r.Leaders[leaderUserID][key] = FullNeeds()
	}
	return nil
}

// PutNeeds stores normalized needs for a member, creating the record if needed.
func (r *Registry) PutNeeds(leaderUserID int, key MemberKey, needs Needs) error {
	if err := validateMember(leaderUserID, key); err != nil {
		return err
	}
	r.ensureMaps()
	if r.Leaders[leaderUserID] == nil {
		r.Leaders[leaderUserID] = map[MemberKey]Needs{}
	}
	r.Leaders[leaderUserID][key] = Normalize(needs)
	return nil
}

// NeedsFor returns normalized needs for a member.
func (r *Registry) NeedsFor(leaderUserID int, key MemberKey) (Needs, bool) {
	if r == nil || r.Leaders == nil {
		return Needs{}, false
	}
	byMember, ok := r.Leaders[leaderUserID]
	if !ok {
		return Needs{}, false
	}
	needs, ok := byMember[key]
	if !ok {
		return Needs{}, false
	}
	return Normalize(needs), true
}

// MustNeedsFor returns needs for a member, panicking when absent.
func (r *Registry) MustNeedsFor(leaderUserID int, key MemberKey) Needs {
	needs, ok := r.NeedsFor(leaderUserID, key)
	if !ok {
		panic("survival: no needs for member")
	}
	return needs
}

// Remove deletes one member record, dropping the leader entry when it empties.
func (r *Registry) Remove(leaderUserID int, key MemberKey) {
	if r == nil || r.Leaders == nil {
		return
	}
	delete(r.Leaders[leaderUserID], key)
	if len(r.Leaders[leaderUserID]) == 0 {
		delete(r.Leaders, leaderUserID)
	}
}

// RemoveLeader drops every record for a leader.
func (r *Registry) RemoveLeader(leaderUserID int) {
	if r == nil || r.Leaders == nil {
		return
	}
	delete(r.Leaders, leaderUserID)
}

// Members lists a leader's member keys: leader first, then companions by ID.
func (r *Registry) Members(leaderUserID int) []MemberKey {
	if r == nil || r.Leaders == nil {
		return nil
	}
	byMember := r.Leaders[leaderUserID]
	if len(byMember) == 0 {
		return nil
	}
	keys := make([]MemberKey, 0, len(byMember))
	for key := range byMember {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == LeaderMemberKey {
			return true
		}
		if keys[j] == LeaderMemberKey {
			return false
		}
		return companionOrdinal(keys[i]) < companionOrdinal(keys[j])
	})
	return keys
}

func companionOrdinal(key MemberKey) int {
	id, err := strconv.Atoi(strings.TrimPrefix(string(key), "companion:"))
	if err != nil {
		return 0
	}
	return id
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Leaders: map[int]map[MemberKey]Needs{}, ReservedNextCompanionIDs: map[int]int{}, AppliedExertion: map[int]map[string]Exertion{}, AppliedRestOperation: map[int]map[string]int{}}
	for leaderUserID, byMember := range r.Leaders {
		clone := make(map[MemberKey]Needs, len(byMember))
		for key, needs := range byMember {
			clone[key] = needs
		}
		out.Leaders[leaderUserID] = clone
	}
	for leaderUserID, nextID := range r.ReservedNextCompanionIDs {
		out.ReservedNextCompanionIDs[leaderUserID] = nextID
	}
	for id, operations := range r.AppliedExertion {
		out.AppliedExertion[id] = map[string]Exertion{}
		for key, cost := range operations {
			out.AppliedExertion[id][key] = cost
		}
	}
	for id, operations := range r.AppliedRestOperation {
		out.AppliedRestOperation[id] = map[string]int{}
		for key, fatigue := range operations {
			out.AppliedRestOperation[id][key] = fatigue
		}
	}
	return out
}

// ConsumeFood raises hunger and/or thirst. At least one benefit must be
// positive; the other may be zero.
func (r *Registry) ConsumeFood(leaderUserID int, key MemberKey, nutrition, hydration int) (hunger, thirst Change, err error) {
	if err := validateMember(leaderUserID, key); err != nil {
		return Change{}, Change{}, err
	}
	if nutrition < 0 || hydration < 0 || (nutrition == 0 && hydration == 0) {
		return Change{}, Change{}, ErrInvalidAmount
	}
	before, ok := r.NeedsFor(leaderUserID, key)
	if !ok {
		return Change{}, Change{}, ErrUnknownMember
	}
	after := before
	after.Hunger = clamp(before.Hunger + nutrition)
	after.Thirst = clamp(before.Thirst + hydration)
	if err := r.PutNeeds(leaderUserID, key, after); err != nil {
		return Change{}, Change{}, err
	}
	return changeFor(before.Hunger, after.Hunger), changeFor(before.Thirst, after.Thirst), nil
}

// ConsumeWater raises thirst.
func (r *Registry) ConsumeWater(leaderUserID int, key MemberKey, hydration int) (Change, error) {
	if err := validateMember(leaderUserID, key); err != nil {
		return Change{}, err
	}
	if hydration <= 0 {
		return Change{}, ErrInvalidAmount
	}
	before, ok := r.NeedsFor(leaderUserID, key)
	if !ok {
		return Change{}, ErrUnknownMember
	}
	after := before
	after.Thirst = clamp(before.Thirst + hydration)
	if err := r.PutNeeds(leaderUserID, key, after); err != nil {
		return Change{}, err
	}
	return changeFor(before.Thirst, after.Thirst), nil
}

// ApplyRestRecovery raises fatigue only.
func (r *Registry) ApplyRestRecovery(leaderUserID int, key MemberKey, fatigue int) (Change, error) {
	if err := validateMember(leaderUserID, key); err != nil {
		return Change{}, err
	}
	if fatigue <= 0 {
		return Change{}, ErrInvalidAmount
	}
	before, ok := r.NeedsFor(leaderUserID, key)
	if !ok {
		return Change{}, ErrUnknownMember
	}
	after := before
	after.Fatigue = clamp(before.Fatigue + fatigue)
	if err := r.PutNeeds(leaderUserID, key, after); err != nil {
		return Change{}, err
	}
	return changeFor(before.Fatigue, after.Fatigue), nil
}

// ApplyExertion lowers the requested needs. Costs are non-negative and at least
// one must be positive.
func (r *Registry) ApplyExertion(leaderUserID int, key MemberKey, cost Exertion) (hunger, thirst, fatigue Change, err error) {
	if err := validateMember(leaderUserID, key); err != nil {
		return Change{}, Change{}, Change{}, err
	}
	if cost.Hunger < 0 || cost.Thirst < 0 || cost.Fatigue < 0 || (cost.Hunger == 0 && cost.Thirst == 0 && cost.Fatigue == 0) {
		return Change{}, Change{}, Change{}, ErrInvalidAmount
	}
	before, ok := r.NeedsFor(leaderUserID, key)
	if !ok {
		return Change{}, Change{}, Change{}, ErrUnknownMember
	}
	after := before
	after.Hunger = clamp(before.Hunger - cost.Hunger)
	after.Thirst = clamp(before.Thirst - cost.Thirst)
	after.Fatigue = clamp(before.Fatigue - cost.Fatigue)
	if err := r.PutNeeds(leaderUserID, key, after); err != nil {
		return Change{}, Change{}, Change{}, err
	}
	return changeFor(before.Hunger, after.Hunger), changeFor(before.Thirst, after.Thirst), changeFor(before.Fatigue, after.Fatigue), nil
}

func changeFor(before, after int) Change {
	return Change{Before: BandFor(before), After: BandFor(after)}
}

// MemberRef is a current company member and its display name.
type MemberRef struct {
	Key  MemberKey
	Name string
	// Dead marks a dead companion awaiting resurrection (Phase 25b). It
	// stays on the roster, but spends and recovers nothing.
	Dead bool
}

// MemberSnapshot is the exact durable survival state for one companion at a
// point in time. Exists is false when the companion had no stored record, so a
// rollback can distinguish "remove the key" from "restore these needs" without
// manufacturing defaults.
type MemberSnapshot struct {
	Exists bool  `yaml:"exists"`
	Needs  Needs `yaml:"needs"`
}

// RosterProvider exposes the authoritative company roster to survival
// consumers for name resolution and status rendering. modules/company
// registers it during init so modules/survival never imports modules/company.
type RosterProvider interface {
	// Roster returns the current members for a leader, leader first. It
	// includes companions whose native mob is temporarily unavailable.
	Roster(leaderUserID int) []MemberRef
}

var (
	rosterMu       sync.RWMutex
	rosterProvider RosterProvider
)

// SetRosterProvider registers the active roster provider. Passing nil clears it.
func SetRosterProvider(p RosterProvider) {
	rosterMu.Lock()
	defer rosterMu.Unlock()
	rosterProvider = p
}

// CurrentRoster returns the authoritative roster when a provider is
// registered, otherwise nil.
func CurrentRoster(leaderUserID int) []MemberRef {
	rosterMu.RLock()
	p := rosterProvider
	rosterMu.RUnlock()
	if p == nil {
		return nil
	}
	return p.Roster(leaderUserID)
}

// Lifecycle synchronizes durable company roster changes with survival state.
// It is implemented by modules/survival and registered during module init.
// modules/company calls these functions after a committed roster change so the
// dependency stays one-way through this package.
type Lifecycle interface {
	EnsureCompanyMember(leaderUserID, companionID int) error
	// NextReservedCompanionID returns survival's durable lower bound for a
	// leader's next companion ID. Company uses it before assigning an ID so a
	// failed company-file write cannot make an issued ID reusable after restart.
	NextReservedCompanionID(leaderUserID int) (int, error)
	RemoveCompanyMember(leaderUserID, companionID int) error
	RemoveAllCompanyMembers(leaderUserID int) error
	// SnapshotCompanyMember captures the exact state before a roster mutation
	// so a failed downstream write can restore it verbatim.
	SnapshotCompanyMember(leaderUserID, companionID int) (MemberSnapshot, error)
	// RestoreCompanyMember writes a captured snapshot back, removing the key
	// when the snapshot recorded no stored state.
	RestoreCompanyMember(leaderUserID, companionID int, snapshot MemberSnapshot) error
	// ReconcileCompanyRosters aligns persisted companion keys with the loaded
	// company rosters, pruning orphans and initializing missing members.
	ReconcileCompanyRosters(rosters map[int][]MemberRef) error
}

var (
	lifecycleMu sync.RWMutex
	lifecycle   Lifecycle
)

// SetLifecycle registers the active lifecycle synchronizer. Passing nil clears it.
func SetLifecycle(l Lifecycle) {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	lifecycle = l
}

func currentLifecycle() Lifecycle {
	lifecycleMu.RLock()
	defer lifecycleMu.RUnlock()
	return lifecycle
}

// EnsureCompanyMember notifies the registered lifecycle that a companion was
// summoned. It is a no-op when no module is registered.
func EnsureCompanyMember(leaderUserID, companionID int) error {
	if l := currentLifecycle(); l != nil {
		return l.EnsureCompanyMember(leaderUserID, companionID)
	}
	return nil
}

// NextReservedCompanionID returns the survival-side durable lower bound for a
// leader's next companion ID. Without a loaded survival module, it defaults to
// one so company behavior remains unchanged.
func NextReservedCompanionID(leaderUserID int) (int, error) {
	if l := currentLifecycle(); l != nil {
		return l.NextReservedCompanionID(leaderUserID)
	}
	if leaderUserID <= 0 {
		return 0, ErrInvalidMember
	}
	return 1, nil
}

// RemoveCompanyMember notifies the registered lifecycle that a companion was
// dismissed. It is a no-op when no module is registered.
func RemoveCompanyMember(leaderUserID, companionID int) error {
	if l := currentLifecycle(); l != nil {
		return l.RemoveCompanyMember(leaderUserID, companionID)
	}
	return nil
}

// RemoveAllCompanyMembers notifies the registered lifecycle that every
// companion was dismissed. It is a no-op when no module is registered.
func RemoveAllCompanyMembers(leaderUserID int) error {
	if l := currentLifecycle(); l != nil {
		return l.RemoveAllCompanyMembers(leaderUserID)
	}
	return nil
}

// SnapshotCompanyMember captures exact state through the registered lifecycle.
// Without a module it returns an empty snapshot, matching the no-op removal.
func SnapshotCompanyMember(leaderUserID, companionID int) (MemberSnapshot, error) {
	if l := currentLifecycle(); l != nil {
		return l.SnapshotCompanyMember(leaderUserID, companionID)
	}
	return MemberSnapshot{}, nil
}

// RestoreCompanyMember restores a captured snapshot through the registered
// lifecycle. It is a no-op when no module is registered.
func RestoreCompanyMember(leaderUserID, companionID int, snapshot MemberSnapshot) error {
	if l := currentLifecycle(); l != nil {
		return l.RestoreCompanyMember(leaderUserID, companionID, snapshot)
	}
	return nil
}

// ReconcileCompanyRosters aligns persisted survival state with the loaded
// company rosters through the registered lifecycle. It is a no-op when no
// module is registered.
func ReconcileCompanyRosters(rosters map[int][]MemberRef) error {
	if l := currentLifecycle(); l != nil {
		return l.ReconcileCompanyRosters(rosters)
	}
	return nil
}

// Provisioner applies a Benefit to a selected company member. It is
// implemented by modules/survival and registered during module init.
type Provisioner interface {
	Provision(leaderUserID int, selector string, benefit Benefit) (ProvisionResult, error)
	// IsMemberSelector reports whether selector names a current company member,
	// letting command parsing distinguish a provision target from item text.
	IsMemberSelector(leaderUserID int, selector string) bool
}

var (
	provisionerMu sync.RWMutex
	provisioner   Provisioner
)

// SetProvisioner registers the active provisioner. Passing nil clears it.
func SetProvisioner(p Provisioner) {
	provisionerMu.Lock()
	defer provisionerMu.Unlock()
	provisioner = p
}

// Provision applies a benefit through the registered module. It returns
// ErrProvisionUnavailable until modules/survival has loaded.
func Provision(leaderUserID int, selector string, benefit Benefit) (ProvisionResult, error) {
	provisionerMu.RLock()
	p := provisioner
	provisionerMu.RUnlock()
	if p == nil {
		return ProvisionResult{}, ErrProvisionUnavailable
	}
	result, err := p.Provision(leaderUserID, selector, benefit)
	if err == nil {
		OnProvision.Fire(Provisioned{LeaderUserID: leaderUserID, Benefit: benefit, Result: result})
	}
	return result, err
}

// Provisioned is a successful Provision (Phase 27b).
type Provisioned struct {
	LeaderUserID int
	Benefit      Benefit
	Result       ProvisionResult
}

// OnProvision fires after each successful Provision, on the caller's
// goroutine (the game loop for eat and drink). Phase 27b: the tutorial's
// Survival lesson counts a real meal and drink.
var OnProvision util.Hook[Provisioned]

// IsMemberSelector reports whether selector names a current company member
// through the registered module. It returns false when no module is loaded, so
// command parsing preserves legacy item matching.
func IsMemberSelector(leaderUserID int, selector string) bool {
	provisionerMu.RLock()
	p := provisioner
	provisionerMu.RUnlock()
	if p == nil {
		return false
	}
	return p.IsMemberSelector(leaderUserID, selector)
}

// CompanyService is implemented by modules/survival. It applies exertion to the
// authoritative company roster and exposes read-only needs for rendering.
// modules/expedition depends on this seam so travel never imports another
// module directly.
type CompanyService interface {
	// ApplyCompanyExertion applies cost to every current company member in one
	// durable write. Costs are non-negative and at least one must be positive.
	ApplyCompanyExertion(leaderUserID int, operationID string, cost Exertion) ([]ExertionResult, error)
	// ApplyCompanyRestRecovery restores fatigue for every current company member in one durable write.
	ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]ExertionResult, error)
	// CompanyNeeds returns the leader plus current companions with their
	// stored needs. It never writes.
	CompanyNeeds(leaderUserID int) []MemberNeeds
}

var (
	companyServiceMu sync.RWMutex
	companyService   CompanyService
)

// SetCompanyService registers the active company service. Passing nil clears it.
func SetCompanyService(s CompanyService) {
	companyServiceMu.Lock()
	defer companyServiceMu.Unlock()
	companyService = s
}

// CompanyServiceAvailable reports whether a company service is registered.
// Travel checks it before starting so a route cannot begin without a way to
// accrue survival cost.
func CompanyServiceAvailable() bool {
	companyServiceMu.RLock()
	defer companyServiceMu.RUnlock()
	return companyService != nil
}

// ApplyCompanyExertion applies an exertion cost through the registered module.
// It returns ErrExertionUnavailable until modules/survival has loaded, which
// prevents a travel route from advancing without its due cost.
func ApplyCompanyExertion(leaderUserID int, operationID string, cost Exertion) ([]ExertionResult, error) {
	companyServiceMu.RLock()
	s := companyService
	companyServiceMu.RUnlock()
	if s == nil {
		return nil, ErrExertionUnavailable
	}
	return s.ApplyCompanyExertion(leaderUserID, operationID, cost)
}

// ApplyCompanyRestRecovery restores fatigue through the registered module.
func ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]ExertionResult, error) {
	companyServiceMu.RLock()
	s := companyService
	companyServiceMu.RUnlock()
	if s == nil {
		return nil, ErrRestUnavailable
	}
	return s.ApplyCompanyRestRecovery(leaderUserID, operationID, fatigue)
}

// CompanyNeeds returns the roster needs through the registered module, or nil
// when no module is loaded.
func CompanyNeeds(leaderUserID int) []MemberNeeds {
	companyServiceMu.RLock()
	s := companyService
	companyServiceMu.RUnlock()
	if s == nil {
		return nil
	}
	return s.CompanyNeeds(leaderUserID)
}

// MemberDrainService is implemented by modules/survival. It applies an
// ambient, per-tick drain (e.g. Phase 15 exposure: thirst in heat, fatigue
// in cold) to one company member. Unlike ApplyCompanyExertion it keeps no
// per-operation ledger, because ticks are frequent and a lost tick after a
// crash is harmless.
type MemberDrainService interface {
	ApplyMemberDrain(leaderUserID int, key MemberKey, cost Exertion) (ExertionResult, error)
}

var (
	memberDrainMu      sync.RWMutex
	memberDrainService MemberDrainService
)

// SetMemberDrainService registers the active drain service. nil clears it.
func SetMemberDrainService(s MemberDrainService) {
	memberDrainMu.Lock()
	defer memberDrainMu.Unlock()
	memberDrainService = s
}

// ApplyMemberDrain consults the registered drain service. It returns
// ErrExertionUnavailable until modules/survival has loaded.
func ApplyMemberDrain(leaderUserID int, key MemberKey, cost Exertion) (ExertionResult, error) {
	memberDrainMu.RLock()
	s := memberDrainService
	memberDrainMu.RUnlock()
	if s == nil {
		return ExertionResult{}, ErrExertionUnavailable
	}
	return s.ApplyMemberDrain(leaderUserID, key, cost)
}
