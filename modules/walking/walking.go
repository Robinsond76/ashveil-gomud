// Package walking owns Phase 16 walking fatigue: the balance config, the
// durable per-member strain carry, the step handler the user Go command
// calls after an ordinary move, the round-driven tick that keeps each
// member's fatigue band buff (Exhausted, Collapsed) in sync, and the
// player-facing strain command. It ships the Well Rested, Exhausted, and
// Collapsed buffs and the well-rested buff flag.
//
// Steps run on the game loop. Ticks react to events.NewRound and only read
// the shared clock; nothing here schedules timers or advances the world
// clock. Lock order: walking -> {encumbrance, mount, weather, exposure} ->
// survival; none of those call back into walking.
package walking

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/walking"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

// Buff ids shipped by this module.
const (
	WellRestedBuffId = 1030
	ExhaustedBuffId  = 1031
	CollapsedBuffId  = 1032
	// RestedBuffId is the Phase 23a camp tier, granted by modules/camping.
	RestedBuffId = 1033

	// WellRestedFlag is the buff flag that halves walking strain.
	WellRestedFlag = "well-rested"
	// RestedFlag is the buff flag of the weaker camp tier (Phase 23a).
	RestedFlag = "rested"

	// ExhaustedMax is the highest fatigue that counts as Exhausted; 0 is
	// Collapsed. It matches survival's critical band (1..25).
	ExhaustedMax = 25
)

// Settings is the parsed module configuration.
type Settings struct {
	TickRounds    int
	Terrain       walking.TerrainTable
	Cold          walking.ColdSettings
	WellRestedPct int
	RestedPct     int
}

// DefaultSettings mirrors files/data-overlays/config.yaml.
func DefaultSettings() Settings {
	return Settings{
		TickRounds:    5,
		Terrain:       walking.DefaultTerrain(),
		Cold:          walking.DefaultColdSettings(),
		WellRestedPct: 50,
		RestedPct:     75,
	}
}

// parseSettings reads config through get, keeping the default for any value
// that is missing or out of range.
func parseSettings(get func(string) any) Settings {
	s := DefaultSettings()
	positive := func(name string, dst *int) {
		if n, ok := configInt(get(name)); ok && n > 0 {
			*dst = n
		}
	}
	nonNegative := func(name string, dst *int) {
		if n, ok := configInt(get(name)); ok && n >= 0 {
			*dst = n
		}
	}
	positive("TickRounds", &s.TickRounds)
	nonNegative("DefaultStrain", &s.Terrain.Default)
	positive("ChilledPct", &s.Cold.ChilledPct)
	positive("FrostbittenPct", &s.Cold.FrostbittenPct)
	positive("HypothermicPct", &s.Cold.HypothermicPct)
	positive("WellRestedPct", &s.WellRestedPct)
	positive("RestedPct", &s.RestedPct)

	if list, ok := get("Settlements").([]any); ok {
		settlements := map[string]bool{}
		for _, entry := range list {
			if biome := strings.ToLower(strings.TrimSpace(configString(entry))); biome != "" {
				settlements[biome] = true
			}
		}
		s.Terrain.Settlements = settlements
	}
	if list, ok := get("Biomes").([]any); ok {
		biomes := map[string]int{}
		for _, entry := range list {
			fields := stringMap(entry)
			biome := strings.ToLower(strings.TrimSpace(configString(fields["biome"])))
			strain, ok := configInt(fields["strain"])
			if biome == "" || !ok || strain < 0 {
				mudlog.Warn("walking: invalid biome strain", "entry", entry)
				continue
			}
			biomes[biome] = strain
		}
		if len(biomes) > 0 {
			s.Terrain.Biomes = biomes
		}
	}
	return s
}

// Registry is the durable strain carry: leader user id -> member key ->
// centi-fatigue carried toward the next fatigue point (1..99; zero entries
// are omitted).
type Registry struct {
	Carry map[int]map[string]int `yaml:"carry"`
}

func newRegistry() Registry { return Registry{Carry: map[int]map[string]int{}} }

