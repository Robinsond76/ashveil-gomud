package sky

import "testing"

func TestPhaseForDayCyclesAndWraps(t *testing.T) {
	want := []MoonPhase{New, WaxingCrescent, FirstQuarter, WaxingGibbous, Full, WaningGibbous, LastQuarter, WaningCrescent}
	for day, expected := range want {
		if got := PhaseForDay(uint64(day), 8); got != expected {
			t.Errorf("day %d: got %v want %v", day, got, expected)
		}
	}
	if got := PhaseForDay(8, 8); got != New {
		t.Errorf("day 8 should wrap to new, got %v", got)
	}
	// A longer cycle holds each phase for several days.
	if got := PhaseForDay(14, 28); got != Full {
		t.Errorf("day 14 of 28 should be full, got %v", got)
	}
	if got := PhaseForDay(3, 28); got != New {
		t.Errorf("day 3 of 28 should still be new, got %v", got)
	}
}

func TestPhaseForDayZeroCycleUsesDefault(t *testing.T) {
	if got, want := PhaseForDay(4, 0), PhaseForDay(4, DefaultCycleDays); got != want {
		t.Fatalf("zero cycle: got %v want %v", got, want)
	}
}

func TestMoonlight(t *testing.T) {
	cases := []struct {
		phase MoonPhase
		cloud int
		want  int
	}{
		{Full, 0, 2},
		{Full, 1, 2},
		{Full, 2, 1},
		{Full, 3, 0},
		{New, 0, 0},
		{WaxingCrescent, 0, 1},
		{FirstQuarter, 2, 0},
		{WaningGibbous, 0, 2},
	}
	for _, c := range cases {
		if got := Moonlight(c.phase, c.cloud); got != c.want {
			t.Errorf("Moonlight(%v, %d) = %d want %d", c.phase, c.cloud, got, c.want)
		}
	}
}

func TestMoonVisible(t *testing.T) {
	if !MoonVisible(2) || MoonVisible(3) {
		t.Fatal("moon should be visible through broken cloud but hidden by overcast")
	}
}

func TestAbsoluteDay(t *testing.T) {
	if got := AbsoluteDay(250, 100); got != 2 {
		t.Fatalf("got %d want 2", got)
	}
	if got := AbsoluteDay(250, 0); got != 0 {
		t.Fatalf("zero rounds-per-day should report day 0, got %d", got)
	}
}

func TestPhaseNamesDistinct(t *testing.T) {
	seen := map[string]bool{}
	for p := New; p <= WaningCrescent; p++ {
		name := p.Name()
		if name == "" || seen[name] {
			t.Fatalf("phase %d has empty or duplicate name %q", p, name)
		}
		seen[name] = true
	}
	if MoonPhase(99).Name() != "" {
		t.Fatal("invalid phase should have no name")
	}
}

func TestCloudCoverName(t *testing.T) {
	for cover := 0; cover <= MaxCloudCover; cover++ {
		if CloudCoverName(cover) == "" {
			t.Fatalf("cover %d has no name", cover)
		}
	}
	if CloudCoverName(-1) != "" || CloudCoverName(MaxCloudCover+1) != "" {
		t.Fatal("out of range cover should have no name")
	}
}

func TestSetCycleDays(t *testing.T) {
	defer SetCycleDays(DefaultCycleDays)
	SetCycleDays(28)
	if CycleDays() != 28 {
		t.Fatalf("got %d", CycleDays())
	}
	SetCycleDays(0)
	if CycleDays() != DefaultCycleDays {
		t.Fatalf("non-positive cycle should reset to default, got %d", CycleDays())
	}
}
