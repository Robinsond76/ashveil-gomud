package company

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	domain "github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/dolls"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 39d wiring: the Doll Master's dolls through the real battle pass,
// ability pass and combat round (shipped config, DoCombat), in the brawl
// world. Tamsin is the Doll Master; every other member's abilities are off
// and every blow lands.

// dollBrawl is a brawl whose first companion is a level-level Doll Master;
// the fight has begun on the captain.
func dollBrawl(t *testing.T, level int) (*brawl, *[]combatstream.Event) {
	t.Helper()
	hooks.ResetDollsForTest()
	domain.ResetDollsForTest()
	t.Cleanup(domain.ResetDollsForTest)
	t.Cleanup(hooks.ResetDollsForTest)
	b, stream := abilityBrawl(t, map[int]string{1: "dollmaster", 2: "cleric", 3: "warrior", 4: "ranger"})
	for _, who := range []string{"garrick", "ysolde", "oswin"} {
		b.cmd("strategy", who+" abilities off")
	}
	alwaysLand(t)
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.HPArchetype = "dollmaster"
	b.start()
	return b, stream
}

// doll is Tamsin's standing doll at an index.
func (b *brawl) doll(index int) *mobs.Mob {
	b.t.Helper()
	master, ok := dolls.Of(7, domain.CompanionMemberKey(1))
	require.True(b.t, ok)
	mob, standing := dolls.Live(master, index)
	require.True(b.t, standing, "doll %d stands", index)
	return mob
}

func TestADollStrikesInItsMastersPlace(t *testing.T) {
	b, stream := dollBrawl(t, 3)
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	round := since(*stream, n)
	pip := b.doll(0)
	assert.Equal(t, "Pip", pip.Character.Name)
	assert.Empty(t, swingsBy(round, "Tamsin Reed"), "the Master's turn is the doll's strike")
	assert.NotEmpty(t, swingsBy(round, "Pip"), "the doll strikes\n%s", out)
	require.NotEmpty(t, abilityEvents(round, "Tamsin Reed"))
	assert.Equal(t, "Puppet Strike", abilityEvents(round, "Tamsin Reed")[0].Status)

	// It holds a cell in the formation but takes no company slot.
	f, _ := domain.FormationFor(7)
	_, _, placed := f.Find(domain.DollMemberKey(domain.CompanionMemberKey(1), 0))
	assert.True(t, placed, "the doll stands in the formation")
	assert.Len(t, b.companyInstances(), 4, "still a company of leader and four")
}

func TestABrokenDollStaysOutUntilMended(t *testing.T) {
	b, _ := dollBrawl(t, 3)
	b.hardenBandits()
	b.fight()
	pip := b.doll(0)
	id := pip.InstanceId
	pip.Character.Health = 0
	hooks.ResetAbilitiesForTest()
	b.toughen()
	b.hardenBandits()
	out := b.fight()
	_ = out
	tamsin := b.companion(1)
	require.Len(t, tamsin.Character.Dolls, 1)
	assert.True(t, tamsin.Character.Dolls[0].Broken, "the record keeps the break")
	_, alive := domain.DollOf(id)
	assert.False(t, alive, "the broken doll leaves the battle")
	assert.Nil(t, mobs.GetInstance(id))
	// Mended, it stands again at the next battle.
	used, _ := dolls.Mend(&tamsin.Character, 1)
	assert.Equal(t, 1, used)
	assert.False(t, tamsin.Character.Dolls[0].Broken)
}

func TestEmergencySpliceStandsADollBackUpOnce(t *testing.T) {
	b, stream := dollBrawl(t, 18)
	b.hardenBandits()
	b.fight()
	tamsin := b.companion(1)
	b.doll(0).Character.Health = 0
	b.toughen()
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	assert.Contains(t, out, "lurches back to its feet")
	pip := b.doll(0)
	assert.Greater(t, pip.Character.Health, 0)
	assert.LessOrEqual(t, pip.Character.Health, pip.Character.HealthLimit()/4+1)
	assert.True(t, tamsin.Character.RT.Spliced)
	// The Master spends that round's turn on the splice: the doll holds.
	assert.Contains(t, out, "works the splice tight")
	assert.Empty(t, swingsBy(since(*stream, n), "Pip"))
	assert.Empty(t, swingsBy(since(*stream, n), "Tamsin Reed"))
	// The next round the doll strikes again.
	b.toughen()
	b.hardenBandits()
	n = len(*stream)
	b.fight()
	assert.NotEmpty(t, swingsBy(since(*stream, n), "Pip"))
	// A second fall breaks it: the splice works once a battle.
	b.doll(0).Character.Health = 0
	b.toughen()
	b.hardenBandits()
	b.fight()
	assert.True(t, tamsin.Character.Dolls[0].Broken)
}

func TestTangleHoldsAFoeBackAndCoolsDown(t *testing.T) {
	b, stream := dollBrawl(t, 12)
	b.hardenBandits()
	n := len(*stream)
	out := b.fight()
	assert.Contains(t, out, "(tangled)")
	tangles := 0
	for _, e := range abilityEvents(since(*stream, n), "Tamsin Reed") {
		if e.Status == "Tangle" {
			tangles++
		}
	}
	assert.Equal(t, 1, tangles)
	// It cools down for the next rounds.
	for i := 0; i < 2; i++ {
		b.toughen()
		b.hardenBandits()
		out = b.fight()
		assert.NotContains(t, out, "(tangled)", "round %d", i+2)
	}
}

func TestGuardStringStepsADollInForAHurtAlly(t *testing.T) {
	b, _ := dollBrawl(t, 8)
	b.hardenBandits()
	b.fight()
	// A foe now strikes Ysolde, the most hurt member: the doll takes it.
	b.companion(2).Character.Health = 1
	var guards int
	for i := 0; i < 6 && guards == 0; i++ {
		b.companion(2).Character.Health = 1
		for _, m := range b.livingBandits() {
			m.Character.SetAggro(0, b.companion(2).InstanceId, characters.DefaultAttack)
		}
		b.toughen()
		b.companion(2).Character.Health = 1
		b.hardenBandits()
		out := b.fight()
		if strings.Contains(out, "steps in") || strings.Contains(out, "guards") {
			guards++
		}
	}
	assert.Positive(t, guards, "the doll guarded a hurt ally")
	assert.Positive(t, b.companion(1).Character.RT.DollGuards, "the Master counts the guard")
}
