package classes

// The route tables. Every number is a starting value for the balance tests
// (faith routes design; the other lineages' tables are 38b's own). A rank is
// one benefit, one every five levels from promotion to level 60.

// rank builds a Rank; set is key, value pairs.
func rank(level int, name, text string, set ...any) Rank {
	e := Effects{}
	for i := 0; i+1 < len(set); i += 2 {
		e[set[i].(string)] = set[i+1].(int)
	}
	return Rank{Level: level, Name: name, Text: text, Set: e}
}

// teaches adds the spells a rank grants.
func teaches(r Rank, spells ...string) Rank {
	r.Spells = spells
	return r
}

func init() {
	// ----- Warrior -----
	register(Class{ID: "knight", Name: "Knight", Lineage: "warrior", Tier: TierAdvanced, Gate: GateGood,
		Role: "armored protector who heals by laying on hands",
		Ranks: []Rank{
			rank(10, "Lay on Hands", "heals itself or an adjacent ally for half a Minor Heal of its level; takes its turn, no chant or mana; 2 uses per rest", LayHands, 2),
			rank(15, "Second wind", "Lay on Hands: 3 uses per rest", LayHands, 3),
			rank(20, "Shield of faith", "+5 block chance while guarding a ward", FaithBlock, 5),
			rank(25, "Staunch hands", "Lay on Hands also stops bleeding", LayBleed, 1),
		}})
	register(Class{ID: "paladin", Name: "Paladin", Lineage: "warrior", Tier: TierElite, Parent: "knight", Gate: GateGood,
		Role: "heavy-armored healer who stands in the front row",
		Ranks: []Rank{
			rank(30, "Paladin", "Lay on Hands heals a full Minor Heal and removes one harmful status", LayFull, 1),
			rank(35, "Aura of Resolve", "allies in the Paladin's row take 10% less damage (auras don't stack within a row)", AuraResolv, 10),
			rank(40, "Tireless hands", "Lay on Hands: 4 uses per rest", LayHands, 4),
			rank(45, "Smite", "the Paladin's blows deal +50% against undead and demons", Smite, 50),
			rank(50, "Rallying aura", "Aura of Resolve also gives the rest of the company 5%", AuraCompan, 5),
			rank(55, "Far hands", "Lay on Hands reaches any ally in reach, not only adjacent ones", LayReach, 1),
			rank(60, "Divine shield", "once a battle the Paladin ignores the next blow; Lay on Hands: 5 uses per rest", DivineShield, 1, LayHands, 5),
		}})
	register(Class{ID: "mercenary", Name: "Mercenary", Lineage: "warrior", Tier: TierAdvanced, Gate: GateAny,
		Role: "physical disruption; a harder, surer Tackle",
		Ranks: []Rank{
			rank(10, "Hard tackle", "Tackle succeeds 10 points more often and is ready a round sooner", TackleHit, 10, TackleCD, 1),
			rank(15, "Practiced hands", "+2 Attack", Attack, 2),
			rank(20, "Pin down", "a tackled foe stays down a round longer", TackleHold, 1),
			rank(25, "Hit while down", "a tackled foe is also left exposed", TackleExpo, 1),
		}})
	register(Class{ID: "warlord", Name: "Warlord", Lineage: "warrior", Tier: TierElite, Parent: "mercenary", Gate: GateAny,
		Role: "pressure on the chosen target", Planned: true})
	register(Class{ID: "blackguard", Name: "Blackguard", Lineage: "warrior", Tier: TierAdvanced, Gate: GateEvil,
		Role: "armored protector who heals by spilling blood",
		Ranks: []Rank{
			rank(10, "Blood Oath", "when it lands a melee blow, half the damage heals the most hurt ally (itself if none is hurt); 3 blows a battle. A foe it wounds has -3 Attack against its allies that round and the next", BloodOath, 3, OathPct, 50, Intimidate, 3),
			rank(15, "Deeper oath", "Blood Oath: 4 blows a battle", BloodOath, 4),
			rank(20, "Fearful wounds", "Intimidation: -5 Attack", Intimidate, 5),
			rank(25, "Deepest oath", "Blood Oath: 5 blows a battle", BloodOath, 5),
		}})
	register(Class{ID: "dread-knight", Name: "Dread Knight", Lineage: "warrior", Tier: TierElite, Parent: "blackguard", Gate: GateEvil,
		Role: "fearsome armored healer, a stronger Blood Oath",
		Ranks: []Rank{
			rank(30, "Dread Knight", "Blood Oath heals 75% of the damage", OathPct, 75),
			rank(35, "Aura of Dread", "foes aimed at the Dread Knight have -5 Attack", AuraDread, 5),
			rank(40, "Spreading oath", "Blood Oath also heals the next most hurt ally for half as much", OathSecond, 1),
			rank(45, "Bottomless oath", "Blood Oath: 6 blows a battle", BloodOath, 6),
			rank(50, "Terror", "its critical hits that land also stagger the target, which loses its next action", TerrorCrit, 1),
			rank(55, "Endless oath", "Blood Oath: 7 blows a battle", BloodOath, 7),
			rank(60, "Unholy vigor", "Blood Oath heals 100% of the damage dealt", OathPct, 100),
		}})

	// ----- Cleric -----
	register(Class{ID: "priest", Name: "Priest", Lineage: "cleric", Tier: TierAdvanced, Gate: GateGood,
		Role: "the best direct healer; wards the company",
		Ranks: []Rank{
			teaches(rank(10, "Ward", "Ward: one ally's next blow is absorbed, up to one average hit of the Priest's level; chant 1, cost 10, one ward per ally", Ward, 1, WardBlows, 1), "ward"),
			teaches(rank(15, "Greater Heal", "heals about two average hits; chant 2, cost 14", GreaterHeal, 1), "greaterheal"),
			rank(20, "Double ward", "a ward absorbs two blows", WardBlows, 2),
			rank(25, "Prayer of Mending", "after-battle patching costs 20% less mana", PatchCost, 20),
		}})
	register(Class{ID: "hierarch", Name: "Hierarch", Lineage: "cleric", Tier: TierElite, Parent: "priest", Gate: GateGood,
		Role: "calls an Angel that grows every five levels",
		Ranks: []Rank{
			teaches(rank(30, "Call the Host", "an Angel arrives with a warrior's health of the Hierarch's level, a radiant blade, Guard once a battle and Mercy; the Hierarch's heals also remove one harmful status, once per patient per battle", Summon, 1, AngelGuards, 1, AngelMercy, 3, AngelBlade, 8, CleanseHeal, 1), "callhost"),
			rank(35, "Armor of the Host", "the Angel gains armor like heavy kit (defense 40) without its slowness", SummonArmor, 40),
			rank(40, "Mercy", "Mercy becomes a full Minor Heal every 2 rounds", AngelMercyFull, 1, AngelMercy, 2),
			rank(45, "Wings of the Host", "allies in the Angel's row gain +5 Evasion while it stands", AngelWings, 5),
			rank(50, "Sword of the Host", "Guard up to 3 times a battle; the blade becomes 1d10, +50% against undead and demons", AngelGuards, 3, AngelBlade, 10),
			rank(55, "Cleansing light", "on arrival, the Angel removes one harmful status from every ally", AngelCleanse, 1),
			rank(60, "Swift Host", "the Angel arrives one chant round sooner, and Mercy heals the two most hurt allies", SummonSooner, 1, AngelMercyTwo, 1),
		}})
	register(Class{ID: "druid", Name: "Druid", Lineage: "cleric", Tier: TierAdvanced, Gate: GateAny,
		Role: "healing over time and nature's protection",
		Ranks: []Rank{
			teaches(rank(10, "Rejuvenation", "one ally heals over 3 rounds for 130% of a Minor Heal, at a Minor Heal's cost", Rejuvenation, 3, RejuvPct, 130), "rejuvenation"),
			teaches(rank(15, "Barkskin", "one ally gains +10 armor for the battle", Barkskin, 10), "barkskin"),
			rank(20, "Lasting growth", "Rejuvenation lasts 4 rounds (160%)", Rejuvenation, 4, RejuvPct, 160),
			rank(25, "Thornhide", "a foe that strikes a Barkskinned ally takes 2 damage", Thornhide, 2),
		}})
	register(Class{ID: "elder-druid", Name: "Elder Druid", Lineage: "cleric", Tier: TierElite, Parent: "druid", Gate: GateAny,
		Role: "heals whole rows at once",
		Ranks: []Rank{
			teaches(rank(30, "Grove", "Rejuvenation on a whole formation row at 60% each", Grove, 1, GrovePct, 60), "grove"),
			rank(35, "Nature's patience", "after-battle patching costs 20% less mana", PatchCost, 20),
			rank(40, "Deep grove", "Grove at 80%", GrovePct, 80),
			teaches(rank(45, "Entangle", "one foe is hobbled for 2 rounds; chant 1, cost 10", Entangle, 1), "entangle"),
			rank(50, "Wide bark", "Barkskin covers a whole row", BarkRow, 1),
			rank(55, "Wild growth", "Rejuvenation also cures poison", WildGrowth, 1),
			rank(60, "Two groves", "Grove covers two rows", Grove, 2),
		}})
	register(Class{ID: "blood-priest", Name: "Blood Priest", Lineage: "cleric", Tier: TierAdvanced, Gate: GateEvil,
		Role: "heals by draining foes; dark healing is hungry",
		Ranks: []Rank{
			teaches(rank(10, "Siphon", "damages one foe for a Magic Missile's worth and heals the most hurt ally for the damage dealt; chant 1, cost 10; ordinary heals and patching cost 25% more mana", Siphon, 1, SiphonCost, 10, HealCost, 25), "siphon"),
			rank(15, "Cheaper siphon", "Siphon costs 8", SiphonCost, 8),
			rank(20, "Twin siphon", "Siphon also strikes a second foe and heals a second ally, at 60%", Siphon, 2),
			rank(25, "Blood ward", "Siphon healing beyond an ally's full health becomes a ward of up to half a hit", BloodWard, 1),
		}})
	register(Class{ID: "demonologist", Name: "Demonologist", Lineage: "cleric", Tier: TierElite, Parent: "blood-priest", Gate: GateEvil,
		Role: "binds a Demon that grows every five levels",
		Ranks: []Rank{
			teaches(rank(30, "Bind the Fiend", "a Demon arrives with 80% of a warrior's health of the Demonologist's level, claws (2d6) and Dread; if the Demonologist falls, the Demon breaks free and attacks the nearest creature, then vanishes", Summon, 2, DemonDread, 1, DemonClaws, 6), "bindfiend"),
			rank(35, "Infernal hide", "the Demon gains armor (defense 30)", SummonArmor, 30),
			rank(40, "Terror", "Dread checks every enemy group, not only its target's", DemonDread, 2),
			rank(45, "Hellfire", "every foe takes 2 damage each round while the Demon stands", DemonHellfire, 2),
			rank(50, "Rending claws", "claws 2d8, +50% against holy creatures", DemonClaws, 8, RendHoly, 50),
			rank(55, "Soul feast", "when a foe falls while the Demon fights, the most hurt ally heals half a Minor Heal and the Demonologist regains 5% of its mana", DemonFeast, 1),
			rank(60, "Mastered binding", "the Demon arrives one chant round sooner, and a falling Demonologist's Demon simply vanishes", SummonSooner, 1, DemonMastered, 1),
		}})

	// ----- Rogue -----
	register(Class{ID: "scout", Name: "Scout", Lineage: "rogue", Tier: TierAdvanced, Gate: GateGood,
		Role: "safe opening attacks against foes caught unprepared",
		Ranks: []Rank{
			rank(10, "Ambush", "Opening Strike opens any foe in the battle's first round, not only a downed or exposed one", Ambush, 1),
			rank(15, "Light step", "+3 Evasion", Evasion, 3),
			rank(20, "Quick opening", "Opening Strike is ready a round sooner", OpenCD, 1),
			rank(25, "Long ambush", "Ambush lasts the battle's first two rounds", Ambush, 2),
		}})
	register(Class{ID: "pathfinder", Name: "Pathfinder", Lineage: "rogue", Tier: TierElite, Parent: "scout", Gate: GateGood,
		Role: "reconnaissance and exposed foes", Planned: true})
	register(Class{ID: "duelist", Name: "Duelist", Lineage: "rogue", Tier: TierAdvanced, Gate: GateAny,
		Role: "precise blade fighting; answers a parry with a blow",
		Ranks: []Rank{
			rank(10, "Riposte", "a blow it parries is answered at once with a counter-blow the size of an Opening Strike", Riposte, 1),
			rank(15, "Parrying drill", "+4 parry chance", Parry, 4),
			rank(20, "Sharper riposte", "the counter-blow is half again as strong", Riposte, 2),
			rank(25, "Duelist's edge", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "swordmaster", Name: "Swordmaster", Lineage: "rogue", Tier: TierElite, Parent: "duelist", Gate: GateAny,
		Role: "parry-based ripostes", Planned: true})
	register(Class{ID: "assassin", Name: "Assassin", Lineage: "rogue", Tier: TierAdvanced, Gate: GateEvil,
		Role: "finishing blows against weakened foes",
		Ranks: []Rank{
			rank(10, "Finisher", "blows against a foe at or below 40% health deal an Opening Strike's damage more", Finisher, 1),
			rank(15, "Deep cuts", "Opening Strike deals +2 damage", OpenBonus, 2),
			rank(20, "Killing eye", "Finisher works on a foe at or below 50% health", Finisher, 2),
			rank(25, "Cold edge", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "nightblade", Name: "Nightblade", Lineage: "rogue", Tier: TierElite, Parent: "assassin", Gate: GateEvil,
		Role: "finishing blows and poison", Planned: true})

	// ----- Ranger -----
	register(Class{ID: "warden", Name: "Warden", Lineage: "ranger", Tier: TierAdvanced, Gate: GateGood,
		Role: "watches over the row it stands in",
		Ranks: []Rank{
			rank(10, "Watchful row", "allies in the Warden's row gain +3 Evasion", AuraEvade, 3),
			rank(15, "Steady shot", "Aimed Shot is ready a round sooner", AimCD, 1),
			rank(20, "Keener watch", "the row's Evasion bonus is +5", AuraEvade, 5),
			rank(25, "Shelter", "allies in the Warden's row take 5% less damage", AuraResolv, 5),
		}})
	register(Class{ID: "sentinel", Name: "Sentinel", Lineage: "ranger", Tier: TierElite, Parent: "warden", Gate: GateGood,
		Role: "ranged protection of vulnerable allies", Planned: true})
	register(Class{ID: "hunter", Name: "Hunter", Lineage: "ranger", Tier: TierAdvanced, Gate: GateAny,
		Role: "accurate ranged focus and a harder Aimed Shot",
		Ranks: []Rank{
			rank(10, "Hunter's eye", "Aimed Shot deals +3 damage", AimBonus, 3),
			rank(15, "Quick draw", "Aimed Shot is ready a round sooner", AimCD, 1),
			rank(20, "True aim", "+3 Attack", Attack, 3),
			rank(25, "Killing shot", "Aimed Shot deals +5 damage", AimBonus, 5),
		}})
	register(Class{ID: "marksman", Name: "Marksman", Lineage: "ranger", Tier: TierElite, Parent: "hunter", Gate: GateAny,
		Role: "advanced Aimed Shot", Planned: true})
	register(Class{ID: "stalker", Name: "Stalker", Lineage: "ranger", Tier: TierAdvanced, Gate: GateEvil,
		Role: "pressure on wounded targets",
		Ranks: []Rank{
			rank(10, "Pursuit", "blows against a foe at or below half health deal 20% more", Wounded, 20),
			rank(15, "Silent step", "+2 Evasion", Evasion, 2),
			rank(20, "Relentless", "Pursuit deals 30% more", Wounded, 30),
			rank(25, "Predator's eye", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "ravager", Name: "Ravager", Lineage: "ranger", Tier: TierElite, Parent: "stalker", Gate: GateEvil,
		Role: "pursuit without blocking every flee", Planned: true})

	// ----- Wizard -----
	register(Class{ID: "theurgist", Name: "Theurgist", Lineage: "wizard", Tier: TierAdvanced, Gate: GateGood,
		Role: "protective arcane support",
		Ranks: []Rank{
			teaches(rank(10, "Arcane Ward", "Arcane Ward: one ally's next blow is absorbed, up to one average hit of the Theurgist's level; chant 1, cost 10, one ward per ally", Ward, 1, WardBlows, 1), "arcaneward"),
			rank(15, "Steady chant", "blows break the Theurgist's chant 25% less often", ChantBreak, 25),
			rank(20, "Double ward", "a ward absorbs two blows", WardBlows, 2),
			rank(25, "Deep reserves", "+10% maximum mana", ManaPct, 10),
		}})
	register(Class{ID: "archon", Name: "Archon", Lineage: "wizard", Tier: TierElite, Parent: "theurgist", Gate: GateGood,
		Role: "disruption of hostile magic", Planned: true})
	register(Class{ID: "arcanist", Name: "Arcanist", Lineage: "wizard", Tier: TierAdvanced, Gate: GateAny,
		Role: "reliable damage and mana-efficient casting",
		Ranks: []Rank{
			rank(10, "Arcane focus", "spells deal 10% more damage", SpellPct, 10),
			rank(15, "Efficient casting", "spells cost 10% less mana", SpellCost, 10),
			rank(20, "Sharper focus", "spells deal 15% more damage", SpellPct, 15),
			rank(25, "Steady chant", "blows break the Arcanist's chant 25% less often", ChantBreak, 25),
		}})
	register(Class{ID: "archmage", Name: "Archmage", Lineage: "wizard", Tier: TierElite, Parent: "arcanist", Gate: GateAny,
		Role: "the strongest reliable damage", Planned: true})
	register(Class{ID: "warlock", Name: "Warlock", Lineage: "wizard", Tier: TierAdvanced, Gate: GateEvil,
		Role: "life-draining magic",
		Ranks: []Rank{
			rank(10, "Life Drain", "damages one foe for a Magic Missile's worth and heals the most hurt ally for the damage dealt; chant 1, cost 10", Siphon, 1, SiphonCost, 10),
			rank(15, "Cheaper drain", "Life Drain costs 8", SiphonCost, 8),
			rank(20, "Twin drain", "Life Drain also strikes a second foe and heals a second ally, at 60%", Siphon, 2),
			rank(25, "Dark focus", "spells deal 10% more damage", SpellPct, 10),
		}})
	register(Class{ID: "necromancer", Name: "Necromancer", Lineage: "wizard", Tier: TierElite, Parent: "warlock", Gate: GateEvil,
		Role: "debuffs and draining", Planned: true})

	// ----- Witch -----
	register(Class{ID: "hedge-witch", Name: "Hedge Witch", Lineage: "witch", Tier: TierAdvanced, Gate: GateGood,
		Role: "hexes that also shield the company",
		Ranks: []Rank{
			rank(10, "Warding hex", "a hex that lands also shields the most hurt ally from one blow, up to one average hit of the Witch's level", HexWard, 1, WardBlows, 1),
			rank(15, "Long slumber", "Slumber lasts a round longer", SlumberLong, 1),
			rank(20, "Double ward", "a ward absorbs two blows", WardBlows, 2),
			rank(25, "Steady chant", "blows break the Witch's chant 25% less often", ChantBreak, 25),
		}})
	register(Class{ID: "wise-one", Name: "Wise One", Lineage: "witch", Tier: TierElite, Parent: "hedge-witch", Gate: GateGood,
		Role: "wards and longer holds", Planned: true})
	register(Class{ID: "coven-sage", Name: "Coven Sage", Lineage: "witch", Tier: TierAdvanced, Gate: GateAny,
		Role: "reach: more hex targets and shorter chants",
		Ranks: []Rank{
			rank(10, "Wider hex", "every hex reaches one more foe", HexReach, 1),
			rank(15, "Quick chant", "two-round hexes chant one round less", HexChant, 1),
			rank(20, "Widest hex", "every hex reaches two more foes", HexReach, 2),
			rank(25, "Thrifty hexes", "spells cost 10% less mana", SpellCost, 10),
		}})
	register(Class{ID: "coven-mother", Name: "Coven Mother", Lineage: "witch", Tier: TierElite, Parent: "coven-sage", Gate: GateAny,
		Role: "reach for the whole group", Planned: true})
	register(Class{ID: "hag", Name: "Hag", Lineage: "witch", Tier: TierAdvanced, Gate: GateEvil,
		Role: "curses: hexed foes take more damage",
		Ranks: []Rank{
			rank(10, "Curse", "foes under one of the Hag's hexes take 15% more damage", HexedDamage, 15),
			rank(15, "Cruel hex", "+5 to a hex's chance to land", HexLand, 5),
			rank(20, "Deeper curse", "hexed foes take 25% more damage", HexedDamage, 25),
			rank(25, "Wider hex", "every hex reaches one more foe", HexReach, 1),
		}})
	register(Class{ID: "crone-of-ash", Name: "Crone of Ash", Lineage: "witch", Tier: TierElite, Parent: "hag", Gate: GateEvil,
		Role: "curses that spread", Planned: true})
}
