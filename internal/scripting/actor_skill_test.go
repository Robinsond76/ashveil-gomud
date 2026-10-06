package scripting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/stretchr/testify/assert"
)

// TestSpellAndHealFactors (Phase 35a2): a damage spell scales 0.5–1.5 by
// the caster's Attack against the target's Evasion, and a heal by the
// healer's healing gear (a holy symbol's +5%).
func TestSpellAndHealFactors(t *testing.T) {
	g := configs.GetGamePlayConfig()
	g.Combat.SkillEdgeSpan = 20
	g.Combat.DefaultAttackRate, g.Combat.DefaultEvasionRate = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	at := func(level int) ScriptActor {
		c := characters.New()
		c.Level = level
		return ScriptActor{characterRecord: c}
	}
	for _, tc := range []struct {
		caster, target int
		want           float64
	}{{20, 20, 1}, {30, 20, 1.25}, {20, 30, 0.75}, {60, 20, 1.5}, {1, 60, 0.5}} {
		assert.InDelta(t, tc.want, at(tc.caster).SpellFactor(at(tc.target)), 1e-9, "%d against %d", tc.caster, tc.target)
	}
	assert.Equal(t, 1.0, ScriptActor{}.SpellFactor(at(10)), "no record: listed damage")

	const symbolID = 99801
	items.SetTestItemSpec(&items.ItemSpec{ItemId: symbolID, Name: "test symbol", Type: items.Offhand, Subtype: items.Wearable, StatMods: map[string]int{"healing": 5}})
	t.Cleanup(func() { items.RemoveTestItemSpec(symbolID) })
	healer := at(10)
	assert.Equal(t, 1.0, healer.HealFactor())
	healer.characterRecord.Equipment.Offhand = items.New(symbolID)
	healer.characterRecord.Validate(true)
	assert.InDelta(t, 1.05, healer.HealFactor(), 1e-9)
}

// Phase 39c: a fogbound caster's spells hit 10% weaker.
func TestSpellFactorUnderFog(t *testing.T) {
	g := configs.GetGamePlayConfig()
	g.Combat.SkillEdgeSpan = 20
	g.Combat.DefaultAttackRate, g.Combat.DefaultEvasionRate = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	mk := func() ScriptActor {
		c := characters.New()
		c.Level = 20
		return ScriptActor{characterRecord: c}
	}
	caster, target := mk(), mk()
	assert.InDelta(t, 1.0, caster.SpellFactor(target), 1e-9)
	caster.characterRecord.Buffs.List = []*buffs.Buff{{BuffId: status.Fogbound, TriggersLeft: 3}}
	assert.InDelta(t, 0.9, caster.SpellFactor(target), 1e-9)
}

// Phase 38e: runed stone takes spells hard.
func TestSpellFactorIsRaisedAgainstAStoneGolem(t *testing.T) {
	g := configs.GetGamePlayConfig()
	g.Combat.SkillEdgeSpan = 20
	g.Combat.DefaultAttackRate, g.Combat.DefaultEvasionRate = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	caster := characters.New()
	caster.Level = 20
	golem := characters.New()
	golem.Level = 20
	person := characters.New()
	person.Level = 20
	golem.HPArchetype = "stone-golem"
	assert.InDelta(t, 1.0, ScriptActor{characterRecord: caster}.SpellFactor(ScriptActor{characterRecord: person}), 1e-9)
	assert.InDelta(t, 1.25, ScriptActor{characterRecord: caster}.SpellFactor(ScriptActor{characterRecord: golem}), 1e-9)
}

// Phase 50 review: a caster's and target's battle condition reach a spell
// through the factor the spell scripts read.
func TestSpellFactorCarriesTheBattleCondition(t *testing.T) {
	g := configs.GetGamePlayConfig()
	g.Combat.SkillEdgeSpan = 20
	g.Combat.DefaultAttackRate, g.Combat.DefaultEvasionRate = 1, 1
	t.Cleanup(configs.SetTestGamePlayConfig(g))
	mk := func(damage, guard int) ScriptActor {
		c := characters.New()
		c.Level = 20
		c.RTState().FareDamage, c.RTState().FareGuard = damage, guard
		return ScriptActor{characterRecord: c}
	}
	assert.InDelta(t, 0.9, mk(-10, 0).SpellFactor(mk(0, 0)), 1e-9, "a starving caster")
	assert.InDelta(t, 1.1, mk(0, 0).SpellFactor(mk(0, -10)), 1e-9, "a parched target")
	assert.InDelta(t, 0.9, mk(0, 0).SpellFactor(mk(0, 10)), 1e-9, "a Hearty target")
}
