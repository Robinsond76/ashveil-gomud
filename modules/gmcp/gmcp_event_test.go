package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/storyevents"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// Phase 78: the web client's `Event` request reaches the story-event
// module's hook with the player's id, on the game loop.
func TestEventWebRequestReachesTheStoryEventHook(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(45, 4545)
	users.SetTestUser(u)
	var got []int
	storyevents.OnPushRequest.Register(func(id int) int { got = append(got, id); return id })
	freshEvents(t)
	assert.True(t, gmcpModule.HandleWebGMCP(u.ConnectionId(), []byte("!!GMCP(Event)")))
	events.ProcessEvents()
	assert.Equal(t, []int{45}, got)
}
