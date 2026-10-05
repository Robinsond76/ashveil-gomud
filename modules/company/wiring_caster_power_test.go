package company

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatstream"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hardSpell makes a spell nearly impossible to cast by the roll for the
// test: difficulty 100.
func hardSpell(t *testing.T, id string) {
	t.Helper()
	sp := spells.GetSpell(id)
	require.NotNil(t, sp)
	was := sp.Difficulty
	sp.Difficulty = 100
	t.Cleanup(func() { sp.Difficulty = was })
}

// outcomes counts a caster's finished casts by outcome.
func outcomes(stream []combatstream.Event, key string) map[string]int {
	out := map[string]int{}
	for _, e := range castsBy(stream, combatstream.CastComplete, key) {
		out[e.Outcome]++
	}
	return out
}

// Phase 35b: a spell a character owns never fizzles in a battle, for the
// player and for a companion, however hard it is.
func TestOwnedSpellsNeverFizzleInBattle(t *testing.T) {
	b := guardBrawl(t)
	forceBlows(t, false) // nobody is struck, so no chant breaks
	noCounters(t)
	hardSpell(t, "heal")
	stream := b.listen()
	oswin := b.companion(2)
	_, inBattle := battle.Current(7)
	require.True(t, inBattle)
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.SpellBook["heal"] = 1
	require.Less(t, b.aria.Character.GetBaseCastSuccessChance("heal"), 10, "the roll would almost always fail")
	for round := 0; round < 30; round++ {
		b.toughen()
		b.hold(nil)
		b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 100, 100
		if oswin.Character.Aggro == nil || oswin.Character.Aggro.Type != characters.SpellCast {
			b.mobCasts(oswin, "heal aria")
		}
		if b.aria.Character.Aggro == nil || b.aria.Character.Aggro.Type != characters.SpellCast {
			b.aria.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
		}
		b.fight()
	}
	for _, who := range []string{key(oswin), "u:7"} {
		got := outcomes(*stream, who)
		assert.Zero(t, got[combatstream.OutcomeFizzled], "%s never fizzles in battle: %v", who, got)
		assert.GreaterOrEqual(t, got[combatstream.OutcomeCast], 5, "%s casts: %v", who, got)
	}
}

// Phase 35b: out of a battle, the roll still decides.
func TestSpellsStillFizzleOutOfBattle(t *testing.T) {
	b := newBrawl(t)
	hardSpell(t, "heal")
	stream := b.listen()
	b.aria.Character.SetSkill("cast", 1)
	b.aria.Character.SpellBook["heal"] = 1
	for round := 0; round < 40; round++ {
		b.aria.Character.ManaMax.Value, b.aria.Character.Mana = 100, 100
		if b.aria.Character.Aggro == nil {
			b.aria.Character.SetCast(1, characters.SpellAggroInfo{SpellId: "heal", TargetUserIds: []int{7}})
		}
		b.fight()
		_, inBattle := battle.Current(7)
		require.False(t, inBattle)
	}
	got := outcomes(*stream, "u:7")
	assert.Positive(t, got[combatstream.OutcomeFizzled], "a hard spell fizzles out of battle: %v", got)
}
