package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbilityUnlocks(t *testing.T) {
	assert.Equal(t, []Ability{Tackle}, CompanionAbilities("Warrior"))
	assert.Equal(t, []Ability{OpeningStrike}, CompanionAbilities("rogue"))
	assert.Equal(t, []Ability{AimedShot}, CompanionAbilities("ranger"))
	assert.Empty(t, CompanionAbilities("cleric"))
	assert.Empty(t, CompanionAbilities(""))

	levels := map[string]int{"brawling": 1, "track": 0}
	assert.Equal(t, []Ability{Tackle}, PlayerAbilities(func(s string) int { return levels[s] }))
	assert.Empty(t, PlayerAbilities(nil))
	assert.Equal(t, []string{"Tackle", "Aimed Shot"}, Names([]Ability{Tackle, AimedShot, "nonsense"}))
}

func TestDecideAbility(t *testing.T) {
	ready := func(Ability) bool { return true }
	cases := []struct {
		name string
		s    AbilitySituation
		want Ability
	}{
		{"tackle a standing foe", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Weapon: Melee, Close: true}, Tackle},
		{"tackle with fists", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Weapon: Unarmed, Close: true}, Tackle},
		{"no tackle at a spear's length", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Weapon: Melee}, ""},
		{"no tackle with a bow", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Weapon: Shooting, Close: true}, ""},
		{"no tackle on a downed foe", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Close: true, FoeDown: true}, ""},
		{"no tackle on a stunned foe", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Close: true, FoeStunned: true}, ""},
		{"no tackle on cooldown", AbilitySituation{Known: []Ability{Tackle}, Ready: func(Ability) bool { return false }, Close: true}, ""},
		{"abilities off", AbilitySituation{Known: []Ability{Tackle}, Ready: ready, Close: true, Off: true}, ""},
		{"opening on a downed foe", AbilitySituation{Known: []Ability{OpeningStrike}, Ready: ready, Backstab: true, FoeDown: true}, OpeningStrike},
		{"opening on an exposed foe", AbilitySituation{Known: []Ability{OpeningStrike}, Ready: ready, Backstab: true, FoeExposed: true}, OpeningStrike},
		{"opening on a staggered foe", AbilitySituation{Known: []Ability{OpeningStrike}, Ready: ready, Backstab: true, FoeStaggered: true}, OpeningStrike},
		{"no opening on a steady foe", AbilitySituation{Known: []Ability{OpeningStrike}, Ready: ready, Backstab: true}, ""},
		{"no opening with a club", AbilitySituation{Known: []Ability{OpeningStrike}, Ready: ready, FoeDown: true}, ""},
		{"aimed shot", AbilitySituation{Known: []Ability{AimedShot}, Ready: ready, Weapon: Shooting}, AimedShot},
		{"no aimed shot at an exposed foe", AbilitySituation{Known: []Ability{AimedShot}, Ready: ready, Weapon: Shooting, FoeExposed: true}, ""},
		{"no aimed shot without a bow", AbilitySituation{Known: []Ability{AimedShot}, Ready: ready, Weapon: Melee}, ""},
		{"first that holds", AbilitySituation{Known: []Ability{Tackle, OpeningStrike}, Ready: ready, Backstab: true, FoeDown: true}, OpeningStrike},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := DecideAbility(c.s)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.want != "", ok)
		})
	}
}

func TestTackleChanceAndReserveParse(t *testing.T) {
	assert.Equal(t, 20, TackleChance(0, 50))
	assert.Equal(t, 80, TackleChance(90, 0))
	assert.Equal(t, 35, TackleChance(20, 5))

	for in, want := range map[string]int{"0": 0, "30": 30, "90%": 90} {
		got, ok := ParseReserve(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"-1", "91", "lots", ""} {
		_, ok := ParseReserve(bad)
		assert.False(t, ok, bad)
	}
}

func TestDecideHonoursPendingHealsAndReserve(t *testing.T) {
	knows := func(string) bool { return true }
	spells := []Spell{{ID: "heal", Use: UseHeal, Cost: 5}, {ID: "healall", Use: UseHealAll, Cost: 10}, {ID: "mm", Use: UseAttack, Cost: 6}}

	// The most hurt is already covered: the next most hurt is healed.
	s := Situation{Role: Healer, Mana: 50, Knows: knows, Spells: spells, Allies: []Ally{
		{HP: 10, MaxHP: 100, Pending: true}, {HP: 30, MaxHP: 100}, {HP: 90, MaxHP: 100}}}
	assert.Equal(t, Action{Kind: Heal, Spell: "heal", Ally: 1}, Decide(s))

	// Two hurt but one covered: a single heal, not the group heal.
	s.Allies[2].HP = 20
	s.Allies[1].Pending = true
	assert.Equal(t, Action{Kind: Heal, Spell: "heal", Ally: 2}, Decide(s))

	// Everyone covered: swing.
	s.Allies[2].Pending = true
	assert.Equal(t, Action{Kind: Swing}, Decide(s))

	// The reserve holds an attack spell, not a heal.
	c := Situation{Role: Caster, Mana: 20, MaxMana: 40, Reserve: 50, Knows: knows, Spells: spells, Foes: 1}
	assert.Equal(t, Action{Kind: Swing}, Decide(c), "20-6 leaves 14, under half of 40")
	c.Mana = 26
	assert.Equal(t, Action{Kind: Attack, Spell: "mm"}, Decide(c), "26-6 leaves exactly half")
	h := Situation{Role: Healer, Mana: 6, MaxMana: 40, Reserve: 90, Knows: knows, Spells: spells, Allies: []Ally{{HP: 1, MaxHP: 100}}}
	assert.Equal(t, Action{Kind: Heal, Spell: "heal", Ally: 0}, Decide(h))
}

func TestStrategyIsZeroCountsTheNewFields(t *testing.T) {
	assert.False(t, Strategy{NoAbilities: true}.IsZero())
	assert.False(t, Strategy{Reserve: 10}.IsZero())
	r := Strategy{NoAbilities: true, Reserve: 20}.Resolve("wizard")
	assert.True(t, r.NoAbilities)
	assert.Equal(t, 20, r.Reserve)
}
