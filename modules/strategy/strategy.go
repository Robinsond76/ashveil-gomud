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
	"github.com/GoMudEngine/GoMud/internal/modconfig"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/modstore"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/orders"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/stance"
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
	// Orders is each player's battle orders (Phase 61): user id -> member
	// key -> up to orders.MaxOrders rules, in the order they are read.
	Orders map[int]map[string][]orders.Order `yaml:"orders,omitempty"`
	// Stances is each player's weapon stances (Phase 69): user id -> member
	// key -> the stance chosen (kept even while the member lacks its weapon).
	Stances map[int]map[string]stance.Stance `yaml:"stances,omitempty"`
}

// NewRegistry is an empty registry.
func NewRegistry() *Registry {
	return &Registry{Players: map[int]map[string]domain.Strategy{}, Tactics: map[int]domain.Tactics{}, Orders: map[int]map[string][]orders.Order{}, Stances: map[int]map[string]stance.Stance{}}
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
	out.Orders = make(map[int]map[string][]orders.Order, len(r.Orders))
	for id, members := range r.Orders {
		m := make(map[string][]orders.Order, len(members))
		for k, v := range members {
			m[k] = append([]orders.Order(nil), v...)
		}
		out.Orders[id] = m
	}
	out.Stances = make(map[int]map[string]stance.Stance, len(r.Stances))
	for id, members := range r.Stances {
		m := make(map[string]stance.Stance, len(members))
		for k, v := range members {
			m[k] = v
		}
		out.Stances[id] = m
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
	return modstore.Load(s.plug, "strategy", func() Registry { return *NewRegistry() }, decodeRegistry, registry)
}

func (s pluginStore) Save(registry Registry) error {
	return modstore.Save(s.plug, "strategy", registry)
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
			clean = cleanWard(key, clean)
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
			mudlog.Warn("strategy: dropped stored tactics", "user", id, "focus", t.Focus, "healing", t.Healing, "patch", t.Patch)
			continue
		}
		if !clean.IsZero() {
			loaded.Tactics[id] = clean
		}
	}
	// Phase 61: orders that don't read, or that are over the limit, are
	// dropped (logged), the rest kept.
	for id, members := range wire.Orders {
		if id <= 0 {
			continue
		}
		for key, list := range members {
			clean := cleanOrders(list)
			if strings.TrimSpace(key) == "" || len(clean) != len(list) {
				mudlog.Warn("strategy: dropped stored orders", "user", id, "member", key, "stored", len(list), "kept", len(clean))
			}
			if strings.TrimSpace(key) == "" || len(clean) == 0 {
				continue
			}
			if loaded.Orders[id] == nil {
				loaded.Orders[id] = map[string][]orders.Order{}
			}
			loaded.Orders[id][key] = clean
		}
	}
	// Phase 69: a stance that isn't one is dropped (logged), the rest kept.
	for id, members := range wire.Stances {
		if id <= 0 {
			continue
		}
		for key, st := range members {
			if strings.TrimSpace(key) == "" || st == stance.None || !st.Valid() {
				mudlog.Warn("strategy: dropped a stored stance", "user", id, "member", key, "stance", st)
				continue
			}
			if loaded.Stances[id] == nil {
				loaded.Stances[id] = map[string]stance.Stance{}
			}
			loaded.Stances[id][key] = st
		}
	}
	*registry = *loaded
	return nil
}

// cleanOrders keeps the stored orders that are valid, up to the limit.
func cleanOrders(list []orders.Order) []orders.Order {
	var out []orders.Order
	for _, o := range list {
		if o.Validate() == nil && len(out) < orders.MaxOrders {
			out = append(out, o)
		}
	}
	return out
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
	if t.Patch != 0 {
		n, ok := domain.ParsePatch(fmt.Sprint(t.Patch))
		if !ok {
			return out, false
		}
		out.Patch = n
	}
	return out, true
}

// cleanWard keeps a stored ward only for a guardian, naming another
// member by a well-formed member key (Phase 30c2); anything else is
// dropped, the rest of the strategy kept.
func cleanWard(key string, s domain.Strategy) domain.Strategy {
	if s.Ward == "" {
		return s
	}
	_, companion := company.CompanionIDFromMemberKey(company.MemberKey(s.Ward))
	if s.Role != domain.Guardian || s.Ward == key || (s.Ward != string(company.LeaderMemberKey) && !companion) {
		mudlog.Warn("strategy: dropped a stored ward", "member", key, "role", s.Role, "ward", s.Ward)
		s.Ward = ""
	}
	return s
}

