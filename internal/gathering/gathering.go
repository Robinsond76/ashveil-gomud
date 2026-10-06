// Package gathering is the pure rules of Phase 40a2: what a company can
// gather from a room (herbs, firewood, fish, game), the pools that deplete
// and regrow in real time, and the yield rolls. It knows no engine type;
// modules/gathering binds it to rooms, companies and the clock.
package gathering

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind is one gathering resource; the ids match the rooms package's.
type Kind string

const (
	Herbs    Kind = "herbs"
	Firewood Kind = "firewood"
	Fishing  Kind = "fishing"
	Game     Kind = "game"
)

// Kinds is every kind in display order.
var Kinds = []Kind{Herbs, Firewood, Fishing, Game}

// Valid reports whether k is a gathering kind.
func (k Kind) Valid() bool {
	for _, have := range Kinds {
		if have == k {
			return true
		}
	}
	return false
}

// Rand returns 0..n-1 (util.Rand's shape), so every roll is deterministic in
// tests.
type Rand func(n int) int

// Pct rolls a percentage chance.
func (r Rand) Pct(chance int) bool {
	if chance <= 0 {
		return false
	}
	return chance >= 100 || r(100) < chance
}

// Between rolls lo..hi inclusive.
func (r Rand) Between(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + r(hi-lo+1)
}

// Drop is a count of one item.
type Drop struct {
	ItemID int
	Count  int
}

// Weighted is one weighted entry of a yield table.
type Weighted struct {
	ItemID int
	Weight int
}

// Table is a weighted yield table.
type Table []Weighted

// Pick rolls one item id; 0 for an empty table.
func (t Table) Pick(rng Rand) int {
	total := 0
	for _, w := range t {
		if w.Weight > 0 {
			total += w.Weight
		}
	}
	if total == 0 {
		return 0
	}
	pick := rng(total)
	for _, w := range t {
		if w.Weight <= 0 {
			continue
		}
		if pick < w.Weight {
			return w.ItemID
		}
		pick -= w.Weight
	}
	return t[len(t)-1].ItemID
}

// Rule is one kind's timing, pool and cost.
type Rule struct {
	Duration time.Duration // real time the work takes
	PoolMax  int           // charges a room holds
	Regrow   time.Duration // real time to regrow one charge
	EffortPc int           // percent of one step's strain each attempt costs
}

// ItemIDs are the items the rules name.
type ItemIDs struct {
	Firewood     int
	DampFirewood int
	FishingLine  int
	RawFish      int
	BitterWeed   int
	RawMeat      int
}

// Settings is every number of the phase, from config.
type Settings struct {
	Rules map[Kind]Rule
	Items ItemIDs

	HuntEncounterBonus int // points added to the room's encounter chance

	HerbMin, HerbMax   int // base herbs per gather
	ForageLevelsPerOne int // a Forage specialist adds one herb per this many levels
	KnifeBonus         int // a bladed weapon in the company adds this many herbs
	DarkPct            int // herb yield in darkness
	BitterWeedPct      int // chance an unskilled gather returns one bitter weed
	ScribePct          int // chance a Scribe finds one rarer herb

	FirewoodBundles int // bundles per gather
	ToolMultiplier  int // a hatchet or axe multiplies the bundles
	FieldSmithBonus int // a Field Smith adds this many bundles
	WetWeather      []string

	FishAttempts  int // chances to land a fish
	FishPct       int // percent per chance
	RangerFishPct int // points a Forage specialist adds per chance
	LineBreakPct  int // chance a fishing line breaks after a gather
	GamePct       int // base chance a hunt brings something down
	GameMeat      int // raw meat per success
	ForageMeat    int // extra meat with a Forage specialist
	RangerGamePct int // points per specialist level
	GameCapPct    int // ceiling on the chance
	SnarePct      int // percent of the chance kept when setting snares
	GoodsPct      int // chance a successful hunt also yields a hide or pelt
	Herb          map[string]Table
	RareHerb      map[string]Table
	Fish          map[string]Table
	Goods         map[string]Table
}

