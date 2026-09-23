// Package market owns the Phase 19 settlement markets: the durable,
// zone-keyed stock ledger, the configured tracked goods per market zone,
// the round-driven bounded stock drift, load/copyover recovery, and the
// read-only market command. The price and drift math lives in
// internal/market.
//
// A settlement is a zone. Only zones configured under Markets in this
// module's config get a ledger; every other zone has no market.
//
// Like modules/weather, markets react to events.NewRound and never read,
// drive, or advance the shared round counter or world clock.
//
// Vendor buy/sell does not consult market prices yet; market-backed
// trading is deferred to Phase 19b (see the Phase 19 design doc).
package market

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/market"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

//go:embed files/*
var files embed.FS

// GoodStock is the current stock of one tracked good.
type GoodStock struct {
	ItemID int `yaml:"itemid"`
	Stock  int `yaml:"stock"`
}

// ZoneMarket is one settlement's current stock per tracked good.
type ZoneMarket struct {
	Goods []GoodStock `yaml:"goods"`
}

// Stock returns the recorded stock for itemID.
func (z ZoneMarket) Stock(itemID int) (int, bool) {
	for _, g := range z.Goods {
		if g.ItemID == itemID {
			return g.Stock, true
		}
	}
	return 0, false
}

// Registry is the durable, zone-keyed set of market stock ledgers.
type Registry struct {
	Zones map[string]ZoneMarket `yaml:"zones"`
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{Zones: map[string]ZoneMarket{}}
}

// Clone returns a deep copy of the registry.
func (r Registry) Clone() Registry {
	out := Registry{Zones: make(map[string]ZoneMarket, len(r.Zones))}
	for zone, zm := range r.Zones {
		out.Zones[zone] = ZoneMarket{Goods: append([]GoodStock(nil), zm.Goods...)}
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
	data, err := s.plug.ReadBytes("market")
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
	return s.plug.WriteStruct("market", registry)
}

// ErrCorruptStore reports stored ledger data that is present but unusable.
var ErrCorruptStore = errors.New("market: corrupt store")

// wireRegistry mirrors Registry with a pointer stock so a record missing
// its stock (a truncated file) is detectable rather than read as zero.
type wireRegistry struct {
	Zones map[string]struct {
		Goods []struct {
			ItemID int  `yaml:"itemid"`
			Stock  *int `yaml:"stock"`
		} `yaml:"goods"`
	} `yaml:"zones"`
}

// decodeRegistry parses stored bytes. An existing but empty file, or a
// record missing its stock, is corrupt (a truncated write), never an empty
// ledger to re-seed. It drops entries keyed by an empty zone name, goods
// with a non-positive item ID, and repeated records for the same good (the
// first wins). Out-of-range stock is retained and clamped when it is next
// used.
func decodeRegistry(data []byte, registry *Registry) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("%w: empty file", ErrCorruptStore)
	}
	var wire wireRegistry
	if err := yaml.Unmarshal(data, &wire); err != nil {
		return err
	}
	loaded := NewRegistry()
	for zone, zm := range wire.Zones {
		if zone == "" {
			continue
		}
		seen := map[int]bool{}
		goods := []GoodStock{}
		for _, g := range zm.Goods {
			if g.Stock == nil {
				return fmt.Errorf("%w: zone %q item %d has no stock", ErrCorruptStore, zone, g.ItemID)
			}
			if g.ItemID <= 0 || seen[g.ItemID] {
				mudlog.Warn("market: dropping stored stock record", "zone", zone, "itemid", g.ItemID)
				continue
			}
			seen[g.ItemID] = true
			goods = append(goods, GoodStock{ItemID: g.ItemID, Stock: *g.Stock})
		}
		loaded.Zones[zone] = ZoneMarket{Goods: goods}
	}
	*registry = *loaded
	return nil
}

// MarketModule owns the durable market ledgers for one plugin.
type MarketModule struct {
	plug       *plugins.Plugin
	store      Store
	roll       func() uint64
	itemExists func(itemID int) bool
	itemName   func(itemID int) string
	zoneExists func(zone string) bool

	// markets is the configured tracked-goods list per market zone, in
	// config order.
	markets map[string][]market.Good
	zones   map[string]ZoneMarket
	loadErr error

	// mu is a leaf lock: nothing that takes another package's lock is
	// called while it is held.
	mu sync.Mutex
}

// errNotLoaded keeps persistence off until the first successful load, so a
// save can never overwrite the durable ledger with an empty registry.
var errNotLoaded = errors.New("not loaded yet")

// module is the registered instance; the end-to-end test drives it.
var module *MarketModule

