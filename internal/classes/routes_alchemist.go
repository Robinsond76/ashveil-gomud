package classes

// The Alchemist's ranks and routes (Phase 39g, neutral classes design §8).
// The lineage is neutral from the start, so every route is open at any
// alignment. Its flasks are spells that cost a flask instead of mana
// (internal/flasks); the ranks here shape them. The three elites (Panacean,
// Grenadier, Transmuter) are named, and open with 39i.

func init() {
	registerBase("alchemist",
		rank(1, "Healing Draught", "a thrown flask heals any ally, a Minor Heal's worth, with no chant and no mana (one flask)"),
		rank(1, "Brew", "a camp rest (or the brew command) refills the satchel from reagents"),
		rank(3, "Antidote", "a flask that cures an ally's poison and bleeding"),
		rank(6, "Fire Flask", "a flask of fire at one foe and the foe beside it, at 70% of Sparks"),
		rank(8, "Bracing Tonic", "a flask that gives one ally +5 Attack and +5 Evasion for 3 rounds"),
	)

	register(Class{ID: "apothecary", Name: "Apothecary", Lineage: "alchemist", Tier: TierAdvanced, Gate: GateAny,
		Role: "a surer healer: stronger draughts, a deeper satchel and a draught that cleans",
		Ranks: []Rank{
			rank(10, "Potent draughts", "Healing Draught heals 30% more", FlaskHeal, 30),
			rank(15, "Deep satchel", "+4 flasks in the satchel", FlaskCap, 4),
			rank(20, "Splash draught", "a Healing Draught also heals the next most hurt ally for 30% as much", FlaskSplash, 30),
			rank(25, "Clean draught", "a Healing Draught also takes one harmful status off its patient (once a battle for each ally)", FlaskClean, 1),
		}})
	register(Class{ID: "panacean", Name: "Panacean", Lineage: "alchemist", Tier: TierElite, Parent: "apothecary", Gate: GateAny,
		Role: "an elixir that keeps a falling ally on its feet, and richer draughts",
		Ranks: []Rank{
			rank(30, "Elixir", "once a battle, a blow that would fell an ally leaves it standing with a quarter of its health", ElixirSave, 25, ElixirUses, 1),
			rank(35, "Richer draughts", "Healing Draught heals 50% more", FlaskHeal, 50),
			rank(40, "Deeper satchel", "+8 flasks in the satchel (in all)", FlaskCap, 8),
			rank(45, "Wide splash", "a Healing Draught's splash heals 50% as much", FlaskSplash, 50),
			rank(50, "Fine elixir", "the Elixir leaves the ally with 40% of its health", ElixirSave, 40),
			rank(55, "Master's satchel", "+12 flasks in the satchel (in all)", FlaskCap, 12),
			rank(60, "Second elixir", "the Elixir works twice a battle", ElixirUses, 2),
		}})

	register(Class{ID: "bombardier", Name: "Bombardier", Lineage: "alchemist", Tier: TierAdvanced, Gate: GateAny,
		Role: "a thrower of fire: a hotter flask, a wider throw and foes left burning",
		Ranks: []Rank{
			rank(10, "Hotter flasks", "Fire Flask burns at 100% of Sparks (from 70%)", FlaskFire, 43),
			rank(15, "Wide throw", "Fire Flask reaches one more foe", FlaskReach, 1),
			rank(20, "Pitch and tar", "foes a Fire Flask burns are left alight (2 damage a round for 4 rounds)", FlaskBurn, 1),
			rank(25, "Deep satchel", "+4 flasks in the satchel", FlaskCap, 4),
		}})
	register(Class{ID: "grenadier", Name: "Grenadier", Lineage: "alchemist", Tier: TierElite, Parent: "bombardier", Gate: GateAny,
		Role: "flasks that burst over most of an enemy company",
		Ranks: []Rank{
			rank(30, "Grenado", "Fire Flask reaches two more foes (four in all)", FlaskReach, 2),
			rank(35, "Hotter still", "Fire Flask burns at 112% of Sparks", FlaskFire, 60),
			rank(40, "Bandolier", "+8 flasks in the satchel (in all)", FlaskCap, 8),
			rank(45, "Barrage", "Fire Flask reaches three more foes (five in all, a whole company)", FlaskReach, 3),
			rank(50, "White fire", "Fire Flask burns at 126% of Sparks", FlaskFire, 80),
			rank(55, "Sapper's frame", "+10% maximum health", HealthPct, 10),
			rank(60, "Arsenal", "+12 flasks in the satchel (in all), and Fire Flask burns at 140% of Sparks", FlaskCap, 12, FlaskFire, 100),
		}})

	register(Class{ID: "mutagenist", Name: "Mutagenist", Lineage: "alchemist", Tier: TierAdvanced, Gate: GateAny,
		Role: "turns its tonic into a mutagen: armor for the battle, at the price of an ally's health",
		Ranks: []Rank{
			rank(10, "Mutagen", "Bracing Tonic also hardens its ally: +10 armor for the battle, for a tenth of the ally's health", MutagenArmr, 10, MutagenCost, 10),
			rank(15, "Stable mutagen", "a Mutagen costs a twentieth of the ally's health", MutagenCost, 5),
			rank(20, "Strong mutagen", "a Mutagen gives +15 armor, and a Bracing Tonic lasts 2 rounds longer", MutagenArmr, 15, TonicLong, 2),
			rank(25, "Pure mutagen", "a Mutagen costs its ally no health", MutagenFree, 1),
		}})
	register(Class{ID: "transmuter", Name: "Transmuter", Lineage: "alchemist", Tier: TierElite, Parent: "mutagenist", Gate: GateAny,
		Role: "a mutagen that hardens more of the row, and a longer, stronger tonic",
		Ranks: []Rank{
			rank(30, "Twin mutagen", "a Mutagen also hardens one ally beside its patient in the same row, paid for by neither", MutagenRow, 1),
			rank(35, "Alloyed mutagen", "a Mutagen gives +20 armor", MutagenArmr, 20),
			rank(40, "Long tonic", "a Bracing Tonic lasts 4 rounds longer (7 in all)", TonicLong, 4),
			rank(45, "Hardened mutagen", "a Mutagen gives +25 armor", MutagenArmr, 25),
			rank(50, "Deeper satchel", "+4 flasks in the satchel", FlaskCap, 4),
			rank(55, "Transmuter's skin", "+8% maximum health", HealthPct, 8),
			rank(60, "Row mutagen", "a Mutagen hardens its patient's whole row", MutagenRow, 5),
		}})
}
