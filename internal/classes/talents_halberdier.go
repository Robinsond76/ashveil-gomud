package classes

// The Halberdier's talents (Phase 39a): the warrior's shared ones, and a
// drill of its own for the glaive's two moves.

var (
	tSweepDrill = defineTalent(Talent{ID: "sweep-drill", Name: "Sweep Drill", Text: "Sweep is ready a round sooner", Max: 1, Add: Effects{SweepCD: 1}})
)

func init() {
	offer("halberdier", tToughness, tHeavyHands, tKeenEdge, tFootwork, tSweepDrill)
}
