// Package exposure owns Phase 15 temperature exposure: the balance config,
// the durable per-member exposure registry, the round-driven tick that moves
// exposure, keeps each member's band buff in sync, deals exposure damage and
// survival drains, and the player-facing temperature command. It ships the
// cold and heat band buffs.
//
// Ticks react to events.NewRound and only read the shared clock; nothing
// here schedules timers or advances the world clock. Offline characters
// don't tick.
package exposure

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
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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

// Band buff ids shipped by this module, indexed by climate.Band (BandMild
// through BandCritical).
var (
	coldBuffs = map[climate.Band]int{climate.BandMild: 1010, climate.BandModerate: 1011, climate.BandSevere: 1012, climate.BandCritical: 1013}
	heatBuffs = map[climate.Band]int{climate.BandMild: 1020, climate.BandModerate: 1021, climate.BandSevere: 1022, climate.BandCritical: 1023}
)

// Settings is the parsed module configuration.
type Settings struct {
	TickRounds         int
	Temperature        climate.TemperatureSettings
	Comfort            climate.ComfortSettings
	Exposure           climate.ExposureSettings
	Warmth             climate.WarmthSettings
	DefaultBiome       climate.BiomeTemperature
	Biomes             map[string]climate.BiomeTemperature
	SevereDamagePct    int
	CriticalDamagePct  int
	ColdFatigueDivisor int
	HeatThirstDivisor  int
}

