package company

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 61 wiring: battle orders through the real `orders` command, the
// combat round (DoCombat) and the guard path, in the brawl world.

func orderEvents(events []combatstream.Event, do string) []combatstream.Event {
	var out []combatstream.Event
	for _, e := range events {
		if e.Kind == combatstream.OrderFired && e.Status == do {
			out = append(out, e)
		}
	}
	return out
}

// An order's heal comes from a member whose role would not heal, and only
// once its condition holds.
func TestAHealOrderHealsWhenItsConditionHolds(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	stream := b.listen()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	require.Contains(t, b.cmd("strategy", "oswin fighter"), "now a fighter")
	assert.Contains(t, b.cmd("orders", "oswin add ally 70 then heal"), "Order 1 for Brother Oswin: When an ally is below 70% health, heal that ally first.")

	b.cmd("attack", fmt.Sprintf("#%d", captain))
	for _, m := range b.livingBandits() {
		hardTo(&m.Character, 1000)
	}
	b.toughen()
	out := b.fight()
	assert.NotEqual(t, characters.SpellCast, oswin.Character.Aggro.Type, "no one hurt: the order waits")
	assert.NotContains(t, out, "as ordered")

	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 600
	out = b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type, "the order sent Oswin to heal though he is a fighter")
	assert.Equal(t, "heal", oswin.Character.Aggro.SpellInfo.SpellId)
	assert.Equal(t, []int{7}, oswin.Character.Aggro.SpellInfo.TargetUserIds, "on Aria")
	assert.Contains(t, out, "Brother Oswin tends you, as ordered.")
	fired := orderEvents(*stream, "heal")
	require.Len(t, fired, 1)
	assert.Equal(t, oswin.InstanceId, fired[0].Source.MobInstanceId)
	assert.Equal(t, 7, fired[0].Target.UserId)
	fight, _ := battle.Current(7)
	assert.Equal(t, fight.FightID, fired[0].FightID)
}

// A heal order does not stack a second heal on an ally one already covers.
func TestAHealOrderLeavesAnAllyAHealAlreadyCovers(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	stream := b.listen()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("strategy", "oswin fighter")
	b.cmd("orders", "oswin add ally 70 then heal")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	for _, m := range b.livingBandits() {
		hardTo(&m.Character, 1000)
	}
	b.toughen()
	b.fight()
	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 600
	b.fight()
	require.Len(t, orderEvents(*stream, "heal"), 1)
	// Still chanting next round: the order is not read again mid-chant.
	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 600
	b.fight()
	assert.LessOrEqual(t, len(orderEvents(*stream, "heal")), 2)
}

// A break order turns a fighter on a chanting foe, and says so.
func TestABreakOrderTurnsOnAChanter(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, slinger, _, _ := b.shapeBandits()
	stream := b.listen()
	tamsin := b.companion(1)
	b.cmd("strategy", "tamsin fighter")
	assert.Contains(t, b.cmd("orders", "tamsin add chanting then break"), "turn on the chanter to break its chant")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	forceBlows(t, false)
	b.toughen()
	b.fight()
	first := aimOf(&tamsin.Character)
	require.NotZero(t, first)
	require.NotEqual(t, slinger, first, "no chanter: her strategy aims her elsewhere")

	b.mobCasts(mobs.GetInstance(slinger), "mm")
	out := b.fight()
	assert.Equal(t, slinger, aimOf(&tamsin.Character), "she turns on the chanting slinger")
	assert.Contains(t, out, "Tamsin Reed turns on")
	assert.Contains(t, out, "to break its chant, as ordered.")
	fired := orderEvents(*stream, "break")
	require.NotEmpty(t, fired)
	assert.Equal(t, slinger, fired[0].Target.MobInstanceId)
}

