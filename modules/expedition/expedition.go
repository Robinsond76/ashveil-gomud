// Package expedition owns data-driven terrain travel profiles, durable
// leader-keyed travel sessions, real-time completion timers, load/copyover
// recovery, and the player-facing travel views.
//
// It is the only component that schedules a travel timer or calls
// rooms.MoveToRoom for travel. Travel uses real UTC time and never mutates
// GoMud's global game time or round count. Survival exertion is applied in
// proportional durable checkpoints through internal/survival's company service.
package expedition

import (
	"embed"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const travelUsage = "Usage: travel status | travel resume | travel return"

// Registry is the durable, leader-keyed set of active travel sessions.
type Registry struct {
	Sessions map[int]expedition.TravelSession `yaml:"sessions"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Sessions: map[int]expedition.TravelSession{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Sessions: make(map[int]expedition.TravelSession, len(r.Sessions))}
	for leaderUserID, session := range r.Sessions {
		out.Sessions[leaderUserID] = session
	}
	return out
}

// Store abstracts durable registry persistence so tests can inject failures.
type Store interface {
	Load(*Registry) error
	Save(Registry) error
}

type pluginStore struct{ plug *plugins.Plugin }

func (s pluginStore) Load(registry *Registry) error {
	// ReadBytes discards YAML decode errors, so decode here to prevent
	// unreadable data from becoming an empty, writable registry.
	data, err := s.plug.ReadBytes("expedition")
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
	return s.plug.WriteStruct("expedition", registry)
}

// decodeRegistry parses stored bytes, dropping malformed or unknown entries.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for leaderUserID, session := range wire.Sessions {
		if leaderUserID <= 0 {
			continue
		}
		session.LeaderUserID = leaderUserID
		loaded.Sessions[leaderUserID] = session
	}
	*registry = *loaded
	return nil
}

// Timer is a cancellable scheduled callback.
type Timer interface {
	Stop() bool
}

// Scheduler schedules a one-shot callback after a delay. Tests inject a
// deterministic implementation.
type Scheduler interface {
	AfterFunc(d time.Duration, f func()) Timer
}

type realScheduler struct{}

func (realScheduler) AfterFunc(d time.Duration, f func()) Timer {
	if d < 0 {
		d = 0
	}
	return realTimer{timer: time.AfterFunc(d, f)}
}

type realTimer struct{ timer *time.Timer }

func (r realTimer) Stop() bool { return r.timer.Stop() }

// Mover relocates a user through GoMud's normal room-movement path.
type Mover interface {
	MoveToRoom(userID, roomID int) error
}

type nativeMover struct{}

func (nativeMover) MoveToRoom(userID, roomID int) error {
	return rooms.MoveToRoom(userID, roomID)
}

// MobSpawner spawns and tracks a Combat interruption's encounter mob.
type MobSpawner interface {
	// SpawnHostileEncounter creates mobTemplateID in roomID, immediately
	// commands it to attack leaderUserID, and returns its instance id.
	SpawnHostileEncounter(roomID, mobTemplateID, leaderUserID int) (int, error)
	// EncounterActive reports whether instanceID is still alive and still in
	// roomID. False (including "no such instance," e.g. after a restart
	// that didn't persist it) means the encounter is resolved.
	EncounterActive(instanceID, roomID int) bool
}

type nativeMobSpawner struct{}

func (nativeMobSpawner) SpawnHostileEncounter(roomID, mobTemplateID, leaderUserID int) (int, error) {
	room := rooms.LoadRoom(roomID)
	if room == nil {
		return 0, fmt.Errorf("expedition: room %d is unavailable", roomID)
	}
	mob := mobs.NewMobById(mobs.MobId(mobTemplateID), roomID)
	if mob == nil {
		return 0, fmt.Errorf("expedition: combat encounter mob template %d is unavailable", mobTemplateID)
	}
	mob.Hostile = true
	mob.MaxWander = 0
	room.AddMob(mob.InstanceId)
	mob.Command(fmt.Sprintf("attack @%d", leaderUserID))
	return mob.InstanceId, nil
}

func (nativeMobSpawner) EncounterActive(instanceID, roomID int) bool {
	mob := mobs.GetInstance(instanceID)
	return mob != nil && mob.Character.Health > 0 && mob.Character.RoomId == roomID
}

// Survival is the Phase 4 company service needed by travel.
type Survival interface {
	ApplyCompanyExertion(leaderUserID int, operationID string, cost survival.Exertion) ([]survival.ExertionResult, error)
	CompanyNeeds(leaderUserID int) []survival.MemberNeeds
	Available() error
}

type nativeSurvival struct{}

func (nativeSurvival) ApplyCompanyExertion(leaderUserID int, operationID string, cost survival.Exertion) ([]survival.ExertionResult, error) {
	return survival.ApplyCompanyExertion(leaderUserID, operationID, cost)
}

func (nativeSurvival) CompanyNeeds(leaderUserID int) []survival.MemberNeeds {
	return survival.CompanyNeeds(leaderUserID)
}

func (nativeSurvival) Available() error {
	if !survival.CompanyServiceAvailable() {
		return survival.ErrExertionUnavailable
	}
	return nil
}

// ExpeditionModule owns profiles and durable travel sessions for one plugin.
type ExpeditionModule struct {
	plug       *plugins.Plugin
	store      Store
	clock      func() time.Time
	scheduler  Scheduler
	mover      Mover
	survival   Survival
	rollUint64 func() uint64
	mobSpawner MobSpawner

	// Phase 16 departure multiplier seams; nil uses the native providers.
	weatherIn        func(zone string) (weather.Condition, bool)
	loadBand         func(leaderUserID int) (encumbrance.LoadBand, bool)
	mountDurationPct func(leaderUserID int) int
	roomZone         func(roomID int) string

	profiles        map[string]expedition.TravelProfile
	sessions        map[int]expedition.TravelSession
	timers          map[int]Timer
	timerGeneration map[int]uint64
	loadErr         error

	mu sync.Mutex
}

var (
	_ expedition.StartProvider    = (*ExpeditionModule)(nil)
	_ expedition.ViewProvider     = (*ExpeditionModule)(nil)
	_ expedition.MovementProvider = (*ExpeditionModule)(nil)
	_ expedition.AbandonProvider  = (*ExpeditionModule)(nil)
)

func init() {
	m := &ExpeditionModule{
		plug:            plugins.New("expedition", "1.0"),
		clock:           time.Now,
		scheduler:       realScheduler{},
		mover:           nativeMover{},
		survival:        nativeSurvival{},
		rollUint64:      rand.Uint64,
		mobSpawner:      nativeMobSpawner{},
		profiles:        map[string]expedition.TravelProfile{},
		sessions:        map[int]expedition.TravelSession{},
		timers:          map[int]Timer{},
		timerGeneration: map[int]uint64{},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("travel", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("expedition: save", "error", err)
		}
	})
	expedition.SetStartProvider(m)
	expedition.SetViewProvider(m)
	expedition.SetMovementProvider(m)
	expedition.SetAbandonProvider(m)
}

func (m *ExpeditionModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("expedition: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("expedition: persistence unavailable")
	}
	return nil
}

// save acquires the module lock and persists the current registry.
func (m *ExpeditionModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *ExpeditionModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	registry := Registry{Sessions: m.sessions}
	if err := m.store.Save(registry); err != nil {
		return fmt.Errorf("expedition: save failed; please retry: %w", err)
	}
	return nil
}

func (m *ExpeditionModule) load() {
	if m.store == nil {
		return
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("expedition: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = loaded.Sessions
	if m.sessions == nil {
		m.sessions = map[int]expedition.TravelSession{}
	}
	m.timers = map[int]Timer{}
	m.timerGeneration = map[int]uint64{}
	m.loadErr = nil
	if m.plug != nil {
		m.profiles = m.loadProfiles()
	}
	m.recoverLocked()
}

// loadProfiles parses the module's configured travel profiles.
func (m *ExpeditionModule) loadProfiles() map[string]expedition.TravelProfile {
	if m.plug == nil {
		return map[string]expedition.TravelProfile{}
	}
	return parseProfiles(m.plug.Config.Get("Profiles"))
}

func (m *ExpeditionModule) profile(name string) (expedition.TravelProfile, bool) {
	profile, ok := m.profiles[name]
	return profile, ok
}

// sessionProfile is the session's route profile scaled by the multipliers
// locked onto it at departure (Phase 16). Every read of a session's
// Duration or Exertion goes through here.
func (m *ExpeditionModule) sessionProfile(session expedition.TravelSession) (expedition.TravelProfile, bool) {
	profile, ok := m.profiles[session.ProfileName]
	if !ok {
		return profile, false
	}
	return session.EffectiveProfile(profile), true
}

// parseProfiles normalizes the configured profile list, rejecting malformed
// profiles and duplicate names rather than applying a guess.
func parseProfiles(raw any) map[string]expedition.TravelProfile {
	profiles := map[string]expedition.TravelProfile{}
	list, ok := raw.([]any)
	if !ok {
		return profiles
	}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		name, _ := fields["name"].(string)
		name = strings.TrimSpace(name)
		duration, ok := parseDuration(fields["duration"])
		if !ok {
			continue
		}
		interruption, ok := parseInterruption(fields["interruption"])
		if !ok {
			continue
		}
		profile := expedition.TravelProfile{
			Name:         name,
			Duration:     duration,
			Exertion:     parseExertion(fields["exertion"]),
			Interruption: interruption,
		}
		if err := profile.Validate(); err != nil {
			mudlog.Warn("expedition: invalid travel profile", "name", name, "error", err)
			continue
		}
		if _, exists := profiles[profile.Name]; exists {
			mudlog.Warn("expedition: duplicate travel profile", "name", profile.Name)
			continue
		}
		profiles[profile.Name] = profile
	}
	return profiles
}

// parseInterruption parses the optional, documented interruption shape.
// Absent interruptions are valid; present malformed interruptions invalidate
// the containing profile instead of being silently discarded.
func parseInterruption(raw any) (*expedition.InterruptionProfile, bool) {
	if raw == nil {
		return nil, true
	}
	fields := stringMap(raw)
	if fields == nil {
		return nil, false
	}
	checkpoint := configInt(fields["checkpoint"])
	if checkpoint < 1 || checkpoint >= expedition.CheckpointCount {
		return nil, false
	}
	interruption := &expedition.InterruptionProfile{
		Kind:       expedition.InterruptionKind(configString(fields["kind"])),
		Checkpoint: uint8(checkpoint),
	}
	if err := interruption.Validate(); err != nil {
		return nil, false
	}
	return interruption, true
}

func (m *ExpeditionModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[evt.UserId]
	if !ok {
		return events.Continue
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return events.Continue
	}
	if err := session.ValidateForProfile(profile); err != nil {
		mudlog.Warn("expedition: spawn invalid session", "leader", evt.UserId, "error", err)
		return events.Continue
	}
	if session.State == expedition.Traveling {
		if err := m.syncAndCompleteLocked(evt.UserId); err != nil {
			return events.Continue
		}
	} else if session.State == expedition.Completed {
		m.recoverCompletedLocked(session, profile)
	}
	return events.Continue
}

// stringMap normalizes the map types produced by YAML decoding and lowercases
// keys so profile fields are matched case-insensitively.
func stringMap(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[strings.ToLower(key)] = item
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if name, ok := key.(string); ok {
				out[strings.ToLower(name)] = item
			}
		}
		return out
	default:
		return nil
	}
}

func parseDuration(raw any) (time.Duration, bool) {
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		if d, err := time.ParseDuration(trimmed); err == nil {
			return d, true
		}
		if seconds, err := strconv.Atoi(trimmed); err == nil {
			return time.Duration(seconds) * time.Second, true
		}
	case int:
		return time.Duration(value) * time.Second, true
	case int64:
		return time.Duration(value) * time.Second, true
	case float64:
		return time.Duration(value * float64(time.Second)), true
	}
	return 0, false
}

func parseExertion(raw any) survival.Exertion {
	fields := stringMap(raw)
	if fields == nil {
		return survival.Exertion{}
	}
	return survival.Exertion{
		Hunger:  configInt(fields["hunger"]),
		Thirst:  configInt(fields["thirst"]),
		Fatigue: configInt(fields["fatigue"]),
	}
}

func configInt(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return n
		}
	}
	return 0
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}

func exertionZero(cost survival.Exertion) bool {
	return cost.Hunger == 0 && cost.Thirst == 0 && cost.Fatigue == 0
}

// StartTravel implements expedition.StartProvider. It is called only for
// exits carrying a travel profile.
func (m *ExpeditionModule) StartTravel(req expedition.StartRequest) (bool, error) {
	if err := m.persistenceAvailable(); err != nil {
		return true, err
	}
	if err := m.survival.Available(); err != nil {
		return true, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.profile(req.ProfileName); !ok {
		return true, fmt.Errorf("expedition: unknown travel profile %q", req.ProfileName)
	}
	if _, exists := m.sessions[req.LeaderUserID]; exists {
		// A second start must not replace, duplicate, or reset the session.
		m.sendToLeader(req.LeaderUserID, "You are already travelling.")
		return true, nil
	}
	// A company with a Collapsed member (fatigue 0) can't set out on a route.
	// Ordinary movement is never refused, so nobody is stranded.
	for _, member := range m.companyNeeds(req.LeaderUserID) {
		if member.Needs.Fatigue <= 0 {
			m.sendToLeader(req.LeaderUserID, "Your company is too exhausted to set out. Rest first.")
			return true, nil
		}
	}

	factors := m.departureFactorsLocked(req)
	session := expedition.TravelSession{
		LeaderUserID:      req.LeaderUserID,
		OriginRoomID:      req.OriginRoomID,
		DestinationRoomID: req.DestinationRoomID,
		ExitName:          req.ExitName,
		ProfileName:       req.ProfileName,
		StartedAtUTC:      m.clock().UTC(),
		State:             expedition.Traveling,
		DurationPct:       factors.durationPct,
		ExertionPct:       factors.exertionPct,
		FatiguePct:        factors.fatiguePct,
	}
	if err := session.Validate(); err != nil {
		return true, err
	}
	m.sessions[session.LeaderUserID] = session
	// Persist before scheduling so a crash cannot leave an unsaved journey.
	if err := m.saveLocked(); err != nil {
		delete(m.sessions, session.LeaderUserID)
		return true, err
	}
	m.scheduleLocked(session)
	m.sendToLeader(session.LeaderUserID, m.departureTextLocked(session))
	if line := factors.line(); line != "" {
		m.sendToLeader(session.LeaderUserID, line)
	}
	return true, nil
}

// departureFactors are the Phase 16 multipliers locked onto a journey at
// departure, plus what produced them for the departure line.
type departureFactors struct {
	durationPct, exertionPct, fatiguePct int
	weatherName                          string
	weatherSlows, weatherTires           bool
	loadSlows                            bool
	mountSpeeds                          bool
}

// neutralPct stores 100 as 0 so a neutral session saves exactly like a
// pre-Phase 16 one.
func neutralPct(pct int) int {
	pct = expedition.ClampPct(pct)
	if pct == 100 {
		return 0
	}
	return pct
}

// departureFactorsLocked reads weather (the origin zone, or the destination
// zone when the origin is untracked), the company load band, and the mount.
// Lock order: expedition -> {encumbrance, mount, weather}; none call back.
func (m *ExpeditionModule) departureFactorsLocked(req expedition.StartRequest) departureFactors {
	weatherIn := m.weatherIn
	if weatherIn == nil {
		weatherIn = weather.CurrentCondition
	}
	loadBand := m.loadBand
	if loadBand == nil {
		loadBand = encumbrance.CurrentBand
	}
	mountDuration := m.mountDurationPct
	if mountDuration == nil {
		mountDuration = mount.TravelDurationPct
	}
	roomZone := m.roomZone
	if roomZone == nil {
		roomZone = func(roomID int) string {
			if room := rooms.LoadRoom(roomID); room != nil {
				return room.Zone
			}
			return ""
		}
	}

	f := departureFactors{}
	weatherDuration, weatherExertion := 100, 100
	condition, ok := weatherIn(roomZone(req.OriginRoomID))
	if !ok {
		condition, ok = weatherIn(roomZone(req.DestinationRoomID))
	}
	if ok {
		weatherDuration = orNeutral(condition.TravelDurationPct)
		weatherExertion = orNeutral(condition.ExertionPct)
		f.weatherName = condition.Name
		f.weatherSlows = weatherDuration > 100
		f.weatherTires = weatherExertion > 100
	}
	loadDuration, loadFatigue := 100, 100
	if band, ok := loadBand(req.LeaderUserID); ok {
		loadDuration = orNeutral(band.TravelDurationPct)
		loadFatigue = orNeutral(band.FatiguePct)
		f.loadSlows = loadDuration > 100 || loadFatigue > 100
	}
	mountPct := orNeutral(mountDuration(req.LeaderUserID))
	f.mountSpeeds = mountPct < 100

	product := int64(weatherDuration) * int64(loadDuration) * int64(mountPct)
	f.durationPct = neutralPct(int((product + 5000) / 10000))
	f.exertionPct = neutralPct(weatherExertion)
	f.fatiguePct = neutralPct(loadFatigue)
	return f
}

func orNeutral(pct int) int {
	if pct <= 0 {
		return 100
	}
	return pct
}

// line names the factors that changed the journey, or "".
func (f departureFactors) line() string {
	var slowers []string
	if f.weatherSlows || f.weatherTires {
		name := f.weatherName
		if name == "" {
			name = "the weather"
		}
		slowers = append(slowers, strings.ToUpper(name[:1])+name[1:])
	}
	if f.loadSlows {
		if len(slowers) == 0 {
			slowers = append(slowers, "A heavy load")
		} else {
			slowers = append(slowers, "a heavy load")
		}
	}
	var parts []string
	switch len(slowers) {
	case 1:
		parts = append(parts, slowers[0]+" slows your pace.")
	case 2:
		parts = append(parts, slowers[0]+" and "+slowers[1]+" slow your pace.")
	}
	if f.mountSpeeds {
		parts = append(parts, "Your mount quickens the journey.")
	}
	return strings.Join(parts, " ")
}

// RenderTravelView implements expedition.ViewProvider.
func (m *ExpeditionModule) RenderTravelView(leaderUserID int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[leaderUserID]; !ok {
		return false, nil
	}
	if err := m.syncAndCompleteLocked(leaderUserID); err != nil {
		mudlog.Warn("expedition: view sync", "leader", leaderUserID, "error", err)
	}
	m.sendToLeader(leaderUserID, m.statusTextLocked(leaderUserID))
	return true, nil
}

// MovementBlocked implements expedition.MovementProvider. While a session is
// active the leader cannot use ordinary exits; the refusal reports progress.
func (m *ExpeditionModule) MovementBlocked(leaderUserID int) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[leaderUserID]
	if !ok || (session.State != expedition.Traveling && session.State != expedition.Interrupted) {
		return false, ""
	}
	return true, m.refusalTextLocked(session)
}

func (m *ExpeditionModule) refusalTextLocked(session expedition.TravelSession) string {
	profile, ok := m.sessionProfile(session)
	if !ok {
		return "You are already travelling."
	}
	now := m.clock().UTC()
	progress := session.ProgressAt(now, profile.Duration)
	remaining := session.RemainingAt(now, profile.Duration)
	if session.State == expedition.Interrupted {
		return fmt.Sprintf("A fallen tree blocks the %s route (%d%% complete, %s remaining). Use \"travel resume\" to continue or \"travel return\" to head back.", session.ProfileName, int(progress*100), remaining.Round(time.Second))
	}
	return fmt.Sprintf("You are already travelling (%d%% complete, %s remaining).", int(progress*100), remaining.Round(time.Second))
}

// Sync applies any earned survival checkpoint for a leader and persists it.
func (m *ExpeditionModule) Sync(leaderUserID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncAndCompleteLocked(leaderUserID)
}

func (m *ExpeditionModule) syncAndCompleteLocked(leaderUserID int) error {
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return nil
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return fmt.Errorf("expedition: unknown travel profile %q", session.ProfileName)
	}
	if err := session.ValidateForProfile(profile); err != nil {
		return err
	}
	if session.State == expedition.Cancelled || session.State == expedition.Completed {
		return nil
	}
	if session.State == expedition.Interrupted {
		// Paused travel has no active elapsed time and must never charge survival
		// exertion merely because a view, spawn, or stale timer is processed.
		m.stopTimerLocked(leaderUserID)
		return nil
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		return err
	}
	session = m.sessions[leaderUserID]
	if session.State == expedition.Traveling {
		if session.InterruptionDue(m.clock().UTC(), profile) {
			if err := m.interruptLocked(session); err != nil {
				return err
			}
			session = m.sessions[leaderUserID]
		}
	}
	if session.State == expedition.Traveling {
		if session.ActiveElapsedAt(m.clock().UTC(), profile.Duration) >= profile.Duration {
			m.completeLocked(session)
		} else {
			m.scheduleLocked(session)
		}
	} else if session.State == expedition.Interrupted {
		m.stopTimerLocked(leaderUserID)
	}
	return nil
}

// syncLocked applies the incremental exertion owed since the last persisted
// checkpoint. A failed survival or checkpoint write blocks progression rather
// than charging twice.
func (m *ExpeditionModule) syncLocked(leaderUserID int) error {
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return nil
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return fmt.Errorf("expedition: unknown travel profile %q", session.ProfileName)
	}
	now := m.clock().UTC()
	pending, due, err := session.PrepareExertion(now, profile)
	if err != nil {
		return err
	}
	if !due {
		return nil
	}
	if err := m.survival.Available(); err != nil {
		return err
	}
	session.PendingExertion = &pending
	m.sessions[leaderUserID] = session
	if err := m.saveLocked(); err != nil {
		return err
	}
	if _, err := m.survival.ApplyCompanyExertion(leaderUserID, pending.OperationID, pending.Cost); err != nil {
		return err
	}
	session.LastExertionCheckpoint = pending.Checkpoint
	session.PendingExertion = nil
	m.sessions[leaderUserID] = session
	if err := m.saveLocked(); err != nil {
		// Keep the advanced in-memory checkpoint so a same-process retry does
		// not charge the same checkpoint twice.
		return err
	}
	return m.syncLocked(leaderUserID)
}

func (m *ExpeditionModule) nextBoundaryDelayLocked(session expedition.TravelSession, profile expedition.TravelProfile) time.Duration {
	target := profile.Duration
	if !session.InterruptionTriggered && profile.Interruption != nil {
		target = profile.Duration * time.Duration(profile.Interruption.Checkpoint) / expedition.CheckpointCount
	}
	delay := target - session.ActiveElapsedAt(m.clock().UTC(), profile.Duration)
	if delay < 0 {
		return 0
	}
	return delay
}

func (m *ExpeditionModule) scheduleLocked(session expedition.TravelSession) {
	profile, ok := m.sessionProfile(session)
	if !ok {
		return
	}
	if session.State != expedition.Traveling {
		m.stopTimerLocked(session.LeaderUserID)
		return
	}
	remaining := m.nextBoundaryDelayLocked(session, profile)
	leaderUserID := session.LeaderUserID
	if m.timerGeneration == nil {
		m.timerGeneration = map[int]uint64{}
	}
	m.timerGeneration[leaderUserID]++
	generation := m.timerGeneration[leaderUserID]
	m.stopTimerLocked(leaderUserID)
	m.timers[leaderUserID] = m.scheduler.AfterFunc(remaining, func() {
		m.onTimer(leaderUserID, generation)
	})
}

func (m *ExpeditionModule) stopTimerLocked(leaderUserID int) {
	if timer, ok := m.timers[leaderUserID]; ok {
		timer.Stop()
		delete(m.timers, leaderUserID)
	}
}

func (m *ExpeditionModule) onTimer(leaderUserID int, generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.timerGeneration[leaderUserID] != generation {
		return
	}
	if _, ok := m.timers[leaderUserID]; !ok {
		return
	}
	delete(m.timers, leaderUserID)
	session, ok := m.sessions[leaderUserID]
	if !ok || session.State != expedition.Traveling {
		return
	}
	if err := m.syncAndCompleteLocked(leaderUserID); err != nil {
		mudlog.Warn("expedition: timer sync", "leader", leaderUserID, "error", err)
	}
}

func (m *ExpeditionModule) interruptLocked(session expedition.TravelSession) error {
	profile, ok := m.sessionProfile(session)
	if !ok {
		return fmt.Errorf("expedition: unknown travel profile %q", session.ProfileName)
	}
	if profile.Interruption != nil && len(profile.Interruption.Kinds) > 0 {
		resolvedKind, err := profile.Interruption.ResolveKind(m.rollUint64())
		if err != nil {
			return err
		}
		resolved := *profile.Interruption
		resolved.Kind = resolvedKind
		resolved.Kinds = nil
		profile.Interruption = &resolved
	}
	now := m.clock().UTC()
	candidate, err := session.Interrupt(now, profile)
	if err != nil {
		return err
	}
	if candidate.Interruption != nil && candidate.Interruption.Kind == expedition.Combat {
		instanceID, spawnErr := m.mobSpawner.SpawnHostileEncounter(candidate.OriginRoomID, profile.Interruption.CombatMobID, candidate.LeaderUserID)
		if spawnErr != nil {
			mudlog.Warn("expedition: combat encounter spawn failed", "leader", candidate.LeaderUserID, "error", spawnErr)
		} else {
			candidate.Interruption.CombatMobInstanceId = instanceID
		}
	}
	original := session
	m.sessions[session.LeaderUserID] = candidate
	if err := m.saveLocked(); err != nil {
		m.sessions[session.LeaderUserID] = original
		return err
	}
	m.stopTimerLocked(session.LeaderUserID)
	m.sendToLeader(session.LeaderUserID, m.interruptionTextLocked(candidate))
	return nil
}

func (m *ExpeditionModule) interruptionTextLocked(session expedition.TravelSession) string {
	if session.Interruption == nil {
		return ""
	}
	return expedition.InterruptionText(session.Interruption.Kind, session.ProfileName)
}

// combatEncounterActiveLocked reports whether an Interrupted session's
// Combat encounter mob is still alive and present, blocking resume/return.
// Every other interruption kind, or a Combat firing whose spawn failed
// (CombatMobInstanceId stays 0), reports false.
func (m *ExpeditionModule) combatEncounterActiveLocked(session expedition.TravelSession) bool {
	if session.Interruption == nil || session.Interruption.Kind != expedition.Combat {
		return false
	}
	if session.Interruption.CombatMobInstanceId == 0 {
		return false
	}
	return m.mobSpawner.EncounterActive(session.Interruption.CombatMobInstanceId, session.OriginRoomID)
}

// completeLocked applies the final earned checkpoint, persists Completed, then
// moves the leader exactly once and cleans up after verifying the destination.
func (m *ExpeditionModule) completeLocked(session expedition.TravelSession) {
	profile, ok := m.sessionProfile(session)
	if !ok {
		mudlog.Error("expedition: complete with unknown profile", "leader", session.LeaderUserID, "profile", session.ProfileName)
		return
	}
	if err := m.syncLocked(session.LeaderUserID); err != nil {
		mudlog.Warn("expedition: sync before completion", "leader", session.LeaderUserID, "error", err)
		return
	}
	session, ok = m.sessions[session.LeaderUserID]
	if !ok || session.State != expedition.Traveling {
		return
	}
	completed, err := session.Transition(expedition.Completed)
	if err != nil {
		return
	}
	m.sessions[completed.LeaderUserID] = completed
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("expedition: persist completion", "leader", completed.LeaderUserID, "error", err)
		return
	}
	m.moveAndFinishLocked(completed, profile)
}

// moveAndFinishLocked performs the single destination move and, once verified,
// announces arrival and removes the terminal record.
func (m *ExpeditionModule) moveAndFinishLocked(session expedition.TravelSession, profile expedition.TravelProfile) {
	if err := m.mover.MoveToRoom(session.LeaderUserID, session.DestinationRoomID); err != nil {
		mudlog.Warn("expedition: move on completion", "leader", session.LeaderUserID, "error", err)
		return
	}
	user := users.GetByUserId(session.LeaderUserID)
	if user == nil || user.Character.RoomId != session.DestinationRoomID {
		// Retain the Completed record so recovery can verify or retry later.
		return
	}
	m.commandCompanionsToFollow(session)
	m.sendToLeader(session.LeaderUserID, m.arrivalTextLocked(session))
	delete(m.sessions, session.LeaderUserID)
	m.stopTimerLocked(session.LeaderUserID)
	if err := m.saveLocked(); err != nil {
		mudlog.Warn("expedition: persist arrival cleanup", "leader", session.LeaderUserID, "error", err)
	}
}

// recoverLocked reconciles persisted sessions with the current clock on module
// load, which also runs after a copyover restore.
func (m *ExpeditionModule) recoverLocked() {
	for leaderUserID, session := range m.sessions {
		profile, ok := m.sessionProfile(session)
		if !ok {
			mudlog.Warn("expedition: recovery unknown profile", "leader", leaderUserID, "profile", session.ProfileName)
			continue
		}
		if err := session.ValidateForProfile(profile); err != nil {
			mudlog.Warn("expedition: recovery invalid session", "leader", leaderUserID, "error", err)
			continue
		}
		switch session.State {
		case expedition.Traveling:
			if err := m.syncAndCompleteLocked(leaderUserID); err != nil {
				mudlog.Warn("expedition: recovery sync", "leader", leaderUserID, "error", err)
			}
		case expedition.Completed:
			m.recoverCompletedLocked(session, profile)
		case expedition.Interrupted:
			m.stopTimerLocked(leaderUserID)
		case expedition.Cancelled:
			m.stopTimerLocked(leaderUserID)
			delete(m.sessions, leaderUserID)
			if err := m.saveLocked(); err != nil {
				// The durable Cancelled record remains for a later cleanup retry.
				m.sessions[leaderUserID] = session
				mudlog.Warn("expedition: persist cancelled cleanup", "leader", leaderUserID, "error", err)
			}
		default:
			mudlog.Warn("expedition: recovery unknown state", "leader", leaderUserID, "state", session.State)
		}
	}
}

// recoverCompletedLocked verifies a terminal record's location. Destination
// means cleanup only, origin means one retry of the move, and any other
// location is retained for operator repair rather than guessed.
func (m *ExpeditionModule) recoverCompletedLocked(session expedition.TravelSession, profile expedition.TravelProfile) {
	user := users.GetByUserId(session.LeaderUserID)
	if user == nil {
		return
	}
	switch user.Character.RoomId {
	case session.DestinationRoomID:
		delete(m.sessions, session.LeaderUserID)
		m.stopTimerLocked(session.LeaderUserID)
		if err := m.saveLocked(); err != nil {
			mudlog.Warn("expedition: persist recovered cleanup", "leader", session.LeaderUserID, "error", err)
		}
	case session.OriginRoomID:
		m.moveAndFinishLocked(session, profile)
	default:
		mudlog.Warn("expedition: completed session at unexpected room", "leader", session.LeaderUserID, "room", user.Character.RoomId)
	}
}

// commandCompanionsToFollow uses native charm movement so current company
// companions follow the leader through the completion move.
func (m *ExpeditionModule) commandCompanionsToFollow(session expedition.TravelSession) {
	origin := rooms.LoadRoom(session.OriginRoomID)
	if origin == nil {
		return
	}
	for _, instanceID := range origin.GetMobs(rooms.FindCharmed) {
		mob := mobs.GetInstance(instanceID)
		if mob == nil {
			continue
		}
		if mob.Character.RoomId != session.OriginRoomID {
			continue
		}
		if mob.Character.IsCharmed(session.LeaderUserID) {
			mob.Command(session.ExitName)
		}
	}
}

func (m *ExpeditionModule) sendToLeader(leaderUserID int, text string) {
	if text == "" {
		return
	}
	if user := users.GetByUserId(leaderUserID); user != nil {
		user.SendText(text)
	}
}

func (m *ExpeditionModule) departureTextLocked(session expedition.TravelSession) string {
	profile, ok := m.sessionProfile(session)
	if !ok {
		return ""
	}
	return fmt.Sprintf("You lead your company from %s toward %s.\nRoute: %s\nEstimated travel: %s",
		roomTitle(session.OriginRoomID), roomTitle(session.DestinationRoomID), profile.Name, profile.Duration)
}

func (m *ExpeditionModule) arrivalTextLocked(session expedition.TravelSession) string {
	return fmt.Sprintf("You have reached %s.", roomTitle(session.DestinationRoomID))
}

// statusTextLocked renders the current session, progress, remaining time, and
// company needs. The module lock must be held.
func (m *ExpeditionModule) statusTextLocked(leaderUserID int) string {
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return "You are not travelling."
	}
	if session.State == expedition.Cancelled || session.State == expedition.Completed {
		return "You are not travelling."
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return fmt.Sprintf("You are travelling via an unknown route (%s).", session.ProfileName)
	}
	now := m.clock().UTC()
	progress := session.ProgressAt(now, profile.Duration)
	remaining := session.RemainingAt(now, profile.Duration)
	lines := []string{fmt.Sprintf("Travelling from %s to %s via the %s route.", roomTitle(session.OriginRoomID), roomTitle(session.DestinationRoomID), profile.Name)}
	if session.State == expedition.Interrupted {
		lines = append(lines,
			fmt.Sprintf("Paused at the fallen tree obstruction: %d%% complete (%s active travel remaining).", int(progress*100), remaining.Round(time.Second)),
			"Use \"travel resume\" to continue or \"travel return\" to head back.")
	} else {
		lines = append(lines, fmt.Sprintf("Progress: %d%% (remaining %s)", int(progress*100), remaining.Round(time.Second)))
	}
	lines = append(lines, "Company:")
	for _, member := range m.companyNeeds(leaderUserID) {
		lines = append(lines, fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			member.Name,
			member.Needs.Hunger, survival.HungerLabel(member.Needs.Hunger),
			member.Needs.Thirst, survival.ThirstLabel(member.Needs.Thirst),
			member.Needs.Fatigue, survival.FatigueLabel(member.Needs.Fatigue)))
	}
	return strings.Join(lines, "\n")
}

func (m *ExpeditionModule) companyNeeds(leaderUserID int) []survival.MemberNeeds {
	if m.survival == nil {
		return nil
	}
	return m.survival.CompanyNeeds(leaderUserID)
}

func roomTitle(roomID int) string {
	if room := rooms.LoadRoom(roomID); room != nil && room.Title != "" {
		return room.Title
	}
	return fmt.Sprintf("room #%d", roomID)
}

func (m *ExpeditionModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[leaderUserID]; !ok {
		return "You are not travelling."
	}
	if err := m.syncAndCompleteLocked(leaderUserID); err != nil {
		mudlog.Warn("expedition: status sync", "leader", leaderUserID, "error", err)
	}
	return m.statusTextLocked(leaderUserID)
}

// resume resolves a valid interrupted session and restarts its active timer.
func (m *ExpeditionModule) resume(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return "You are not travelling."
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return "Unable to resume travel: route profile is unavailable."
	}
	if session.State != expedition.Interrupted {
		if err := session.ValidateForProfile(profile); err != nil {
			return "Unable to resume travel: the paused journey record is invalid."
		}
		return "Unable to resume travel: travel is not currently interrupted."
	}
	if m.combatEncounterActiveLocked(session) {
		return "You can't do that while you're still fighting!"
	}
	resumed, err := session.ResumeForProfile(m.clock().UTC(), profile)
	if err != nil {
		return "Unable to resume travel: the paused journey record is invalid."
	}
	m.sessions[leaderUserID] = resumed
	if err := m.saveLocked(); err != nil {
		m.sessions[leaderUserID] = session
		return err.Error()
	}
	m.scheduleLocked(resumed)
	return fmt.Sprintf("You resume travel along the %s route.", profile.Name)
}

// returnToOrigin resolves an interrupted session as Cancelled and removes it
// only after the cleanup write succeeds. The terminal record is retained when
// cleanup persistence fails so recovery can retry without moving the leader.
func (m *ExpeditionModule) returnToOrigin(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return "You are not travelling."
	}
	profile, ok := m.sessionProfile(session)
	if !ok {
		return "Unable to return: route profile is unavailable."
	}
	if session.State != expedition.Interrupted {
		if err := session.ValidateForProfile(profile); err != nil {
			return "Unable to return: the paused journey record is invalid."
		}
		return "Unable to return: travel is not currently interrupted."
	}
	if m.combatEncounterActiveLocked(session) {
		return "You can't do that while you're still fighting!"
	}
	cancelled, err := session.ReturnForProfile(profile)
	if err != nil {
		return "Unable to return: the paused journey record is invalid."
	}
	m.sessions[leaderUserID] = cancelled
	if err := m.saveLocked(); err != nil {
		m.sessions[leaderUserID] = session
		return err.Error()
	}
	m.stopTimerLocked(leaderUserID)
	delete(m.sessions, leaderUserID)
	if err := m.saveLocked(); err != nil {
		m.sessions[leaderUserID] = cancelled
		mudlog.Warn("expedition: persist return cleanup", "leader", leaderUserID, "error", err)
	}
	return fmt.Sprintf("You return to %s; the journey is cancelled.", roomTitle(cancelled.OriginRoomID))
}

// AbandonForDeath implements expedition.AbandonProvider (Phase 25a). The
// leader died, so their journey ends, in any state, in one save and without
// moving anyone: a completed record would otherwise move them from the church
// to the destination. A travel encounter's mob stays where it is. A failed
// save keeps the session and its timer. Travel data that couldn't be read
// may hold a journey this module doesn't know about, which would move the
// leader once repaired, so that is an error too.
func (m *ExpeditionModule) AbandonForDeath(leaderUserID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	session, ok := m.sessions[leaderUserID]
	if !ok {
		return nil
	}
	delete(m.sessions, leaderUserID)
	if err := m.saveLocked(); err != nil {
		m.sessions[leaderUserID] = session
		return err
	}
	m.stopTimerLocked(leaderUserID)
	return nil
}

func (m *ExpeditionModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	if len(args) == 0 {
		user.SendText(travelUsage)
		return true, nil
	}
	switch args[0] {
	case "status":
		user.SendText(m.status(user.UserId))
	case "resume":
		user.SendText(m.resume(user.UserId))
	case "return":
		user.SendText(m.returnToOrigin(user.UserId))
	default:
		user.SendText(travelUsage)
	}
	return true, nil
}
