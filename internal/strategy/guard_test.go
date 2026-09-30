package strategy

import "testing"

func TestGuardianRoleWords(t *testing.T) {
	for _, in := range []string{"guardian", "Guard", "protector"} {
		if got, ok := ParseRole(in); !ok || got != Guardian {
			t.Errorf("ParseRole(%q) = %q, %v", in, got, ok)
		}
	}
	// 30c2 (owner decision 10): "guard" is the guardian now, not defend.
	if _, ok := ParseRule("guard"); ok {
		t.Error("guard is no longer a target rule")
	}
	for _, in := range []string{"defend", "protect"} {
		if got, ok := ParseRule(in); !ok || got != Defend {
			t.Errorf("ParseRule(%q) = %q, %v", in, got, ok)
		}
	}
	for _, arch := range []string{"", "warrior", "cleric", "wizard", "rogue", "ranger"} {
		if Default(arch).Role == Guardian {
			t.Errorf("%q defaults to guardian", arch)
		}
	}
	if Roles[len(Roles)-1] != Guardian {
		t.Errorf("Roles = %v", Roles)
	}
}

func TestGuardianStrategyKeepsWard(t *testing.T) {
	s := Strategy{Role: Guardian, Ward: "leader"}.Resolve("warrior")
	if s.Role != Guardian || s.Ward != "leader" || s.Rule != Weakest {
		t.Errorf("Resolve = %+v", s)
	}
	if (Strategy{Ward: "leader"}).IsZero() {
		t.Error("a ward is a setting")
	}
	if a := Decide(Situation{Role: Guardian, Foes: 3, Mana: 50}); a.Kind != Swing {
		t.Errorf("a guardian swings, got %+v", a)
	}
}

func TestGuardWard(t *testing.T) {
	members := []Guarded{
		{Key: "leader", HP: 10, MaxHP: 10, InReach: true},
		{Key: "companion:1", HP: 6, MaxHP: 10, InReach: true},
		{Key: "companion:2", HP: 3, MaxHP: 10, InReach: false},
		{Key: "companion:3", HP: 12, MaxHP: 20, InReach: true},
	}
	cases := []struct {
		name, guardian, ward, want string
		ok                         bool
	}{
		{"set ward in reach", "companion:1", "leader", "leader", true},
		{"set ward out of reach", "companion:1", "companion:2", "", false},
		{"never itself", "companion:1", "companion:1", "", false},
		{"most hurt in reach, ties by order", "companion:9", "", "companion:1", true},
		{"most hurt skips itself", "companion:1", "", "companion:3", true},
		{"a ward gone reads as none", "leader", "companion:7", "companion:1", true},
	}
	for _, c := range cases {
		got, ok := GuardWard(c.guardian, c.ward, members)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: GuardWard = %q, %v; want %q, %v", c.name, got, ok, c.want, c.ok)
		}
	}
	// Nobody hurt: nobody guarded.
	full := []Guarded{{Key: "leader", HP: 10, MaxHP: 10, InReach: true}, {Key: "companion:1", HP: 5, MaxHP: 5, InReach: true}}
	if got, ok := GuardWard("companion:2", "", full); ok {
		t.Errorf("nobody hurt, got %q", got)
	}
	// Equal fractions: formation order.
	even := []Guarded{{Key: "companion:4", HP: 5, MaxHP: 10, InReach: true}, {Key: "leader", HP: 10, MaxHP: 20, InReach: true}}
	if got, _ := GuardWard("companion:2", "", even); got != "companion:4" {
		t.Errorf("tie: got %q", got)
	}
}
