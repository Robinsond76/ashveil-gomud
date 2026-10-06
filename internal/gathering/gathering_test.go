package gathering

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// seq returns the given values in turn (then the last one forever).
func seq(values ...int) Rand {
	i := 0
	return func(n int) int {
		v := values[len(values)-1]
		if i < len(values) {
			v = values[i]
			i++
		}
		if v >= n {
			v = n - 1
		}
		return v
	}
}

func testSettings() Settings {
	s := DefaultSettings()
	s.Herb["*"] = Table{{ItemID: 30018, Weight: 1}}
	s.RareHerb["*"] = Table{{ItemID: 30008, Weight: 1}}
	s.Fish["*"] = Table{{ItemID: 43, Weight: 1}}
	s.Goods["*"] = Table{{ItemID: 28, Weight: 1}}
	return s
}

func TestTablePick(t *testing.T) {
	table := Table{{ItemID: 1, Weight: 3}, {ItemID: 2, Weight: 1}}
	assert.Equal(t, 1, table.Pick(seq(0)))
	assert.Equal(t, 1, table.Pick(seq(2)))
	assert.Equal(t, 2, table.Pick(seq(3)))
	assert.Equal(t, 0, Table{}.Pick(seq(0)), "an empty table picks nothing")
	assert.Equal(t, 3, TableFor(map[string]Table{"Road": {{ItemID: 3, Weight: 1}}, AnyZone: {{ItemID: 9, Weight: 1}}}, "road").Pick(seq(0)), "zones match without case")
	assert.Equal(t, 9, TableFor(map[string]Table{AnyZone: {{ItemID: 9, Weight: 1}}}, "Elsewhere").Pick(seq(0)), "others fall back")
}

func TestRollHerbsFollowsTheTable(t *testing.T) {
	s := testSettings()
	// no weed (99), then 1..2 → 1, one pick.
	drops, weed := s.RollHerbs(HerbContext{Zone: "x", HasForaging: true}, seq(0, 0))
	assert.False(t, weed)
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 1}}, drops)

	// base 2 + level 4 / 2 + knife 1 = 5 herbs.
	drops, _ = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true, ForageLevel: 4, Knife: true}, seq(1, 0))
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 5}}, drops, "a ranger and a blade add herbs")

	// darkness halves it, never below one.
	drops, _ = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true, ForageLevel: 4, Knife: true, Dark: true}, seq(1, 0))
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 3}}, drops)
	drops, _ = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true, Dark: true}, seq(0, 0))
	assert.Equal(t, 1, drops[0].Count)
}

func TestBitterWeedOnlyWithoutAForager(t *testing.T) {
	s := testSettings()
	// roll 0 < 15: weed.
	drops, weed := s.RollHerbs(HerbContext{Zone: "x"}, seq(0))
	assert.True(t, weed)
	assert.Equal(t, []Drop{{ItemID: s.Items.BitterWeed, Count: 1}}, drops)

	// roll 99: no weed, then the normal gather.
	drops, weed = s.RollHerbs(HerbContext{Zone: "x"}, seq(99, 0, 0))
	assert.False(t, weed)
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 1}}, drops)

	// a forager is never fooled, however the dice fall.
	_, weed = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true}, seq(0))
	assert.False(t, weed)
}

func TestScribeMayFindARarerHerb(t *testing.T) {
	s := testSettings()
	// forager: base 1, one pick, then the scribe roll (0 < 10) finds one rare herb.
	drops, _ := s.RollHerbs(HerbContext{Zone: "x", HasForaging: true, Scribe: true}, seq(0, 0, 0, 0))
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 1}, {ItemID: 30008, Count: 1}}, drops)
	drops, _ = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true, Scribe: true}, seq(0, 0, 50))
	assert.Equal(t, []Drop{{ItemID: 30018, Count: 1}}, drops, "a miss on the 10% roll")
	drops, _ = s.RollHerbs(HerbContext{Zone: "x", HasForaging: true}, seq(0, 0, 0, 0))
	assert.Len(t, drops, 1, "no Scribe, no rare herb")
}

func TestRollFirewood(t *testing.T) {
	s := testSettings()
	dry, damp := s.RollFirewood(FirewoodContext{})
	assert.Equal(t, 2, dry)
	assert.Zero(t, damp)
	dry, _ = s.RollFirewood(FirewoodContext{Axe: true})
	assert.Equal(t, 4, dry, "an axe doubles it")
	dry, _ = s.RollFirewood(FirewoodContext{Axe: true, FieldSmith: true})
	assert.Equal(t, 5, dry, "and a Field Smith adds one")
	dry, damp = s.RollFirewood(FirewoodContext{Wet: true})
	assert.Equal(t, 1, dry)
	assert.Equal(t, 1, damp, "rain halves what is fit to burn")
	dry, damp = s.RollFirewood(FirewoodContext{Wet: true, Axe: true})
	assert.Equal(t, 2, dry)
	assert.Equal(t, 2, damp)
	assert.True(t, s.IsWet("Storm"))
	assert.False(t, s.IsWet("clear"))
}

