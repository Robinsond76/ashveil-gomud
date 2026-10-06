package strategy

import (
	"errors"
	"testing"
)

func TestCastersRule(t *testing.T) {
	if r, ok := ParseRule("casters"); !ok || r != Casters {
		t.Fatalf("casters parses: %v %v", r, ok)
	}
	if r, ok := ParseRule("mages"); !ok || r != Casters {
		t.Errorf("mages alias: %v %v", r, ok)
	}
	if _, ok := ParseRole("casters"); ok {
		t.Error("casters is a rule, not a role")
	}
	listed := false
	for _, r := range Rules {
		listed = listed || r == Casters
	}
	if !listed {
		t.Error("casters is listed")
	}
	fs := []Foe{
		{ID: 1, HP: 10, MaxHP: 10, Row: 0, Col: 0, Reachable: true},
		{ID: 2, HP: 10, MaxHP: 10, Row: 0, Col: 1, Reachable: true, Caster: true},
		{ID: 3, HP: 10, MaxHP: 10, Row: 1, Col: 0, Reachable: true, Caster: true, Chanting: true},
	}
	if got, _ := Pick(Casters, fs, 0, false); got != 3 {
		t.Errorf("chanting first: got %d", got)
	}
	fs[2].Chanting = false
	if got, _ := Pick(Casters, fs, 0, false); got != 2 {
		t.Errorf("then a caster, front first: got %d", got)
	}
	fs[1].Caster, fs[2].Caster = false, false
	// Phase 35d, casters first: with no caster standing the rule falls back
	// to the weakest foe.
	fs[2].HP = 4
	if got, ok := Choose(Casters, fs, 0); !ok || got != 3 {
		t.Errorf("no caster: the weakest, got %d, %v", got, ok)
	}
	if got, _ := Pick(Casters, fs, 0, false); got != 3 {
		t.Errorf("no caster: the weakest, got %d", got)
	}
}

func TestDefaultFocusAtLeaderLevel(t *testing.T) {
	for level, want := range map[int]Rule{1: NoFocus, 9: NoFocus, 10: Weakest, 24: Weakest, 25: Casters, 60: Casters} {
		if got := DefaultFocusAt(level); got != want {
			t.Errorf("DefaultFocusAt(%d) = %q, want %q", level, got, want)
		}
	}
}

