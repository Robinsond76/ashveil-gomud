// Package survival persists per-company-member hunger, thirst, and fatigue,
// resolves provisioning selectors against the authoritative company roster,
// and renders the read-only `survival` status command.
//
// It owns the durable registry and the command adapter. It never imports
// modules/company; the roster and lifecycle stay one-way through the
// internal/survival interface.
package survival

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	domain "github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const survivalUsage = "Usage: survival"

// Store abstracts durable registry persistence so tests can inject failures.
type Store interface {
	Load(*domain.Registry) error
	Save(domain.Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *domain.Registry) error {
	// ReadBytes discards YAML decode errors, so decode here to prevent
	// unreadable data from becoming an empty, writable registry.
	data, err := s.plug.ReadBytes("survival")
	if errors.Is(err, os.ErrNotExist) {
		*registry = *domain.NewRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeRegistry(data, registry)
}

func (s pluginStore) Save(registry domain.Registry) error {
	return s.plug.WriteStruct("survival", registry)
}

// decodeRegistry parses stored bytes, normalizing values and dropping
// malformed or unknown entries.
func decodeRegistry(data []byte, registry *domain.Registry) error {
	var wire domain.Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := domain.NewRegistry()
	for leaderUserID, byMember := range wire.Leaders {
		if leaderUserID <= 0 {
			continue
		}
		for key, needs := range byMember {
			if !domain.ValidMemberKey(key) {
				continue
			}
			if err := loaded.PutNeeds(leaderUserID, key, needs); err != nil {
				continue
			}
		}
	}
	for leaderUserID, nextID := range wire.ReservedNextCompanionIDs {
		if leaderUserID <= 0 || nextID < 1 {
			continue
		}
		if err := loaded.ReserveNextCompanionID(leaderUserID, nextID); err != nil {
			continue
		}
	}
	for leaderUserID, operations := range wire.AppliedExertion {
		if leaderUserID <= 0 {
			continue
		}
		for operationID, cost := range operations {
			if strings.TrimSpace(operationID) == "" || cost.Hunger < 0 || cost.Thirst < 0 || cost.Fatigue < 0 {
				continue
			}
			if loaded.AppliedExertion[leaderUserID] == nil {
				loaded.AppliedExertion[leaderUserID] = map[string]domain.Exertion{}
			}
			loaded.AppliedExertion[leaderUserID][operationID] = cost
		}
	}
	for leaderUserID, operations := range wire.AppliedRestOperation {
		if leaderUserID <= 0 {
			continue
		}
		for operationID, fatigue := range operations {
			if strings.TrimSpace(operationID) == "" || fatigue <= 0 {
				continue
			}
			if loaded.AppliedRestOperation[leaderUserID] == nil {
				loaded.AppliedRestOperation[leaderUserID] = map[string]int{}
			}
			loaded.AppliedRestOperation[leaderUserID][operationID] = fatigue
		}
	}
	*registry = *loaded
	return nil
}

// SurvivalModule owns the durable survival registry for one plugin instance.
type SurvivalModule struct {
	plug     *plugins.Plugin
	store    Store
	registry domain.Registry
	loadErr  error

	// mu guards registry and dirty. Survival is called from the game loop
	// and from travel/camping timer goroutines, so every exported entry
	// point takes it. It is a leaf lock: nothing called while holding it
	// calls back into survival.
	mu sync.Mutex
	// dirty marks unsaved ambient drains (ApplyMemberDrain), flushed by the
	// next save of any kind, including the periodic plugin save.
	dirty bool
}

var (
	_ domain.Provisioner    = (*SurvivalModule)(nil)
	_ domain.Lifecycle      = (*SurvivalModule)(nil)
	_ domain.CompanyService = (*SurvivalModule)(nil)
)

func init() {
	m := &SurvivalModule{plug: plugins.New("survival", "1.0")}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("survival", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.flush(); err != nil {
			mudlog.Error("survival: save", "error", err)
		}
	})
	domain.SetProvisioner(m)
	domain.SetLifecycle(m)
	domain.SetCompanyService(m)
	domain.SetMemberDrainService(m)
}

func (m *SurvivalModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("survival: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("survival: persistence unavailable")
	}
	return nil
}

func (m *SurvivalModule) save() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.store.Save(m.registry); err != nil {
		return fmt.Errorf("survival: save failed; please retry: %w", err)
	}
	m.dirty = false
	return nil
}

// flush takes the lock and persists the registry (the periodic plugin save).
func (m *SurvivalModule) flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.save()
}

