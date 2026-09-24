// Package death is the Phase 25a death module: when a player dies, the death
// costs one level and the player wakes at the church of the last city they
// visited, with their living companions. It owns the settlement registry
// (config), keeps each player's checkpoint up to date, and implements the
// internal/death provider the engine's suicide command hands a death to.
// Everything it stores lives on the character (MiscData), in the user file.
package death

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	domain "github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//go:embed files/*
var files embed.FS

const (
	defaultFallbackRoomID = 18
	defaultVitalsPct      = 50
)

// settings is the parsed module config.
type settings struct {
	registry   domain.Registry
	fallbackID int
	vitalsPct  int
}

func defaultSettings() settings {
	registry, _ := domain.NewRegistry(nil)
	return settings{registry: registry, fallbackID: defaultFallbackRoomID, vitalsPct: defaultVitalsPct}
}

// DeathModule is the registered death provider.
type DeathModule struct {
	plug *plugins.Plugin

	// Seams; the natives are set in init.
	lookupUser    func(userID int) *users.UserRecord
	loadRoom      func(roomID int) *rooms.Room
	moveToRoom    func(userID, roomID int) error
	abandonTravel func(leaderUserID int) error
	abandonCamp   func(leaderUserID int) error
	relocate      func(leaderUserID, roomID int) int
	round         func() uint64
	// Phase 25b resurrection seams.
	keeper         func(room *rooms.Room, mobID int) (name string, ok bool)
	deadCompanions func(leaderUserID int) []company.DeadCompanionView
	raise          func(leaderUserID int, selector string, roomID int) (company.ResurrectionResult, error)

	// mu guards cfg, written at load, and returned. It is a leaf lock.
	mu  sync.Mutex
	cfg settings
	// returned is the round each user was last returned to a church, in
	// memory only: it only has to outlast one round's queued commands.
	returned map[int]uint64
}

// module is the registered instance, for wiring tests.
var module *DeathModule

var _ domain.Provider = (*DeathModule)(nil)