func init() {
	m := &MarketModule{
		plug:       plugins.New("market", "1.0"),
		roll:       rand.Uint64,
		itemExists: func(itemID int) bool { return items.GetItemSpec(itemID) != nil },
		itemName: func(itemID int) string {
			if spec := items.GetItemSpec(itemID); spec != nil {
				return spec.Name
			}
			return fmt.Sprintf("item #%d", itemID)
		},
		zoneExists: zoneExists,
		markets:    map[string][]market.Good{},
		zones:      map[string]ZoneMarket{},
		loadErr:    errNotLoaded,
	}
	if err := m.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	m.store = pluginStore{plug: m.plug}
	m.plug.AddUserCommand("market", m.userCommand, true, false)
	m.plug.Callbacks.SetOnLoad(m.load)
	m.plug.Callbacks.SetOnSave(func() {
		if err := m.save(); err != nil {
			mudlog.Error("market: save", "error", err)
		}
	})
	events.RegisterListener(events.NewRound{}, m.onNewRound)
	module = m
}

func zoneExists(zone string) bool {
	for _, name := range rooms.GetAllZoneNames() {
		if name == zone {
			return true
		}
	}
	return false
}

func (m *MarketModule) persistenceAvailable() error {
	if m.loadErr != nil {
		return fmt.Errorf("market: persistence unavailable until a successful reload: %w", m.loadErr)
	}
	if m.store == nil {
		return fmt.Errorf("market: persistence unavailable")
	}
	return nil
}

func (m *MarketModule) save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *MarketModule) saveLocked() error {
	if err := m.persistenceAvailable(); err != nil {
		return err
	}
	if err := m.store.Save(Registry{Zones: m.zones}); err != nil {
		return fmt.Errorf("market: save failed; please retry: %w", err)
	}
	return nil
}

// load parses the configured markets, loads the durable ledger, and seeds
// any configured good that has no stock record yet at its StartStock (a
// first boot or a newly added market). Existing stock, including records
// for goods or zones no longer configured, is kept as-is, so a restart or
// copyover restores exactly the persisted stock. A load failure disables
// market effects until a successful reload rather than overwriting the
// store.
func (m *MarketModule) load() {
	if m.store == nil {
		return
	}
	var markets map[string][]market.Good
	if m.plug != nil {
		markets = parseMarkets(m.plug.Config.Get("Markets"), m.itemExists, m.zoneExists)
	}
	loaded := NewRegistry()
	err := m.store.Load(loaded)

	m.mu.Lock()
	defer m.mu.Unlock()
	if markets != nil {
		m.markets = markets
	}
	if err != nil {
		m.loadErr = err
		mudlog.Error("market: load", "error", err)
		return
	}
	m.zones = loaded.Zones
	if m.zones == nil {
		m.zones = map[string]ZoneMarket{}
	}
	m.loadErr = nil
	if m.seedLocked() {
		if err := m.saveLocked(); err != nil {
			mudlog.Error("market: seed save", "error", err)
		}
	}
}

// seedLocked adds a StartStock record for every configured good missing
// one, reporting whether anything changed.
func (m *MarketModule) seedLocked() bool {
	changed := false
	for zone, goods := range m.markets {
		zm := m.zones[zone]
		for _, g := range goods {
			if _, ok := zm.Stock(g.ItemID); ok {
				continue
			}
			zm.Goods = append(zm.Goods, GoodStock{ItemID: g.ItemID, Stock: g.StartStock})
			changed = true
		}
		m.zones[zone] = zm
	}
	return changed
}

