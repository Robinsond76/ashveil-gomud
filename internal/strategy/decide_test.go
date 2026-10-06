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
	want := map[string]Use{"heal": UseHeal, "healall": UseHealAll, "mm": UseAttack, "sparks": UseAttackAll,
		"binding": UseHex, "slumber": UseHex, "earthbind": UseHex, "frailty": UseHex,
		"leaden": UseHex, "miasma": UseHex, "dread": UseHex, "blight": UseHex, "hex": UseAttack,
		"greaterheal": UseBigHeal, "rejuvenation": UseRejuv, "grove": UseGrove, "siphon": UseSiphon,
		"ward": UseWard, "arcaneward": UseWard, "barkskin": UseBark, "bless": UseBless, "entangle": UseHex, "callhost": UseSummon, "bindfiend": UseSummon, "raisefallen": UseRaise,
		"rain": UseWeather, "chillwind": UseWeather, "callfog": UseWeather, "lightning": UseStorm, "gust": UseAttack, "stoneskin": UseBark, "arcanelance": UseBurst}
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

func TestSummonerCallsItsSummonFirstOnceABattle(t *testing.T) {
	list := []Spell{{ID: "callhost", Use: UseSummon, Cost: 30}, {ID: "heal", Use: UseHeal, Cost: 6}}
	knows := func(id string) bool { return true }
	sit := Situation{Role: Healer, Mana: 100, MaxMana: 100, Knows: knows, Spells: list, Foes: 3,
		Allies: []Ally{{HP: 100, MaxHP: 100}}}
	act := Decide(sit)
	if act.Kind != Summon || act.Spell != "callhost" {
		t.Fatalf("Decide = %+v, want the summon", act)
	}
	sit.Summoned = true
	if act := Decide(sit); act.Kind == Summon {
		t.Fatalf("a summon already called is called again: %+v", act)
	}
	sit.Summoned, sit.Foes = false, 0
	if act := Decide(sit); act.Kind == Summon {
		t.Fatal("no foes, no summon")
	}
}

// Phase 38b review: a summon is for a battle worth it (three or more foes,
// or a boss), and keeps the mana reserve.
func TestSummonerCallsOnlyForABattleWorthIt(t *testing.T) {
	list := []Spell{{ID: "bindfiend", Use: UseSummon, Cost: 30}}
	knows := func(id string) bool { return true }
	sit := Situation{Role: Healer, Mana: 100, MaxMana: 100, Knows: knows, Spells: list, Foes: 2,
		Allies: []Ally{{HP: 100, MaxHP: 100}}}
	if act := Decide(sit); act.Kind == Summon {
		t.Fatal("two ordinary foes are not worth a summon")
	}
	sit.Boss = true
	if act := Decide(sit); act.Kind != Summon {
		t.Fatalf("a boss is worth one: %+v", act)
	}
	sit.Boss, sit.Foes, sit.Reserve = false, 3, 80
	if act := Decide(sit); act.Kind == Summon {
		t.Fatal("the call would break the mana reserve")
	}
	sit.Reserve = 70
	if act := Decide(sit); act.Kind != Summon {
		t.Fatalf("the reserve holds after the call: %+v", act)
	}
}

func shamanSpells() []Spell {
	return []Spell{
		{ID: "rain", Use: UseWeather, Cost: 10},
		{ID: "chillwind", Use: UseWeather, Cost: 8},
		{ID: "callfog", Use: UseWeather, Cost: 6},
		{ID: "lightning", Use: UseStorm, Cost: 12},
		{ID: "gust", Use: UseAttack, Cost: 5},
		{ID: "stoneskin", Use: UseBark, Cost: 8},
	}
}

// Phase 39c: a Shaman calls a weather first when none is up, then Lightning,
// then Gust; Rain waits for Lightning and fog or a chill for foes that
// shoot or cast.
func TestDecideShaman(t *testing.T) {
	s := Situation{Role: Caster, Mana: 40, Knows: known("callfog", "chillwind", "rain", "lightning", "gust"), Spells: shamanSpells(), Foes: 2}
	if a := Decide(s); a.Kind != Weather || a.Spell != "rain" {
		t.Errorf("no weather up: %+v", a)
	}
	s.Weather = "rain"
	if a := Decide(s); a.Kind != Storm || a.Spell != "lightning" {
		t.Errorf("weather up, lightning first: %+v", a)
	}
	s.Mana = 11 // lightning costs 12
	if a := Decide(s); a.Kind != Attack || a.Spell != "gust" {
		t.Errorf("no mana for lightning: %+v", a)
	}
	s = Situation{Role: Caster, Mana: 40, Knows: known("callfog", "gust"), Spells: shamanSpells(), Foes: 1,
		CanWeather: func(id string) bool { return false }}
	if a := Decide(s); a.Kind != Attack || a.Spell != "gust" {
		t.Errorf("no weather worth calling: %+v", a)
	}
	s.CanWeather = func(id string) bool { return id == "callfog" }
	if a := Decide(s); a.Kind != Weather || a.Spell != "callfog" {
		t.Errorf("fog against shooters: %+v", a)
	}
	s.Mana = 5
	if a := Decide(s); a.Kind != Attack {
		t.Errorf("cannot pay for fog: %+v", a)
	}
	// An Earthspeaker turns Stoneskin on an ally before the bolt.
	s = Situation{Role: Caster, Mana: 40, Weather: "fog", Knows: known("stoneskin", "gust"), Spells: shamanSpells(), Foes: 1,
		Allies: []Ally{{HP: 20, MaxHP: 20}}}
	if a := Decide(s); a.Kind == Swing {
		t.Errorf("stoneskin or gust expected: %+v", a)
	}
}

