// Package strategy is the Phase 32d strategy module: it stores each
// player's battle strategies (a role and a target rule for themselves and
// each companion), answers internal/strategy.For for the combat round, and
// provides the "strategy" command. Strategies are durable: they survive
// restart and copyover in the plugin's own file.
package strategy

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/spells"
	domain "github.com/GoMudEngine/GoMud/internal/strategy"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

// Registry is every player's stored strategies: user id -> member key
// ("leader", "companion:<id>") -> the settings that differ from default.
type Registry struct {
	Players map[int]map[string]domain.Strategy `yaml:"players,omitempty"`
	// Tactics is each player's company tactics (Phase 30c) that differ
	// from the defaults.
	Tactics map[int]domain.Tactics `yaml:"tactics,omitempty"`
}

// NewRegistry is an empty registry.
func NewRegistry() *Registry {
	return &Registry{Players: map[int]map[string]domain.Strategy{}, Tactics: map[int]domain.Tactics{}}
}

// Clone is a deep copy.
func (r Registry) Clone() Registry {
	out := Registry{Players: make(map[int]map[string]domain.Strategy, len(r.Players))}
	for id, members := range r.Players {
		m := make(map[string]domain.Strategy, len(members))
		for k, v := range members {
			m[k] = v
		}
		out.Players[id] = m
	}
	out.Tactics = make(map[int]domain.Tactics, len(r.Tactics))
	for id, t := range r.Tactics {
		out.Tactics[id] = t
	}
	return out
}

// Store persists the registry.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *Registry) error {
	data, err := s.plug.ReadBytes("strategy")
	if errors.Is(err, os.ErrNotExist) {
		*registry = *NewRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeRegistry(data, registry)
}

func (s pluginStore) Save(registry Registry) error {
	return s.plug.WriteStruct("strategy", registry)
}

// decodeRegistry parses stored bytes, dropping entries with a bad user id,
// a blank member key, or a role or rule it doesn't know (logged), so one
// bad line never loses the rest.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for id, members := range wire.Players {
		if id <= 0 {
			continue
		}
		for key, s := range members {
			clean, ok := cleanStrategy(s)
			if strings.TrimSpace(key) == "" || !ok {
				mudlog.Warn("strategy: dropped a stored strategy", "user", id, "member", key, "role", s.Role, "rule", s.Rule)
				continue
			}
			if clean.IsZero() {
				continue
			}
			if loaded.Players[id] == nil {
				loaded.Players[id] = map[string]domain.Strategy{}
			}
			loaded.Players[id][key] = clean
		}
	}
	for id, t := range wire.Tactics {
		clean, ok := cleanTactics(t)
		if id <= 0 || !ok {
			mudlog.Warn("strategy: dropped stored tactics", "user", id, "focus", t.Focus, "healing", t.Healing)
			continue
		}
		if !clean.IsZero() {
			loaded.Tactics[id] = clean
		}
	}
	*registry = *loaded
	return nil
}

func cleanTactics(t domain.Tactics) (domain.Tactics, bool) {
	var out domain.Tactics
	if t.Focus != "" {
		f, ok := domain.ParseFocus(string(t.Focus))
		if !ok {
			return out, false
		}
		out.Focus = f
	}
	if t.Healing != 0 {
		h, ok := domain.ParseHealing(fmt.Sprint(t.Healing))
		if !ok {
			return out, false
		}
		out.Healing = h
	}
	return out, true
}

func cleanStrategy(s domain.Strategy) (domain.Strategy, bool) {
	var out domain.Strategy
	if s.Role != "" {
		r, ok := domain.ParseRole(string(s.Role))
		if !ok {
			return out, false
		}
		out.Role = r
	}
	if s.Rule != "" {
		r, ok := domain.ParseRule(string(s.Rule))
		if !ok {
			return out, false
		}
		out.Rule = r
	}
	return out, true
}

// StrategyModule owns the strategies.
type StrategyModule struct {
	plug  *plugins.Plugin
	store Store

	// mu guards registry, loadErr, and autoSpells. It is a leaf lock:
	// never call the engine while holding it.
	mu         sync.Mutex
	registry   *Registry
	loadErr    error
	autoSpells []domain.Spell

	// Engine seams (native by default; tests replace them).
	env env
}

// module is the registered instance, for wiring tests.
var module *StrategyModule

func newModule() *StrategyModule {
	return &StrategyModule{registry: NewRegistry(), autoSpells: domain.DefaultAutoSpells(), env: nativeEnv()}
}

func init() {
	m := newModule()
	m.plug = plugins.New("strategy", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("strategy", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("strategy: save", "error", err)
		}
	})
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	domain.SetProvider(m)
	domain.SetTacticsProvider(m)
	module = m
}