func TestRollFish(t *testing.T) {
	s := testSettings()
	// two chances at 40: 39 hits, 40 misses.
	drops := s.RollFish(FishContext{Zone: "x"}, seq(39, 0, 40))
	assert.Equal(t, []Drop{{ItemID: 43, Count: 1}}, drops)
	assert.Empty(t, s.RollFish(FishContext{Zone: "x"}, seq(99)))
	// a ranger: 50 per chance, so 49 and 49 both hit.
	drops = s.RollFish(FishContext{Zone: "x", Ranger: true}, seq(49, 0, 49, 0))
	assert.Equal(t, []Drop{{ItemID: 43, Count: 2}}, drops)
}

func TestHuntChanceAndYield(t *testing.T) {
	s := testSettings()
	assert.Equal(t, 50, s.HuntChance(HuntContext{}))
	assert.Equal(t, 25, s.HuntChance(HuntContext{Snares: true}), "snares keep half")
	assert.Equal(t, 70, s.HuntChance(HuntContext{HasForaging: true, ForageLevel: 4}), "5 points per level")
	assert.Equal(t, 90, s.HuntChance(HuntContext{HasForaging: true, ForageLevel: 20}), "capped at 90")
	assert.Equal(t, 45, s.HuntChance(HuntContext{HasForaging: true, ForageLevel: 20, Snares: true}))

	drops := s.RollHunt(HuntContext{Zone: "x"}, seq(49, 99))
	assert.Equal(t, []Drop{{ItemID: s.Items.RawMeat, Count: 2}}, drops, "two meat, no hide on a high roll")
	drops = s.RollHunt(HuntContext{Zone: "x", HasForaging: true, ForageLevel: 1}, seq(0, 0, 0))
	assert.Equal(t, []Drop{{ItemID: s.Items.RawMeat, Count: 3}, {ItemID: 28, Count: 1}}, drops, "a forager adds meat; the goods roll adds a hide")
	assert.Nil(t, s.RollHunt(HuntContext{Zone: "x"}, seq(50)), "50 is a miss at 50")
}

func TestPoolSpendRefuseAndRegrow(t *testing.T) {
	rule := Rule{PoolMax: 3, Regrow: 20 * time.Minute}
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	l := NewLedger()
	assert.Equal(t, 3, l.Charges(1, Herbs, rule, now), "a room never gathered from is full")
	assert.Empty(t, l.Rooms, "and holds no record")

	for i := 0; i < 3; i++ {
		assert.True(t, l.Spend(1, Herbs, rule, now))
	}
	assert.False(t, l.Spend(1, Herbs, rule, now), "an empty pool refuses")
	assert.Zero(t, l.Charges(1, Herbs, rule, now))
	assert.Equal(t, 3, l.Charges(2, Herbs, rule, now), "pools are per room")
	assert.Equal(t, 3, l.Charges(1, Firewood, rule, now), "and per resource")

	assert.Zero(t, l.Charges(1, Herbs, rule, now.Add(19*time.Minute)))
	assert.Equal(t, 1, l.Charges(1, Herbs, rule, now.Add(20*time.Minute)), "one charge per 20 minutes")
	assert.Equal(t, 2, l.Charges(1, Herbs, rule, now.Add(41*time.Minute)))
	assert.Equal(t, 3, l.Charges(1, Herbs, rule, now.Add(3*time.Hour)), "back to full, never past it")

	// Regrowth keeps the unfinished part of an interval: 30 minutes later one
	// charge is back and ten minutes of the next are already earned.
	assert.True(t, l.Spend(1, Herbs, rule, now.Add(30*time.Minute)))
	assert.Zero(t, l.Charges(1, Herbs, rule, now.Add(30*time.Minute)))
	assert.Equal(t, 1, l.Charges(1, Herbs, rule, now.Add(40*time.Minute)), "the next charge comes at the 40 minute mark")
}

func TestLedgerSurvivesASaveAndLoad(t *testing.T) {
	rule := Rule{PoolMax: 2, Regrow: 40 * time.Minute}
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	rules := map[Kind]Rule{Game: rule, Herbs: {PoolMax: 3, Regrow: 20 * time.Minute}}
	l := NewLedger()
	assert.True(t, l.Spend(7, Game, rule, now))
	assert.True(t, l.Spend(7, Game, rule, now))

	data, err := yaml.Marshal(l)
	assert.NoError(t, err)
	var loaded Ledger
	assert.NoError(t, yaml.Unmarshal(data, &loaded))
	assert.Zero(t, loaded.Charges(7, Game, rule, now), "a restart never refills a pool early")
	assert.Equal(t, 1, loaded.Charges(7, Game, rule, now.Add(40*time.Minute)), "and real time still regrows it")

	loaded.Prune(rules, now.Add(2*time.Hour))
	assert.Empty(t, loaded.Rooms, "pools back at full are forgotten")
	l.Prune(rules, now)
	assert.Equal(t, []int{7}, l.RoomIDs(), "a pool in recovery is kept")
	assert.True(t, Kind("herbs").Valid())
	assert.False(t, Kind("gold").Valid())

	copy := l.Clone()
	assert.True(t, copy.Spend(7, Game, rule, now.Add(40*time.Minute)))
	assert.Equal(t, 1, l.Charges(7, Game, rule, now.Add(40*time.Minute)), "a clone is independent")
}
