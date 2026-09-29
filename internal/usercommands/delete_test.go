package usercommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// deleter is a user who can type `delete character` and answer its
// questions as the world would (world.processInput answers the open
// question, then runs the prompt's command again), with what they are
// told and the despawns queued.
type deleter struct {
	t        *testing.T
	user     *users.UserRecord
	heard    *[]string
	despawns *[]events.PlayerDespawn
}

func newDeleter(t *testing.T) *deleter {
	t.Helper()
	mudlog.SetupLogger(nil, "", "", false)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "users"), 0755))
	previous := configs.Flatten(configs.GetOverrides())
	flat := configs.Flatten(configs.GetOverrides())
	flat["FilePaths.DataFiles"] = dir
	require.NoError(t, configs.RestoreOverrides(flat))
	t.Cleanup(func() { require.NoError(t, configs.RestoreOverrides(previous)) })
	events.ProcessEvents()

	u := users.NewUserRecord(7, 0)
	u.Username = "acctdain"
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter22"), bcrypt.MinCost)
	require.NoError(t, err)
	u.Password = string(hash)
	u.Character.Name = "Dain"
	u.Character.RoomId = 1
	u.Character.Health = 10
	users.SetTestUser(u)
	t.Cleanup(func() { users.RemoveTestUser(7) })

	d := &deleter{t: t, user: u, heard: &[]string{}, despawns: &[]events.PlayerDespawn{}}
	mid := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m := e.(events.Message); m.UserId == 7 {
			*d.heard = append(*d.heard, m.Text)
		}
		return events.Continue
	})
	did := events.RegisterListener(events.PlayerDespawn{}, func(e events.Event) events.ListenerReturn {
		*d.despawns = append(*d.despawns, e.(events.PlayerDespawn))
		return events.Cancel // the despawn itself is the hooks' test
	})
	t.Cleanup(func() {
		events.UnregisterListener(events.Message{}, mid)
		events.UnregisterListener(events.PlayerDespawn{}, did)
	})
	return d
}

func (d *deleter) run(rest string) string {
	d.t.Helper()
	*d.heard = nil
	_, err := Delete(rest, d.user, nil, 0)
	require.NoError(d.t, err)
	events.ProcessEvents()
	return strings.Join(*d.heard, "")
}

// answer types a reply to the open question.
func (d *deleter) answer(text string) string {
	d.t.Helper()
	p := d.user.GetPrompt()
	require.NotNil(d.t, p, "a question is open")
	q := p.GetNextQuestion()
	require.NotNil(d.t, q)
	q.Answer(text)
	return d.run(p.Rest)
}

func (d *deleter) openQuestion() string {
	if p := d.user.GetPrompt(); p != nil {
		if q := p.GetNextQuestion(); q != nil {
			return q.Question
		}
	}
	return ""
}

// TestDeleteCharacterDeletesOnPasswordAndName (Phase 32h): the warning,
// a masked password question, then the name; both right flag the record,
// save it, and take the character out of the world with a hand-off.
func TestDeleteCharacterDeletesOnPasswordAndName(t *testing.T) {
	d := newDeleter(t)
	got := d.run("character")
	assert.Contains(t, got, "for good")
	assert.Contains(t, got, "Your login stays")
	require.Equal(t, "Type your password to delete Dain, or anything else to cancel:", d.openQuestion())
	assert.True(t, d.user.GetPrompt().GetNextQuestion().Masked, "the password isn't echoed")

	d.answer("hunter22")
	require.Equal(t, "Type Dain to confirm, or anything else to cancel:", d.openQuestion())
	assert.False(t, d.user.GetPrompt().GetNextQuestion().Masked)

	got = d.answer("dain")
	assert.Contains(t, got, "is gone.")
	assert.Nil(t, d.user.GetPrompt())
	assert.True(t, d.user.Deleting)
	saved, err := users.LoadUserFile(7)
	require.NoError(t, err)
	assert.True(t, saved.Deleting, "flagged on file before leaving")
	require.Len(t, *d.despawns, 1)
	assert.True(t, (*d.despawns)[0].HandOff, "the connection stays for the new character")
	assert.Equal(t, 7, (*d.despawns)[0].UserId)
}

// TestDeleteCharacterCancels: a wrong name after the right password, or a
// wrong password, deletes nothing; three wrong passwords lock the command
// until the next login.
func TestDeleteCharacterCancels(t *testing.T) {
	d := newDeleter(t)
	d.run("character")
	d.answer("hunter22")
	assert.Contains(t, d.answer("Brom"), DeleteNothing)
	assert.False(t, d.user.Deleting)
	assert.Empty(t, *d.despawns)

	for i := 1; i <= 3; i++ {
		d.run("character")
		got := d.answer("wrong")
		assert.Contains(t, got, DeleteNothing, "try %d", i)
		assert.Nil(t, d.user.GetPrompt())
	}
	assert.Contains(t, d.run("character"), "too many times", "locked")
	assert.Nil(t, d.user.GetPrompt(), "no question asked while locked")

	// A new login starts the count again.
	d.user.SetTempData(deleteFailsKey, nil)
	d.run("character")
	assert.NotEmpty(t, d.openQuestion())
	assert.False(t, d.user.Deleting)
	assert.Empty(t, *d.despawns)
}

// TestDeleteCharacterRefusals: only "delete character"; never in a fight,
// in a battle, while down, in a replay, or before a character exists.
func TestDeleteCharacterRefusals(t *testing.T) {
	d := newDeleter(t)
	assert.Contains(t, d.run(""), "delete character</ansi>. See")
	assert.Contains(t, d.run("dain"), "delete character</ansi>. See")
	assert.Nil(t, d.user.GetPrompt())

	d.user.Character.Aggro = &characters.Aggro{MobInstanceId: 5}
	assert.Contains(t, d.run("character"), "too busy fighting")
	d.user.Character.Aggro = nil

	d.user.Character.Health = 0
	assert.Contains(t, d.run("character"), "while you're down")
	d.user.Character.Health = 10

	d.user.ReplayOf = 3
	assert.Contains(t, d.run("character"), "practice character")
	d.user.ReplayOf = 0

	d.user.Character.RoomId = -1
	assert.Contains(t, d.run("character"), "nothing to delete")
	d.user.Character.RoomId = 1

	// A fight that starts mid-question cancels it.
	d.run("character")
	d.answer("hunter22")
	d.user.Character.Aggro = &characters.Aggro{MobInstanceId: 5}
	assert.Contains(t, d.answer("Dain"), "too busy fighting")
	assert.False(t, d.user.Deleting)
	assert.Empty(t, *d.despawns)
	assert.Nil(t, d.user.GetPrompt())
}

// TestWrongPasswordsCountInARow (32h review): a right password clears the
// count, and the lock covers `password` too, so neither command is a way
// to guess a password.
func TestWrongPasswordsCountInARow(t *testing.T) {
	d := newDeleter(t)
	for i := 0; i < 2; i++ {
		d.run("character")
		d.answer("wrong")
	}
	d.run("character")
	d.answer("hunter22")
	d.answer("Brom") // right password, wrong name: cancelled, count cleared
	d.run("character")
	assert.NotContains(t, d.answer("wrong"), "too many times", "one wrong in a row")
	d.run("character")
	d.answer("wrong")
	d.run("character")
	assert.Contains(t, d.answer("wrong"), "too many times", "three in a row")

	*d.heard = nil
	_, err := Password("", d.user, nil, 0)
	require.NoError(t, err)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*d.heard, ""), "too many times", "password is locked too")
	assert.Nil(t, d.user.GetPrompt())
}
