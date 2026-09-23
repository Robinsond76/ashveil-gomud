package archetypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeProvider struct{ disarmed map[string]bool }

func (fakeProvider) CanTrain(int, string) (bool, string)      { return false, "no" }
func (fakeProvider) CanLearnSpell(int, string) (bool, string) { return false, "no spell" }
func (fakeProvider) Exists(id string) bool                    { return id == "wizard" }
func (fakeProvider) ArchetypeName(id string) (string, bool) {
	if id == "wizard" {
		return "Wizard", true
	}
	return "", false
}
func (fakeProvider) PlayerArchetype(userID int) (string, bool) { return "wizard", userID == 1 }
func (f fakeProvider) TrapArmed(lockID string) bool            { return !f.disarmed[lockID] }

type plainProvider struct{ fakeProvider }

// TrapArmed is shadowed away so plainProvider doesn't satisfy TrapProvider.
func (plainProvider) TrapArmed() {}

func TestSeamWithoutProvider(t *testing.T) {
	SetProvider(nil)
	ok, reason := CanTrain(1, "cast")
	assert.True(t, ok)
	assert.Empty(t, reason)
	ok, _ = CanLearnSpell(1, "heal")
	assert.True(t, ok)
	assert.False(t, Exists("wizard"))
	_, ok = Name("wizard")
	assert.False(t, ok)
	_, ok = PlayerArchetype(1)
	assert.False(t, ok)
	assert.True(t, TrapArmed("1-chest"))
}

func TestSeamDelegates(t *testing.T) {
	SetProvider(fakeProvider{disarmed: map[string]bool{"1-chest": true}})
	t.Cleanup(func() { SetProvider(nil) })

	ok, reason := CanTrain(1, "cast")
	assert.False(t, ok)
	assert.Equal(t, "no", reason)
	ok, reason = CanLearnSpell(1, "heal")
	assert.False(t, ok)
	assert.Equal(t, "no spell", reason)
	assert.True(t, Exists("wizard"))
	name, ok := Name("wizard")
	assert.True(t, ok)
	assert.Equal(t, "Wizard", name)
	id, ok := PlayerArchetype(1)
	assert.True(t, ok)
	assert.Equal(t, "wizard", id)
	assert.False(t, TrapArmed("1-chest"))
	assert.True(t, TrapArmed("1-door"))
}

func TestTrapArmedWithoutTrapProvider(t *testing.T) {
	SetProvider(plainProvider{})
	t.Cleanup(func() { SetProvider(nil) })
	assert.True(t, TrapArmed("1-chest"))
}
