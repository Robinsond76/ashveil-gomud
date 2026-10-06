package milestones

import "testing"

func TestNextMilestone(t *testing.T) {
	for _, tc := range []struct{ level, next int }{{1, 3}, {2, 3}, {3, 5}, {5, 10}, {10, 15}, {15, 20}, {20, 25}, {25, 30}, {30, 0}, {60, 0}} {
		got := Next(tc.level, "warrior")
		if got.Level != tc.next {
			t.Errorf("Next(%d)=%d, want %d", tc.level, got.Level, tc.next)
		}
		if got.Level > 0 && got.Shipped {
			t.Errorf("Next(%d): martial classes have no shipped milestone yet", tc.level)
		}
	}
}

// Phase 35b: casters' level-3 option (a second spell) has shipped; the
// martial classes' level-3 options are still coming.
func TestLevelThreeShipsForCasters(t *testing.T) {
	for _, a := range []string{"wizard", "cleric"} {
		got := Next(1, a)
		if got.Level != 3 || !got.Shipped || got.String() != "level 3: second class option" {
			t.Errorf("%s: %+v (%s)", a, got, got)
		}
		if next := Next(3, a); next.Shipped {
			t.Errorf("%s: level 5 is not shipped", a)
		}
	}
	for _, a := range []string{"warrior", "rogue", "ranger", ""} {
		if got := Next(2, a); got.Shipped || got.String() != "level 3: second class option (coming)" {
			t.Errorf("%q: %+v", a, got)
		}
	}
}
