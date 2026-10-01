package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 33d review: a leader's party leave handed leadership to the first other
// member even when offline, stranding the online members, and announced
// the successor by raw player id.
func TestPartyLeavePrefersAnOnlineSuccessorByName(t *testing.T) {
	t.Cleanup(parties.UseMemoryForTest())
	leader := users.NewUserRecord(93401, 0)
	leader.Character.Name = "Leader"
	online := users.NewUserRecord(93403, 0)
	online.Character.Name = "Wren"
	other := users.NewUserRecord(93404, 0)
	other.Character.Name = "Tam"
	for _, u := range []*users.UserRecord{leader, online, other} {
		users.SetTestUser(u)
		t.Cleanup(func() { users.RemoveTestUser(u.UserId) })
	}
	p := parties.New(leader.UserId)
	require.NotNil(t, p)
	for _, id := range []int{93402, online.UserId, other.UserId} { // 93402 is offline
		p.InvitePlayer(id)
		require.True(t, p.AcceptInvite(id))
	}
	heard := map[int][]string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		m := e.(events.Message)
		heard[m.UserId] = append(heard[m.UserId], m.Text)
		return events.Cancel
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })

	_, err := Party("leave", leader, &rooms.Room{}, 0)
	require.NoError(t, err)
	events.ProcessEvents()

	assert.Equal(t, online.UserId, p.LeaderUserId)
	assert.False(t, p.IsMember(leader.UserId))
	assert.Contains(t, strings.Join(heard[online.UserId], "\n"), "You are now the leader of the party")
	out := strings.Join(heard[other.UserId], "\n")
	assert.Contains(t, out, "Wren")
	assert.NotContains(t, out, "#93")
}