// load reads the automatic spells from config and the durable registry.
func (m *StrategyModule) load() {
	var raw any
	if m.plug != nil {
		raw = m.plug.Config.Get("AutoSpells")
	}
	autoSpells := parseAutoSpells(raw, spellExists)
	registry := NewRegistry()
	var err error
	if m.store != nil {
		err = m.store.Load(registry)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(autoSpells) > 0 {
		m.autoSpells = autoSpells
	}
	if err != nil {
		mudlog.Error("strategy: load", "error", err)
		m.loadErr = err
		return
	}
	m.loadErr = nil
	m.registry = registry
}

func spellExists(id string) bool { return spells.GetSpell(id) != nil }

// parseAutoSpells reads AutoSpells ([{Spell, Use}]), dropping (and
// logging) unknown spells and uses.
func parseAutoSpells(raw any, exists func(string) bool) []domain.Spell {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []domain.Spell
	for _, entry := range list {
		fields := lowerKeys(entry)
		if fields == nil {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(fmt.Sprint(fields["spell"])))
		use, ok := domain.ParseUse(fmt.Sprint(fields["use"]))
		if !ok || id == "" || (exists != nil && !exists(id)) {
			mudlog.Warn("strategy: dropped an automatic spell", "spell", fields["spell"], "use", fields["use"])
			continue
		}
		out = append(out, domain.Spell{ID: id, Use: use})
	}
	return out
}

func lowerKeys(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[strings.ToLower(k)] = v
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			if name, ok := k.(string); ok {
				out[strings.ToLower(name)] = v
			}
		}
		return out
	}
	return nil
}

func (m *StrategyModule) persistenceAvailableLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("strategies can't be changed until they load again: %w", m.loadErr)
	}
	if m.store == nil {
		return errors.New("strategies can't be saved right now")
	}
	return nil
}

func (m *StrategyModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailableLocked(); err != nil {
		return err
	}
	return m.store.Save(m.registry.Clone())
}

// Stored implements domain.Provider.
func (m *StrategyModule) Stored(userID int, key string) domain.Strategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Players[userID][key]
}

// AutoSpells implements domain.Provider.
func (m *StrategyModule) AutoSpells() []domain.Spell {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Spell(nil), m.autoSpells...)
}

// StoredTactics implements domain.TacticsProvider.
func (m *StrategyModule) StoredTactics(userID int) domain.Tactics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Tactics[userID]
}

// SetTactics implements domain.TacticsProvider: it stores the player's
// tactics (the defaults clear them) and saves, rolling back if the save
// fails.
func (m *StrategyModule) SetTactics(userID int, t domain.Tactics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailableLocked(); err != nil {
		return err
	}
	before, had := m.registry.Tactics[userID]
	m.putTactics(userID, t)
	if err := m.store.Save(m.registry.Clone()); err != nil {
		if had {
			m.registry.Tactics[userID] = before
		} else {
			delete(m.registry.Tactics, userID)
		}
		return fmt.Errorf("the tactics couldn't be saved; please try again: %w", err)
	}
	return nil
}

func (m *StrategyModule) putTactics(userID int, t domain.Tactics) {
	if m.registry.Tactics == nil {
		m.registry.Tactics = map[int]domain.Tactics{}
	}
	if t.IsZero() {
		delete(m.registry.Tactics, userID)
		return
	}
	m.registry.Tactics[userID] = t
}

// set stores a member's strategy (the zero value clears it) and saves,
// rolling back if the save fails.
func (m *StrategyModule) set(userID int, key string, s domain.Strategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailableLocked(); err != nil {
		return err
	}
	before, had := m.registry.Players[userID][key]
	m.put(userID, key, s)
	if err := m.store.Save(m.registry.Clone()); err != nil {
		if had {
			m.put(userID, key, before)
		} else {
			m.put(userID, key, domain.Strategy{})
		}
		return fmt.Errorf("the strategy couldn't be saved; please try again: %w", err)
	}
	return nil
}

func (m *StrategyModule) put(userID int, key string, s domain.Strategy) {
	if s.IsZero() {
		delete(m.registry.Players[userID], key)
		if len(m.registry.Players[userID]) == 0 {
			delete(m.registry.Players, userID)
		}
		return
	}
	if m.registry.Players[userID] == nil {
		m.registry.Players[userID] = map[string]domain.Strategy{}
	}
	m.registry.Players[userID][key] = s
}

// prune drops stored strategies for companions the player no longer has
// (dismissed, deserted, lost), and saves if anything changed.
func (m *StrategyModule) prune(userID int, keep map[string]bool) {
	m.mu.Lock()
	changed := false
	for key := range m.registry.Players[userID] {
		if !keep[key] {
			delete(m.registry.Players[userID], key)
			changed = true
		}
	}
	if changed && len(m.registry.Players[userID]) == 0 {
		delete(m.registry.Players, userID)
	}
	m.mu.Unlock()
	if changed {
		if err := m.save(); err != nil {
			mudlog.Error("strategy: save after prune", "user", userID, "error", err)
		}
	}
}

// onUserPurged (32b's purge) forgets a purged user's strategies.
func (m *StrategyModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	_, had := m.registry.Players[evt.UserId]
	_, hadTactics := m.registry.Tactics[evt.UserId]
	had = had || hadTactics
	delete(m.registry.Players, evt.UserId)
	delete(m.registry.Tactics, evt.UserId)
	m.mu.Unlock()
	if had {
		if err := m.save(); err != nil {
			mudlog.Error("strategy: save after purge", "user", evt.UserId, "error", err)
		}
	}
	return events.Continue
}
