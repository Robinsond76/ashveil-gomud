package classes

// The Alchemist's talents (Phase 39g): a deeper satchel and hands that make
// each flask a little better.

var (
	tDeepSatchel = defineTalent(Talent{ID: "deep-satchel", Name: "Deep Satchel", Text: "+2 flasks in the satchel", Add: Effects{FlaskCap: 2}})
	tSteadyBrew  = defineTalent(Talent{ID: "steady-brew", Name: "Steady Brew", Text: "Healing Draught heals 10% more", Add: Effects{FlaskHeal: 10}})
	tHotBrew     = defineTalent(Talent{ID: "hot-brew", Name: "Hot Brew", Text: "Fire Flask burns 10% harder", Add: Effects{FlaskFire: 10}})

	// Phase 39i2: the elite talents.
	tMasterBrewer = defineEliteTalent(Talent{ID: "master-brewer", Name: "Master Brewer", Text: "+4 flasks in the satchel", Add: Effects{FlaskCap: 4}})
	tPotentFlasks = defineEliteTalent(Talent{ID: "potent-flasks", Name: "Potent Flasks", Text: "Healing Draught heals 15% more and Fire Flask burns 15% harder", Add: Effects{FlaskHeal: 15, FlaskFire: 15}})
)

func init() {
	offer("alchemist", tToughness, tFootwork, tDeepSatchel, tSteadyBrew, tHotBrew)
	offerElite("alchemist", tIronHide, tMasterBrewer, tPotentFlasks)
}
