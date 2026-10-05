package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 35b: a guardian's guards grow with its level, fixed as the real
// battle begins; a level gained mid-battle waits for the next battle.
func TestGuardsAreCapturedByLevelAsTheBattleBegins(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	b.cmd("strategy", "tamsin guard me")
	tamsin := b.companion(1)
	tamsin.Character.Level = 20
	b.cmd("attack", fmt.Sprintf("#%d", b.bandits["bandit captain"][0]))
	b.toughen()
	b.hold(nil)
	for _, m := range b.livingBandits() {
		m.Character.SetAggro(0, b.companion(4).InstanceId, characters.DefaultAttack)
		m.Character.Aggro.RoundsWaiting = 1
	}
	b.fight()
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	assert.Equal(t, 4, battle.GuardsLeft(7, "companion:1"), "level 20 guards four times")

	tamsin.Character.Level = 30
	assert.Equal(t, 4, battle.GuardsLeft(7, "companion:1"), "fixed for this battle")
}
