package connections

import "github.com/GoMudEngine/GoMud/internal/term"

// Ashveil 32h: masking in-game input. An in-game prompt that asks for a
// password (delete character, password) masks the connection while that
// question is open, the way the login prompt masks its password steps:
// the web client switches its input field to a password field
// (TEXTMASK), Mudlet reads telnet WILL ECHO as "hide this field", and raw
// telnet gets a star per character from the echo handler instead of the
// text. Masked input is also kept out of the input history.

// SetInputMasked masks or unmasks a connection's input. It tells the
// client only when the state changes.
func SetInputMasked(id ConnectionId, masked bool) {
	lock.Lock()
	cd, ok := netConnections[id]
	if !ok || cd.inputMasked == masked {
		lock.Unlock()
		return
	}
	cd.inputMasked = masked
	isWebsocket := cd.IsWebSocket()
	isMudlet := cd.clientSettings.IsMudlet
	lock.Unlock()

	if isWebsocket {
		if masked {
			SendTo([]byte(`TEXTMASK:true`), id)
		} else {
			SendTo([]byte(`TEXTMASK:false`), id)
		}
	}
	if isMudlet {
		if masked {
			SendTo(term.TelnetWILL(term.TELNET_OPT_ECHO), id)
		} else {
			SendTo(term.TelnetWONT(term.TELNET_OPT_ECHO), id)
		}
	}
}

// InputMasked reports whether a connection's input is masked.
func InputMasked(id ConnectionId) bool {
	lock.Lock()
	defer lock.Unlock()
	if cd, ok := netConnections[id]; ok {
		return cd.inputMasked
	}
	return false
}
