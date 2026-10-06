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
	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/banter"
	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/enemyparty"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/userstate"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/companyview"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
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

const campUsage = "Usage: camp | camp status | camp fire | camp rest | camp cook | camp duties [member] [duty] | camp break | camp sharpen [status | auto on|off] | camp supplies | camp prepare [supply] [member] | camp poison [assign|unassign|preview|apply] | camp coat"
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
	// Phase 43b: each leader's camp poison preparation list.
	PoisonPlans map[int][]PoisonAssign `yaml:"poison_plans,omitempty"`
	// Phase 33f3: a completed, unbroken camp rest whose Forage and Vigil
	// are still owed, by leader (the rest's operation ID and camp room),
	// and when each leader's company last earned them (once per
	// CampRewardCooldown).
	CampRewards     map[int]campReward `yaml:"camp_rewards,omitempty"`
	LastCampRewards map[int]time.Time  `yaml:"last_camp_rewards,omitempty"`
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
		PoisonPlans:        map[int][]PoisonAssign{},
		CampRewards:        map[int]campReward{},
		LastCampRewards:    map[int]time.Time{},
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
		PoisonPlans:        clonePlans(r.PoisonPlans),
		CampRewards:        make(map[int]campReward, len(r.CampRewards)),
		LastCampRewards:    make(map[int]time.Time, len(r.LastCampRewards)),
	}
	for leaderUserID, reward := range r.CampRewards {
		out.CampRewards[leaderUserID] = reward
	}
	for leaderUserID, at := range r.LastCampRewards {
		out.LastCampRewards[leaderUserID] = at
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
	for leaderUserID, plan := range wire.PoisonPlans {
		var kept []PoisonAssign
		for _, a := range plan {
			if _, ok := items.PoisonByID(a.Poison); ok && (a.Hand == "main" || a.Hand == "off") && a.Member >= 0 {
				kept = append(kept, a)
			}
		}
		if leaderUserID > 0 && len(kept) > 0 {
			loaded.PoisonPlans[leaderUserID] = kept
		}
	}
	for leaderUserID, reward := range wire.CampRewards {
		if leaderUserID > 0 && reward.Op != "" && reward.RoomID > 0 {
			loaded.CampRewards[leaderUserID] = reward
		}
	}
	for leaderUserID, at := range wire.LastCampRewards {
		if leaderUserID > 0 && !at.IsZero() {
			loaded.LastCampRewards[leaderUserID] = at
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
// bonusSurvival is the optional bedroll-aware recovery of a Survival seam.
type bonusSurvival interface {
	ApplyCompanyRestRecoveryBonus(leaderUserID int, operationID string, fatigue int, bonusPct map[survival.MemberKey]int) ([]survival.ExertionResult, error)
}

// cappedSurvival is the duty-aware recovery of a Survival seam (Phase 51).
type cappedSurvival interface {
	ApplyCompanyRestRecoveryCapped(leaderUserID int, operationID string, fatigue int, bonusPct map[survival.MemberKey]int, ceilings map[survival.MemberKey]int) ([]survival.ExertionResult, error)
}

type Survival interface {
	ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]survival.ExertionResult, error)
	CompanyNeeds(leaderUserID int) []survival.MemberNeeds
	Available() error
}

type nativeSurvival struct{}

func (nativeSurvival) ApplyCompanyRestRecovery(leaderUserID int, operationID string, fatigue int) ([]survival.ExertionResult, error) {
	return survival.ApplyCompanyRestRecovery(leaderUserID, operationID, fatigue)
}

// ApplyCompanyRestRecoveryBonus is the bedroll-aware recovery (Phase 40a3).
func (nativeSurvival) ApplyCompanyRestRecoveryBonus(leaderUserID int, operationID string, fatigue int, bonusPct map[survival.MemberKey]int) ([]survival.ExertionResult, error) {
	return survival.ApplyCompanyRestRecoveryBonus(leaderUserID, operationID, fatigue, bonusPct)
}

// ApplyCompanyRestRecoveryCapped is the duty-aware recovery (Phase 51).
func (nativeSurvival) ApplyCompanyRestRecoveryCapped(leaderUserID int, operationID string, fatigue int, bonusPct map[survival.MemberKey]int, ceilings map[survival.MemberKey]int) ([]survival.ExertionResult, error) {
	return survival.ApplyCompanyRestRecoveryCapped(leaderUserID, operationID, fatigue, bonusPct, ceilings)
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
	poisonPlans        map[int][]PoisonAssign
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
	// spendSupply overrides the company's bandages and splints in tests
	// (Phase 30b).
	spendSupply func(leaderUserID int, item wounds.Item) bool
	removeBuff  func(c *characters.Character, buffID int)
	hasBuff     func(c *characters.Character, buffID int) bool
	// buffRounds overrides a held buff's rounds left in tests (Phase 26a).
	buffRounds   func(c *characters.Character, buffID int) int
	roundSeconds func() int
	travelling   func(leaderUserID int) bool
	// specialist is the leader's best present company specialist at a
	// utility (Phase 33f3; nil: none).
	specialist func(leaderUserID int, utility string, roomIDs ...int) (archetypes.Specialist, bool)
	// Phase 33f3: rolls (0..n-1), the raid spawner, the owed camp rewards,
	// and camp-specialist config.
	roll     func(n int) int
	inBattle func(userID int) bool
	// memberLevel is one member's own utility level (Phase 51 watchers and
	// foragers; companionID 0 is the leader). Nil reads the archetypes.
	memberLevel func(leaderUserID, companionID int, utility string) int
	// onDuties is the camp-fire banter hook (Phase 49): called once a rest
	// begins with the duties locked on it (never with none).
	onDuties func(leaderUserID int, duties map[string]string)
	// companionCooks stands in for the live companions' cooking ranks
	// in tests (Phase 35c); nil reads the live mobs.
	companionCooks func(leaderUserID int, recipes []campRecipe) []campCook
	spawnRaid      func(roomID, mobTemplateID, leaderUserID int) (int, error)
	campRewards    map[int]campReward
	lastRewards    map[int]time.Time
	// raiders, and the seams that find, check, and send off raid groups.
	raiders       map[int]raiders
	raidGroup     func(roomID, first int) []int
	raiderAlive   func(instanceID int) bool
	despawnRaider func(instanceID int)
	campCfg       campSettings
	campCfgLoaded bool
	// Phase 40a2 fuel: the company's item count and spend (nil: native),
	// and the leaders whose first try with damp wood failed.
	// surgery stands in for the company's field surgery in tests (Phase
	// 40a3).
	surgery func(leaderUserID int) ([]string, bool)
	// theft stands in for the company's camp theft in tests (Phase 40a4).
	theft     func(leaderUserID, sharePct, maxUnits int, pick func(n int) int, protect func(itemID int) bool) []company.TheftLoss
	itemCount func(leaderUserID, itemID int) int
	spendItem func(leaderUserID, itemID int) bool
	dampTried map[int]bool

	mu sync.Mutex

	// roomCamps is a snapshot of the camps in each room (whose, and
	// whether lit), read by the rooms light-fixture query and look under
	// litMu only (lock order: mu, then litMu).
	litMu     sync.RWMutex
	roomCamps map[int][]camping.RoomCamp
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
		specialist:      archetypes.BestSpecialist,
		roll:            util.Rand,
		spawnRaid:       enemyparty.SpawnAmbush,
		raidGroup:       raidGroupOf,
		raiderAlive: func(id int) bool {
			mob := mobs.GetInstance(id)
			return mob != nil && mob.Character.Health > 0
		},
		despawnRaider: func(id int) {
			if mob := mobs.GetInstance(id); mob != nil {
				mob.Command("despawn raid over")
			}
		},
		inBattle: func(userID int) bool {
			_, busy := battle.Current(userID)
			return busy
		},
	}
	m.resetInnState()
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("camp", m.userCommand, false, false)
	m.plug.AddUserCommand("inn", m.innCommand, false, false)
	m.plug.AddUserCommand("sharpen", m.sharpenCommand, false, false)
	m.plug.AddUserCommand("coat", m.coatCommand, false, false)
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
	userstate.Register(stateContributor{m})
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("camping: save", "error", err)
		}
	})
	// Phase 26a: the default rest tier buffs are rest conditions even if
	// the camping data never loads; load files the configured ones.
	d := defaultInnSettings()
	companyview.RegisterBuffGroup(companyview.GroupRest, d.RestedBuffId, d.WellRestedBuffId)
	camping.SetViewProvider(m)
	camping.SetMovementProvider(m)
	camping.SetAbandonProvider(m)
	camping.SetCampAbandoner(m)
	rooms.RegisterLightFixture(m.RoomHasLitFire)
	// Phase 40a: forage and shelter act on a camp rest, so the room shows
	// them only where a camp can be made.
	rooms.SetCampableCheck(func(r *rooms.Room) bool { return roomEligible(r, m.roomTag()) })
	// Phase 32a: look shows the camps in a room.
	camping.SetRoomCampsReader(m.RoomCamps)
	// A lit campfire also warms its room (Phase 15).
	// A fire of damp wood (40a2) lights its room but gives no warmth.
	climate.RegisterHeatSource(m.RoomWarmedByFire)
}

