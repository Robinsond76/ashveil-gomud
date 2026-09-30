package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/status"
)

// Phase 30d1b: heavy force always breaks a chant.
func TestHeavyBlow(t *testing.T) {
	cases := []struct {
		name string
		r    combat.AttackResult
		want bool
	}{
		{"an ordinary hit", combat.AttackResult{Hit: true, DamageToTarget: 5}, false},
		{"a bleeding cut", combat.AttackResult{Hit: true, BuffTarget: []int{status.Bleeding}}, false},
		{"a critical hit", combat.AttackResult{Hit: true, Crit: true}, true},
		{"a stagger", combat.AttackResult{Hit: true, BuffTarget: []int{status.Staggered}}, true},
		{"a knockdown", combat.AttackResult{Hit: true, BuffTarget: []int{status.Bleeding, status.KnockedDown}}, true},
		{"a stun", combat.AttackResult{Hit: true, BuffTarget: []int{status.Stunned}}, true},
	}
	for _, c := range cases {
		if got := heavyBlow(c.r); got != c.want {
			t.Errorf("%s: heavyBlow = %v, want %v", c.name, got, c.want)
		}
	}
}
