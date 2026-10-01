package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
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
