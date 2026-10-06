package classes

// Effect keys. A route rank sets them (the value replaces an earlier rank's)
// and a talent adds to them. The combat, spell and company code read them by
// these names. Keep a key only while something reads it.
const (
	// Rating and damage modifiers read straight off the character.
	Attack     = "attack"     // Attack rating
	Evasion    = "evasion"    // Evasion rating
	Damage     = "damage"     // flat damage on every landed blow
	Armor      = "armor"      // percent damage reduction, added to worn armor
	Block      = "block"      // block chance, in points
	Parry      = "parry"      // parry chance, in points
	HealthPct  = "healthpct"  // percent more maximum health
	ManaPct    = "manapct"    // percent more maximum mana
	HealPct    = "healpct"    // percent more healing cast
	HealCost   = "healcost"   // percent more mana for heals (a Blood Priest's hungry healing)
	PatchCost  = "patchcost"  // percent less mana for after-battle patching
	ChantBreak = "chantbreak" // percent fewer chant breaks
	ChantEvade = "chantevade" // Evasion while chanting
	SpellPct   = "spellpct"   // percent more spell damage
	SpellCost  = "spellcost"  // percent less mana for spells

	// The Witch's routes.
	HexLand     = "hexland"     // points added to a hex's chance to land
	HexReach    = "hexreach"    // extra foes a hex reaches
	HexChant    = "hexchant"    // rounds off a two-round hex's chant
	HexWard     = "hexward"     // a landed hex shields the most hurt ally
	HexedDamage = "hexeddamage" // percent more damage to foes under a hex
	SlumberLong = "slumberlong" // rounds added to Slumber

	// Ability tuning.
	TackleCD   = "tacklecd"   // rounds off Tackle's cooldown
	OpenCD     = "opencd"     // rounds off Opening Strike's cooldown
	AimCD      = "aimcd"      // rounds off Aimed Shot's cooldown
	OpenBonus  = "openbonus"  // damage added to Opening Strike
	AimBonus   = "aimbonus"   // damage added to Aimed Shot
	TackleHit  = "tacklehit"  // points added to Tackle's chance
	TackleHold = "tacklehold" // rounds Tackle's knockdown lasts beyond its own
	TackleExpo = "tacklexpo"  // a tackled foe is also left exposed

	// The Warlord's pressure on the chosen target (Phase 38c1).
	MarkRuin   = "markruin"   // a foe its Tackle lands on is marked for 2 rounds: allies have this much Attack against it
	BattleCry  = "battlecry"  // every ally has this much Attack for the battle's first 2 rounds
	Sunder     = "sunder"     // a Tackle also breaks the target's armor for 2 rounds
	Relentless = "relentless" // action meter points gained when a foe it knocked down stands up
	WarCommand = "warcommand" // once a battle, when an ally falls, every standing ally's meter gains this much
	SecondWind = "secondwind" // once a battle, below 25% health at the start of its turn, heals this percent of maximum health

	// Conditional blows and auras.
	Ambush     = "ambush"     // Opening Strike opens any foe in the battle's first N rounds
	Riposte    = "riposte"    // a parried blow is answered at once (1: Opening Strike size, 2: half again)
	Finisher   = "finisher"   // extra damage to a foe at or below a share of its health (1: 40%, 2: 50%)
	Wounded    = "wounded"    // percent more damage to a foe at or below half health
	Smite      = "smite"      // percent more damage to undead and demons
	RendHoly   = "rendholy"   // percent more damage to holy creatures
	AuraEvade  = "auraevade"  // Evasion given to allies in the holder's row
	AuraResolv = "auraresolv" // percent less damage to allies in the holder's row
	AuraCompan = "auracompan" // percent less damage to the rest of the company
	AuraDread  = "auradread"  // Attack lost by foes aimed at the holder

	// Wards: one blow absorbed, up to a share of a hit.
	Ward      = "ward"      // knows Ward (Priest) or Arcane Ward (Theurgist)
	WardBlows = "wardblows" // blows a ward absorbs

	// Cleric lineage spells and over-time heals.
	GreaterHeal  = "greaterheal"  // knows Greater Heal
	Rejuvenation = "rejuvenation" // rounds of Rejuvenation
	RejuvPct     = "rejuvpct"     // percent of a Minor Heal it heals over its rounds
	Barkskin     = "barkskin"     // Barkskin armor
	BarkRow      = "barkrow"      // rows a Barkskin covers
	Thornhide    = "thornhide"    // damage a foe takes for striking a Barkskinned ally
	Grove        = "grove"        // rows a Grove covers
	GrovePct     = "grovepct"     // percent each ally of a Grove heals at
	Entangle     = "entangle"     // knows Entangle
	WildGrowth   = "wildgrowth"   // Rejuvenation also cures poison
	Siphon       = "siphon"       // Siphon's reach in foes
	SiphonCost   = "siphoncost"   // Siphon's mana cost
	BloodWard    = "bloodward"    // overheal from Siphon becomes a ward
	CleanseHeal  = "cleanseheal"  // a heal also removes one harmful status, once per patient per battle

	// Summons.
	Summon         = "summon"      // 1: the Hierarch's Angel, 2: the Demonologist's Demon
	SummonArmor    = "summonarmor" // the summon's armor
	SummonSooner   = "summonfast"  // chant rounds off the summon
	AngelMercy     = "angelmercy"  // rounds between the Angel's Mercy heals
	AngelMercyFull = "angelfull"   // Mercy is a full Minor Heal
	AngelMercyTwo  = "angeltwo"    // Mercy heals the two most hurt allies
	AngelGuards    = "angelguards" // guards a battle
	AngelBlade     = "angelblade"  // blade dice sides
	AngelWings     = "angelwings"  // Evasion to allies in the Angel's row
	AngelCleanse   = "angelclean"  // the Angel removes a status from every ally on arrival
	DemonDread     = "demondread"  // 1: Dread checks the target's group, 2: every enemy group
	DemonHellfire  = "hellfire"    // damage foes in the Demon's reach take each round
	DemonClaws     = "demonclaws"  // claw dice sides
	DemonFeast     = "soulfeast"   // the Demon's kills heal and refund mana
	DemonMastered  = "mastered"    // a falling Demonologist's Demon simply vanishes

	// Fighting healers.
	LayHands     = "layhands"     // Lay on Hands uses per rest
	LayFull      = "layfull"      // heals a full Minor Heal and removes a status
	LayBleed     = "laybleed"     // Lay on Hands stops bleeding
	LayReach     = "layreach"     // reaches any ally, not only adjacent ones
	FaithBlock   = "faithblock"   // block chance added while guarding a ward
	DivineShield = "divineshield" // once a battle the next blow is ignored
	BloodOath    = "bloodoath"    // blows a battle that heal
	OathPct      = "oathpct"      // percent of the damage dealt healed
	OathSecond   = "oathsecond"   // also heals the next most hurt ally for half
	Intimidate   = "intimidate"   // Attack a wounded foe loses against its allies
	TerrorCrit   = "terrorcrit"   // a critical hit that lands also staggers the target

	// The Halberdier's routes (Phase 39a). Hook and the wider Sweep are the
	// lineage's own, by level, so they have no key.
	SweepPct    = "sweeppct"    // percent of a blow's damage each foe a Sweep strikes takes (a Sweeper's 100, else 90)
	SweepCD     = "sweepcd"     // rounds off Sweep's cooldown
	BraceCol    = "bracecol"    // Brace answers a blow at anyone in the Halberdier's column, not only itself
	BraceDown   = "bracedown"   // the held blow knocks the foe down when it hits
	ChargedMana = "chargedmana" // mana a Charged Sweep spends for lightning on every foe it strikes
	ChargedDice = "chargeddice" // sides of the Charged Sweep's lightning die (1dN)
	// The Samurai's lineage (Phase 39b).
	Iai       = "iai"       // knows Iaijutsu: a stronger, surer first strike each battle
	IaiDamage = "iaidamage" // percent more damage on the first strike
	IaiCrit   = "iaicrit"   // critical chance points on the first strike
	IaiPierce = "iaipierce" // percent of the target's armor the first strike ignores
	OpenMeter = "openmeter" // action meter points the character starts a battle with
	Focus     = "focus"     // critical chance points gained each round no blow lands on it
	FocusMax  = "focusmax"  // the most Focus can add
	Zanshin   = "zanshin"   // action meter points gained when it fells a foe, once a round
	Crit      = "crit"      // critical chance points on every blow
	Bodyguard = "bodyguard" // times a battle it steps in for the company leader
	Vengeance = "vengeance" // percent more damage for each fallen ally

	// The Doll Master's lineage (Phase 39d). The doll is a durable fighter
	// in a cell of its own that acts on its Master's turn.
	DollGuards  = "dollguards"  // times a battle the doll guards its most hurt neighbour (Guard String)
	DollTangle  = "tangle"      // knows Tangle: strings snag a foe and push its action meter back
	TangleFoes  = "tanglefoes"  // foes a Tangle snags (one when absent)
	TangleCD    = "tanglecd"    // rounds off Tangle's cooldown
	TanglePush  = "tanglepush"  // action meter points added to a Tangle's push
	Splice      = "splice"      // knows Emergency Splice: a doll that would break stands back up once a battle
	SplicePct   = "splicepct"   // percent of its health a spliced doll stands back up at (25 when absent)
	DollCount   = "dollcount"   // dolls beyond the first
	DollHPPct   = "dollhppct"   // each doll's health as a percent of the standard doll's (100 when absent)
	DollHPBonus = "dollhpbonus" // percent points added to that
	DollArmor   = "dollarmor"   // armor each doll carries on its own body
	DollNoWear  = "dollnowear"  // a doll that cannot wear armor (a golem's body is its armor)
	DollAttack  = "dollattack"  // Attack added to each doll's blows
	DollDamage  = "dolldamage"  // damage added to each doll's blows
)
