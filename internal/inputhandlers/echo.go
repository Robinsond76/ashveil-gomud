package inputhandlers

import (
	"bytes"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

// clickLabelKey is the shared-state key under which a connection's read loop
// leaves the label of a UI click for EchoInputHandler.
const clickLabelKey = "clickLabel"

// echoPrefix opens a line the web client sends for a UI click:
// "!!ECHO(look at Rusty Sword)look !40004:1-032b..." runs the command after
// the parenthesis and shows only the label in the terminal. An empty label
// shows nothing. Typed commands never carry it, so they echo as typed.
var echoPrefix = []byte("!!ECHO(")

// SplitClickLine returns the label and command of a click line, or false
// when the line is not one.
func SplitClickLine(in []byte) (label, cmd []byte, ok bool) {
	if !bytes.HasPrefix(in, echoPrefix) {
		return nil, nil, false
	}
	rest := in[len(echoPrefix):]
	end := bytes.IndexByte(rest, ')')
	if end < 0 {
		return nil, nil, false
	}
	return rest[:end], rest[end+1:], true
}

// NoteClickLine strips the click prefix from a logged-in player's line. It
// returns the command to run, and leaves the label for EchoInputHandler.
func NoteClickLine(line []byte, sharedState map[string]any) []byte {
	delete(sharedState, clickLabelKey)
	label, cmd, ok := SplitClickLine(line)
	if !ok {
		return line
	}
	sharedState[clickLabelKey] = string(label)
	return append([]byte{}, cmd...)
}

func EchoInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) (nextHandler bool) {

	// A UI click: echo only its readable label (nothing when it has none).
	if label, ok := sharedState[clickLabelKey].(string); ok {
		delete(sharedState, clickLabelKey)
		if label != "" {
			connections.SendTo([]byte(label), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
		}
		return true
	}

	// If no actual input, for now just do/change nothing
	if len(clientInput.DataIn) > 0 {
		// echo it back (Ashveil 32h: as stars while the input is masked)
		if connections.InputMasked(clientInput.ConnectionId) {
			connections.SendTo(maskEcho(clientInput.DataIn), clientInput.ConnectionId)
		} else {
			connections.SendTo(clientInput.DataIn, clientInput.ConnectionId)
		}
	}

	// if they didn't hit enter, just keep buffering, go next.
	if !clientInput.EnterPressed {
		return false
	}

	// Echo back their Enter press
	connections.SendTo(term.CRLF, clientInput.ConnectionId)

	return true
}

// maskEcho is input with every printable character replaced by a star;
// control bytes (erasing, line ends) pass through.
func maskEcho(in []byte) []byte {
	out := make([]byte, len(in))
	for i, b := range in {
		if b >= 32 && b != 127 {
			out[i] = '*'
		} else {
			out[i] = b
		}
	}
	return out
}
