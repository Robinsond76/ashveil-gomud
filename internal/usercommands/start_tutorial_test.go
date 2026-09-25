package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/tutorial"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTutorial struct {
	placed bool
	calls  []int
}

func (f *fakeTutorial) Begin(userID int) bool {
	f.calls = append(f.calls, userID)
	return f.placed
}

// startToTutorial drives `start` for a user whose race and name are set
// (and no archetype provider) to the tutorial question, answers "no", and
// returns the text sent.
func startToTutorial(t *testing.T, id int) (*users.UserRecord, string) {
	t.Helper()
	u := users.NewUserRecord(id, 1)
	u.Username = "acct" + str(id)
	u.Character.Name = "Aria"
	u.Character.RaceId = 1
	u.Character.RoomId = -1
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(id) })
	void := &rooms.Room{RoomId: -1, Title: "The Void"}

	events.ProcessEvents()
	var messages []string
	lid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == id {
			messages = append(messages, m.Text)
		}
		return events.Continue
	})
	defer events.UnregisterListener(events.Message{}, lid)

	_, err := Start("", u, void, 0)
	require.NoError(t, err)
	q := u.GetPrompt().GetNextQuestion()
	require.NotNil(t, q)
	require.Equal(t, `Would you like to skip the tutorial?`, q.Question)
	q.Answer("no")
	_, err = Start("", u, void, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	return u, strings.Join(messages, "\n")
}

// TestStartHandsTheTutorialToTheProvider: with a provider that places the
// player, `start` stops there (Phase 27a).
func TestStartHandsTheTutorialToTheProvider(t *testing.T) {
	f := &fakeTutorial{placed: true}
	tutorial.SetProvider(f)
	t.Cleanup(func() { tutorial.SetProvider(nil) })

	u, text := startToTutorial(t, 4101)
	assert.Equal(t, []int{4101}, f.calls)
	assert.Contains(t, text, "vortex")
	assert.NotContains(t, text, "fully occupied", "the legacy path doesn't run")
	assert.Nil(t, u.GetPrompt(), "creation is finished")
}

// TestStartFallsBackWithoutTheTutorial: without a provider, or when it
// can't place the player, the engine's own tutorial path runs (here with
// no tutorial rooms loaded, so it reports the zone full).
func TestStartFallsBackWithoutTheTutorial(t *testing.T) {
	tutorial.SetProvider(nil)
	_, text := startToTutorial(t, 4102)
	assert.Contains(t, text, "fully occupied")

	f := &fakeTutorial{placed: false}
	tutorial.SetProvider(f)
	t.Cleanup(func() { tutorial.SetProvider(nil) })
	_, text = startToTutorial(t, 4103)
	assert.Equal(t, []int{4103}, f.calls)
	assert.Contains(t, text, "fully occupied")
}
