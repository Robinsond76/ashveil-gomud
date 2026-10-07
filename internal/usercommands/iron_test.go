package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/blessings"
	"github.com/GoMudEngine/GoMud/internal/creation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type earnedFake struct{ ids []string }

func (e earnedFake) Earned(int) []string { return e.ids }

// through drives a new character through the steps up to the Iron
// question.
func throughCreation(t *testing.T, id int) *creationUser {
	t.Helper()
	creationWorld(t)
	cu := newCreationUser(t, id, 1)
	cu.start()
	cu.runTo("start", "", `Keep this character as described?`, nil)
	cu.answer("start", "", "confirm")
	require.Equal(t, ironTitle, cu.pending())
	return cu
}

func TestIronOptionIsAskedOnceAndConfirmed(t *testing.T) {
	cu := throughCreation(t, 7701)
	view, ok := creation.Get(7701)
	require.True(t, ok, "the web panel sees the question")
	assert.Equal(t, "iron", view.Step)
	assert.Equal(t, creation.KindChoice, view.Kind)
	require.Len(t, view.Options, 2)
	assert.Equal(t, "standard", view.Options[0].ID)
	assert.Equal(t, "iron", view.Options[1].ID)
	assert.False(t, cu.u.Character.IsIron(), "nothing is decided before the answer")

	cu.answer("start", "", "2") // Iron, by its number, as the web panel sends it
	assert.Equal(t, ironConfirm, cu.pending(), "Iron is confirmed before it is final")
	assert.False(t, cu.u.Character.IsIron(), "still not Iron until confirmed")
	view, _ = creation.Get(7701)
	assert.Equal(t, "iron-confirm", view.Key)

	cu.answer("start", "", "yes")
	assert.True(t, cu.u.Character.IsIron())
	assert.True(t, cu.u.Character.IronOffered())
	assert.Equal(t, tutorialQuestion, cu.pending())
	assert.Contains(t, cu.text(), "You take the Iron")
	_, open := creation.Get(7701)
	assert.False(t, open, "the panel closes with the question")

	// Answering the tutorial question re-runs `start`; the Iron question
	// must not come back, and the choice stays.
	cu.u.GetPrompt().GetNextQuestion().Answer("no")
	cu.start()
	assert.NotEqual(t, ironTitle, cu.pending())
	assert.True(t, cu.u.Character.IsIron())
}

func TestIronCanBeDeclinedAtTheConfirmation(t *testing.T) {
	cu := throughCreation(t, 7702)
	cu.answer("start", "", "iron")
	require.Equal(t, ironConfirm, cu.pending())
	cu.answer("start", "", "no")
	assert.False(t, cu.u.Character.IsIron(), "no, standard")
	assert.True(t, cu.u.Character.IronOffered())
	assert.Equal(t, tutorialQuestion, cu.pending())
}

func TestStandardIsTheOtherAnswer(t *testing.T) {
	cu := throughCreation(t, 7703)
	cu.standard()
	assert.False(t, cu.u.Character.IsIron())
	assert.Equal(t, tutorialQuestion, cu.pending())
}

func TestBlessingsAreGivenOnceAtCreation(t *testing.T) {
	cu := throughCreation(t, 7704)
	require.NoError(t, blessings.Load())
	t.Cleanup(func() { blessings.SetData(nil); blessings.SetProvider(nil) })
	blessings.SetProvider(earnedFake{ids: []string{"road-tested", "company-keeper", "no-such-blessing"}})

	c := cu.u.Character
	before := len(c.Items)
	cu.standard()
	assert.Equal(t, []string{"road-tested", "company-keeper"}, c.Blessings, "an id the world no longer has is skipped")
	assert.Equal(t, before+2, len(c.Items), "road-tested gives two flasks of broth")
	assert.Equal(t, 5, blessings.DiscountPercent(c))
	assert.Contains(t, cu.text(), "Your earlier road follows you")

	// The tutorial answer re-runs start: nothing is given twice.
	cu.u.GetPrompt().GetNextQuestion().Answer("no")
	cu.start()
	assert.Equal(t, before+2, len(c.Items))
	assert.Equal(t, []string{"road-tested", "company-keeper"}, c.Blessings)
}

func TestNoBlessingsMeansNothingGiven(t *testing.T) {
	cu := throughCreation(t, 7705)
	c := cu.u.Character
	before := len(c.Items)
	cu.standard()
	assert.Empty(t, c.Blessings)
	assert.Equal(t, before, len(c.Items))
	assert.NotContains(t, cu.text(), "Your earlier road")
}

func TestOnlineListShowsTheIronBadge(t *testing.T) {
	cu := newCreationUser(t, 7706, 1)
	assert.Equal(t, "Mara", onlineName(cu.u, "Mara"))
	cu.u.Character.SetIron(true)
	assert.Equal(t, "Mara (Iron)", onlineName(cu.u, "Mara"))
}
