package classes

// The Gryphon Rider's talents (Phase 39f): the shared fighters' ones, and a
// drill of its own for the Dive.

var (
	tWingDrill = defineTalent(Talent{ID: "wing-drill", Name: "Wing Drill", Text: "Dive is ready a round sooner", Max: 1, Add: Effects{DiveCD: 1}})
)

func init() {
	offer("gryphon-rider", tKeenEdge, tFootwork, tToughness, tSharpEye, tWingDrill)
}
