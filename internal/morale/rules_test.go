package morale

import "testing"

func TestProbabilities(t *testing.T) {
	for _, tt := range []struct {
		s                 string
		hold, yield, flee int
	}{{"", 100, 0, 0}, {"unbreakable", 100, 0, 0}, {"steadfast", 85, 10, 5}, {"wary", 65, 25, 10}, {"craven", 40, 40, 20}, {"skittish", 65, 0, 35}} {
		var counts [3]int
		for r := 0; r < 100; r++ {
			counts[Enemy(tt.s, r)]++
		}
		if counts != [3]int{tt.hold, tt.yield, tt.flee} {
			t.Fatalf("%s: %v", tt.s, counts)
		}
	}
	if Validate("unknown") == nil {
		t.Fatal("invalid accepted")
	}
}
func TestTriggersConsumedAndDepartureNotDeath(t *testing.T) {
	g := Group{Leader: 1, Specialist: 2, Size: 4}
	ms := []Member{{ID: 1, HP: 0, Dead: true}, {ID: 2, HP: 0, Dead: true}, {ID: 3, HP: 25, Max: 100, Active: true}, {ID: 4, HP: 80, Max: 100}}
	if !g.Check(ms) || g.Check(ms) {
		t.Fatal("simultaneous checks must combine once")
	}
	ms[2].HP = 100
	g.Check(ms)
	ms[2].HP = 25
	if g.Check(ms) {
		t.Fatal("low-health farming")
	}
	g = Group{Leader: 1, Size: 2}
	if g.Check([]Member{{ID: 1, HP: 40, Max: 100}, {ID: 2, HP: 100, Max: 100, Active: true}}) {
		t.Fatal("flight treated as death")
	}
}
func TestNerveAndReactions(t *testing.T) {
	for _, tt := range []struct {
		loyalty              int
		weak                 bool
		hold, hesitate, flee int
	}{{24, false, 75, 20, 5}, {25, true, 90, 10, 0}, {25, false, 100, 0, 0}} {
		var count [4]int
		for r := 0; r < 100; r++ {
			count[Nerve(tt.loyalty, tt.weak, r)]++
		}
		if count[Hold] != tt.hold || count[Hesitate] != tt.hesitate || count[Flee] != tt.flee {
			t.Fatal(count)
		}
	}
	if !Losing(4, 2, 100, 100) || !Losing(4, 0, 25, 100) || Losing(4, 1, 26, 100) {
		t.Fatal("threshold")
	}
	for _, a := range []int{-25, 0, 25} {
		for _, spare := range []bool{false, true} {
			v := Reaction(a, spare)
			if a == 0 && v != 0 || a != 0 && ((a > 0) == spare) && v != 2 || a != 0 && ((a > 0) != spare) && v != -1 {
				t.Fatal(a, spare, v)
			}
		}
	}
}

// Phase 38c2: a hunted foe flees less often, never not at all.
func TestEnemyHuntedShrinksTheFleeBandButKeepsAFloor(t *testing.T) {
	flees := func(s string, penalty int) int {
		n := 0
		for roll := 0; roll < 100; roll++ {
			if EnemyHunted(s, roll, penalty) == Flee {
				n++
			}
		}
		return n
	}
	for _, tc := range []struct {
		s       string
		penalty int
		want    int
	}{
		{"wary", 0, 10}, {"skittish", 0, 35}, {"skittish", 25, 10}, {"skittish", 40, 5},
		{"wary", 90, 5}, {"unbreakable", 25, 0}, {"", 25, 0},
	} {
		if got := flees(tc.s, tc.penalty); got != tc.want {
			t.Errorf("%q penalty %d: %d fleeing rolls, want %d", tc.s, tc.penalty, got, tc.want)
		}
	}
	for roll := 0; roll < 100; roll++ {
		if Enemy("skittish", roll) != Flee && EnemyHunted("skittish", roll, 25) != Enemy("skittish", roll) {
			t.Errorf("roll %d: only flight may change", roll)
		}
	}
}
