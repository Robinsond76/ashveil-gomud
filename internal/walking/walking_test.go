package walking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerrainCost(t *testing.T) {
	table := DefaultTerrain()
	cases := []struct {
		name string
		in   Terrain
		want int
	}{
		{"city is free", Terrain{Biome: "city"}, 0},
		{"slums is free", Terrain{Biome: "slums"}, 0},
		{"house is free", Terrain{Biome: "house"}, 0},
		{"fort is free", Terrain{Biome: "fort"}, 0},
		{"any lit biome is free", Terrain{Biome: "glowmarsh", Lit: true}, 0},
		{"indoor tag is free", Terrain{Biome: "forest", Tags: []string{"Indoor"}}, 0},
		{"road", Terrain{Biome: "road"}, 25},
		{"land", Terrain{Biome: "land"}, 35},
		{"farmland", Terrain{Biome: "farmland"}, 35},
		{"shore", Terrain{Biome: "shore"}, 35},
		{"forest", Terrain{Biome: "Forest"}, 50},
		{"cave", Terrain{Biome: "cave"}, 50},
		{"dungeon", Terrain{Biome: "dungeon"}, 50},
		{"water", Terrain{Biome: "water"}, 60},
		{"swamp", Terrain{Biome: "swamp"}, 80},
		{"desert", Terrain{Biome: "desert"}, 80},
		{"snow", Terrain{Biome: "snow"}, 90},
		{"mountains", Terrain{Biome: "mountains"}, 100},
		{"cliffs", Terrain{Biome: "cliffs"}, 100},
		{"unknown biome uses default", Terrain{Biome: "spiderweb"}, 40},
		{"strain tag overrides the biome", Terrain{Biome: "forest", Tags: []string{"strain:70"}}, 70},
		{"strain tag overrides a settlement", Terrain{Biome: "city", Tags: []string{"strain:5"}}, 5},
		{"strain tag of zero makes it free", Terrain{Biome: "snow", Tags: []string{"STRAIN: 0"}}, 0},
		{"malformed strain tag is ignored", Terrain{Biome: "forest", Tags: []string{"strain:lots"}}, 50},
		{"negative strain tag is ignored", Terrain{Biome: "forest", Tags: []string{"strain:-5"}}, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, TerrainCost(tc.in, table))
		})
	}
}

func TestStepCost(t *testing.T) {
	cases := []struct {
		name    string
		terrain int
		f       Factors
		want    int
	}{
		{"zero factors are neutral", 50, Factors{}, 50},
		{"settlement stays free", 0, Factors{LoadPct: 400, WeatherPct: 300}, 0},
		{"load alone", 50, Factors{LoadPct: 130}, 65},
		{"weather alone", 50, Factors{WeatherPct: 130}, 65},
		{"cold alone", 50, Factors{ColdPct: 150}, 75},
		{"mount alone", 100, Factors{MountPct: 75}, 75},
		{"well rested alone", 50, Factors{WellRestedPct: 50}, 25},
		{"combined", 100, Factors{LoadPct: 130, WeatherPct: 115, ColdPct: 125, MountPct: 75, WellRestedPct: 50}, 70},
		{"product clamped to 400%", 50, Factors{LoadPct: 300, WeatherPct: 300}, 200},
		{"product clamped to 25%", 100, Factors{MountPct: 25, WellRestedPct: 25}, 25},
		{"rounds half up", 1, Factors{WellRestedPct: 50}, 1},
		{"rounds down below half", 1, Factors{WellRestedPct: 49, LoadPct: 100}, 0},
		// The design's worked examples.
		{"forest clear unladen", 50, Factors{}, 50},
		{"forest rain 90% loaded", 50, Factors{WeatherPct: 115, LoadPct: 115}, 66},
		{"snow frostbitten fully loaded", 90, Factors{ColdPct: 150, LoadPct: 130}, 176},
		{"same riding the pack-horse, well rested", 90, Factors{ColdPct: 150, LoadPct: 130, MountPct: 75, WellRestedPct: 50}, 66},
		{"same, well rested, walking beside the horse", 90, Factors{ColdPct: 150, LoadPct: 130, WellRestedPct: 50}, 88},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, StepCost(tc.terrain, tc.f))
		})
	}
}

func TestCombinedPct(t *testing.T) {
	assert.Equal(t, 100, CombinedPct(Factors{}))
	assert.Equal(t, 132, CombinedPct(Factors{WeatherPct: 115, LoadPct: 115}))
	assert.Equal(t, PctMax, CombinedPct(Factors{LoadPct: 300, WeatherPct: 300}))
	assert.Equal(t, PctMin, CombinedPct(Factors{MountPct: 25, WellRestedPct: 25}))
}

func TestAccrue(t *testing.T) {
	cases := []struct {
		carry, cost, wantCarry, wantDrain int
	}{
		{0, 0, 0, 0},
		{0, 50, 50, 0},
		{50, 50, 0, 1},
		{90, 25, 15, 1},
		{99, 176, 75, 2},
		{0, 450, 50, 4},
		{-5, 10, 10, 0}, // corrupt carry is treated as 0
		{150, 0, 50, 1}, // an out-of-range carry is normalised
		{20, -10, 20, 0},
	}
	for _, tc := range cases {
		carry, drain := Accrue(tc.carry, tc.cost)
		assert.Equal(t, tc.wantCarry, carry, "carry %d cost %d", tc.carry, tc.cost)
		assert.Equal(t, tc.wantDrain, drain, "carry %d cost %d", tc.carry, tc.cost)
		assert.GreaterOrEqual(t, carry, 0)
		assert.Less(t, carry, CentiPerPoint)
	}
}

func TestColdPct(t *testing.T) {
	s := DefaultColdSettings()
	cases := []struct {
		exposure int
		want     int
	}{
		{0, 100},
		{-24, 100},
		{-25, 125},  // chilled
		{-50, 150},  // frostbitten
		{-75, 200},  // hypothermic
		{-100, 200}, // freezing
		{25, 100},   // heat bands never add walking strain
		{100, 100},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ColdPct(tc.exposure, s), "exposure %d", tc.exposure)
	}
}

type recordingStepper struct {
	calls [][3]int
}

func (r *recordingStepper) Stepped(userID, fromRoomID, toRoomID int) {
	r.calls = append(r.calls, [3]int{userID, fromRoomID, toRoomID})
}

func TestSteppedProvider(t *testing.T) {
	SetStepProvider(nil)
	Stepped(1, 2, 3) // no provider: a no-op, no panic

	rec := &recordingStepper{}
	SetStepProvider(rec)
	t.Cleanup(func() { SetStepProvider(nil) })
	Stepped(7, 2001, 2003)
	assert.Equal(t, [][3]int{{7, 2001, 2003}}, rec.calls)
}
