// Package townsfolk is Phase 68's module: town NPCs tagged as talkers
// mention what a company has done. Rules, line shapes and the speaking seam
// are in internal/townsfolk; this module owns what each player has been told
// (saved, so a restart never repeats a line), the line data, the `townsfolk`
// command and the `Company.Townsfolk` GMCP message the web client's
// Chronicle tab reads.
//
// It reads the chronicle (Phase 63) for deeds and never keeps its own tally.
// Everything runs on the game loop and uses real time only.
package townsfolk

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

	"github.com/GoMudEngine/GoMud/internal/chronicle"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/townsfolk"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"gopkg.in/yaml.v2"
)

const (
	// maxHeard is how many told deeds a player's memory keeps: the chronicle
	// keeps 300, so an older deed is gone before its memory is.
	maxHeard = chronicle.MaxEntries
	// maxTold is how many recent tellings the views show. It is also how
	// far back "heard lately" looks: a background line the listener heard
	// within this many tellings no longer beats plain lines.
	maxTold = 12
	// queryLimit is how many of the newest deeds a choice reads.
	queryLimit = 80
)

//go:embed lines/*.yaml
var shipped embed.FS

// Told is one telling, kept for the views.
type Told struct {
	Seq  int    `yaml:"seq"`
	At   int64  `yaml:"at"`
	Line string `yaml:"line"`
	Text string `yaml:"text"`
}

// UserState is what one player has been told.
type UserState struct {
	Heard []int  `yaml:"heard,omitempty"` // deed seqs told, oldest first
	Told  []Told `yaml:"told,omitempty"`  // newest last
	Total int    `yaml:"total,omitempty"` // tellings ever
}

func (s UserState) heard(seq int) bool {
	for _, h := range s.Heard {
		if h == seq {
			return true
		}
	}
	return false
}

// toldLine reports whether a line is among the player's recent tellings.
func (s UserState) toldLine(id string) bool {
	for _, t := range s.Told {
		if t.Line == id {
			return true
		}
	}
	return false
}