// RoomHasLitFire reports whether a lit campfire burns in roomID, which
// lights that room for everyone (Phase 14). It reads a snapshot guarded by
// its own RWMutex, never m.mu, so look and combat never wait on a camping
// save or timer that holds m.mu.
func (m *CampingModule) RoomHasLitFire(roomID int) bool {
	m.litMu.RLock()
	defer m.litMu.RUnlock()
	for _, camp := range m.roomCamps[roomID] {
		if camp.FireLit {
			return true
		}
	}
	return false
}

// RoomWarmedByFire reports whether a campfire that warms its room burns in
// roomID: lit, and not a fire of damp wood (Phase 40a2).
func (m *CampingModule) RoomWarmedByFire(roomID int) bool {
	m.litMu.RLock()
	defer m.litMu.RUnlock()
	for _, camp := range m.roomCamps[roomID] {
		// Embers keep their warmth until the camp is broken, and a tent
		// shuts the cold out of a rest (Phase 40a3).
		if (camp.FireLit || camp.Embers) && !camp.Damp || camp.Tent && camp.Resting {
			return true
		}
	}
	return false
}

// RoomCamps returns a copy of the camps pitched in roomID, leader order,
// from the same snapshot as RoomHasLitFire, so look never waits on m.mu
// (Phase 32a).
func (m *CampingModule) RoomCamps(roomID int) []camping.RoomCamp {
	m.litMu.RLock()
	defer m.litMu.RUnlock()
	return slices.Clone(m.roomCamps[roomID])
}

