// Package encounters is Phase 37's random room encounters: after an
// ordinary step or a journey's arrival into a room that opted in, a chance
// (the room's, else the zone's) springs a battle of 2-4 foes (or a boss and
// its escorts) levelled from the zone's band and set upon the mover's
// company alone. Policy and content live in internal/encounters, the spawn
// in internal/enemyparty and the rewards in internal/loot; this module is
// the glue: it hears the movement, applies the eligibility rules, keeps
// each leader's saved grace after a battle, and cleans up groups nobody is
// fighting any more.
//
// Everything runs on the game loop and uses real time only: it never
// advances the world's clock. An unresolved group is runtime state, like
// every mob in this codebase: a restart or copyover ends it without
// reward, and the grace that follows a battle is what persists.
package encounters

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actionpolicy"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/encounters"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"gopkg.in/yaml.v2"
)

// defaultText opens an encounter whose composition has no text of its own.
const defaultText = `Shapes stir nearby, and a party of foes closes in on your company!`

// Registry is what the module saves: each leader's grace.
type Registry struct {
	Graces map[int]encounters.Grace `yaml:"graces"`
	// Bosses is when each company's lairs wake again: leader, then the
	// boss composition's id, to the real time it may spring (37b).
	Bosses map[int]map[string]time.Time `yaml:"bosses,omitempty"`
}

// Store abstracts persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(r *Registry) error {
	data, err := s.plug.ReadBytes("encounters")
	if errors.Is(err, os.ErrNotExist) {
		*r = Registry{Graces: map[int]encounters.Grace{}, Bosses: map[int]map[string]time.Time{}}
		return nil
	}
	if err != nil {
		return err
	}
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := Registry{Graces: map[int]encounters.Grace{}, Bosses: map[int]map[string]time.Time{}}
	for id, g := range wire.Graces {
		if id > 0 {
			loaded.Graces[id] = g
		}
	}
	for id, comps := range wire.Bosses {
		if id > 0 && len(comps) > 0 {
			loaded.Bosses[id] = comps
		}
	}
	*r = loaded
	return nil
}

func (s pluginStore) Save(r Registry) error { return s.plug.WriteStruct("encounters", r) }

// record is one unresolved random group.
type record struct {
	enemyparty.Encounter
	ownerless time.Time // when nobody was last fighting it; zero while someone is
	comp      string    // the composition's id
	cooled    bool      // a fallen boss's lair cooldown has been started
}

// EncountersModule is the module's state.
type EncountersModule struct {
	plug *plugins.Plugin

	mu      sync.Mutex
	store   Store
	loadErr error
	graces  map[int]encounters.Grace
	bosses  map[int]map[string]time.Time
	active  map[string]*record // by group id

	clock func() time.Time
	rng   encounters.Rand
	spawn func(roomID, owner int, foes []encounters.Foe) (enemyparty.Encounter, error)
	warn  map[string]bool // diagnostics already logged
}

func init() {
	m := newModule()
	m.plug = plugins.New("encounters", "1.0")
	m.store = pluginStore{plug: m.plug}
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("encounters: save", "error", err)
		}
	})
	events.RegisterListener(events.BattleEnded{}, m.onBattleEnded)
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	walking.AddStepListener(m.onStep)
	walking.AddArrivalListener(m.onArrival)
}

func newModule() *EncountersModule {
	return &EncountersModule{
		graces: map[int]encounters.Grace{},
		bosses: map[int]map[string]time.Time{},
		active: map[string]*record{},
		clock:  time.Now,
		rng:    util.Rand,
		spawn:  enemyparty.SpawnEncounter,
		warn:   map[string]bool{},
	}
}

