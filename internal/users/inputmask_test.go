package users

import (
	"io"
	"net"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncInputMask (Ashveil 32h): the connection is masked exactly while
// the user's open question is a masked one, and unmasked once it is
// answered or the prompt is cleared.
func TestSyncInputMask(t *testing.T) {
	server, client := net.Pipe()
	go io.Copy(io.Discard, client)
	cd := connections.Add(server, nil)
	id := cd.ConnectionId()
	t.Cleanup(func() {
		connections.Remove(id)
		client.Close()
	})

	u := NewUserRecord(7, uint64(id))
	u.SyncInputMask()
	assert.False(t, connections.InputMasked(id), "no prompt")

	p, _ := u.StartPrompt("delete", "character")
	q := p.Ask("Type your password:", []string{})
	u.SyncInputMask()
	assert.False(t, connections.InputMasked(id), "an ordinary question")

	q.Masked = true
	u.SyncInputMask()
	assert.True(t, connections.InputMasked(id), "a password question")

	q.Done = true
	u.SyncInputMask()
	assert.False(t, connections.InputMasked(id), "answered")

	q.Done = false
	u.SyncInputMask()
	assert.True(t, connections.InputMasked(id))
	u.ClearPrompt()
	u.SyncInputMask()
	assert.False(t, connections.InputMasked(id), "cleared")
}

// TestReconnectKeepsAPasswordQuestionMasked (32h review): a player who
// drops mid-password and reconnects on a new connection finds that
// connection masked before they type the answer.
func TestReconnectKeepsAPasswordQuestionMasked(t *testing.T) {
	replayDataDir(t)
	pipe := func() connections.ConnectionId {
		server, client := net.Pipe()
		go io.Copy(io.Discard, client)
		cd := connections.Add(server, nil)
		t.Cleanup(func() {
			connections.Remove(cd.ConnectionId())
			client.Close()
		})
		return cd.ConnectionId()
	}
	first, second := pipe(), pipe()

	u := veteran()
	loggedIn, _, err := LoginUser(u, first)
	require.NoError(t, err)
	p, _ := loggedIn.StartPrompt("delete", "character")
	p.Ask("Type your password:", []string{}).Masked = true
	loggedIn.SyncInputMask()
	require.True(t, connections.InputMasked(first))

	SetLinkDeadUser(loggedIn.UserId)
	again, msg, err := LoginUser(veteran(), second)
	require.NoError(t, err)
	require.Equal(t, "Reconnecting...", msg)
	assert.Same(t, loggedIn, again, "the same record, prompt and all")
	assert.True(t, connections.InputMasked(second), "masked before the answer is typed")
}

// TestAMaskedPromptRedrawsTheTypingAsStars (32h review M1): raw telnet
// redraws the prompt with what's been typed so far; a password shows as
// stars, with no suggestion.
func TestAMaskedPromptRedrawsTheTypingAsStars(t *testing.T) {
	server, client := net.Pipe()
	go io.Copy(io.Discard, client)
	cd := connections.Add(server, nil)
	id := cd.ConnectionId()
	t.Cleanup(func() {
		connections.Remove(id)
		client.Close()
	})
	u := NewUserRecord(7, uint64(id))
	p, _ := u.StartPrompt("delete", "character")
	p.Ask("Type your password:", []string{}).Masked = true
	u.SyncInputMask()
	u.SetUnsentText("hunter2", "2")
	got := u.GetCommandPrompt()
	assert.NotContains(t, got, "hunter2")
	assert.Contains(t, got, "*******")
}
