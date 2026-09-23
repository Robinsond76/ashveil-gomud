// Package climate is the GoMud-free, pure temperature and exposure model
// (Phase 15): air temperature from biome, night, weather, shelter, and heat
// sources; warmth from worn clothing; the comfort band and stress; and the
// signed exposure meter's per-tick step and penalty bands. Nothing here is
// stored or scheduled, and nothing advances the world clock.
package climate

import (
	"math"
	"sync"
)

// BiomeTemperature is a biome's base daytime temperature and how much
// colder it gets at night, in °C.
type BiomeTemperature struct {
	Base      int
	NightDrop int
}

// TemperatureInputs are what an observer's air temperature depends on.
type TemperatureInputs struct {
	Biome BiomeTemperature
	// Indoor is under a roof. Furnished marks a lit or indoor-tagged
	// interior, which is kept at the indoor temperature.
	Indoor     bool
	Furnished  bool
	Night      bool
	WeatherMod int
	HeatSource bool
}

// TemperatureSettings are the configured temperature constants.
type TemperatureSettings struct {
	IndoorTemperature int
	FireWarmth        int
}

// AirTemperature is the air temperature in °C.
func AirTemperature(in TemperatureInputs, s TemperatureSettings) int {
	var temp int
	switch {
	case in.Indoor && in.Furnished:
		temp = s.IndoorTemperature
	case in.Indoor:
		// Caves and dungeons hold a steady temperature.
		temp = in.Biome.Base
	default:
		temp = in.Biome.Base + in.WeatherMod
		if in.Night {
			temp -= in.Biome.NightDrop
		}
	}
	if in.HeatSource {
		temp += s.FireWarmth
	}
	return temp
}

// TemperatureName is a player-facing word for a temperature.
func TemperatureName(temp int) string {
	switch {
	case temp < -5:
		return "freezing"
	case temp < 5:
		return "cold"
	case temp < 12:
		return "cool"
	case temp < 22:
		return "mild"
	case temp < 30:
		return "warm"
	case temp < 38:
		return "hot"
	}
	return "scorching"
}

// WornPiece is one worn item: its equipment slot and its explicit warmth.
// 0 uses the slot's default; a negative value means no warmth at all (for
// jewellery and the like, since YAML can't tell an explicit 0 from unset).
type WornPiece struct {
	Slot   string
	Warmth int
}

// WarmthSettings are the per-slot default warmth values and the bonus for
// the upstream `warmed` buff flag.
type WarmthSettings struct {
	SlotDefaults map[string]int
	WarmedBonus  int
}

// Warmth is the total insulation of what a character wears.
func Warmth(worn []WornPiece, warmed bool, s WarmthSettings) int {
	total := 0
	for _, piece := range worn {
		switch {
		case piece.Warmth > 0:
			total += piece.Warmth
		case piece.Warmth == 0:
			total += s.SlotDefaults[piece.Slot]
		}
	}
	if warmed {
		total += s.WarmedBonus
	}
	return total
}

// ComfortSettings define the comfortable air-temperature band for someone
// wearing nothing, and how much warmth shifts its hot edge.
type ComfortSettings struct {
	ComfortLow  int
	ComfortHigh int
	HeatFactor  float64
}

// ComfortRange is the comfortable air-temperature range for warmth.
func ComfortRange(warmth int, s ComfortSettings) (low, high int) {
	return s.ComfortLow - warmth, s.ComfortHigh - int(math.Round(float64(warmth)*s.HeatFactor))
}

// Stress is how far air lies outside the comfort range for warmth:
// negative for cold, positive for heat, 0 when comfortable.
func Stress(air, warmth int, s ComfortSettings) int {
	low, high := ComfortRange(warmth, s)
	switch {
	case air < low:
		return air - low
	case air > high:
		return air - high
	}
	return 0
}

// ExposureMax is the magnitude of lethal exposure.
const ExposureMax = 100

// ExposureSettings control how exposure moves each tick.
type ExposureSettings struct {
	CeilingPerStress int
	Recovery         int
	ShelterRecovery  int
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func clampExposure(n int) int {
	if n > ExposureMax {
		return ExposureMax
	}
	if n < -ExposureMax {
		return -ExposureMax
	}
	return n
}

// StepExposure moves a signed exposure value (negative cold, positive heat)
// one tick toward the ceiling set by stress. It grows by max(1, |stress|/2)
// while moving away from zero, and recovers by the (sheltered) recovery rate
// otherwise; recovery never crosses zero in a single step, so switching from
// cold to heat first returns to 0. Only |stress| * CeilingPerStress >= 100
// can reach lethal exposure.
func StepExposure(current, stress int, sheltered bool, s ExposureSettings) int {
	target := clampExposure(stress * s.CeilingPerStress)
	if current == target {
		return current
	}
	growing := (current >= 0 && target > current) || (current <= 0 && target < current)
	if growing {
		step := abs(stress) / 2
		if step < 1 {
			step = 1
		}
		if target > current {
			return min(current+step, target)
		}
		return max(current-step, target)
	}
	step := s.Recovery
	if sheltered {
		step = s.ShelterRecovery
	}
	if step < 1 {
		step = 1
	}
	var next int
	if target > current {
		next = min(current+step, target)
		if current < 0 && next > 0 {
			next = 0
		}
	} else {
		next = max(current-step, target)
		if current > 0 && next < 0 {
			next = 0
		}
	}
	return next
}

// Band is an exposure penalty band.
type Band int

const (
	BandNone Band = iota
	BandMild
	BandModerate
	BandSevere
	BandCritical
)

// BandFor is the penalty band for an exposure value of either sign.
func BandFor(exposure int) Band {
	switch e := abs(exposure); {
	case e >= ExposureMax:
		return BandCritical
	case e >= 75:
		return BandSevere
	case e >= 50:
		return BandModerate
	case e >= 25:
		return BandMild
	}
	return BandNone
}

var (
	heatMu      sync.RWMutex
	heatSources []func(roomId int) bool
)

// RegisterHeatSource adds a provider reporting rooms warmed by a heat source
// it owns (e.g. modules/camping's lit campfires).
func RegisterHeatSource(provider func(roomId int) bool) {
	heatMu.Lock()
	defer heatMu.Unlock()
	heatSources = append(heatSources, provider)
}

// ResetHeatSources clears every heat-source provider (tests).
func ResetHeatSources() {
	heatMu.Lock()
	defer heatMu.Unlock()
	heatSources = nil
}

// RoomHasHeatSource reports whether any provider warms roomId.
func RoomHasHeatSource(roomId int) bool {
	heatMu.RLock()
	providers := append([]func(int) bool(nil), heatSources...)
	heatMu.RUnlock()
	for _, p := range providers {
		if p(roomId) {
			return true
		}
	}
	return false
}

// Provider reports the air temperature in a room. modules/exposure
// implements it so other modules (the weather command) can show it without
// importing that module.
type Provider interface {
	AirTemperatureIn(roomId int) (int, bool)
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// SetProvider registers the active temperature provider. nil clears it.
func SetProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// AirTemperatureIn asks the registered provider for a room's temperature.
func AirTemperatureIn(roomId int) (int, bool) {
	providerMu.RLock()
	p := provider
	providerMu.RUnlock()
	if p == nil {
		return 0, false
	}
	return p.AirTemperatureIn(roomId)
}
