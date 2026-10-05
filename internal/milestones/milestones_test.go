package milestones

import "testing"

func TestNextMilestone(t *testing.T) {
	for _, tc := range []struct{ level, next int }{{1, 3}, {2, 3}, {3, 5}, {5, 10}, {10, 15}, {15, 20}, {20, 25}, {25, 30}, {30, 0}, {60, 0}} {
		got := Next(tc.level)
		if got.Level != tc.next {
			t.Errorf("Next(%d)=%d, want %d", tc.level, got.Level, tc.next)
		}
		if got.Level > 0 && got.Shipped {
			t.Error("35a ships no milestone choices")
		}
	}
}