// refreshLitRoomsLocked rebuilds the room-camps snapshot from m.camps.
// Callers hold m.mu; it is deferred after every m.mu section so the snapshot
// always matches the final (possibly reverted) camp state.
func (m *CampingModule) refreshLitRoomsLocked() {
	byRoom := map[int][]camping.RoomCamp{}
	for _, camp := range m.camps {
		resting := camp.Rest != nil && camp.Rest.State == camping.Resting
		byRoom[camp.RoomID] = append(byRoom[camp.RoomID], camping.RoomCamp{LeaderUserID: camp.LeaderUserID, FireLit: camp.FireLit, Damp: camp.Damp,
			Embers: camp.Embers, Tent: camp.Tent, Resting: resting})
	}
	for _, list := range byRoom {
		slices.SortFunc(list, func(a, b camping.RoomCamp) int { return a.LeaderUserID - b.LeaderUserID })
	}
	m.litMu.Lock()
	m.roomCamps = byRoom
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
		PoisonPlans:        m.poisonPlans,
		CampRewards:        m.campRewards,
		LastCampRewards:    m.lastRewards,
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
	if loaded.PoisonPlans != nil {
		m.poisonPlans = loaded.PoisonPlans
	}
	if loaded.CampRewards != nil {
		m.campRewards = loaded.CampRewards
	}
	if loaded.LastCampRewards != nil {
		m.lastRewards = loaded.LastCampRewards
	}
	if m.plug != nil {
		m.campCfg = parseCampSettings(m.plug.Config.Get)
		m.campCfgLoaded = true
		m.innCfg = parseInnSettings(m.plug.Config.Get)
		m.innCfgLoaded = true
		m.registerBuffGroupsLocked()
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
	tent := m.gearCount(user.UserId, tentItemID) > 0 // read before m.mu: it calls the company module
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
	camp.Tent = tent // Phase 40a3
	m.camps[user.UserId] = camp
	if err := m.saveLocked(); err != nil {
		delete(m.camps, user.UserId)
		return err.Error()
	}
	if tent {
		return "You make camp here, pegging out your oiled canvas tent."
	}
	return "You make camp here."
}

// lightFire lights the leader's camp fire. The leader must be at the camp,
// and (Phase 40a2) the fire needs fuel: the room's own deadfall, or a
// firewood bundle from the company.
func (m *CampingModule) lightFire(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	m.mu.Lock()
	camp, ok := m.camps[user.UserId]
	var refusal string
	switch {
	case !ok:
		refusal = "You have no camp here. Use \"camp\" to make one."
	case camp.RoomID != room.RoomId:
		refusal = "Your camp is not here."
	case camp.FireLit:
		refusal = "Your campfire is already lit."
	}
	m.mu.Unlock()
	if refusal != "" {
		return refusal
	}
	// The fuel is spent outside m.mu: the company's packs and cargo have
	// their own locks.
	damp, steel, refusal := m.takeFuel(user, room)
	if refusal != "" {
		return refusal
	}
	tent := m.gearCount(user.UserId, tentItemID) > 0 // Phase 40a3

	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	camp, ok = m.camps[user.UserId]
	if !ok || camp.RoomID != room.RoomId {
		return "Your camp is not here."
	}
	lit, err := camp.LightFire()
	if err != nil {
		if errors.Is(err, camping.ErrFireAlreadyLit) {
			return "Your campfire is already lit."
		}
		return "You can't light a fire here."
	}
	lit.Damp = damp
	lit.Tent = tent
	m.camps[user.UserId] = lit
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = camp
		return err.Error()
	}
	if damp {
		return "The damp wood catches at last, and you coax a smoky, sullen fire to life. It gives light, but little warmth."
	}
	if steel {
		return "A few strikes of the fire steel into the tinder, and even the damp wood catches. You light a crackling campfire."
	}
	if camp.Embers {
		return "You feed the embers fresh fuel, and the fire crackles up again."
	}
	return "You light a crackling campfire."
}