// DefaultSettings are the shipped numbers; config overlays them.
func DefaultSettings() Settings {
	return Settings{
		Rules: map[Kind]Rule{
			Herbs:    {Duration: 20 * time.Second, PoolMax: 3, Regrow: 20 * time.Minute, EffortPc: 100},
			Firewood: {Duration: 20 * time.Second, PoolMax: 4, Regrow: 20 * time.Minute, EffortPc: 100},
			Fishing:  {Duration: 30 * time.Second, PoolMax: 4, Regrow: 20 * time.Minute, EffortPc: 50},
			Game:     {Duration: 30 * time.Second, PoolMax: 2, Regrow: 40 * time.Minute, EffortPc: 100},
		},
		Items:              ItemIDs{Firewood: 40, DampFirewood: 41, FishingLine: 42, RawFish: 43, BitterWeed: 44, RawMeat: 29},
		HuntEncounterBonus: 10,
		HerbMin:            1,
		HerbMax:            2,
		ForageLevelsPerOne: 2,
		KnifeBonus:         1,
		DarkPct:            50,
		BitterWeedPct:      15,
		ScribePct:          10,
		FirewoodBundles:    2,
		ToolMultiplier:     2,
		FieldSmithBonus:    1,
		WetWeather:         []string{"rain", "storm"},
		FishAttempts:       2,
		FishPct:            40,
		RangerFishPct:      10,
		LineBreakPct:       5,
		GamePct:            50,
		GameMeat:           2,
		ForageMeat:         1,
		RangerGamePct:      5,
		GameCapPct:         90,
		SnarePct:           50,
		GoodsPct:           35,
		Herb:               map[string]Table{},
		RareHerb:           map[string]Table{},
		Fish:               map[string]Table{},
		Goods:              map[string]Table{},
	}
}

// AnyZone is the table key every zone falls back to.
const AnyZone = "*"

// TableFor is the zone's table, else the fallback's.
func TableFor(tables map[string]Table, zone string) Table {
	for name, t := range tables {
		if strings.EqualFold(name, zone) && len(t) > 0 {
			return t
		}
	}
	return tables[AnyZone]
}

// IsWet reports whether the weather condition is heavy rain.
func (s Settings) IsWet(condition string) bool {
	condition = strings.ToLower(strings.TrimSpace(condition))
	for _, w := range s.WetWeather {
		if strings.ToLower(w) == condition {
			return true
		}
	}
	return false
}

// Add puts count of an item into a drop list, merging repeats.
func Add(drops []Drop, id, count int) []Drop {
	if id <= 0 || count <= 0 {
		return drops
	}
	for i := range drops {
		if drops[i].ItemID == id {
			drops[i].Count += count
			return drops
		}
	}
	return append(drops, Drop{ItemID: id, Count: count})
}

// HerbContext is what shapes a herb gather.
type HerbContext struct {
	Zone        string
	ForageLevel int  // best Forage specialist level present
	HasForaging bool // a Forage specialist is present
	Knife       bool // a bladed weapon in the company
	Dark        bool
	Scribe      bool // a Scribe in the company
}

// RollHerbs rolls one herb gather. weed is true when the pick was bitter
// weed instead of anything useful.
func (s Settings) RollHerbs(c HerbContext, rng Rand) (drops []Drop, weed bool) {
	if !c.HasForaging && rng.Pct(s.BitterWeedPct) {
		return []Drop{{ItemID: s.Items.BitterWeed, Count: 1}}, true
	}
	count := rng.Between(s.HerbMin, s.HerbMax)
	if c.HasForaging && s.ForageLevelsPerOne > 0 {
		count += c.ForageLevel / s.ForageLevelsPerOne
	}
	if c.Knife {
		count += s.KnifeBonus
	}
	if c.Dark {
		count = (count*s.DarkPct + 99) / 100
	}
	if count < 1 {
		count = 1
	}
	table := TableFor(s.Herb, c.Zone)
	for i := 0; i < count; i++ {
		drops = Add(drops, table.Pick(rng), 1)
	}
	if c.Scribe && rng.Pct(s.ScribePct) {
		if rare := TableFor(s.RareHerb, c.Zone); len(rare) > 0 {
			drops = Add(drops, rare.Pick(rng), 1)
		}
	}
	return drops, false
}

