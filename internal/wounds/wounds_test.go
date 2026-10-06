package wounds

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func zero(int) int { return 0 }

// top rolls the highest face.
func top(n int) int { return n - 1 }

func TestLimit(t *testing.T) {
	cases := []struct {
		name string
		max  int
		ws   []Wound
		want int
	}{
		{"unwounded", 16, nil, 16},
		{"one wound", 16, []Wound{{Points: 3}}, 13},
		{"light counts", 16, []Wound{{Points: 3}, {Points: 1, Light: true}}, 12},
		{"floored at a quarter", 16, []Wound{{Points: 20}}, 4},
		{"quarter rounds up", 10, []Wound{{Points: 20}}, 3},
		{"at least 1", 1, []Wound{{Points: 5}}, 1},
	}
	for _, c := range cases {
		if got := Limit(c.max, c.ws); got != c.want {
			t.Errorf("%s: Limit = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFromCrit(t *testing.T) {
	cases := []struct {
		sub    items.ItemSubType
		damage int
		kind   Kind
		points int
	}{
		{items.Slashing, 6, Cut, 3},
		{items.Cleaving, 7, Cut, 4},
		{items.Whipping, 1, Cut, 1},
		{items.Claws, 4, Cut, 2},
		{items.Generic, 5, Cut, 3},
		{items.Stabbing, 5, Puncture, 3},
		{items.Shooting, 2, Puncture, 1},
		{items.Bludgeoning, 8, Fracture, 4},
	}
	for _, c := range cases {
		w := FromCrit(c.sub, c.damage, zero)
		if w.Kind != c.kind || w.Points != c.points || w.Light || w.Place == "" {
			t.Errorf("%s %d: got %+v, want %s %d lasting with a place", c.sub, c.damage, w, c.kind, c.points)
		}
	}
}

func TestCrushing(t *testing.T) {
	if _, ok := Crushing(3, 16, zero); ok {
		t.Fatal("3 of 16 is under a quarter and should not wound")
	}
	w, ok := Crushing(4, 16, zero)
	if !ok || !w.Light || w.Points != 1 || w.Kind != Bruise {
		t.Fatalf("4 of 16: got %+v %v, want a 1-point light bruise", w, ok)
	}
	if w, _ := Crushing(9, 16, zero); w.Points != 3 {
		t.Fatalf("9 damage: points %d, want 3", w.Points)
	}
	if _, ok := Crushing(0, 0, zero); ok {
		t.Fatal("no damage never wounds")
	}
}

func TestBled(t *testing.T) {
	w := Bled(3, zero)
	if !w.Light || w.Points != 3 || w.Kind != Cut {
		t.Fatalf("got %+v", w)
	}
}

func TestCloseWorstFirst(t *testing.T) {
	ws := []Wound{{Kind: Cut, Points: 2}, {Kind: Fracture, Points: 5}, {Kind: Cut, Points: 9, Light: true}}
	out, was, closed, ok := Close(ws, 3)
	if !ok || was.Kind != Fracture || closed != 3 || Total(out) != 2+2+9 {
		t.Fatalf("got %+v %+v %d %v", out, was, closed, ok)
	}
	if ws[1].Points != 5 {
		t.Fatal("Close changed its input")
	}
	out, _, closed, _ = Close(out, 10)
	if closed != 2 || len(out) != 2 {
		t.Fatalf("closing more than a wound holds: closed %d, left %+v", closed, out)
	}
	if _, _, _, ok := Close([]Wound{{Points: 4, Light: true}}, 1); ok {
		t.Fatal("light wounds are not closed by treatment")
	}
}

func TestCloseLightAndLasting(t *testing.T) {
	ws := []Wound{{Kind: Cut, Points: 2}, {Kind: Bruise, Points: 1, Light: true}}
	if !HasLight(ws) || HasLight(CloseLight(ws)) || len(CloseLight(ws)) != 1 {
		t.Fatal("CloseLight should keep only the lasting wound")
	}
	if len(Lasting(ws)) != 1 {
		t.Fatal("Lasting should list the lasting wound")
	}
}

func TestTreat(t *testing.T) {
	ws := []Wound{{Kind: Fracture, Points: 5}, {Kind: Cut, Points: 2}}
	out, was, closed, ok := Treat(ws, Bandage)
	if !ok || was.Kind != Cut || closed != 2 || len(out) != 1 {
		t.Fatalf("bandage: %+v %+v %d %v", out, was, closed, ok)
	}
	out, was, closed, ok = Treat(out, Splint)
	if !ok || was.Kind != Fracture || closed != SplintPoints || Total(out) != 1 {
		t.Fatalf("splint: %+v %+v %d %v", out, was, closed, ok)
	}
	if _, _, _, ok := Treat([]Wound{{Kind: Fracture, Points: 2}}, Bandage); ok {
		t.Fatal("a bandage does not set a bone")
	}
}

func TestDescribe(t *testing.T) {
	if got := Describe(Wound{Kind: Fracture, Place: "arm"}); got != "a broken arm" {
		t.Fatal(got)
	}
	if got := Describe(Wound{Kind: Fracture, Place: "ribs"}); got != "cracked ribs" {
		t.Fatal(got)
	}
	if got := Possessive("Tamsin Reed's", Wound{Kind: Fracture, Place: "arm"}); got != "Tamsin Reed's broken arm" {
		t.Fatal(got)
	}
}

func TestPlanHealersFirstThenItems(t *testing.T) {
	patients := []Patient{
		{Key: "leader", Health: 12, Max: 14},
		{Key: "tamsin", Health: 10, Max: 16, Wounds: []Wound{{Kind: Fracture, Place: "arm", Points: 6}}},
	}
	healers := []Healer{{Key: "oswin", Mana: 8, Tend: true, Heal: true}}
	res := Plan(patients, healers, Stock{Bandages: 3, Splints: 1}, DefaultRules, zero)

	if len(res.Steps) == 0 || res.Steps[0].Patient != "tamsin" || res.Steps[0].Kind != StepTend {
		t.Fatalf("the worst hurt is tended first: %+v", res.Steps)
	}
	// oswin tends twice (4 mana each, 2 points each) and is spent.
	if res.Healers[0].Mana != 0 || res.Steps[1].Kind != StepTend {
		t.Fatalf("mana left %d, want 0 after two tends: %+v", res.Healers[0].Mana, res.Steps)
	}
	var tamsin, leader Patient
	for _, p := range res.Patients {
		switch p.Key {
		case "tamsin":
			tamsin = p
		case "leader":
			leader = p
		}
	}
	// the splint closes the rest of the fracture (2 left after the tends).
	if len(tamsin.Wounds) != 0 || res.Used.Splints != 1 {
		t.Fatalf("tamsin %+v, used %+v", tamsin, res.Used)
	}
	if tamsin.Health != 10 {
		t.Fatalf("tamsin health %d, want 10 (no mana left to heal)", tamsin.Health)
	}
	// the leader is above half their limit: no bandage spent on them.
	if leader.Health != 12 || res.Used.Bandages != 0 {
		t.Fatalf("leader %+v, bandages used %d", leader, res.Used.Bandages)
	}
	for _, p := range patients {
		if p.Key == "tamsin" && len(p.Wounds) != 1 {
			t.Fatal("Plan changed its input")
		}
	}
}

func TestPlanHealsToLimitOnly(t *testing.T) {
	patients := []Patient{{Key: "a", Health: 5, Max: 16, Wounds: []Wound{{Kind: Cut, Points: 10}}}}
	healers := []Healer{{Key: "h", Mana: 100, Heal: true}} // can't tend
	res := Plan(patients, healers, Stock{}, DefaultRules, top)
	p := res.Patients[0]
	if p.Health != 6 || p.Limit() != 6 {
		t.Fatalf("health %d limit %d, want both 6", p.Health, p.Limit())
	}
	if res.Healers[0].Mana != 97 {
		t.Fatalf("one heal should reach the limit, mana %d", res.Healers[0].Mana)
	}
}

func TestPlanBandagesPlainDamage(t *testing.T) {
	patients := []Patient{{Key: "a", Health: 2, Max: 14}}
	res := Plan(patients, nil, Stock{Bandages: 5}, DefaultRules, zero)
	p := res.Patients[0]
	// 2 -> 5 -> 8: stops once at half the limit (7) or above.
	if p.Health != 8 || res.Used.Bandages != 2 || res.Stock.Bandages != 3 {
		t.Fatalf("health %d, used %d, left %d", p.Health, res.Used.Bandages, res.Stock.Bandages)
	}
}

func TestPlanSkipsTheDowned(t *testing.T) {
	patients := []Patient{{Key: "a", Health: 0, Max: 14, Wounds: []Wound{{Kind: Cut, Points: 2}}}}
	res := Plan(patients, []Healer{{Key: "h", Mana: 9, Tend: true, Heal: true}}, Stock{Bandages: 2}, DefaultRules, zero)
	if len(res.Steps) != 0 {
		t.Fatalf("a downed member is not treated here: %+v", res.Steps)
	}
}

func TestOrder(t *testing.T) {
	ps := []Patient{
		{Key: "b", Health: 3, Max: 10},
		{Key: "a", Health: 9, Max: 10, Wounds: []Wound{{Points: 1}}},
		{Key: "c", Health: 8, Max: 10},
	}
	Order(ps)
	if ps[0].Key != "a" || ps[1].Key != "b" || ps[2].Key != "c" {
		t.Fatalf("order %s %s %s", ps[0].Key, ps[1].Key, ps[2].Key)
	}
}

// 30g6: a healer's heal adds its HealBonus (the caster's level), as the
// heal spell does in battle, still capped at the wound limit.
func TestPlanHealAddsTheHealersBonus(t *testing.T) {
	patients := []Patient{{Key: "a", Health: 10, Max: 100}}
	res := Plan(patients, []Healer{{Key: "h", Mana: 3, Heal: true, HealBonus: 30}}, Stock{}, DefaultRules, zero)
	if got := res.Patients[0].Health; got != 10+2+30 {
		t.Fatalf("one heal of 2d3 (lowest roll) plus 30, health %d", got)
	}
	res = Plan([]Patient{{Key: "a", Health: 10, Max: 20}}, []Healer{{Key: "h", Mana: 3, Heal: true, HealBonus: 30}}, Stock{}, DefaultRules, zero)
	if got := res.Patients[0].Health; got != 20 {
		t.Fatalf("capped at the limit, health %d", got)
	}
}

func TestBeatenIsALastingWoundOfThePercentage(t *testing.T) {
	w := Beaten(100, 20, func(n int) int { return n - 1 })
	if w.Points != 20 || w.Light || w.Kind != Fracture || w.Place == "" {
		t.Fatalf("Beaten(100, 20) = %+v, want a lasting 20-point fracture", w)
	}
	if got := Beaten(3, 20, nil).Points; got != 1 {
		t.Fatalf("a small max health still leaves 1 point, got %d", got)
	}
	if got := Beaten(40, 10, nil).Kind; got != Cut {
		t.Fatalf("no roll picks a cut, got %s", got)
	}
}