// Firewood item ids (Phase 40a2); the gathering module hands them out.
const (
	firewoodItemID     = 40
	dampFirewoodItemID = 41
)

// takeFuel settles what a new fire burns: a room with its own deadfall
// (a firewood room) costs nothing; otherwise one dry bundle is spent, and a
// damp bundle lights only on a second try and gives no warmth. A refusal
// spends nothing.
func (m *CampingModule) takeFuel(user *users.UserRecord, room *rooms.Room) (damp, steel bool, refusal string) {
	if room.HasResource(rooms.ResourceFirewood) {
		return false, false, "" // the room's own deadfall feeds the fire
	}
	count, spend := m.itemCount, m.spendItem
	if count == nil || spend == nil {
		count, spend = company.CompanyItemCount, company.SpendCompanyItem
	}
	if count(user.UserId, firewoodItemID) > 0 {
		if spend(user.UserId, firewoodItemID) {
			return false, false, ""
		}
	}
	if count(user.UserId, dampFirewoodItemID) > 0 {
		// Phase 40a3: fire steel and tinder light damp wood first time,
		// at full warmth.
		if count(user.UserId, fireSteelItemID) > 0 && spend(user.UserId, dampFirewoodItemID) {
			return false, true, ""
		}
		m.mu.Lock()
		tried := m.dampTried[user.UserId]
		if !tried {
			if m.dampTried == nil {
				m.dampTried = map[int]bool{}
			}
			m.dampTried[user.UserId] = true
			m.mu.Unlock()
			return false, false, "The damp wood smokes and sputters and will not catch. Try again."
		}
		delete(m.dampTried, user.UserId)
		m.mu.Unlock()
		if spend(user.UserId, dampFirewoodItemID) {
			return true, false, ""
		}
	}
	return false, false, `Your company has no firewood to light a fire with. Buy firewood bundles from a provisioner or market, or gather deadfall where firewood grows (<ansi fg="command">help gathering</ansi>).`
}

// unlitCampRoom is the room of the leader's camp while its fire is unlit.
func (m *CampingModule) unlitCampRoom(leaderUserID int) (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	camp, ok := m.camps[leaderUserID]
	if !ok || camp.FireLit {
		return 0, false
	}
	return camp.RoomID, true
}

// fuelLine (Phase 40a2 review) tells the leader what a fire in the camp's
// room would burn, so a missing bundle is known before "camp fire" fails.
func (m *CampingModule) fuelLine(user *users.UserRecord, room *rooms.Room) string {
	if room != nil && room.HasResource(rooms.ResourceFirewood) {
		return "Fuel: the deadfall here feeds a fire for nothing."
	}
	count := m.itemCount
	if count == nil {
		count = company.CompanyItemCount
	}
	dry, damp := count(user.UserId, firewoodItemID), count(user.UserId, dampFirewoodItemID)
	switch {
	case dry > 0:
		return fmt.Sprintf("Fuel: %s; a fire here burns one.", bundles(dry, "firewood bundle"))
	case damp > 0:
		return fmt.Sprintf("Fuel: %s only; it lights on a second try and gives no warmth.", bundles(damp, "damp firewood bundle"))
	}
	return `Fuel: none. A fire here needs a firewood bundle: buy one at a market or <ansi fg="command">gather firewood</ansi> where it grows (<ansi fg="command">help gathering</ansi>).`
}

func bundles(n int, name string) string {
	if n == 1 {
		return "1 " + name
	}
	return fmt.Sprintf("%d %ss", n, name)
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
	// Phase 40a3: the gear is counted before m.mu (it calls the company
	// module) and locked on the rest; the bells wear a use once the rest
	// is on, also outside m.mu.
	gear := m.gearOf(user.UserId)
	// 40a4 review: a finished rest's thieves come before another rest.
	m.settleTheft(user.UserId)
	// Phase 43a: so are the queued broth and incense.
	funded := m.fundPrepared(user, room)
	funded.present = m.presentKeys(user)
	text, started := m.startRestLocked(user, room, gear, m.companyMembers(user.UserId), funded)
	if started && gear.Bells {
		spend := m.spendItem
		if spend == nil {
			spend = company.SpendCompanyItem
		}
		spend(user.UserId, campBellsItemID)
	}
	if started {
		m.settlePrepared(user.UserId, funded)
		m.announceDuties(user.UserId)
		// Phase 49: the company talks as it settles in.
		if said := company.CampBanter(user.UserId, banter.CtxCamp); len(said) > 0 {
			text += "\n\n" + banter.Format(said)
		}
	}
	return text
}

