package company

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 38b review: two summons of a kind in one company keep apart.
func TestSummonKeysAreUniquePerInstance(t *testing.T) {
	t.Cleanup(ResetSummonsForTest)
	a := RegisterSummon(101, 7, "angel")
	b := RegisterSummon(102, 7, "angel")
	assert.NotEqual(t, a, b)
	assert.True(t, IsSummonKey(a))
	assert.True(t, IsSummonKey(b))
}
