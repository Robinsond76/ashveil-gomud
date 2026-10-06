package battle

import "github.com/GoMudEngine/GoMud/internal/sigils"

// Phase 54: the sigil the company stood in when its battle began. It is
// fixed at the battle's start (a battle that outlasts the sigil keeps its
// effect to the end) and is runtime state only, like the battle: the sigil
// itself is saved with the leader's character.

// SetSigil records the sigil under a player's battle and the Unix second it
// fades. A battle without one passes sigils.None.
func SetSigil(userId int, kind sigils.Kind, expires int64) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		b.Sigil, b.SigilExpires = kind, expires
	}
}

// SigilOf is the sigil of the player's battle: None when it has none.
func SigilOf(userId int) sigils.Kind {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := battles[userId]; ok {
		return b.Sigil
	}
	return sigils.None
}
