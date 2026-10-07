// Package chronicle is Phase 63's module: each company's durable record of
// its deeds. The entry shapes, prose and query seam are in
// internal/chronicle; this module owns the saved logs, the `chronicle`
// command, the `Company.Chronicle` GMCP message the web client's Chronicle
// tab reads, and the hooks that need no other module's help (a leader's
// death). Other modules record their own deeds from their real sources with
// chronicle.Record.
//
// Everything runs on the game loop and uses real time only.
package chronicle

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"gopkg.in/yaml.v2"
)

// Registry is what the module saves: one log per company leader.
type Registry struct {
	Companies map[int]chronicle.Log `yaml:"companies"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("chronicle")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Companies: map[int]chronicle.Log{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Companies: map[int]chronicle.Log{}}
	for id, l := range wire.Companies {
		if id > 0 {
			loaded.Companies[id] = l
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("chronicle", r) }

// Module is the module's state.
type Module struct {
	plug *plugins.Plugin

	// mu guards the logs. It is a leaf lock: nothing reaches into the world
	// while holding it.
	mu      sync.Mutex
	store   Store
	loadErr error
	logs    map[int]chronicle.Log

	clock func() time.Time
	w     world
}

var module *Module

func init() {
	m := newModule()
	m.plug = plugins.New("chronicle", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("chronicle: save", "error", err)
		}
	})
	m.plug.AddUserCommand("chronicle", m.chronicleCommand, true, false)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.PlayerDeath{}, m.onPlayerDeath)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	userstate.Register(stateContributor{m})
	chronicle.SetProvider(m)
	module = m
}

func newModule() *Module {
	return &Module{
		logs:  map[int]chronicle.Log{},
		clock: time.Now,
		w:     liveWorld{},
	}
}

func (m *Module) load() {
	if m.store == nil {
		return
	}
	var loaded Registry
	if err := m.store.Load(&loaded); err != nil {
		m.mu.Lock()
		m.loadErr = err
		m.mu.Unlock()
		mudlog.Error("chronicle: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = loaded.Companies
	if m.logs == nil {
		m.logs = map[int]chronicle.Log{}
	}
	m.loadErr = nil
}

func (m *Module) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Module) saveLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("chronicle: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Companies: m.logs})
}

// Record implements chronicle.Provider. The deed is saved at once; a failed
// save leaves it in memory for the next save, since a deed is a fact that
// has already happened and must not be refused.
func (m *Module) Record(leaderUserID int, e chronicle.Entry) {
	if e.At == 0 {
		e.At = m.clock().Unix()
	}
	if e.Place == "" {
		e.Place = m.w.Place(leaderUserID)
	}
	m.mu.Lock()
	l := m.logs[leaderUserID]
	l.Add(e)
	m.logs[leaderUserID] = l
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("chronicle: save after a deed", "user", leaderUserID, "kind", e.Kind, "error", err)
	}
	m.mu.Unlock()
	m.push(leaderUserID)
}

// Log implements chronicle.Provider.
func (m *Module) Log(leaderUserID int) chronicle.Log {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs[leaderUserID].Clone()
}

func (m *Module) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, held := m.logs[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.logs, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("chronicle: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}

// onPlayerSpawn gives the web client's Chronicle tab its log at login and
// after a copyover.
func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		m.push(evt.UserId)
	}
	return events.Continue
}

// onPlayerDeath records the leader's own fall; a companion's death is
// recorded by the company module, which knows which companion it was.
func (m *Module) onPlayerDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDeath)
	if !ok || evt.UserId <= 0 {
		return events.Continue
	}
	m.Record(evt.UserId, chronicle.Entry{
		Kind:    chronicle.Fell,
		Members: []string{m.w.Name(evt.UserId, evt.CharacterName)},
		Subject: m.w.MobName(evt.KillerMobId),
		Place:   m.w.RoomTitle(evt.RoomId),
	})
	return events.Continue
}
