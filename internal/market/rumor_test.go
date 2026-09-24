package market

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func rumorHide() Good {
	return Good{ItemID: 28, BasePrice: 14, MinPrice: 8, MaxPrice: 30, MaxStock: 40, TargetStock: 20, StartStock: 20, DriftStep: 2}
}

func rumorPostHide() Good {
	return Good{ItemID: 28, BasePrice: 9, MinPrice: 4, MaxPrice: 20, MaxStock: 30, TargetStock: 15, StartStock: 15, DriftStep: 2}
}

func kinds(rumors []Rumor) []RumorKind {
	out := []RumorKind{}
	for _, r := range rumors {
		out = append(out, r.Kind)
	}
	return out
}

func TestRumorsScarceAndGlut(t *testing.T) {
	g := rumorHide()
	cases := []struct {
		stock int
		want  []RumorKind
	}{
		{0, []RumorKind{RumorScarce}},
		{9, []RumorKind{RumorScarce}},  // scarce: below half the target
		{10, []RumorKind{}},            // short: not worth a rumour
		{20, []RumorKind{}},            // steady
		{29, []RumorKind{}},            // still steady
		{30, []RumorKind{RumorGlut}},   // plentiful
		{40, []RumorKind{RumorGlut}},   // glutted
		{99, []RumorKind{RumorGlut}},   // clamped
		{-5, []RumorKind{RumorScarce}}, // clamped
	}
	for _, c := range cases {
		got := Rumors([]Sighting{{Place: "Dunmar", Good: g, Stock: c.stock}}, 20)
		assert.Equal(t, c.want, kinds(got), "stock %d", c.stock)
		for _, r := range got {
			assert.Equal(t, 28, r.ItemID)
			assert.Equal(t, "Dunmar", r.Place)
		}
	}
}

func TestRumorsCheapestAndBestBuyerNeedTwoMarkets(t *testing.T) {
	town := Sighting{Place: "Square", Good: rumorHide(), Stock: 20}
	post := Sighting{Place: "Post", Good: rumorPostHide(), Stock: 15}
	assert.Empty(t, Rumors([]Sighting{town}, 20), "one market alone has no cheapest/best buyer")

	got := Rumors([]Sighting{town, post}, 20)
	assert.Equal(t, []Rumor{
		{Kind: RumorCheapest, ItemID: 28, Place: "Post"},
		{Kind: RumorBestBuyer, ItemID: 28, Place: "Square"},
	}, got)

	// A different good in the other market doesn't make a comparison.
	meat := Sighting{Place: "Post", Good: Good{ItemID: 29, BasePrice: 3, MinPrice: 1, MaxPrice: 8, MaxStock: 40, TargetStock: 20, DriftStep: 1}, Stock: 20}
	assert.Empty(t, Rumors([]Sighting{town, meat}, 20))
}

func TestRumorsSkipTiedExtremes(t *testing.T) {
	a := Sighting{Place: "A", Good: rumorHide(), Stock: 20}
	b := Sighting{Place: "B", Good: rumorHide(), Stock: 20}
	assert.Empty(t, Rumors([]Sighting{a, b}, 20), "identical markets: no unique extreme")

	// Three markets: two tie for cheapest, the third is the unique best buyer.
	c := Sighting{Place: "C", Good: rumorHide(), Stock: 15}
	got := Rumors([]Sighting{a, b, c}, 20)
	assert.Equal(t, []Rumor{{Kind: RumorBestBuyer, ItemID: 28, Place: "C"}}, got)
}

func TestRumorsSkipClosedSides(t *testing.T) {
	// The post is sold out (no ask), so the square, the only market
	// selling, is cheapest; the empty post bids highest.
	town := Sighting{Place: "Square", Good: rumorHide(), Stock: 20}
	post := Sighting{Place: "Post", Good: rumorPostHide(), Stock: 0}
	got := Rumors([]Sighting{town, post}, 20)
	assert.Equal(t, []Rumor{
		{Kind: RumorScarce, ItemID: 28, Place: "Post"},
		{Kind: RumorCheapest, ItemID: 28, Place: "Square"},
		{Kind: RumorBestBuyer, ItemID: 28, Place: "Post"},
	}, got)

	// A full market has no bid, so the square is the only buyer.
	full := Sighting{Place: "Post", Good: rumorPostHide(), Stock: 30}
	got = Rumors([]Sighting{town, full}, 20)
	assert.Equal(t, []Rumor{
		{Kind: RumorGlut, ItemID: 28, Place: "Post"},
		{Kind: RumorCheapest, ItemID: 28, Place: "Post"},
		{Kind: RumorBestBuyer, ItemID: 28, Place: "Square"},
	}, got)

	// Both sold out: nobody is cheapest.
	emptyTown := Sighting{Place: "Square", Good: rumorHide(), Stock: 0}
	got = Rumors([]Sighting{emptyTown, post}, 20)
	assert.NotContains(t, kinds(got), RumorCheapest)
}

func TestRumorsDeterministicOrder(t *testing.T) {
	sightings := []Sighting{
		{Place: "Z", Good: Good{ItemID: 29, BasePrice: 5, MinPrice: 2, MaxPrice: 12, MaxStock: 60, TargetStock: 30, DriftStep: 3}, Stock: 0},
		{Place: "B", Good: rumorHide(), Stock: 39},
		{Place: "A", Good: rumorHide(), Stock: 5},
	}
	want := Rumors(sightings, 20)
	reversed := []Sighting{sightings[2], sightings[1], sightings[0]}
	assert.Equal(t, want, Rumors(reversed, 20))
	assert.Equal(t, []Rumor{
		{Kind: RumorScarce, ItemID: 28, Place: "A"},
		{Kind: RumorGlut, ItemID: 28, Place: "B"},
		{Kind: RumorCheapest, ItemID: 28, Place: "B"},
		{Kind: RumorBestBuyer, ItemID: 28, Place: "A"},
		{Kind: RumorScarce, ItemID: 29, Place: "Z"},
	}, want)
}

func TestPickRumorsDistinctAndBounded(t *testing.T) {
	all := []Rumor{
		{Kind: RumorScarce, ItemID: 1, Place: "A"},
		{Kind: RumorGlut, ItemID: 2, Place: "B"},
		{Kind: RumorCheapest, ItemID: 3, Place: "C"},
		{Kind: RumorBestBuyer, ItemID: 4, Place: "D"},
	}
	for seed := uint64(0); seed < 50; seed++ {
		n := seed
		roll := func() uint64 { n = n*6364136223846793005 + 1442695040888963407; return n }
		got := PickRumors(all, 3, roll)
		assert.Len(t, got, 3)
		seen := map[Rumor]bool{}
		for _, r := range got {
			assert.Contains(t, all, r)
			assert.False(t, seen[r], "distinct")
			seen[r] = true
		}
	}
	zero := func() uint64 { return 0 }
	assert.ElementsMatch(t, all, PickRumors(all, 10, zero), "n above the count returns all")
	assert.Empty(t, PickRumors(all, 0, zero))
	assert.Empty(t, PickRumors(nil, 3, zero))
	assert.Len(t, all, 4, "input untouched")
	assert.Equal(t, RumorScarce, all[0].Kind, "input order untouched")
}