func (r Registry) clone() Registry {
	out := newRegistry()
	for leader, members := range r.Carry {
		copied := make(map[string]int, len(members))
		for key, value := range members {
			copied[key] = value
		}
		out.Carry[leader] = copied
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
	data, err := s.plug.ReadBytes("walking")
	if errors.Is(err, os.ErrNotExist) {
		*registry = newRegistry()
		return nil
	}
	if err != nil {
		return err
	}
	return decodeRegistry(data, registry)
}

func (s pluginStore) Save(registry Registry) error {
	return s.plug.WriteStruct("walking", registry)
}

// decodeRegistry parses stored bytes, dropping invalid leaders, member keys,
// and out-of-range carries.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := newRegistry()
	for leader, members := range wire.Carry {
		if leader <= 0 {
			continue
		}
		for key, value := range members {
			if !survival.ValidMemberKey(survival.MemberKey(key)) || value <= 0 || value >= walking.CentiPerPoint {
				continue
			}
			if loaded.Carry[leader] == nil {
				loaded.Carry[leader] = map[string]int{}
			}
			loaded.Carry[leader][key] = value
		}
	}
	*registry = loaded
	return nil
}

// member is one company member. Character is nil when the member isn't
// spawned right now.
type member struct {
	Key         survival.MemberKey
	CompanionID int
	Name        string
	Character   *characters.Character
	RoomId      int
}

// WalkingModule owns walking strain for one plugin.
type WalkingModule struct {
	plug     *plugins.Plugin
	store    Store
	settings Settings
	registry Registry
	dirty    bool
	loadErr  error

	// Seams; tests override them.
	lookupUser  func(userID int) *users.UserRecord
	onlineUsers func() []*users.UserRecord
	// companions lists the leader's roster companions; Character is nil for
	// one not currently spawned. known is false when no roster is available.
	companions   func(leaderUserID int) (roster []member, known bool)
	loadRoom     func(roomId int) *rooms.Room
	weatherIn    func(zone string) (weather.Condition, bool)
	loadBand     func(leaderUserID int) (encumbrance.LoadBand, bool)
	mountRelief  func(leaderUserID int) (fatiguePct, riders int)
	exposureOf   func(leaderUserID int, memberKey string) (int, bool)
	drain        func(leaderUserID int, key survival.MemberKey, cost survival.Exertion) error
	companyNeeds func(leaderUserID int) []survival.MemberNeeds

	mu sync.Mutex
}

var _ walking.StepProvider = (*WalkingModule)(nil)

func init() {
	m := newModule()
	m.plug = plugins.New("walking", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("strain", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("walking: save", "error", err)
		}
	})
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	walking.SetStepProvider(m)
}

func newModule() *WalkingModule {
	return &WalkingModule{
		settings:     DefaultSettings(),
		registry:     newRegistry(),
		lookupUser:   users.GetByUserId,
		onlineUsers:  users.GetAllActiveUsers,
		companions:   companionRoster,
		loadRoom:     rooms.LoadRoom,
		weatherIn:    weather.CurrentCondition,
		loadBand:     encumbrance.CurrentBand,
		mountRelief:  mount.Relief,
		exposureOf:   climate.ExposureOf,
		companyNeeds: survival.CompanyNeeds,
		drain: func(leaderUserID int, key survival.MemberKey, cost survival.Exertion) error {
			_, err := survival.ApplyMemberDrain(leaderUserID, key, cost)
			return err
		},
	}
}

// companionRoster lists a leader's company companions, with the live
// character for those currently spawned.
func companionRoster(leaderUserID int) ([]member, bool) {
	roster := survival.CurrentRoster(leaderUserID)
	if roster == nil {
		return nil, false
	}
	var out []member
	for _, ref := range roster {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok {
			continue
		}
		mb := member{Key: ref.Key, CompanionID: companionID, Name: ref.Name}
		if instanceId, ok := company.InstanceFor(leaderUserID, companionID); ok {
			if mob := mobs.GetInstance(instanceId); mob != nil {
				mb.Character = &mob.Character
				mb.RoomId = mob.Character.RoomId
			}
		}
		out = append(out, mb)
	}
	return out, true
}

