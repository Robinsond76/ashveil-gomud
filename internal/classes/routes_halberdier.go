package classes

// The Halberdier's routes (Phase 39a, neutral classes design §2). The
// lineage is neutral from the start, so every route is open at any
// alignment. Ranks 10-25 are the advanced route; the three elites
// (Reaper, Linebreaker, Tempest Lancer) are named, and open with 39i.

func init() {
	register(Class{ID: "sweeper", Name: "Sweeper", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "a wider, harder Sweep that is ready sooner",
		Ranks: []Rank{
			rank(10, "Full sweep", "Sweep deals 100% of a blow's damage to every foe it strikes, not 90%", SweepPct, 100),
			rank(15, "Quick sweep", "Sweep is ready a round sooner", SweepCD, 1),
			rank(20, "Practiced hands", "+2 Attack", Attack, 2),
			rank(25, "Heavy edge", "+1 damage on every blow", Damage, 1),
		}})
	register(Class{ID: "reaper", Name: "Reaper", Lineage: "halberdier", Tier: TierElite, Parent: "sweeper", Gate: GateAny,
		Role: "a Sweep that also strikes the row behind, ready every round",
		Ranks: []Rank{
			rank(30, "Reaping sweep", "Sweep also strikes the foes in the row behind its target, each at 50% of a blow's damage", SweepBehind, 50),
			rank(35, "Long glaive", "+3 Attack", Attack, 5),
			rank(40, "Wider arc", "the row behind takes 75% of a blow's damage from a Sweep", SweepBehind, 75),
			rank(45, "Reaper's toll", "+1 damage on every blow", Damage, 2),
			rank(50, "Hardened", "+10% maximum health", HealthPct, 10),
			rank(55, "Reaper's eye", "+5% critical chance", Crit, 5),
			rank(60, "Harvest", "Sweep is ready two rounds sooner than a Sweeper's: every other round is a Sweep", SweepCD, 2),
		}})

	register(Class{ID: "vanguard", Name: "Vanguard", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "holds a column: Brace answers blows at anyone in it",
		Ranks: []Rank{
			rank(10, "Hold the line", "Brace answers the first blow struck at anyone in the Vanguard's column, and the held blow knocks the foe down when it hits", BraceCol, 1, BraceDown, 1),
			rank(15, "Parrying drill", "+4 parry chance", Parry, 4),
			rank(20, "Hardened guard", "+4 armor", Armor, 4),
			rank(25, "Unmoved", "+8% maximum health", HealthPct, 8),
		}})
	register(Class{ID: "linebreaker", Name: "Linebreaker", Lineage: "halberdier", Tier: TierElite, Parent: "vanguard", Gate: GateAny,
		Role: "a column that takes less damage while the Linebreaker stands, and a Brace that answers twice",
		Ranks: []Rank{
			rank(30, "Shield line", "allies in its column take 10% less damage while it stands (columns take the best cut; they don't stack)", ColumnGuard, 10),
			rank(35, "Plated greaves", "+4 armor", Armor, 8),
			rank(40, "Parrying wall", "+4 parry chance", Parry, 8),
			rank(45, "Deeper line", "its column takes 15% less damage", ColumnGuard, 15),
			rank(50, "Unbroken", "+8% maximum health", HealthPct, 16),
			rank(55, "Iron wall", "+4 armor", Armor, 12),
			rank(60, "Twin brace", "Brace answers the first two foes that strike into its column", BraceTwice, 1),
		}})

	register(Class{ID: "valkyrie", Name: "Valkyrie", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "a Sweep charged with lightning from its mana",
		Ranks: []Rank{
			rank(10, "Charged Sweep", "a Sweep spends 8 mana to add 1d6 lightning, which ignores armor, to every foe it strikes; your mana pool doubles", ChargedMana, 8, ChargedDice, 6, ManaPct, 100),
			rank(15, "Rising storm", "the lightning is 1d8", ChargedDice, 8),
			rank(20, "Gathered storm", "the lightning is 1d10", ChargedDice, 10),
			rank(25, "Deep reserves", "your mana pool is two and a half times its base", ManaPct, 150),
		}})
	register(Class{ID: "tempest-lancer", Name: "Tempest Lancer", Lineage: "halberdier", Tier: TierElite, Parent: "valkyrie", Gate: GateAny,
		Role: "lightning that arcs to a foe in the next row, and a charge that can paralyze",
		Ranks: []Rank{
			rank(30, "Arcing bolt", "each bolt of a Charged Sweep also arcs to a foe in the row behind the one it struck, for half its damage", ChargedArc, 50),
			rank(35, "Rolling thunder", "the lightning is 1d12", ChargedDice, 12),
			rank(40, "Practiced hands", "+3 Attack", Attack, 3),
			rank(45, "Forked arc", "the arc deals 75% of the bolt", ChargedArc, 75),
			rank(50, "Thunderhead", "the lightning is 1d14", ChargedDice, 14),
			rank(55, "Storm heart", "your mana pool is three times its base", ManaPct, 200),
			rank(60, "Stormstruck", "a bolt that strikes may leave its foe paralyzed for a round (35%; a boss resists, 10%); a foe is never paralyzed twice in 3 rounds", ChargedStun, 35),
		}})
}
