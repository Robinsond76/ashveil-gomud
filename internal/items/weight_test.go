package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 28 review: an item carrying its own spec copy (taken from cargo,
// enchanted, renamed, or saved before weights existed) still weighs what
// its base data says.
func TestWeightReadsBaseData(t *testing.T) {
	SetTestItemSpec(&ItemSpec{ItemId: 988100, Name: "sword", Weight: 1500})
	t.Cleanup(func() { RemoveTestItemSpec(988100) })

	plain := Item{ItemId: 988100}
	assert.Equal(t, 1500, plain.Weight())
	frozen := Item{ItemId: 988100, Spec: &ItemSpec{ItemId: 988100, Name: "sword of old", Weight: 0}}
	assert.Equal(t, 1500, frozen.Weight(), "an old override's zero doesn't hide the weight")
	orphan := Item{ItemId: 988101, Spec: &ItemSpec{ItemId: 988101, Weight: 40}}
	assert.Equal(t, 40, orphan.Weight(), "with no base data, its own spec")
	assert.Zero(t, (&Item{ItemId: 988102}).Weight(), "nothing known")
}