func (m *CampingModule) startRestLocked(user *users.UserRecord, room *rooms.Room, gear campGear, members int, funded restPrep) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	camp, ok := m.camps[user.UserId]
	if !ok {
		return "You have no camp here. Use \"camp\" to make one.", false
	}
	if camp.RoomID != room.RoomId {
		return "Your camp is not here.", false
	}
	if stay, ok := m.stays[user.UserId]; ok && stay.Resting() {
		return "You are already resting at the inn.", false
	}
	// A finished rest's recovery must be in before another rest begins.
	if camp.Rest != nil && camp.Rest.State == camping.Completed && !m.recoveryApplied[user.UserId] {
		if err := m.syncLocked(user.UserId); err != nil || !m.recoveryApplied[user.UserId] {
			return "Your last rest is still settling. Try again in a moment.", false
		}
		camp = m.camps[user.UserId]
	}
	resting, err := camp.StartRest(m.clock().UTC())
	if err != nil {
		switch {
		case errors.Is(err, camping.ErrFireNotLit) && camp.Embers:
			return "The fire has burned down to embers. Feed it with \"camp fire\" to rest again.", false
		case errors.Is(err, camping.ErrFireNotLit):
			return "You need a lit campfire to rest. Use \"camp fire\" first.", false
		case errors.Is(err, camping.ErrRestAlreadyStarted):
			return "You are already resting.", false
		}
		return "You can't rest here.", false
	}
	delete(m.recoveryApplied, user.UserId) // the new rest's recovery is not in yet
	// Phase 16: the weather at the camp scales its recovery, locked now.
	recovery, condition, scaled := m.campRecovery(room, gear.Tent)
	rest := *resting.Rest
	rest.Recovery = recovery
	rest.Bedrolls = gear.Bedrolls
	rest.Bells = gear.Bells
	rest.Kit = gear.Kit
	// Phase 43a: the broth and incense funded for this rest, locked now.
	rest.Broth = funded.Broth
	rest.Incense = funded.Incense
	resting.Prepared = funded.clearQueue(resting.Prepared)
	resting.Tent = gear.Tent
	// Phase 51: the duties of the members at the camp, locked now.
	rest.Duties = camping.LockDuties(camp.Duties, funded.present)
	// Phase 33f3: whether raiders come, and when, is settled now.
	rest.Raid = m.planRaidLocked(room, rest.StartedAtUTC)
	// Phase 40a4: so is whether thieves come; bells and trip lines never
	// let them.
	if !gear.Bells {
		rest.Theft = m.planTheftLocked(room)
	}
	resting.Rest = &rest
	m.camps[user.UserId] = resting
	if err := m.saveLocked(); err != nil {
		m.camps[user.UserId] = camp
		return err.Error(), false
	}
	m.scheduleLocked(resting)
	text := fmt.Sprintf("You settle in by the fire to rest. (%s)", camping.RestDuration)
	if scaled {
		text += fmt.Sprintf("\nThe %s makes for a poorer rest.", condition.Name)
		if room.HasResource(rooms.ResourceShelter) {
			text += " The shelter here softens it."
		} else if gear.Tent {
			text += " The tent softens it."
		}
	}
	if line := restGearText(resting.Rest, gear.Tent, members); line != "" {
		text += "\n" + line
	}
	if line := restPrepText(funded, m.prepName(user)); line != "" {
		text += "\n" + line
	}
	if line := m.dutyStartText(user, rest.Duties); line != "" {
		text += "\n" + line
	}
	// 40a4 review: warn of thieves on a road they work, whether or not
	// they come this time.
	if !gear.Bells && m.theftRiskIn(room) {
		text += "\nNo bells or trip lines are strung: on this road, thieves may slip into the camp while you sleep."
	}
	return text, true
}

// breakCamp removes an idle camp. A resting camp cannot be broken.
func (m *CampingModule) breakCamp(user *users.UserRecord, room *rooms.Room) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	// 40a4 review: breaking camp straight after a rest does not dodge its
	// thieves.
	m.settleTheft(user.UserId)
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
// died, so their camp (resting or not) and any inn stay are left behind; see
// abandon.
func (m *CampingModule) AbandonForDeath(leaderUserID int) error {
	return m.abandon(leaderUserID, true)
}

var _ camping.CampAbandoner = (*CampingModule)(nil)

// AbandonCamp implements camping.CampAbandoner (Phase 27b): the leader's
// camp alone, resting or not, on the same terms as AbandonForDeath. The
// tutorial strikes its course camps with it, since a camp in a room copy
// can't outlive the copy.
func (m *CampingModule) AbandonCamp(leaderUserID int) error {
	return m.abandon(leaderUserID, false)
}

