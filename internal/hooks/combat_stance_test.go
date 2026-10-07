package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/stretchr/testify/assert"
)

type fixedStance struct{ st stance.Stance }

func (f *fixedStance) StoredStance(int, string) stance.Stance { return f.st }

// Phase 69 review: a battle reads the stance once; a write to the store
// mid-battle (a test-area restore, a prune) changes nothing until the next.
func TestAStanceIsReadOnceABattle(t *testing.T) {
	store := &fixedStance{st: stance.Heavy}
	stance.SetProvider(store)
	t.Cleanup(func() { stance.SetProvider(nil) })

	c := &characters.Character{Name: "Aria"}
	who := caster{userId: 987654}
	applyStance(who, c)
	assert.Equal(t, stance.Heavy, c.RT.Stance)

	store.st = stance.Keen
	applyStance(who, c)
	assert.Equal(t, stance.Heavy, c.RT.Stance, "kept for the rest of the battle")

	c.EndFightRT()
	applyStance(who, c)
	assert.Equal(t, stance.Keen, c.RT.Stance, "the next battle reads it afresh")
}