func (m *SurvivalModule) load() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store == nil {
		return
	}
	loaded := domain.NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("survival: load", "error", err)
		return
	}
	m.registry = *loaded
	m.loadErr = nil
}

// EnsureCompanyMember creates default state for a newly summoned companion and
// persists it. It rolls the in-memory registry back when the write fails.
func (m *SurvivalModule) EnsureCompanyMember(leaderUserID, companionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if leaderUserID <= 0 || companionID <= 0 {
		return domain.ErrInvalidMember
	}
	snapshot := m.registry.Clone()
	if err := m.registry.ReserveNextCompanionID(leaderUserID, companionID+1); err != nil {
		return err
	}
	if err := m.registry.Ensure(leaderUserID, domain.CompanionMemberKey(companionID)); err != nil {
		return err
	}
	if err := m.save(); err != nil {
		m.registry = snapshot
		return err
	}
	return nil
}

// NextReservedCompanionID returns the survival-side durable reservation used
// by company before assigning a new companion ID.
func (m *SurvivalModule) NextReservedCompanionID(leaderUserID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return 0, err
	}
	return m.registry.NextReservedCompanionID(leaderUserID)
}

// RemoveCompanyMember prunes one dismissed companion and persists the change.
func (m *SurvivalModule) RemoveCompanyMember(leaderUserID, companionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if leaderUserID <= 0 || companionID <= 0 {
		return domain.ErrInvalidMember
	}
	snapshot := m.registry.Clone()
	m.registry.Remove(leaderUserID, domain.CompanionMemberKey(companionID))
	if err := m.save(); err != nil {
		m.registry = snapshot
		return err
	}
	return nil
}

// RemoveAllCompanyMembers prunes every companion while preserving leader state.
func (m *SurvivalModule) RemoveAllCompanyMembers(leaderUserID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if leaderUserID <= 0 {
		return domain.ErrInvalidMember
	}
	snapshot := m.registry.Clone()
	for _, key := range m.registry.Members(leaderUserID) {
		if key == domain.LeaderMemberKey {
			continue
		}
		m.registry.Remove(leaderUserID, key)
	}
	if err := m.save(); err != nil {
		m.registry = snapshot
		return err
	}
	return nil
}

// SnapshotCompanyMember returns the exact stored state for a companion without
// mutating the registry. A companion with no record reports Exists false.
func (m *SurvivalModule) SnapshotCompanyMember(leaderUserID, companionID int) (domain.MemberSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return domain.MemberSnapshot{}, err
	}
	if leaderUserID <= 0 || companionID <= 0 {
		return domain.MemberSnapshot{}, domain.ErrInvalidMember
	}
	needs, ok := m.registry.NeedsFor(leaderUserID, domain.CompanionMemberKey(companionID))
	if !ok {
		return domain.MemberSnapshot{}, nil
	}
	return domain.MemberSnapshot{Exists: true, Needs: needs}, nil
}

// RestoreCompanyMember writes a captured snapshot back and persists it. When
// the snapshot recorded no stored state, the key is removed instead.
func (m *SurvivalModule) RestoreCompanyMember(leaderUserID, companionID int, snapshot domain.MemberSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if leaderUserID <= 0 || companionID <= 0 {
		return domain.ErrInvalidMember
	}
	key := domain.CompanionMemberKey(companionID)
	before := m.registry.Clone()
	if snapshot.Exists {
		if err := m.registry.PutNeeds(leaderUserID, key, snapshot.Needs); err != nil {
			return err
		}
	} else {
		m.registry.Remove(leaderUserID, key)
	}
	if err := m.save(); err != nil {
		m.registry = before
		return err
	}
	return nil
}