// Registry is what the module saves: one memory per player.
type Registry struct {
	Users map[int]UserState `yaml:"users"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("townsfolk")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Users: map[int]UserState{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Users: map[int]UserState{}}
	for id, st := range wire.Users {
		if id > 0 {
			loaded.Users[id] = st
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("townsfolk", r) }

// Module is the module's state.
type Module struct {
	plug *plugins.Plugin

	// mu guards state and catalog. It is a leaf lock: nothing reaches the
	// world while holding it.
	mu      sync.Mutex
	store   Store
	loadErr error
	state   map[int]UserState
	catalog *townsfolk.Catalog

	clock     func() time.Time
	readFiles func() map[string][]byte
	w         world
}

var module *Module

func init() {
	m := newModule()
	m.plug = plugins.New("townsfolk", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("townsfolk: save", "error", err)
		}
	})
	m.plug.AddUserCommand("townsfolk", m.command, true, false)
	m.plug.AddUserCommand("renown", m.command, true, false)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	// A new deed may be one the towns can speak of: refresh the view.
	chronicle.OnRecord(func(uid int, _ chronicle.Entry) { m.push(uid) })
	userstate.Register(stateContributor{m})
	townsfolk.SetProvider(m)
	module = m
}

func newModule() *Module {
	return &Module{
		state:     map[int]UserState{},
		clock:     time.Now,
		readFiles: readLineFiles,
		w:         liveWorld{},
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
		mudlog.Error("townsfolk: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = loaded.Users
	if m.state == nil {
		m.state = map[int]UserState{}
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
		return fmt.Errorf("townsfolk: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Users: m.state})
}

// readLineFiles reads the shipped lines and the world's own `townsfolk`
// folder; a file of the world wins over a shipped one of the same name.
func readLineFiles() map[string][]byte {
	out := map[string][]byte{}
	entries, _ := fs.ReadDir(shipped, "lines")
	for _, e := range entries {
		if data, err := shipped.ReadFile("lines/" + e.Name()); err == nil {
			out[e.Name()] = data
		}
	}
	dir := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "townsfolk")
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

// lines returns the validated catalog, read once and logging each problem
// once.
func (m *Module) lines() townsfolk.Catalog {
	m.mu.Lock()
	if m.catalog != nil {
		defer m.mu.Unlock()
		return *m.catalog
	}
	m.mu.Unlock()
	files := m.readFiles()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var all []townsfolk.Line
	for _, n := range names {
		list, err := townsfolk.Parse(files[n])
		if err != nil {
			mudlog.Warn("townsfolk: skipping a file", "file", n, "error", err)
			continue
		}
		all = append(all, list...)
	}
	cat, problems := townsfolk.NewCatalog(all)
	for _, p := range problems {
		mudlog.Warn("townsfolk: skipping a line", "problem", p)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.catalog == nil {
		m.catalog = &cat
	}
	return *m.catalog
}

// maxWindow is the longest any deed line keeps a deed worth telling.
func maxWindow(cat townsfolk.Catalog) int64 {
	var w int64
	for _, l := range cat.Lines() {
		if l.IsDeed() && l.Window() > w {
			w = l.Window()
		}
	}
	return w
}

// Speak implements townsfolk.Provider: the first listener with a deed this
// NPC may tell hears it (once: confirming the speech saves it before the
// queued line runs); with none, the NPC
// falls back to a state line for the room, or says nothing.
func (m *Module) Speak(npc townsfolk.NPC, listeners []int) (townsfolk.Speech, bool) {
	cat := m.lines()
	if cat.Len() == 0 {
		return townsfolk.Speech{}, false
	}
	now := m.clock().Unix()
	since := now - maxWindow(cat)
	var fallback *townsfolk.Choice
	for _, uid := range listeners {
		name := m.w.Name(uid)
		if name == "" {
			continue
		}
		m.mu.Lock()
		st := m.state[uid]
		m.mu.Unlock()
		ctx := townsfolk.Context{
			NPC:       npc,
			Now:       now,
			Leader:    name,
			Entries:   chronicle.Query(uid, chronicle.Filter{Since: since, Limit: queryLimit}),
			Heard:     st.heard,
			LineHeard: st.toldLine,
			Flag:      func(f string) bool { return m.w.Flag(uid, f) },
			MemberTag: func(key, tag string) bool { return m.w.MemberTag(uid, key, tag) },
			Weather:   m.w.Weather(npc.Zone),
			Night:     m.w.Night(),
			Rand:      m.w.Rand,
		}
		ch, ok := cat.Choose(ctx)
		if !ok {
			continue
		}
		if ch.Entry == nil {
			if fallback == nil {
				c := ch
				fallback = &c
			}
			continue
		}
		uid := uid
		return townsfolk.Speech{To: fmt.Sprintf("@%d", uid), Text: ch.Text, Said: func() { m.told(uid, ch, now) }}, true
	}
	if fallback != nil {
		return townsfolk.Speech{Text: fallback.Text}, true
	}
	return townsfolk.Speech{}, false
}

// told remembers a deed was spoken of, leaves the line's mark, and refreshes
// the player's view. The memory is saved before the NPC speaks, so a crash
// never repeats a line.
func (m *Module) told(uid int, ch townsfolk.Choice, now int64) {
	m.mu.Lock()
	st := m.state[uid]
	st.Heard = append(append([]int(nil), st.Heard...), ch.Entry.Seq)
	if over := len(st.Heard) - maxHeard; over > 0 {
		st.Heard = st.Heard[over:]
	}
	st.Told = append(append([]Told(nil), st.Told...), Told{Seq: ch.Entry.Seq, At: now, Line: ch.Line.ID, Text: ch.Text})
	if over := len(st.Told) - maxTold; over > 0 {
		st.Told = st.Told[over:]
	}
	st.Total++
	m.state[uid] = st
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("townsfolk: save after a telling", "user", uid, "error", err)
	}
	m.mu.Unlock()
	if ch.Line.Sets != "" {
		if err := m.w.SetFlag(uid, ch.Line.Sets); err != nil {
			mudlog.Warn("townsfolk: leaving a mark", "user", uid, "mark", ch.Line.Sets, "error", err)
		}
	}
	m.push(uid)
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
		mudlog.Error("townsfolk: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}

// onPlayerSpawn gives the web client's Chronicle tab its view at login and
// after a copyover.
func (m *Module) onPlayerSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		m.push(evt.UserId)
	}
	return events.Continue
}
