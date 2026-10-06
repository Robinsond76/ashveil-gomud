package classes

// The Alchemist's talents (Phase 39g): a deeper satchel and hands that make
// each flask a little better.

var (
	tDeepSatchel = defineTalent(Talent{ID: "deep-satchel", Name: "Deep Satchel", Text: "+2 flasks in the satchel", Add: Effects{FlaskCap: 2}})
	tSteadyBrew  = defineTalent(Talent{ID: "steady-brew", Name: "Steady Brew", Text: "Healing Draught heals 10% more", Add: Effects{FlaskHeal: 10}})
	tHotBrew     = defineTalent(Talent{ID: "hot-brew", Name: "Hot Brew", Text: "Fire Flask burns 10% harder", Add: Effects{FlaskFire: 10}})
)

func init() {
	offer("alchemist", tToughness, tFootwork, tDeepSatchel, tSteadyBrew, tHotBrew)
}
