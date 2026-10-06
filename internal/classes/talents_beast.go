package classes

// The Beast Tamer's talents (Phase 39e): the beast's own, and one shared.

var (
	tThickPelt   = defineTalent(Talent{ID: "thick-pelt", Name: "Thick Pelt", Text: "your beast has 10% more health", Add: Effects{BeastHPBonus: 10}})
	tStrongJaws  = defineTalent(Talent{ID: "strong-jaws", Name: "Strong Jaws", Text: "+1 damage on your beast's blows", Add: Effects{BeastDamage: 1}})
	tSteadyLeash = defineTalent(Talent{ID: "steady-leash", Name: "Steady Leash", Text: "Rally works one more time a battle", Max: 1, Add: Effects{Rally: 1}})
	tFieldHand   = defineTalent(Talent{ID: "field-hand", Name: "Field Hand", Text: "+1 Evasion", Add: Effects{Evasion: 1}})
)

func init() {
	offer("beasttamer", tToughness, tThickPelt, tStrongJaws, tSteadyLeash, tFieldHand)
}