// ApplyCompanyExertion applies cost to every current company member in one
// durable write. It rolls the in-memory registry back when the write fails, so
// travel never advances without its due exertion.
func (m *SurvivalModule) ApplyCompanyExertion(leaderUserID int, operationID string, cost domain.Exertion) ([]domain.ExertionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	if leaderUserID <= 0 {
		return nil, domain.ErrInvalidMember
	}
	if strings.TrimSpace(operationID) == "" {
		return nil, domain.ErrInvalidAmount
	}
	if cost.Hunger < 0 || cost.Thirst < 0 || cost.Fatigue < 0 ||
		(cost.Hunger == 0 && cost.Thirst == 0 && cost.Fatigue == 0) {
		return nil, domain.ErrInvalidAmount
	}
	if prior, ok := m.registry.AppliedExertion[leaderUserID][operationID]; ok {
		if prior != cost {
			return nil, domain.ErrExertionConflict
		}
		return m.companyExertionResults(leaderUserID), nil
	}
	refs := m.memberRefs(leaderUserID)
	if len(refs) == 0 {
		refs = []domain.MemberRef{{Key: domain.LeaderMemberKey, Name: m.leaderName(leaderUserID)}}
	}
	snapshot := m.registry.Clone()
	results := make([]domain.ExertionResult, 0, len(refs))
	for _, ref := range refs {
		if err := m.registry.Ensure(leaderUserID, ref.Key); err != nil {
			m.registry = snapshot
			return nil, err
		}
		hunger, thirst, fatigue, err := m.registry.ApplyExertion(leaderUserID, ref.Key, cost)
		if err != nil {
			m.registry = snapshot
			return nil, err
		}
		results = append(results, domain.ExertionResult{
			Member:  ref.Key,
			Name:    ref.Name,
			Needs:   m.registry.MustNeedsFor(leaderUserID, ref.Key),
			Hunger:  hunger,
			Thirst:  thirst,
			Fatigue: fatigue,
		})
	}
	if m.registry.AppliedExertion[leaderUserID] == nil {
		m.registry.AppliedExertion[leaderUserID] = map[string]domain.Exertion{}
	}
	m.registry.AppliedExertion[leaderUserID][operationID] = cost
	if err := m.save(); err != nil {
		m.registry = snapshot
		return nil, err
	}
	return results, nil
}

// ApplyMemberDrain applies an ambient drain to one member. It keeps no
// operation ledger and does not write to disk itself: it marks the registry
// dirty, and the next save of any kind (including the periodic plugin save)
// persists it. Frequent ticks (Phase 15 exposure) would otherwise rewrite
// the whole registry many times a minute; losing the drains since the last
// save in a crash is harmless.
func (m *SurvivalModule) ApplyMemberDrain(leaderUserID int, key domain.MemberKey, cost domain.Exertion) (domain.ExertionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return domain.ExertionResult{}, err
	}
	if leaderUserID <= 0 || !domain.ValidMemberKey(key) {
		return domain.ExertionResult{}, domain.ErrInvalidMember
	}
	if cost.Hunger < 0 || cost.Thirst < 0 || cost.Fatigue < 0 ||
		(cost.Hunger == 0 && cost.Thirst == 0 && cost.Fatigue == 0) {
		return domain.ExertionResult{}, domain.ErrInvalidAmount
	}
	snapshot := m.registry.Clone()
	if err := m.registry.Ensure(leaderUserID, key); err != nil {
		m.registry = snapshot
		return domain.ExertionResult{}, err
	}
	hunger, thirst, fatigue, err := m.registry.ApplyExertion(leaderUserID, key, cost)
	if err != nil {
		m.registry = snapshot
		return domain.ExertionResult{}, err
	}
	m.dirty = true
	return domain.ExertionResult{
		Member:  key,
		Needs:   m.registry.MustNeedsFor(leaderUserID, key),
		Hunger:  hunger,
		Thirst:  thirst,
		Fatigue: fatigue,
	}, nil
}

// ApplyCompanyRestRecovery restores fatigue to every current company member
// in one durable write. A leader-scoped operation ledger makes completion
// retries idempotent across restarts and copyover.
func (m *SurvivalModule) ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]domain.ExertionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return nil, err
	}
	if leaderUserID <= 0 {
		return nil, domain.ErrInvalidMember
	}
	if strings.TrimSpace(operationID) == "" || fatigue <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if prior, ok := m.registry.AppliedRestOperation[leaderUserID][operationID]; ok {
		if prior != fatigue {
			return nil, domain.ErrRestConflict
		}
		return m.companyRestResults(leaderUserID)
	}
	refs := m.memberRefs(leaderUserID)
	if len(refs) == 0 {
		refs = []domain.MemberRef{{Key: domain.LeaderMemberKey, Name: m.leaderName(leaderUserID)}}
	}
	snapshot := m.registry.Clone()
	results := make([]domain.ExertionResult, 0, len(refs))
	for _, ref := range refs {
		if err := m.registry.Ensure(leaderUserID, ref.Key); err != nil {
			m.registry = snapshot
			return nil, err
		}
		change, err := m.registry.ApplyRestRecovery(leaderUserID, ref.Key, fatigue)
		if err != nil {
			m.registry = snapshot
			return nil, err
		}
		results = append(results, domain.ExertionResult{Member: ref.Key, Name: ref.Name, Needs: m.registry.MustNeedsFor(leaderUserID, ref.Key), Fatigue: change})
	}
	if m.registry.AppliedRestOperation[leaderUserID] == nil {
		m.registry.AppliedRestOperation[leaderUserID] = map[string]int{}
	}
	m.registry.AppliedRestOperation[leaderUserID][operationID] = fatigue
	if err := m.save(); err != nil {
		m.registry = snapshot
		return nil, err
	}
	return results, nil
}

