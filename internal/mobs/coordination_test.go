package mobs

import "testing"

func TestCoordinationTemplateFields(t *testing.T) {
	m := Mob{}
	if m.EnemyRole() != "fighter" || !m.TakesWounds() {
		t.Fatalf("defaults: role %q, wounds %v", m.EnemyRole(), m.TakesWounds())
	}
	m = Mob{Role: " Healer ", WoundsRule: "None", Coordination: 3}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if m.EnemyRole() != "healer" || m.TakesWounds() {
		t.Fatalf("role %q, wounds %v", m.EnemyRole(), m.TakesWounds())
	}
	for _, bad := range []Mob{{Role: "bard"}, {Coordination: 5}, {WoundsRule: "lasting"}} {
		if bad.Validate() == nil {
			t.Errorf("%+v accepted", bad)
		}
	}
}