func (m *EncountersModule) load() {
	if m.store == nil {
		return
	}
	var loaded Registry
	if err := m.store.Load(&loaded); err != nil {
		m.mu.Lock()
		m.loadErr = err
		m.mu.Unlock()
		mudlog.Error("encounters: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graces = loaded.Graces
	if m.graces == nil {
		m.graces = map[int]encounters.Grace{}
	}
	m.bosses = loaded.Bosses
	if m.bosses == nil {
		m.bosses = map[int]map[string]time.Time{}
	}
	m.loadErr = nil
}

func (m *EncountersModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *EncountersModule) saveLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("encounters: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return nil
	}
	return m.store.Save(Registry{Graces: m.graces, Bosses: m.bosses})
}

func (m *EncountersModule) onStep(userID, _, toRoomID int) { m.Entered(userID, toRoomID) }

func (m *EncountersModule) onArrival(userID, _, toRoomID int) { m.Entered(userID, toRoomID) }

// lookup finds a template's facts for validation.
func lookup(mobID int) (encounters.Template, bool) {
	spec := mobs.GetMobSpec(mobs.MobId(mobID))
	if spec == nil {
		return encounters.Template{}, false
	}
	return encounters.Template{Solitary: spec.Solitary, Healer: spec.Role == "healer"}, true
}

// zoneTables is the zone's validated encounter content, logging each
// diagnostic once.
func (m *EncountersModule) zoneTables(zone string) (encounters.ZoneConfig, bool) {
	cfg := rooms.GetZoneConfig(zone)
	if cfg == nil || len(cfg.Encounters.Tables) == 0 {
		return encounters.ZoneConfig{}, false
	}
	valid, diags := cfg.Encounters.Validate(lookup)
	for _, d := range diags {
		key := zone + "|" + d.String()
		if !m.warn[key] {
			m.warn[key] = true
			mudlog.Warn("encounters: content disabled", "zone", zone, "reason", d.String())
		}
	}
	return valid, len(valid.Tables) > 0
}

// Entered is called after an ordinary step or a journey's arrival: it
// decides whether the room springs an encounter on the mover's company.
// It never rolls on look, scout, login, spawn, relocation or a failed
// move, because only those two producers call it.
func (m *EncountersModule) Entered(userID, roomID int) {
	user := users.GetByUserId(userID)
	if user == nil || user.Character == nil || user.Character.RoomId != roomID {
		return
	}
	room := rooms.LoadRoom(roomID)
	if room == nil || room.Encounter == nil || !room.Encounter.Enabled {
		return
	}
	// A follower in a player party does not roll: the leader's move did.
	if p := parties.Get(userID); p != nil && !p.IsLeader(userID) {
		return
	}
	if user.Character.Health < 1 || user.Character.CombatWithdrawn || actionpolicy.InBattle(user) {
		return
	}
	if blocked, _ := expedition.MovementBlocked(userID); blocked {
		return
	}
	if blocked, _ := camping.MovementBlocked(userID); blocked {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	zone, ok := m.zoneTables(room.Zone)
	if !ok {
		return
	}
	table := zone.Tables[room.Encounter.Table]
	// A lair that this company emptied lately stays quiet (37b).
	table = encounters.Available(table, func(id string) bool { return m.bossCoolingLocked(userID, id) })
	if len(table) == 0 {
		return
	}
	// An engaging party takes precedence, and one leader holds one group.
	if m.engagedLocked(room, userID) {
		return
	}
	inRoom := 0
	for _, r := range m.active {
		if r.Owner == userID {
			return
		}
		if r.RoomID == roomID {
			inRoom++
		}
	}
	if inRoom >= encounters.RoomGroupLimit {
		return
	}
	now := m.clock()
	g, seen := m.graces[userID]
	if !seen {
		g = encounters.NewGrace(now) // a fresh character starts with the grace
	}
	before := g
	suppressed := g.Suppresses(now)
	m.graces[userID] = g
	// Save only when the grace changed: most entries consume nothing, and a
	// save on every step would write the registry on the game loop.
	if !seen || g != before {
		if err := m.saveLocked(); err != nil {
			mudlog.Warn("encounters: save grace", "user", userID, "error", err)
		}
	}
	if suppressed {
		return
	}
	if !encounters.Roll(room.Encounter.ChanceIn(zone), m.rng) {
		return
	}
	comp, ok := encounters.Pick(table, m.rng)
	if !ok {
		return
	}
	enc, err := m.spawn(roomID, userID, encounters.Plan(comp, zone.Band, m.rng))
	if err != nil {
		mudlog.Warn("encounters: spawn failed", "room", roomID, "composition", comp.ID, "error", err)
		return
	}
	m.active[enc.ID] = &record{Encounter: enc, comp: comp.ID}
	text := comp.Text
	if text == "" {
		text = defaultText
	}
	room.SendText(text)
}

// engagedLocked reports whether a foe in the room is already set on the user.
func (m *EncountersModule) engagedLocked(room *rooms.Room, userID int) bool {
	for _, id := range room.GetMobs() {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if a := mob.Character.Aggro; a != nil && a.UserId == userID {
			return true
		}
	}
	return false
}

// onBattleEnded starts the leader's grace: two eligible entries and 30
// seconds before the next roll. It is saved, so crossing a zone, logging
// out or a restart cannot reset it.
func (m *EncountersModule) onBattleEnded(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.BattleEnded)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graces[evt.UserId] = encounters.NewGrace(m.clock())
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("encounters: save grace", "user", evt.UserId, "error", err)
	}
	m.settleLocked(true)
	return events.Continue
}

