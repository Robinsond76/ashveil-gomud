// Package encumbrance owns the durable company cargo container, the
// configured capacity/load-band thresholds, personal-plus-cargo weight
// calculation, the read-only query seam other modules can consult, and the
// player-facing cargo command.
//
// This is a party/expedition-level weight system. Since Phase 32f it is
// the only carrying limit: capacity comes from the members (a base, their
// Strength, and one pack each) and the horses, and a full company takes
// on nothing more (encumbrance.WouldExceed).
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

	"github.com/GoMudEngine/GoMud/internal/company"
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
	// companionGear is the living companions' gear weight (Phase 28); nil
	// counts none.
	companionGear func(leaderUserID int) int
	// companionCarry is each counted companion's carrying share (Phase
	// 32f); nil counts none.
	companionCarry func(leaderUserID int) []company.MemberCarry

	// memberBaseGrams is each member's base share of the capacity and
	// strengthGrams what a point of Strength adds (Phase 32f).
	memberBaseGrams int
	strengthGrams   int
	bands           []encumbrance.LoadBand

	cargo   map[int]encumbrance.Cargo
	loadErr error

	mu sync.Mutex
}

var (
	_ encumbrance.Provider      = (*EncumbranceModule)(nil)
	_ encumbrance.BandProvider  = (*EncumbranceModule)(nil)
	_ encumbrance.CargoProvider = (*EncumbranceModule)(nil)
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
		userLookup:     users.GetByUserId,
		companionGear:  company.CompanionGearGrams,
		companionCarry: company.CompanionCarry,
		cargo:          map[int]encumbrance.Cargo{},
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("cargo", m.userCommand, false, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	events.RegisterListener(events.UserPurged{}, m.onUserPurged)
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
		if m.plug.Config.Get("CapacityKg") != nil {
			mudlog.Warn("encumbrance: CapacityKg is retired (Phase 32f); capacity comes from MemberBaseKg, StrengthKg, packs, and horses")
		}
		m.memberBaseGrams, m.strengthGrams, m.bands = parseConfig(m.plug.Config.Get("MemberBaseKg"), m.plug.Config.Get("StrengthKg"), m.plug.Config.Get("LoadBands"))
	}
	for leaderUserID, cargo := range m.cargo {
		if err := cargo.Validate(); err != nil {
			mudlog.Warn("encumbrance: recovery invalid cargo", "leader", leaderUserID, "error", err)
		}
	}
}

// CurrentLoad implements encumbrance.Provider. An unconfigured (non-positive)
// member base reports untracked rather than a guessed default.
//
// Capacity (Phase 32f) is each member's share (the base, their Strength,
// and their largest pack) plus the horses' (mount.CapacityBonus). The
// members are the leader and the companions CompanionGearGrams weighs.
// Game loop only: it reads the leader's character and the company.
func (m *EncumbranceModule) CurrentLoad(leaderUserID int) (encumbrance.Load, bool) {
	if leaderUserID <= 0 {
		return encumbrance.Load{}, false
	}
	m.mu.Lock()
	baseGrams, strengthGrams := m.memberBaseGrams, m.strengthGrams
	cargo, tracked := m.cargo[leaderUserID]
	m.mu.Unlock()
	if baseGrams <= 0 {
		return encumbrance.Load{}, false
	}
	memberGrams := m.leaderCapacity(leaderUserID, baseGrams, strengthGrams)
	if m.companionCarry != nil {
		for _, c := range m.companionCarry(leaderUserID) {
			memberGrams += encumbrance.MemberCapacity(baseGrams, strengthGrams, c.Strength, c.PackGrams)
		}
	}
	mountGrams := mount.CapacityBonus(leaderUserID)
	cargoGrams := 0
	if tracked {
		cargoGrams = m.cargoGramsOf(cargo)
	}
	companionGrams := 0
	if m.companionGear != nil {
		companionGrams = m.companionGear(leaderUserID)
	}
	return encumbrance.Load{
		PersonalGrams:       m.personalGrams(leaderUserID),
		CompanionGrams:      companionGrams,
		CargoGrams:          cargoGrams,
		CapacityGrams:       memberGrams + mountGrams,
		MemberCapacityGrams: memberGrams,
		MountCapacityGrams:  mountGrams,
	}, true
}

