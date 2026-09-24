// Package walking is the GoMud-free, pure Phase 16 walking-strain model: the
// terrain cost of stepping into a room, the multipliers that scale it (load,
// weather, cold, mount, Well Rested), and the per-member carry that turns
// fractional strain into whole fatigue points. Nothing here is stored,
// scheduled, or clocked, and nothing advances the world clock.
package walking

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/climate"
)

// Strain is measured in centi-fatigue: CentiPerPoint is one fatigue point.
const CentiPerPoint = 100

// PctMin and PctMax bound the combined multiplier product. 100 is neutral.
const (
	PctMin = 25
	PctMax = 400
)

// StrainTagPrefix marks a room tag that overrides its terrain cost, e.g.
// "strain:70".
const StrainTagPrefix = "strain:"

// TerrainTable is the configured per-biome step cost in centi-fatigue.
// Settlement biomes, any lit biome, and any room tagged indoor cost 0.
type TerrainTable struct {
	Settlements map[string]bool
	Biomes      map[string]int
	Default     int
}

// DefaultTerrain mirrors modules/walking's shipped config.
func DefaultTerrain() TerrainTable {
	return TerrainTable{
		Settlements: map[string]bool{"city": true, "slums": true, "house": true, "fort": true},
		Biomes: map[string]int{
			"road": 25, "land": 35, "farmland": 35, "shore": 35,
			"forest": 50, "cave": 50, "dungeon": 50, "water": 60,
			"swamp": 80, "desert": 80, "snow": 90, "mountains": 100, "cliffs": 100,
		},
		Default: 40,
	}
}

// Terrain describes the room being walked into.
type Terrain struct {
	Biome string
	// Lit is the biome's lit-area flag: lit biomes are settled ground.
	Lit  bool
	Tags []string
}

// strainOverride reads a valid strain:<n> tag.
func strainOverride(tags []string) (int, bool) {
	for _, tag := range tags {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if !strings.HasPrefix(lower, StrainTagPrefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(lower, StrainTagPrefix)))
		if err != nil || n < 0 {
			continue
		}
		return n, true
	}
	return 0, false
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), want) {
			return true
		}
	}
	return false
}

// TerrainCost is the centi-fatigue cost of stepping into a room, before
// multipliers. A strain:<n> tag wins; then settlements are free; then the
// biome table, falling back to the default.
func TerrainCost(in Terrain, table TerrainTable) int {
	if n, ok := strainOverride(in.Tags); ok {
		return n
	}
	biome := strings.ToLower(strings.TrimSpace(in.Biome))
	if in.Lit || table.Settlements[biome] || hasTag(in.Tags, "indoor") {
		return 0
	}
	if cost, ok := table.Biomes[biome]; ok {
		return cost
	}
	return table.Default
}

// Factors are the percentage multipliers on one member's step. 0 means 100
// (neutral), so a zero value is always safe.
type Factors struct {
	LoadPct    int
	WeatherPct int
	ColdPct    int
	MountPct   int
	// WellRestedPct is the member's rest tier: Well Rested or, since
	// Phase 23a, the weaker camp Rested. Only one tier ever applies.
	WellRestedPct int
}

func (f Factors) list() []int {
	return []int{f.LoadPct, f.WeatherPct, f.ColdPct, f.MountPct, f.WellRestedPct}
}

// product is the multiplier product scaled by 100^5 (100 per factor),
// clamped to [PctMin, PctMax] percent.
func (f Factors) product() int64 {
	const unit = int64(100 * 100 * 100 * 100) // 100^4: a percentage, scaled
	p := int64(1)
	for _, pct := range f.list() {
		if pct <= 0 {
			pct = 100
		}
		p *= int64(pct)
	}
	lo, hi := int64(PctMin)*unit, int64(PctMax)*unit
	if p < lo {
		p = lo
	}
	if p > hi {
		p = hi
	}
	return p
}

// scale is 100^5: a product of five percentages.
const scale = int64(100 * 100 * 100 * 100 * 100)

// CombinedPct is the clamped multiplier product as a whole percentage,
// rounded half up.
func CombinedPct(f Factors) int {
	return int((f.product()*100 + scale/2) / scale)
}

// StepCost is one member's centi-fatigue cost for one step: the terrain
// cost times the clamped multiplier product, rounded half up.
func StepCost(terrain int, f Factors) int {
	if terrain <= 0 {
		return 0
	}
	return int((int64(terrain)*f.product() + scale/2) / scale)
}

// Accrue adds cost to a member's carry and returns the new carry (always in
// 0..CentiPerPoint-1) and the whole fatigue points to drain.
func Accrue(carry, cost int) (newCarry, drain int) {
	if carry < 0 {
		carry = 0
	}
	if cost < 0 {
		cost = 0
	}
	total := carry + cost
	return total % CentiPerPoint, total / CentiPerPoint
}

// ColdSettings are the walking multipliers for Phase 15's cold bands.
type ColdSettings struct {
	ChilledPct     int
	FrostbittenPct int
	// HypothermicPct covers both the severe and critical bands.
	HypothermicPct int
}

// DefaultColdSettings mirrors modules/walking's shipped config.
func DefaultColdSettings() ColdSettings {
	return ColdSettings{ChilledPct: 125, FrostbittenPct: 150, HypothermicPct: 200}
}

// ColdPct is the walking multiplier for a signed exposure value. Only the
// cold side adds strain; heat already drains thirst.
func ColdPct(exposure int, s ColdSettings) int {
	if exposure >= 0 {
		return 100
	}
	switch climate.BandFor(exposure) {
	case climate.BandMild:
		return s.ChilledPct
	case climate.BandModerate:
		return s.FrostbittenPct
	case climate.BandSevere, climate.BandCritical:
		return s.HypothermicPct
	}
	return 100
}
