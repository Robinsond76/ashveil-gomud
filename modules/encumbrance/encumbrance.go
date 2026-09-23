// Package encumbrance owns the durable company cargo container, the
// configured capacity/load-band thresholds, personal-plus-cargo weight
// calculation, the read-only query seam other modules can consult, and the
// player-facing cargo command.
//
// This is a party/expedition-level weight system, separate from GoMud's
// native per-character, count-based Character.CarryCapacity() throttle
// (internal/characters/character.go), which it never touches.
package encumbrance

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

const cargoUsage = `Usage: cargo | cargo put <item> | cargo take <item>`

// Registry is the durable, leader-keyed set of company cargo containers.
type Registry struct {
	Cargo map[int]encumbrance.Cargo `yaml:"cargo"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Cargo: map[int]encumbrance.Cargo{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Cargo: make(map[int]encumbrance.Cargo, len(r.Cargo))}
	for leaderUserID, cargo := range r.Cargo {
		out.Cargo[leaderUserID] = cargo
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
	data, err := s.plug.ReadBytes("encumbrance")
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
	return s.plug.WriteStruct("encumbrance", registry)
}

// decodeRegistry parses stored bytes, dropping only entries keyed by an
// invalid leader ID. Invalid cargo content is retained for operator repair.
func decodeRegistry(data []byte, registry *Registry) error {
	var wire Registry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for leaderUserID, cargo := range wire.Cargo {
		if leaderUserID <= 0 {
			continue
		}
		cargo.LeaderUserID = leaderUserID
		loaded.Cargo[leaderUserID] = cargo
	}
	*registry = *loaded
	return nil
}

// EncumbranceModule owns durable leader-keyed cargo containers for one
// plugin.
type EncumbranceModule struct {
	plug       *plugins.Plugin
	store      Store
	itemSpec   func(itemId int) (items.ItemSpec, bool)
	userLookup func(userId int) *users.UserRecord

	capacityGrams int
	bands         []encumbrance.LoadBand

	cargo   map[int]encumbrance.Cargo
	loadErr error

	mu sync.Mutex
}

var (
	_ encumbrance.Provider     = (*EncumbranceModule)(nil)
	_ encumbrance.BandProvider = (*EncumbranceModule)(nil)
)

func init() {
	m := &EncumbranceModule{
		plug: plugins.New("encumbrance", "1.0"),
		itemSpec: func(itemId int) (items.ItemSpec, bool) {
			spec := items.GetItemSpec(itemId)
			if spec == nil {
				return items.ItemSpec{}, false
			}
			return *spec, true
		},
		userLookup: users.GetByUserId,
		cargo:      map[int]encumbrance.Cargo{},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("cargo", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("encumbrance: save", "error", err)
		}
	})
	encumbrance.SetProvider(m)
}

func (m *EncumbranceModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("encumbrance: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("encumbrance: persistence unavailable")
	}
	return nil
}

func (m *EncumbranceModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *EncumbranceModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	registry := Registry{Cargo: m.cargo}
	if err := m.store.Save(registry); err != nil {
		return fmt.Errorf("encumbrance: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the capacity/load-band configuration, loads the durable
// registry, and logs (without discarding) any invalid cargo record for
// operator repair. Unlike expedition/camping/weather there is no real-time
// or round-driven state to reschedule here: cargo has no timers.
func (m *EncumbranceModule) load() {
	if m.store == nil {
		return
	}
	loaded := NewRegistry()
	if err := m.store.Load(loaded); err != nil {
		m.loadErr = err
		mudlog.Error("encumbrance: load", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cargo = loaded.Cargo
	if m.cargo == nil {
		m.cargo = map[int]encumbrance.Cargo{}
	}
	m.loadErr = nil
	if m.plug != nil {
		m.capacityGrams, m.bands = parseConfig(m.plug.Config.Get("CapacityKg"), m.plug.Config.Get("LoadBands"))
	}
	for leaderUserID, cargo := range m.cargo {
		if err := cargo.Validate(); err != nil {
			mudlog.Warn("encumbrance: recovery invalid cargo", "leader", leaderUserID, "error", err)
		}
	}
}

// CurrentLoad implements encumbrance.Provider. An unconfigured (non-positive)
// capacity reports untracked rather than a guessed default.
func (m *EncumbranceModule) CurrentLoad(leaderUserID int) (encumbrance.Load, bool) {
	if leaderUserID <= 0 {
		return encumbrance.Load{}, false
	}
	m.mu.Lock()
	capacityGrams := m.capacityGrams
	cargo, tracked := m.cargo[leaderUserID]
	m.mu.Unlock()
	if capacityGrams <= 0 {
		return encumbrance.Load{}, false
	}
	// A mount's cargo-capacity bonus (if any) adds to the configured base
	// capacity; without a tracked mount, mount.CapacityBonus is 0.
	capacityGrams += mount.CapacityBonus(leaderUserID)
	cargoGrams := 0
	if tracked {
		cargoGrams = m.cargoGramsOf(cargo)
	}
	return encumbrance.Load{
		PersonalGrams: m.personalGrams(leaderUserID),
		CargoGrams:    cargoGrams,
		CapacityGrams: capacityGrams,
	}, true
}

// CurrentBand implements encumbrance.BandProvider: the configured load band
// for the leader's current load.
func (m *EncumbranceModule) CurrentBand(leaderUserID int) (encumbrance.LoadBand, bool) {
	load, ok := m.CurrentLoad(leaderUserID)
	if !ok {
		return encumbrance.LoadBand{}, false
	}
	return encumbrance.ResolveBand(load.Ratio(), m.bandsSnapshot()), true
}

func (m *EncumbranceModule) personalGrams(leaderUserID int) int {
	if m.userLookup == nil {
		return 0
	}
	user := m.userLookup(leaderUserID)
	if user == nil || user.Character == nil {
		return 0
	}
	total := 0
	for _, item := range user.Character.Items {
		total += item.GetSpec().Weight
	}
	for _, item := range user.Character.Equipment.GetAllItems() {
		total += item.GetSpec().Weight
	}
	return total
}

func (m *EncumbranceModule) cargoGramsOf(cargo encumbrance.Cargo) int {
	total := 0
	for _, stack := range cargo.Stacks {
		if spec, ok := m.itemSpec(stack.ItemId); ok {
			total += spec.Weight * stack.Count
		}
	}
	return total
}

func (m *EncumbranceModule) bandsSnapshot() []encumbrance.LoadBand {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]encumbrance.LoadBand(nil), m.bands...)
}

// put moves one matching item from the leader's backpack into the company
// cargo. Moving an item between backpack and cargo never changes total
// party weight, so no capacity check applies here — only obtaining more
// items (looting, buying) increases total weight.
func (m *EncumbranceModule) put(user *users.UserRecord, itemName string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if itemName == "" {
		return cargoUsage
	}
	matchItem, found := user.Character.FindInBackpack(itemName)
	if !found {
		return fmt.Sprintf(`You don't have a "%s" to put in the cargo.`, itemName)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	original, hadEntry := m.cargo[user.UserId]
	cargo := original
	if !hadEntry {
		established, err := encumbrance.Established(user.UserId)
		if err != nil {
			return "You can't use the company cargo."
		}
		cargo = established
	}
	updated, err := cargo.Deposit(matchItem.ItemId, 1)
	if err != nil {
		return "You can't put that in the cargo."
	}
	m.cargo[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		if hadEntry {
			m.cargo[user.UserId] = original
		} else {
			delete(m.cargo, user.UserId)
		}
		return err.Error()
	}
	user.Character.RemoveItem(matchItem)
	return fmt.Sprintf(`You stow the <ansi fg="item">%s</ansi> in the company cargo.`, matchItem.DisplayName())
}

