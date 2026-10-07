// Package blessings is Phase 77's module: the account blessings (small
// perks a player's later characters start with) and the `blessings`
// command. What a blessing is, and what its perk does, is in
// internal/blessings; this module owns the saved list of what each account
// has earned.
//
// A blessing is earned when a character's chronicle reaches its milestone
// (checked on every deed and at login) and is kept for the account when the
// character is deleted. The character that earned it does not get it: each
// new character is given the account's blessings once, as creation ends
// (usercommands.Start asks blessings.EarnedFor and applies them).
package blessings

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"gopkg.in/yaml.v2"
)

// Earned is one blessing an account has earned.
type Earned struct {
	ID string `yaml:"id"`
	At int64  `yaml:"at"`
	By string `yaml:"by,omitempty"` // the character whose deeds earned it
}

// Registry is what the module saves: the blessings each account has earned,
// in the order it earned them.
type Registry struct {
	Accounts map[int][]Earned `yaml:"accounts"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("blessings")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Accounts: map[int][]Earned{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Accounts: map[int][]Earned{}}
	for id, list := range wire.Accounts {
		if id > 0 {
			loaded.Accounts[id] = list
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("blessings", r) }

// Module is the module's state.
type Module struct {
	plug *plugins.Plugin

	// mu guards the accounts. It is a leaf lock: nothing reaches the world
	// while holding it.
	mu       sync.Mutex
	store    Store
	loadErr  error
	accounts map[int][]Earned

	clock func() time.Time
	w     world
}

var module *Module

func init() {
	m := newModule()
	m.plug = plugins.New("blessings", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("blessings: save", "error", err)
		}
	})
	m.plug.AddUserCommand("blessings", m.blessingsCommand, true, false)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	// A purge that keeps the account (a deleted character) keeps its
	// blessings: that is their point. Only a purge that removes the whole
	// account drops them.
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	chronicle.OnRecord(m.onDeed)
	blessings.SetProvider(m)
	module = m
}

func newModule() *Module {
	return &Module{
		accounts: map[int][]Earned{},
		clock:    time.Now,
		w:        liveWorld{},
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
		mudlog.Error("blessings: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accounts = loaded.Accounts
	if m.accounts == nil {
		m.accounts = map[int][]Earned{}
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
		return fmt.Errorf("blessings: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Accounts: m.accounts})
}

// Earned implements blessings.Provider: the account's earned ids, in the
// order earned.
func (m *Module) Earned(userID int) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.accounts[userID]
	out := make([]string, 0, len(list))
	for _, e := range list {
		out = append(out, e.ID)
	}
	return out
}

func (m *Module) earnedRecords(userID int) []Earned {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Earned(nil), m.accounts[userID]...)
}

// grant records a blessing for the account, once. It reports whether it
// was new. A failed save keeps the blessing in memory for the next save: an
// earned blessing is a fact, never refused.
func (m *Module) grant(userID int, id, by string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.accounts[userID] {
		if e.ID == id {
			return false
		}
	}
	m.accounts[userID] = append(m.accounts[userID], Earned{ID: id, At: m.clock().Unix(), By: by})
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("blessings: save after a grant", "user", userID, "blessing", id, "error", err)
	}
	return true
}

// evaluate grants every blessing the user's current chronicle has earned
// that the account does not have yet, and tells the player. Offline users
// are skipped: the check runs again at their next login.
func (m *Module) evaluate(userID int) {
	who, ok := m.w.Character(userID)
	if !ok {
		return
	}
	total := func(k chronicle.Kind) int { return chronicle.Total(userID, k) }
	for _, b := range blessings.Earned(total, who.Iron) {
		if !m.grant(userID, b.ID, who.Name) {
			continue
		}
		m.w.Tell(userID, fmt.Sprintf(
			`<ansi fg="yellow-bold">A blessing is yours:</ansi> <ansi fg="itemname">%s</ansi>. %s Your next character will %s. <ansi fg="black-bold">(blessings)</ansi>`,
			b.Name, b.Text, b.PerkText()))
		m.push(userID)
	}
}

// Given implements blessings.Watcher: a new character was given the
// account's blessings, so the panel moves them from waiting to carried.
func (m *Module) Given(userID int) { m.push(userID) }

func (m *Module) onDeed(leaderUserID int, _ chronicle.Entry) { m.evaluate(leaderUserID) }

func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		m.evaluate(evt.UserId)
		m.push(evt.UserId)
	}
	return events.Continue
}

func (m *Module) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok || evt.KeepAccount {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, held := m.accounts[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.accounts, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("blessings: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}
