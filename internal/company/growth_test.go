package company

import "testing"

func TestDealIsPrefixStableAndExact(t *testing.T) {
	weights := []GrowthWeights{
		{4, 2, 0, 3, 0, 1},
		{0, 0, 3, 0, 4, 2},
		EvenGrowth,
		{},
		{1, 0, 0, 0, 0, 0},
	}
	for _, w := range weights {
		prev := Deal(0, w)
		for n := 1; n <= 120; n++ {
			got := Deal(n, w)
			sum := 0
			grew := 0
			for i := range got {
				sum += got[i]
				if got[i] < prev[i] {
					t.Fatalf("weights %v: Deal(%d) lowered stat %d: %v -> %v", w, n, i, prev, got)
				}
				grew += got[i] - prev[i]
			}
			if sum != n || grew != 1 {
				t.Fatalf("weights %v: Deal(%d) = %v (sum %d, grew %d)", w, n, got, sum, grew)
			}
			prev = got
		}
	}
}

func TestDealFollowsWeights(t *testing.T) {
	w := GrowthWeights{4, 2, 0, 3, 0, 1}
	got := Deal(100, w)
	want := [6]int{40, 20, 0, 30, 0, 10}
	if got != want {
		t.Fatalf("Deal(100) = %v, want %v", got, want)
	}
	if got := Deal(6, GrowthWeights{}); got != [6]int{1, 1, 1, 1, 1, 1} {
		t.Fatalf("empty weights should deal evenly, got %v", got)
	}
	if got := Deal(-3, w); got != [6]int{} {
		t.Fatalf("negative points dealt %v", got)
	}
}

func TestGrowthFocusAndParsing(t *testing.T) {
	w := GrowthWeights{4, 2, 0, 3, 0, 1}
	if got := w.WithFocus("smarts"); got != (GrowthWeights{4, 2, 2, 3, 0, 1}) {
		t.Fatalf("WithFocus(smarts) = %v", got)
	}
	if got := w.WithFocus(""); got != w {
		t.Fatalf("empty focus changed weights: %v", got)
	}
	if got := (GrowthWeights{}).WithFocus("vitality"); got != (GrowthWeights{1, 1, 1, 3, 1, 1}) {
		t.Fatalf("focus on even growth = %v", got)
	}
	for in, want := range map[string]int{"str": 0, "Speed": 1, "myst": 4, "perception": 5} {
		if i, ok := GrowthStatIndex(in); !ok || i != want {
			t.Fatalf("GrowthStatIndex(%q) = %d, %v", in, i, ok)
		}
	}
	for _, in := range []string{"", "s", "luck"} {
		if _, ok := GrowthStatIndex(in); ok {
			t.Fatalf("GrowthStatIndex(%q) should fail", in)
		}
	}
	if w, ok := GrowthWeightsFrom(map[string]int{"Strength": 3, "vitality": 2, "luck": 5, "speed": -1, "str": 4}); !ok || w != (GrowthWeights{3, 0, 0, 2, 0, 0}) {
		t.Fatalf("GrowthWeightsFrom = %v, %v", w, ok)
	}
	if _, ok := GrowthWeightsFrom(map[string]int{"luck": 1}); ok {
		t.Fatal("weights with nothing usable should report !ok")
	}
}
