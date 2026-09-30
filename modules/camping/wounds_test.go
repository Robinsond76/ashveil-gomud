package camping

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// supplies gives the camp bandages and splints and counts what it spends.
func supplies(e *innEnv, bandages, splints int) map[wounds.Item]int {
	left := map[wounds.Item]int{wounds.Bandage: bandages, wounds.Splint: splints}
	spent := map[wounds.Item]int{}
	e.module.spendSupply = func(_ int, item wounds.Item) bool {
		if left[item] <= 0 {
			return false
		}
		left[item]--
		spent[item]++
		return true
	}
	return spent
}

// Owner, 2026-09-30: a camp rest closes a broken bone only with a splint
// and a cut or a puncture only with a bandage, one each; the rest stay.
func TestCampRestHealsWoundsOnlyWithTheirItem(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	spent := supplies(e, 1, 0)
	user.Character.Wounds = []wounds.Wound{
		{Kind: wounds.Fracture, Place: "arm", Points: 3},
		{Kind: wounds.Cut, Place: "hand", Points: 2},
		{Kind: wounds.Bruise, Place: "back", Points: 1, Light: true},
	}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Puncture, Place: "leg", Points: 2}}
	e.completeCamp(t, user)
	assert.NotEmpty(t, user.Character.Wounds, "the timer heals nothing: the grant does")

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, map[wounds.Item]int{wounds.Bandage: 1}, spent, "no splint to spend; one bandage")
	assert.Equal(t, []wounds.Wound{{Kind: wounds.Fracture, Place: "arm", Points: 3}}, user.Character.Wounds,
		"the broken arm needs a splint; the cut took the bandage; the light one closed")
	assert.Len(t, e.companion.Wounds, 1, "no bandage left for Bran")
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Your cut hand has knit. (wound healed; a bandage used)")
	assert.NotContains(t, out, "broken arm has knit")
	assert.Contains(t, out, "With no splints or bandages left, 2 wounds stay open. Splints, bandages, an inn, or a physician will close them")
	assert.NotContains(t, out, "back", "a light wound has no line")
}

func TestCampRestSetsABoneWithASplint(t *testing.T) {
	e, user := heroEnv(t)
	messages := captureMessages(t)
	spent := supplies(e, 5, 1)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "ribs", Points: 3}}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 2}}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, map[wounds.Item]int{wounds.Splint: 1}, spent, "bandages never set a bone")
	assert.Empty(t, user.Character.Wounds)
	assert.Len(t, e.companion.Wounds, 1)
	events.ProcessEvents()
	out := strings.Join(*messages, "\n")
	assert.Contains(t, out, "Your cracked ribs have knit. (wound healed; a splint used)")
	assert.Contains(t, out, "With no splints left, 1 wound stays open. Splints, an inn, or a physician will close it")
}

func TestCampRestWithNoSuppliesHealsNoWounds(t *testing.T) {
	e, user := heroEnv(t)
	supplies(e, 0, 0)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeCamp(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Len(t, user.Character.Wounds, 1)
}

// An inn stay heals every wound, with nothing spent.
func TestInnStayHealsWounds(t *testing.T) {
	e, user := heroEnv(t)
	spent := supplies(e, 5, 5)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.companion.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 4}}
	e.completeInn(t, user)
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, user.Character.Wounds)
	assert.Empty(t, e.companion.Wounds)
	assert.Empty(t, spent, "an inn needs no bandages or splints")
}

// A rest that couldn't be saved heals nothing yet: it retries.
func TestUnsavedRestHealsNoWounds(t *testing.T) {
	e, user := heroEnv(t)
	supplies(e, 5, 5)
	user.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "hand", Points: 2}}
	e.completeCamp(t, user)
	e.store.saveErr = assert.AnError
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Len(t, user.Character.Wounds, 1)
	e.store.saveErr = nil
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Empty(t, user.Character.Wounds)
}
