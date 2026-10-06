package classes

// The Halberdier's routes (Phase 39a, neutral classes design §2). The
// lineage is neutral from the start, so every route is open at any
// alignment. Ranks 10-25 are the advanced route; the three elites
// (Reaper, Linebreaker, Tempest Lancer) are named, and open with 39i.

func init() {
	register(Class{ID: "sweeper", Name: "Sweeper", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "a wider, harder Sweep that is ready sooner",
		Ranks: []Rank{
			rank(10, "Full sweep", "Sweep deals 100% of a blow's damage to every foe it strikes, not 80%", SweepPct, 100),
			rank(15, "Quick sweep", "Sweep is ready a round sooner", SweepCD, 1),
			rank(20, "Practiced hands", "+2 Attack", Attack, 2),
			rank(25, "Heavy edge", "+1 damage on every blow", Damage, 1),
		}})
	register(Class{ID: "reaper", Name: "Reaper", Lineage: "halberdier", Tier: TierElite, Parent: "sweeper", Gate: GateAny,
		Role: "a Sweep that also strikes the row behind", Planned: true})

	register(Class{ID: "vanguard", Name: "Vanguard", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "holds a column: Brace answers blows at anyone in it",
		Ranks: []Rank{
			rank(10, "Hold the line", "Brace answers the first blow struck at anyone in the Vanguard's column, and the held blow knocks the foe down when it hits", BraceCol, 1, BraceDown, 1),
			rank(15, "Parrying drill", "+4 parry chance", Parry, 4),
			rank(20, "Hardened guard", "+4 armor", Armor, 4),
			rank(25, "Unmoved", "+8% maximum health", HealthPct, 8),
		}})
	register(Class{ID: "linebreaker", Name: "Linebreaker", Lineage: "halberdier", Tier: TierElite, Parent: "vanguard", Gate: GateAny,
		Role: "a column that takes less damage while the Linebreaker stands", Planned: true})

	register(Class{ID: "valkyrie", Name: "Valkyrie", Lineage: "halberdier", Tier: TierAdvanced, Gate: GateAny,
		Role: "a Sweep charged with lightning from its mana",
		Ranks: []Rank{
			rank(10, "Charged Sweep", "a Sweep spends 8 mana to add 1d6 lightning, which ignores armor, to every foe it strikes; your mana pool doubles", ChargedMana, 8, ChargedDice, 6, ManaPct, 100),
			rank(15, "Rising storm", "the lightning is 1d8", ChargedDice, 8),
			rank(20, "Gathered storm", "the lightning is 1d10", ChargedDice, 10),
			rank(25, "Deep reserves", "your mana pool is two and a half times its base", ManaPct, 150),
		}})
	register(Class{ID: "tempest-lancer", Name: "Tempest Lancer", Lineage: "halberdier", Tier: TierElite, Parent: "valkyrie", Gate: GateAny,
		Role: "lightning that arcs to a foe in the next row", Planned: true})
}
