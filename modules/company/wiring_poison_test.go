package company

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43b review: a coated blade poisons a foe through the real combat
// round (DoCombat, the Buff event, ApplyBuffs) with no player input, the hit
// line names it, and it ticks and wears off like any combat status.
func TestCoatedBladePoisonsAFoeThroughTheRealRound(t *testing.T) {
	b := newBrawl(t)
	loadStatusBuffs(t)
	buffListener := events.RegisterListener(events.Buff{}, hooks.ApplyBuffs)
	t.Cleanup(func() { events.UnregisterListener(events.Buff{}, buffListener) })
	b.aimAt("bandit captain")
	captain := b.captain()

	b.aria.Character.Equipment.Weapon = items.New(10004) // a dagger
	weapon := &b.aria.Character.Equipment.Weapon
	require.Positive(t, weapon.ItemId, "the leader fights with a dagger")
	now := time.Now()
	require.True(t, weapon.Coat("bitterleaf", items.CoatExpiry(now), items.CoatContacts, now))

	var out strings.Builder
	for i := 0; i < 40 && !status.PoisonLive(&captain.Character); i++ {
		b.toughen()
		captain.Character.HealthMax.Value, captain.Character.Health = 5000, 5000
		out.WriteString(b.fight())
		if !weapon.Coated(time.Now()) {
			require.True(t, weapon.Coat("bitterleaf", items.CoatExpiry(time.Now()), items.CoatContacts, time.Now()))
		}
	}
	require.True(t, status.PoisonLive(&captain.Character), "a coated blade poisons within forty rounds:\n%s", out.String())
	assert.True(t, captain.Character.HasBuff(status.Bitterleaf))
	assert.Contains(t, out.String(), "bitterleaf", "the blow that poisons names it")

	b.toughen()
	captain.Character.HealthMax.Value, captain.Character.Health = 5000, 5000
	assert.Contains(t, b.fight(), "Bitterleaf burns in the bandit captain's veins.", "it ticks on the next round")
}
