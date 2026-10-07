package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 82b: speed-ordered turns through the real round. newBrawl pins
// every tempo at one, so raw Speed decides the order.

// firstSwing is the sequence number of a fighter's first attack event.
func firstSwing(stream []combatstream.Event, source string) (uint64, bool) {
	for _, e := range stream {
		if e.Kind == combatstream.Attack && e.Source.Name == source {
			return e.Seq, true
		}
	}
	return 0, false
}

func TestTurnOrderAFastFoeStrikesBeforeASlowCompany(t *testing.T) {
	b := newBrawl(t)
	b.toughen()
	b.hold(nil)
	forceBlows(t, true)
	b.aimAt("bandit captain")
	captain := b.captain()
	captain.Character.Stats.Speed.ValueAdj = 60
	for _, m := range b.livingBandits() {
		if m != captain {
			m.Character.Stats.Speed.ValueAdj = -20
		}
	}
	b.aria.Character.Stats.Speed.ValueAdj = 0
	for id := 1; id <= 4; id++ {
		b.companion(id).Character.Stats.Speed.ValueAdj = -10
	}
	seen := b.listen()
	b.fight()
	captainSwing, ok := firstSwing(*seen, "bandit captain")
	require.True(t, ok, "the captain swung")
	for _, who := range []string{"Aria", "Tamsin Reed", "Brother Oswin", "Garrick Vane", "Ysolde"} {
		if seq, ok := firstSwing(*seen, who); ok {
			assert.Less(t, captainSwing, seq, "%s acts after the faster captain", who)
		}
	}
	// The slowest fighters of all (the other bandits) come after the company.
	for _, m := range b.livingBandits() {
		if m == captain {
			continue
		}
		if seq, ok := firstSwing(*seen, m.Character.Name); ok {
			if aria, ok := firstSwing(*seen, "Aria"); ok {
				assert.Greater(t, seq, aria, "%s is slower than Aria", m.Character.Name)
			}
		}
	}
}

func TestTurnOrderAKillBeforeTheVictimsTurnDeniesItsBlow(t *testing.T) {
	b := newBrawl(t)
	b.toughen()
	b.hold(nil)
	forceBlows(t, true)
	b.aimAt("bandit captain")
	captain := b.captain()
	captain.Character.Health = 1
	captain.Character.SetAggro(7, 0, characters.DefaultAttack, 0)
	b.actsFirst(b.aria.Character)
	b.aria.Character.SetAggro(0, captain.InstanceId, characters.DefaultAttack)
	seen := b.listen()
	b.fight()
	assert.Empty(t, swingsBy(*seen, "bandit captain"), "felled before its turn, the captain never swings")
	assert.NotEmpty(t, swingsBy(*seen, "Aria"))
}

func TestTurnOrderIsListedForTheBattleDataInSlotOrder(t *testing.T) {
	b := newBrawl(t)
	b.toughen()
	b.hold(nil)
	b.aimAt("bandit captain")
	seen := b.listen()
	b.fight()
	order := hooks.RoundTurnOrder(7)
	require.NotEmpty(t, order)
	// Every fighter in the battle has one slot (tempo one): Aria, four
	// companions and the living bandits.
	assert.Len(t, order, 5+len(b.livingBandits()))
	slotOf := map[int]int{}
	for i, s := range order {
		assert.Equal(t, i+1, s.Slot)
		if s.UserId > 0 {
			slotOf[-s.UserId] = s.Slot
		} else {
			slotOf[s.MobInstanceId] = s.Slot
		}
	}
	// Attack events carry the slot of the fighter acting, and the slots
	// rise with the round's sequence.
	last := 0
	for _, e := range *seen {
		if e.Kind != combatstream.Attack {
			continue
		}
		id := e.Source.MobInstanceId
		if e.Source.UserId > 0 {
			id = -e.Source.UserId
		}
		assert.Equal(t, slotOf[id], e.Slot, "%s's blow carries its slot", e.Source.Name)
		assert.GreaterOrEqual(t, e.Slot, last, "slots rise through the round")
		last = e.Slot
	}
	assert.Zero(t, hooks.CurrentTurnSlot(), "no slot between rounds")
	assert.Empty(t, hooks.RoundTurnOrder(99), "a player in no fight has no order")
}

// Phase 82b: a chant is released at its caster's turn, so a faster foe's
// blow in the same round now breaks it before it lands. `help tempo` makes
// that claim; before this phase the company always acted first and a chant
// due this round could only be broken in an earlier one.
func TestTurnOrderAFasterFoeBreaksAChantBeforeItsRelease(t *testing.T) {
	for _, tc := range []struct {
		name     string
		foeFirst bool
	}{
		{"a faster foe breaks it", true},
		{"a faster caster lands it", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newBrawl(t)
			b.withArchetypes("wizard")
			b.unplaced()
			b.shapeBandits()
			forceBlows(t, true)
			noCounters(t)
			breakDice(t, 0) // Phase 30d1b: the blow breaks the chant
			stream := b.listen()
			b.aria.Character.SetSkill("cast", 1)
			b.aria.Character.LearnSpell("mm")
			b.aimAt("bandit captain")
			captain := b.captain()
			b.toughen()
			b.hold(nil)
			b.strike(0, false) // the captain's is the round's only blow, on Aria
			if tc.foeFirst {
				b.actsFirst(&captain.Character)
			} else {
				b.actsFirst(b.aria.Character)
			}

			// Her chant is due this round: it is released at her turn.
			b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 20, 20
			b.aria.Character.SetCast(0, characters.SpellAggroInfo{SpellId: "mm", TargetMobInstanceIds: []int{captain.InstanceId}})
			require.NotNil(t, b.aria.Character.Aggro)
			b.aria.Character.Aggro.RoundsWaiting = 0
			out := b.fight()

			broken := interruptsOf(*stream, "u:7")
			var landed []combatstream.Event
			for _, e := range castsBy(*stream, combatstream.CastComplete, "u:7") {
				if e.Outcome != combatstream.OutcomeInterrupted {
					landed = append(landed, e)
				}
			}
			if tc.foeFirst {
				require.Len(t, broken, 1, "the faster captain broke her chant:\n%s", out)
				assert.Equal(t, "Magic Missile", broken[0].Status)
				assert.Empty(t, landed, "the missile never left her hands")
				return
			}
			assert.Empty(t, broken, "she released it before his blow:\n%s", out)
			require.Len(t, landed, 1, "the missile landed:\n%s", out)
		})
	}
}
