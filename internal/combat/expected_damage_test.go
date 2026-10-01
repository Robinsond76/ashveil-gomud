//go:debug randseednop=0

package combat

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 33i1: the assessment's ExpectedDamage is the weapon rankings'
// estimate, and draws no random number: the gameplay dice stay where they
// were. (randseednop=0 lets rand.Seed reseed the global source here.)
func TestExpectedDamageDrawsNoDice(t *testing.T) {
	burdenSpecs(t)
	defenseOdds(t, 0, 0, 50)
	attacker, target := armed(edgeSwordID), heavy(armed(0))
	assert.Equal(t, expectedDPS(*attacker, *target), ExpectedDamage(attacker, target))
	assert.Positive(t, ExpectedDamage(attacker, target))
	assert.Zero(t, ExpectedDamage(nil, target))

	rand.Seed(33)
	want := rand.Int63()
	rand.Seed(33)
	for i := 0; i < 10; i++ {
		ExpectedDamage(attacker, target)
	}
	assert.Equal(t, want, rand.Int63(), "no draw from the gameplay RNG")
}
