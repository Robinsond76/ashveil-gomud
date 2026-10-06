package classes

// The Doll Master's talents (Phase 39d): the doll's own, and two of the
// shared ones.

var (
	tHardwood    = defineTalent(Talent{ID: "hardwood", Name: "Hardwood", Text: "your dolls have 10% more health", Add: Effects{DollHPBonus: 10}})
	tStoutString = defineTalent(Talent{ID: "stout-strings", Name: "Stout Strings", Text: "Guard String works one more time a battle", Max: 1, Add: Effects{DollGuards: 1}})
	tFineCarving = defineTalent(Talent{ID: "fine-carving", Name: "Fine Carving", Text: "+1 damage on the doll's blows", Add: Effects{DollDamage: 1}})
	tSteadyHand  = defineTalent(Talent{ID: "steady-hand", Name: "Steady Hand", Text: "+2 Attack on the doll's blows", Add: Effects{DollAttack: 2}})
)

func init() {
	offer("dollmaster", tToughness, tHardwood, tStoutString, tFineCarving, tSteadyHand)
}
