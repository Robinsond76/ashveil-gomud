package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRecoveredPartySpawnPublishesAuthorityAndOfflineMembers(t *testing.T) {
	t.Cleanup(parties.UseMemoryForTest())
	p := parties.New(94901)
	p.InvitePlayer(94902)
	require.True(t, p.AcceptInvite(94902))
	u := users.NewUserRecord(94901, 0)
	u.Character.Name = "Owner"
	u.Character.HealthMax.Value = 10
	u.Character.Health = 10
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	p.SetSupport(u.UserId, true)
	g := &GMCPPartyModule{}
	var refresh bool
	id := events.RegisterListener(events.PartyUpdated{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.PartyUpdated)
		if evt.Action == "recovered" {
			refresh = true
			assert.Equal(t, p.GetMembers(), evt.UserIds)
		}
		return events.Cancel
	}, events.First)
	t.Cleanup(func() { events.UnregisterListener(events.PartyUpdated{}, id) })
	assert.Equal(t, events.Continue, g.playerSpawnHandler(events.PlayerSpawn{UserId: u.UserId}))
	events.ProcessEvents()
	assert.True(t, refresh)
	raw, _ := g.GetPartyNode(p, "Party")
	payload := raw.(GMCPPartyModule_Payload)
	require.Len(t, payload.Members, 2)
	assert.Equal(t, 94901, payload.Members[0].OwnerUserId)
	assert.True(t, payload.Members[0].Online)
	assert.True(t, payload.Members[0].Support)
	assert.Equal(t, 94902, payload.Members[1].OwnerUserId)
	assert.False(t, payload.Members[1].Online)
}
