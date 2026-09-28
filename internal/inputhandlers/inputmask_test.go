package inputhandlers

import (
	"bytes"
	"net"
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
