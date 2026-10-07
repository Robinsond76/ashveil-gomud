// Package storyevents is Phase 60's module: story events, short scenes the
// company walks into at a cliff, a stranger's fire or a ruined shrine. A page
// of text with choices opens when a trigger fires (a tagged room, a
// journey's end, a finished camp rest); a choice may need a member's skill,
// class, personality or alignment, and the best qualified member present
// takes it and is named. Outcomes change wounds, ailments, needs, supplies,
// gold, loyalty and flags, set a named group on the company, or carry it to
// another room.
//
// The rules and data shapes live in internal/storyevents; this module owns
// each company's saved state (the page waiting for an answer, what it has
// done, its flags), the triggers, the `event` and `choose` commands, the
// effects on the world and the `Event` GMCP message the web client's modal
// reads. Everything runs on the game loop and uses real time only: it never
// advances the world's clock.
//
// A choice is committed before it is applied: the page advances (or the
// event ends and is marked done) and is saved, then the outcomes run. A
// crash in between loses an outcome and never doubles one.
package storyevents

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"gopkg.in/yaml.v2"
)

//go:embed events/*.yaml
var shipped embed.FS

// Pending is the page a company is looking at.
type Pending struct {
	Event string `yaml:"event"`
	Page  string `yaml:"page"`
	// Room is where the scene is; leaving it (a death, a teleport) lets the
	// moment pass.
	Room  int   `yaml:"room"`
	Since int64 `yaml:"since"`
}

// State is one company's saved story events.
type State struct {
	Pending *Pending `yaml:"pending,omitempty"`
	// Done is when each finished event ended, in Unix seconds.
	Done  map[string]int64 `yaml:"done,omitempty"`
	Flags []string         `yaml:"flags,omitempty"`
	// Ops numbers the outcomes that keep an operation id, so a retry after
	// a crash is not applied twice.
	Ops int `yaml:"ops,omitempty"`
}

// clone is a copy that shares nothing with s.
func (s State) clone() State {
	out := s
	if s.Pending != nil {
		p := *s.Pending
		out.Pending = &p
	}
	if s.Done != nil {
		out.Done = make(map[string]int64, len(s.Done))
		for k, v := range s.Done {
			out.Done[k] = v
		}
	}
	out.Flags = append([]string(nil), s.Flags...)
	return out
}

func (s State) flagSet() map[string]bool {
	out := make(map[string]bool, len(s.Flags))
	for _, f := range s.Flags {
		out[f] = true
	}
	return out
}

func (s *State) addFlag(flag string) {
	for _, f := range s.Flags {
		if f == flag {
			return
		}
	}
	s.Flags = append(s.Flags, flag)
	sort.Strings(s.Flags)
}

