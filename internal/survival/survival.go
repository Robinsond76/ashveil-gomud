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

// Errors returned by the survival domain and provider seam.
var (
	ErrInvalidMember          = errors.New("survival: invalid company member")
	ErrUnknownMember          = errors.New("survival: unknown company member")
	ErrAmbiguousMember        = errors.New("survival: ambiguous company member")
	ErrInvalidAmount          = errors.New("survival: amount must be positive")
	ErrPersistenceUnavailable = errors.New("survival: persistence unavailable")
	ErrProvisionUnavailable   = errors.New("survival: provisioning unavailable")
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
	Leaders map[int]map[MemberKey]Needs `yaml:"leaders"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Leaders: map[int]map[MemberKey]Needs{}}
}

func (r *Registry) ensureMaps() {
	if r.Leaders == nil {
		r.Leaders = map[int]map[MemberKey]Needs{}
	}
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
	out := Registry{Leaders: map[int]map[MemberKey]Needs{}}
	for leaderUserID, byMember := range r.Leaders {
		clone := make(map[MemberKey]Needs, len(byMember))
		for key, needs := range byMember {
			clone[key] = needs
		}
		out.Leaders[leaderUserID] = clone
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
	return p.Provision(leaderUserID, selector, benefit)
}

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