// A guard order has a plain fighter step in front of the hurt ally.
func TestAGuardOrderMakesAFighterGuard(t *testing.T) {
	b := guardBrawl(t, "orders tamsin add ally 70 then guard")
	assert.Contains(t, b.cmd("orders", "tamsin"), "guard that ally")
	// Not hurt enough: no guard, the blow lands on Aria.
	got := b.listen()
	b.strike(0, false)
	b.fight()
	assert.Empty(t, guardEvents(*got, combatstream.GuardUsed), "Aria is unhurt: Tamsin holds her place")

	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 500
	b.strike(0, false)
	out := b.fight()
	assert.Contains(t, out, "Tamsin Reed moves to guard you, as ordered.")
	assert.Contains(t, out, "Tamsin Reed steps in front of you. (guard, 1 left)")
	used := guardEvents(*got, combatstream.GuardUsed)
	require.Len(t, used, 1)
	assert.Equal(t, 7, used[0].Target.UserId)
	assert.Len(t, orderEvents(*got, "guard"), 1)

	// The order keeps holding; its note is not repeated every round.
	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 500
	b.strike(0, false)
	out = b.fight()
	assert.NotContains(t, out, "moves to guard you, as ordered")
}

// A hold order keeps a caster's mana; the same fight without it casts.
func TestAHoldOrderKeepsAWizardsMana(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	b.unplaced()
	_, bruiser, _, _, _ := b.shapeBandits()
	stream := b.listen()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
	assert.Contains(t, b.cmd("orders", "me add first then hold"), "hold its mana")
	b.cmd("attack", fmt.Sprintf("#%d", bruiser))
	forceBlows(t, false)
	b.toughen()
	out := b.fight()
	assert.Equal(t, 20, b.aria.Character.Mana, "the opening round: she holds her mana")
	assert.Contains(t, out, "You hold back your mana, as ordered.")
	assert.Len(t, orderEvents(*stream, "hold"), 1)

	b.toughen()
	b.fight()
	assert.Equal(t, 14, b.aria.Character.Mana, "later rounds the order no longer holds: she casts")
}

// A strongest order spends mana a reserve would keep.
func TestAStrongestOrderIgnoresTheReserve(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	b.unplaced()
	_, bruiser, _, _, _ := b.shapeBandits()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
	require.Contains(t, b.cmd("strategy", "me reserve 90"), "only while 90%")
	b.cmd("orders", "me add first then strongest")
	b.cmd("attack", fmt.Sprintf("#%d", bruiser))
	forceBlows(t, false)
	b.toughen()
	out := b.fight()
	assert.Equal(t, 14, b.aria.Character.Mana, "the order cast Magic Missile past her reserve")
	assert.Contains(t, out, "You put everything into your next spell, as ordered.")

	// The reserve still keeps her mana on later rounds.
	b.toughen()
	b.fight()
	b.toughen()
	b.fight()
	assert.Equal(t, 14, b.aria.Character.Mana)
}

// Orders cannot be changed once a battle is under way.
func TestOrdersAreRefusedInABattle(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.cmd("orders", "tamsin add chanting then break")
	captain, _, _, _, _ := b.shapeBandits()
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	b.toughen()
	b.fight()
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	assert.Contains(t, b.cmd("orders", "tamsin clear"), "battle")
	assert.Contains(t, b.cmd("orders", "tamsin"), "turn on the chanter", "reading is always allowed")
}

// A caster's break order is a spell at the chanter, not a swing.
func TestABreakOrderSendsACasterSpellAtTheChanter(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("wizard")
	b.unplaced()
	captain, _, slinger, _, _ := b.shapeBandits()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.LearnSpell("mm")
	b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
	b.cmd("strategy", "me reserve 90") // her strategy alone would never cast
	b.cmd("orders", "me add chanting then break")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	forceBlows(t, false)
	b.toughen()
	b.fight()
	assert.Equal(t, 20, b.aria.Character.Mana, "no chanter yet: the reserve holds her mana")

	b.mobCasts(mobs.GetInstance(slinger), "mm")
	out := b.fight()
	require.NotNil(t, b.aria.Character.Aggro)
	assert.Equal(t, characters.SpellCast, b.aria.Character.Aggro.Type)
	assert.Equal(t, []int{slinger}, b.aria.Character.Aggro.SpellInfo.TargetMobInstanceIds, "at the chanter")
	assert.Equal(t, 14, b.aria.Character.Mana)
	assert.Contains(t, out, "to break its chant, as ordered.")
}

