package climate

import "testing"

var (
	testTemps   = TemperatureSettings{IndoorTemperature: 18, FireWarmth: 10}
	testComfort = ComfortSettings{ComfortLow: 16, ComfortHigh: 30, HeatFactor: 0.5}
	testExpo    = ExposureSettings{CeilingPerStress: 4, Recovery: 5, ShelterRecovery: 12}
	forest      = BiomeTemperature{Base: 12, NightDrop: 8}
	snow        = BiomeTemperature{Base: -12, NightDrop: 10}
	desert      = BiomeTemperature{Base: 36, NightDrop: 22}
	cave        = BiomeTemperature{Base: 10}
)

func TestAirTemperature(t *testing.T) {
	cases := []struct {
		name string
		in   TemperatureInputs
		want int
	}{
		{"forest noon", TemperatureInputs{Biome: forest}, 12},
		{"forest night", TemperatureInputs{Biome: forest, Night: true}, 4},
		{"forest night in rain", TemperatureInputs{Biome: forest, Night: true, WeatherMod: -3}, 1},
		{"snowfield night", TemperatureInputs{Biome: snow, Night: true}, -22},
		{"desert noon", TemperatureInputs{Biome: desert}, 36},
		{"desert night", TemperatureInputs{Biome: desert, Night: true}, 14},
		{"furnished interior ignores weather and night", TemperatureInputs{Biome: snow, Indoor: true, Furnished: true, Night: true, WeatherMod: -6}, 18},
		{"unlit cave keeps biome base only", TemperatureInputs{Biome: cave, Indoor: true, Night: true, WeatherMod: -6}, 10},
		{"campfire warms a snowfield", TemperatureInputs{Biome: snow, Night: true, HeatSource: true}, -12},
	}
	for _, c := range cases {
		if got := AirTemperature(c.in, testTemps); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestWarmth(t *testing.T) {
	s := WarmthSettings{SlotDefaults: map[string]int{"body": 5, "legs": 3, "feet": 2}, WarmedBonus: 20}
	worn := []WornPiece{{Slot: "body"}, {Slot: "legs"}, {Slot: "feet", Warmth: 6}, {Slot: "ring"}}
	if got := Warmth(worn, false, s); got != 5+3+6 {
		t.Fatalf("slot defaults plus an explicit override: got %d", got)
	}
	if got := Warmth(worn, true, s); got != 5+3+6+20 {
		t.Fatalf("warmed flag adds its bonus: got %d", got)
	}
	locket := []WornPiece{{Slot: "neck", Warmth: -1}}
	if got := Warmth(locket, false, WarmthSettings{SlotDefaults: map[string]int{"neck": 1}}); got != 0 {
		t.Fatalf("a negative warmth means none, not the slot default: got %d", got)
	}
	if got := Warmth(nil, false, s); got != 0 {
		t.Fatalf("naked is 0, got %d", got)
	}
}

func TestComfortAndStress(t *testing.T) {
	cases := []struct {
		name        string
		air, warmth int
		want        int
	}{
		{"naked forest noon is mildly cold", 12, 0, -4},
		{"clothed forest night in rain is comfortable", 1, 15, 0},
		{"naked snowfield night is lethal cold", -22, 0, -38},
		{"heavy furs in desert is lethal heat", 36, 40, 26},
		{"clothed in desert is hot", 36, 15, 14},
		{"naked in desert is only warm", 36, 0, 6},
		{"clothed mild day is comfortable", 18, 15, 0},
	}
	for _, c := range cases {
		if got := Stress(c.air, c.warmth, testComfort); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
	low, high := ComfortRange(15, testComfort)
	if low != 1 || high != 22 {
		t.Fatalf("comfort range for warmth 15: got %d..%d", low, high)
	}
}

func TestExposureStepGrowsToCeiling(t *testing.T) {
	// Mild cold (stress -4) settles at a penalty-free level.
	e := 0
	for i := 0; i < 50; i++ {
		e = StepExposure(e, -4, false, testExpo)
	}
	if e != -16 {
		t.Fatalf("mild cold should settle at -16, got %d", e)
	}
	if BandFor(e) != BandNone {
		t.Fatalf("mild cold must not penalise, band %v", BandFor(e))
	}
}

func TestExposureLethalOnlyWhenExtreme(t *testing.T) {
	for stress := 1; stress <= 60; stress++ {
		e := 0
		for i := 0; i < 200; i++ {
			e = StepExposure(e, -stress, false, testExpo)
		}
		lethal := BandFor(e) == BandCritical
		if lethal != (stress >= 25) {
			t.Fatalf("stress %d: lethal=%v (exposure %d); only stress >= 25 may be lethal", stress, lethal, e)
		}
	}
}

func TestExposureStepRatesAndRecovery(t *testing.T) {
	if got := StepExposure(0, -38, false, testExpo); got != -19 {
		t.Fatalf("naked snowfield grows by stress/2: got %d", got)
	}
	if got := StepExposure(-10, -1, false, testExpo); got != -5 {
		t.Fatalf("recovery toward a smaller ceiling: got %d", got)
	}
	if got := StepExposure(-60, 0, false, testExpo); got != -55 {
		t.Fatalf("recovery in comfort: got %d", got)
	}
	if got := StepExposure(-60, 0, true, testExpo); got != -48 {
		t.Fatalf("sheltered recovery is faster: got %d", got)
	}
	if got := StepExposure(-3, 20, false, testExpo); got != 0 {
		t.Fatalf("switching from cold to heat stops at zero first: got %d", got)
	}
	if got := StepExposure(0, 20, false, testExpo); got != 10 {
		t.Fatalf("then heat grows from zero: got %d", got)
	}
	if got := StepExposure(95, 40, false, testExpo); got != 100 {
		t.Fatalf("growth clamps at 100: got %d", got)
	}
}

func TestBandFor(t *testing.T) {
	cases := map[int]Band{0: BandNone, 24: BandNone, -25: BandMild, 49: BandMild, 50: BandModerate, -74: BandModerate, 75: BandSevere, -99: BandSevere, 100: BandCritical, -100: BandCritical}
	for e, want := range cases {
		if got := BandFor(e); got != want {
			t.Errorf("BandFor(%d) = %v want %v", e, got, want)
		}
	}
}

func TestHeatSourceRegistry(t *testing.T) {
	t.Cleanup(ResetHeatSources)
	if RoomHasHeatSource(3) {
		t.Fatal("no providers, no heat")
	}
	RegisterHeatSource(func(roomId int) bool { return roomId == 3 })
	if !RoomHasHeatSource(3) || RoomHasHeatSource(4) {
		t.Fatal("provider should heat room 3 only")
	}
}

func TestTemperatureName(t *testing.T) {
	for _, c := range []struct {
		temp int
		want string
	}{{-20, "freezing"}, {0, "cold"}, {8, "cool"}, {15, "mild"}, {25, "warm"}, {33, "hot"}, {45, "scorching"}} {
		if got := TemperatureName(c.temp); got != c.want {
			t.Errorf("TemperatureName(%d) = %q want %q", c.temp, got, c.want)
		}
	}
}
