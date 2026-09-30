package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// bandages gives the camp n bandages and counts what it spends.
func bandages(e *innEnv, n int) *int {
	spent := 0
	e.module.spendBandage = func(int) bool {
		if n <= 0 {
			return false
		}
		n--
		spent++
		return true
	}
	return &spent
}

// Phase 30b follow-up (owner, 2026-09-30): a camp rest closes a lasting
// wound only with a bandage, one each; the rest stay open.
func TestCampRestHealsWoundsOnlyWithBandages(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	spent := bandages(e, 1)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}, {Kind: wounds.Bruise, Place: "back", Points: 1, Light: true}}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "ribs", Points: 2}}
	e.completeCamp(t, user)
	assert.NotEmpty(t, user.Character.Wounds, "the timer heals nothing: the grant does")

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 1, *spent)
	assert.Empty(t, user.Character.Wounds, "one bandage: the leader's wound; the light one closes too")
	assert.Len(t, e.companion.Wounds, 1, "no bandage left for Bran")
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Your broken arm has knit. (wound healed; a bandage used)")
	assert.Contains(t, out, "With no bandages left, 1 wound stays open. Bandages, an inn, or a physician will close them")
	assert.NotContains(t, out, "back", "a light wound has no line")
}

func TestCampRestWithNoBandagesHealsNoWounds(t *testing.T) {
	e, user := heroEnv(t)
	bandages(e, 0)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Len(t, user.Character.Wounds, 1)
}

// An inn stay heals every wound, with no bandage.
func TestInnStayHealsWounds(t *testing.T) {
	e, user := heroEnv(t)
	spent := bandages(e, 5)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 4}}
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, user.Character.Wounds)
	assert.Empty(t, e.companion.Wounds)
	assert.Equal(t, 0, *spent, "an inn needs no bandages")
}

// A rest that couldn't be saved heals nothing yet: it retries.
func TestUnsavedRestHealsNoWounds(t *testing.T) {
	e, user := heroEnv(t)
	bandages(e, 5)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeCamp(t, user)
	e.store.saveErr = assert.AnError
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Len(t, user.Character.Wounds, 1)
	e.store.saveErr = nil
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Empty(t, user.Character.Wounds)
}
