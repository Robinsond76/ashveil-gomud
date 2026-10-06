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
		Role: "a Dive ready every other round", Planned: true})

	register(Class{ID: "skyscout", Name: "Skyscout", Lineage: "gryphon-rider", Tier: TierAdvanced, Gate: GateAny,
		Role: "a watcher on the wing: it spots ambushes and marks its prey",
		Ranks: []Rank{
			rank(10, "Eagle eye", "+6 Perception when your company looks for an ambush in the open (not indoors)", SkyEye, 6),
			rank(15, "Quick stoop", "Dive is ready a round sooner", DiveCD, 1),
			rank(20, "Hawk's mark", "a Dive that lands leaves the foe exposed", DiveExpo, 1),
			rank(25, "Far sight", "Eagle eye gives +12 Perception", SkyEye, 12),
		}})
	register(Class{ID: "falcon-marshal", Name: "Falcon Marshal", Lineage: "gryphon-rider", Tier: TierElite, Parent: "skyscout", Gate: GateAny,
		Role: "allies strike harder at the foe it dove on", Planned: true})

	register(Class{ID: "wyvern-rider", Name: "Wyvern Rider", Lineage: "gryphon-rider", Tier: TierAdvanced, Gate: GateAny,
		Role: "a venomous stoop on a wyvern's tougher hide",
		Ranks: []Rank{
			rank(10, "Venom stoop", "a Dive that lands poisons the foe", DivePois, 1),
			rank(15, "Wyvern hide", "+8% maximum health", HealthPct, 8),
			rank(20, "Stinging tail", "+2 Attack", Attack, 2),
			rank(25, "Wyvern guile", "+3 Evasion", Evasion, 3),
		}})
	register(Class{ID: "wyvern-lord", Name: "Wyvern Lord", Lineage: "gryphon-rider", Tier: TierElite, Parent: "wyvern-rider", Gate: GateAny,
		Role: "poisoned foes it dives on take more damage", Planned: true})
}
