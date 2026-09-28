package strategy

import "testing"

func TestParseRoleAndRule(t *testing.T) {
	for in, want := range map[string]Role{"fighter": Fighter, "Healer": Healer, "caster": Caster, "fight": Fighter, "heal": Healer, "cast": Caster} {
		if got, ok := ParseRole(in); !ok || got != want {
			t.Errorf("ParseRole(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := ParseRole("bard"); ok {
		t.Error("bard is not a role")
	}
	for _, r := range Rules {
		if got, ok := ParseRule(string(r)); !ok || got != r {
			t.Errorf("ParseRule(%q) = %q, %v", r, got, ok)
		}
	}
	for in, want := range map[string]Rule{"weak": Weakest, "strong": Strongest, "front": Nearest, "back": Furthest, "focus": Assist, "protect": Defend} {
		if got, ok := ParseRule(in); !ok || got != want {
			t.Errorf("ParseRule(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := ParseRule("best"); ok {
		t.Error("best is not a rule")
	}
}

func TestDefaults(t *testing.T) {
	cases := map[string]Role{"cleric": Healer, "wizard": Caster, "warrior": Fighter, "rogue": Fighter, "ranger": Fighter, "": Fighter}
	for arch, role := range cases {
		s := Default(arch)
		if s.Role != role || s.Rule != Weakest {
			t.Errorf("Default(%q) = %+v", arch, s)
		}
	}
	// Blank fields fall back to the archetype's default; set ones stay.
	got := Strategy{Rule: Leader}.Resolve("cleric")
	if got.Role != Healer || got.Rule != Leader {
		t.Errorf("Resolve = %+v", got)
	}
	if !(Strategy{}).IsZero() || (Strategy{Role: Caster}).IsZero() {
		t.Error("IsZero")
	}
}

func TestValidFor(t *testing.T) {
	if err := (Strategy{Rule: Assist}).ValidFor(true); err == nil {
		t.Error("the player can't assist themselves")
	}
	if err := (Strategy{Rule: Assist}).ValidFor(false); err != nil {
		t.Errorf("a companion may assist: %v", err)
	}
}

// A 3-wide formation: row 0 is the front.
func foes() []Foe {
	return []Foe{
		{ID: 1, HP: 30, MaxHP: 30, Row: 0, Col: 0, Reachable: true, Leader: true, StrikesPct: -1},
		{ID: 2, HP: 12, MaxHP: 20, Row: 0, Col: 1, Reachable: true, StrikesPct: 90},
		{ID: 3, HP: 8, MaxHP: 8, Row: 1, Col: 0, Reachable: false, StrikesPct: 40},
		{ID: 4, HP: 3, MaxHP: 10, Row: 2, Col: 2, Reachable: false, StrikesPct: -1},
	}
}

func TestPickRules(t *testing.T) {
	cases := []struct {
		rule        Rule
		assist      int
		ignoreReach bool
		want        int
	}{
		{Weakest, 0, false, 2},   // 3 and 4 are weaker, but out of reach
		{Weakest, 0, true, 4},    // a spell reaches anyone
		{Strongest, 0, false, 1}, // most health left
		{Wounded, 0, false, 2},   // 12/20 is lower than 30/30
		{Wounded, 0, true, 4},    // 3/10
		{Nearest, 0, false, 1},   // front row, left first
		{Furthest, 0, false, 1},  // both reachable foes are in the front row: left first
		{Furthest, 0, true, 4},   // the back row
		{Leader, 0, false, 1},
		{Assist, 2, false, 2},
		{Assist, 3, false, 1},   // the player's target is out of reach: the nearest
		{Assist, 3, true, 3},    // a spell reaches it
		{Defend, 0, false, 2},   // the only reachable foe striking one of us
		{Defend, 0, true, 3},    // it strikes the one at 40%
		{Leader, 0, true, 1},
	}
	for _, c := range cases {
		got, ok := Pick(c.rule, foes(), c.assist, c.ignoreReach)
		if !ok || got != c.want {
			t.Errorf("Pick(%s, assist %d, spell %v) = %d, %v; want %d", c.rule, c.assist, c.ignoreReach, got, ok, c.want)
		}
	}
}

func TestPickFallbacks(t *testing.T) {
	fs := foes()
	fs[0].Leader, fs[3].Leader = false, true // the leader is at the back
	if got, _ := Pick(Leader, fs, 0, false); got != 1 {
		t.Errorf("leader out of reach: got %d, want the nearest (1)", got)
	}
	// Nobody strikes us: defend takes the nearest.
	for i := range fs {
		fs[i].StrikesPct = -1
	}
	if got, _ := Pick(Defend, fs, 0, false); got != 1 {
		t.Errorf("defend with no one striking: got %d", got)
	}
	// None in reach: the front-most.
	for i := range fs {
		fs[i].Reachable = false
	}
	if got, ok := Pick(Weakest, fs, 0, false); !ok || got != 1 {
		t.Errorf("none reachable: got %d, %v", got, ok)
	}
	if _, ok := Pick(Weakest, nil, 0, false); ok {
		t.Error("no foes: nothing to pick")
	}
}

func TestPickTiesByFormation(t *testing.T) {
	fs := []Foe{
		{ID: 7, HP: 5, MaxHP: 10, Row: 1, Col: 0, Reachable: true, StrikesPct: -1},
		{ID: 5, HP: 5, MaxHP: 10, Row: 0, Col: 2, Reachable: true, StrikesPct: -1},
		{ID: 6, HP: 5, MaxHP: 10, Row: 0, Col: 1, Reachable: true, StrikesPct: -1},
	}
	for _, r := range []Rule{Weakest, Strongest, Wounded, Nearest} {
		if got, _ := Pick(r, fs, 0, false); got != 6 {
			t.Errorf("%s tie: got %d, want 6 (front row, left)", r, got)
		}
	}
	if got, _ := Pick(Furthest, fs, 0, false); got != 7 {
		t.Errorf("furthest: got %d", got)
	}
}

func TestReaimsEachRound(t *testing.T) {
	if !Assist.ReaimsEachRound() || !Defend.ReaimsEachRound() || Weakest.ReaimsEachRound() {
		t.Error("only assist and defend follow something each round")
	}
}
