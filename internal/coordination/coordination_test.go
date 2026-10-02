package coordination

import "testing"

func TestForLevelBands(t *testing.T) {
	cases := []struct {
		level int
		want  Tier
	}{
		{0, Rabble}, {1, Rabble}, {9, Rabble},
		{10, Band}, {24, Band},
		{25, Drilled}, {44, Drilled},
		{45, Veteran}, {99, Veteran},
	}
	for _, c := range cases {
		if got := ForLevel(c.level); got != c.want {
			t.Errorf("ForLevel(%d) = %d, want %d", c.level, got, c.want)
		}
	}
}

func TestOfAveragesAndOverrides(t *testing.T) {
	cases := []struct {
		name     string
		levels   []int
		explicit []int
		want     Tier
	}{
		{"empty", nil, nil, Rabble},
		{"average rounds down", []int{9, 10}, nil, Rabble}, // 9.5 -> 9
		{"average", []int{5, 15, 10}, nil, Band},
		{"boss among rats", []int{30, 1, 1, 1}, nil, Rabble},
		{"explicit raises", []int{3, 3}, []int{0, 3}, Drilled},
		{"explicit lowers", []int{50, 50}, []int{1, 0}, Rabble},
		{"highest explicit wins", []int{5}, []int{2, 4, 1}, Veteran},
		{"out of range ignored", []int{12}, []int{9}, Band},
	}
	for _, c := range cases {
		if got := Of(c.levels, c.explicit); got != c.want {
			t.Errorf("%s: Of = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFocusCount(t *testing.T) {
	cases := []struct {
		tier      Tier
		fighters  int
		wantCount int
	}{
		{Rabble, 5, 0},
		{Band, 1, 1}, {Band, 3, 2}, {Band, 4, 2}, {Band, 5, 3},
		{Drilled, 5, 5}, {Veteran, 2, 2},
		{Band, 0, 0},
	}
	for _, c := range cases {
		if got := FocusCount(c.tier, c.fighters); got != c.wantCount {
			t.Errorf("FocusCount(%d, %d) = %d, want %d", c.tier, c.fighters, got, c.wantCount)
		}
	}
}

func TestNoise(t *testing.T) {
	cases := []struct {
		tier Tier
		own  int
		want int
	}{
		{Rabble, 0, 30}, {Rabble, 50, 50},
		{Band, 0, 15}, {Drilled, 0, 5}, {Drilled, 20, 20},
		{Veteran, 40, 0},
		{Rabble, 200, 100},
	}
	for _, c := range cases {
		if got := Noise(c.tier, c.own); got != c.want {
			t.Errorf("Noise(%d, %d) = %d, want %d", c.tier, c.own, got, c.want)
		}
	}
}

func TestSpecsMatchTheDesign(t *testing.T) {
	want := map[Tier]struct {
		word      string
		healBelow int
		guards    int
	}{
		Rabble:  {"a rabble", 30, 0},
		Band:    {"a band", 50, 1},
		Drilled: {"a drilled company", 60, 2},
		Veteran: {"a veteran company", 70, 2},
	}
	for tier, w := range want {
		s := SpecOf(tier)
		if s.Word != w.word || s.HealBelow != w.healBelow || s.Guards != w.guards {
			t.Errorf("tier %d: %+v", tier, s)
		}
	}
	if SpecOf(Rabble).HealsPerRound != 1 || SpecOf(Rabble).Announce {
		t.Error("a rabble heals once a round and announces nothing")
	}
	if !SpecOf(Veteran).BreakHeals || SpecOf(Drilled).BreakHeals {
		t.Error("only a veteran company breaks heals")
	}
	if SpecOf(None).Tier != Rabble || SpecOf(9).Tier != Veteran {
		t.Error("out-of-range tiers clamp")
	}
}

func TestValidate(t *testing.T) {
	for _, n := range []int{0, 1, 4} {
		if err := ValidateTier(n); err != nil {
			t.Errorf("tier %d: %v", n, err)
		}
	}
	for _, n := range []int{-1, 5} {
		if ValidateTier(n) == nil {
			t.Errorf("tier %d accepted", n)
		}
	}
	for _, r := range []string{"", "Healer", "guardian", " caster "} {
		if err := ValidateRole(r); err != nil {
			t.Errorf("role %q: %v", r, err)
		}
	}
	if ValidateRole("bard") == nil {
		t.Error("unknown role accepted")
	}
	if ValidateWounds("") != nil || ValidateWounds("None") != nil || ValidateWounds("lasting") == nil {
		t.Error("wounds validation")
	}
}
