package wounds

import "testing"

// Phase 35b: after-battle patching heals to the healing threshold with
// heal spells only, most hurt first, and each healer keeps its reserve.
func TestPatchHealsToTheThreshold(t *testing.T) {
	zero := func(int) int { return 0 } // each die rolls 1
	rules := Rules{HealCost: 3, HealDice: [2]int{2, 4}}
	patients := []Patient{
		{Key: "a", Health: 10, Max: 100},
		{Key: "b", Health: 45, Max: 100},
		{Key: "c", Health: 80, Max: 100},
		{Key: "down", Health: 0, Max: 100},
		{Key: "wounded", Health: 10, Max: 100, Wounds: []Wound{{Kind: Fracture, Place: "arm", Points: 60}}},
	}
	healers := []Healer{{Key: "h", Mana: 300, Heal: true, Tend: true, HealBonus: 8}}
	res := Patch(patients, healers, rules, 50, zero)
	got := map[string]int{}
	for _, p := range res.Patients {
		got[p.Key] = p.Health
	}
	// Each heal is 2 + 8 = 10.
	if got["a"] != 50 || got["b"] != 55 || got["c"] != 80 {
		t.Errorf("patched to %v: want a 50, b 55 (one heal past the line), c untouched", got)
	}
	if got["down"] != 0 {
		t.Error("the downed aren't raised")
	}
	if got["wounded"] != 20 {
		t.Errorf("a wound limit of 40 makes the threshold 20, got %d", got["wounded"])
	}
	for _, s := range res.Steps {
		if s.Kind != StepHeal {
			t.Errorf("patching only heals, got %s", s.Kind)
		}
	}
	for _, p := range res.Patients {
		if p.Key == "wounded" && len(p.Wounds) != 1 {
			t.Error("no wound is tended")
		}
	}
}

func TestPatchKeepsEachHealersReserve(t *testing.T) {
	zero := func(int) int { return 0 }
	rules := Rules{HealCost: 3, HealDice: [2]int{2, 4}}
	patients := []Patient{{Key: "a", Health: 1, Max: 200}}
	healers := []Healer{{Key: "x", Mana: 10, Heal: true, Reserve: 4}, {Key: "y", Mana: 9, Heal: true, Reserve: 0}}
	res := Patch(patients, healers, rules, 90, zero)
	left := map[string]int{}
	for _, h := range res.Healers {
		left[h.Key] = h.Mana
	}
	if left["x"] != 4 || left["y"] != 0 {
		t.Errorf("healers ended at %v: x keeps its reserve of 4, y spends all", left)
	}
	if len(res.Steps) != 5 {
		t.Errorf("five heals (x twice, y three times), got %d", len(res.Steps))
	}
	if got := Patch(patients, []Healer{{Key: "t", Mana: 50, Tend: true}}, rules, 90, zero); len(got.Steps) != 0 {
		t.Error("a healer who only tends doesn't patch")
	}
}

func TestHealTarget(t *testing.T) {
	p := Patient{Max: 33}
	for _, c := range []struct{ pct, want int }{{50, 17}, {10, 4}, {90, 30}, {0, 0}, {150, 33}} {
		if got := HealTarget(p, c.pct); got != c.want {
			t.Errorf("HealTarget(33, %d) = %d, want %d", c.pct, got, c.want)
		}
	}
}

// Phase 35b: a crit from a foe 3 or more levels below leaves no wound
// while the target stays at half health or more, else a light one.
func TestEasyCrit(t *testing.T) {
	if EasyFight(10, 12) || !EasyFight(9, 12) || !EasyFight(1, 30) || EasyFight(12, 9) {
		t.Error("easy means the attacker is 3 or more levels below")
	}
	crit := Wound{Kind: Cut, Place: "arm", Points: 6}
	if _, ok := EasyCrit(crit, 12, 50, 100); ok {
		t.Error("at exactly half, no wound")
	}
	w, ok := EasyCrit(crit, 12, 49, 100)
	if !ok || !w.Light || w.Points != 3 || w.Kind != Cut {
		t.Errorf("below half, a light wound of a crushing blow's size: %+v %v", w, ok)
	}
}
