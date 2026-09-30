package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30b wiring: wounds through the real combat round (DoCombat).

func lightWounds(c *characters.Character) int {
	n := 0
	for _, w := range c.Wounds {
		if w.Light {
			n++
		}
	}
	return n
}

// A bleed that runs its course leaves a light wound on a companion; one the
// fight's end clears leaves nothing.
func TestABleedThatRunsOutLeavesALightWound(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	b.aimAt("bandit captain")
	b.toughen()
	tamsin := b.companion(1)
	tamsin.Character.Wounds = nil
	require.NoError(t, tamsin.Character.AddBuff(status.Bleeding, false))
	require.NoError(t, tamsin.Character.AddBuff(status.Bleeding, false))
	for i := 0; i < 3; i++ {
		// toughen, not the fixture's full refresh, so the company can't fall
		b.toughen()
		for _, m := range b.livingBandits() {
			m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
		}
		tamsin.Character.Wounds = wounds.Lasting(tamsin.Character.Wounds) // only the bleed's
		b.fight()
	}
	require.False(t, status.Has(&tamsin.Character), "the bleed ran its three rounds")
	var bled *wounds.Wound
	for i, w := range tamsin.Character.Wounds {
		if w.Light && w.Kind == wounds.Cut {
			bled = &tamsin.Character.Wounds[i]
		}
	}
	require.NotNil(t, bled, "a light cut from the bleed: %+v", tamsin.Character.Wounds)
	assert.Equal(t, 2, bled.Points, "one point a stack")
}

// Light wounds close at the fight's end; lasting ones stay.
func TestLightWoundsCloseWithTheFight(t *testing.T) {
	b := newBrawl(t)
	b.aimAt("bandit captain")
	b.toughen()
	b.fight()
	for i := 0; i < 200 && len(b.livingBandits()) > 0; i++ {
		b.toughen()
		b.fight()
	}
	require.Empty(t, b.livingBandits())
	// The last bandit fell; add the wounds before the round that ends it.
	b.aria.Character.Wounds = []wounds.Wound{
		{Kind: wounds.Bruise, Place: "ribs", Points: 2, Light: true},
		{Kind: wounds.Fracture, Place: "arm", Points: 3},
	}
	b.fight()
	assert.Equal(t, 0, lightWounds(b.aria.Character), "light wounds close with the fight")
	require.Len(t, b.aria.Character.Wounds, 1)
	assert.Equal(t, wounds.Fracture, b.aria.Character.Wounds[0].Kind, "the lasting wound stays")
}

// A light wound with no fight (a restart's leftover) closes quietly.
func TestStrayLightWoundCloses(t *testing.T) {
	b := newBrawl(t)
	b.aria.Character.Aggro = nil
	tamsin := b.companion(1)
	tamsin.Character.Aggro = nil
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Bruise, Points: 1, Light: true}}
	tamsin.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Points: 1, Light: true}, {Kind: wounds.Cut, Points: 2}}
	b.fight()
	assert.Empty(t, b.aria.Character.Wounds)
	assert.Len(t, tamsin.Character.Wounds, 1, "only the lasting one stays")
}

// The strategy healer reads the wound limit: a member at their limit is not
// healed, however far below max health.
func TestAClericDoesNotHealAMemberAtTheirLimit(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	for _, m := range b.livingBandits() {
		m.Character.HealthMax.Value, m.Character.Health = 1000, 1000
	}
	b.toughen()
	b.fight()

	// Aria at 300 of 1000, but wounded down to a limit of 300.
	b.aria.Character.HealthMax.Value, b.aria.Character.Health = 1000, 300
	b.aria.Character.Wounds = []wounds.Wound{{Kind: wounds.Fracture, Place: "leg", Points: 700}}
	require.Equal(t, 300, b.aria.Character.HealthLimit())
	b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type, "at her limit: nothing to heal")
	assert.Equal(t, 20, oswin.Character.Mana)
}
