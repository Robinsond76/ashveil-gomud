package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestNarrationPoolsVoice (Phase 29c): every opener and closing is in the
// narration voice, and every keyed pool has a partner.
func TestNarrationPoolsVoice(t *testing.T) {
	for _, pools := range []map[string][]string{fightOpeners, fightClosings} {
		assert.NotEmpty(t, pools[""], "a generic pool")
		for key, pool := range pools {
			assert.NotEmpty(t, pool, key)
			for _, line := range pool {
				assert.Empty(t, util.NarrationVoiceProblem(line), line)
			}
		}
	}
	for key := range fightOpeners {
		assert.Contains(t, fightClosings, key, "a group with an opener has a closing")
	}
}

func TestNarrationPoolByGroup(t *testing.T) {
	assert.Equal(t, fightOpeners["bandits"], narrationPool(fightOpeners, []string{"bandits"}))
	assert.Equal(t, fightClosings["bandits"], narrationPool(fightClosings, []string{"wolves", "bandits"}), "the first group with a pool")
	assert.Equal(t, fightOpeners[""], narrationPool(fightOpeners, []string{"wolves"}), "an unknown group falls back")
	assert.Equal(t, fightOpeners[""], narrationPool(fightOpeners, nil))
	assert.Contains(t, fightClosings["practice-squad"], fightClosing([]string{"practice-squad"}))
	assert.Contains(t, fightOpeners[""], fightOpener([]string{""}))
}
