package mobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 27c: a practice foe waits for its player; boredom never despawns it.
func TestPracticeMobsNeverDespawn(t *testing.T) {
	assert.True(t, (&Mob{}).Despawns())
	assert.False(t, (&Mob{Practice: true}).Despawns())
}