func (m *SurvivalModule) companyRestResults(leaderUserID int) ([]domain.ExertionResult, error) {
	refs := m.memberRefs(leaderUserID)
	if len(refs) == 0 {
		refs = []domain.MemberRef{{Key: domain.LeaderMemberKey, Name: m.leaderName(leaderUserID)}}
	}
	out := make([]domain.ExertionResult, 0, len(refs))
	for _, ref := range refs {
		needs, ok := m.registry.NeedsFor(leaderUserID, ref.Key)
		if !ok {
			needs = domain.FullNeeds()
		}
		out = append(out, domain.ExertionResult{Member: ref.Key, Name: ref.Name, Needs: needs})
	}
	return out, nil
}

func (m *SurvivalModule) companyExertionResults(leaderUserID int) []domain.ExertionResult {
	refs := m.memberRefs(leaderUserID)
	if len(refs) == 0 {
		refs = []domain.MemberRef{{Key: domain.LeaderMemberKey, Name: m.leaderName(leaderUserID)}}
	}
	out := make([]domain.ExertionResult, 0, len(refs))
	for _, ref := range refs {
		_ = m.registry.Ensure(leaderUserID, ref.Key)
		out = append(out, domain.ExertionResult{Member: ref.Key, Name: ref.Name, Needs: m.registry.MustNeedsFor(leaderUserID, ref.Key)})
	}
	return out
}

// CompanyNeeds returns the leader plus current companions with their stored
// needs. It is read-only: it may initialize a default leader record and prune
// stale companions in memory, but never writes to the durable store.
func (m *SurvivalModule) CompanyNeeds(leaderUserID int) []domain.MemberNeeds {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return nil
	}
	_ = m.registry.Ensure(leaderUserID, domain.LeaderMemberKey)
	m.pruneStale(leaderUserID)
	refs := m.memberRefs(leaderUserID)
	needs := make([]domain.MemberNeeds, 0, len(refs))
	for _, ref := range refs {
		stored, ok := m.registry.NeedsFor(leaderUserID, ref.Key)
		if !ok {
			stored = domain.FullNeeds()
		}
		needs = append(needs, domain.MemberNeeds{Key: ref.Key, Name: ref.Name, Needs: stored})
	}
	return needs
}

