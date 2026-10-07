package orders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/strategy"
)

func TestParseRoundTripsEveryMenuEntry(t *testing.T) {
	for _, text := range []string{
		"ally 50 then heal", "ally 25 then guard", "self 40 then heal", "chanting then break",
		"boss then strongest", "foe healer then break", "first then hold", "foe caster then break",
		"boss then break", "ally 75 then hold",
	} {
		o, err := Parse(splitWords(text))
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		if o.Command() != text {
			t.Errorf("%q came back as %q", text, o.Command())
		}
		if o.Describe() == "" {
			t.Errorf("%q has no description", text)
		}
	}
}

func TestParseRejectsNonsense(t *testing.T) {
	for _, text := range []string{
		"", "ally then heal", "ally 52 then heal", "ally 5 then heal", "ally 95 then heal", "self 50 then guard",
		"chanting then heal", "first then guard", "boss then heal", "foe dragon then break", "foe then break",
		"chanting 50 then break", "ally 50 then dance", "ally 50", "ally 50 then heal break", "wat 50 then heal",
		"first then break",
	} {
		if o, err := Parse(splitWords(text)); err == nil {
			t.Errorf("%q should be refused, got %+v", text, o)
		}
	}
}

func splitWords(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func mustParse(t *testing.T, text string) Order {
	t.Helper()
	o, err := Parse(splitWords(text))
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestEvaluateTakesTheFirstOrderThatHolds(t *testing.T) {
	list := []Order{mustParse(t, "ally 40 then heal"), mustParse(t, "chanting then break")}
	snap := Snapshot{
		Allies: []Ally{{HP: 50, MaxHP: 100, Self: true}, {HP: 60, MaxHP: 100}},
		Foes:   []FoeInfo{{ID: 7, Chanting: true}},
	}
	f, ok := Evaluate(list, snap, nil)
	if !ok || f.Index != 1 || f.Foe != 7 {
		t.Fatalf("the chant order should fire, got %+v %v", f, ok)
	}
	snap.Allies[1].HP = 30
	f, ok = Evaluate(list, snap, nil)
	if !ok || f.Index != 0 || f.Ally != 1 {
		t.Fatalf("the heal order is first and should fire for ally 1, got %+v %v", f, ok)
	}
}

func TestEvaluateNamesTheMostHurtAllyAndSkipsSelfAndTheCovered(t *testing.T) {
	list := []Order{mustParse(t, "ally 50 then heal")}
	snap := Snapshot{Allies: []Ally{
		{HP: 5, MaxHP: 100, Self: true}, // the member itself is not "an ally"
		{HP: 40, MaxHP: 100},
		{HP: 10, MaxHP: 100, Pending: true}, // a heal already covers it
		{HP: 20, MaxHP: 100},
		{HP: 0, MaxHP: 100}, // fallen
	}}
	f, ok := Evaluate(list, snap, nil)
	if !ok || f.Ally != 3 {
		t.Fatalf("want the most hurt uncovered ally (3), got %+v %v", f, ok)
	}
	// A guard order does not skip the covered ally: a heal on its way does not stop a blow.
	f, ok = Evaluate([]Order{mustParse(t, "ally 50 then guard")}, snap, nil)
	if !ok || f.Ally != 2 {
		t.Fatalf("guard should name the most hurt ally even when covered (2), got %+v %v", f, ok)
	}
}

func TestEvaluateSkipsAnOrderThatCannotBeCarriedOut(t *testing.T) {
	list := []Order{mustParse(t, "chanting then break"), mustParse(t, "first then hold")}
	snap := Snapshot{FirstRound: true, Foes: []FoeInfo{{ID: 3, Chanting: true}}}
	f, ok := Evaluate(list, snap, func(f Fire) bool { return f.Order.Do != Break })
	if !ok || f.Index != 1 {
		t.Fatalf("the unusable break should be passed over, got %+v %v", f, ok)
	}
	if _, ok := Evaluate(list, snap, func(Fire) bool { return false }); ok {
		t.Fatal("nothing usable should fire nothing")
	}
}

func TestEveryConditionReadsItsSnapshot(t *testing.T) {
	snap := Snapshot{
		FirstRound: true,
		Allies:     []Ally{{HP: 30, MaxHP: 100, Self: true}, {HP: 90, MaxHP: 100}},
		Foes:       []FoeInfo{{ID: 1}, {ID: 2, Boss: true}, {ID: 3, Caster: true}, {ID: 4, Healer: true}},
	}
	for text, want := range map[string]struct {
		ally, foe int
		fires     bool
	}{
		"self 40 then heal":     {0, 0, true},
		"self 25 then heal":     {0, 0, false},
		"ally 90 then guard":    {0, 0, false}, // 90% is not below 90%
		"ally 95 then guard":    {0, 0, false},
		"boss then strongest":   {-1, 2, true},
		"foe caster then break": {-1, 3, true},
		"foe healer then break": {-1, 4, true},
		"chanting then break":   {-1, 0, false},
		"first then hold":       {-1, 0, true},
	} {
		o, err := Parse(splitWords(text))
		if err != nil {
			if want.fires {
				t.Errorf("%q: %v", text, err)
			}
			continue
		}
		f, ok := Evaluate([]Order{o}, snap, nil)
		if ok != want.fires {
			t.Errorf("%q fires=%v, want %v", text, ok, want.fires)
			continue
		}
		if ok && (f.Ally != want.ally || f.Foe != want.foe) {
			t.Errorf("%q named ally %d foe %d, want %d %d", text, f.Ally, f.Foe, want.ally, want.foe)
		}
	}
	snap.FirstRound = false
	if _, ok := Evaluate([]Order{mustParse(t, "first then hold")}, snap, nil); ok {
		t.Error("first round should not hold later")
	}
}

func TestEveryPresetIsValid(t *testing.T) {
	for _, arch := range []string{"warrior", "cleric", "alchemist", "wizard", "shaman", "witch", "rogue", "ranger", "unknown", ""} {
		list := Preset(arch)
		if len(list) == 0 || len(list) > MaxOrders {
			t.Errorf("%q preset has %d orders", arch, len(list))
		}
		for _, o := range list {
			if err := o.Validate(); err != nil {
				t.Errorf("%q preset %+v: %v", arch, o, err)
			}
		}
	}
}

func casterWith(mana int, known ...string) Caster {
	set := map[string]bool{}
	for _, k := range known {
		set[k] = true
	}
	return Caster{
		Spells: []strategy.Spell{
			{ID: "heal", Use: strategy.UseHeal, Cost: 5}, {ID: "greaterheal", Use: strategy.UseBigHeal, Cost: 12},
			{ID: "mm", Use: strategy.UseAttack, Cost: 4}, {ID: "sparks", Use: strategy.UseAttackAll, Cost: 8},
			{ID: "lightning", Use: strategy.UseStorm, Cost: 15},
		},
		Knows: func(id string) bool { return set[id] },
		Mana:  mana,
	}
}

func TestHealPicksTheHeavyHealOnlyForSomeoneInTrouble(t *testing.T) {
	c := casterWith(30, "heal", "greaterheal")
	if sp, ok := c.Heal(300); !ok || sp.ID != "greaterheal" {
		t.Errorf("30%% health should get the heavy heal, got %v %v", sp, ok)
	}
	if sp, ok := c.Heal(450); !ok || sp.ID != "heal" {
		t.Errorf("45%% health should get the plain heal, got %v %v", sp, ok)
	}
	if _, ok := casterWith(3, "heal").Heal(300); ok {
		t.Error("no mana, no heal")
	}
	if _, ok := casterWith(30, "mm").Heal(300); ok {
		t.Error("no healing spell known, no heal")
	}
}

func TestAttackPrefersTheHeavyBoltAndHonoursSingleTarget(t *testing.T) {
	c := casterWith(40, "mm", "sparks", "lightning")
	if sp, kind, ok := c.Attack(3, false); !ok || sp.ID != "lightning" || kind != strategy.Storm {
		t.Errorf("lightning first, got %v %v %v", sp, kind, ok)
	}
	c = casterWith(40, "mm", "sparks")
	if sp, kind, _ := c.Attack(3, false); sp.ID != "sparks" || kind != strategy.AttackAll {
		t.Errorf("a group spell for three foes, got %v %v", sp, kind)
	}
	if sp, kind, _ := c.Attack(3, true); sp.ID != "mm" || kind != strategy.Attack {
		t.Errorf("a single target takes the single spell, got %v %v", sp, kind)
	}
	if sp, _, _ := c.Attack(1, false); sp.ID != "mm" {
		t.Errorf("one foe takes the single spell, got %v", sp)
	}
	if _, _, ok := casterWith(2, "mm").Attack(1, false); ok {
		t.Error("no mana, no attack")
	}
	if !casterWith(0, "mm").KnowsAttack() || casterWith(40, "heal").KnowsAttack() {
		t.Error("KnowsAttack should read the known spells, not the mana")
	}
}

// Phase 61 review: a foe condition tries each foe that meets it, so a
// chanter out of reach gives way to another chanter in reach.
func TestEvaluateTriesEachFoeThatMeetsTheCondition(t *testing.T) {
	list := []Order{mustParse(t, "chanting then break")}
	snap := Snapshot{Foes: []FoeInfo{{ID: 3, Chanting: true}, {ID: 4}, {ID: 5, Chanting: true}}}
	f, ok := Evaluate(list, snap, func(f Fire) bool { return f.Foe != 3 })
	if !ok || f.Foe != 5 {
		t.Fatalf("the second chanter should be taken, got %+v %v", f, ok)
	}
}
