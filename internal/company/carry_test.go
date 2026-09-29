package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func TestBestPackGrams(t *testing.T) {
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 988031, Name: "satchel", CarryBonus: 5000})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 988032, Name: "frame pack", CarryBonus: 15000})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: 988033, Name: "rope"})
	t.Cleanup(func() {
		for _, id := range []int{988031, 988032, 988033} {
			items.RemoveTestItemSpec(id)
		}
	})
	assert.Zero(t, BestPackGrams(nil))
	assert.Zero(t, BestPackGrams([]items.Item{{ItemId: 988033}}))
	assert.Equal(t, 15000, BestPackGrams([]items.Item{{ItemId: 988031}, {ItemId: 988032}, {ItemId: 988031}}), "one pack counts: the largest")
}

func TestCountedMembersWithoutProvider(t *testing.T) {
	SetFormationProvider(nil)
	assert.Equal(t, 1, CountedMembers(7), "a leader alone")
	assert.Nil(t, CompanionCarry(7))
}
