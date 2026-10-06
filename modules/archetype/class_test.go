package archetype

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Phase 38b: a player's class and talents in the archetype registry.

func classModule(t *testing.T, archetype string) (*ArchetypeModule, *fakeStore) {
	t.Helper()
	m, store := testModule(t)
	m.registry.Players[7] = archetype
	return m, store
}

func TestPromotePlayerSavesAndReads(t *testing.T) {
	m, store := classModule(t, "cleric")
	require.NoError(t, m.PromotePlayer(7, "priest"))
	assert.Equal(t, classes.State{Class: "priest"}, m.PlayerClass(7))
	require.NotNil(t, store.saved)
	assert.Equal(t, "priest", store.saved.Classes[7].Class, "saved at once")
	saves := store.saves
	// A repeated promotion to the same class changes nothing.
	require.NoError(t, m.PromotePlayer(7, "priest"))
	assert.Equal(t, saves, store.saves)
	// The elite step moves up; going back down does nothing.
	require.NoError(t, m.PromotePlayer(7, "hierarch"))
	assert.Equal(t, "hierarch", m.PlayerClass(7).Class)
	require.NoError(t, m.PromotePlayer(7, "priest"))
	assert.Equal(t, "hierarch", m.PlayerClass(7).Class, "a class is never lowered")
}

func TestPromotePlayerRefusals(t *testing.T) {
	m, _ := testModule(t)
	assert.ErrorIs(t, m.PromotePlayer(7, "priest"), ErrNotPromotable, "no archetype yet")
	m.registry.Players[7] = "cleric"
	assert.ErrorIs(t, m.PromotePlayer(7, "knight"), classes.ErrNotAvailable, "another lineage's class")
	assert.ErrorIs(t, m.PromotePlayer(7, "nonsense"), classes.ErrNoSuchClass)
	assert.Empty(t, m.PlayerClass(7).Class)
}

func TestPromotePlayerRollsBackWhenTheSaveFails(t *testing.T) {
	m, store := classModule(t, "cleric")
	store.saveErr = errors.New("disk full")
	assert.Error(t, m.PromotePlayer(7, "priest"))
	assert.Empty(t, m.PlayerClass(7).Class, "nothing changed")

	store.saveErr = nil
	require.NoError(t, m.PromotePlayer(7, "priest"))
	store.saveErr = errors.New("disk full")
	assert.Error(t, m.PromotePlayer(7, "hierarch"))
	assert.Equal(t, "priest", m.PlayerClass(7).Class, "the earlier class is kept")
}

func TestPickPlayerTalentChecksTheRulesAndRollsBack(t *testing.T) {
	m, store := classModule(t, "cleric")
	assert.ErrorIs(t, m.PickPlayerTalent(7, 4, "deep-well"), classes.ErrNoTalentOwed)
	assert.ErrorIs(t, m.PickPlayerTalent(7, 5, "toughness"), classes.ErrUnknownTalent, "another lineage's talent")

	require.NoError(t, m.PickPlayerTalent(7, 5, "deep-well"))
	assert.Equal(t, []string{"deep-well"}, m.PlayerClass(7).Talents)
	assert.Equal(t, []string{"deep-well"}, store.saved.Classes[7].Talents, "saved at once")
	assert.ErrorIs(t, m.PickPlayerTalent(7, 14, "deep-well"), classes.ErrNoTalentOwed)

	store.saveErr = errors.New("disk full")
	assert.Error(t, m.PickPlayerTalent(7, 15, "deep-well"))
	assert.Equal(t, []string{"deep-well"}, m.PlayerClass(7).Talents, "a failed save keeps the earlier picks")
	store.saveErr = nil
	require.NoError(t, m.PickPlayerTalent(7, 15, "deep-well"))
	assert.ErrorIs(t, m.PickPlayerTalent(7, 25, "deep-well"), classes.ErrTalentMaxed)

	// No archetype, no talents.
	delete(m.registry.Players, 7)
	assert.ErrorIs(t, m.PickPlayerTalent(7, 25, "mending-hands"), ErrNotPromotable)
}

func TestClassStateDecodesAndOldFilesReadAsUnpromoted(t *testing.T) {
	old := []byte("players:\n  7: cleric\n")
	reg := NewRegistry()
	require.NoError(t, decodeRegistry(old, reg))
	assert.Equal(t, "cleric", reg.Players[7])
	assert.Empty(t, reg.Classes, "an old save has no class state")

	data, err := yaml.Marshal(Registry{Players: map[int]string{7: "cleric"}, Classes: map[int]ClassRecord{7: {Class: "Priest", Talents: []string{"deep-well"}}, 8: {}, 0: {Class: "priest"}}})
	require.NoError(t, err)
	reg = NewRegistry()
	require.NoError(t, decodeRegistry(data, reg))
	assert.Equal(t, ClassRecord{Class: "priest", Talents: []string{"deep-well"}}, reg.Classes[7])
	assert.NotContains(t, reg.Classes, 8, "an empty record is dropped")
	assert.NotContains(t, reg.Classes, 0, "an invalid user id is dropped")

	cp := reg.Clone()
	cp.Classes[7].Talents[0] = "changed"
	assert.Equal(t, "deep-well", reg.Classes[7].Talents[0], "a clone is independent")
}

func TestClassStateIsClearedWithTheCharacter(t *testing.T) {
	m, store := classModule(t, "cleric")
	require.NoError(t, m.PromotePlayer(7, "priest"))
	require.NoError(t, m.clearCharacter(7))
	assert.Empty(t, m.PlayerClass(7).Class, "a permanent death clears the class with the archetype")
	assert.NotContains(t, store.saved.Classes, 7)

	assert.ErrorIs(t, m.PromotePlayer(7, "priest"), ErrNotPromotable, "no archetype now")
}

func TestAdminResetClearsTheClassToo(t *testing.T) {
	m, _ := classModule(t, "cleric")
	require.NoError(t, m.PromotePlayer(7, "priest"))
	_, err := m.resetChoice(7)
	require.NoError(t, err)
	assert.Empty(t, m.PlayerClass(7).Class)
}

func TestPurgeDropsClassState(t *testing.T) {
	m, _ := classModule(t, "cleric")
	require.NoError(t, m.PromotePlayer(7, "priest"))
	assert.True(t, m.purge(7))
	assert.Empty(t, m.PlayerClass(7).Class)
}

func TestClassSpellsAreTaughtAtTheirRank(t *testing.T) {
	m, _ := classModule(t, "cleric")
	user := newUser(7)
	user.Character.Level = 10
	require.NoError(t, m.PromotePlayer(7, "priest"))
	// users.GetByUserId isn't wired in this harness, so grant directly.
	got := m.grantClassSpells(user)
	assert.ElementsMatch(t, classes.SpellsAt("priest", 10), got)
	assert.Empty(t, m.grantClassSpells(user), "granting is idempotent")
	user.Character.Level = 15
	assert.ElementsMatch(t, classes.SpellsAt("priest", 15)[len(classes.SpellsAt("priest", 10)):], m.grantClassSpells(user))
}