// abandon removes the leader's camp and, with inn, their inn stay: a rest
// still running grants nothing and an inn stay refunds nothing. A rest that
// had already finished keeps its recovery, applied here first if it hadn't
// been. Everything is removed in one save; a failed save keeps it all,
// timers included. Camp data that couldn't be read may hold a camp this
// module doesn't know about, so that is an error too.
func (m *CampingModule) abandon(leaderUserID int, inn bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	_, hasCamp := m.camps[leaderUserID]
	_, hasStay := m.stays[leaderUserID]
	hasStay = hasStay && inn
	if !hasCamp && !hasStay {
		return nil
	}
	if hasCamp {
		if err := m.syncLocked(leaderUserID); err != nil {
			mudlog.Warn("camping: abandon sync", "leader", leaderUserID, "error", err)
		}
	}
	if hasStay {
		if err := m.syncStayLocked(leaderUserID); err != nil {
			mudlog.Warn("camping: abandon inn sync", "leader", leaderUserID, "error", err)
		}
	}
	camp, hasCamp := m.camps[leaderUserID]
	applied, hasApplied := m.recoveryApplied[leaderUserID]
	stay, hasStay := m.stays[leaderUserID]
	innApplied, hasInnApplied := m.innRecoveryApplied[leaderUserID]
	hasStay, hasInnApplied = hasStay && inn, hasInnApplied && inn
	delete(m.camps, leaderUserID)
	delete(m.recoveryApplied, leaderUserID)
	if inn {
		delete(m.stays, leaderUserID)
		delete(m.innRecoveryApplied, leaderUserID)
	}
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
	if inn {
		m.stopInnTimerLocked(leaderUserID)
	}
	return nil
}

