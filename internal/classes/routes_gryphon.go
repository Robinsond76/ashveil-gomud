package classes

// The Gryphon Rider's ranks and routes (Phase 39f, neutral classes design
// §5). The lineage is neutral from the start, so every route is open at any
// alignment. Dive itself is an automatic ability (internal/strategy); its
// ranks here shape it. The three elites (Gryphon Lord, Falcon Marshal, Wyvern
// Lord) are named, and open with 39i.

func init() {
	registerBase("gryphon-rider",
		rank(1, "Dive", "a stooping blow that can pass a standing front-row foe to strike the middle or back row (cooldown 3 rounds, open sky only); the rider has -10 Evasion until its next turn"),
		rank(3, "Talons", "a Dive that lands leaves the foe bleeding", Talons, 1),
		rank(8, "Power dive", "a Dive deals 25% more damage", DiveDmg, 25),
	)

	register(Class{ID: "gryphon-knight", Name: "Gryphon Knight", Lineage: "gryphon-rider", Tier: TierAdvanced, Gate: GateAny,
		Role: "a heavy stoop that knocks its foe down",
		Ranks: []Rank{
			rank(10, "Lance charge", "a Dive with a two-handed reach weapon (a war spear) knocks the foe down when it lands", DiveDown, 1),
			rank(15, "Plated barding", "+4 armor", Armor, 4),
			rank(20, "Falling stone", "a Dive deals 50% more damage", DiveDmg, 50),
			rank(25, "Steady wings", "a Dive costs no Evasion", DiveSteady, 1),
		}})
	register(Class{ID: "gryphon-lord", Name: "Gryphon Lord", Lineage: "gryphon-rider", Tier: TierElite, Parent: "gryphon-knight", Gate: GateAny,
		Role: "a Dive ready every other round, and a landing that shakes the foes beside its target",
		Ranks: []Rank{
			rank(30, "Lord's stoop", "Dive is ready every other round (cooldown 2)", DiveCD, 1),
			rank(35, "Lord's plate", "+8 armor in all", Armor, 8),
			rank(40, "Mountain stoop", "a Dive deals 75% more damage", DiveDmg, 75),
			rank(45, "Gryphon's vigor", "+10% maximum health", HealthPct, 10),
			rank(50, "Avalanche", "a Dive deals 100% more damage", DiveDmg, 100),
			rank(55, "Sky lord", "+4 Attack", Attack, 4),
			rank(60, "Thunder landing", "a Dive that knocks its foe down also knocks down the foes beside it", DiveQuake, 1),
		}})

	register(Class{ID: "skyscout", Name: "Skyscout", Lineage: "gryphon-rider", Tier: TierAdvanced, Gate: GateAny,
		Role: "a watcher on the wing: it spots ambushes and marks its prey",
		Ranks: []Rank{
			rank(10, "Eagle eye", "+6 Perception when your company looks for an ambush in the open (not indoors)", SkyEye, 6),
			rank(15, "Quick stoop", "Dive is ready a round sooner", DiveCD, 1),
			rank(20, "Hawk's mark", "a Dive that lands leaves the foe exposed", DiveExpo, 1),
			rank(25, "Far sight", "Eagle eye gives +12 Perception", SkyEye, 12),
		}})
	register(Class{ID: "falcon-marshal", Name: "Falcon Marshal", Lineage: "gryphon-rider", Tier: TierElite, Parent: "skyscout", Gate: GateAny,
		Role: "allies strike harder at the foe it dove on",
		Ranks: []Rank{
			rank(30, "Marshal's mark", "a foe a Dive lands on is marked for 2 rounds: every ally has +5 Attack against it", DiveMark, 5),
			rank(35, "Keen horizon", "Eagle eye gives +18 Perception", SkyEye, 18),
			rank(40, "Falcon's talons", "+4 Attack", Attack, 4),
			rank(45, "Rallying cry", "the mark gives +8 Attack", DiveMark, 8),
			rank(50, "Swift wings", "a Dive deals 50% more damage", DiveDmg, 50),
			rank(55, "Marshal's guard", "+6 Evasion", Evasion, 6),
			rank(60, "Sky commander", "the mark gives +12 Attack", DiveMark, 12),
		}})

	register(Class{ID: "wyvern-rider", Name: "Wyvern Rider", Lineage: "gryphon-rider", Tier: TierAdvanced, Gate: GateAny,
		Role: "a venomous stoop on a wyvern's tougher hide",
		Ranks: []Rank{
			rank(10, "Venom stoop", "a Dive that lands poisons the foe", DivePois, 1),
			rank(15, "Wyvern hide", "+8% maximum health", HealthPct, 8),
			rank(20, "Stinging tail", "+2 Attack", Attack, 2),
			rank(25, "Wyvern guile", "+3 Evasion", Evasion, 3),
		}})
	register(Class{ID: "wyvern-lord", Name: "Wyvern Lord", Lineage: "gryphon-rider", Tier: TierElite, Parent: "wyvern-rider", Gate: GateAny,
		Role: "poisoned foes it dives on take more damage, and a tail that lashes a second foe",
		Ranks: []Rank{
			rank(30, "Rotting venom", "a Dive deals 25% more damage to a poisoned foe", DivePoisX, 25),
			rank(35, "Wyvern scales", "+14% maximum health in all", HealthPct, 14),
			rank(40, "Barbed tail", "+5 Attack", Attack, 5),
			rank(45, "Deep venom", "a Dive deals 40% more damage to a poisoned foe", DivePoisX, 40),
			rank(50, "Wyvern cunning", "+7 Evasion", Evasion, 7),
			rank(55, "Power dive", "a Dive deals 50% more damage", DiveDmg, 50),
			rank(60, "Lashing tail", "a Dive that lands also lashes the foe beside its target with the wyvern's tail for half the damage, and poisons it", DiveTail, 50),
		}})
}