// FirewoodContext is what shapes a firewood gather.
type FirewoodContext struct {
	Axe        bool // a hatchet or axe in the company
	FieldSmith bool // a Field Smith is present
	Wet        bool
}

// RollFirewood returns how many dry and how many damp bundles come back.
func (s Settings) RollFirewood(c FirewoodContext) (dry, damp int) {
	n := s.FirewoodBundles
	if c.Axe && s.ToolMultiplier > 1 {
		n *= s.ToolMultiplier
	}
	if c.FieldSmith {
		n += s.FieldSmithBonus
	}
	if n < 1 {
		n = 1
	}
	if c.Wet {
		dry = n / 2
		return dry, n - dry
	}
	return n, 0
}

// FishContext is what shapes a fishing gather.
type FishContext struct {
	Zone   string
	Ranger bool // a Forage specialist is present
}

// RollFish returns the fish landed.
func (s Settings) RollFish(c FishContext, rng Rand) []Drop {
	pct := s.FishPct
	if c.Ranger {
		pct += s.RangerFishPct
	}
	table := TableFor(s.Fish, c.Zone)
	var drops []Drop
	for i := 0; i < s.FishAttempts; i++ {
		if rng.Pct(pct) {
			drops = Add(drops, table.Pick(rng), 1)
		}
	}
	return drops
}

// HuntContext is what shapes a hunt.
type HuntContext struct {
	Zone        string
	ForageLevel int  // best Forage specialist level present
	HasForaging bool // a Forage specialist is present
	Snares      bool // no ranged weapon: the company sets snares
}

// HuntChance is the percent chance a hunt brings something down.
func (s Settings) HuntChance(c HuntContext) int {
	pct := s.GamePct
	if c.HasForaging {
		pct += s.RangerGamePct * c.ForageLevel
	}
	if pct > s.GameCapPct {
		pct = s.GameCapPct
	}
	if c.Snares {
		pct = pct * s.SnarePct / 100
	}
	return pct
}

// RollHunt returns what the hunt brought back; nil when it failed.
func (s Settings) RollHunt(c HuntContext, rng Rand) []Drop {
	if !rng.Pct(s.HuntChance(c)) {
		return nil
	}
	meat := s.GameMeat
	if c.HasForaging {
		meat += s.ForageMeat
	}
	drops := Add(nil, s.Items.RawMeat, meat)
	if goods := TableFor(s.Goods, c.Zone); len(goods) > 0 && rng.Pct(s.GoodsPct) {
		drops = Add(drops, goods.Pick(rng), 1)
	}
	return drops
}

// Pool is one room's stock of one resource. A room never gathered from has
// no pool and is full.
type Pool struct {
	Charges       int       `yaml:"charges"`
	LastRegrowUTC time.Time `yaml:"lastregrowutc"`
}

// Regrown is the pool after real time has passed: one charge per Regrow
// elapsed, the unfinished part of the interval carried over. full is true
// once it is back at PoolMax (and may be forgotten).
func (p Pool) Regrown(rule Rule, now time.Time) (Pool, bool) {
	if p.Charges >= rule.PoolMax {
		return Pool{Charges: rule.PoolMax, LastRegrowUTC: now.UTC()}, true
	}
	if rule.Regrow > 0 && now.After(p.LastRegrowUTC) {
		steps := int(now.Sub(p.LastRegrowUTC) / rule.Regrow)
		if steps > 0 {
			p.Charges += steps
			if p.Charges >= rule.PoolMax {
				return Pool{Charges: rule.PoolMax, LastRegrowUTC: now.UTC()}, true
			}
			p.LastRegrowUTC = p.LastRegrowUTC.Add(time.Duration(steps) * rule.Regrow)
		}
	}
	return p, p.Charges >= rule.PoolMax
}