func cleanStrategy(s domain.Strategy) (domain.Strategy, bool) {
	out := domain.Strategy{Ward: strings.TrimSpace(s.Ward), NoAbilities: s.NoAbilities}
	// Phase 33e: a reserve out of range is dropped, the rest kept.
	if s.Reserve != 0 {
		if r, ok := domain.ParseReserve(fmt.Sprint(s.Reserve)); ok {
			out.Reserve = r
		} else {
			mudlog.Warn("strategy: dropped a stored mana reserve", "reserve", s.Reserve)
		}
	}
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
	userstate.Register(stateContributor{m})
	domain.SetProvider(m)
	domain.SetTacticsProvider(m)
	orders.SetProvider(m)
	m.plug.AddUserCommand("orders", m.ordersCommand, true, false)
	userstate.Register(ordersContributor{m})
	stance.SetProvider(m)
	m.plug.AddUserCommand("stance", m.stanceCommand, true, false)
	userstate.Register(stanceContributor{m})
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
		fields := modconfig.Map(entry)
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

func (m *StrategyModule) persistenceAvailableLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("strategies and tactics can't be changed until they load again: %w", m.loadErr)
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

// StoredOrders implements orders.Provider.
func (m *StrategyModule) StoredOrders(userID int, key string) []orders.Order {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]orders.Order(nil), m.registry.Orders[userID][key]...)
}

// setOrders stores a member's orders (none clears them) and saves, rolling
// back if the save fails.
func (m *StrategyModule) setOrders(userID int, key string, list []orders.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailableLocked(); err != nil {
		return err
	}
	before, had := m.registry.Orders[userID][key]
	m.putOrders(userID, key, list)
	if err := m.store.Save(m.registry.Clone()); err != nil {
		if had {
			m.putOrders(userID, key, before)
		} else {
			m.putOrders(userID, key, nil)
		}
		return fmt.Errorf("the orders couldn't be saved; please try again: %w", err)
	}
	return nil
}

func (m *StrategyModule) putOrders(userID int, key string, list []orders.Order) {
	if len(list) == 0 {
		delete(m.registry.Orders[userID], key)
		if len(m.registry.Orders[userID]) == 0 {
			delete(m.registry.Orders, userID)
		}
		return
	}
	if m.registry.Orders == nil {
		m.registry.Orders = map[int]map[string][]orders.Order{}
	}
	if m.registry.Orders[userID] == nil {
		m.registry.Orders[userID] = map[string][]orders.Order{}
	}
	m.registry.Orders[userID][key] = append([]orders.Order(nil), list...)
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
	for key, s := range m.registry.Players[userID] {
		if !keep[key] {
			delete(m.registry.Players[userID], key)
			changed = true
			continue
		}
		// Phase 30c2: a ward no longer in the company reads as none.
		if s.Ward != "" && !keep[s.Ward] {
			s.Ward = ""
			m.put(userID, key, s)
			changed = true
		}
	}
	if changed && len(m.registry.Players[userID]) == 0 {
		delete(m.registry.Players, userID)
	}
	// Phase 61: orders for a member no longer on the record go too.
	for key := range m.registry.Orders[userID] {
		if !keep[key] {
			delete(m.registry.Orders[userID], key)
			changed = true
		}
	}
	if len(m.registry.Orders[userID]) == 0 {
		delete(m.registry.Orders, userID)
	}
	// Phase 69: so do stances.
	for key := range m.registry.Stances[userID] {
		if !keep[key] {
			delete(m.registry.Stances[userID], key)
			changed = true
		}
	}
	if len(m.registry.Stances[userID]) == 0 {
		delete(m.registry.Stances, userID)
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
	_, hadOrders := m.registry.Orders[evt.UserId]
	_, hadStances := m.registry.Stances[evt.UserId]
	had = had || hadTactics || hadOrders || hadStances
	delete(m.registry.Stances, evt.UserId)
	delete(m.registry.Players, evt.UserId)
	delete(m.registry.Tactics, evt.UserId)
	delete(m.registry.Orders, evt.UserId)
	m.mu.Unlock()
	if had {
		if err := m.save(); err != nil {
			mudlog.Error("strategy: save after purge", "user", evt.UserId, "error", err)
		}
	}
	return events.Continue
}