func (m *WalkingModule) load() {
	if m.plug != nil {
		m.settings = parseSettings(m.plug.Config.Get)
	}
	if m.store == nil {
		return
	}
	loaded := newRegistry()
	if err := m.store.Load(&loaded); err != nil {
		m.loadErr = err
		mudlog.Error("walking: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = loaded
	m.dirty = false
	m.loadErr = nil
}

// save persists the carry when a step has changed it since the last save.
// Steps never write themselves; a crash loses under 1 fatigue point per
// member, which is harmless.
func (m *WalkingModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.dirty {
		return nil
	}
	if m.loadErr != nil {
		return fmt.Errorf("walking: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("walking: persistence unavailable")
	}
	if err := m.store.Save(m.registry); err != nil {
		return err
	}
	m.dirty = false
	return nil
}

// stepMember is one member's share of a step.
type stepMember struct {
	member
	factors walking.Factors
	riding  bool
	cold    bool
	rested  string // the rest tier's view note, "" for none
	cost    int
}

// stepPlan is a whole step: the terrain and each present member's cost.
type stepPlan struct {
	terrain     int
	biome       string
	loadPct     int
	weatherName string
	weatherPct  int
	mountPct    int
	riders      int
	members     []stepMember
	roster      []member
	rosterKnown bool
}

// planStep works out what stepping into dest costs each member walking with
// the leader: the leader, plus spawned companions in the origin or
// destination room. It only reads seams and takes no module lock.
func (m *WalkingModule) planStep(leader *users.UserRecord, dest *rooms.Room, fromRoomID int) stepPlan {
	p := stepPlan{loadPct: 100, weatherPct: 100, mountPct: 100}
	biomeInfo := dest.GetBiome()
	terrain := walking.Terrain{Tags: dest.Tags}
	if biomeInfo != nil {
		terrain.Biome = biomeInfo.BiomeId
		terrain.Lit = biomeInfo.IsLit()
	}
	p.biome = terrain.Biome
	p.terrain = walking.TerrainCost(terrain, m.settings.Terrain)

	if band, ok := m.loadBand(leader.UserId); ok && band.FatiguePct > 0 {
		p.loadPct = band.FatiguePct
	}
	if !dest.IsIndoor() {
		if condition, ok := m.weatherIn(dest.Zone); ok && condition.ExertionPct > 0 {
			p.weatherPct = condition.ExertionPct
			p.weatherName = condition.Name
		}
	}
	p.mountPct, p.riders = m.mountRelief(leader.UserId)
	if p.mountPct <= 0 {
		p.mountPct = 100
	}

	present := []member{{Key: survival.LeaderMemberKey, Name: leader.Character.Name, Character: leader.Character, RoomId: leader.Character.RoomId}}
	p.roster, p.rosterKnown = m.companions(leader.UserId)
	companions := append([]member(nil), p.roster...)
	// Riders are the leader first, then companions by ascending id.
	sort.Slice(companions, func(i, j int) bool { return companions[i].CompanionID < companions[j].CompanionID })
	for _, c := range companions {
		if c.Character == nil {
			continue // not spawned: didn't walk, doesn't take a seat
		}
		if c.RoomId != fromRoomID && c.RoomId != dest.RoomId {
			continue // not walking with the leader
		}
		present = append(present, c)
	}

	for i, mb := range present {
		sm := stepMember{member: mb}
		sm.factors = walking.Factors{LoadPct: p.loadPct, WeatherPct: p.weatherPct}
		if i < p.riders {
			sm.riding = true
			sm.factors.MountPct = p.mountPct
		}
		if exposure, ok := m.exposureOf(leader.UserId, string(mb.Key)); ok {
			sm.factors.ColdPct = walking.ColdPct(exposure, m.settings.Cold)
			sm.cold = sm.factors.ColdPct > 100
		}
		// The better rest tier wins; the two never stack.
		switch {
		case mb.Character == nil:
		case mb.Character.HasBuffFlag(WellRestedFlag):
			sm.rested = "well rested"
			sm.factors.WellRestedPct = m.settings.WellRestedPct
		case mb.Character.HasBuffFlag(RestedFlag):
			sm.rested = "rested"
			sm.factors.WellRestedPct = m.settings.RestedPct
		}
		sm.cost = walking.StepCost(p.terrain, sm.factors)
		p.members = append(p.members, sm)
	}
	return p
}

// Stepped implements walking.StepProvider. The user Go command calls it
// after an ordinary move succeeds; it charges every member walking with the
// mover their share of the step's strain.
func (m *WalkingModule) Stepped(userID, fromRoomID, toRoomID int) {
	leader := m.lookupUser(userID)
	if leader == nil || leader.Character == nil {
		return
	}
	dest := m.loadRoom(toRoomID)
	if dest == nil {
		return
	}
	plan := m.planStep(leader, dest, fromRoomID)
	if plan.terrain == 0 {
		return // settlements: no strain, and no survival touch
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return
	}
	carries := m.registry.Carry[userID]
	if carries == nil {
		carries = map[string]int{}
		m.registry.Carry[userID] = carries
	}
	for _, sm := range plan.members {
		carry, drain := walking.Accrue(carries[string(sm.Key)], sm.cost)
		if carry == 0 {
			delete(carries, string(sm.Key))
		} else {
			carries[string(sm.Key)] = carry
		}
		if drain > 0 && m.drain != nil {
			if err := m.drain(userID, sm.Key, survival.Exertion{Fatigue: drain}); err != nil && !errors.Is(err, survival.ErrExertionUnavailable) {
				mudlog.Warn("walking: fatigue drain", "leader", userID, "member", sm.Key, "error", err)
			}
		}
	}
	// Forget members no longer in the company at all.
	if plan.rosterKnown {
		onRoster := map[string]bool{string(survival.LeaderMemberKey): true}
		for _, c := range plan.roster {
			onRoster[string(c.Key)] = true
		}
		for key := range carries {
			if !onRoster[key] {
				delete(carries, key)
			}
		}
	}
	if len(carries) == 0 {
		delete(m.registry.Carry, userID)
	}
	m.dirty = true
}

func (m *WalkingModule) onNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}
	if m.settings.TickRounds < 1 || evt.RoundNumber%uint64(m.settings.TickRounds) != 0 {
		return events.Continue
	}
	m.tick()
	return events.Continue
}

// tick keeps each online leader's and spawned companion's fatigue band
// buff in sync with their stored fatigue.
func (m *WalkingModule) tick() {
	refresh := 2*m.settings.TickRounds + 1
	for _, user := range m.onlineUsers() {
		if user == nil || user.Character == nil {
			continue
		}
		fatigue := map[survival.MemberKey]int{}
		for _, needs := range m.companyNeeds(user.UserId) {
			fatigue[needs.Key] = needs.Needs.Fatigue
		}
		leaderFatigue, ok := fatigue[survival.LeaderMemberKey]
		if !ok {
			leaderFatigue = 100
		}
		if msg := bandMessage(syncBandBuff(user.Character, leaderFatigue, refresh), true, user.Character.Name); msg != "" {
			user.SendText(msg)
		}
		roster, _ := m.companions(user.UserId)
		for _, c := range roster {
			if c.Character == nil {
				continue
			}
			value, ok := fatigue[c.Key]
			if !ok {
				value = 100
			}
			if msg := bandMessage(syncBandBuff(c.Character, value, refresh), false, c.Name); msg != "" {
				user.SendText(msg)
			}
		}
	}
}

// bandBuffFor is the band buff for a fatigue value, or 0.
func bandBuffFor(fatigue int) int {
	switch {
	case fatigue <= 0:
		return CollapsedBuffId
	case fatigue <= ExhaustedMax:
		return ExhaustedBuffId
	}
	return 0
}

// hasActiveBuff reports an unexpired buff. Removing a buff only marks it
// expired until the round tick prunes it, so HasBuff alone isn't enough.
func hasActiveBuff(c *characters.Character, buffId int) bool {
	return len(c.Buffs.GetBuffs(buffId)) > 0
}

// bandChange is a band transition: the buff before and after (0 = none).
type bandChange struct{ before, after int }

// syncBandBuff keeps exactly the buff for fatigue's band active on c. Like
// Phase 15's band buffs it is non-permanent and refreshed every tick, so
// the engine's permabuff reconciliation never strips it.
func syncBandBuff(c *characters.Character, fatigue, refreshRounds int) bandChange {
	change := bandChange{after: bandBuffFor(fatigue)}
	for _, id := range []int{ExhaustedBuffId, CollapsedBuffId} {
		if hasActiveBuff(c, id) {
			change.before = id
			if id != change.after {
				c.RemoveBuff(id)
			}
		}
	}
	if change.after != 0 {
		if err := c.AddBuff(change.after, false, refreshRounds); err != nil {
			mudlog.Warn("walking: add band buff", "buff", change.after, "error", err)
		}
	}
	return change
}

// bandMessage announces a band change, or "" when the band holds.
func bandMessage(change bandChange, self bool, name string) string {
	if change.before == change.after {
		return ""
	}
	switch change.after {
	case ExhaustedBuffId:
		if change.before == CollapsedBuffId {
			return pick(self, "You find your feet again, though you are still exhausted.", name+" is back on their feet, though still exhausted.")
		}
		return pick(self, "You are exhausted. Rest at a camp or an inn.", name+" is exhausted.")
	case CollapsedBuffId:
		return pick(self, "You collapse with exhaustion! Your company can't set out on a route until you rest.", name+" collapses with exhaustion!")
	}
	return pick(self, "You feel less exhausted.", name+" looks less exhausted.")
}

func pick(self bool, selfText, otherText string) string {
	if self {
		return selfText
	}
	return otherText
}

func (m *WalkingModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	user.SendText(strings.Join(m.report(user, room), "\n"))
	return true, nil
}

// report explains what a step into this room costs each member and why.
func (m *WalkingModule) report(user *users.UserRecord, room *rooms.Room) []string {
	if room == nil {
		return []string{"You can't judge the ground here."}
	}
	plan := m.planStep(user, room, room.RoomId)
	ground := plan.biome
	if ground == "" {
		ground = "unknown"
	}
	if plan.terrain == 0 {
		return []string{fmt.Sprintf("Walking here (%s) costs no fatigue: settled ground.", ground)}
	}
	lines := []string{fmt.Sprintf("Walking here (%s) costs %d strain per step before modifiers (100 strain = 1 fatigue).", ground, plan.terrain)}
	var factors []string
	if plan.loadPct != 100 {
		factors = append(factors, fmt.Sprintf("load %d%%", plan.loadPct))
	}
	if plan.weatherPct != 100 {
		factors = append(factors, fmt.Sprintf("%s %d%%", plan.weatherName, plan.weatherPct))
	}
	if plan.riders > 0 && plan.mountPct != 100 {
		factors = append(factors, fmt.Sprintf("mount %d%% for %d rider(s)", plan.mountPct, plan.riders))
	}
	if len(factors) > 0 {
		lines = append(lines, "Company factors: "+strings.Join(factors, ", ")+".")
	}
	for _, sm := range plan.members {
		who := sm.Name
		if sm.Key == survival.LeaderMemberKey {
			who = "You"
		}
		var notes []string
		if sm.riding && plan.mountPct != 100 {
			notes = append(notes, "riding")
		}
		if sm.cold {
			notes = append(notes, fmt.Sprintf("cold %d%%", sm.factors.ColdPct))
		}
		if sm.rested != "" {
			notes = append(notes, fmt.Sprintf("%s %d%%", sm.rested, sm.factors.WellRestedPct))
		}
		line := fmt.Sprintf("  %s: %d strain per step (about %d fatigue per 10 steps)", who, sm.cost, (sm.cost*10+50)/walking.CentiPerPoint)
		if len(notes) > 0 {
			line += " — " + strings.Join(notes, ", ")
		}
		lines = append(lines, line+".")
	}
	return lines
}

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
	}
	return map[string]any{}
}

func configInt(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		return n, err == nil
	}
	return 0, false
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}
