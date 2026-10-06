package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Tempo is the actor's own turns per combat round; target and formation
// never enter the calculation. ValueAdj includes training and live modifiers.
func Tempo(c *characters.Character) float64 {
	cfg := configs.GetCombatConfig()
	speed := 1 + (float64(c.Stats.Speed.ValueAdj)-float64(cfg.TempoSpeedRef))/float64(cfg.TempoSpeedSpan)
	// Phase 35a2: armor bulk costs turns on top of burden; Strength never
	// offsets it.
	rate := (speed + 0.1*float64(c.StatMod("attacks"))) * (1 - 0.35*c.Burden()) * c.BulkTempoFactor()
	return math.Max(float64(cfg.TempoMin), math.Min(float64(cfg.TempoMax), rate))
}

// Meter is ephemeral battle state. Its zero value grants one opening turn,
// then carries fractional progress only, even when earned turns are forfeited.
type Meter struct {
	Points float64
	// Bonus is extra meter a character starts the battle with (Phase 39b:
	// a Samurai's Iaijutsu), read once when the meter first fills.
	Bonus   float64
	started bool
}

func (m *Meter) Fill(tempo float64, cap int) int {
	if !m.started {
		m.Points = 100 - 100*tempo + m.Bonus
		m.started = true
	}
	m.Points += 100 * tempo
	turns := int(math.Floor((m.Points + 1e-9) / 100))
	turns = max(0, min(cap, turns))
	m.Points = math.Max(0, math.Min(99, m.Points-100*float64(turns)))
	return turns
}