func TestParsePatch(t *testing.T) {
	for in, want := range map[string]int{"50": 50, "80": 80, "100%": 100, " 65 ": 65} {
		if got, ok := ParsePatch(in); !ok || got != want {
			t.Errorf("ParsePatch(%q) = %d, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "49", "101", "-1", "many", "80.5"} {
		if _, ok := ParsePatch(in); ok {
			t.Errorf("ParsePatch(%q) accepted", in)
		}
	}
}

func TestParseFocus(t *testing.T) {
	for in, want := range map[string]Rule{
		"none": NoFocus, "off": NoFocus, "leader": Leader, "casters": Casters,
		"nearest": Nearest, "front": Nearest, "weakest": Weakest, "Strongest": Strongest, "wounded": Wounded, "hurt": Wounded,
	} {
		if got, ok := ParseFocus(in); !ok || got != want {
			t.Errorf("ParseFocus(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"assist", "defend", "furthest", "", "bogus"} {
		if _, ok := ParseFocus(in); ok {
			t.Errorf("ParseFocus(%q) should refuse", in)
		}
	}
	if len(FocusRules) != 7 || FocusRules[0] != NoFocus {
		t.Errorf("seven focus values, none first: %v", FocusRules)
	}
}

func TestParseHealing(t *testing.T) {
	for in, want := range map[string]int{"10": 10, "50": 50, "90%": 90, " 70 % ": 70} {
		if got, ok := ParseHealing(in); !ok || got != want {
			t.Errorf("ParseHealing(%q) = %d, %v", in, got, ok)
		}
	}
	for _, in := range []string{"0", "5", "45", "95", "100", "half", ""} {
		if _, ok := ParseHealing(in); ok {
			t.Errorf("ParseHealing(%q) should refuse", in)
		}
	}
}

func TestTacticsResolve(t *testing.T) {
	var zero Tactics
	if !zero.IsZero() {
		t.Error("zero tactics are zero")
	}
	r := zero.Resolve()
	if r.Focus != NoFocus || r.Healing != DefaultHealing || DefaultHealing != 50 {
		t.Errorf("resolved defaults: %+v", r)
	}
	if _, ok := r.FocusRule(); ok {
		t.Error("none is no focus")
	}
	if rule, ok := (Tactics{Focus: Leader}).FocusRule(); !ok || rule != Leader {
		t.Errorf("a focus: %v %v", rule, ok)
	}
}

type fakeTactics struct {
	stored map[int]Tactics
	err    error
}

func (f *fakeTactics) StoredTactics(userID int) Tactics { return f.stored[userID] }
func (f *fakeTactics) SetTactics(userID int, t Tactics) error {
	if f.err != nil {
		return f.err
	}
	f.stored[userID] = t
	return nil
}

func TestTacticsProvider(t *testing.T) {
	SetTacticsProvider(nil)
	if got := TacticsFor(3); got.Focus != NoFocus || got.Healing != 50 {
		t.Errorf("no store: defaults, got %+v", got)
	}
	if err := SaveTactics(3, Tactics{Focus: Leader}); !errors.Is(err, ErrNoTacticsStore) {
		t.Errorf("no store: %v", err)
	}
	f := &fakeTactics{stored: map[int]Tactics{}}
	SetTacticsProvider(f)
	t.Cleanup(func() { SetTacticsProvider(nil) })
	if err := SaveTactics(3, Tactics{Focus: Strongest, Healing: 70}); err != nil {
		t.Fatal(err)
	}
	if got := TacticsFor(3); got.Focus != Strongest || got.Healing != 70 {
		t.Errorf("stored: %+v", got)
	}
}

func TestDecideHealBelow(t *testing.T) {
	s := Situation{Role: Healer, Mana: 20, Knows: known("heal"), Spells: testSpells(), Allies: []Ally{{HP: 45, MaxHP: 100}}, Foes: 1}
	s.HealBelow = 40
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("45%% with a 40 threshold: %+v", a)
	}
	s.HealBelow = 60
	if a := Decide(s); a.Kind != Heal {
		t.Errorf("45%% with a 60 threshold: %+v", a)
	}
	s.HealBelow = 0 // unset: half
	if a := Decide(s); a.Kind != Heal {
		t.Errorf("45%% with the default: %+v", a)
	}
	s.Allies = []Ally{{HP: 50, MaxHP: 100}}
	if a := Decide(s); a.Kind != Swing {
		t.Errorf("exactly half is not below half: %+v", a)
	}
	s.HealBelow = 10
	s.Allies = []Ally{{HP: 0, MaxHP: 100, Downed: true}}
	if a := Decide(s); a.Kind != Heal {
		t.Errorf("a downed player is healed at any threshold: %+v", a)
	}
}

func TestEnemyPick(t *testing.T) {
	company := []Foe{
		{ID: 1, HP: 30, MaxHP: 30, Row: 0, Col: 0, Reachable: true, Leader: true},
		{ID: 2, HP: 12, MaxHP: 40, Row: 0, Col: 1, Reachable: true},
		{ID: 3, HP: 10, MaxHP: 10, Row: 1, Col: 1, Reachable: true, Caster: true},
		{ID: 4, HP: 2, MaxHP: 20, Row: 2, Col: 2, Reachable: false, Chanting: true},
	}
	never := func(n int) int { return n - 1 }
	cases := map[Rule]int{
		Weakest:   3, // 4 is weaker, but out of reach
		Strongest: 1,
		Wounded:   2, // 12/40
		Nearest:   1,
		Furthest:  3,
		Leader:    1,
		Casters:   3, // the chanting one is out of reach: the caster
		Assist:    3, // reads as weakest
		Defend:    3,
		"bogus":   3,
	}
	for rule, want := range cases {
		if got, ok := EnemyPick(rule, company, 0, never); !ok || got != want {
			t.Errorf("EnemyPick(%s) = %d, %v; want %d", rule, got, ok, want)
		}
	}
	// Noise: a roll under it takes a random reachable member.
	rolls := []int{5, 2} // 5 < 20: noisy; then index 2 of the reachable, in formation order
	roll := func(n int) int { r := rolls[0]; rolls = rolls[1:]; return r % n }
	if got, _ := EnemyPick(Strongest, company, 20, roll); got != 3 {
		t.Errorf("noisy pick: got %d, want 3", got)
	}
	rolls = []int{20}
	if got, _ := EnemyPick(Strongest, company, 20, roll); got != 1 {
		t.Errorf("a roll at the noise misses: got %d", got)
	}
	for i := range company {
		company[i].Reachable = false
	}
	if _, ok := EnemyPick(Weakest, company, 0, never); ok {
		t.Error("none reachable: nothing")
	}
}

// Phase 35e: the healers rule, a chanting healer first, then the weakest
// idle healer, then the casters order; reach comes first.
func TestHealersRule(t *testing.T) {
	if r, ok := ParseFocus("healers"); !ok || r != Healers {
		t.Fatalf("healers is a focus, got %q %v", r, ok)
	}
	if r, ok := ParseRule("healer"); !ok || r != Healers {
		t.Errorf("healer reads as the rule, got %q %v", r, ok)
	}
	if _, ok := ParseRole("healers"); ok {
		t.Error("healers is a rule, not a role")
	}
	fs := []Foe{
		{ID: 1, HP: 10, MaxHP: 10, Row: 0, Col: 0, Reachable: true},
		{ID: 2, HP: 9, MaxHP: 10, Row: 0, Col: 1, Reachable: true, Healer: true, Caster: true},
		{ID: 3, HP: 5, MaxHP: 10, Row: 1, Col: 0, Reachable: true, Healer: true, Caster: true},
		{ID: 4, HP: 10, MaxHP: 10, Row: 1, Col: 1, Reachable: true, Caster: true, Chanting: true},
	}
	if got, _ := Pick(Healers, fs, 0, false); got != 3 {
		t.Errorf("the weakest idle healer: got %d", got)
	}
	fs[1].Chanting = true
	if got, _ := Pick(Healers, fs, 0, false); got != 2 {
		t.Errorf("a chanting healer before an idle one: got %d", got)
	}
	fs[1].Chanting = false
	if got, _ := Pick(Healers, fs, 0, false); got == 4 {
		t.Errorf("a chanting non-healer outranks a healer")
	}
	// Reach first: the healers out of reach, the casters order among the rest.
	fs[1].Reachable, fs[2].Reachable = false, false
	if got, _ := Pick(Healers, fs, 0, false); got != 4 {
		t.Errorf("no healer in reach falls to the casters order: got %d", got)
	}
	// No healer at all: the casters order, then the weakest.
	for i := range fs {
		fs[i].Healer, fs[i].Reachable, fs[i].Caster, fs[i].Chanting = false, true, false, false
	}
	if got, _ := Pick(Healers, fs, 0, false); got != 3 {
		t.Errorf("no healer, no caster: the weakest, got %d", got)
	}
}

func TestHealersDefaultFromLevelFive(t *testing.T) {
	f := &fakeTactics{stored: map[int]Tactics{}}
	SetTacticsProvider(f)
	t.Cleanup(func() { SetTacticsProvider(nil) })
	for level, want := range map[int]bool{1: false, 4: false, 5: true, 9: true, 10: true, 30: true} {
		if got := HealersDefault(1, level); got != want {
			t.Errorf("HealersDefault(level %d) = %v, want %v", level, got, want)
		}
	}
	f.stored[1] = Tactics{Focus: NoFocus}
	if HealersDefault(1, 12) {
		t.Error("an explicit none beats the default")
	}
	f.stored[1] = Tactics{Focus: Weakest}
	if HealersDefault(1, 12) {
		t.Error("an explicit focus beats the default")
	}
}
