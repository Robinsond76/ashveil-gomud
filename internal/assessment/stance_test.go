package assessment

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/stance"
	"github.com/stretchr/testify/assert"
)

type storedStances map[string]stance.Stance

func (s storedStances) StoredStance(_ int, key string) stance.Stance { return s[key] }

// Phase 69 review: before a battle a stance is only in the store, so the
// estimate reads a copy carrying it and leaves the live member alone.
func TestTheEstimateReadsAStanceBeforeABattle(t *testing.T) {
	stance.SetProvider(storedStances{"companion:3": stance.Quick})
	t.Cleanup(func() { stance.SetProvider(nil) })

	live := &characters.Character{Name: "Wren"}
	got := withStance(live, 1, company.CompanionMemberKey(3))
	if assert.NotSame(t, live, got) && assert.NotNil(t, got.RT) {
		assert.Equal(t, stance.Quick, got.RT.Stance)
	}
	assert.Nil(t, live.RT, "the live member is untouched")

	plain := &characters.Character{Name: "Dain"}
	assert.Same(t, plain, withStance(plain, 1, company.CompanionMemberKey(4)), "no stance, no copy")
}