// DefaultSettings mirrors files/data-overlays/config.yaml.
func DefaultSettings() Settings {
	return Settings{
		TickRounds:  5,
		Temperature: climate.TemperatureSettings{IndoorTemperature: 18, FireWarmth: 10},
		Comfort:     climate.ComfortSettings{ComfortLow: 16, ComfortHigh: 30, HeatFactor: 0.5},
		Exposure:    climate.ExposureSettings{CeilingPerStress: 4, Recovery: 5, ShelterRecovery: 12},
		Warmth: climate.WarmthSettings{
			SlotDefaults: map[string]int{"body": 5, "legs": 3, "feet": 2, "head": 2, "gloves": 2, "neck": 1},
			WarmedBonus:  20,
		},
		DefaultBiome:       climate.BiomeTemperature{Base: 15, NightDrop: 6},
		Biomes:             map[string]climate.BiomeTemperature{},
		SevereDamagePct:    2,
		CriticalDamagePct:  10,
		ColdFatigueDivisor: 5,
		HeatThirstDivisor:  4,
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
	anyInt := func(name string, dst *int) {
		if n, ok := configInt(get(name)); ok {
			*dst = n
		}
	}
	positive("TickRounds", &s.TickRounds)
	anyInt("IndoorTemperature", &s.Temperature.IndoorTemperature)
	nonNegative("FireWarmth", &s.Temperature.FireWarmth)
	anyInt("ComfortLow", &s.Comfort.ComfortLow)
	anyInt("ComfortHigh", &s.Comfort.ComfortHigh)
	if f, ok := configFloat(get("HeatFactor")); ok && f >= 0 {
		s.Comfort.HeatFactor = f
	}
	positive("CeilingPerStress", &s.Exposure.CeilingPerStress)
	positive("Recovery", &s.Exposure.Recovery)
	positive("ShelterRecovery", &s.Exposure.ShelterRecovery)
	nonNegative("WarmedBonus", &s.Warmth.WarmedBonus)
	nonNegative("SevereDamagePct", &s.SevereDamagePct)
	nonNegative("CriticalDamagePct", &s.CriticalDamagePct)
	positive("ColdFatigueDivisor", &s.ColdFatigueDivisor)
	positive("HeatThirstDivisor", &s.HeatThirstDivisor)
	anyInt("DefaultBase", &s.DefaultBiome.Base)
	nonNegative("DefaultNightDrop", &s.DefaultBiome.NightDrop)
	if s.Comfort.ComfortHigh <= s.Comfort.ComfortLow {
		d := DefaultSettings()
		s.Comfort.ComfortLow, s.Comfort.ComfortHigh = d.Comfort.ComfortLow, d.Comfort.ComfortHigh
	}

	if list, ok := get("Biomes").([]any); ok {
		for _, entry := range list {
			fields := stringMap(entry)
			biome := strings.ToLower(strings.TrimSpace(configString(fields["biome"])))
			base, okBase := configInt(fields["base"])
			drop, okDrop := configInt(fields["nightdrop"])
			if biome == "" || !okBase || (okDrop && drop < 0) {
				mudlog.Warn("exposure: invalid biome temperature", "entry", entry)
				continue
			}
			s.Biomes[biome] = climate.BiomeTemperature{Base: base, NightDrop: drop}
		}
	}
	if list, ok := get("SlotWarmth").([]any); ok {
		slots := map[string]int{}
		for _, entry := range list {
			fields := stringMap(entry)
			slot := strings.ToLower(strings.TrimSpace(configString(fields["slot"])))
			warmth, ok := configInt(fields["warmth"])
			if slot == "" || !ok || warmth < 0 {
				mudlog.Warn("exposure: invalid slot warmth", "entry", entry)
				continue
			}
			slots[slot] = warmth
		}
		if len(slots) > 0 {
			s.Warmth.SlotDefaults = slots
		}
	}
	return s
}

// Registry is the durable exposure state: leader user id -> member key ->
// signed exposure (negative cold, positive heat). Zero entries are omitted.
type Registry struct {
	Exposure map[int]map[string]int `yaml:"exposure"`
}

func newRegistry() Registry { return Registry{Exposure: map[int]map[string]int{}} }

func (r Registry) clone() Registry {
	out := newRegistry()
	for leader, members := range r.Exposure {
		copied := make(map[string]int, len(members))
		for key, value := range members {
			copied[key] = value
		}
		out.Exposure[leader] = copied
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
	data, err := s.plug.ReadBytes("exposure")
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
	return s.plug.WriteStruct("exposure", registry)
}

// decodeRegistry parses stored bytes, dropping invalid leaders, member keys,
// and zero values, and clamping to the lethal bound.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := newRegistry()
	for leader, members := range wire.Exposure {
		if leader <= 0 {
			continue
		}
		for key, value := range members {
			if !survival.ValidMemberKey(survival.MemberKey(key)) || value == 0 {
				continue
			}
			if value > climate.ExposureMax {
				value = climate.ExposureMax
			} else if value < -climate.ExposureMax {
				value = -climate.ExposureMax
			}
			if loaded.Exposure[leader] == nil {
				loaded.Exposure[leader] = map[string]int{}
			}
			loaded.Exposure[leader][key] = value
		}
	}
	*registry = loaded
	return nil
}

// member is one company member present in the world this tick.
type member struct {
	Key       survival.MemberKey
	Name      string
	Character *characters.Character
	RoomId    int
}

// ExposureModule owns durable exposure for one plugin.
type ExposureModule struct {
	plug     *plugins.Plugin
	store    Store
	settings Settings
	registry Registry
	loadErr  error

	// Seams; tests override them.
	onlineUsers func() []*users.UserRecord
	companions  func(leaderUserID int) []member
	loadRoom    func(roomId int) *rooms.Room
	isNight     func() bool
	drain       func(leaderUserID int, key survival.MemberKey, cost survival.Exertion) error

	mu sync.Mutex
}

var _ climate.Provider = (*ExposureModule)(nil)

func init() {
	m := newModule()
	m.plug = plugins.New("exposure", "1.0")
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("temperature", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("exposure: save", "error", err)
		}
	})
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	climate.SetProvider(m)
}

func newModule() *ExposureModule {
	return &ExposureModule{
		settings:    DefaultSettings(),
		registry:    newRegistry(),
		onlineUsers: users.GetAllActiveUsers,
		companions:  liveCompanions,
		loadRoom:    rooms.LoadRoom,
		isNight:     func() bool { return gametime.GetDate().Night },
		drain: func(leaderUserID int, key survival.MemberKey, cost survival.Exertion) error {
			_, err := survival.ApplyMemberDrain(leaderUserID, key, cost)
			return err
		},
	}
}

// liveCompanions lists a leader's currently spawned company companions.
func liveCompanions(leaderUserID int) []member {
	var out []member
	for _, ref := range survival.CurrentRoster(leaderUserID) {
		companionID, ok := company.CompanionIDFromMemberKey(ref.Key)
		if !ok {
			continue
		}
		instanceId, ok := company.InstanceFor(leaderUserID, companionID)
		if !ok {
			continue
		}
		mob := mobs.GetInstance(instanceId)
		if mob == nil {
			continue
		}
		out = append(out, member{Key: ref.Key, Name: ref.Name, Character: &mob.Character, RoomId: mob.Character.RoomId})
	}
	return out
}

