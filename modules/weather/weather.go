// Package weather owns the durable, zone-keyed weather registry, the
// biome condition-table configuration, round-driven transitions, load/
// copyover recovery, the read-only query seam other modules consult, and
// the player-facing weather command.
//
// Unlike modules/camping and modules/expedition, weather is scheduled by
// the shared round counter, never real UTC time: it reacts to
// events.NewRound and never drives or advances it.
package weather

import (
	"embed"
	"errors"
	"fmt"
	"github.com/GoMudEngine/GoMud/internal/modconfig"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/climate"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/modstore"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sky"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

var ErrInvalidBiomeTable = errors.New("weather: invalid biome table")

// Registry is the durable, zone-keyed set of tracked weather records.
type Registry struct {
	Zones map[string]weather.ZoneWeather `yaml:"zones"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Zones: map[string]weather.ZoneWeather{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Zones: make(map[string]weather.ZoneWeather, len(r.Zones))}
	for zone, zw := range r.Zones {
		out.Zones[zone] = zw
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
	return modstore.Load(s.plug, "weather", func() Registry { return *NewRegistry() }, decodeRegistry, registry)
}

func (s pluginStore) Save(registry Registry) error {
	return modstore.Save(s.plug, "weather", registry)
}

// decodeRegistry parses stored bytes, dropping only entries keyed by an
// empty zone name. Invalid zone weather content is retained for operator
// repair by the module's recovery pass rather than dropped here.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for zone, zw := range wire.Zones {
		if zone == "" {
			continue
		}
		zw.Zone = zone
		loaded.Zones[zone] = zw
	}
	*registry = *loaded
	return nil
}

// weightedCondition pairs a Condition with its weighted-random selection
// weight within a biome table.
type weightedCondition struct {
	weather.Condition
	Weight int
}

// biomeTable is one biome's configured condition set and change-interval
// round range.
type biomeTable struct {
	ChangeMin  uint64
	ChangeMax  uint64
	Conditions []weightedCondition
}

func (t biomeTable) Validate() error {
	if t.ChangeMin < 1 || t.ChangeMax < t.ChangeMin {
		return ErrInvalidBiomeTable
	}
	if len(t.Conditions) == 0 {
		return ErrInvalidBiomeTable
	}
	return nil
}

// WeatherModule owns the durable zone weather registry for one plugin.
type WeatherModule struct {
	plug      *plugins.Plugin
	store     Store
	rng       func(maxExclusive int) int
	roundFn   func() uint64
	zoneNames func() []string
	zoneBiome func(zone string) string
	// skyView and timeOfDay read the room's sky and the shared clock for
	// the weather command; tests override them.
	skyView   func(*rooms.Room) weather.SkyView
	timeOfDay func() string
	// Phase 33f2 Weather Sense: the leader's best present forecaster, and
	// rounds per game hour.
	forecaster    func(leaderUserID int, roomIDs ...int) (archetypes.Specialist, bool)
	roundsPerHour func() uint64

	biomes  map[string]biomeTable
	zones   map[string]weather.ZoneWeather
	loadErr error

	mu sync.Mutex
}

var _ weather.Provider = (*WeatherModule)(nil)

func init() {
	m := &WeatherModule{
		plug:      plugins.New("weather", "1.0"),
		rng:       util.Rand,
		roundFn:   util.GetRoundCount,
		zoneNames: rooms.GetAllZoneNames,
		zoneBiome: rooms.GetZoneBiome,
		skyView:   (*rooms.Room).SkyView,
		timeOfDay: func() string { return gametime.GetDate().String() },
		forecaster: func(leaderUserID int, roomIDs ...int) (archetypes.Specialist, bool) {
			return archetypes.BestSpecialist(leaderUserID, archetypes.UtilityWeather, roomIDs...)
		},
		roundsPerHour: func() uint64 { return uint64(max(gametime.GetDate().RoundsPerDay/24, 1)) },
		biomes:        map[string]biomeTable{},
		zones:         map[string]weather.ZoneWeather{},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("weather", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("weather: save", "error", err)
		}
	})
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	weather.SetProvider(m)
}

func (m *WeatherModule) persistenceAvailable() error {
	return modstore.Available("weather", m.loadErr, m.store != nil)
}

func (m *WeatherModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *WeatherModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	registry := Registry{Zones: m.zones}
	if err := m.store.Save(registry); err != nil {
		return fmt.Errorf("weather: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the biome table configuration, loads the durable registry,
// and reconciles it against the current round and known zones.
func (m *WeatherModule) load() {
	if m.store == nil {
		return
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("weather: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones = loaded.Zones
	if m.zones == nil {
		m.zones = map[string]weather.ZoneWeather{}
	}
	m.loadErr = nil
	if m.plug != nil {
		m.biomes = parseBiomeTables(m.plug.Config.Get("Biomes"))
		sky.SetCycleDays(parseMoonCycleDays(m.plug.Config.Get("MoonCycleDays")))
	}
	m.recoverLocked()
}

// tableForZoneLocked resolves a zone's configured biome table, if any. A
// zone whose biome has no configured table is untracked, never defaulted.
func (m *WeatherModule) tableForZoneLocked(zone string) (biomeTable, bool) {
	if m.zoneBiome == nil {
		return biomeTable{}, false
	}
	table, ok := m.biomes[m.zoneBiome(zone)]
	return table, ok
}

func conditionKnown(table biomeTable, name string) bool {
	for _, c := range table.Conditions {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (m *WeatherModule) intervalLocked(table biomeTable) uint64 {
	span := table.ChangeMax - table.ChangeMin
	return table.ChangeMin + uint64(m.rng(int(span)+1))
}

// rollConditionLocked picks a weighted-random condition from the table.
// The table is validated non-empty with positive weights before it is ever
// stored, so this always returns a condition.
func (m *WeatherModule) rollConditionLocked(table biomeTable) weather.Condition {
	total := 0
	for _, c := range table.Conditions {
		total += c.Weight
	}
	roll := m.rng(total)
	for _, c := range table.Conditions {
		if roll < c.Weight {
			return c.Condition
		}
		roll -= c.Weight
	}
	return table.Conditions[len(table.Conditions)-1].Condition
}

func (m *WeatherModule) establishLocked(zone string, table biomeTable, currentRound uint64) (weather.ZoneWeather, error) {
	condition := m.rollConditionLocked(table)
	nextRound := currentRound + m.intervalLocked(table)
	zw, err := weather.Established(zone, condition, nextRound, currentRound)
	if err != nil {
		return zw, err
	}
	zw.Next = m.rollConditionLocked(table).Name
	return zw, nil
}

// advanceLocked moves a zone to its foretold next condition (Phase 33f2:
// rolled when the current one began, so a forecast was true) and foretells
// the one after. A zone with no usable foretold condition rolls one now.
func (m *WeatherModule) advanceLocked(existing weather.ZoneWeather, table biomeTable, currentRound uint64) (weather.ZoneWeather, error) {
	condition, ok := conditionNamed(table, existing.Next)
	if !ok {
		condition = m.rollConditionLocked(table)
	}
	nextRound := currentRound + m.intervalLocked(table)
	advanced, err := existing.Advance(currentRound, condition, nextRound)
	if err != nil {
		return advanced, err
	}
	advanced.Next = m.rollConditionLocked(table).Name
	return advanced, nil
}

// conditionNamed finds a condition in a table by name.
func conditionNamed(table biomeTable, name string) (weather.Condition, bool) {
	for _, c := range table.Conditions {
		if c.Name == name {
			return c.Condition, true
		}
	}
	return weather.Condition{}, false
}

// recoverLocked reconciles the persisted registry with the current round
// and the currently known zones/biomes on module load, which also runs
// after a copyover restore. A newly trackable zone (no persisted record)
// rolls an initial condition. An already-tracked zone that is overdue
// rolls forward exactly once to the current round; weather has no
// side effects to double-apply, so unlike travel/rest recovery there is
// nothing to replay. An invalid persisted record, or one whose condition no
// longer exists in the configured table, is retained untouched for operator
// repair rather than guessed.
func (m *WeatherModule) recoverLocked() {
	if m.zoneNames == nil {
		return
	}
	currentRound := m.roundFn()
	changed := false
	for _, zone := range m.zoneNames() {
		table, ok := m.tableForZoneLocked(zone)
		if !ok {
			continue
		}
		existing, tracked := m.zones[zone]
		if !tracked {
			zw, err := m.establishLocked(zone, table, currentRound)
			if err != nil {
				mudlog.Warn("weather: recovery establish", "zone", zone, "error", err)
				continue
			}
			m.zones[zone] = zw
			changed = true
			continue
		}
		if err := existing.Validate(); err != nil {
			mudlog.Warn("weather: recovery invalid zone weather", "zone", zone, "error", err)
			continue
		}
		if !conditionKnown(table, existing.Current) {
			mudlog.Warn("weather: recovery unknown condition", "zone", zone, "condition", existing.Current)
			continue
		}
		if !existing.Due(currentRound) {
			// Phase 33f2: a zone saved before forecasts (or whose foretold
			// condition left the table) foretells its next condition now.
			if !conditionKnown(table, existing.Next) {
				existing.Next = m.rollConditionLocked(table).Name
				m.zones[zone] = existing
				changed = true
			}
			continue
		}
		advanced, err := m.advanceLocked(existing, table, currentRound)
		if err != nil {
			mudlog.Warn("weather: recovery advance", "zone", zone, "error", err)
			continue
		}
		m.zones[zone] = advanced
		changed = true
	}
	if changed {
		if err := m.saveLocked(); err != nil {
			mudlog.Error("weather: recovery save", "error", err)
		}
	}
}

// onNewRound advances every already-tracked zone whose weather is due. It
// never establishes a newly trackable zone; that happens only on load, per
// recoverLocked's doc comment.
func (m *WeatherModule) onNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}
	if err := m.persistenceAvailable(); err != nil {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	for zone, existing := range m.zones {
		table, ok := m.tableForZoneLocked(zone)
		if !ok {
			continue
		}
		if err := existing.Validate(); err != nil {
			continue
		}
		if !conditionKnown(table, existing.Current) {
			continue
		}
		if !existing.Due(evt.RoundNumber) {
			continue
		}
		advanced, err := m.advanceLocked(existing, table, evt.RoundNumber)
		if err != nil {
			mudlog.Warn("weather: advance", "zone", zone, "error", err)
			continue
		}
		m.zones[zone] = advanced
		changed = true
	}
	if changed {
		if err := m.saveLocked(); err != nil {
			mudlog.Error("weather: round save", "error", err)
		}
	}
	return events.Continue
}

// CurrentCondition implements weather.Provider. A zone with no configured
// biome table, or with no tracked record yet, reports false rather than a
// guessed default.
func (m *WeatherModule) CurrentCondition(zone string) (weather.Condition, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	table, ok := m.tableForZoneLocked(zone)
	if !ok {
		return weather.Condition{}, false
	}
	zw, ok := m.zones[zone]
	if !ok {
		return weather.Condition{}, false
	}
	for _, c := range table.Conditions {
		if c.Name == zw.Current {
			return c.Condition, true
		}
	}
	return weather.Condition{}, false
}

// Forecast (Phase 33f2) is a zone's current and foretold conditions and the
// round the change comes; ok is false for an untracked zone or one with no
// foretold condition yet.
func (m *WeatherModule) Forecast(zone string) (current, next weather.Condition, changeRound uint64, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	table, tracked := m.tableForZoneLocked(zone)
	if !tracked {
		return current, next, 0, false
	}
	zw, tracked := m.zones[zone]
	if !tracked {
		return current, next, 0, false
	}
	current, okCurrent := conditionNamed(table, zw.Current)
	next, okNext := conditionNamed(table, zw.Next)
	return current, next, zw.NextChangeRound, okCurrent && okNext
}

func (m *WeatherModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	view := m.skyView(room)
	condition, tracked := m.CurrentCondition(view.WeatherZone)
	lines := []string{fmt.Sprintf("It is %s.", m.timeOfDay())}
	if temp, ok := climate.AirTemperatureIn(room.RoomId); ok {
		lines = append(lines, fmt.Sprintf("Temperature here: %d°C (%s).", temp, climate.TemperatureName(temp)))
	}
	lines = append(lines, weather.RenderSky(view, condition, tracked, true)...)
	lines = append(lines, m.forecastLines(user, room, view.WeatherZone)...)
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// parseMoonCycleDays reads the configured lunar cycle length, falling back
// to sky.DefaultCycleDays when it is missing or non-positive.
func parseMoonCycleDays(raw any) int {
	days := modconfig.IntOr(raw, 0)
	if days < 1 {
		return sky.DefaultCycleDays
	}
	return days
}

// parseBiomeTables normalizes the configured biome table list, rejecting a
// malformed table or condition rather than applying a guess, matching
// modules/expedition's parseProfiles.
func parseBiomeTables(raw any) map[string]biomeTable {
	tables := map[string]biomeTable{}
	list, ok := raw.([]any)
	if !ok {
		return tables
	}
	for _, entry := range list {
		fields := modconfig.Map(entry)
		if fields == nil {
			continue
		}
		biome := strings.TrimSpace(modconfig.String(fields["biome"]))
		if biome == "" {
			continue
		}
		interval := modconfig.Map(fields["changeintervalrounds"])
		table := biomeTable{
			ChangeMin: uint64(modconfig.IntOr(interval["min"], 0)),
			ChangeMax: uint64(modconfig.IntOr(interval["max"], 0)),
		}
		condList, ok := fields["conditions"].([]any)
		if !ok {
			mudlog.Warn("weather: invalid biome table conditions", "biome", biome)
			continue
		}
		seen := map[string]bool{}
		for _, condRaw := range condList {
			condFields := modconfig.Map(condRaw)
			if condFields == nil {
				continue
			}
			name := strings.TrimSpace(modconfig.String(condFields["name"]))
			condition := weather.Condition{
				Name:              name,
				Description:       modconfig.String(condFields["description"]),
				TravelDurationPct: modconfig.IntOr(condFields["traveldurationpct"], 0),
				ExertionPct:       modconfig.IntOr(condFields["exertionpct"], 0),
				RestRecoveryPct:   modconfig.IntOr(condFields["restrecoverypct"], 0),
				CloudCover:        modconfig.IntOr(condFields["cloudcover"], 0),
				VisibilityMod:     modconfig.IntOr(condFields["visibilitymod"], 0),
				TemperatureMod:    modconfig.IntOr(condFields["temperaturemod"], 0),
			}
			weight := modconfig.IntOr(condFields["weight"], 0)
			if err := condition.Validate(); err != nil {
				mudlog.Warn("weather: invalid biome condition", "biome", biome, "name", name, "error", err)
				continue
			}
			if weight <= 0 {
				mudlog.Warn("weather: invalid biome condition weight", "biome", biome, "name", name)
				continue
			}
			if seen[name] {
				mudlog.Warn("weather: duplicate biome condition", "biome", biome, "name", name)
				continue
			}
			seen[name] = true
			table.Conditions = append(table.Conditions, weightedCondition{Condition: condition, Weight: weight})
		}
		if err := table.Validate(); err != nil {
			mudlog.Warn("weather: invalid biome table", "biome", biome, "error", err)
			continue
		}
		if _, exists := tables[biome]; exists {
			mudlog.Warn("weather: duplicate biome table", "biome", biome)
			continue
		}
		tables[biome] = table
	}
	return tables
}
