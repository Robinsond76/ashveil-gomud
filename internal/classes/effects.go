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

	// The rogue and ranger elites (Phase 38c2).
	PathEye      = "patheye"      // the Pathfinder: enemy ambushes on the company happen half as often
	PathOpens    = "pathopens"    // Opening Strike opens a foe that hasn't acted yet, this many times a battle
	ExposeWeak   = "exposeweak"   // an Opening Strike that hits leaves the target exposed for 2 rounds
	ScoutMeter   = "scoutmeter"   // every ally's opening action meter starts this much higher
	Vanish       = "vanish"       // once a battle, struck below 30% health: this much Evasion until its next turn
	TrailGold    = "trailgold"    // percent more gold in the cache a won random encounter leaves
	TrailLoot    = "trailloot"    // points added to that cache's chance of holding equipment
	AmbushFlip   = "ambushflip"   // an ambush that would catch the company ambushes instead
	RipostePct   = "ripostepct"   // percent more riposte damage
	RiposteRound = "ripostemax"   // ripostes a round (1 by default)
	RiposteDaze  = "ripostedaze"  // a riposte that crits staggers the target
	RiposteExpo  = "riposteexpo"  // a riposte leaves the target exposed for a round
	RiposteFree  = "ripostefree"  // no riposte limit while above half health
	PerfectParry = "perfectparry" // once a battle the next melee blow is parried automatically
	DeathMark    = "deathmark"    // its first target each battle is marked: percent more damage from it
	Envenom      = "envenom"      // percent chance a blow poisons the target
	Shadowstep   = "shadowstep"   // rounds between steps that strike the marked foe in the middle row
	DeathCrit    = "deathcrit"    // critical chance points against the marked foe
	Spree        = "spree"        // action meter points gained on felling the marked foe
	Coup         = "coup"         // percent of health below which a landed blow fells the foe outright
	Overwatch    = "overwatch"    // the Sentinel may hold its turn to shoot the first foe striking a middle or back row ally
	OverwatchAtk = "overwatchatk" // Attack the Overwatch shot adds
	OverwatchDwn = "overwatchdwn" // an Overwatch hit knocks down a leaping foe
	OverwatchCh  = "overwatchch"  // Overwatch also answers a foe beginning a chant, and a hit breaks it
	OverwatchMax = "overwatchmax" // shots an Overwatch can fire in a round
	GuardArrow   = "guardarrow"   // an Overwatch hit stops the blow entirely
	WatchBack    = "watchback"    // Evasion for back-row allies while the holder stands
	CritDamage   = "critdamage"   // percent more damage on a critical hit
	RangedCrit   = "rangedcrit"   // critical chance points on shooting-weapon blows
	RangedPct    = "rangedpct"    // percent more damage with a shooting weapon
	RangedAttack = "rangedattack" // Attack with a shooting weapon
	PinCrit      = "pincrit"      // an Aimed Shot that lands hobbles the target for this many rounds
	BackAttack   = "backattack"   // Attack against foes in the enemy's back row
	NoBlockCrit  = "noblockcrit"  // its critical hits can't be blocked
	SecondNock   = "secondnock"   // an Aimed Shot that fells its target shoots again, once a round
	PerfectShot  = "perfectshot"  // its first Aimed Shot each battle can't miss or be avoided
	HuntDown     = "huntdown"     // its blows open a bleed on a foe at half health and add stacks to a bleeding one
	HuntBleed    = "huntbleed"    // percent more damage against bleeding foes
	FleePenalty  = "fleepenalty"  // points off a struck foe's chance to lose its nerve and flee
	Harrow       = "harrow"       // Attack allies gain against the group of a foe it wounded that breaks
	BleedLong    = "bleedlong"    // rounds added to the bleeding it causes
	RendArmor    = "rendarmor"    // its critical hits break the target's armor for 2 rounds
	Apex         = "apex"         // each foe it fells forces a morale check on that foe's group

	// The Shaman's lineage (Phase 39c).
	Chain       = "chain"       // percent of a Lightning bolt a second foe takes
	FogEvade    = "fogevade"    // Evasion allies gain while its Fog lasts
	WeatherLong = "weatherlong" // extra rounds its weather calls last
	Stoneskin   = "stoneskin"   // armor its Stoneskin gives one ally for the battle
)