// status renders the current camp/fire/rest state, syncing an overdue rest
// completion first.
func (m *CampingModule) status(leaderUserID int) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	// The gear is counted before m.mu: it calls the company module.
	gear, members := m.gearOf(leaderUserID), m.companyMembers(leaderUserID)
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.refreshLitRoomsLocked()
	if _, ok := m.camps[leaderUserID]; !ok {
		return "You have no camp."
	}
	if err := m.syncLocked(leaderUserID); err != nil {
		mudlog.Warn("camping: status sync", "leader", leaderUserID, "error", err)
	}
	text := m.statusTextLocked(leaderUserID)
	if prepared := m.camps[leaderUserID].Prepared; !prepared.Empty() {
		text += "\nSet by for the next rest (camp prepare status): " + m.queuedTextLocked(leaderUserID, prepared) + "."
	}
	// 40a3 review: between rests, show what the gear at hand will do (a
	// running rest already reports the gear locked for it).
	if camp := m.camps[leaderUserID]; camp.Rest == nil || camp.Rest.State != camping.Resting {
		if lines := gear.lines(members); len(lines) > 0 {
			text += "\nCamp gear at hand (help camp gear):\n" + strings.Join(lines, "\n")
		}
	}
	return text
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
	var err error
	if capped, ok := m.survival.(cappedSurvival); ok && len(fatigueCeilings(camp.Rest.Duties)) > 0 {
		// Phase 51: a watcher stayed up, so ends no better than Ready.
		_, err = capped.ApplyCompanyRestRecoveryCapped(leaderUserID, operationID, camp.Rest.RecoveryAmount(), bedrollBonuses(camp.Rest), fatigueCeilings(camp.Rest.Duties))
	} else if bonus, ok := m.survival.(bonusSurvival); ok && len(camp.Rest.Bedrolls) > 0 {
		_, err = bonus.ApplyCompanyRestRecoveryBonus(leaderUserID, operationID, camp.Rest.RecoveryAmount(), bedrollBonuses(camp.Rest))
	} else {
		_, err = m.survival.ApplyCompanyRestRecovery(leaderUserID, operationID, camp.Rest.RecoveryAmount())
	}
	if err != nil {
		return err
	}
	m.recoveryApplied[leaderUserID] = true
	// Phase 33f3: a raid that caught the company asleep spoils the rest: no
	// Rested tier and no camp rewards.
	if camp.Rest.Broken {
		if err := m.saveLocked(); err != nil {
			return err
		}
		if announce {
			m.sendToLeader(leaderUserID, "The rest is over, but after the raid nobody feels rested.")
		}
		return nil
	}
	// Phase 23a: the Rested tier is owed in the same save, and granted on
	// the game loop (this can run on a timer goroutine). Phase 33f3: so
	// are the rest's Forage and Vigil.
	m.restedPending[leaderUserID] = true
	// Forage and Vigil come at most once per CampRewardCooldown (33f3
	// review: a free one-minute rest must not be farmed).
	if last, ok := m.lastRewards[leaderUserID]; !ok || camp.Rest.StartedAtUTC.Sub(last) >= m.campSettings().RewardCooldown {
		m.campRewards[leaderUserID] = campReward{Op: operationID, RoomID: camp.RoomID, Foragers: strings.Join(camping.DutyMembers(camp.Rest.Duties, camping.DutyForage), ",")}
		m.lastRewards[leaderUserID] = camp.Rest.StartedAtUTC
	}
	if err := m.saveLocked(); err != nil {
		// The in-memory applied marker is kept so a same-process retry cannot
		// call survival a second time; persistence retries on the next save.
		return err
	}
	if announce {
		text := "Your company feels rested."
		if n := len(camp.Rest.Bedrolls); n == 1 {
			text += " The bedroll made for a deeper sleep."
		} else if n > 1 {
			text += fmt.Sprintf(" The %d bedrolls made for a deeper sleep.", n)
		}
		if camp.Rest.Bells && camp.Rest.Raid == nil {
			text += " The bells hung quiet."
		}
		text += " The fire has burned down to embers."
		// Phase 49: and talks as the rest ends.
		if said := company.CampBanter(leaderUserID, banter.CtxRested); len(said) > 0 {
			text += "\n\n" + banter.Format(said)
		}
		m.sendToLeader(leaderUserID, text)
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
			// A camp saved before Phase 40a3 kept its fire lit after the
			// rest; it burns down now, so resting again needs fuel.
			if camp.FireLit {
				camp.BurnDown()
				m.camps[leaderUserID] = camp
				if err := m.saveLocked(); err != nil {
					mudlog.Warn("camping: recovery burn down", "leader", leaderUserID, "error", err)
				}
			}
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
	switch {
	case camp.FireLit && camp.Damp:
		lines = append(lines, "The campfire is lit, but its damp wood gives light and no warmth.")
	case camp.FireLit:
		lines = append(lines, "The campfire is lit.")
	case camp.Embers:
		lines = append(lines, `The fire has burned down to glowing embers. They keep the camp warm; feed the fire (camp fire) to rest again.`)
	default:
		lines = append(lines, "There is no fire lit.")
	}
	if camp.Tent {
		lines = append(lines, "An oiled canvas tent is pitched here.")
	}
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		if line := restGearText(camp.Rest, camp.Tent, m.companyMembers(leaderUserID)); line != "" {
			lines = append(lines, line)
		}
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
	// Phase 51: a camp with duties set shows each member's, the locked
	// ones while resting.
	duties := camp.Duties
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		duties = camp.Rest.Duties
	}
	for _, member := range m.survival.CompanyNeeds(leaderUserID) {
		line := fmt.Sprintf("  %s: Hunger %d (%s), Thirst %d (%s), Fatigue %d (%s)",
			member.Name,
			member.Needs.Hunger, survival.HungerLabel(member.Needs.Hunger),
			member.Needs.Thirst, survival.ThirstLabel(member.Needs.Thirst),
			member.Needs.Fatigue, survival.FatigueLabel(member.Needs.Fatigue))
		if len(duties) > 0 {
			line += fmt.Sprintf(", Duty %s", camping.DutyOf(duties, string(member.Key)))
		}
		lines = append(lines, line)
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
		text := m.establish(user, room)
		if strings.HasPrefix(text, "You make camp here.") && room != nil {
			text += " " + m.fuelLine(user, room)
		}
		user.SendText(text)
		return true, nil
	}
	switch args[0] {
	case "status":
		text := m.status(user.UserId)
		if roomID, unlit := m.unlitCampRoom(user.UserId); unlit {
			text += "\n" + m.fuelLine(user, rooms.LoadRoom(roomID))
		}
		if m.hasCamp(user.UserId) {
			text += "\n" + m.sharpenPreview(user)
		}
		user.SendText(text)
	case "sharpen":
		user.SendText(m.sharpenArgs(user, args[1:]))
	case "coat":
		user.SendText(m.coatArgs(user, args[1:]))
	case "poison":
		user.SendText(m.poisonArgs(user, args[1:]))
	case "fire":
		user.SendText(m.lightFire(user, room))
	case "rest":
		user.SendText(m.startRest(user, room))
	case "break":
		user.SendText(m.breakCamp(user, room))
	case "cook":
		user.SendText(m.cook(user, room)) // Phase 33f3
	case "duties", "duty":
		user.SendText(m.dutiesCommand(user, room, args[1:])) // Phase 51
	case "supplies":
		user.SendText(m.suppliesCommand(user)) // Phase 43a
	case "prepare":
		user.SendText(m.prepareCommand(user, room, args[1:])) // Phase 43a
	default:
		user.SendText(campUsage)
	}
	return true, nil
}

// --- Phase 26a: read-only views for the information surfaces ---

var _ camping.RestProvider = (*CampingModule)(nil)

// LeaderRest implements camping.RestProvider: the leader's inn stay or
// camp, read under the module mutex with no side effects.
func (m *CampingModule) LeaderRest(leaderUserID int) (camping.RestActivity, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stay, ok := m.stays[leaderUserID]; ok && stay.Resting() {
		return camping.RestActivity{Inn: true, Resting: true, Remaining: stay.RemainingAt(m.clock().UTC())}, true
	}
	camp, ok := m.camps[leaderUserID]
	if !ok {
		return camping.RestActivity{}, false
	}
	if camp.Rest != nil && camp.Rest.State == camping.Resting {
		return camping.RestActivity{Resting: true, Remaining: m.remainingLocked(camp)}, true
	}
	return camping.RestActivity{}, true
}

