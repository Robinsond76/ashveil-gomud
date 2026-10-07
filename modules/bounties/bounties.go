// Package bounties is Phase 76's bounty boards. A room tagged `bounty-board`
// lists bounties for lair bosses and named groups in nearby zones; a company
// takes one, kills the mark, and claims the gold at a board. The proof is the
// chronicle's boss and group deeds. The rules (the posting, the pay, the
// saved shapes) are in internal/bounty; this module owns the saved holdings,
// the `bounty` and `bounties` commands, and the `Company.Bounties` GMCP
// message the Company window's Bounties tab reads.
//
// Real time only; a bounty never scales a foe or moves the world clock.
package bounties

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/bounty"
	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"gopkg.in/yaml.v2"
)

// Registry is what the module saves: one state per company leader.
type Registry struct {
	Companies map[int]bounty.State `yaml:"companies"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("bounties")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Companies: map[int]bounty.State{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Companies: map[int]bounty.State{}}
	for id, st := range wire.Companies {
		if id > 0 {
			loaded.Companies[id] = st
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("bounties", r) }

// Module is the module's state.
type Module struct {
	plug *plugins.Plugin

	// mu guards state. It is a leaf lock: nothing reaches the world or the
	// chronicle while holding it.
	mu      sync.Mutex
	store   Store
	loadErr error
	state   map[int]bounty.State

	clock func() time.Time
	w     world
}

var module *Module

func init() {
	m := newModule()
	m.plug = plugins.New("bounties", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("bounties: save", "error", err)
		}
	})
	m.plug.AddUserCommand("bounty", m.command, false, false)
	m.plug.AddUserCommand("bounties", m.command, false, false)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	// A new deed may be proof: refresh the progress the tab shows.
	chronicle.OnRecord(func(uid int, _ chronicle.Entry) { m.push(uid) })
	userstate.Register(stateContributor{m})
	bounty.SetHunting(m.hunting)
	module = m
}

// hunting reports whether the company holds a bounty, not yet lapsed, on
// the target with this chronicle reference in this zone.
func (m *Module) hunting(userID int, ref, zone string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.clock().Unix()
	for _, h := range m.state[userID].Held {
		if h.Target.Ref == ref && h.Target.Zone == zone && now < h.Due {
			return true
		}
	}
	return false
}

func newModule() *Module {
	return &Module{state: map[int]bounty.State{}, clock: time.Now, w: liveWorld{}}
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
		mudlog.Error("bounties: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = loaded.Companies
	if m.state == nil {
		m.state = map[int]bounty.State{}
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
		return fmt.Errorf("bounties: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Companies: m.state})
}

// stateOf is the company's bounties with lapsed ones removed (the lapse is
// saved). It returns how many just lapsed.
func (m *Module) stateOf(userID int) (bounty.State, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, held := m.state[userID]
	if !held {
		return bounty.State{}, 0
	}
	st = st.Clone()
	lapsed := st.Expired(m.clock().Unix())
	if lapsed > 0 || len(st.Done) != len(m.state[userID].Done) {
		m.putLocked(userID, st)
		if err := m.saveLocked(); err != nil {
			mudlog.Warn("bounties: save after a lapse", "user", userID, "error", err)
		}
	}
	return st, lapsed
}

func (m *Module) putLocked(userID int, st bounty.State) {
	if st.Empty() {
		delete(m.state, userID)
		return
	}
	m.state[userID] = st
}

// commit stores st for the user and saves, putting the old state back when
// the save fails, so a refused take or claim leaves nothing half done.
func (m *Module) commit(userID int, st bounty.State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, had := m.state[userID]
	m.putLocked(userID, st)
	if err := m.saveLocked(); err != nil {
		if had {
			m.state[userID] = old
		} else {
			delete(m.state, userID)
		}
		return err
	}
	return nil
}

// postings is the board's current list.
func (m *Module) postings(b board) []bounty.Posting {
	return bounty.Post(b.Key, b.Band, bounty.Window(m.clock().Unix()), m.w.Candidates())
}

// progress is how many of the bounty's marks the company's chronicle shows
// since it was taken (at most the count asked).
func progress(userID int, h bounty.Held) int {
	kind := chronicle.Boss
	if h.Target.Kind == bounty.Group {
		kind = chronicle.Group
	}
	n := chronicle.Count(userID, chronicle.Filter{Kinds: []chronicle.Kind{kind}, Ref: h.Target.Ref, Zone: h.Target.Zone, AfterSeq: h.AfterSeq, Since: h.TakenAt})
	return min(n, max(1, h.Target.Count))
}

// newestSeq is the number of the company's newest deed (0 when none).
func newestSeq(userID int) int {
	if e := chronicle.Query(userID, chronicle.Filter{Limit: 1}); len(e) > 0 {
		return e[0].Seq
	}
	return 0
}

func (m *Module) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, held := m.state[evt.UserId]; !held {
		return events.Continue
	}
	delete(m.state, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("bounties: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}

// onPlayerSpawn gives the web client's Bounties tab its list at login and
// after a copyover.
func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		m.push(evt.UserId)
	}
	return events.Continue
}
