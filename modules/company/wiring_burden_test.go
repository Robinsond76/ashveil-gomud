package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 30g3 through the real round (DoCombat): with every blow landing
// and every dodge certain, unburdened bandits twist aside from all of the
// company's blows; loaded with 40 kg each, they keep only 40% of their
// dodge and blows land.
func TestBurdenLowersDodgeThroughTheRound(t *testing.T) {
	b := newBrawl(t)
	forceBlows(t, true)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 100, 100
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	stream := b.listen()

	bandits := map[string]*mobs.Mob{}
	for _, ids := range b.bandits {
		for _, id := range ids {
			if m := mobs.GetInstance(id); m != nil {
				bandits[key(m)] = m
			}
		}
	}
	keepStanding := func() {
		b.toughen()
		for _, m := range bandits {
			hardTo(&m.Character, 100000)
		}
	}
	// blowsOnBandits tallies the company's strikes on bandits since
	// the stream was last cleared: those that landed and those dodged.
	blowsOnBandits := func() (events, landed, dodged int) {
		for _, e := range ofKind(*stream, combatstream.Attack) {
			if bandits[e.Target.Key()] == nil || bandits[e.Source.Key()] != nil {
				continue
			}
			events++
			if e.Outcome == combatstream.OutcomeHit || e.Outcome == combatstream.OutcomeCrit {
				landed++
			}
			for _, d := range e.Defenses {
				if d == combat.DefenseDodged {
					dodged++
				}
			}
		}
		return
	}

	b.aimAt("bandit captain")
	for i := 0; i < 3; i++ {
		keepStanding()
		b.fight()
	}
	events, landed, dodged := blowsOnBandits()
	require.Greater(t, events, 3, "the company struck at the bandits")
	assert.Zero(t, landed, "unburdened, every blow is dodged")
	assert.Greater(t, dodged, 3)

	// Two tree trunks each (40 kg): fully burdened at a bandit's Strength.
	for _, m := range bandits {
		m.Character.StoreItem(items.New(10013))
		m.Character.StoreItem(items.New(10013))
		require.Equal(t, 1.0, m.Character.Burden(), m.Character.Name)
	}
	*stream = nil
	for i := 0; i < 6; i++ {
		keepStanding()
		b.fight()
	}
	events, landed, dodged = blowsOnBandits()
	require.Greater(t, events, 6, "the company struck at the bandits")
	assert.Greater(t, landed, 0, "burdened, blows land")
	assert.Greater(t, dodged, 0, "and some are still dodged (40%)")
}

// Phase 30g3: look at a character names its burden (a companion, an
// enemy), and look and scout of a group name its burdened members.
func TestBurdenInLookAndScout(t *testing.T) {
	b := newBrawl(t)

	tamsin := b.cmd("look", "tamsin")
	assert.Contains(t, tamsin, "Burden: "+util.CapitalizeFirst(b.companion(1).Character.BurdenWord()), "a companion's burden, from her live mob")

	captain := mobs.GetInstance(b.bandits["bandit captain"][0])
	require.NotNil(t, captain)
	captain.Character.StoreItem(items.New(10013))
	captain.Character.StoreItem(items.New(10013))
	got := b.cmd("look", "bandit captain")
	assert.Contains(t, got, "Burden: Heavily burdened")
	assert.NotContains(t, got, "40.0", "a word, never the weight")

	ruffians := spawnedHostiles(t, 920103)
	require.Len(t, ruffians, 2)
	b.into(920103)
	got = b.cmd("look", "band")
	assert.Contains(t, got, "a ruffian (unhurt), a ruffian (unhurt)", "the unburdened carry no word")
	assert.NotContains(t, b.cmd("scout", "ruffians"), "Burdened:")

	ruffians[0].Character.StoreItem(items.New(10013))
	ruffians[0].Character.StoreItem(items.New(10013))
	assert.Contains(t, b.cmd("look", "band"), "a ruffian (unhurt, heavily burdened), a ruffian (unhurt)")
	got = b.cmd("scout", "ruffians")
	assert.Contains(t, got, "as they stand")
	assert.Contains(t, got, "Burdened: ruffian (heavily burdened). A burdened fighter dodges less.")
}
