package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeletingFlagRoundTrips (Phase 32h): the flag is durable.
func TestDeletingFlagRoundTrips(t *testing.T) {
	replayDataDir(t)
	u := veteran()
	u.Deleting = true
	require.NoError(t, SaveUser(*u))
	loaded, err := LoadUserFile(u.UserId)
	require.NoError(t, err)
	assert.True(t, loaded.Deleting)
}

// TestResetDeletedCharacterKeepsTheAccount: the login and the player's own
// settings stay; the character is new, in the Void; the old name is free;
// the flag clears; an online user is refused; a second run changes nothing.
func TestResetDeletedCharacterKeepsTheAccount(t *testing.T) {
	replayDataDir(t)
	u := veteran()
	u.Role = RoleMod
	u.Macros = map[string]string{"=1": "look"}
	u.ConfigOptions = map[string]any{"prompt": "{hp}>"}
	u.TipsComplete = map[string]bool{"look": true}
	u.Deleting = true
	require.NoError(t, SaveUser(*u))
	GetCharacterIndex().Add("Aria", u.UserId)
	GetCharacterIndex().Add("Brom", 8)
	t.Cleanup(func() { GetCharacterIndex().Remove("Brom") })

	SetTestUser(u)
	assert.Error(t, ResetDeletedCharacter(u.UserId), "never while online")
	RemoveTestUser(u.UserId)

	require.NoError(t, ResetDeletedCharacter(u.UserId))
	got, err := LoadUserFile(u.UserId)
	require.NoError(t, err)
	assert.False(t, got.Deleting)
	assert.Equal(t, "acctaria", got.Username)
	assert.Equal(t, "$2a$hash", got.Password)
	assert.Equal(t, RoleMod, got.Role)
	assert.True(t, got.ScreenReader)
	assert.Equal(t, "kill", got.Aliases["k"])
	assert.Equal(t, "look", got.Macros["=1"])
	assert.Equal(t, "{hp}>", got.ConfigOptions["prompt"])
	assert.True(t, got.TipsComplete["look"])

	fresh := NewUserRecord(1, 0).Character
	fresh.Validate()
	assert.Equal(t, fresh.Name, got.Character.Name, "a new character's name, until creation")
	assert.Equal(t, -1, got.Character.RoomId, "the Void, where creation runs")
	assert.Equal(t, 1, got.Character.Level)
	assert.Equal(t, fresh.Experience, got.Character.Experience)
	assert.Equal(t, characterNewGold(), got.Character.Gold)
	assert.Empty(t, got.Character.GetAllBackpackItems())
	assert.Nil(t, got.Character.GetMiscData("tutorial-state"))

	_, found := GetCharacterIndex().Find("Aria")
	assert.False(t, found, "the old name is free")
	id, found := GetCharacterIndex().Find("Brom")
	assert.True(t, found && id == 8, "another owner's name stays")

	// A new character made since, then a second reset: nothing changes.
	got.Character.Name = "Dain"
	require.NoError(t, SaveUser(*got))
	require.NoError(t, ResetDeletedCharacter(u.UserId))
	again, err := LoadUserFile(u.UserId)
	require.NoError(t, err)
	assert.Equal(t, "Dain", again.Character.Name)
	assert.NoError(t, ResetDeletedCharacter(4242), "a missing file is fine")
}

// TestDeletingUserIdsListsFlaggedRecords: only flagged records, online or
// not; replays and unflagged users aren't listed.
func TestDeletingUserIdsListsFlaggedRecords(t *testing.T) {
	replayDataDir(t)
	flagged := veteran()
	flagged.Deleting = true
	require.NoError(t, SaveUser(*flagged))
	plain := veteran()
	plain.UserId = 8
	require.NoError(t, SaveUser(*plain))
	assert.Equal(t, []int{7}, DeletingUserIds())
}

// TestLoginRefusesAFlaggedRecord: a flagged record waits for its reset.
func TestLoginRefusesAFlaggedRecord(t *testing.T) {
	replayDataDir(t)
	u := veteran()
	u.Deleting = true
	got, msg, err := LoginUser(u, 99)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, DeletingLoginRefusal, msg)
	assert.Nil(t, GetByUserId(u.UserId), "not logged in")
}
