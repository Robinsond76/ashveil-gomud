package classes

// The Arbalist's talents (Phase 39h): the ranged fighters' shared ones, and
// a heavier stock for its bolt.

var (
	tHeavyBolts = defineTalent(Talent{ID: "heavy-bolts", Name: "Heavy Bolts", Text: "a Piercing Bolt deals 10% more damage", Add: Effects{BoltDmg: 10}})
)

func init() {
	offer("arbalist", tKeenEdge, tFootwork, tToughness, tSharpEye, tHeavyBolts)
}