// RestTierOf implements camping.RestProvider: the best rest tier the
// character holds (TierNone when none) and its real time left; ok is false
// only for a user who isn't online. Call it on the game loop: it
// reads the character's buffs.
func (m *CampingModule) RestTierOf(userID int) (camping.Tier, time.Duration, bool) {
	user := m.userByID(userID)
	if user == nil || user.Character == nil {
		return camping.TierNone, 0, false
	}
	m.mu.Lock()
	s := m.innSettings()
	m.mu.Unlock()
	tier := m.heldTier(user.Character, s)
	if tier == camping.TierNone {
		return camping.TierNone, 0, true // known: not rested
	}
	rounds := m.buffRoundsLeft(user.Character, s.tierBuff(tier))
	return tier, time.Duration(rounds*m.roundLength()) * time.Second, true
}

// buffRoundsLeft is the rounds a held buff has left.
func (m *CampingModule) buffRoundsLeft(c *characters.Character, buffID int) int {
	if m.buffRounds != nil {
		return m.buffRounds(c, buffID)
	}
	for i := range c.Buffs.List {
		if b := c.Buffs.List[i]; b.BuffId == buffID && b.TriggersLeft > 0 {
			if spec := buffs.GetBuffSpec(buffID); spec != nil {
				rounds, _ := buffs.GetDurations(b, spec)
				return rounds
			}
		}
	}
	return 0
}

// registerBuffGroupsLocked files the rest tier buffs under the conditions
// rest group. The caller holds m.mu (load).
func (m *CampingModule) registerBuffGroupsLocked() {
	s := m.innSettings()
	companyview.RegisterBuffGroup(companyview.GroupRest, s.RestedBuffId, s.WellRestedBuffId)
	// Phase 43a: the Fortified sizes show with rest, the draughts with the
	// climate conditions.
	for _, tier := range camping.BrothBuffs {
		companyview.RegisterBuffGroup(companyview.GroupRest, tier.BuffID)
	}
	companyview.RegisterBuffGroup(companyview.GroupSurvival, camping.WarmingBuffID, camping.CoolingBuffID)
}

var _ camping.CampStateProvider = (*CampingModule)(nil)

// CampStateOf implements camping.CampStateProvider (Phase 32g): the
// leader's camp seen from a room with those tags. It reads state only; the
// camp's room title is looked up after the lock is released.
func (m *CampingModule) CampStateOf(leaderUserID, roomID int, roomTags []string) (camping.CampState, bool) {
	has := func(tag string) bool {
		for _, t := range roomTags {
			if t == tag {
				return true
			}
		}
		return false
	}
	// Phase 40a4: the gear the company carries calls into the company
	// module, so it is read before m.mu; only a leader with a camp shows it.
	m.mu.Lock()
	_, hasCamp := m.camps[leaderUserID]
	m.mu.Unlock()
	var gear, supplies []string
	bells := false
	if hasCamp {
		carried := m.gearOf(leaderUserID)
		gear, bells = carried.labels(m.companyMembers(leaderUserID)), carried.Bells
		supplies = m.supplyLabels(leaderUserID)
	}
	m.mu.Lock()
	camp, ok := m.camps[leaderUserID]
	s := camping.CampState{Inn: has(m.innSettings().RoomTag), Gear: gear, Supplies: supplies}
	if !ok {
		s.CanCamp = has(m.roomTag())
	} else {
		s.HasCamp, s.Here, s.FireLit = true, camp.RoomID == roomID, camp.FireLit
		s.Embers, s.Tent = camp.Embers, camp.Tent
		s.RoomID = camp.RoomID
		if !camp.Prepared.Empty() {
			s.Prepared = strings.Split(m.queuedTextLocked(leaderUserID, camp.Prepared), ", ")
		}
		if camp.Rest != nil && camp.Rest.State == camping.Completed {
			s.Rested = true
		}
		if camp.Rest != nil && camp.Rest.State == camping.Resting {
			left := m.remainingLocked(camp)
			s.Resting = true
			s.RestSeconds = int(left.Round(time.Second).Seconds())
			s.RestPercent = int(camp.ProgressAt(m.clock().UTC()) * 100)
		}
	}
	m.mu.Unlock()
	if s.HasCamp && !s.Here {
		s.RoomTitle = roomTitle(camp.RoomID)
	}
	// Phase 51: the duty picker, for a camp the leader stands at.
	if s.HasCamp && s.Here {
		if user := m.userByID(leaderUserID); user != nil && user.Character != nil && user.Character.RoomId == camp.RoomID {
			s.Duties = m.dutyRows(user, camp)
			s.DutiesLocked = camp.Rest != nil && camp.Rest.State == camping.Resting
		}
	}
	// 40a4 review: the Camp tab warns when thieves work the camp's road
	// and no bells are carried.
	if s.HasCamp && !bells {
		s.TheftRisk = m.theftRiskIn(rooms.LoadRoom(camp.RoomID))
	}
	return s, true
}
