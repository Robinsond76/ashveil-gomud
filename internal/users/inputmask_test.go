package users

import (
	"io"
	"net"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/stretchr/testify/assert"
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
