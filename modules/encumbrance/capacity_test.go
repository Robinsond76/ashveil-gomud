package encumbrance

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mount"
	"github.com/GoMudEngine/GoMud/internal/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func packItem(bonus int) items.Item {
	return items.Item{ItemId: 988201, UUID: uuid.New(items.UUIDItem), Spec: &items.ItemSpec{ItemId: 988201, Name: "satchel", Weight: 600, CarryBonus: bonus}}
}

// Phase 32f: capacity is every member's share plus the horses'.
func TestCapacityComesFromMembersAndHorses(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Stats.Strength.ValueAdj = 4
	user.Character.Items = []items.Item{packItem(5000), packItem(3000)}
	module := newTestModule(&fakeStore{}, user)
	module.memberBaseGrams, module.strengthGrams = 20000, 500
	mount.SetProvider(nil)

	load, ok := module.CurrentLoad(7)
	require.True(t, ok)
	assert.Equal(t, 20000+2000+5000, load.CapacityGrams, "alone: the base, Strength, and the larger pack")
	assert.Equal(t, 1200, load.PersonalGrams, "a second pack is only weight")

	module.companionCarry = func(leader int) []company.MemberCarry {
		return []company.MemberCarry{{Strength: 10, PackGrams: 10000}, {}}
	}
	mount.SetProvider(fakeMountProvider{bonusGrams: 100000})
	t.Cleanup(func() { mount.SetProvider(nil) })
	load, _ = module.CurrentLoad(7)
	assert.Equal(t, 27000+(20000+5000+10000)+20000, load.MemberCapacityGrams)
	assert.Equal(t, 100000, load.MountCapacityGrams)
	assert.Equal(t, load.MemberCapacityGrams+100000, load.CapacityGrams)
	assert.Contains(t, module.status(7), "Capacity: members 82.0 kg, horses 100.0 kg.")
}

// Phase 32f: WouldExceed through the registered module.
func TestWouldExceedThroughModule(t *testing.T) {
	user := testUser(t, 7)
	user.Character.Items = []items.Item{testItem(rockId)} // 500g
	module := newTestModule(&fakeStore{}, user)
	module.memberBaseGrams = 1000
	encumbrance.SetProvider(module)
	t.Cleanup(func() { encumbrance.SetProvider(nil) })

	_, refuse := encumbrance.WouldExceed(7, 500)
	assert.False(t, refuse, "exactly full is allowed")
	load, refuse := encumbrance.WouldExceed(7, 501)
	assert.True(t, refuse)
	assert.Equal(t, 500, load.TotalGrams())
}