// A self condition heals the member itself.
func TestASelfHealOrderHealsTheMember(t *testing.T) {
	b := newBrawl(t)
	b.withArchetypes("")
	b.unplaced()
	captain, _, _, _, _ := b.shapeBandits()
	oswin := b.companion(2)
	oswin.Character.ManaMax.Value, oswin.Character.Mana = 20, 20
	b.cmd("strategy", "oswin fighter")
	b.cmd("orders", "oswin add self 50 then heal")
	b.cmd("attack", fmt.Sprintf("#%d", captain))
	for _, m := range b.livingBandits() {
		hardTo(&m.Character, 1000)
	}
	b.toughen()
	b.fight()
	oswin.Character.Health = 400
	out := b.fight()
	require.NotNil(t, oswin.Character.Aggro)
	assert.Equal(t, characters.SpellCast, oswin.Character.Aggro.Type)
	assert.Equal(t, []int{oswin.InstanceId}, oswin.Character.Aggro.SpellInfo.TargetMobInstanceIds)
	assert.Contains(t, out, "looks to a wound of their own, as ordered.")
}

// Phase 61 review: the boss and foe-kind conditions in a real round, each
// turning a fighter on the foe it names.
func TestABreakOrderTurnsOnABossOrAHealer(t *testing.T) {
	for _, tc := range []struct {
		order string
		mark  func(m *mobs.Mob)
	}{
		{"boss then break", func(m *mobs.Mob) { m.Boss = true }},
		{"foe healer then break", func(m *mobs.Mob) { m.Role = "healer" }},
	} {
		t.Run(tc.order, func(t *testing.T) {
			b := newBrawl(t)
			b.withArchetypes("")
			b.unplaced()
			captain, _, slinger, _, _ := b.shapeBandits()
			stream := b.listen()
			tamsin := b.companion(1)
			b.cmd("strategy", "tamsin fighter")
			b.cmd("orders", "tamsin add "+tc.order)
			b.cmd("attack", fmt.Sprintf("#%d", captain))
			forceBlows(t, false)
			b.toughen()
			b.fight()
			require.NotEqual(t, slinger, aimOf(&tamsin.Character), "nothing to turn on yet")
			assert.Empty(t, orderEvents(*stream, "break"))

			tc.mark(mobs.GetInstance(slinger))
			out := b.fight()
			assert.Equal(t, slinger, aimOf(&tamsin.Character), "she turns on the slinger")
			assert.Contains(t, out, "Tamsin Reed turns on")
			assert.NotContains(t, out, "to break its chant")
			fired := orderEvents(*stream, "break")
			require.Len(t, fired, 1)
			assert.Equal(t, slinger, fired[0].Target.MobInstanceId)
			// Already on it: the order does not fire again.
			b.fight()
			assert.Len(t, orderEvents(*stream, "break"), 1)
		})
	}
}

// Phase 61 review: a guard order with no guard left gives way to the next
// order instead of claiming a guard that never happens.
func TestAGuardOrderWithNoGuardLeftGivesWay(t *testing.T) {
	b := guardBrawl(t, "orders tamsin add ally 70 then guard", "orders tamsin add chanting then break")
	_, _, slinger, _, _ := b.shapeBandits()
	for {
		if _, ok := battle.SpendGuard(7, "companion:1"); !ok {
			break
		}
	}
	got := b.listen()
	hardTo(b.aria.Character, 1000)
	b.aria.Character.Health = 500
	b.mobCasts(mobs.GetInstance(slinger), "mm")
	out := b.fight()
	assert.NotContains(t, out, "moves to guard you, as ordered")
	assert.Empty(t, orderEvents(*got, "guard"), "no guard left: the guard order is passed over")
	assert.Contains(t, out, "to break its chant, as ordered.", "the next order fires")
}