func (m *ExposureModule) load() {
	if m.plug != nil {
		m.settings = parseSettings(m.plug.Config.Get)
	}
	if m.store == nil {
		return
	}
	loaded := newRegistry()
	if err := m.store.Load(&loaded); err != nil {
		m.loadErr = err
		mudlog.Error("exposure: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = loaded
	m.loadErr = nil
}

func (m *ExposureModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *ExposureModule) saveLocked() error {
	if m.loadErr != nil {
		return fmt.Errorf("exposure: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("exposure: persistence unavailable")
	}
	return m.store.Save(m.registry)
}

// temperatureInputs gathers a room's temperature inputs.
func (m *ExposureModule) temperatureInputs(room *rooms.Room) climate.TemperatureInputs {
	in := climate.TemperatureInputs{Biome: m.settings.DefaultBiome}
	biome := room.GetBiome()
	if biome != nil {
		if bt, ok := m.settings.Biomes[strings.ToLower(biome.BiomeId)]; ok {
			in.Biome = bt
		}
	}
	in.Indoor = room.IsIndoor()
	in.Furnished = in.Indoor && biome != nil && !biome.IsDark()
	in.Night = m.isNight()
	if condition, ok := weather.CurrentCondition(room.Zone); ok {
		in.WeatherMod = condition.TemperatureMod
	}
	in.HeatSource = climate.RoomHasHeatSource(room.RoomId)
	return in
}

func (m *ExposureModule) airTemperature(room *rooms.Room) int {
	return climate.AirTemperature(m.temperatureInputs(room), m.settings.Temperature)
}

// AirTemperatureIn implements climate.Provider.
func (m *ExposureModule) AirTemperatureIn(roomId int) (int, bool) {
	room := m.loadRoom(roomId)
	if room == nil {
		return 0, false
	}
	return m.airTemperature(room), true
}

// warmthOf is the insulation a character is wearing.
func (m *ExposureModule) warmthOf(c *characters.Character) int {
	worn := []climate.WornPiece{}
	for _, item := range c.Equipment.GetAllItems() {
		spec := item.GetSpec()
		worn = append(worn, climate.WornPiece{Slot: strings.ToLower(string(spec.Type)), Warmth: spec.Warmth})
	}
	return climate.Warmth(worn, c.HasBuffFlag("warmed"), m.settings.Warmth)
}

func (m *ExposureModule) onNewRound(e events.Event) events.ListenerReturn {
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

// tick advances exposure for every online leader and their live companions.
func (m *ExposureModule) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return
	}
	before := m.registry.clone()
	for _, user := range m.onlineUsers() {
		if user == nil || user.Character == nil {
			continue
		}
		leader := member{Key: survival.LeaderMemberKey, Name: user.Character.Name, Character: user.Character, RoomId: user.Character.RoomId}
		present := map[string]bool{string(leader.Key): true}
		m.tickMemberLocked(user, leader)
		for _, companion := range m.companions(user.UserId) {
			present[string(companion.Key)] = true
			m.tickMemberLocked(user, companion)
		}
		// Forget members no longer in the company.
		for key := range m.registry.Exposure[user.UserId] {
			if !present[key] {
				delete(m.registry.Exposure[user.UserId], key)
			}
		}
		if len(m.registry.Exposure[user.UserId]) == 0 {
			delete(m.registry.Exposure, user.UserId)
		}
	}
	if !registriesEqual(before, m.registry) {
		if err := m.saveLocked(); err != nil {
			mudlog.Error("exposure: tick save", "error", err)
		}
	}
}

func registriesEqual(a, b Registry) bool {
	if len(a.Exposure) != len(b.Exposure) {
		return false
	}
	for leader, members := range a.Exposure {
		other := b.Exposure[leader]
		if len(members) != len(other) {
			return false
		}
		for key, value := range members {
			if other[key] != value {
				return false
			}
		}
	}
	return true
}

// tickMemberLocked moves one member's exposure one tick and applies its
// consequences: band buff, band-change message, damage, survival drain.
func (m *ExposureModule) tickMemberLocked(leader *users.UserRecord, mb member) {
	room := m.loadRoom(mb.RoomId)
	if room == nil || mb.Character == nil {
		return
	}
	air := m.airTemperature(room)
	stress := climate.Stress(air, m.warmthOf(mb.Character), m.settings.Comfort)
	sheltered := room.IsIndoor() || climate.RoomHasHeatSource(room.RoomId)

	prev := m.registry.Exposure[leader.UserId][string(mb.Key)]
	next := climate.StepExposure(prev, stress, sheltered, m.settings.Exposure)
	if next == 0 {
		delete(m.registry.Exposure[leader.UserId], string(mb.Key))
	} else {
		if m.registry.Exposure[leader.UserId] == nil {
			m.registry.Exposure[leader.UserId] = map[string]int{}
		}
		m.registry.Exposure[leader.UserId][string(mb.Key)] = next
	}

	syncBandBuff(mb.Character, next)
	if msg := bandChangeMessage(prev, next, mb.Key == survival.LeaderMemberKey, mb.Name); msg != "" {
		leader.SendText(msg)
	}

	band := climate.BandFor(next)
	pct := 0
	switch band {
	case climate.BandSevere:
		pct = m.settings.SevereDamagePct
	case climate.BandCritical:
		pct = m.settings.CriticalDamagePct
	}
	// A downed character already bleeds out through the normal death path.
	if pct > 0 && mb.Character.Health > 0 {
		damage := mb.Character.HealthMax.Value * pct / 100
		if damage < 1 {
			damage = 1
		}
		mb.Character.ApplyHealthChange(-damage)
		if next < 0 {
			leader.SendText(fmt.Sprintf(`<ansi fg="51">The cold bites %s for <ansi fg="damage">%d damage</ansi>!</ansi>`, who(mb, "you"), damage))
		} else {
			leader.SendText(fmt.Sprintf(`<ansi fg="202">The heat saps %s for <ansi fg="damage">%d damage</ansi>!</ansi>`, who(mb, "you"), damage))
		}
	}

	cost := survival.Exertion{}
	if stress < 0 && band >= climate.BandModerate {
		cost.Fatigue = max(1, -stress/m.settings.ColdFatigueDivisor)
	}
	if stress > 0 {
		cost.Thirst = max(1, stress/m.settings.HeatThirstDivisor)
	}
	if cost != (survival.Exertion{}) && m.drain != nil {
		if err := m.drain(leader.UserId, mb.Key, cost); err != nil && !errors.Is(err, survival.ErrExertionUnavailable) {
			mudlog.Warn("exposure: survival drain", "leader", leader.UserId, "member", mb.Key, "error", err)
		}
	}
}

func who(mb member, self string) string {
	if mb.Key == survival.LeaderMemberKey {
		return self
	}
	return mb.Name
}

// hasActiveBuff reports an unexpired buff. Removing a buff only marks it
// expired until the round tick prunes it, so HasBuff alone isn't enough.
func hasActiveBuff(c *characters.Character, buffId int) bool {
	return len(c.Buffs.GetBuffs(buffId)) > 0
}

// syncBandBuff keeps exactly the buff for exposure's band active on c.
func syncBandBuff(c *characters.Character, exposure int) {
	want := 0
	if band := climate.BandFor(exposure); band != climate.BandNone {
		if exposure < 0 {
			want = coldBuffs[band]
		} else {
			want = heatBuffs[band]
		}
	}
	for _, table := range []map[climate.Band]int{coldBuffs, heatBuffs} {
		for _, id := range table {
			if id != want && hasActiveBuff(c, id) {
				c.RemoveBuff(id)
			}
		}
	}
	// AddBuff revives an expired-but-unpruned entry as well as adding a new one.
	if want != 0 && !hasActiveBuff(c, want) {
		if err := c.AddBuff(want, true); err != nil {
			mudlog.Warn("exposure: add band buff", "buff", want, "error", err)
		}
	}
}

var coldMessages = map[climate.Band]string{
	climate.BandMild:     "%s getting chilled.",
	climate.BandModerate: "The cold bites deep: %s going numb.",
	climate.BandSevere:   "%s dangerously cold and shivering uncontrollably!",
	climate.BandCritical: "%s freezing to death! Find shelter, a fire, or warm clothing now!",
}

var heatMessages = map[climate.Band]string{
	climate.BandMild:     "%s overheating.",
	climate.BandModerate: "The heat makes %s head swim.",
	climate.BandSevere:   "%s close to collapse from the heat!",
	climate.BandCritical: "%s suffering heatstroke! Get out of the heat and drink!",
}

// bandChangeMessage announces a band change, or "" when the band holds.
func bandChangeMessage(prev, next int, self bool, name string) string {
	prevBand, nextBand := climate.BandFor(prev), climate.BandFor(next)
	prevCold, nextCold := prev < 0, next < 0
	if prevBand == nextBand && (nextBand == climate.BandNone || prevCold == nextCold) {
		return ""
	}
	if nextBand == climate.BandNone {
		if prevCold {
			return subject(self, name, "You feel warmer.", "%s looks warmer.")
		}
		return subject(self, name, "You feel cooler.", "%s looks cooler.")
	}
	if nextCold {
		return fmt.Sprintf(coldMessages[nextBand], bandSubject(self, name, nextBand, true))
	}
	return fmt.Sprintf(heatMessages[nextBand], bandSubject(self, name, nextBand, false))
}

func subject(self bool, name, selfText, otherFmt string) string {
	if self {
		return selfText
	}
	return fmt.Sprintf(otherFmt, name)
}

// bandSubject fills the %s in a band message for the member.
func bandSubject(self bool, name string, band climate.Band, cold bool) string {
	if !cold && band == climate.BandModerate {
		if self {
			return "your"
		}
		return name + "'s"
	}
	if cold && band == climate.BandModerate {
		if self {
			return "you are"
		}
		return name + " is"
	}
	if self {
		return "You are"
	}
	return name + " is"
}

func (m *ExposureModule) exposureFor(leaderUserID int, key survival.MemberKey) int {
	return m.registry.Exposure[leaderUserID][string(key)]
}

func (m *ExposureModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := m.reportLocked(user, room)
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

func (m *ExposureModule) reportLocked(user *users.UserRecord, room *rooms.Room) []string {
	air := m.airTemperature(room)
	in := m.temperatureInputs(room)
	lines := []string{fmt.Sprintf("Air temperature here: %d°C (%s).", air, climate.TemperatureName(air))}
	switch {
	case in.Indoor && in.Furnished:
		lines = append(lines, "You are indoors, sheltered from the weather.")
	case in.Indoor:
		lines = append(lines, "You are underground, where the air holds a steady temperature.")
	default:
		detail := []string{}
		if in.Night {
			detail = append(detail, "it is night")
		}
		if in.WeatherMod < 0 {
			detail = append(detail, "the weather is chilling the air")
		} else if in.WeatherMod > 0 {
			detail = append(detail, "the weather is warming the air")
		}
		if len(detail) > 0 {
			lines = append(lines, "Outdoors: "+strings.Join(detail, "; ")+".")
		}
	}
	if in.HeatSource {
		lines = append(lines, "A fire here warms you and speeds your recovery.")
	}

	warmth := m.warmthOf(user.Character)
	low, high := climate.ComfortRange(warmth, m.settings.Comfort)
	lines = append(lines, fmt.Sprintf("Your clothing gives %d warmth: you are comfortable from %d°C to %d°C.", warmth, low, high))
	lines = append(lines, memberStatus("You", m.exposureFor(user.UserId, survival.LeaderMemberKey)))

	companions := m.companions(user.UserId)
	sort.Slice(companions, func(i, j int) bool { return companions[i].Name < companions[j].Name })
	for _, c := range companions {
		lines = append(lines, memberStatus(c.Name, m.exposureFor(user.UserId, c.Key)))
	}
	return lines
}

var bandNames = map[bool]map[climate.Band]string{
	true:  {climate.BandMild: "chilled", climate.BandModerate: "frostbitten", climate.BandSevere: "hypothermic", climate.BandCritical: "freezing to death"},
	false: {climate.BandMild: "overheated", climate.BandModerate: "heatstricken", climate.BandSevere: "heat exhausted", climate.BandCritical: "suffering heatstroke"},
}

func memberStatus(name string, exposure int) string {
	verb := "is"
	if name == "You" {
		verb = "are"
	}
	band := climate.BandFor(exposure)
	switch {
	case exposure == 0:
		return fmt.Sprintf("%s %s comfortable.", name, verb)
	case band == climate.BandNone && exposure < 0:
		return fmt.Sprintf("%s %s a little cold (exposure %d/100).", name, verb, -exposure)
	case band == climate.BandNone:
		return fmt.Sprintf("%s %s a little warm (exposure %d/100).", name, verb, exposure)
	case exposure < 0:
		return fmt.Sprintf("%s %s %s (exposure %d/100).", name, verb, bandNames[true][band], -exposure)
	}
	return fmt.Sprintf("%s %s %s (exposure %d/100).", name, verb, bandNames[false][band], exposure)
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

func configFloat(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return f, err == nil
	}
	return 0, false
}

func configString(raw any) string {
	value, _ := raw.(string)
	return value
}
