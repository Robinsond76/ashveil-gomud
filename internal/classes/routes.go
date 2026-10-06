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
		Role: "pressure on the chosen target: marks it for the whole company",
		Ranks: []Rank{
			rank(30, "Marked for Ruin", "a foe the Warlord's Tackle lands on is marked for 2 rounds; every ally has +5 Attack against it", MarkRuin, 5),
			rank(35, "Battle Cry", "at the start of each battle, every ally has +3 Attack for 2 rounds", BattleCry, 3),
			rank(40, "Quicker tackle", "Tackle is ready another round sooner", TackleCD, 2),
			rank(45, "Sunder", "Tackle also breaks the target's armor for 2 rounds", Sunder, 1),
			rank(50, "Ruinous mark", "Marked for Ruin gives +10 Attack", MarkRuin, 10),
			rank(55, "Relentless", "when a foe it knocked down stands up, the Warlord's action meter gains 25", Relentless, 25),
			rank(60, "Warlord's Command", "once a battle, when an ally falls, every standing ally's action meter gains 25", WarCommand, 25),
		}})
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
			rank(35, "Aura of Dread", "foes aimed at the Dread Knight have -1 Attack", AuraDread, 1),
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
			teaches(rank(30, "Grove", "Rejuvenation on a whole formation row at 100% each, half of it healed at once; chant 1, cost 8", Grove, 1, GrovePct, 100), "grove"),
			rank(35, "Nature's patience", "after-battle patching costs 20% less mana", PatchCost, 20),
			rank(40, "Deep grove", "Grove at 130%", GrovePct, 130),
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
		Role: "reconnaissance: fewer ambushes and openings on foes caught unprepared",
		Ranks: []Rank{
			rank(30, "Pathfinder's Eye", "enemy ambushes on the company happen half as often, and Opening Strike opens one foe that hasn't acted yet each battle", PathEye, 1, PathOpens, 1),
			rank(35, "Expose Weakness", "an Opening Strike that hits leaves the target exposed for 2 rounds", ExposeWeak, 1),
			rank(40, "Scouted ground", "every ally's action meter starts the battle 10 points higher", ScoutMeter, 10),
			rank(45, "Vanish", "once a battle, when struck below 30% health, it has +20 Evasion until its next turn", Vanish, 20),
			rank(50, "Double opening", "Pathfinder's Eye opens a foe that hasn't acted yet twice a battle", PathOpens, 2),
			rank(55, "Trailwise", "the cache a won random encounter leaves holds 15% more gold and is 10 points likelier to hold equipment", TrailGold, 15, TrailLoot, 10),
			rank(60, "Ambush Master", "when an ambush would catch the company off guard, the company ambushes instead", AmbushFlip, 1),
		}})
	register(Class{ID: "duelist", Name: "Duelist", Lineage: "rogue", Tier: TierAdvanced, Gate: GateAny,
		Role: "precise blade fighting; answers a parry with a blow",
		Ranks: []Rank{
			rank(10, "Riposte", "a blow it parries is answered at once with a counter-blow the size of an Opening Strike", Riposte, 1),
			rank(15, "Parrying drill", "+4 parry chance", Parry, 4),
			rank(20, "Sharper riposte", "the counter-blow is half again as strong", Riposte, 2),
			rank(25, "Duelist's edge", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "swordmaster", Name: "Swordmaster", Lineage: "rogue", Tier: TierElite, Parent: "duelist", Gate: GateAny,
		Role: "parry-based ripostes that grow harder and more frequent",
		Ranks: []Rank{
			rank(30, "Blade Dance", "ripostes deal 50% more damage", RipostePct, 50),
			rank(35, "Parry mastery", "+5 parry chance", Parry, 9),
			rank(40, "Disarming Riposte", "a riposte that crits staggers the target, which loses its next action", RiposteDaze, 1),
			rank(45, "Twin ripostes", "it can riposte twice a round", RiposteRound, 2),
			rank(50, "Perfect Parry", "once a battle, it parries the next melee blow automatically", PerfectParry, 1),
			rank(55, "Counter-strike", "a riposte also leaves its target exposed for a round", RiposteExpo, 1),
			rank(60, "Unbroken Guard", "no riposte limit while it is above half health", RiposteFree, 1),
		}})
	register(Class{ID: "assassin", Name: "Assassin", Lineage: "rogue", Tier: TierAdvanced, Gate: GateEvil,
		Role: "finishing blows against weakened foes",
		Ranks: []Rank{
			rank(10, "Finisher", "blows against a foe at or below 40% health deal an Opening Strike's damage more", Finisher, 1),
			rank(15, "Deep cuts", "Opening Strike deals +2 damage", OpenBonus, 2),
			rank(20, "Killing eye", "Finisher works on a foe at or below 50% health", Finisher, 2),
			rank(25, "Cold edge", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "nightblade", Name: "Nightblade", Lineage: "rogue", Tier: TierElite, Parent: "assassin", Gate: GateEvil,
		Role: "a marked victim, finishing blows and poison",
		Ranks: []Rank{
			rank(30, "Death Mark", "its first target each battle is marked and takes 25% more damage from it; when that foe falls, the mark passes to the most hurt foe in reach", DeathMark, 25),
			rank(35, "Envenom", "its blows poison the target a quarter of the time", Envenom, 25),
			rank(40, "Keener finish", "Finisher works on a foe at or below 60% health", Finisher, 3),
			rank(45, "Shadowstep", "once every 3 rounds it may strike the marked foe in the middle row as if it had extended reach; guardians can still intercept", Shadowstep, 3),
			rank(50, "Hunting the mark", "+15% critical chance against the marked foe", DeathCrit, 15),
			rank(55, "Killing Spree", "when it fells the marked foe, its action meter gains 50, once a round", Spree, 50),
			rank(60, "Coup de Grace", "a blow that lands on a foe below 20% health fells it outright; against a boss it deals double damage instead", Coup, 20),
		}})

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
		Role: "ranged protection: shoots the foe that strikes a vulnerable ally",
		Ranks: []Rank{
			rank(30, "Overwatch", "the Sentinel holds its turn when a foe can reach its middle or back row; the first foe that strikes one of those allies is shot before the blow resolves, and on a hit the blow lands at half damage", Overwatch, 1, OverwatchMax, 1),
			rank(35, "Steady overwatch", "the Overwatch shot has +5 Attack", OverwatchAtk, 5),
			rank(40, "Pull down", "an Overwatch hit knocks down a leaping foe", OverwatchDwn, 1),
			rank(45, "Watchful", "back-row allies have +5 Evasion while the Sentinel stands", WatchBack, 5),
			rank(50, "Spoil the chant", "Overwatch also answers a foe beginning a chant; a hit breaks the chant", OverwatchCh, 1),
			rank(55, "Twin watch", "Overwatch can fire twice a round; the second shot spends the Sentinel's next turn too", OverwatchMax, 2),
			rank(60, "Guardian Arrow", "an Overwatch hit stops the blow entirely", GuardArrow, 1),
		}})
	register(Class{ID: "hunter", Name: "Hunter", Lineage: "ranger", Tier: TierAdvanced, Gate: GateAny,
		Role: "accurate ranged focus and a harder Aimed Shot",
		Ranks: []Rank{
			rank(10, "Hunter's eye", "Aimed Shot deals +3 damage", AimBonus, 3),
			rank(15, "Quick draw", "Aimed Shot is ready a round sooner", AimCD, 1),
			rank(20, "True aim", "+3 Attack", Attack, 3),
			rank(25, "Killing shot", "Aimed Shot deals +5 damage", AimBonus, 5),
		}})
	register(Class{ID: "marksman", Name: "Marksman", Lineage: "ranger", Tier: TierElite, Parent: "hunter", Gate: GateAny,
		Role: "the critical-hit archer: a deadlier Aimed Shot",
		Ranks: []Rank{
			rank(30, "Called Shot", "its shots have +15% critical chance, and its critical hits deal 50% more damage", RangedCrit, 15, CritDamage, 50),
			rank(35, "Quicker aim", "Aimed Shot is ready another round sooner", AimCD, 2),
			rank(40, "Pinning Crit", "an Aimed Shot that lands hobbles the target for 2 rounds", PinCrit, 2),
			rank(45, "Back-line eye", "+10 Attack against foes in the back row", BackAttack, 10),
			rank(50, "Unblockable", "its critical hits can't be blocked", NoBlockCrit, 1),
			rank(55, "Second Nock", "an Aimed Shot that fells its target shoots again at a new target, once a round", SecondNock, 1),
			rank(60, "Perfect Shot", "its first Aimed Shot each battle can't miss or be avoided", PerfectShot, 1),
		}})
	register(Class{ID: "stalker", Name: "Stalker", Lineage: "ranger", Tier: TierAdvanced, Gate: GateEvil,
		Role: "pressure on wounded targets",
		Ranks: []Rank{
			rank(10, "Pursuit", "blows against a foe at or below half health deal 20% more", Wounded, 20),
			rank(15, "Silent step", "+2 Evasion", Evasion, 2),
			rank(20, "Relentless", "Pursuit deals 30% more", Wounded, 30),
			rank(25, "Predator's eye", "+3 Attack", Attack, 3),
		}})
	register(Class{ID: "ravager", Name: "Ravager", Lineage: "ranger", Tier: TierElite, Parent: "stalker", Gate: GateEvil,
		Role: "pursuit: bleeding wounds, and a harder flight for the hunted",
		Ranks: []Rank{
			rank(30, "Hunt Down", "its blows open a bleeding wound on a foe at or below 75% health and add a stack to one already bleeding; a foe it struck this round has 25 points less chance to lose its nerve and flee (it can still flee)", HuntDown, 75, FleePenalty, 25),
			rank(35, "Bloodscent", "+20% damage against bleeding foes", HuntBleed, 20),
			rank(40, "Harrow", "when a foe it wounded loses its nerve, its whole group is rattled: every ally has +5 Attack against them for 2 rounds", Harrow, 5),
			rank(45, "Deep wounds", "bleeding it causes lasts a round longer", BleedLong, 1),
			rank(50, "Rend", "its critical hits also break the target's armor for 2 rounds", RendArmor, 1),
			rank(55, "Relentless hunt", "Hunt Down's flee penalty is 40 points", FleePenalty, 40),
			rank(60, "Apex", "each foe it fells forces a morale check on that foe's group", Apex, 1),
		}})

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

	// ----- Samurai (Phase 39b: a neutral lineage, no alignment gates) -----
	registerBase("samurai",
		rank(1, "Iaijutsu", "starts a battle with half a turn on its action meter (it acts sooner), and its first strike each battle deals 50% more damage with +10% critical chance", Iai, 1, IaiDamage, 50, IaiCrit, 10, OpenMeter, 50),
		rank(3, "Focus", "+3% critical chance for each round in which no blow lands on it, up to +9%; a blow that lands resets it", Focus, 3, FocusMax, 9),
		rank(8, "Zanshin", "when it fells a foe, its action meter gains half a turn (once a round)", Zanshin, 50),
	)
	register(Class{ID: "kensai", Name: "Kensai", Lineage: "samurai", Tier: TierAdvanced, Gate: GateAny,
		Role: "a master of the first cut, and of a patient, deepening focus",
		Ranks: []Rank{
			rank(10, "Piercing draw", "Iaijutsu also ignores half of the target's armor", IaiPierce, 50),
			rank(15, "Clean cut", "+2 Attack", Attack, 2),
			rank(20, "Deep focus", "Focus builds up to +20%", FocusMax, 20),
			rank(25, "Opening edge", "Iaijutsu deals 75% more damage", IaiDamage, 75),
		}})
	register(Class{ID: "sword-saint", Name: "Sword Saint", Lineage: "samurai", Tier: TierElite, Parent: "kensai", Gate: GateAny,
		Role: "Focus to +25%, and Iaijutsu that strikes twice", Planned: true})
	register(Class{ID: "hatamoto", Name: "Hatamoto", Lineage: "samurai", Tier: TierAdvanced, Gate: GateAny,
		Role: "a sworn bodyguard who stands in for the company leader",
		Ranks: []Rank{
			rank(10, "Bodyguard", "steps in front of the company leader twice a battle, like a guardian (a standing swordsman in reach)", Bodyguard, 2),
			rank(15, "Standard bearer", "allies in its row gain +3 Evasion", AuraEvade, 3),
			rank(20, "Loyal blade", "Bodyguard: 3 times a battle", Bodyguard, 3),
			rank(25, "Shield wall", "allies in its row take 5% less damage", AuraResolv, 5),
		}})
	register(Class{ID: "shogun", Name: "Shogun", Lineage: "samurai", Tier: TierElite, Parent: "hatamoto", Gate: GateAny,
		Role: "a commander whose whole company starts a battle sooner", Planned: true})
	register(Class{ID: "ronin", Name: "Ronin", Lineage: "samurai", Tier: TierAdvanced, Gate: GateAny,
		Role: "a masterless blade that grows fiercer as the company falls",
		Ranks: []Rank{
			rank(10, "Vengeance", "+10% damage for each fallen ally in the company", Vengeance, 10),
			rank(15, "Grim resolve", "+2 Attack", Attack, 2),
			rank(20, "Deeper vengeance", "+15% damage for each fallen ally", Vengeance, 15),
			rank(25, "Lone blade", "+3 Evasion", Evasion, 3),
		}})
	register(Class{ID: "kenshi", Name: "Kenshi", Lineage: "samurai", Tier: TierElite, Parent: "ronin", Gate: GateAny,
		Role: "vengeance that cannot be knocked down when it stands alone", Planned: true})

	// ----- Shaman (Phase 39c: a neutral lineage, no alignment gates) -----
	register(Class{ID: "stormcaller", Name: "Stormcaller", Lineage: "shaman", Tier: TierAdvanced, Gate: GateAny,
		Role: "lightning that leaps from one foe to the next",
		Ranks: []Rank{
			rank(10, "Chain lightning", "Lightning also strikes a second foe for 50% of its damage", Chain, 50),
			rank(15, "Thunderhead", "+10% spell damage", SpellPct, 10),
			rank(20, "Forked lightning", "the second foe takes 75% of Lightning's damage", Chain, 75),
			rank(25, "Stormborn", "+20% spell damage", SpellPct, 20),
		}})
	register(Class{ID: "tempest-lord", Name: "Tempest Lord", Lineage: "shaman", Tier: TierElite, Parent: "stormcaller", Gate: GateAny,
		Role: "Rain that lasts the whole battle, and lightning that chains through a row", Planned: true})
	register(Class{ID: "mistweaver", Name: "Mistweaver", Lineage: "shaman", Tier: TierAdvanced, Gate: GateAny,
		Role: "a fog that hides the company and weather that lingers",
		Ranks: []Rank{
			rank(10, "Veil of mist", "while its Fog lasts, allies gain +5 Evasion", FogEvade, 5),
			rank(15, "Long mist", "its weather calls last a round longer", WeatherLong, 1),
			rank(20, "Deep veil", "Fog gives allies +8 Evasion", FogEvade, 8),
			rank(25, "Lingering weather", "its weather calls last two rounds longer", WeatherLong, 2),
		}})
	register(Class{ID: "veil-mother", Name: "Veil Mother", Lineage: "shaman", Tier: TierElite, Parent: "mistweaver", Gate: GateAny,
		Role: "weather that lasts five rounds, and a fog that hides the back row", Planned: true})
	register(Class{ID: "earthspeaker", Name: "Earthspeaker", Lineage: "shaman", Tier: TierAdvanced, Gate: GateAny,
		Role: "turns the earth's armor on an ally",
		Ranks: []Rank{
			teaches(rank(10, "Stoneskin", "one ally gains +10 armor for the battle; chant 1, cost 8", Stoneskin, 10), "stoneskin"),
			rank(15, "Hard earth", "Stoneskin gives +15 armor", Stoneskin, 15),
			rank(20, "Steady ground", "allies in its row take 5% less damage", AuraResolv, 5),
			rank(25, "Deep roots", "Stoneskin gives +20 armor", Stoneskin, 20),
		}})
	register(Class{ID: "mountain-speaker", Name: "Mountain Speaker", Lineage: "shaman", Tier: TierElite, Parent: "earthspeaker", Gate: GateAny,
		Role: "Stoneskin over a whole row, and a tremor that knocks foes down", Planned: true})
}
