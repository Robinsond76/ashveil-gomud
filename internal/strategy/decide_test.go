package strategy

import "testing"

func known(ids ...string) func(string) bool {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

func TestDecideHealer(t *testing.T) {
	allies := []Ally{{HP: 20, MaxHP: 20}, {HP: 9, MaxHP: 20}, {HP: 4, MaxHP: 20}}
	s := Situation{Role: Healer, Mana: 20, Knows: known("heal", "healall"), Spells: testSpells(), Allies: allies, Foes: 3}

	// Two below half: the group heal.
	if a := Decide(s); a.Kind != HealAll || a.Spell != "healall" {
		t.Errorf("two hurt: %+v", a)
	}
	// One below half: the single heal on the most hurt.
	s.Allies = []Ally{{HP: 20, MaxHP: 20}, {HP: 9, MaxHP: 20}, {HP: 11, MaxHP: 20}}
	if a := Decide(s); a.Kind != Heal || a.Spell != "heal" || a.Ally != 1 {
		t.Errorf("one hurt: %+v", a)
	}
	// Exactly half is not below half.
	s.Allies = []Ally{{HP: 10, MaxHP: 20}}
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("at half: %+v", a)
	}
	// Two hurt, no group heal known: the single heal on the most hurt.
	s.Allies = allies
	s.Knows = known("heal")
	if a := Decide(s); a.Kind != Heal || a.Ally != 2 {
		t.Errorf("no group heal: %+v", a)
	}
	// One hurt, only the group heal known: it heals them too.
	s.Knows = known("healall")
	s.Allies = []Ally{{HP: 2, MaxHP: 20}}
	if a := Decide(s); a.Kind != HealAll {
		t.Errorf("only group heal: %+v", a)
	}
	// Not enough mana: swing.
	s.Mana = 5
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("no mana: %+v", a)
	}
	// A fallen ally isn't healed (aid is another matter).
	s = Situation{Role: Healer, Mana: 20, Knows: known("heal"), Spells: testSpells(), Allies: []Ally{{HP: 0, MaxHP: 20}}}
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("fallen ally: %+v", a)
	}
	// 32d review: a downed player (bleeding out) is healed first.
	s.Allies = []Ally{{HP: 8, MaxHP: 20}, {HP: -3, MaxHP: 20, Downed: true}}
	if a := Decide(s); a.Kind != Heal || a.Ally != 1 {
		t.Errorf("downed player: %+v", a)
	}
}

func TestDecideCaster(t *testing.T) {
	s := Situation{Role: Caster, Mana: 20, Knows: known("mm", "sparks"), Spells: testSpells(), Foes: 3}
	if a := Decide(s); a.Kind != AttackAll || a.Spell != "sparks" {
		t.Errorf("three foes: %+v", a)
	}
	s.Foes = 1
	if a := Decide(s); a.Kind != Attack || a.Spell != "mm" {
		t.Errorf("one foe: %+v", a)
	}
	s.Mana = 5 // mm costs 6
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("no mana: %+v", a)
	}
	s = Situation{Role: Caster, Mana: 20, Knows: known("sparks"), Spells: testSpells(), Foes: 1}
	if a := Decide(s); a.Kind != AttackAll {
		t.Errorf("only the area spell, one foe: %+v", a)
	}
	// Two foes, not enough for the area spell: the single one.
	s = Situation{Role: Caster, Mana: 7, Knows: known("mm", "sparks"), Spells: testSpells(), Foes: 2}
	if a := Decide(s); a.Kind != Attack {
		t.Errorf("area too dear: %+v", a)
	}
	// A spell known but not configured is never cast.
	s = Situation{Role: Caster, Mana: 50, Knows: known("poly"), Spells: testSpells(), Foes: 1}
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("polymorph: %+v", a)
	}
}

func TestDecideFighter(t *testing.T) {
	s := Situation{Role: Fighter, Mana: 50, Knows: known("mm", "heal"), Spells: testSpells(), Foes: 2,
		Allies: []Ally{{HP: 1, MaxHP: 20}}}
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("a fighter swings: %+v", a)
	}
}

func TestSpellFor(t *testing.T) {
	spells := testSpells()
	if sp, ok := SpellFor(spells, UseHeal, known("heal", "healall")); !ok || sp.ID != "heal" {
		t.Errorf("SpellFor heal = %+v %v", sp, ok)
	}
	if _, ok := SpellFor(spells, UseAttack, known("heal")); ok {
		t.Error("no attack spell known")
	}
}

// testSpells is the shipped list with the shipped costs.
func testSpells() []Spell {
	return []Spell{
		{ID: "heal", Use: UseHeal, Cost: 3},
		{ID: "healall", Use: UseHealAll, Cost: 6},
		{ID: "mm", Use: UseAttack, Cost: 6},
		{ID: "sparks", Use: UseAttackAll, Cost: 10},
	}
}

func TestDefaultAutoSpells(t *testing.T) {
	want := map[string]Use{"heal": UseHeal, "healall": UseHealAll, "mm": UseAttack, "sparks": UseAttackAll}
	got := DefaultAutoSpells()
	if len(got) != len(want) {
		t.Fatalf("DefaultAutoSpells = %+v", got)
	}
	for _, sp := range got {
		if want[sp.ID] != sp.Use {
			t.Errorf("%s: %s", sp.ID, sp.Use)
		}
	}
	if u, ok := ParseUse("attack-all"); !ok || u != UseAttackAll {
		t.Error("ParseUse")
	}
}
