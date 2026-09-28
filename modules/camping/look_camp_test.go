package camping

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCampShowsInLook drives Phase 32a's camp line through the real look
// command in the shipped Fork at the Black Oak: the camp command pitches,
// lights, and breaks it; look reads the module's lock-free snapshot, and a
// second player sees whose camp it is.
func TestCampShowsInLook(t *testing.T) {
	loadShippedWorld(t)
	keywords.LoadAliases() // look's map legend
	fork := rooms.LoadRoom(2002)
	require.NotNil(t, fork)
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, func() time.Time { return baseTime() })
	camping.SetRoomCampsReader(module.RoomCamps)
	t.Cleanup(func() { camping.SetRoomCampsReader(nil) })

	dain := campUser(t, 7, fork.RoomId)
	dain.Character.Name = "Dain"
	mira := users.NewUserRecord(8, 2)
	mira.Character.Name = "Mira"
	mira.Character.RoomId = fork.RoomId
	users.SetTestUser(mira)
	fork.AddPlayer(dain.UserId)
	fork.AddPlayer(mira.UserId)
	t.Cleanup(func() {
		fork.RemovePlayer(dain.UserId)
		fork.RemovePlayer(mira.UserId)
	})

	tags := regexp.MustCompile(`<[^>]*>`)
	var seen []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		seen = append(seen, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	look := func(u *users.UserRecord) string {
		t.Helper()
		events.ProcessEvents()
		seen = nil
		_, err := usercommands.Look("", u, fork, events.CmdSecretly)
		require.NoError(t, err)
		events.ProcessEvents()
		return tags.ReplaceAllString(strings.Join(seen, "\n"), "")
	}
	camp := func(rest string) {
		t.Helper()
		_, err := module.userCommand(rest, dain, fork, 0)
		require.NoError(t, err)
	}

	assert.NotContains(t, look(dain), "camp is pitched here", "no camp yet")
	camp("")
	assert.Contains(t, look(dain), "A camp is pitched here, around a cold fire pit.")
	assert.Contains(t, look(mira), "Dain's camp is pitched here, around a cold fire pit.")
	camp("fire")
	assert.Contains(t, look(dain), "A camp is pitched here: bedrolls around a crackling campfire.")
	assert.Contains(t, look(mira), "Dain's camp is pitched here: bedrolls around a crackling campfire.")
	assert.True(t, module.RoomHasLitFire(fork.RoomId), "the lit-fire query reads the same snapshot")
	camp("break")
	assert.NotContains(t, look(dain), "camp is pitched here", "struck")
	assert.NotContains(t, look(mira), "camp is pitched here")
	assert.False(t, module.RoomHasLitFire(fork.RoomId))
}
