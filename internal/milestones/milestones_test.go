package milestones

import "testing"

func TestNextMilestone(t *testing.T) {
	for _, tc := range []struct{ level, next int }{{1, 3}, {2, 3}, {3, 5}, {5, 10}, {10, 15}, {15, 20}, {20, 25}, {25, 30}, {30, 35}, {35, 40}, {40, 45}, {45, 50}, {50, 55}, {55, 60}, {60, 0}} {
		got := Next(tc.level, "warrior")
		if got.Level != tc.next {
			t.Errorf("Next(%d)=%d, want %d", tc.level, got.Level, tc.next)
		}
	}
}

// Phase 38b delivered talents, promotion and the advanced ranks; the elite
// step is delivered per lineage (38c1 warriors and clerics, 38c3 wizards and
// witches), so only they see it without "(coming)".
func TestEliteStepIsShippedForShippedLineagesOnly(t *testing.T) {
	for _, tc := range []struct {
		level     int
		archetype string
		want      string
	}{
		{4, "warrior", "level 5: talent"},
		{9, "rogue", "level 10: class promotion"},
		{25, "wizard", "level 30: elite promotion"},
		{25, "witch", "level 30: elite promotion"},
		{25, "warrior", "level 30: elite promotion"},
		{25, "cleric", "level 30: elite promotion"},
		{29, "rogue", "level 30: elite promotion (coming)"},
		{34, "ranger", "level 35: elite rank and talent (coming)"},
		{34, "warrior", "level 35: elite rank and talent"},
		{59, "cleric", "level 60: elite capstone"},
		{60, "warrior", "No upcoming milestone announced."},
	} {
		if got := Next(tc.level, tc.archetype).String(); got != tc.want {
			t.Errorf("Next(%d, %s) = %q, want %q", tc.level, tc.archetype, got, tc.want)
		}
	}
}

// Phase 35b: casters' level-3 option (a second spell) has shipped; the
// martial classes' level-3 options are still coming. Talents (38b) shipped.
func TestLevelThreeShipsForCasters(t *testing.T) {
	for _, a := range []string{"wizard", "cleric"} {
		got := Next(1, a)
		if got.Level != 3 || !got.Shipped || got.String() != "level 3: second class option" {
			t.Errorf("%s: %+v (%s)", a, got, got)
		}
		if next := Next(3, a); !next.Shipped || next.Level != 5 {
			t.Errorf("%s: level 5 talents shipped with 38b: %+v", a, next)
		}
	}
	for _, a := range []string{"warrior", "rogue", "ranger", ""} {
		if got := Next(2, a); got.Shipped || got.String() != "level 3: second class option (coming)" {
			t.Errorf("%q: %+v", a, got)
		}
	}
}
