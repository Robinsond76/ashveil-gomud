package classes

// The Arbalist's ranks and routes (Phase 39h, neutral classes design §9).
// The lineage is neutral from the start, so every route is open at any
// alignment. Piercing Bolt is an automatic ability (internal/strategy); its
// ranks here shape it. The three elites (Siege Master, Deadeye, Bastion) are
// named, and open with 39i.

func init() {
	registerBase("arbalist",
		rank(1, "Piercing Bolt", "a heavy crossbow bolt that ignores half the target's armor (cooldown 2 rounds); winding the crossbow afterwards takes the Arbalist's next turn", BoltPierce, 50),
		rank(3, "Steady Aim", "a bolt has +10 Attack when nothing has hurt the Arbalist since its last one", SteadyAim, 10),
		rank(6, "Crippling Bolt", "a bolt that lands hobbles the foe for 2 rounds", BoltCripple, 2),
		rank(8, "Armor-breaker", "a Piercing Bolt ignores all of the target's armor", BoltPierce, 100),
	)

	register(Class{ID: "siegebreaker", Name: "Siegebreaker", Lineage: "arbalist", Tier: TierAdvanced, Gate: GateAny,
		Role: "bolts that break armor down, bolt by bolt",
		Ranks: []Rank{
			rank(10, "Sundering bolts", "each bolt that lands takes 10 armor off the foe for the battle, up to 30", Shred, 10, ShredCap, 30),
			rank(15, "Heavy stock", "a Piercing Bolt deals 20% more damage", BoltDmg, 20),
			rank(20, "Deep sunder", "each bolt that lands takes 15 armor off the foe, up to 45", Shred, 15, ShredCap, 45),
			rank(25, "Siege bolts", "a Piercing Bolt deals 35% more damage", BoltDmg, 35),
		}})
	register(Class{ID: "siege-master", Name: "Siege Master", Lineage: "arbalist", Tier: TierElite, Parent: "siegebreaker", Gate: GateAny,
		Role: "a bolt that passes through to the foe behind", Planned: true})

	register(Class{ID: "sharpshooter", Name: "Sharpshooter", Lineage: "arbalist", Tier: TierAdvanced, Gate: GateAny,
		Role: "a surer eye: critical bolts that a shield cannot stop",
		Ranks: []Rank{
			rank(10, "Keen sight", "+15% critical chance with a shooting weapon, and its critical hits can't be blocked", RangedCrit, 15, NoBlockCrit, 1),
			rank(15, "Steady hands", "Steady Aim gives +20 Attack", SteadyAim, 20),
			rank(20, "Practiced loader", "the first bolt of each battle needs no reload", FirstLoaded, 1),
			rank(25, "Dead eye", "+4 Attack with a shooting weapon", RangedAttack, 4),
		}})
	register(Class{ID: "deadeye", Name: "Deadeye", Lineage: "arbalist", Tier: TierElite, Parent: "sharpshooter", Gate: GateAny,
		Role: "a crossbow that is ready again at once", Planned: true})

	register(Class{ID: "warden-of-the-wall", Name: "Warden of the Wall", Lineage: "arbalist", Tier: TierAdvanced, Gate: GateAny,
		Role: "sets a pavise: the row behind it takes less damage",
		Ranks: []Rank{
			rank(10, "Pavise", "allies in the Warden's row take 10% less damage (the Arbalist included; auras don't stack within a row)", AuraResolv, 10),
			rank(15, "Padded coat", "+4 armor", Armor, 4),
			rank(20, "Deeper pavise", "the row's damage cut is 15%", AuraResolv, 15),
			rank(25, "Wall's keeper", "+8% maximum health", HealthPct, 8),
		}})
	register(Class{ID: "bastion", Name: "Bastion", Lineage: "arbalist", Tier: TierElite, Parent: "warden-of-the-wall", Gate: GateAny,
		Role: "answers a foe that strikes its column with the shot it was holding", Planned: true})
}
