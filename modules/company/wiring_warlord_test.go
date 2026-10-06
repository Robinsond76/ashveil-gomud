package company

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38c1 wiring: the Warlord's ranks and the elite talent Second Wind
// through the real strategy command, ability pass and combat round (shipped
// config, DoCombat), in the brawl world. Tamsin is the Warlord; her
// company-mates use no abilities of their own.

// warlordBrawl is an ability brawl with Tamsin a Warlord of the level, and
// every tackle landing.
func warlordBrawl(t *testing.T, level int, talents ...string) *brawl {
	t.Helper()
	b, _ := abilityBrawl(t, map[int]string{1: "warrior", 2: "cleric", 3: "warrior", 4: "ranger"})
	hooks.ResetWarlordForTest()
	t.Cleanup(hooks.ResetWarlordForTest)
	for _, who := range []string{"garrick", "ysolde"} {
		b.cmd("strategy", who+" abilities off")
	}
	tamsin := b.companion(1)
	tamsin.Character.Level = level
	tamsin.Character.SetClassState("warlord", talents)
	return b
}

func TestWarlordMarksTheFoeItTacklesForTwoRounds(t *testing.T) {
	b := warlordBrawl(t, 30)
	b.start()
	tamsin := b.companion(1)
	foe := mobs.GetInstance(aimOf(&tamsin.Character))
	require.NotNil(t, foe)

	out := b.fight()
	assert.Regexp(t, `Tamsin Reed marks the .* for ruin\.`, out)
	assert.Contains(t, out, "marked for ruin: +5 Attack for your allies, 2 rounds")
	require.NotNil(t, foe.Character.RT)
	assert.Equal(t, 5, foe.Character.RT.Mark, "rank 30: +5 Attack for allies against the mark")

	b.toughen()
	b.hardenBandits()
	b.fight()
	assert.Equal(t, 5, foe.Character.RT.Mark, "the mark holds the round after")
	b.toughen()
	b.hardenBandits()
	b.fight()
	assert.Zero(t, foe.Character.RT.Mark, "two rounds, then it lifts")
}

func TestWarlordMarkGrowsAtRankFiftyAndNeedsRankThirty(t *testing.T) {
	for _, tc := range []struct{ level, want int }{{29, 0}, {30, 5}, {49, 5}, {50, 10}} {
		t.Run(fmt.Sprint(tc.level), func(t *testing.T) {
			b := warlordBrawl(t, tc.level)
			if tc.level < 30 {
				b.companion(1).Character.SetClassState("mercenary", nil)
			}
			b.start()
			foe := mobs.GetInstance(aimOf(&b.companion(1).Character))
			require.NotNil(t, foe)
			b.fight()
			got := 0
			if foe.Character.RT != nil {
				got = foe.Character.RT.Mark
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWarlordBattleCryRalliesTheCompanyForTheFirstTwoRounds(t *testing.T) {
	b := warlordBrawl(t, 35)
	b.start()
	out := b.fight()
	assert.Regexp(t, `Tamsin Reed roars a battle cry, and the company takes it up\.`, out)
	assert.Contains(t, out, "battle cry: +3 Attack for your allies, 2 rounds")
	assert.Equal(t, 3, b.aria.Character.Aura.Attack, "the leader is rallied")
	assert.Equal(t, 3, b.companion(2).Character.Aura.Attack)

	b.toughen()
	b.hardenBandits()
	out = b.fight()
	assert.Equal(t, 3, b.aria.Character.Aura.Attack, "second round")
	assert.NotContains(t, out, "battle cry", "announced once")

	b.toughen()
	b.hardenBandits()
	b.fight()
	assert.Zero(t, b.aria.Character.Aura.Attack, "two rounds only")
}

func TestWarlordBattleCryComesAtRankThirtyFive(t *testing.T) {
	b := warlordBrawl(t, 34)
	b.start()
	out := b.fight()
	assert.NotContains(t, out, "battle cry")
	assert.Zero(t, b.aria.Character.Aura.Attack)
}

func TestWarlordSunderBreaksTheTargetsArmorFromRankFortyFive(t *testing.T) {
	b := warlordBrawl(t, 45)
	b.start()
	foe := mobs.GetInstance(aimOf(&b.companion(1).Character))
	require.NotNil(t, foe)
	out := b.fight()
	assert.Contains(t, out, "sunder: armor broken, 2 rounds")
	assert.True(t, status.Live(&foe.Character, status.ArmorBroken))

	b = warlordBrawl(t, 44)
	b.start()
	foe = mobs.GetInstance(aimOf(&b.companion(1).Character))
	out = b.fight()
	assert.NotContains(t, out, "sunder")
	assert.False(t, status.Live(&foe.Character, status.ArmorBroken))
}

func TestWarlordTackleIsReadyARoundSoonerFromRankForty(t *testing.T) {
	readyAfter := func(level int) int {
		b := warlordBrawl(t, level)
		b.start()
		tackles := 0
		for i := 0; i < 8; i++ {
			out := b.fight()
			if strings.Contains(out, "Tamsin Reed tackles") {
				tackles++
			}
			b.toughen()
			b.hardenBandits()
			for _, m := range b.livingBandits() {
				status.Clear(&m.Character)
			}
		}
		return tackles
	}
	assert.Greater(t, readyAfter(40), readyAfter(39), "a shorter cooldown tackles more often")
}

func TestWarlordRelentlessQuickensItWhenItsFoeStandsUp(t *testing.T) {
	b := warlordBrawl(t, 55)
	b.start()
	foe := mobs.GetInstance(aimOf(&b.companion(1).Character))
	require.NotNil(t, foe)
	b.fight()
	require.True(t, status.Live(&foe.Character, status.KnockedDown))
	told := false
	for i := 0; i < 12 && !told; i++ {
		b.toughen()
		b.hardenBandits()
		out := b.fight()
		told = strings.Contains(out, "relentless: +25 action meter")
	}
	assert.True(t, told, "the Warlord presses the advantage when the foe stands")
}

func TestWarlordsCommandQuickensTheStandingWhenAnAllyFallsOnce(t *testing.T) {
	b := warlordBrawl(t, 60)
	b.start()
	b.fight() // first pass records who is standing
	b.toughen()
	b.hardenBandits()
	b.companion(2).Character.Health = 0
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed barks an order, and the company closes up around the fallen.")
	assert.Contains(t, out, "warlord's command: +25 action meter")

	b.aria.Character.Health = b.aria.Character.HealthMax.Value
	for _, id := range []int{1, 4} {
		b.companion(id).Character.Health = b.companion(id).Character.HealthMax.Value
	}
	b.companion(3).Character.Health = 0
	b.hardenBandits()
	out = b.fight()
	assert.NotContains(t, out, "warlord's command", "once a battle")
}

func TestSecondWindHealsOnceBelowAQuarterAndTheTurnIsStillTaken(t *testing.T) {
	// Four talent slots at level 35; Second Wind is the fourth pick.
	b := warlordBrawl(t, 35, "toughness", "toughness", "keen-edge", "second-wind")
	b.companion(1).Character.SetClassState("paladin", []string{"toughness", "toughness", "keen-edge", "second-wind"})
	b.start()
	tamsin := &b.companion(1).Character
	b.toughen()
	tamsin.Health = tamsin.HealthMax.Value / 5
	before := tamsin.Health
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed catches their second wind.")
	assert.Greater(t, tamsin.Health, before)
	assert.True(t, tamsin.RT.WindUsed)

	tamsin.Health = tamsin.HealthMax.Value / 5
	b.hardenBandits()
	out = b.fight()
	assert.NotContains(t, out, "second wind", "once a battle")
}
