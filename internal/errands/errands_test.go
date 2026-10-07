package errands

import "testing"

func TestWordsResolve(t *testing.T) {
	for word, want := range map[string]Kind{"escort": Escort, "Hunt": Hunt, "scout": Scout, "esc": Escort} {
		if got, ok := KindByWord(word); !ok || got != want {
			t.Errorf("KindByWord(%q) = %q, %v", word, got, ok)
		}
	}
	if _, ok := KindByWord("e"); ok {
		t.Error("one letter should not match a kind")
	}
	for word, want := range map[string]Length{"short": Short, "2h": Medium, "LONG": Long, "30m": Short} {
		if got, ok := LengthByWord(word); !ok || got != want {
			t.Errorf("LengthByWord(%q) = %q, %v", word, got, ok)
		}
	}
	if _, ok := LengthByWord("forever"); ok {
		t.Error("an unknown length matched")
	}
}

func TestNewSetsReturnAndFallbackBand(t *testing.T) {
	e := New(Hunt, Medium, "Brindle Downs", 1000, 6, 0, 0, 80, 9)
	if e.ReturnsAt != 1000+2*3600 || e.BandLow != 6 || e.BandHigh != 6 {
		t.Fatalf("errand = %+v", e)
	}
	if e.Due(1000) || !e.Due(e.ReturnsAt) || e.Remaining(1000) != 2*3600 || e.Remaining(e.ReturnsAt+5) != 0 {
		t.Fatalf("timing wrong for %+v", e)
	}
}

func TestPayScalesWithBandKindAndLengthButSlowerThanTime(t *testing.T) {
	short := New(Escort, Short, "z", 0, 8, 7, 9, 80, 1)
	if got := Pay(short); got != 32 {
		t.Fatalf("level 8 in 7-9, short escort pays %d, want 32", got)
	}
	long := New(Escort, Long, "z", 0, 8, 7, 9, 80, 1)
	hours := (Long.Info().Seconds / Short.Info().Seconds)
	if Pay(long) >= Pay(short)*int(hours) {
		t.Fatalf("a long errand (%d) must pay less than 16 short ones (%d each)", Pay(long), Pay(short))
	}
	if Pay(New(Hunt, Short, "z", 0, 8, 7, 9, 80, 1)) <= Pay(short) || Pay(New(Scout, Short, "z", 0, 8, 7, 9, 80, 1)) >= Pay(short) {
		t.Fatal("a hunt should pay more and a scouting job less than an escort")
	}
	if Pay(New(Escort, Short, "z", 0, 8, 13, 15, 80, 1)) <= Pay(short) {
		t.Fatal("a deeper band should pay more")
	}
}

func TestWoundRiskGrowsUnderTheBand(t *testing.T) {
	if got := WoundRisk(New(Escort, Short, "z", 0, 8, 7, 9, 80, 1)); got != 0 {
		t.Fatalf("an escort in band risks %d", got)
	}
	in := WoundRisk(New(Hunt, Short, "z", 0, 8, 7, 9, 80, 1))
	under := WoundRisk(New(Hunt, Short, "z", 0, 5, 7, 9, 80, 1))
	over := WoundRisk(New(Hunt, Short, "z", 0, 12, 7, 9, 80, 1))
	if !(over < in && in < under) || under != 15+16 {
		t.Fatalf("risk over/in/under = %d/%d/%d", over, in, under)
	}
	if got := WoundRisk(New(Scout, Short, "z", 0, 1, 20, 22, 80, 1)); got != 60 {
		t.Fatalf("risk should cap at 60, got %d", got)
	}
}

func TestResolveIsStableForASeed(t *testing.T) {
	e := New(Hunt, Medium, "z", 0, 8, 7, 9, 80, 424242)
	first := Resolve(e, true)
	for i := 0; i < 20; i++ {
		if got := Resolve(e, true); got != first {
			t.Fatalf("outcome changed: %+v then %+v", first, got)
		}
	}
}

func TestResolveSpreadsAndStaysInBand(t *testing.T) {
	counts := map[Kind]map[OutcomeKind]int{}
	for _, kind := range []Kind{Escort, Hunt, Scout} {
		counts[kind] = map[OutcomeKind]int{}
		for seed := int64(0); seed < 4000; seed++ {
			e := New(kind, Short, "z", 0, 8, 7, 9, 80, seed)
			o := Resolve(e, true)
			counts[kind][o.Kind]++
			switch o.Kind {
			case Gold, Item:
				if o.Gold != Pay(e) {
					t.Fatalf("%s %s pay %d, want %d", kind, o.Kind, o.Gold, Pay(e))
				}
			case Rumour, Wound:
				if o.Gold != 0 {
					t.Fatalf("%s %s carries gold %d", kind, o.Kind, o.Gold)
				}
			}
			if o.Kind == Wound && o.WoundPct != WoundPct {
				t.Fatalf("wound pct %d", o.WoundPct)
			}
		}
	}
	if counts[Escort][Wound] != 0 {
		t.Errorf("an escort in band wounded %d times", counts[Escort][Wound])
	}
	if h := counts[Hunt][Wound]; h < 400 || h > 800 {
		t.Errorf("hunts wounded %d of 4000, want about 15%%", h)
	}
	if counts[Scout][Rumour] < counts[Scout][Gold] || counts[Hunt][Item] < counts[Hunt][Rumour] {
		t.Errorf("weights look wrong: %v", counts)
	}
}

func TestNoLairMeansNoRumour(t *testing.T) {
	for seed := int64(0); seed < 2000; seed++ {
		e := New(Scout, Short, "z", 0, 8, 7, 9, 80, seed)
		if o := Resolve(e, false); o.Kind == Rumour {
			t.Fatal("a rumour with no lair to name")
		}
	}
}

func TestSpan(t *testing.T) {
	for in, want := range map[int64]string{30: "a minute", 1800: "about 30 minutes", 7200: "about 2 hours", 3 * 86400: "about 3 days"} {
		if got := Span(in); got != want {
			t.Errorf("Span(%d) = %q, want %q", in, got, want)
		}
	}
}
