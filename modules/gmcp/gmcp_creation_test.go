package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type creationSent struct {
	user int
	body map[string]any
}

func captureCreation(t *testing.T) *[]creationSent {
	t.Helper()
	var got []creationSent
	prev := creationSend
	creationSend = func(userID int, payload []byte) {
		var m map[string]any
		require.NoError(t, json.Unmarshal(payload, &m))
		got = append(got, creationSent{userID, m})
	}
	t.Cleanup(func() { creationSend = prev })
	return &got
}

// Phase 72a: Char.Creation carries each step to the player it is for, an
// {"active":false} when the steps end, and the current step on request.
func TestCharCreationPayload(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	u := users.NewUserRecord(8101, 810)
	u.Character.Name = "Wren"
	users.SetTestUser(u)
	AcceptGMCPForTest(u.ConnectionId())
	t.Cleanup(func() { creation.Forget(u.UserId) })
	got := captureCreation(t)

	creation.Publish(u.UserId, creation.View{
		Mode: "new", Step: "looks", Key: "skin", Kind: creation.KindChoice,
		Title: "What is your skin like?", Number: 5, Total: 12,
		Options: []creation.Option{{ID: "pale", Name: "pale", Color: "#ecd0b4"}, {ID: "dark", Name: "dark"}},
		Lineage: "warrior", Skin: "#ecd0b4", Hair: "#2a1a10", CanBack: true,
	})
	require.Len(t, *got, 1)
	sent := (*got)[0]
	assert.Equal(t, u.UserId, sent.user)
	assert.Equal(t, true, sent.body["active"])
	assert.Equal(t, "skin", sent.body["key"])
	assert.Equal(t, float64(5), sent.body["number"])
	assert.Equal(t, "#ecd0b4", sent.body["skin"])
	assert.Equal(t, "warrior", sent.body["lineage"])
	opts, _ := sent.body["options"].([]any)
	require.Len(t, opts, 2)
	assert.Equal(t, "#ecd0b4", opts[0].(map[string]any)["color"])

	// A request resends the step the player is on.
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	events.AddToQueue(GMCPCreationRequest{UserId: u.UserId})
	events.ProcessEvents()
	require.Len(t, *got, 2)
	assert.Equal(t, "skin", (*got)[1].body["key"])

	// The steps end: one inactive message, and a request then also says so.
	creation.Clear(u.UserId)
	require.Len(t, *got, 3)
	assert.Equal(t, false, (*got)[2].body["active"])
	events.AddToQueue(GMCPCreationRequest{UserId: u.UserId})
	events.ProcessEvents()
	require.Len(t, *got, 4)
	assert.Equal(t, false, (*got)[3].body["active"])

	// Leaving forgets the player's step.
	creation.Publish(u.UserId, creation.View{Mode: "new", Key: "hair"})
	events.AddToQueue(events.PlayerDespawn{UserId: u.UserId})
	events.ProcessEvents()
	_, open := creation.Get(u.UserId)
	assert.False(t, open, "a despawned player has no open step")
}