// Phase 38c1 review: an Elder Druid sows a Grove before anyone is in danger
// (two scratched allies without Rejuvenation), and at two hurt allies; a
// Druid without Grove keeps its Barkskin.
func TestElderDruidSowsAGroveEarly(t *testing.T) {
	list := []Spell{{ID: "heal", Use: UseHeal, Cost: 3}, {ID: "rejuvenation", Use: UseRejuv, Cost: 3},
		{ID: "grove", Use: UseGrove, Cost: 8}, {ID: "barkskin", Use: UseBark, Cost: 4}}
	knows := func(id string) bool { return true }
	sit := Situation{Role: Healer, Mana: 100, MaxMana: 100, Knows: knows, Spells: list, Foes: 3,
		Allies: []Ally{{HP: 90, MaxHP: 100}, {HP: 70, MaxHP: 100}, {HP: 80, MaxHP: 100}}}
	if act := Decide(sit); act.Kind != Row || act.Spell != "grove" || act.Ally != 1 {
		t.Fatalf("two scratched allies: %+v, want a Grove on the most hurt", act)
	}
	sit.Allies[1].Rejuv = true
	if act := Decide(sit); act.Spell == "grove" {
		t.Fatalf("only one scratched ally lacks Rejuvenation: %+v", act)
	}
	sit.Allies = []Ally{{HP: 40, MaxHP: 100}, {HP: 45, MaxHP: 100}, {HP: 100, MaxHP: 100}}
	if act := Decide(sit); act.Kind != Row || act.Spell != "grove" {
		t.Fatalf("two hurt allies: %+v, want a Grove", act)
	}
	druid := func(id string) bool { return id != "grove" }
	sit.Knows = druid
	sit.Allies = []Ally{{HP: 90, MaxHP: 100}, {HP: 70, MaxHP: 100}}
	if act := Decide(sit); act.Kind != Buff || act.Spell != "barkskin" {
		t.Fatalf("a Druid without Grove: %+v, want Barkskin", act)
	}
}

// Phase 38c3: a Necromancer raises a fallen foe before anything else, once
// a foe has fallen and the cast leaves its mana reserve.
func TestNecromancerRaisesAFallenFoe(t *testing.T) {
	list := []Spell{{ID: "raisefallen", Use: UseRaise, Cost: 30}, {ID: "mm", Use: UseAttack, Cost: 6}}
	knows := func(id string) bool { return true }
	sit := Situation{Role: Caster, Mana: 100, MaxMana: 100, Knows: knows, Spells: list, Foes: 2,
		Allies: []Ally{{HP: 100, MaxHP: 100}}}
	if act := Decide(sit); act.Kind == Raise {
		t.Fatalf("nothing has fallen: %+v", act)
	}
	sit.CanRaise = true
	if act := Decide(sit); act.Kind != Raise || act.Spell != "raisefallen" {
		t.Fatalf("a foe has fallen: %+v", act)
	}
	sit.Reserve = 80
	if act := Decide(sit); act.Kind == Raise {
		t.Fatalf("the raise would break the mana reserve: %+v", act)
	}
	sit.Reserve, sit.Mana = 0, 20
	if act := Decide(sit); act.Kind == Raise {
		t.Fatalf("not enough mana: %+v", act)
	}
	sit.Mana, sit.Role = 100, Fighter
	if act := Decide(sit); act.Kind == Raise {
		t.Fatalf("a fighter raises nothing: %+v", act)
	}
}

// A Warlock's Life Drain is cast while an ally is hurt enough to want it.
func TestWarlockDrainsWhenAnAllyIsHurt(t *testing.T) {
	list := []Spell{{ID: "siphon", Use: UseSiphon, Cost: 10}, {ID: "mm", Use: UseAttack, Cost: 6}}
	knows := func(id string) bool { return true }
	sit := Situation{Role: Caster, Mana: 100, MaxMana: 100, Knows: knows, Spells: list, Foes: 1,
		Allies: []Ally{{HP: 100, MaxHP: 100}, {HP: 80, MaxHP: 100}}}
	if act := Decide(sit); act.Kind == Drain {
		t.Fatalf("no one is hurt enough: %+v", act)
	}
	sit.Allies[1].HP = 70
	if act := Decide(sit); act.Kind != Drain || act.Spell != "siphon" {
		t.Fatalf("an ally at 70%%: %+v", act)
	}
	sit.Allies[1].Pending = true
	if act := Decide(sit); act.Kind == Drain {
		t.Fatalf("a heal is already coming for them: %+v", act)
	}
}
