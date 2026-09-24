package archetype

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wiring (Phase 22a): these drive the real upstream `start` command through
// its prompt, with the real module registered as the archetypes provider.

const tutorialQuestion = `Would you like to skip the tutorial?`

// creating is a user in the void whose race and name are already set, so
// `start` begins at the archetype step.
func creating(t *testing.T, id int) *users.UserRecord {
	t.Helper()
	u := newUser(id)
	u.Username = "acct" + u.Character.Name
	u.Character.Name = "Aria"
	u.Character.RaceId = 1
	u.Character.RoomId = -1
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(id) })
	return u
}

var voidRoom = &rooms.Room{RoomId: -1, Title: "The Void"}

// start runs the start command once, returning the text it sent.
func start(t *testing.T, u *users.UserRecord) string {
	t.Helper()
	return captureText(t, func() {
		_, err := usercommands.Start("", u, voidRoom, 0)
		require.NoError(t, err)
	})
}

// answer answers the pending question, then re-enters start, as the input
// handler does.
func answer(t *testing.T, u *users.UserRecord, response string) string {
	t.Helper()
	q := u.GetPrompt().GetNextQuestion()
	require.NotNil(t, q, "a question is pending")
	q.Answer(response)
	return start(t, u)
}

func pending(u *users.UserRecord) string {
	if p := u.GetPrompt(); p != nil {
		if q := p.GetNextQuestion(); q != nil {
			return q.Question
		}
	}
	return ""
}

func TestWiringStartArchetypeStepGrantsEachKit(t *testing.T) {
	m := registered(t)
	for i, choice := range m.CreationChoices() {
		t.Run(choice.ID, func(t *testing.T) {
			u := creating(t, 3001+i)
			text := start(t, u)
			assert.Contains(t, text, "Starter kit:")
			assert.Contains(t, text, choice.Name)
			assert.Equal(t, `Which archetype will you follow?`, pending(u))

			// Answer by number for even entries, by name for odd ones.
			response := choice.Name
			if i%2 == 0 {
				response = []string{"1", "2", "3", "4", "5"}[i]
			}
			answer(t, u, response)
			assert.Contains(t, pending(u), "Become a "+choice.Name)

			text = answer(t, u, "yes")
			assert.Contains(t, text, "You are now a "+choice.Name)
			assert.Contains(t, text, "starter kit")
			id, chosen := archetypes.PlayerArchetype(u.UserId)
			assert.True(t, chosen)
			assert.Equal(t, choice.ID, id)
			assert.Equal(t, kitOf(t, m, choice.ID), backpackIDs(u))
			assert.Equal(t, tutorialQuestion, pending(u), "creation carries on to the tutorial")
		})
	}
}

func TestWiringStartReconnectResumes(t *testing.T) {
	m := registered(t)
	u := creating(t, 3011)
	start(t, u)
	answer(t, u, "warrior")

	// Disconnect before confirming: the prompt is gone, the step asks again.
	u.ClearPrompt()
	start(t, u)
	assert.Equal(t, `Which archetype will you follow?`, pending(u))
	_, chosen := m.PlayerArchetype(3011)
	assert.False(t, chosen)

	answer(t, u, "warrior")
	answer(t, u, "yes")
	kit := kitOf(t, m, "warrior")
	assert.Equal(t, kit, backpackIDs(u))

	// Disconnect after choosing: creation resumes at the tutorial question.
	u.ClearPrompt()
	text := start(t, u)
	assert.NotContains(t, text, "Starter kit:")
	assert.Equal(t, tutorialQuestion, pending(u))
	assert.Equal(t, kit, backpackIDs(u), "no second kit")
}

func TestWiringStartConfirmNoAsksAgain(t *testing.T) {
	m := registered(t)
	u := creating(t, 3012)
	start(t, u)
	answer(t, u, "rogue")
	answer(t, u, "no")
	assert.Equal(t, `Which archetype will you follow?`, pending(u))
	_, chosen := m.PlayerArchetype(3012)
	assert.False(t, chosen)
	assert.Empty(t, u.Character.Items)

	answer(t, u, "cleric")
	answer(t, u, "yes")
	id, _ := m.PlayerArchetype(3012)
	assert.Equal(t, "cleric", id)
}

func TestWiringStartRejectsUnknownAnswer(t *testing.T) {
	registered(t)
	u := creating(t, 3013)
	start(t, u)
	text := answer(t, u, "bard")
	assert.Contains(t, text, "isn't one of the archetypes")
	assert.Equal(t, `Which archetype will you follow?`, pending(u))
	text = answer(t, u, "9")
	assert.Contains(t, text, "isn't one of the archetypes")
}

func TestWiringStartCommitFailureContinues(t *testing.T) {
	m := registered(t)
	m.store.(*fakeStore).saveErr = errors.New("disk full")
	u := creating(t, 3014)
	start(t, u)
	answer(t, u, "wizard")
	text := answer(t, u, "yes")
	assert.Contains(t, text, "retry")
	assert.Contains(t, text, "archetype choose")
	assert.Equal(t, tutorialQuestion, pending(u), "creation is never blocked on the module")
	assert.Empty(t, u.Character.Items)

	// Re-entering the same prompt doesn't ask again.
	start(t, u)
	assert.Equal(t, tutorialQuestion, pending(u))
}

func TestWiringStartWithoutProviderUnchanged(t *testing.T) {
	archetypes.SetProvider(nil)
	u := creating(t, 3015)
	text := start(t, u)
	assert.NotContains(t, text, "archetype")
	assert.Equal(t, tutorialQuestion, pending(u), "the upstream flow")
}

func TestWiringStartSkipsChosenArchetype(t *testing.T) {
	m := registered(t)
	m.registry.Players[3016] = "warrior" // chosen before creation reached here
	u := creating(t, 3016)
	start(t, u)
	assert.Equal(t, tutorialQuestion, pending(u))
}

func TestWiringPlayerSpawnRecoversKit(t *testing.T) {
	m := registered(t)
	id := events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	t.Cleanup(func() { events.UnregisterListener(events.PlayerSpawn{}, id) })

	// A committed choice whose grant was lost before any user save.
	m.registry.Players[3017] = "ranger"
	m.registry.Kits[3017] = "ranger"
	u := creating(t, 3017)

	text := captureText(t, func() {
		events.AddToQueue(events.PlayerSpawn{UserId: 3017})
		events.ProcessEvents()
	})
	assert.Contains(t, text, "Ranger starter kit")
	kit := kitOf(t, m, "ranger")
	assert.Equal(t, kit, backpackIDs(u))

	events.AddToQueue(events.PlayerSpawn{UserId: 3017})
	events.ProcessEvents()
	assert.Equal(t, kit, backpackIDs(u), "exactly once")
}