// onNewRound drifts every configured good's stock one bounded step toward
// its target and saves once if anything moved. It only reacts to the
// event; it never reads or advances the round counter.
func (m *MarketModule) onNewRound(e events.Event) events.ListenerReturn {
	if _, ok := e.(events.NewRound); !ok {
		return events.Continue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.persistenceAvailable() != nil {
		return events.Continue
	}
	changed := false
	for zone, goods := range m.markets {
		zm, ok := m.zones[zone]
		if !ok {
			continue
		}
		for _, g := range goods {
			for i := range zm.Goods {
				if zm.Goods[i].ItemID != g.ItemID {
					continue
				}
				next := g.DriftStock(zm.Goods[i].Stock, m.roll())
				if next != zm.Goods[i].Stock {
					zm.Goods[i].Stock = next
					changed = true
				}
				break
			}
		}
	}
	if changed {
		if err := m.saveLocked(); err != nil {
			mudlog.Error("market: round save", "error", err)
		}
	}
	return events.Continue
}

// Quote is one tracked good's current market listing.
type Quote struct {
	ItemID int
	Price  int
	Level  string
}

// Quotes returns the zone's current listings in config order, and whether
// the zone is a market. It returns an error while persistence is
// unavailable.
func (m *MarketModule) Quotes(zone string) ([]Quote, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.persistenceAvailable(); err != nil {
		return nil, false, err
	}
	goods, ok := m.markets[zone]
	if !ok {
		return nil, false, nil
	}
	zm := m.zones[zone]
	quotes := make([]Quote, 0, len(goods))
	for _, g := range goods {
		stock, ok := zm.Stock(g.ItemID)
		if !ok {
			continue
		}
		quotes = append(quotes, Quote{ItemID: g.ItemID, Price: g.PriceForStock(stock), Level: g.StockLevel(stock)})
	}
	return quotes, true, nil
}

func (m *MarketModule) userCommand(_ string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
	if room == nil {
		return true, nil
	}
	quotes, ok, err := m.Quotes(room.Zone)
	if err != nil {
		user.SendText("The market ledgers are unavailable right now.")
		return true, nil
	}
	if !ok {
		user.SendText("There's no market here.")
		return true, nil
	}
	names := make([]string, len(quotes))
	width := 0
	for i, q := range quotes {
		names[i] = m.itemName(q.ItemID)
		width = max(width, len(names[i]))
	}
	lines := []string{fmt.Sprintf(`Market prices in <ansi fg="zone">%s</ansi>:`, room.Zone)}
	for i, q := range quotes {
		lines = append(lines, fmt.Sprintf(`  <ansi fg="itemname">%-*s</ansi>  <ansi fg="gold">%d gold</ansi>  (%s)`, width, names[i], q.Price, q.Level))
	}
	user.SendText(strings.Join(lines, "\n"))
	return true, nil
}

// parseMarkets normalizes the configured market list. An invalid good or
// one naming a missing item spec is warned about and omitted. A zone that
// lists the same item twice, appears in more than one entry, names an
// unknown zone, or has no valid goods gets no market.
func parseMarkets(raw any, itemExists func(int) bool, zoneExists func(string) bool) map[string][]market.Good {
	markets := map[string][]market.Good{}
	if raw == nil {
		return markets
	}
	list, ok := raw.([]any)
	if !ok {
		mudlog.Warn("market: Markets config is not a list")
		return markets
	}
	entries := []map[string]any{}
	zoneCount := map[string]int{}
	for _, entry := range list {
		fields := stringMap(entry)
		if fields == nil {
			mudlog.Warn("market: skipping market entry that is not a map")
			continue
		}
		zone, _ := fields["zone"].(string)
		zone = strings.TrimSpace(zone)
		if zone == "" {
			mudlog.Warn("market: market entry missing a zone name")
			continue
		}
		fields["zone"] = zone
		zoneCount[zone]++
		entries = append(entries, fields)
	}
	for _, fields := range entries {
		zone := fields["zone"].(string)
		if zoneCount[zone] > 1 {
			mudlog.Warn("market: zone listed in more than one market entry; no market", "zone", zone)
			continue
		}
		if zoneExists != nil && !zoneExists(zone) {
			mudlog.Warn("market: unknown market zone", "zone", zone)
			continue
		}
		if goods, ok := parseGoods(zone, fields["goods"], itemExists); ok {
			markets[zone] = goods
		}
	}
	return markets
}

// parseGoods parses one zone's goods, reporting false when the zone gets
// no market.
func parseGoods(zone string, raw any, itemExists func(int) bool) ([]market.Good, bool) {
	goodList, _ := raw.([]any)
	goods := []market.Good{}
	seenItems := map[int]bool{}
	for _, goodRaw := range goodList {
		gf := stringMap(goodRaw)
		if gf == nil {
			mudlog.Warn("market: skipping good that is not a map", "zone", zone)
			continue
		}
		g := market.Good{
			ItemID:      configInt(gf["itemid"]),
			BasePrice:   configInt(gf["baseprice"]),
			MinPrice:    configInt(gf["minprice"]),
			MaxPrice:    configInt(gf["maxprice"]),
			MaxStock:    configInt(gf["maxstock"]),
			TargetStock: configInt(gf["targetstock"]),
			StartStock:  configInt(gf["startstock"]),
			DriftStep:   configInt(gf["driftstep"]),
		}
		if err := g.Validate(); err != nil {
			mudlog.Warn("market: invalid good", "zone", zone, "itemid", g.ItemID, "error", err)
			continue
		}
		if seenItems[g.ItemID] {
			mudlog.Warn("market: zone lists an item twice; no market", "zone", zone, "itemid", g.ItemID)
			return nil, false
		}
		seenItems[g.ItemID] = true
		if itemExists != nil && !itemExists(g.ItemID) {
			mudlog.Warn("market: good has no item spec", "zone", zone, "itemid", g.ItemID)
			continue
		}
		goods = append(goods, g)
	}
	if len(goods) == 0 {
		mudlog.Warn("market: zone has no valid goods; no market", "zone", zone)
		return nil, false
	}
	return goods, true
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
		if value == math.Trunc(value) && math.Abs(value) <= math.MaxInt32 {
			return int(value)
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return n
		}
	}
	return 0
}