// take moves one matching item from the company cargo into the leader's
// backpack.
func (m *EncumbranceModule) take(user *users.UserRecord, itemName string) string {
	if err := m.persistenceAvailable(); err != nil {
		return err.Error()
	}
	if itemName == "" {
		return cargoUsage
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	cargo, ok := m.cargo[user.UserId]
	if !ok || len(cargo.Stacks) == 0 {
		return "The company cargo is empty."
	}
	itemId, found := m.matchCargoItem(cargo, itemName)
	if !found {
		return fmt.Sprintf(`The company cargo has no "%s".`, itemName)
	}
	updated, err := cargo.Withdraw(itemId, 1)
	if err != nil {
		return "You can't take that from the cargo."
	}
	m.cargo[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		m.cargo[user.UserId] = cargo
		return err.Error()
	}
	newItem := m.newCargoItem(itemId)
	user.Character.StoreItem(newItem)
	return fmt.Sprintf(`You take the <ansi fg="item">%s</ansi> from the company cargo.`, newItem.DisplayName())
}

// newCargoItem builds a fresh Item instance for a cargo-stack's item ID,
// via the module's item-spec lookup rather than items.New, so it resolves
// correctly under an injected test lookup as well as the live registry.
func (m *EncumbranceModule) newCargoItem(itemId int) items.Item {
	itm := items.Item{ItemId: itemId, UUID: uuid.New(items.UUIDItem)}
	if spec, ok := m.itemSpec(itemId); ok {
		itm.Spec = &spec
		if spec.Uses > 0 {
			itm.Uses = spec.Uses
		}
	}
	itm.Validate()
	return itm
}

// matchCargoItem reuses items.FindMatchIn's partial/exact matching over
// representative instances of each stacked item, matching drop/give's
// established item-name matching pattern.
func (m *EncumbranceModule) matchCargoItem(cargo encumbrance.Cargo, itemName string) (int, bool) {
	instances := make([]items.Item, 0, len(cargo.Stacks))
	for _, s := range cargo.Stacks {
		instances = append(instances, m.newCargoItem(s.ItemId))
	}
	partial, full := items.FindMatchIn(itemName, instances...)
	if full.ItemId > 0 {
		return full.ItemId, true
	}
	if partial.ItemId > 0 {
		return partial.ItemId, true
	}
	return 0, false
}

// status renders the current party load, modifier bands, and cargo
// contents.
func (m *EncumbranceModule) status(leaderUserID int) string {
	load, ok := m.CurrentLoad(leaderUserID)
	if !ok {
		return "The company has no cargo capacity configured."
	}
	lines := []string{
		fmt.Sprintf("Party load: %.1f kg / %.1f kg (%.0f%%)", float64(load.TotalGrams())/1000, float64(load.CapacityGrams)/1000, load.Ratio()*100),
	}
	band := encumbrance.ResolveBand(load.Ratio(), m.bandsSnapshot())
	if band.TravelDurationPct != 100 || band.FatiguePct != 100 {
		lines = append(lines, fmt.Sprintf("Travel duration modifier: %+d%%. Fatigue modifier: %+d%%.", band.TravelDurationPct-100, band.FatiguePct-100))
	}
	m.mu.Lock()
	cargo, tracked := m.cargo[leaderUserID]
	m.mu.Unlock()
	if !tracked || len(cargo.Stacks) == 0 {
		lines = append(lines, "The company cargo is empty.")
	} else {
		lines = append(lines, "Cargo:")
		for _, s := range cargo.Stacks {
			stackItem := m.newCargoItem(s.ItemId)
			lines = append(lines, fmt.Sprintf("  %s x%d", stackItem.DisplayName(), s.Count))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *EncumbranceModule) userCommand(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		user.SendText(m.status(user.UserId))
		return true, nil
	}
	itemName := strings.TrimSpace(strings.Join(args[1:], " "))
	switch strings.ToLower(args[0]) {
	case "put":
		user.SendText(m.put(user, itemName))
	case "take":
		user.SendText(m.take(user, itemName))
	default:
		user.SendText(cargoUsage)
	}
	return true, nil
}

// parseConfig normalizes the configured capacity and load-band table,
// rejecting a malformed band rather than applying a guess, matching
// modules/weather's parseBiomeTables.
func parseConfig(capacityRaw, bandsRaw any) (int, []encumbrance.LoadBand) {
	capacityGrams := int(configFloat(capacityRaw) * 1000)
	if capacityGrams < 0 {
		capacityGrams = 0
	}

	bands := []encumbrance.LoadBand{}
	list, ok := bandsRaw.([]any)
	if !ok {
		return capacityGrams, bands
	}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			continue
		}
		band := encumbrance.LoadBand{
			MinRatio:          configFloat(fields["minratio"]),
			TravelDurationPct: configInt(fields["traveldurationpct"]),
			FatiguePct:        configInt(fields["fatiguepct"]),
		}
		if err := band.Validate(); err != nil {
			mudlog.Warn("encumbrance: invalid load band", "error", err)
			continue
		}
		bands = append(bands, band)
	}
	sort.Slice(bands, func(i, j int) bool { return bands[i].MinRatio < bands[j].MinRatio })
	return capacityGrams, bands
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
	return nil
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

func configFloat(raw any) float64 {
	switch value := raw.(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			return f
		}
	}
	return 0
}
