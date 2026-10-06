package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// TestAnEasyFoesCritWoundsOnlyBelowHalf: Phase 35b. Through
// AttackMobVsPlayer, a forced crit from a foe 3+ levels below the player
// leaves no wound while the player stays at half health or more, and a
// light wound of a quarter of the blow below that. An even foe's crit
// still leaves its lasting wound.
func TestAnEasyFoesCritWoundsOnlyBelowHalf(t *testing.T) {
	foe := woundFight(t)
	foe.Character.Level = 2
	user := users.NewUserRecord(90330, 90330)
	*user.Character = *edgeFighter(90231)
	user.Character.Level = 5

	AttackMobVsPlayer(foe, user)
	if len(user.Character.Wounds) != 0 {
		t.Fatalf("a healthy player takes no wound from an easy foe: %+v", user.Character.Wounds)
	}

	user.Character.Health = user.Character.HealthMax.Value / 2
	result := AttackMobVsPlayer(foe, user)
	if len(user.Character.Wounds) != 1 {
		t.Fatalf("wounds %+v, want one below half", user.Character.Wounds)
	}
	if w := user.Character.Wounds[0]; !w.Light || w.Points != max(1, (result.DamageToTarget+3)/4) {
		t.Fatalf("wound %+v for %d damage, want light and a quarter", w, result.DamageToTarget)
	}

	foe.Character.Level = 3 // within 2 levels: no longer easy
	user.Character.Wounds = nil
	AttackMobVsPlayer(foe, user)
	if len(user.Character.Wounds) != 1 || user.Character.Wounds[0].Light {
		t.Fatalf("an even foe's crit leaves a lasting wound: %+v", user.Character.Wounds)
	}
}
