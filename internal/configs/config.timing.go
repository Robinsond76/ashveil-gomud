package configs

import (
	"math"
	"time"
)

type Timing struct {
	TurnMs            ConfigInt `yaml:"TurnMs"`
	RoundSeconds      ConfigInt `yaml:"RoundSeconds"`
	RoundsPerAutoSave ConfigInt `yaml:"RoundsPerAutoSave"`
	// CombatEveryRounds is how many game rounds make one combat round
	// (Ashveil Phase 29f). Only combat resolution runs at this cadence;
	// every other round-based system keeps RoundSeconds.
	CombatEveryRounds ConfigInt `yaml:"CombatEveryRounds"`

	// Protected values
	turnsPerRound   int     // calculated and cached when data is validated.
	turnsPerSave    int     // calculated and cached when data is validated.
	turnsPerSecond  int     // calculated and cached when data is validated.
	roundsPerMinute float64 // calculated and cached when data is validated.
}

func (e *Timing) Validate() {

	if e.TurnMs < 10 {
		e.TurnMs = 100 // default
	}

	if e.RoundSeconds < 1 {
		e.RoundSeconds = 4 // default
	}

	if e.RoundsPerAutoSave < 1 {
		e.RoundsPerAutoSave = 900 // default of 15 minutes worth of rounds
	}

	if e.CombatEveryRounds < 1 {
		e.CombatEveryRounds = 2 // default: an 8-second combat round
	}

	// Pre-calculate and cache useful values
	e.turnsPerRound = int((e.RoundSeconds * 1000) / e.TurnMs)
	e.turnsPerSave = int(e.RoundsPerAutoSave) * e.turnsPerRound
	e.turnsPerSecond = int(1000 / e.TurnMs)
	e.roundsPerMinute = 60 / float64(e.RoundSeconds)

}

func (e Timing) TurnsPerRound() int {
	return e.turnsPerRound
}

// CombatRoundDue reports whether combat resolves on this game round.
func (e Timing) CombatRoundDue(roundNumber uint64) bool {
	every := uint64(e.CombatEveryRounds)
	if every < 1 {
		every = 1
	}
	return roundNumber%every == 0
}

// CombatRoundDuration is the real time between two combat rounds.
func (e Timing) CombatRoundDuration() time.Duration {
	every := int(e.CombatEveryRounds)
	if every < 1 {
		every = 1
	}
	return time.Duration(every) * time.Duration(e.RoundSeconds) * time.Second
}

func (e Timing) TurnsPerAutoSave() int {
	return e.turnsPerSave
}

func (e Timing) TurnsPerSecond() int {
	return e.turnsPerSecond
}

func (e Timing) MinutesToRounds(minutes int) int {
	return int(math.Ceil(e.roundsPerMinute * float64(minutes)))
}

func (e Timing) SecondsToRounds(seconds int) int {
	return int(math.Ceil(float64(seconds) / float64(e.RoundSeconds)))
}

func (e Timing) MinutesToTurns(minutes int) int {
	return int(math.Ceil(float64(minutes*60*1000) / float64(e.TurnMs)))
}

func (e Timing) SecondsToTurns(seconds int) int {
	return int(math.Ceil(float64(seconds*1000) / float64(e.TurnMs)))
}

func (e Timing) RoundsToSeconds(rounds int) int {
	return int(math.Ceil(float64(rounds) * float64(e.RoundSeconds)))
}

func GetTimingConfig() Timing {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Timing
}
