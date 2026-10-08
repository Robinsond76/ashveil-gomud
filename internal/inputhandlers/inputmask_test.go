package inputhandlers

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/stretchr/testify/assert"
)

func maskedPipe(t *testing.T) (connections.ConnectionId, func() string) {
	t.Helper()
	server, client := net.Pipe()
	cd := connections.Add(server, nil)
	var mu sync.Mutex
	var buf bytes.Buffer
	go func() {
		chunk := make([]byte, 1024)
		for {
			n, err := client.Read(chunk)
			mu.Lock()
			buf.Write(chunk[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		connections.Remove(cd.ConnectionId())
		client.Close()
	})
	return cd.ConnectionId(), func() string {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		out := buf.String()
		buf.Reset()
		return out
	}
}

// TestEchoAndHistoryWhileMasked (Ashveil 32h): while a connection is
// masked, what's typed echoes as stars and never reaches the input
// history; unmasked, it echoes and is kept as before.
func TestEchoAndHistoryWhileMasked(t *testing.T) {
	id, sent := maskedPipe(t)
	in := &connections.ClientInput{ConnectionId: id}

	connections.SetInputMasked(id, true)
	sent()
	in.DataIn = []byte("hunter2")
	in.Buffer = []byte("hunter2")
	assert.False(t, EchoInputHandler(in, nil))
	assert.Equal(t, "*******", sent())
	in.DataIn = nil
	in.EnterPressed = true
	assert.True(t, EchoInputHandler(in, nil))
	assert.True(t, HistoryInputHandler(in, nil))
	assert.Nil(t, in.History.Get(), "a password isn't kept")
	assert.NotContains(t, sent(), "hunter2")

	connections.SetInputMasked(id, false)
	in.DataIn = []byte("look")
	in.Buffer = []byte("look")
	in.EnterPressed = false
	EchoInputHandler(in, nil)
	assert.Equal(t, "look", sent())
	in.EnterPressed = true
	HistoryInputHandler(in, nil)
	assert.Equal(t, []byte("look"), in.History.Get())
}

func TestMaskEchoKeepsControlBytes(t *testing.T) {
	assert.Equal(t, []byte("**\b*\r\n"), maskEcho([]byte("ab\b \r\n")))
}

// TestClickLineEchoesItsLabel: a UI click carries a readable label; the
// command runs without its raw form (item ids) reaching the terminal.
func TestClickLineEchoesItsLabel(t *testing.T) {
	id, sent := maskedPipe(t)
	in := &connections.ClientInput{ConnectionId: id, EnterPressed: true}
	state := map[string]any{}

	line := NoteClickLine([]byte("!!ECHO(look at Rusty Sword)look !40004:1-032b99fdf952200-01"), state)
	assert.Equal(t, "look !40004:1-032b99fdf952200-01", string(line))
	in.DataIn, in.Buffer = line, line
	sent()
	assert.True(t, EchoInputHandler(in, state))
	assert.Equal(t, "look at Rusty Sword", strings.TrimSpace(sent()))

	// The label is used once: the next typed line echoes as typed.
	line = NoteClickLine([]byte("look"), state)
	in.DataIn, in.Buffer = line, line
	assert.True(t, EchoInputHandler(in, state))
	assert.Equal(t, "look", strings.TrimSpace(sent()))

	// An empty label hides the echo entirely.
	line = NoteClickLine([]byte("!!ECHO()look !40004:1-0"), state)
	assert.Equal(t, "look !40004:1-0", string(line))
	in.DataIn, in.Buffer = line, line
	assert.True(t, EchoInputHandler(in, state))
	assert.Equal(t, "", sent())

	// An unclosed marker is ordinary typed text.
	line = NoteClickLine([]byte("!!ECHO(look"), state)
	assert.Equal(t, "!!ECHO(look", string(line))
}
