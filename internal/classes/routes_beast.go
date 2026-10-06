package classes

// The Beast Tamer's lineage (Phase 39e, neutral classes design §4). The
// lineage is neutral from the start: every route is open at any alignment.
// Its base ranks come by level (Sic, Rally, Pack Sense); ranks 10-25 are the
// advanced route, which also chooses the beast (a warhound, a war bear or a
// drake hatchling). The three elites (Packlord, Beastlord, Dragon Lord) are
// named and open with 39i.

// Beast kinds a route raises (BeastKind).
const (
	KindWarhound = 1
	KindBear     = 2
	KindDrake    = 3
)

func init() {
	registerBase("beasttamer",
		rank(1, "Sic", "your turn sends your bonded beast at your target with +10 Attack on its strike, and your whip reaches like a polearm", BeastSic, 1, SicAttack, 10),
		rank(3, "Rally", "heal your beast for a Minor Heal's worth, with no mana, twice a battle", Rally, 2),
		rank(8, "Pack sense", "+5 Evasion while your beast stands", PackSense, 5),
	)

	register(Class{ID: "houndmaster", Name: "Houndmaster", Lineage: "beasttamer", Tier: TierAdvanced, Gate: GateAny,
		Role: "a warhound whose bites hobble wounded foes",
		Ranks: []Rank{
			rank(10, "Warhound", "your beast is a warhound with 110% of a wolf's health; its bites hobble a foe below half health", BeastKind, KindWarhound, BeastHPPct, 110, BeastHobble, 1),
			rank(15, "Keen nose", "+2 Attack on the warhound's bites", BeastAttack, 2),
			rank(20, "Sharp fangs", "+1 damage on the warhound's bites", BeastDamage, 1),
			rank(25, "Heavy build", "the warhound has 130% of a wolf's health", BeastHPPct, 130),
		}})
	register(Class{ID: "packlord", Name: "Packlord", Lineage: "beasttamer", Tier: TierElite, Parent: "houndmaster", Gate: GateAny,
		Role: "a hunting pack: hounds that expose what they hobble, and that strike first",
		Ranks: []Rank{
			rank(30, "Pack hunt", "a warhound's bite that lands on a foe it has hobbled leaves it exposed", BeastHunt, 1),
			rank(35, "Pack leader", "the warhound has 150% of a wolf's health", BeastHPPct, 150),
			rank(40, "Hunting cry", "+4 Attack on the warhound's bites (in all)", BeastAttack, 4),
			rank(45, "Wider hobble", "the warhound's bites hobble a foe below three quarters of its health", BeastHobbleAt, 75),
			rank(50, "Hound of war", "the warhound has 175% of a wolf's health", BeastHPPct, 175),
			rank(55, "Rending fangs", "+3 damage on the warhound's bites (in all)", BeastDamage, 3),
			rank(60, "First strike", "the warhound starts a battle with 50 points on its action meter, so it strikes before the foes do", BeastOpen, 50),
		}})

	register(Class{ID: "bearward", Name: "Bearward", Lineage: "beasttamer", Tier: TierAdvanced, Gate: GateAny,
		Role: "a war bear that guards the Tamer and the company",
		Ranks: []Rank{
			rank(10, "War bear", "your beast is a war bear with 120% of a wolf's health; it guards your most hurt ally (you included) three times a battle", BeastKind, KindBear, BeastHPPct, 120, BeastGuards, 3),
			rank(15, "Thick hide", "+1 damage on the bear's blows", BeastDamage, 1),
			rank(20, "Old bear", "the bear has 140% of a wolf's health", BeastHPPct, 140),
			rank(25, "Steady guard", "the bear guards one more time a battle", BeastGuards, 4),
		}})
	register(Class{ID: "beastlord", Name: "Beastlord", Lineage: "beasttamer", Tier: TierElite, Parent: "bearward", Gate: GateAny,
		Role: "a bear whose swipes hit two foes, and that stands back up",
		Ranks: []Rank{
			rank(30, "Cleaving swipe", "the bear's blows also strike the foe beside its target for half the damage", BeastSwipe, 50),
			rank(35, "Great bear", "the bear has 160% of a wolf's health", BeastHPPct, 160),
			rank(40, "Heavy paws", "+3 damage on the bear's blows (in all)", BeastDamage, 3),
			rank(45, "Faithful guard", "the bear guards six times a battle", BeastGuards, 6),
			rank(50, "Raking swipe", "the bear's second foe takes three quarters of the damage", BeastSwipe, 75),
			rank(55, "Elder bear", "the bear has 190% of a wolf's health", BeastHPPct, 190),
			rank(60, "Stands again", "once a battle, a bear that falls stands back up with half its health", BeastRise, 50),
		}})

	register(Class{ID: "dragon-tamer", Name: "Dragon Tamer", Lineage: "beasttamer", Tier: TierAdvanced, Gate: GateAny,
		Role: "a drake hatchling that breathes fire on a foe and its neighbours",
		Ranks: []Rank{
			rank(10, "Drake hatchling", "your beast is a drake hatchling; every 3 rounds it breathes fire on its foe and up to two foes beside it", BeastKind, KindDrake, BeastBreath, 3),
			rank(15, "Scorching breath", "+2 damage on the drake's bites and its Breath", BeastDamage, 2),
			rank(20, "Growing drake", "the drake has 120% of a wolf's health", BeastHPPct, 120),
			rank(25, "Quick fire", "Breath comes every 2 rounds", BeastBreathCut, 1),
		}})
	register(Class{ID: "dragon-lord", Name: "Dragon Lord", Lineage: "beasttamer", Tier: TierElite, Parent: "dragon-tamer", Gate: GateAny,
		Role: "a larger drake whose breath leaves foes burning, and comes at once",
		Ranks: []Rank{
			rank(30, "Searing breath", "foes the drake's Breath strikes are left alight (2 damage a round for 4 rounds)", BreathBurn, 1),
			rank(35, "Greater drake", "the drake has 150% of a wolf's health", BeastHPPct, 150),
			rank(40, "Dragonfire", "+4 damage on the drake's bites and its Breath (in all)", BeastDamage, 4),
			rank(45, "Wide breath", "Breath reaches four foes", BreathFoes, 4),
			rank(50, "Drake ascendant", "the drake has 180% of a wolf's health", BeastHPPct, 180),
			rank(55, "Razor claws", "+4 Attack on the drake's bites", BeastAttack, 4),
			rank(60, "Furnace heart", "the drake's first Breath comes in the battle's first round", BreathFirst, 1),
		}})
}
