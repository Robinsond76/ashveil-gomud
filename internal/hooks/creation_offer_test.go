package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/appearance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lifestory"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// creationOffers runs offerCreation for a user and returns the `creation`
// commands it queued for them.
func creationOffers(t *testing.T, u *users.UserRecord) int {
	t.Helper()
	events.ClearQueueForTest()
	t.Cleanup(events.ClearQueueForTest)
	n := 0
	id := events.RegisterListener(events.Input{}, func(e events.Event) events.ListenerReturn {
		if in, ok := e.(events.Input); ok && in.UserId == u.UserId && in.InputText == `creation` {
			n++
		}
		return events.Continue
	})
	defer events.UnregisterListener(events.Input{}, id)
	offerCreation(u)
	events.ProcessEvents()
	return n
}

// Phase 72a: a character with neither looks nor a life story is offered the
// creation steps when they join, once; the offer is skipped for a character
// that has both, was already asked, is still in the void, or is in a fight.
func TestJoinOffersCreationOnce(t *testing.T) {
	newHandOffWorld(t)
	appearance.SetData(&appearance.Data{})
	lifestory.SetData(&lifestory.Data{})
	t.Cleanup(func() { appearance.SetData(nil); lifestory.SetData(nil) })

	fresh := func(id int, name string) *users.UserRecord {
		return saveUser(t, id, name)
	}

	u := fresh(932201, "Legacy")
	assert.Equal(t, 1, creationOffers(t, u), "an existing character is offered the steps")

	u.Character.MarkCreationOffered()
	assert.Equal(t, 0, creationOffers(t, u), "asked once, never again")

	done := fresh(932202, "Written")
	done.Character.Looks = map[string]string{"build": "lean"}
	done.Character.LifeStory = map[string]string{"homeland": "coast"}
	assert.Equal(t, 0, creationOffers(t, done), "looks and a story already chosen")

	partial := fresh(932203, "Partial")
	partial.Character.Looks = map[string]string{"build": "lean"}
	assert.Equal(t, 1, creationOffers(t, partial), "a missing half is still offered")

	void := fresh(932204, "Newborn")
	void.Character.RoomId = -1
	assert.Equal(t, 0, creationOffers(t, void), "creation's own steps run for a character in the void")
}
