package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/sky"
)

func withTestBiomes(t *testing.T) {
	t.Helper()
	saved := biomes
	biomes = map[string]*BiomeInfo{
		`default`: {BiomeId: `default`, Name: `Default`, Symbol: `•`, LitArea: true},
		`forest`:  {BiomeId: `forest`, Name: `Forest`, Symbol: `♣`},
		`house`:   {BiomeId: `house`, Name: `House`, Symbol: `⌂`, LitArea: true, Indoor: true},
	}
	t.Cleanup(func() { biomes = saved })
}

func TestIsIndoorFromBiome(t *testing.T) {
	withTestBiomes(t)
	if (&Room{Biome: `forest`}).IsIndoor() {
		t.Fatal("forest should be outdoors")
	}
	if !(&Room{Biome: `house`}).IsIndoor() {
		t.Fatal("house should be indoors")
	}
}

func TestIsIndoorTagOverrides(t *testing.T) {
	withTestBiomes(t)
	if !(&Room{Biome: `forest`, Tags: []string{`indoor`}}).IsIndoor() {
		t.Fatal("indoor tag should make a forest room indoors")
	}
	if (&Room{Biome: `house`, Tags: []string{`Outdoor`}}).IsIndoor() {
		t.Fatal("outdoor tag should make a house room outdoors")
	}
}

func TestSkyViewOutdoorUsesOwnZone(t *testing.T) {
	withTestBiomes(t)
	r := &Room{RoomId: 1, Zone: `Dunmar`, Biome: `forest`}
	gd := gametime.GameDate{RoundNumber: 400, RoundsPerDay: 100, DayNumber: 4, Hour24: 20, Night: true, MoonCount: 1}
	v := r.skyView(func(int) *Room { return nil }, gd, 8)
	if v.Indoor || v.WeatherZone != `Dunmar` || !v.Night || !v.HasMoon || v.Moon != sky.Full {
		t.Fatalf("unexpected outdoor sky view: %+v", v)
	}
	gd.MoonCount = 0
	if v := r.skyView(func(int) *Room { return nil }, gd, 8); v.HasMoon {
		t.Fatal("moon_count 0 should mean no moon")
	}
}

func TestOutdoorGlimpsePicksSortedNonSecretExit(t *testing.T) {
	withTestBiomes(t)
	outside := map[int]*Room{
		2: {RoomId: 2, Zone: `Secret`, Biome: `forest`},
		3: {RoomId: 3, Zone: `Inn`, Biome: `house`},
		4: {RoomId: 4, Zone: `Road`, Biome: `forest`},
		5: {RoomId: 5, Zone: `Yard`, Biome: `forest`},
	}
	r := &Room{RoomId: 1, Zone: `Inn`, Biome: `house`, Exits: map[string]exit.RoomExit{
		`a-hatch`: {RoomId: 2, Secret: true},
		`b-hall`:  {RoomId: 3},
		`west`:    {RoomId: 5},
		`north`:   {RoomId: 4},
		`gone`:    {RoomId: 99},
	}}
	load := func(id int) *Room { return outside[id] }
	v := r.skyView(load, gametime.GameDate{RoundsPerDay: 100}, 8)
	if !v.Indoor || v.GlimpseExit != `north` || v.WeatherZone != `Road` {
		t.Fatalf("expected a glimpse north into Road, got %+v", v)
	}

	sealed := &Room{RoomId: 6, Zone: `Inn`, Biome: `house`, Exits: map[string]exit.RoomExit{`b-hall`: {RoomId: 3}}}
	if v := sealed.skyView(load, gametime.GameDate{RoundsPerDay: 100}, 8); v.GlimpseExit != `` {
		t.Fatalf("no outdoor exit should mean no glimpse, got %+v", v)
	}
}