// ReconcileCompanyRosters aligns the persisted registry with the loaded
// company rosters in at most one write: companion keys absent from a roster
// are pruned and current companions without a record start at FullNeeds. The
// leader key is always preserved.
func (m *SurvivalModule) ReconcileCompanyRosters(rosters map[int][]domain.MemberRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	leaders := map[int]struct{}{}
	for leaderUserID := range rosters {
		if leaderUserID > 0 {
			leaders[leaderUserID] = struct{}{}
		}
	}
	for leaderUserID := range m.registry.Leaders {
		if leaderUserID > 0 {
			leaders[leaderUserID] = struct{}{}
		}
	}

	before := m.registry.Clone()
	changed := false
	for leaderUserID := range leaders {
		valid := map[domain.MemberKey]bool{domain.LeaderMemberKey: true}
		for _, ref := range rosters[leaderUserID] {
			if ref.Key == domain.LeaderMemberKey || !domain.ValidMemberKey(ref.Key) {
				continue
			}
			valid[ref.Key] = true
			if _, ok := m.registry.NeedsFor(leaderUserID, ref.Key); !ok {
				if err := m.registry.Ensure(leaderUserID, ref.Key); err != nil {
					m.registry = before
					return err
				}
				changed = true
			}
		}
		for _, key := range m.registry.Members(leaderUserID) {
			if !valid[key] {
				m.registry.Remove(leaderUserID, key)
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	if err := m.save(); err != nil {
		m.registry = before
		return err
	}
	return nil
}

// Provision implements domain.Provisioner. It resolves the target member,
// applies the benefit, and persists before reporting success.
func (m *SurvivalModule) Provision(leaderUserID int, selector string, benefit domain.Benefit) (domain.ProvisionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return domain.ProvisionResult{}, err
	}
	if benefit.Nutrition < 0 || benefit.Hydration < 0 || benefit.Fatigue < 0 ||
		(benefit.Nutrition == 0 && benefit.Hydration == 0 && benefit.Fatigue == 0) {
		return domain.ProvisionResult{}, domain.ErrInvalidAmount
	}
	key, name, err := m.resolveMember(leaderUserID, selector)
	if err != nil {
		return domain.ProvisionResult{}, err
	}
	snapshot := m.registry.Clone()
	if err := m.registry.Ensure(leaderUserID, key); err != nil {
		return domain.ProvisionResult{}, err
	}
	result, err := m.applyBenefit(leaderUserID, key, name, benefit)
	if err != nil {
		m.registry = snapshot
		return domain.ProvisionResult{}, err
	}
	if err := m.save(); err != nil {
		m.registry = snapshot
		return domain.ProvisionResult{}, err
	}
	return result, nil
}

func (m *SurvivalModule) applyBenefit(leaderUserID int, key domain.MemberKey, name string, benefit domain.Benefit) (domain.ProvisionResult, error) {
	result := domain.ProvisionResult{Member: key, Name: name}
	if benefit.Nutrition > 0 {
		hunger, thirst, err := m.registry.ConsumeFood(leaderUserID, key, benefit.Nutrition, benefit.Hydration)
		if err != nil {
			return domain.ProvisionResult{}, err
		}
		result.Hunger, result.Thirst = hunger, thirst
	} else if benefit.Hydration > 0 {
		thirst, err := m.registry.ConsumeWater(leaderUserID, key, benefit.Hydration)
		if err != nil {
			return domain.ProvisionResult{}, err
		}
		result.Thirst = thirst
	}
	if benefit.Fatigue > 0 {
		fatigue, err := m.registry.ApplyRestRecovery(leaderUserID, key, benefit.Fatigue)
		if err != nil {
			return domain.ProvisionResult{}, err
		}
		result.Fatigue = fatigue
	}
	result.Needs = m.registry.MustNeedsFor(leaderUserID, key)
	return result, nil
}

// resolveMember maps a selector to a current member. The leader is always
// valid; companions are valid only while the authoritative roster holds them.
// Persisted survival state never authorizes a target, so a stale record cannot
// name a dismissed companion and a current companion with no record can still
// be targeted.
func (m *SurvivalModule) resolveMember(leaderUserID int, selector string) (domain.MemberKey, string, error) {
	if leaderUserID <= 0 {
		return "", "", domain.ErrInvalidMember
	}
	s := strings.ToLower(strings.TrimSpace(selector))
	if s == "" || s == "leader" || s == "me" || s == "self" {
		return domain.LeaderMemberKey, m.leaderName(leaderUserID), nil
	}
	if id, ok := parseCompanionSelector(s); ok {
		key := domain.CompanionMemberKey(id)
		ref, ok := currentRosterMember(leaderUserID, key)
		if !ok {
			return "", "", domain.ErrUnknownMember
		}
		name := ref.Name
		if name == "" {
			name = m.companionName(leaderUserID, id)
		}
		return key, name, nil
	}
	return matchCompanionName(domain.CurrentRoster(leaderUserID), s)
}

// IsMemberSelector reports whether selector names the leader or a current
// companion, using the same rules as Provision so command parsing never
// forwards a token that provisioning would reject.
func (m *SurvivalModule) IsMemberSelector(leaderUserID int, selector string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if leaderUserID <= 0 {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(selector))
	if s == "" {
		return false
	}
	if s == "leader" || s == "me" || s == "self" {
		return true
	}
	if id, ok := parseCompanionSelector(s); ok {
		_, ok := currentRosterMember(leaderUserID, domain.CompanionMemberKey(id))
		return ok
	}
	_, _, err := matchCompanionName(domain.CurrentRoster(leaderUserID), s)
	return err == nil
}

// currentRosterMember resolves a member key against the authoritative company
// roster. It is the single authorization source for companion selectors, so
// persisted survival records can never authorize a dismissed companion.
func currentRosterMember(leaderUserID int, key domain.MemberKey) (domain.MemberRef, bool) {
	for _, ref := range domain.CurrentRoster(leaderUserID) {
		if ref.Key == key {
			return ref, true
		}
	}
	return domain.MemberRef{}, false
}

func parseCompanionSelector(selector string) (int, bool) {
	id, err := strconv.Atoi(strings.TrimPrefix(selector, "#"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// matchCompanionName matches an exact name, then a single unambiguous
// substring, against the current companion roster.
func matchCompanionName(roster []domain.MemberRef, selector string) (domain.MemberKey, string, error) {
	var partial []domain.MemberRef
	for _, ref := range roster {
		if ref.Key == domain.LeaderMemberKey {
			continue
		}
		if strings.EqualFold(ref.Name, selector) {
			return ref.Key, ref.Name, nil
		}
		if ref.Name != "" && strings.Contains(strings.ToLower(ref.Name), selector) {
			partial = append(partial, ref)
		}
	}
	if len(partial) == 1 {
		return partial[0].Key, partial[0].Name, nil
	}
	if len(partial) > 1 {
		return "", "", domain.ErrAmbiguousMember
	}
	return "", "", domain.ErrUnknownMember
}

func (m *SurvivalModule) leaderName(leaderUserID int) string {
	if user := users.GetByUserId(leaderUserID); user != nil && user.Character != nil && user.Character.Name != "" {
		return user.Character.Name
	}
	return "leader"
}

func (m *SurvivalModule) companionName(leaderUserID, companionID int) string {
	key := domain.CompanionMemberKey(companionID)
	for _, ref := range domain.CurrentRoster(leaderUserID) {
		if ref.Key == key && ref.Name != "" {
			return ref.Name
		}
	}
	return fmt.Sprintf("#%d", companionID)
}

// status renders the leader plus the current roster. It is read-only: it may
// create a default leader record and prune stale companions in memory, but it
// never writes to the durable store.
func (m *SurvivalModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	_ = m.registry.Ensure(leaderUserID, domain.LeaderMemberKey)
	m.pruneStale(leaderUserID)

	refs := m.memberRefs(leaderUserID)
	lines := []string{"Company survival:"}
	for _, ref := range refs {
		needs, ok := m.registry.NeedsFor(leaderUserID, ref.Key)
		if !ok {
			needs = domain.FullNeeds()
		}
		lines = append(lines, fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			ref.Name,
			needs.Hunger, domain.HungerLabel(needs.Hunger),
			needs.Thirst, domain.ThirstLabel(needs.Thirst),
			needs.Fatigue, domain.FatigueLabel(needs.Fatigue)))
	}
	return strings.Join(lines, "\n")
}

// memberRefs lists the authoritative roster when a provider is registered,
// falling back to the registry projection otherwise. Roster members without a
// stored record still render at full default needs.
func (m *SurvivalModule) memberRefs(leaderUserID int) []domain.MemberRef {
	keys := []domain.MemberKey{}
	names := map[domain.MemberKey]string{}
	if roster := domain.CurrentRoster(leaderUserID); len(roster) > 0 {
		for _, ref := range roster {
			keys = append(keys, ref.Key)
			names[ref.Key] = ref.Name
		}
	} else {
		keys = m.registry.Members(leaderUserID)
	}
	refs := make([]domain.MemberRef, 0, len(keys))
	for _, key := range keys {
		name := names[key]
		if name == "" {
			if key == domain.LeaderMemberKey {
				name = m.leaderName(leaderUserID)
			} else {
				name = m.displayName(key)
			}
		}
		refs = append(refs, domain.MemberRef{Key: key, Name: name})
	}
	return refs
}

func (m *SurvivalModule) displayName(key domain.MemberKey) string {
	if key == domain.LeaderMemberKey {
		return "leader"
	}
	if id, ok := parseCompanionSelector(strings.TrimPrefix(string(key), "companion:")); ok {
		return fmt.Sprintf("#%d", id)
	}
	return string(key)
}

// pruneStale removes companion records no longer present in the authoritative
// roster. It is skipped when no roster provider is registered.
func (m *SurvivalModule) pruneStale(leaderUserID int) {
	roster := domain.CurrentRoster(leaderUserID)
	if len(roster) == 0 {
		return
	}
	valid := map[domain.MemberKey]bool{}
	for _, ref := range roster {
		valid[ref.Key] = true
	}
	for _, key := range m.registry.Members(leaderUserID) {
		if key == domain.LeaderMemberKey {
			continue
		}
		if !valid[key] {
			m.registry.Remove(leaderUserID, key)
		}
	}
}

func (m *SurvivalModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rest = strings.ToLower(strings.TrimSpace(rest))
	if rest != "" && rest != "status" {
		user.SendText(survivalUsage)
		return true, nil
	}
	user.SendText(m.status(user.UserId))
	return true, nil
}
