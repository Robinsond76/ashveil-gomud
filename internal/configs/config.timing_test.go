package configs

import (
	"testing"
	"time"
)

func TestCombatEveryRoundsDefaultsAndCadence(t *testing.T) {
	for _, tc := range []struct {
		set, want ConfigInt
	}{{0, 2}, {-3, 2}, {1, 1}, {3, 3}} {
		timing := Timing{TurnMs: 50, RoundSeconds: 4, CombatEveryRounds: tc.set}
		timing.Validate()
		if timing.CombatEveryRounds != tc.want {
			t.Fatalf("CombatEveryRounds %d validated to %d, want %d", tc.set, timing.CombatEveryRounds, tc.want)
		}
		if got, want := timing.CombatRoundDuration(), time.Duration(tc.want)*4*time.Second; got != want {
			t.Fatalf("CombatRoundDuration = %v, want %v", got, want)
		}
	}

	every2 := Timing{TurnMs: 50, RoundSeconds: 4, CombatEveryRounds: 2}
	every2.Validate()
	var due []uint64
	for r := uint64(1); r <= 6; r++ {
		if every2.CombatRoundDue(r) {
			due = append(due, r)
		}
	}
	if len(due) != 3 || due[0] != 2 || due[1] != 4 || due[2] != 6 {
		t.Fatalf("every 2: due rounds %v, want [2 4 6]", due)
	}

	every1 := Timing{TurnMs: 50, RoundSeconds: 4, CombatEveryRounds: 1}
	every1.Validate()
	for r := uint64(1); r <= 4; r++ {
		if !every1.CombatRoundDue(r) {
			t.Fatalf("every 1: round %d not due", r)
		}
	}
}
