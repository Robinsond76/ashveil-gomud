package classes

// The creature recruits' base ranks (Phase 38e). A creature is a species, not
// a career: it has ranks by level but no promotion routes and no talents (the
// Warhound and Runic Golem forms are named in the design and wait for a later
// phase). The family rules (food, bond, repair, carrying) are
// internal/creatures; these ranks are the combat side.

func init() {
	registerBase("hound",
		rank(1, "Run down", "its bites deal +20% damage to a foe that is exposed, knocked down or hobbled", Pounce, 20),
		rank(5, "Worry", "its bites deal +15% damage to a foe at or below half health", Wounded, 15),
		rank(10, "Fleet", "+2 Evasion", Evasion, 2),
		rank(20, "Savage pursuit", "its pounce grows to +35% damage against a foe that is exposed, knocked down or hobbled", Pounce, 35),
	)

	registerBase("stone-golem",
		rank(1, "Stone body", "its stone gives +30 armor, but it acts 10% less often and spells hit it 25% harder", Armor, 30, Slow, 10, SpellWeak, 25),
		rank(1, "Anchor", "allies in its row take 10% less damage (auras don't stack within a row)", AuraResolv, 10),
		rank(10, "Granite", "its stone hardens to +40 armor in all, and its fist lands 3 harder", Armor, 40, Damage, 3),
		rank(20, "Bedrock", "its row takes 15% less damage in all, and its fist lands 5 harder in all", AuraResolv, 15, Damage, 5),
	)
}