func init() {
	m := newModule()
	m.plug = plugins.New("death", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.plug.ReserveTags(domain.ChurchTag, domain.ShamanTag)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.AddUserCommand("resurrect", m.resurrectCommand, false, false)
	events.RegisterListener(events.RoomChange{}, m.onRoomChange)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	domain.SetProvider(m)
	module = m
}

// newModule returns a module wired to the native world, with default
// settings.
func newModule() *DeathModule {
	return &DeathModule{
		lookupUser: users.GetByUserId,
		loadRoom:   rooms.LoadRoom,
		moveToRoom: func(userID, roomID int) error {
			return rooms.MoveToRoom(userID, roomID)
		},
		abandonTravel:  expedition.AbandonForDeath,
		abandonCamp:    camping.AbandonForDeath,
		relocate:       company.RelocateCompany,
		round:          util.GetRoundCount,
		keeper:         keeperInRoom,
		deadCompanions: company.DeadCompanions,
		raise:          company.ResurrectCompanion,
		cfg:            defaultSettings(),
		returned:       map[int]uint64{},
	}
}

// --- config ---

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

func configInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

// parseSettings reads the module config. A settlement entry that isn't a
// map, or that the registry rejects, is skipped with a warning; a fallback
// room below 1 or a vitals percentage outside 1..100 uses its default.
func parseSettings(get func(string) any) settings {
	s := defaultSettings()
	if id, ok := configInt(get("FallbackRoomId")); ok && id > 0 {
		s.fallbackID = id
	} else if get("FallbackRoomId") != nil {
		mudlog.Warn("death: FallbackRoomId must be a room ID; using the default", "value", get("FallbackRoomId"))
	}
	if pct, ok := configInt(get("RespawnVitalsPct")); ok && pct >= 1 && pct <= 100 {
		s.vitalsPct = pct
	} else if get("RespawnVitalsPct") != nil {
		mudlog.Warn("death: RespawnVitalsPct must be 1..100; using the default", "value", get("RespawnVitalsPct"))
	}
	var entries []domain.Settlement
	list, _ := get("Settlements").([]any)
	for _, raw := range list {
		fields := lowerKeys(raw)
		if fields == nil {
			mudlog.Warn("death: settlement entry is not a map; skipped")
			continue
		}
		zone, _ := fields["zone"].(string)
		kind, _ := fields["kind"].(string)
		roomID, _ := configInt(fields["serviceroomid"])
		mobID, _ := configInt(fields["servicemobid"])
		entries = append(entries, domain.Settlement{Zone: zone, Kind: domain.Kind(kind), ServiceRoomID: roomID, ServiceMobID: mobID})
	}
	registry, errs := domain.NewRegistry(entries)
	for _, err := range errs {
		mudlog.Warn("death: settlement skipped", "error", err)
	}
	s.registry = registry
	return s
}

func (m *DeathModule) load() {
	cfg := parseSettings(m.plug.Config.Get)
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
	for _, s := range cfg.registry.Settlements() {
		room := m.loadRoom(s.ServiceRoomID)
		if room == nil || !room.HasTag(s.ServiceTag()) || room.Zone != s.Zone {
			mudlog.Warn("death: settlement service room is not usable", "zone", s.Zone, "room", s.ServiceRoomID, "tag", s.ServiceTag())
		}
		if s.ServiceMobID == 0 {
			mudlog.Warn("death: settlement has no ServiceMobId; no one there can resurrect", "zone", s.Zone)
		}
	}
	if m.loadRoom(cfg.fallbackID) == nil {
		mudlog.Error("death: fallback church doesn't load; deaths without a checkpoint will wait for repair", "room", cfg.fallbackID)
	}
}

func (m *DeathModule) config() settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// --- checkpoint ---

// validChurch reports whether roomID is a registered city's church: it
// loads, carries the church tag, and is the church its zone is registered
// with.
func (m *DeathModule) validChurch(roomID int) bool {
	room := m.loadRoom(roomID)
	if room == nil || !room.HasTag(domain.ChurchTag) {
		return false
	}
	church, ok := m.config().registry.ChurchFor(room.Zone)
	return ok && church == roomID
}

func (m *DeathModule) roomLoads(roomID int) bool {
	return m.loadRoom(roomID) != nil
}

func checkpoint(c *characters.Character) int {
	id, _ := configInt(c.GetMiscData(domain.CheckpointKey))
	return id
}

func pendingOp(c *characters.Character) string {
	op, _ := c.GetMiscData(domain.PendingKey).(string)
	return op
}

func roomTitle(room *rooms.Room) string {
	if room == nil || room.Title == "" {
		return "the church"
	}
	return room.Title
}

// visit records the church of the city the user is in as their checkpoint,
// and tells them when it changes. Villages and cities without a valid
// church change nothing.
func (m *DeathModule) visit(user *users.UserRecord, roomID int) {
	if user == nil || user.Character == nil {
		return
	}
	room := m.loadRoom(roomID)
	if room == nil {
		return
	}
	church, ok := m.config().registry.ChurchFor(room.Zone)
	if !ok || !m.validChurch(church) || checkpoint(user.Character) == church {
		return
	}
	user.Character.SetMiscData(domain.CheckpointKey, church)
	user.SendText(fmt.Sprintf("Should you fall, you will wake in %s.", roomTitle(m.loadRoom(church))))
}

func (m *DeathModule) onRoomChange(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId <= 0 {
		return events.Continue
	}
	m.visit(m.lookupUser(evt.UserId), evt.ToRoomId)
	return events.Continue
}

func (m *DeathModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	if user := m.lookupUser(evt.UserId); user != nil && user.Character != nil {
		m.visit(user, user.Character.RoomId)
	}
	return events.Continue
}

// --- death ---

// Pending implements domain.Provider.
func (m *DeathModule) Pending(userID int) bool {
	user := m.lookupUser(userID)
	return user != nil && user.Character != nil && pendingOp(user.Character) != ""
}

// JustReturned implements domain.Provider.
func (m *DeathModule) JustReturned(userID int) bool {
	user := m.lookupUser(userID)
	if user == nil || user.Character == nil || user.Character.Health < 1 {
		return false
	}
	m.mu.Lock()
	round, ok := m.returned[userID]
	m.mu.Unlock()
	return ok && m.round() <= round+1
}

// Respawn implements domain.Provider. It runs on the game loop, from the
// suicide command. A new death takes the level and marks the death pending,
// in the character (and so in the same user file); a retry never takes a
// level. Then it closes the leader's journeys, moves them to the church with
// their living companions, and clears the mark. When that can't happen the
// player stays where they fell, at -10 health, so the engine's own death
// triggers retry it.
func (m *DeathModule) Respawn(userID int, newDeath bool) {
	user := m.lookupUser(userID)
	if user == nil || user.Character == nil {
		return
	}
	c := user.Character
	if !newDeath && pendingOp(c) == "" {
		return // nothing owed
	}
	firstAttempt := newDeath
	if firstAttempt {
		from, to := c.LoseLevel()
		op := fmt.Sprintf("death-%d-%d", userID, m.round())
		c.SetMiscData(domain.PendingKey, op)
		mudlog.Info("death: level taken", "user", userID, "op", op, "from", from, "to", to)
		if from > to {
			user.SendText(fmt.Sprintf(`You lose a level (now level <ansi fg="yellow">%d</ansi>).`, to))
		} else {
			user.SendText("You lose what you had learned toward level 2.")
		}
	}
	cfg := m.config()
	dest, ok := domain.Destination(checkpoint(c), cfg.fallbackID, m.validChurch, m.roomLoads)
	if !ok {
		m.hold(user, firstAttempt, "no church can be loaded", nil)
		return
	}
	if err := m.abandonTravel(userID); err != nil {
		m.hold(user, firstAttempt, "the journey couldn't be ended", err)
		return
	}
	if err := m.abandonCamp(userID); err != nil {
		m.hold(user, firstAttempt, "the camp couldn't be left", err)
		return
	}
	c.Aggro = nil
	if err := m.moveToRoom(userID, dest); err != nil || c.RoomId != dest {
		m.hold(user, firstAttempt, "the move to the church failed", err)
		return
	}
	c.Health = max(1, c.HealthMax.Value*cfg.vitalsPct/100)
	c.Mana = c.ManaMax.Value * cfg.vitalsPct / 100
	moved := m.relocate(userID, dest)
	op := pendingOp(c)
	c.SetMiscData(domain.PendingKey, nil)
	m.mu.Lock()
	m.returned[userID] = m.round()
	m.mu.Unlock()
	mudlog.Info("death: returned", "user", userID, "op", op, "room", dest, "companions", moved)

	church := m.loadRoom(dest)
	lines := []string{fmt.Sprintf("You wake before the altar of %s.", roomTitle(church))}
	if moved > 0 {
		lines = append(lines, "Your company is with you.")
	}
	user.SendText(strings.Join(lines, "\n"))
	if church != nil {
		church.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is carried in and laid before the altar.`, c.Name), userID)
	}
	events.AddToQueue(events.CharacterVitalsChanged{UserId: userID})
	user.Command("look")
}

// hold leaves a pending death where it is, at -10 health so the engine
// retries it, and tells the player once.
func (m *DeathModule) hold(user *users.UserRecord, firstAttempt bool, reason string, err error) {
	user.Character.Health = -10
	// Out of any fight, so AutoHeal (which skips fighters) retries too.
	user.Character.Aggro = nil
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})
	if firstAttempt {
		mudlog.Error("death: return to a church is waiting for repair", "user", user.UserId, "reason", reason, "error", err)
		user.SendText("The way back is closed to you. The gods have been told.")
		return
	}
	mudlog.Warn("death: return to a church still waiting", "user", user.UserId, "reason", reason, "error", err)
}
