package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestPartyLogoutConsentAndSuccessorFeedback(t *testing.T) {
	for _, withSuccessor := range []bool{false, true} {
		t.Run(map[bool]string{false: "solo disbands", true: "promotes member"}[withSuccessor], func(t *testing.T) {
			newHandOffWorld(t)
			conn, _ := connect(t)
			leader := saveUser(t, 93501, "PartyLeader")
			_, _, err := users.LoginUser(leader, conn)
			require.NoError(t, err)
			require.NoError(t, rooms.MoveToRoom(leader.UserId, 932001, true))
			p := parties.New(leader.UserId)
			require.NotNil(t, p)
			t.Cleanup(p.Disband)
			if withSuccessor {
				member := saveUser(t, 93502, "PartyMember")
				users.SetTestUser(member)
				p.InvitePlayer(member.UserId)
				p.AcceptInvite(member.UserId)
				p.SetFollow(member.UserId, true)
				p.SetSupport(leader.UserId, true)
				p.SetSupport(member.UserId, true)
			}
			var messages []events.Message
			mid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
				messages = append(messages, e.(events.Message))
				return events.Cancel
			})
			t.Cleanup(func() { events.UnregisterListener(events.Message{}, mid) })
			var updates []events.PartyUpdated
			pid := events.RegisterListener(events.PartyUpdated{}, func(e events.Event) events.ListenerReturn {
				updates = append(updates, e.(events.PartyUpdated))
				return events.Cancel
			})
			t.Cleanup(func() { events.UnregisterListener(events.PartyUpdated{}, pid) })
			events.AddToQueue(events.PlayerDespawn{UserId: leader.UserId, RoomId: 932001, HandOff: true})
			events.ProcessEvents()
			assert.Nil(t, parties.Get(leader.UserId))
			require.Len(t, updates, 1)
			assert.Contains(t, updates[0].UserIds, leader.UserId)
			if withSuccessor {
				assert.Equal(t, 93502, p.LeaderUserId)
				assert.Empty(t, p.Followers)
				assert.Empty(t, parties.AlliedLeaders(93502))
				found := false
				for _, m := range messages {
					if m.UserId == 93502 && strings.Contains(m.Text, "You are now the leader of the party. Following is off for everyone.") {
						found = true
					}
				}
				assert.True(t, found, "successor receives leadership and follow-reset notice")
			} else {
				for _, m := range messages {
					assert.NotContains(t, m.Text, "You are now the leader")
				}
			}
		})
	}
}
