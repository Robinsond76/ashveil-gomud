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
	*registry = *loaded
	return nil
}

// SurvivalModule owns the durable survival registry for one plugin instance.
type SurvivalModule struct {
	plug     *plugins.Plugin
	store    Store
	registry domain.Registry
	loadErr  error
}

func init() {
	m := &SurvivalModule{plug: plugins.New("survival", "1.0")}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("survival", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("survival: save", "error", err)
		}
	})
	domain.SetProvisioner(m)
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
	return nil
}

func (m *SurvivalModule) load() {
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
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if leaderUserID <= 0 || companionID <= 0 {
		return domain.ErrInvalidMember
	}
	snapshot := m.registry.Clone()
	if err := m.registry.Ensure(leaderUserID, domain.CompanionMemberKey(companionID)); err != nil {
		return err
	}
	if err := m.save(); err != nil {
		m.registry = snapshot
		return err
	}
	return nil
}

// RemoveCompanyMember prunes one dismissed companion and persists the change.
func (m *SurvivalModule) RemoveCompanyMember(leaderUserID, companionID int) error {
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

// Provision implements domain.Provisioner. It resolves the target member,
// applies the benefit, and persists before reporting success.
func (m *SurvivalModule) Provision(leaderUserID int, selector string, benefit domain.Benefit) (domain.ProvisionResult, error) {
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
// valid; companions are valid only while the roster projection holds them.
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
		if !m.hasMember(leaderUserID, key) {
			return "", "", domain.ErrUnknownMember
		}
		return key, m.companionName(leaderUserID, id), nil
	}
	key, name, err := matchCompanionName(domain.CurrentRoster(leaderUserID), s)
	if err != nil {
		return "", "", err
	}
	if !m.hasMember(leaderUserID, key) {
		return "", "", domain.ErrUnknownMember
	}
	return key, name, nil
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

func (m *SurvivalModule) hasMember(leaderUserID int, key domain.MemberKey) bool {
	_, ok := m.registry.NeedsFor(leaderUserID, key)
	return ok
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
		needs, _ := m.registry.NeedsFor(leaderUserID, ref.Key)
		lines = append(lines, fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			ref.Name,
			needs.Hunger, domain.HungerLabel(needs.Hunger),
			needs.Thirst, domain.ThirstLabel(needs.Thirst),
			needs.Fatigue, domain.FatigueLabel(needs.Fatigue)))
	}
	return strings.Join(lines, "\n")
}

func (m *SurvivalModule) memberRefs(leaderUserID int) []domain.MemberRef {
	names := map[domain.MemberKey]string{domain.LeaderMemberKey: m.leaderName(leaderUserID)}
	for _, ref := range domain.CurrentRoster(leaderUserID) {
		if ref.Name != "" {
			names[ref.Key] = ref.Name
		}
	}
	keys := m.registry.Members(leaderUserID)
	refs := make([]domain.MemberRef, 0, len(keys))
	for _, key := range keys {
		name := names[key]
		if name == "" {
			name = m.displayName(key)
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
	rest = strings.ToLower(strings.TrimSpace(rest))
	if rest != "" && rest != "status" {
		user.SendText(survivalUsage)
		return true, nil
	}
	user.SendText(m.status(user.UserId))
	return true, nil
}