// leaderCapacity is the leader's own share: the base, their Strength, and
// their largest carried pack.
func (m *EncumbranceModule) leaderCapacity(leaderUserID, baseGrams, strengthGrams int) int {
	strength, pack := 0, 0
	if m.userLookup != nil {
		if user := m.userLookup(leaderUserID); user != nil && user.Character != nil {
			strength = user.Character.Stats.Strength.ValueAdj
			pack = company.BestPackGrams(user.Character.Items)
		}
	}
	return encumbrance.MemberCapacity(baseGrams, strengthGrams, strength, pack)
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
	// Weight reads the base data, so an item holding a stale spec copy
	// (from cargo, an enchantment, an old save) is still weighed.
	for i := range user.Character.Items {
		total += user.Character.Items[i].Weight()
	}
	for _, item := range user.Character.Equipment.GetAllItems() {
		total += item.Weight()
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
	updated, err := cargo.DepositUses(matchItem.ItemId, m.partialUses(matchItem), 1)
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
	updated, uses, err := cargo.WithdrawOne(itemId)
	if err != nil {
		return "You can't take that from the cargo."
	}
	m.cargo[user.UserId] = updated
	if err := m.saveLocked(); err != nil {
		m.cargo[user.UserId] = cargo
		return err.Error()
	}
	newItem := m.newCargoItem(itemId)
	if uses > 0 {
		newItem.Uses = uses
	}
	user.Character.StoreItem(newItem)
	return fmt.Sprintf(`You take the <ansi fg="item">%s</ansi> from the company cargo.`, newItem.DisplayName())
}

// partialUses is the uses a partly used item has left, or 0 for a full
// one (or an item without uses), so it stacks apart in cargo (Phase 32f).
func (m *EncumbranceModule) partialUses(itm items.Item) int {
	spec, ok := m.itemSpec(itm.ItemId)
	if !ok || spec.Uses <= 0 || itm.Uses <= 0 || itm.Uses >= spec.Uses {
		return 0
	}
	return itm.Uses
}

// CargoContents implements encumbrance.CargoProvider: a copy of the
// leader's stacks.
func (m *EncumbranceModule) CargoContents(leaderUserID int) []encumbrance.CargoStack {
	m.mu.Lock()
	defer m.mu.Unlock()
	cargo, ok := m.cargo[leaderUserID]
	if !ok || len(cargo.Stacks) == 0 {
		return nil
	}
	return append([]encumbrance.CargoStack(nil), cargo.Stacks...)
}

// ConsumeCargoUse implements encumbrance.CargoProvider: one use from one
// item, a partly used one first, saved (and rolled back if the save
// fails).
func (m *EncumbranceModule) ConsumeCargoUse(leaderUserID, itemId int) error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	fullUses := 0
	if spec, ok := m.itemSpec(itemId); ok {
		fullUses = spec.Uses
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cargo, ok := m.cargo[leaderUserID]
	if !ok {
		return encumbrance.ErrInsufficientCargo
	}
	updated, err := cargo.ConsumeUse(itemId, fullUses)
	if err != nil {
		return err
	}
	m.cargo[leaderUserID] = updated
	if err := m.saveLocked(); err != nil {
		m.cargo[leaderUserID] = cargo
		return err
	}
	return nil
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
		fmt.Sprintf("You %.1f kg, companions %.1f kg, cargo %.1f kg.", float64(load.PersonalGrams)/1000, float64(load.CompanionGrams)/1000, float64(load.CargoGrams)/1000),
		fmt.Sprintf("Capacity: members %.1f kg, horses %.1f kg.", float64(load.MemberCapacityGrams)/1000, float64(load.MountCapacityGrams)/1000),
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
			line := fmt.Sprintf("  %s x%d", stackItem.DisplayName(), s.Count)
			if s.Uses > 0 {
				line += fmt.Sprintf(" (%d %s left)", s.Uses, pluralUses(s.Uses))
			}
			lines = append(lines, line)
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

func pluralUses(n int) string {
	if n == 1 {
		return "use"
	}
	return "uses"
}

// parseConfig normalizes the configured member base, the Strength bonus
// (both in kg), and the load-band table, rejecting a malformed band rather
// than applying a guess, matching modules/weather's parseBiomeTables.
func parseConfig(baseRaw, strengthRaw, bandsRaw any) (int, int, []encumbrance.LoadBand) {
	baseGrams := max(int(configFloat(baseRaw)*1000), 0)
	strengthGrams := max(int(configFloat(strengthRaw)*1000), 0)

	bands := []encumbrance.LoadBand{}
	list, ok := bandsRaw.([]any)
	if !ok {
		return baseGrams, strengthGrams, bands
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
	return baseGrams, strengthGrams, bands
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