// onNewRound clears groups nobody is fighting any more.
func (m *EncountersModule) onNewRound(e events.Event) events.ListenerReturn {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settleLocked(false)
	return events.Continue
}

// settleLocked retires won groups and removes the survivors of a group its
// participants have left: at once when a battle just ended (flee, retreat,
// the leader's fall), else once the group has stood ownerless for the
// timeout. A foe never disappears while someone is still fighting it.
func (m *EncountersModule) settleLocked(battleJustEnded bool) {
	now := m.clock()
	ids := make([]string, 0, len(m.active))
	for id := range m.active {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		r := m.active[id]
		m.noteBossFallenLocked(r, now)
		if r.standing() == 0 {
			delete(m.active, id)
			continue
		}
		if r.participating() {
			r.ownerless = time.Time{}
			continue
		}
		if r.ownerless.IsZero() {
			r.ownerless = now
		}
		if battleJustEnded || now.Sub(r.ownerless) >= encounters.AbandonSeconds*time.Second {
			r.Remove()
			delete(m.active, id)
		}
	}
}

// bossCoolingLocked reports whether the user's lair for composition id is
// still quiet.
func (m *EncountersModule) bossCoolingLocked(userID int, id string) bool {
	ready, ok := m.bosses[userID][id]
	return ok && m.clock().Before(ready)
}

// noteBossFallenLocked starts the lair's cooldown for the owner and the
// owner's party the first time a boss record's boss is seen fallen, whether
// or not its escorts still stand (they are cleared with the group).
func (m *EncountersModule) noteBossFallenLocked(r *record, now time.Time) {
	if !r.Boss || r.cooled || len(r.Foes) == 0 {
		return
	}
	if boss := mobs.GetInstance(r.Foes[0]); boss != nil && boss.Character.Health > 0 {
		return
	}
	r.cooled = true
	ready := now.Add(encounters.BossRespawnSeconds * time.Second)
	holders := []int{r.Owner}
	if p := parties.Get(r.Owner); p != nil {
		holders = p.GetMembers()
		if !p.IsMember(r.Owner) {
			holders = append(holders, r.Owner)
		}
	}
	for _, uid := range holders {
		if m.bosses[uid] == nil {
			m.bosses[uid] = map[string]time.Time{}
		}
		m.bosses[uid][r.comp] = ready
		for id, t := range m.bosses[uid] {
			if !now.Before(t) {
				delete(m.bosses[uid], id) // expired: keep the saved file small
			}
		}
	}
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("encounters: save boss cooldown", "user", r.Owner, "error", err)
	}
	if user := users.GetByUserId(r.Owner); user != nil {
		user.SendText(`<ansi fg="yellow">The lair falls quiet. Nothing more will stir here for about half an hour.</ansi>`)
	}
}

func (r *record) standing() int {
	n := 0
	for _, id := range r.Foes {
		if mob := mobs.GetInstance(id); mob != nil && mob.Character.Health > 0 {
			n++
		}
	}
	return n
}

// participating is true while the owner (alive) stands with a foe, or any
// battle still involves a foe.
func (r *record) participating() bool {
	owner := users.GetByUserId(r.Owner)
	for _, id := range r.Foes {
		mob := mobs.GetInstance(id)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if battle.Engaged(id) {
			return true
		}
		if owner != nil && owner.Character != nil && owner.Character.Health > 0 && owner.Character.RoomId == mob.Character.RoomId {
			return true
		}
	}
	return false
}

// onUserPurged forgets a purged leader's grace and clears their group.
func (m *EncountersModule) onUserPurged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.UserPurged)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, r := range m.active {
		if r.Owner == evt.UserId {
			r.Remove()
			delete(m.active, id)
		}
	}
	_, held := m.graces[evt.UserId]
	_, boss := m.bosses[evt.UserId]
	if !held && !boss {
		return events.Continue
	}
	delete(m.graces, evt.UserId)
	delete(m.bosses, evt.UserId)
	if err := m.saveLocked(); err != nil {
		mudlog.Error("encounters: save after purge", "user", evt.UserId, "error", err)
	}
	return events.Continue
}