// Ledger is every room's pools: roomId -> resource -> pool. Only rooms that
// have been gathered from appear.
type Ledger struct {
	Rooms map[int]map[Kind]Pool `yaml:"rooms"`
}

// NewLedger is an empty ledger.
func NewLedger() Ledger { return Ledger{Rooms: map[int]map[Kind]Pool{}} }

// Clone copies the ledger.
func (l Ledger) Clone() Ledger {
	out := NewLedger()
	for room, pools := range l.Rooms {
		copied := make(map[Kind]Pool, len(pools))
		for k, p := range pools {
			copied[k] = p
		}
		out.Rooms[room] = copied
	}
	return out
}

// Charges is how many charges the room's resource holds at now.
func (l Ledger) Charges(room int, k Kind, rule Rule, now time.Time) int {
	p, ok := l.Rooms[room][k]
	if !ok {
		return rule.PoolMax
	}
	grown, _ := p.Regrown(rule, now)
	return grown.Charges
}

// Spend takes one charge, reporting false when the pool is empty. A pool
// that has regrown to full is forgotten, so the ledger only holds rooms in
// recovery.
func (l *Ledger) Spend(room int, k Kind, rule Rule, now time.Time) bool {
	if l.Rooms == nil {
		l.Rooms = map[int]map[Kind]Pool{}
	}
	p, ok := l.Rooms[room][k]
	if !ok {
		p = Pool{Charges: rule.PoolMax, LastRegrowUTC: now.UTC()}
	} else {
		p, _ = p.Regrown(rule, now)
	}
	if p.Charges < 1 {
		l.store(room, k, p)
		return false
	}
	if p.Charges >= rule.PoolMax {
		p.LastRegrowUTC = now.UTC() // a full pool starts regrowing from the first spend
	}
	p.Charges--
	l.store(room, k, p)
	return true
}

func (l *Ledger) store(room int, k Kind, p Pool) {
	if l.Rooms[room] == nil {
		l.Rooms[room] = map[Kind]Pool{}
	}
	l.Rooms[room][k] = p
}

// Prune forgets pools that have regrown to full.
func (l *Ledger) Prune(rules map[Kind]Rule, now time.Time) {
	for room, pools := range l.Rooms {
		for k, p := range pools {
			rule, ok := rules[k]
			if !ok {
				delete(pools, k)
				continue
			}
			if _, full := p.Regrown(rule, now); full {
				delete(pools, k)
			}
		}
		if len(pools) == 0 {
			delete(l.Rooms, room)
		}
	}
}

// RoomIDs lists the rooms that hold a pool, ascending.
func (l Ledger) RoomIDs() []int {
	ids := make([]int, 0, len(l.Rooms))
	for id := range l.Rooms {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// Progress is a company's gathering work in progress (Phase 45): what it is
// doing and how long is left, for the prompt and the status sheet.
type Progress struct {
	Kind      Kind
	Label     string // "gathering herbs", "fishing"
	Total     time.Duration
	Remaining time.Duration
}

// Percent is how much of the work is done, 0..99 while it runs.
func (p Progress) Percent() int {
	if p.Total <= 0 {
		return 0
	}
	done := int((p.Total - p.Remaining) * 100 / p.Total)
	return min(max(done, 0), 99)
}

var (
	progressMu       sync.RWMutex
	progressProvider func(userID int) (Progress, bool)
)

// SetProgressProvider installs the module that knows the work in progress;
// nil removes it.
func SetProgressProvider(fn func(userID int) (Progress, bool)) {
	progressMu.Lock()
	progressProvider = fn
	progressMu.Unlock()
}

// ProgressOf is the leader's gathering work in progress, if any.
func ProgressOf(userID int) (Progress, bool) {
	progressMu.RLock()
	fn := progressProvider
	progressMu.RUnlock()
	if fn == nil {
		return Progress{}, false
	}
	return fn(userID)
}
