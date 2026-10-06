package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/death"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommittedSharedAssetsRefreshCharInventoryThroughListeners(t *testing.T) {
	u := inventoryUser(t)
	u.Character.CompanyCargo = true
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	users.SetTestUser(u)
	AcceptGMCPForTest(u.ConnectionId())
	var received *GMCPCharModule_Payload_Inventory
	freshEvents(t)
	id := events.RegisterListener(GMCPOut{}, func(e events.Event) events.ListenerReturn {
		out := e.(GMCPOut)
		if out.Module == "Char.Inventory" && out.UserId == u.UserId {
			received = out.Payload.(*GMCPCharModule_Payload_Inventory)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(GMCPOut{}, id) })
	u.Character.Items = nil
	events.AddToQueue(events.CompanyAssetsChanged{UserId: u.UserId})
	events.ProcessEvents()
	require.NotNil(t, received)
	assert.Empty(t, received.Backpack.Items, "shared consumption cannot leave an old item in Gear")
	assert.True(t, received.Backpack.Summary.Shared)
}

func TestSharedCompanyInventoryUsesExactCargoRefs(t *testing.T) {
	u := inventoryUser(t)
	u.Character.CompanyCargo = true
	u.Character.Gold = 41
	p := buildInventoryPayload(u, inventoryTestSources(nil))
	assert.True(t, p.Shared)
	assert.Equal(t, 41, p.Treasury)
	assert.Empty(t, p.Members[0].Carried)
	require.Len(t, p.Cargo, len(u.Character.Items))
	for i, itm := range u.Character.Items {
		assert.Equal(t, itm.ShorthandId(), p.Cargo[i].Ref)
	}
}

func TestSharedInventoryContainersTrackAssignedCarrierAvailability(t *testing.T) {
	u := inventoryUser(t)
	u.Character.CompanyCargo = true
	u.Character.Health = 20
	testItemSpecs(t, items.ItemSpec{ItemId: 989203, Name: "cloth knapsack", Type: items.Pack, Subtype: items.Wearable, Weight: 400, CarryBonus: 10000})
	u.Character.Equipment.Pack = items.New(989203)
	// A spare adds cargo weight, never an assigned container.
	spare := items.New(989203)
	u.Character.Items = append(u.Character.Items, spare)
	src := inventoryTestSources(nil)
	src.companions = func(int) ([]company.InventoryMember, bool) {
		state := company.MemberState{}
		state.Equipment.Pack = items.New(989203)
		present := company.InventoryMemberOf(company.CompanionMemberKey(1), "Present", state)
		present.Available = true
		fallen := company.InventoryMemberOf(company.CompanionMemberKey(2), "Fallen", state)
		fallen.Fallen = true
		return []company.InventoryMember{present, fallen}, true
	}
	p := buildInventoryPayload(u, src)
	require.Len(t, p.Containers, 3)
	assert.True(t, p.Containers[0].Available)
	assert.Equal(t, u.Character.Equipment.Pack.ShorthandId(), p.Containers[0].Ref)
	assert.True(t, p.Containers[1].Available)
	assert.False(t, p.Containers[2].Available)
	for _, c := range p.Containers {
		assert.Equal(t, 10000, c.CapacityG)
		assert.NotEqual(t, spare.ShorthandId(), c.Ref)
	}
	require.Len(t, p.Cargo, 2)
	assert.Equal(t, spare.ShorthandId(), p.Cargo[1].Ref)
	u.Character.Health = 0
	assert.False(t, buildInventoryPayload(u, src).Containers[0].Available)
}

// Phase 48: the payload lists every gear slot, so the client shows a member's
// empty ones and can offer cargo for them.
func TestInventoryPayloadListsEveryGearSlot(t *testing.T) {
	u := inventoryUser(t)
	u.Character.CompanyCargo = true
	p := buildInventoryPayload(u, inventoryTestSources(nil))
	require.Len(t, p.Slots, 11)
	assert.Equal(t, inventorySlot{Slot: "weapon", Label: "Weapon"}, p.Slots[0])
	assert.Equal(t, "pack", p.Slots[len(p.Slots)-1].Slot)
}

// TestInventoryPayloadReportsHeldGoods (Phase 53): a captured leader's
// pack and gold show as a held notice with the capture room's title, and
// nothing is reported when nothing is held.
func TestInventoryPayloadReportsHeldGoods(t *testing.T) {
	c := characters.New()
	assert.Nil(t, seizedOf(c), "nothing held")
	c.Seized = []items.Item{{ItemId: 1}, {ItemId: 2}}
	c.SetMiscData(death.SeizedGoldKey, int64(150)) // YAML may bring it back as another integer type
	held := seizedOf(c)
	require.NotNil(t, held)
	assert.Equal(t, 2, held.Items)
	assert.Equal(t, 150, held.Gold)
	assert.False(t, held.Here)
	c.Seized = nil
	c.SetMiscData(death.SeizedGoldKey, nil)
	assert.Nil(t, seizedOf(c))
}