// Registry is what the module saves.
type Registry struct {
	Companies map[int]State `yaml:"companies"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("storyevents")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Companies: map[int]State{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Companies: map[int]State{}}
	for id, st := range wire.Companies {
		if id > 0 {
			loaded.Companies[id] = st
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("storyevents", r) }

// Module is the module's state.
type Module struct {
	plug *plugins.Plugin

	// mu guards state, catalog and the warnings. It is a leaf lock: nothing
	// reaches into the world while holding it.
	mu      sync.Mutex
	store   Store
	loadErr error
	state   map[int]State
	catalog *storyevents.Catalog
	warned  map[string]bool

	clock func() time.Time
	rng   func(n int) int
	w     world
	// readFiles returns every events file's bytes; tests replace it.
	readFiles func() map[string][]byte
}

var module *Module

func init() {
	m := newModule()
	m.plug = plugins.New("storyevents", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("storyevents: save", "error", err)
		}
	})
	m.plug.AddUserCommand("event", m.eventCommand, true, false)
	m.plug.AddUserCommand("choose", m.chooseCommand, true, false)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	userstate.Register(stateContributor{m})
	walking.AddStepListener(func(userID, _, roomID int) { m.entered(userID, roomID, false) })
	walking.AddArrivalListener(func(userID, _, roomID int) { m.entered(userID, roomID, true) })
	camping.AddRestEndListener(m.campRestEnded)
	storyevents.SetMovementProvider(m)
	storyevents.SetFlagProvider(m)
	module = m
}

func newModule() *Module {
	return &Module{
		state:     map[int]State{},
		warned:    map[string]bool{},
		clock:     time.Now,
		rng:       util.Rand,
		w:         liveWorld{},
		readFiles: readEventFiles,
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
		mudlog.Error("storyevents: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = loaded.Companies
	if m.state == nil {
		m.state = map[int]State{}
	}
	m.loadErr = nil
	m.catalog = nil // read the files again on the next use
}

func (m *Module) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Module) saveLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("storyevents: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Companies: m.state})
}

// readEventFiles reads the shipped events and the world's own `events`
// folder; a file of the world wins over a shipped one of the same name.
func readEventFiles() map[string][]byte {
	out := map[string][]byte{}
	entries, _ := fs.ReadDir(shipped, "events")
	for _, e := range entries {
		if data, err := shipped.ReadFile("events/" + e.Name()); err == nil {
			out[e.Name()] = data
		}
	}
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "events")
	if disk, err := os.ReadDir(dir); err == nil {
		for _, e := range disk {
			if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
				out[e.Name()] = data
			}
		}
	}
	return out
}

// events returns the validated catalog, read once and logging each problem
// once.
func (m *Module) events() storyevents.Catalog {
	m.mu.Lock()
	if m.catalog != nil {
		defer m.mu.Unlock()
		return *m.catalog
	}
	m.mu.Unlock()
	// Built outside the lock: validation reaches into the world (rooms,
	// items, mobs), and mu stays a leaf lock.
	files := m.readFiles()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var all []storyevents.Event
	for _, n := range names {
		list, err := storyevents.Parse(files[n])
		if err != nil {
			m.warnOnce("file:"+n, "storyevents: events file skipped", "file", n, "error", err)
			continue
		}
		all = append(all, list...)
	}
	cat, problems := storyevents.NewCatalog(all, m.w.Lookups())
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.catalog != nil {
		return *m.catalog
	}
	for _, p := range problems {
		m.warnOnceLocked("event:"+p, "storyevents: event disabled", "reason", p)
	}
	m.catalog = &cat
	return cat
}

func (m *Module) warnOnce(key, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnOnceLocked(key, msg, args...)
}

func (m *Module) warnOnceLocked(key, msg string, args ...any) {
	if m.warned[key] {
		return
	}
	m.warned[key] = true
	mudlog.Warn(msg, args...)
}

// MovementBlocked implements storyevents.MovementProvider: a company with a
// page waiting stays where it is until it answers.
func (m *Module) MovementBlocked(leaderUserID int) (bool, string) {
	if _, _, ok := m.waiting(leaderUserID); !ok {
		return false, ""
	}
	return true, `<ansi fg="yellow">A scene is waiting on your company. Type <ansi fg="command">choose</ansi> and a number to answer it, or <ansi fg="command">event</ansi> to read it again.</ansi>`
}

// waiting is the page the company is looking at, if its room is still the
// company's. A page whose room the leader has left (a death, a teleport)
// has passed: it is dropped, not done, so the scene can open again.
func (m *Module) waiting(userID int) (storyevents.Event, Pending, bool) {
	m.mu.Lock()
	st, ok := m.state[userID]
	m.mu.Unlock()
	if !ok || st.Pending == nil {
		return storyevents.Event{}, Pending{}, false
	}
	p := *st.Pending
	ev, found := m.events().Get(p.Event)
	if !found {
		m.drop(userID, p, false)
		return storyevents.Event{}, Pending{}, false
	}
	if _, ok := ev.Pages[p.Page]; !ok {
		m.drop(userID, p, false)
		return storyevents.Event{}, Pending{}, false
	}
	user := m.w.User(userID)
	if user == nil || user.Character == nil || user.Character.RoomId != p.Room {
		// Past the first page an answer has already paid out, so the
		// scene counts as done: leaving mid-scene must not reopen it.
		m.drop(userID, p, p.Page != ev.StartPage())
		return storyevents.Event{}, Pending{}, false
	}
	return ev, p, true
}

// drop forgets a page that can no longer be answered; done marks the event
// finished, as a scene left after an answer was.
func (m *Module) drop(userID int, p Pending, done bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.state[userID]
	if !ok || cur.Pending == nil || *cur.Pending != p {
		return
	}
	st := cur.clone()
	st.Pending = nil
	if done {
		if st.Done == nil {
			st.Done = map[string]int64{}
		}
		st.Done[p.Event] = m.clock().Unix()
	}
	m.state[userID] = st
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("storyevents: save after dropping a page", "user", userID, "error", err)
	}
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
		mudlog.Error("storyevents: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}

// onPlayerSpawn shows a page that was waiting through a logout, restart or
// copyover.
func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	m.reshow(evt.UserId, "")
	return events.Continue
}

// CompanyFlags implements storyevents.FlagProvider: the company's flags,
// sorted.
func (m *Module) CompanyFlags(leaderUserID int) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.state[leaderUserID].Flags...)
}

// SetCompanyFlag implements storyevents.FlagProvider: the flag is saved at
// once and rolled back if the save fails.
func (m *Module) SetCompanyFlag(leaderUserID int, flag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, had := m.state[leaderUserID]
	st := cur.clone()
	st.addFlag(flag)
	if len(st.Flags) == len(cur.Flags) {
		return nil
	}
	m.state[leaderUserID] = st
	if err := m.saveLocked(); err != nil {
		if had {
			m.state[leaderUserID] = cur
		} else {
			delete(m.state, leaderUserID)
		}
		return err
	}
	return nil
}
