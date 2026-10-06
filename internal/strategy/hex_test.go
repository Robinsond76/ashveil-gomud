package strategy

import (
	"reflect"
	"testing"
)

func TestHexTargets(t *testing.T) {
	foes := []Foe{
		{ID: 1, Row: 0, Col: 0},
		{ID: 2, Row: 0, Col: 1},
		{ID: 3, Row: 0, Col: 2},
		{ID: 4, Row: 1, Col: 0, Caster: true, Chanting: true},
		{ID: 5, Row: 2, Col: 0},
	}
	cases := []struct {
		name    string
		foes    []Foe
		row     bool
		reach   int
		winding func(int) bool
		want    []int
	}{
		{"one foe at level 1: the nearest", foes, false, 1, nil, []int{4}},
		{"a chanting caster is the primary, then its row, then the nearest others", foes, false, 3, nil, []int{4, 1, 2}},
		{"a wind-up outranks a chanting caster", foes, false, 2, func(id int) bool { return id == 3 }, []int{3, 1}},
		{"a row hex keeps to the primary's row", foes, true, 4, nil, []int{4}},
		{"a row hex covers a full row", foes[:3], true, 4, nil, []int{1, 2, 3}},
		{"the whole group", foes, false, 1000, nil, []int{4, 1, 2, 3, 5}},
		{"no foes", nil, false, 2, nil, nil},
		{"no reach", foes, false, 0, nil, nil},
	}
	for _, c := range cases {
		got := HexTargets(c.foes, c.row, c.reach, c.winding)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestControllerRole(t *testing.T) {
	if r, ok := ParseRole("hexer"); !ok || r != Controller {
		t.Errorf("ParseRole(hexer) = %q, %v", r, ok)
	}
	if got := Default("witch"); got.Role != Controller {
		t.Errorf("a witch defaults to %q", got.Role)
	}
	if Default("wizard").Role != Caster {
		t.Error("a wizard still casts")
	}
}

func TestControllerHexesThenCursesThenSwings(t *testing.T) {
	spells := []Spell{
		{ID: "binding", Use: UseHex, Cost: 12},
		{ID: "slumber", Use: UseHex, Cost: 6},
		{ID: "blight", Use: UseHex, Cost: 12},
		{ID: "hex", Use: UseAttack, Cost: 8},
	}
	knows := func(known ...string) func(string) bool {
		return func(id string) bool {
			for _, k := range known {
				if k == id {
					return true
				}
			}
			return false
		}
	}
	base := Situation{Role: Controller, Mana: 50, Spells: spells, Foes: 3, Knows: knows("slumber", "hex")}
	if a := Decide(base); a.Kind != Hex || a.Spell != "slumber" {
		t.Errorf("first known hex: %+v", a)
	}
	both := base
	both.Knows = knows("slumber", "binding", "hex")
	if a := Decide(both); a.Spell != "binding" {
		t.Errorf("the listed order is the preference: %+v", a)
	}
	worth := both
	worth.CanHex = func(id string) bool { return id != "binding" }
	if a := Decide(worth); a.Spell != "slumber" {
		t.Errorf("a hex with no foe worth it is skipped: %+v", a)
	}
	none := base
	none.CanHex = func(string) bool { return false }
	if a := Decide(none); a.Kind != Attack || a.Spell != "hex" {
		t.Errorf("no hex has a target: the weak curse: %+v", a)
	}
	poor := none
	poor.Mana = 5
	if a := Decide(poor); a.Kind != Swing {
		t.Errorf("no mana: swing %+v", a)
	}
	broke := base
	broke.Mana = 5
	if a := Decide(broke); a.Kind != Swing {
		t.Errorf("a hex it can't pay for is skipped: %+v", a)
	}
	empty := base
	empty.Foes = 0
	if a := Decide(empty); a.Kind != Swing {
		t.Errorf("no foes: %+v", a)
	}
}
