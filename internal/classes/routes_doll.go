package classes

// The Doll Master's lineage (Phase 39d, neutral classes design §3). The
// lineage is neutral from the start: every route is open at any alignment.
// Its base ranks come by level (Guard String, Tangle, Emergency Splice);
// ranks 10-25 are the advanced route, and the three elites (Grand
// Puppeteer, Golem Lord, String Sovereign) are named and open with 39i.

func init() {
	registerBase("dollmaster",
		rank(5, "Guard String", "the doll guards the most hurt ally beside it, twice a battle", DollGuards, 2),
		rank(8, "Stronger strings", "Guard String works three times a battle", DollGuards, 3),
		rank(12, "Tangle", "strings snag a foe in reach of the doll and push its action meter back by half a turn (a quarter for a boss); ready every 3 rounds, and the same foe can't be tangled again for 2 rounds", DollTangle, 1),
		rank(18, "Emergency Splice", "once a battle, when the doll would break, you spend your next turn and it stands back up at 25% health", Splice, 1),
	)

	register(Class{ID: "puppeteer", Name: "Puppeteer", Lineage: "dollmaster", Tier: TierAdvanced, Gate: GateAny,
		Role: "two dolls, each with half the health: more bodies in the line",
		Ranks: []Rank{
			rank(10, "Two dolls", "you drive a second doll; each doll has 50% of the health of a single doll. Strike drives the first standing doll, and Guard String works for both", DollCount, 1, DollHPPct, 50),
			rank(15, "Finer joints", "+2 Attack on each doll's blows", DollAttack, 2),
			rank(20, "Seasoned wood", "each doll has 60% of the health of a single doll", DollHPPct, 60),
			rank(25, "Lacquered limbs", "each doll carries 4% armor of its own", DollArmor, 4),
		}})
	register(Class{ID: "grand-puppeteer", Name: "Grand Puppeteer", Lineage: "dollmaster", Tier: TierElite, Parent: "puppeteer", Gate: GateAny,
		Role: "dolls with more health, and a strike that hits with both", Planned: true})

	register(Class{ID: "golemancer", Name: "Golemancer", Lineage: "dollmaster", Tier: TierAdvanced, Gate: GateAny,
		Role: "one great golem: more health, its own armor, stronger strings",
		Ranks: []Rank{
			rank(10, "Golem doll", "your doll is a golem with 130% of the health and 15% armor of its own, but it cannot wear armor; Guard String works 4 times a battle", DollHPPct, 130, DollArmor, 15, DollNoWear, 1, DollGuards, 4),
			rank(15, "Heavy fists", "+1 damage on the golem's blows", DollDamage, 1),
			rank(20, "Stone heart", "the golem has 150% of the health of a single doll", DollHPPct, 150),
			rank(25, "Unyielding", "the golem carries 25% armor of its own", DollArmor, 25),
		}})
	register(Class{ID: "golem-lord", Name: "Golem Lord", Lineage: "dollmaster", Tier: TierElite, Parent: "golemancer", Gate: GateAny,
		Role: "a golem whose blows knock foes down", Planned: true})

	register(Class{ID: "marionettist", Name: "Marionettist", Lineage: "dollmaster", Tier: TierAdvanced, Gate: GateAny,
		Role: "strings that snag more foes, sooner",
		Ranks: []Rank{
			rank(10, "Nimble strings", "Tangle snags two foes and is ready every 2 rounds", TangleFoes, 2, TangleCD, 1),
			rank(15, "Quick fingers", "+2 Attack", Attack, 2),
			rank(20, "Taut line", "Tangle pushes a foe's action meter back by 60", TanglePush, 10),
			rank(25, "Steady hands", "+2 damage on the doll's blows", DollDamage, 2),
		}})
	register(Class{ID: "string-sovereign", Name: "String Sovereign", Lineage: "dollmaster", Tier: TierElite, Parent: "marionettist", Gate: GateAny,
		Role: "a tangled foe's next attack is weaker", Planned: true})
}
