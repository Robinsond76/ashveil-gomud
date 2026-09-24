// Package camping owns durable leader-keyed campsites, real-time rest
// timers, room-tag eligibility, load/copyover recovery, the player-facing
// camp commands and views, and the survival rest-recovery seam.
//
// It is the only component that schedules a rest completion timer or calls
// the survival company rest service. Rest uses real UTC time and never
// mutates GoMud's global game time or round count.
package camping

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/events"
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

const campUsage = "Usage: camp | camp status | camp fire | camp rest | camp break | camp sharpen [status | auto on|off]"
const defaultRoomTag = "camping"

// Registry is the durable, leader-keyed set of active camps plus which
// completed rests have already had their survival recovery applied.
type Registry struct {
	Camps           map[int]camping.Camp `yaml:"camps"`
	RecoveryApplied map[int]bool         `yaml:"recovery_applied,omitempty"`
	// Phase 16 inn stays: the stay itself, whether its survival recovery has
	// been applied, and whether its Well Rested buff is still owed (granted
	// on the game loop, never from a timer).
	Stays              map[int]camping.InnStay `yaml:"stays,omitempty"`
	InnRecoveryApplied map[int]bool            `yaml:"inn_recovery_applied,omitempty"`
	WellRestedPending  map[int]bool            `yaml:"well_rested_pending,omitempty"`
	// Phase 23a: a completed camp rest whose Rested tier is still owed, and
	// tiers owed to companions that had no live mob at the grant, by
	// leader, then companion ID.
	RestedPending map[int]bool                      `yaml:"rested_pending,omitempty"`
	Owed          map[int]map[int]camping.OwedGrant `yaml:"owed,omitempty"`
	// Phase 23b: leaders whose company sharpens its blades when a camp
	// rest's Rested grant is made.
	AutoSharpen map[int]bool `yaml:"auto_sharpen,omitempty"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		Camps:              map[int]camping.Camp{},
		RecoveryApplied:    map[int]bool{},
		Stays:              map[int]camping.InnStay{},
		InnRecoveryApplied: map[int]bool{},
		WellRestedPending:  map[int]bool{},
		RestedPending:      map[int]bool{},
		Owed:               map[int]map[int]camping.OwedGrant{},
		AutoSharpen:        map[int]bool{},
	}
}

func cloneOwed(in map[int]map[int]camping.OwedGrant) map[int]map[int]camping.OwedGrant {
	out := make(map[int]map[int]camping.OwedGrant, len(in))
	for leaderUserID, byCompanion := range in {
		inner := make(map[int]camping.OwedGrant, len(byCompanion))
		for companionID, grant := range byCompanion {
			inner[companionID] = grant
		}
		out[leaderUserID] = inner
	}
	return out
}

func cloneBools(in map[int]bool) map[int]bool {
	out := make(map[int]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{
		Camps:              make(map[int]camping.Camp, len(r.Camps)),
		RecoveryApplied:    cloneBools(r.RecoveryApplied),
		Stays:              make(map[int]camping.InnStay, len(r.Stays)),
		InnRecoveryApplied: cloneBools(r.InnRecoveryApplied),
		WellRestedPending:  cloneBools(r.WellRestedPending),
		RestedPending:      cloneBools(r.RestedPending),
		Owed:               cloneOwed(r.Owed),
		AutoSharpen:        cloneBools(r.AutoSharpen),
	}
	for leaderUserID, camp := range r.Camps {
		out.Camps[leaderUserID] = camp
	}
	for leaderUserID, stay := range r.Stays {
		out.Stays[leaderUserID] = stay
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
	data, err := s.plug.ReadBytes("camping")
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
	return s.plug.WriteStruct("camping", registry)
}

// decodeRegistry parses stored bytes, dropping only entries keyed by an
// invalid leader ID. Invalid camp content is retained for operator repair by
// the module's recovery pass rather than dropped here.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for leaderUserID, camp := range wire.Camps {
		if leaderUserID <= 0 {
			continue
		}
		camp.LeaderUserID = leaderUserID
		loaded.Camps[leaderUserID] = camp
	}
	for leaderUserID, applied := range wire.RecoveryApplied {
		if leaderUserID <= 0 {
			continue
		}
		loaded.RecoveryApplied[leaderUserID] = applied
	}
	for leaderUserID, stay := range wire.Stays {
		if leaderUserID <= 0 {
			continue
		}
		stay.LeaderUserID = leaderUserID
		loaded.Stays[leaderUserID] = stay
	}
	for leaderUserID, applied := range wire.InnRecoveryApplied {
		if leaderUserID > 0 && applied {
			loaded.InnRecoveryApplied[leaderUserID] = true
		}
	}
	for leaderUserID, pending := range wire.WellRestedPending {
		if leaderUserID > 0 && pending {
			loaded.WellRestedPending[leaderUserID] = true
		}
	}
	for leaderUserID, pending := range wire.RestedPending {
		if leaderUserID > 0 && pending {
			loaded.RestedPending[leaderUserID] = true
		}
	}
	for leaderUserID, on := range wire.AutoSharpen {
		if leaderUserID > 0 && on {
			loaded.AutoSharpen[leaderUserID] = true
		}
	}
	for leaderUserID, byCompanion := range wire.Owed {
		if leaderUserID <= 0 {
			continue
		}
		for companionID, grant := range byCompanion {
			if companionID <= 0 || !grant.Valid() {
				continue
			}
			if loaded.Owed[leaderUserID] == nil {
				loaded.Owed[leaderUserID] = map[int]camping.OwedGrant{}
			}
			loaded.Owed[leaderUserID][companionID] = grant
		}
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

// Survival is the Phase 4/7 company rest-recovery seam needed by camping.
type Survival interface {
	ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]survival.ExertionResult, error)
	CompanyNeeds(leaderUserID int) []survival.MemberNeeds
	Available() error
}

type nativeSurvival struct{}

func (nativeSurvival) ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]survival.ExertionResult, error) {
	return survival.ApplyCompanyRestRecovery(leaderUserID, operationID, fatigue)
}

func (nativeSurvival) CompanyNeeds(leaderUserID int) []survival.MemberNeeds {
	return survival.CompanyNeeds(leaderUserID)
}

func (nativeSurvival) Available() error {
	if !survival.CompanyServiceAvailable() {
		return survival.ErrRestUnavailable
	}
	return nil
}

// CampingModule owns durable leader-keyed camps for one plugin.
type CampingModule struct {
	plug      *plugins.Plugin
	store     Store
	clock     func() time.Time
	scheduler Scheduler
	survival  Survival

	camps           map[int]camping.Camp
	recoveryApplied map[int]bool
	timers          map[int]Timer
	timerGeneration map[int]uint64
	loadErr         error

	// Phase 16 inn stays, with their own timers so breaking an idle camp
	// elsewhere never stops a stay's timer.
	stays              map[int]camping.InnStay
	innRecoveryApplied map[int]bool
	wellRestedPending  map[int]bool
	restedPending      map[int]bool
	owed               map[int]map[int]camping.OwedGrant
	autoSharpen        map[int]bool
	innTimers          map[int]Timer
	innTimerGeneration map[int]uint64
	innCfg             innSettings
	innCfgLoaded       bool

	// Seams; nil uses the native implementation.
	weatherIn   func(zone string) (weather.Condition, bool)
	lookupUser  func(userID int) *users.UserRecord
	companySize func(leaderUserID int) int
	// companionsOf returns the leader's live companion characters by
	// companion ID, and every rostered companion ID.
	companionsOf func(leaderUserID int) (map[int]*characters.Character, []int)
	grantBuff    func(c *characters.Character, buffID, rounds int) error
	removeBuff   func(c *characters.Character, buffID int)
	hasBuff      func(c *characters.Character, buffID int) bool
	roundSeconds func() int
	travelling   func(leaderUserID int) bool

	mu sync.Mutex

	// litRooms is a snapshot of rooms with a lit campfire, read by the
	// rooms light-fixture query under litMu only (lock order: mu, then litMu).
	litMu    sync.RWMutex
	litRooms map[int]bool
}

var (
	_ camping.ViewProvider     = (*CampingModule)(nil)
	_ camping.MovementProvider = (*CampingModule)(nil)
	_ camping.AbandonProvider  = (*CampingModule)(nil)
)

func init() {
	m := &CampingModule{
		plug:            plugins.New("camping", "1.0"),
		clock:           time.Now,
		scheduler:       realScheduler{},
		survival:        nativeSurvival{},
		camps:           map[int]camping.Camp{},
		recoveryApplied: map[int]bool{},
		timers:          map[int]Timer{},
		timerGeneration: map[int]uint64{},
	}
	m.resetInnState()
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("camp", m.userCommand, false, false)
	m.plug.AddUserCommand("inn", m.innCommand, false, false)
	m.plug.AddUserCommand("sharpen", m.sharpenCommand, false, false)
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("camping: save", "error", err)
		}
	})
	camping.SetViewProvider(m)
	camping.SetMovementProvider(m)
	camping.SetAbandonProvider(m)
	rooms.RegisterLightFixture(m.RoomHasLitFire)
	// A lit campfire also warms its room (Phase 15).
	climate.RegisterHeatSource(m.RoomHasLitFire)
}

// RoomHasLitFire reports whether a lit campfire burns in roomID, which
// lights that room for everyone (Phase 14). It reads a snapshot guarded by
// its own RWMutex, never m.mu, so look and combat never wait on a camping
// save or timer that holds m.mu.
func (m *CampingModule) RoomHasLitFire(roomID int) bool {
	m.litMu.RLock()
	defer m.litMu.RUnlock()
	return m.litRooms[roomID]
}

// refreshLitRoomsLocked rebuilds the lit-campfire snapshot from m.camps.
// Callers hold m.mu; it is deferred after every m.mu section so the snapshot
// always matches the final (possibly reverted) camp state.
func (m *CampingModule) refreshLitRoomsLocked() {
	lit := map[int]bool{}
	for _, camp := range m.camps {
		if camp.FireLit {
			lit[camp.RoomID] = true
		}
	}
	m.litMu.Lock()
	m.litRooms = lit
	m.litMu.Unlock()
}

func (m *CampingModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("camping: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("camping: persistence unavailable")
	}
	return nil
}

// save acquires the module lock and persists the current registry.
func (m *CampingModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	return m.saveLocked()
}

func (m *CampingModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	registry := Registry{
		Camps:              m.camps,
		RecoveryApplied:    m.recoveryApplied,
		Stays:              m.stays,
		InnRecoveryApplied: m.innRecoveryApplied,
		WellRestedPending:  m.wellRestedPending,
		RestedPending:      m.restedPending,
		Owed:               m.owed,
		AutoSharpen:        m.autoSharpen,
	}
	if err := m.store.Save(registry); err != nil {
		return fmt.Errorf("camping: save failed; please retry: %w", err)
	}
	return nil
}

func (m *CampingModule) load() {
	if m.store == nil {
		return
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("camping: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	m.camps = loaded.Camps
	if m.camps == nil {
		m.camps = map[int]camping.Camp{}
	}
	m.recoveryApplied = loaded.RecoveryApplied
	if m.recoveryApplied == nil {
		m.recoveryApplied = map[int]bool{}
	}
	m.timers = map[int]Timer{}
	m.timerGeneration = map[int]uint64{}
	m.resetInnState()
	if loaded.Stays != nil {
		m.stays = loaded.Stays
	}
	if loaded.InnRecoveryApplied != nil {
		m.innRecoveryApplied = loaded.InnRecoveryApplied
	}
	if loaded.WellRestedPending != nil {
		m.wellRestedPending = loaded.WellRestedPending
	}
	if loaded.RestedPending != nil {
		m.restedPending = loaded.RestedPending
	}
	if loaded.Owed != nil {
		m.owed = loaded.Owed
	}
	if loaded.AutoSharpen != nil {
		m.autoSharpen = loaded.AutoSharpen
	}
	if m.plug != nil {
		m.innCfg = parseInnSettings(m.plug.Config.Get)
		m.innCfgLoaded = true
	}
	m.loadErr = nil
	m.recoverLocked()
	m.recoverStaysLocked()
}

// roomTag returns the configured eligibility tag, defaulting when unset or
// malformed rather than guessing an alternate value.
func (m *CampingModule) roomTag() string {
	if m.plug == nil {
		return defaultRoomTag
	}
	tag := strings.TrimSpace(configString(m.plug.Config.Get("RoomTag")))
	if tag == "" {
		return defaultRoomTag
	}
	return tag
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}

func roomEligible(room *rooms.Room, tag string) bool {
	if room == nil {
		return false
	}
	for _, t := range room.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// restOperationID derives a deterministic rest-recovery operation ID from
// stable leader/room/start identity, so recovery replay after a crash or
// restart cannot restore fatigue twice.
func restOperationID(camp camping.Camp) string {
	return fmt.Sprintf("camp-rest-%d-%d-%d", camp.LeaderUserID, camp.RoomID, camp.Rest.StartedAtUTC.UnixNano())
}

// establish creates a durable camp in an eligible room, refusing an
// ineligible room or a duplicate camp without any persistence change.
func (m *CampingModule) establish(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if !roomEligible(room, m.roomTag()) {
		return "There is nowhere here to make camp."
	}
	if _, exists := m.camps[user.UserId]; exists {
		return "You already have a camp. Use \"camp status\" to check it."
	}
	camp, err := camping.Established(user.UserId, room.RoomId)
	if err != nil {
		return "You can't make camp here."
	}
	m.camps[user.UserId] = camp
	if err := m.saveLocked(); err != nil {
		delete(m.camps, user.UserId)
		return err.Error()
	}
	return "You make camp here."
}

// lightFire lights the leader's camp fire. The leader must be at the camp.
func (m *CampingModule) lightFire(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	camp, ok := m.camps[user.UserId]
	if !ok {
		return "You have no camp here. Use \"camp\" to make one."
	}
	if camp.RoomID != room.RoomId {
		return "Your camp is not here."
	}
	lit, err := camp.LightFire()
	if err != nil {
		if errors.Is(err, camping.ErrFireAlreadyLit) {
			return "Your campfire is already lit."
		}
		return "You can't light a fire here."
	}
	m.camps[user.UserId] = lit
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = camp
		return err.Error()
	}
	return "You light a crackling campfire."
}

// startRest begins the one configured real-time rest session. The leader
// must be at a lit camp, and survival must be available before any change is
// made.
func (m *CampingModule) startRest(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if err := m.survival.Available(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	camp, ok := m.camps[user.UserId]
	if !ok {
		return "You have no camp here. Use \"camp\" to make one."
	}
	if camp.RoomID != room.RoomId {
		return "Your camp is not here."
	}
	if stay, ok := m.stays[user.UserId]; ok && stay.Resting() {
		return "You are already resting at the inn."
	}
	resting, err := camp.StartRest(m.clock().UTC())
	if err != nil {
		switch {
		case errors.Is(err, camping.ErrFireNotLit):
			return "You need a lit campfire to rest. Use \"camp fire\" first."
		case errors.Is(err, camping.ErrRestAlreadyStarted):
			return "You are already resting."
		case errors.Is(err, camping.ErrRestAlreadyCompleted):
			return "Your company has already rested at this camp."
		}
		return "You can't rest here."
	}
	// Phase 16: the weather at the camp scales its recovery, locked now.
	recovery, condition, scaled := m.campRecovery(room)
	rest := *resting.Rest
	rest.Recovery = recovery
	resting.Rest = &rest
	m.camps[user.UserId] = resting
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = camp
		return err.Error()
	}
	m.scheduleLocked(resting)
	text := fmt.Sprintf("You settle in by the fire to rest. (%s)", camping.RestDuration)
	if scaled {
		text += fmt.Sprintf("\nThe %s makes for a poorer rest.", condition.Name)
	}
	return text
}

// breakCamp removes an idle camp. A resting camp cannot be broken.
func (m *CampingModule) breakCamp(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	camp, ok := m.camps[user.UserId]
	if !ok {
		return "You have no camp to break."
	}
	if camp.RoomID != room.RoomId {
		return "Your camp is not here."
	}
	if err := m.syncLocked(user.UserId); err != nil {
		mudlog.Warn("camping: break sync", "leader", user.UserId, "error", err)
	}
	camp = m.camps[user.UserId]
	if _, err := camp.Break(); err != nil {
		if errors.Is(err, camping.ErrRestInProgress) {
			return "You can't break camp while resting."
		}
		return "You can't break camp."
	}
	m.stopTimerLocked(user.UserId)
	delete(m.camps, user.UserId)
	delete(m.recoveryApplied, user.UserId)
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = camp
		return err.Error()
	}
	return "You break camp."
}

// AbandonForDeath implements camping.AbandonProvider (Phase 25a). The leader
// died, so their camp (resting or not) and any inn stay are left behind: a
// rest still running grants nothing and an inn stay refunds nothing. A rest
// that had already finished keeps its recovery, applied here first if it
// hadn't been. Everything is removed in one save; a failed save keeps it all,
// timers included.
func (m *CampingModule) AbandonForDeath(leaderUserID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	_, hasCamp := m.camps[leaderUserID]
	_, hasStay := m.stays[leaderUserID]
	if !hasCamp && !hasStay {
		return nil
	}
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: abandon sync", "leader", leaderUserID, "error", err)
	}
	if err := m.syncStayLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: abandon inn sync", "leader", leaderUserID, "error", err)
	}
	camp, hasCamp := m.camps[leaderUserID]
	applied, hasApplied := m.recoveryApplied[leaderUserID]
	stay, hasStay := m.stays[leaderUserID]
	innApplied, hasInnApplied := m.innRecoveryApplied[leaderUserID]
	delete(m.camps, leaderUserID)
	delete(m.recoveryApplied, leaderUserID)
	delete(m.stays, leaderUserID)
	delete(m.innRecoveryApplied, leaderUserID)
	if err := m.saveLocked(); err != nil {
		if hasCamp {
			m.camps[leaderUserID] = camp
		}
		if hasApplied {
			m.recoveryApplied[leaderUserID] = applied
		}
		if hasStay {
			m.stays[leaderUserID] = stay
		}
		if hasInnApplied {
			m.innRecoveryApplied[leaderUserID] = innApplied
		}
		return err
	}
	m.stopTimerLocked(leaderUserID)
	m.stopInnTimerLocked(leaderUserID)
	return nil
}

// status renders the current camp/fire/rest state, syncing an overdue rest
// completion first.
func (m *CampingModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if _, ok := m.camps[leaderUserID]; !ok {
		return "You have no camp."
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: status sync", "leader", leaderUserID, "error", err)
	}
	return m.statusTextLocked(leaderUserID)
}

// RenderCampView implements camping.ViewProvider. It replaces ordinary room
// rendering only while the leader is actively resting.
func (m *CampingModule) RenderCampView(leaderUserID int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if stay, ok := m.stays[leaderUserID]; ok && stay.Resting() {
		if err := m.syncStayLocked(leaderUserID); err != nil {
			mudlog.Warn("camping: inn view sync", "leader", leaderUserID, "error", err)
		}
		m.sendToLeader(leaderUserID, m.innStatusTextLocked(leaderUserID))
		return true, nil
	}
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil || camp.Rest.State != camping.Resting {
		return false, nil
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: view sync", "leader", leaderUserID, "error", err)
	}
	m.sendToLeader(leaderUserID, m.statusTextLocked(leaderUserID))
	return true, nil
}

// MovementBlocked implements camping.MovementProvider. Only an active rest
// blocks ordinary movement; an idle or broken camp never does.
func (m *CampingModule) MovementBlocked(leaderUserID int) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if stay, ok := m.stays[leaderUserID]; ok && stay.Resting() {
		return true, fmt.Sprintf("You are resting at the inn (%s remaining). Wait for your company to recover.", stay.RemainingAt(m.clock().UTC()).Round(time.Second))
	}
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil || camp.Rest.State != camping.Resting {
		return false, ""
	}
	remaining := m.remainingLocked(camp)
	return true, fmt.Sprintf("You are resting at camp (%s remaining). Wait for your company to recover, or it will be interrupted.", remaining.Round(time.Second))
}

func (m *CampingModule) remainingLocked(camp camping.Camp) time.Duration {
	if camp.Rest == nil {
		return 0
	}
	remaining := camping.RestDuration - m.clock().UTC().Sub(camp.Rest.StartedAtUTC)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// syncLocked completes a due rest and applies its survival recovery, or
// retries only the recovery call for a completed rest whose recovery has not
// yet been durably marked applied. Both paths are idempotent: a crash between
// persisting Completed and applying recovery retries recovery alone on the
// next call, and a crash after recovery succeeds but before the applied
// marker saves retries the (deduplicated) survival call harmlessly.
func (m *CampingModule) syncLocked(leaderUserID int) error {
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil {
		return nil
	}
	if camp.Rest.State != camping.Completed {
		now := m.clock().UTC()
		if !camp.RestDue(now) {
			return nil
		}
		completed, err := camp.CompleteRest(now)
		if err != nil {
			return err
		}
		original := camp
		m.camps[leaderUserID] = completed
		if err := m.saveLocked(); err != nil {
			m.camps[leaderUserID] = original
			return err
		}
		m.stopTimerLocked(leaderUserID)
		return m.applyRestRecoveryLocked(leaderUserID, completed, true)
	}
	if m.recoveryApplied[leaderUserID] {
		return nil
	}
	return m.applyRestRecoveryLocked(leaderUserID, camp, false)
}

func (m *CampingModule) applyRestRecoveryLocked(leaderUserID int, camp camping.Camp, announce bool) error {
	operationID := restOperationID(camp)
	if _, err := m.survival.ApplyCompanyRestRecovery(leaderUserID, operationID, camp.Rest.RecoveryAmount()); err != nil {
		return err
	}
	m.recoveryApplied[leaderUserID] = true
	// Phase 23a: the Rested tier is owed in the same save, and granted on
	// the game loop (this can run on a timer goroutine).
	m.restedPending[leaderUserID] = true
	if err := m.saveLocked(); err != nil {
		// The in-memory applied marker is kept so a same-process retry cannot
		// call survival a second time; persistence retries on the next save.
		return err
	}
	if announce {
		m.sendToLeader(leaderUserID, "Your company feels rested.")
	}
	return nil
}

func (m *CampingModule) scheduleLocked(camp camping.Camp) {
	leaderUserID := camp.LeaderUserID
	if camp.Rest == nil || camp.Rest.State != camping.Resting {
		m.stopTimerLocked(leaderUserID)
		return
	}
	remaining := m.remainingLocked(camp)
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

func (m *CampingModule) stopTimerLocked(leaderUserID int) {
	if timer, ok := m.timers[leaderUserID]; ok {
		timer.Stop()
		delete(m.timers, leaderUserID)
	}
}

func (m *CampingModule) onTimer(leaderUserID int, generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if m.timerGeneration[leaderUserID] != generation {
		return
	}
	if _, ok := m.timers[leaderUserID]; !ok {
		return
	}
	delete(m.timers, leaderUserID)
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.Rest == nil || camp.Rest.State != camping.Resting {
		return
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: timer sync", "leader", leaderUserID, "error", err)
	}
}

// recoverLocked reconciles persisted camps with the current clock on module
// load, which also runs after a copyover restore. An active rest reschedules
// its remaining duration or completes once if overdue; a completed rest
// retries only its idempotent recovery. Invalid camps are retained untouched
// for operator repair.
func (m *CampingModule) recoverLocked() {
	for leaderUserID, camp := range m.camps {
		if err := camp.Validate(); err != nil {
			mudlog.Warn("camping: recovery invalid camp", "leader", leaderUserID, "error", err)
			continue
		}
		if camp.Rest == nil {
			continue
		}
		switch camp.Rest.State {
		case camping.Resting:
			if err := m.syncLocked(leaderUserID); err != nil {
				mudlog.Warn("camping: recovery sync", "leader", leaderUserID, "error", err)
				continue
			}
			if refreshed, ok := m.camps[leaderUserID]; ok && refreshed.Rest != nil && refreshed.Rest.State == camping.Resting {
				m.scheduleLocked(refreshed)
			}
		case camping.Completed:
			if err := m.syncLocked(leaderUserID); err != nil {
				mudlog.Warn("camping: recovery completion retry", "leader", leaderUserID, "error", err)
			}
		default:
			mudlog.Warn("camping: recovery unknown rest state", "leader", leaderUserID, "state", camp.Rest.State)
		}
	}
}

func (m *CampingModule) sendToLeader(leaderUserID int, text string) {
	if text == "" {
		return
	}
	if user := users.GetByUserId(leaderUserID); user != nil {
		user.SendText(text)
	}
}

// statusTextLocked renders the current camp, fire, rest state, and company
// needs. The module lock must be held.
func (m *CampingModule) statusTextLocked(leaderUserID int) string {
	camp, ok := m.camps[leaderUserID]
	if !ok {
		return "You have no camp."
	}
	lines := []string{fmt.Sprintf("Camp at %s.", roomTitle(camp.RoomID))}
	if camp.FireLit {
		lines = append(lines, "The campfire is lit.")
	} else {
		lines = append(lines, "There is no fire lit.")
	}
	if camp.Rest != nil {
		switch camp.Rest.State {
		case camping.Resting:
			progress := camp.ProgressAt(m.clock().UTC())
			remaining := m.remainingLocked(camp)
			lines = append(lines, fmt.Sprintf("Resting: %d%% complete (%s remaining).", int(progress*100), remaining.Round(time.Second)))
		case camping.Completed:
			lines = append(lines, "Your company has rested here.")
		}
	}
	lines = append(lines, "Company:")
	for _, member := range m.survival.CompanyNeeds(leaderUserID) {
		lines = append(lines, fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			member.Name,
			member.Needs.Hunger, survival.HungerLabel(member.Needs.Hunger),
			member.Needs.Thirst, survival.ThirstLabel(member.Needs.Thirst),
			member.Needs.Fatigue, survival.FatigueLabel(member.Needs.Fatigue)))
	}
	return strings.Join(lines, "\n")
}

func roomTitle(roomID int) string {
	if room := rooms.LoadRoom(roomID); room != nil && room.Title != "" {
		return room.Title
	}
	return fmt.Sprintf("room #%d", roomID)
}

func (m *CampingModule) userCommand(rest string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(rest)))
	if len(args) == 0 {
		user.SendText(m.establish(user, room))
		return true, nil
	}
	switch args[0] {
	case "status":
		text := m.status(user.UserId)
		if m.hasCamp(user.UserId) {
			text += "\n" + m.sharpenPreview(user)
		}
		user.SendText(text)
	case "sharpen":
		user.SendText(m.sharpenArgs(user, args[1:]))
	case "fire":
		user.SendText(m.lightFire(user, room))
	case "rest":
		user.SendText(m.startRest(user, room))
	case "break":
		user.SendText(m.breakCamp(user, room))
	default:
		user.SendText(campUsage)
	}
	return true, nil
}
