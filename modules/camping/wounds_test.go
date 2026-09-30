package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 30b: a completed camp rest knits every wound of the leader and the
// live companions, with a line for each lasting one.
func TestCampRestHealsWounds(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "back", Points: 1, Light: true}}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "ribs", Points: 2}}
	e.completeCamp(t, user)
	assert.NotEmpty(t, user.Character.Wounds, "the timer heals nothing: the grant does")

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, user.Character.Wounds)
	assert.Empty(t, e.companion.Wounds)
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Your broken arm has knit. (wound healed)")
	assert.Contains(t, out, "Bran</ansi>'s cracked ribs have knit. (wound healed)")
	assert.NotContains(t, out, "back", "a light wound has no line")
}

// An inn stay heals wounds too.
func TestInnStayHealsWounds(t *testing.T) {
	e, user := heroEnv(t)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, user.Character.Wounds)
}

// A rest that couldn't be saved heals nothing yet: it retries.
func TestUnsavedRestHealsNoWounds(t *testing.T) {
	e, user := heroEnv(t)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeCamp(t, user)
	e.store.saveErr = assert.AnError
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Len(t, user.Character.Wounds, 1)
	e.store.saveErr = nil
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Empty(t, user.Character.Wounds)
}
