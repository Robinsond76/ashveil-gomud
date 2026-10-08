package company

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSeededDiceDoNotOutliveTheirTest: a test that seeds the dice gets the
// seeded stream while it runs, and the tests after it get dice that are not
// the rest of that stream.
func TestSeededDiceDoNotOutliveTheirTest(t *testing.T) {
	const seed = 29
	drawn := 0
	t.Run("seeded", func(t *testing.T) {
		seedDice(t, seed)
		ref := rand.New(rand.NewSource(seed))
		for i := 0; i < 5; i++ {
			require.Equal(t, ref.Intn(1000), rand.Intn(1000), "draw %d is the seeded stream", i)
			drawn++
		}
	})
	rest := rand.New(rand.NewSource(seed))
	for i := 0; i < drawn; i++ {
		rest.Intn(1000)
	}
	same := 0
	for i := 0; i < 8; i++ {
		if rand.Intn(1000) == rest.Intn(1000) {
			same++
		}
	}
	assert.Less(t, same, 8, "the dice go on as the seeded stream after its test ended")
}
