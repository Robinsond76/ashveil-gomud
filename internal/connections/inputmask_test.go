package connections

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/stretchr/testify/assert"
)

// pipeClient drains the client end of a piped connection.
type pipeClient struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (p *pipeClient) take() []byte {
	time.Sleep(20 * time.Millisecond)
	p.mu.Lock()
	defer p.mu.Unlock()
	out := append([]byte(nil), p.buf.Bytes()...)
	p.buf.Reset()
	return out
}

func pipeConnection(t *testing.T) (ConnectionId, *pipeClient) {
	t.Helper()
	server, client := net.Pipe()
	cd := Add(server, nil)
	out := &pipeClient{}
	go func() {
		chunk := make([]byte, 1024)
		for {
			n, err := client.Read(chunk)
			out.mu.Lock()
			out.buf.Write(chunk[:n])
			out.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		Remove(cd.ConnectionId())
		client.Close()
	})
	return cd.ConnectionId(), out
}

// TestSetInputMasked (Ashveil 32h): masking tells a Mudlet client WILL
// ECHO once, and unmasking WONT ECHO; a raw telnet client is told
// nothing (the echo handler stars its input). An unknown connection is
// never masked.
func TestSetInputMasked(t *testing.T) {
	id, out := pipeConnection(t)
	cs := GetClientSettings(id)
	cs.IsMudlet = true
	OverwriteClientSettings(id, cs)

	assert.False(t, InputMasked(id))
	SetInputMasked(id, true)
	assert.True(t, InputMasked(id))
	assert.Equal(t, term.TelnetWILL(term.TELNET_OPT_ECHO), out.take())
	SetInputMasked(id, true)
	assert.Empty(t, out.take(), "nothing twice")
	SetInputMasked(id, false)
	assert.False(t, InputMasked(id))
	assert.Equal(t, term.TelnetWONT(term.TELNET_OPT_ECHO), out.take())

	raw, rawOut := pipeConnection(t)
	SetInputMasked(raw, true)
	assert.True(t, InputMasked(raw))
	assert.Empty(t, rawOut.take(), "raw telnet is told nothing")

	SetInputMasked(ConnectionId(999999), true)
	assert.False(t, InputMasked(ConnectionId(999999)))
}
