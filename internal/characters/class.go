package characters

import (
	"slices"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/classes"
	"github.com/GoMudEngine/GoMud/internal/hexes"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// Phase 38b: a character's class (the advanced or elite route promoted into
// from its archetype) and talents. A player's are the archetype registry's,
// a companion's are copied onto its mob by the company module; everything
// they give is derived from them and the character's current level.

// SetClassState records a companion's class and talents on its runtime
// character; the company record is the durable source.
func (c *Character) SetClassState(class string, talents []string) {
	c.HPClass = class
	c.HPTalents = slices.Clone(talents)
}

// ClassState is the character's class id ("" before promotion) and talents.
func (c *Character) ClassState() (string, []string) {
	if c == nil {
		return "", nil
	}
	if c.userId > 0 {
		s := classes.PlayerClass(c.userId)
		return s.Class, s.Talents
	}
	return c.HPClass, c.HPTalents
}

// ClassEffects are the character's class benefits at its current level:
// its route's ranks reached and the talents it has earned. Nil for a
// character with neither.
func (c *Character) ClassEffects() classes.Effects {
	own := c.classOwnEffects()
	if c == nil {
		return own
	}
	key, gear := c.wornGear()
	if key == "" {
		return own
	}
	// Phase 36d: worn relics add their signature and set bonuses on top.
	// classOwnEffects and wornGear drop mergedFx whenever either side is
	// remade, so the merge is redone only then.
	if c.mergedFx == nil {
		c.mergedFx = classes.WithGear(own, gear)
	}
	return c.mergedFx
}

// WornGear is what the character's worn relics grant (Phase 36d): the
// gear effects and each set's progress. Empty for a character in plain gear.
func (c *Character) WornGear() (map[string]int, []items.ActiveSet) {
	key, fx := c.wornGear()
	if key == "" {
		return nil, nil
	}
	return fx, c.gearSets
}

// wornGear returns a key naming the relics worn ("" when none) and the
// effects they grant, recomputed only when the relics worn change.
func (c *Character) wornGear() (string, map[string]int) {
	key := ""
	for _, slot := range AllSlots() {
		if slot == items.Pack {
			continue
		}
		if it := c.Equipment.Get(slot); it.ItemId > 0 && items.IsRelicItem(it.ItemId) {
			key += strconv.Itoa(it.ItemId) + ":" + strconv.Itoa(it.AwakenedMask()) + "," // Phase 67: a waking relic changes the key
		}
	}
	if key == "" {
		c.gearKey = ""
		return "", nil
	}
	if key != c.gearKey {
		var worn []items.Item
		for _, slot := range AllSlots() {
			if slot != items.Pack {
				worn = append(worn, *c.Equipment.Get(slot))
			}
		}
		c.gearFx, c.gearSets = items.GearEffects(worn)
		c.gearKey = key
		c.mergedFx = nil
	}
	return key, c.gearFx
}

func (c *Character) classOwnEffects() classes.Effects {
	class, talents := c.ClassState()
	// Phase 39b: a neutral lineage's base ranks (the Samurai's Iaijutsu)
	// count from level 1, before any promotion.
	lineage := ""
	if c != nil {
		if id := c.ArchetypeID(); classes.HasBase(id) {
			lineage = id
		}
	}
	if class == "" && len(talents) == 0 && lineage == "" {
		if c != nil && c.fxValid {
			c.fxValid, c.mergedFx = false, nil // 36d review: no class now, so no merge on the old one
		}
		return nil
	}
	if c.fxValid && c.fxClass == class && c.fxLineage == lineage && c.fxLevel == c.Level && slices.Equal(c.fxTalents, talents) {
		return c.fx
	}
	c.fx = classes.EffectsForLineage(lineage, class, c.Level, talents)
	c.mergedFx = nil // 36d review: the gear merge sits on top of the old map
	c.fxClass, c.fxLineage, c.fxLevel, c.fxTalents, c.fxValid = class, lineage, c.Level, slices.Clone(talents), true
	return c.fx
}

// classPct raises a value by a percent class effect.
func classPct(value, pct int) int {
	if pct == 0 {
		return value
	}
	return value + (value*pct+50)/100
}

// SpellCost is a spell's mana cost for this character (Phase 38b): the
// spell's own, less the class's spell discount, and for a restoration
// spell more by the class's heal cost. A spell never costs less than 1.
func (c *Character) SpellCost(sp *spells.SpellData) int {
	if sp == nil {
		return 0
	}
	cost := sp.Cost
	if sp.SpellId == "callhost" || sp.SpellId == "bindfiend" || sp.SpellId == "raisefallen" {
		return max(1, c.ManaMax.Value/10) // a summon costs a tenth of the mana
	}
	fx := c.ClassEffects()
	if cost <= 0 || fx == nil {
		return cost
	}
	if sp.SpellId == "siphon" {
		// Siphon is dark healing's efficient heal (Phase 38b review): its
		// rank sets its cost and the hungry heal tax doesn't apply.
		if n := fx.Int(classes.SiphonCost); n > 0 {
			cost = n
		}
		return max(1, cost-(cost*fx.Int(classes.SpellCost)+50)/100)
	}
	if sp.SpellId == "arcanelance" {
		// Phase 38d: the Lance's rank sets its cost, then the spell discount.
		if n := fx.Int(classes.LanceCost); n > 0 {
			cost = n
		}
	}
	pct := fx.Int(classes.SpellCost)
	// Phase 38c3: a Coven Mother's hexes cost less again.
	if _, isHex := hexes.For(sp.SpellId); isHex {
		pct += fx.Int(classes.HexCost)
	}
	if pct != 0 {
		cost -= (cost*pct + 50) / 100
	}
	if sp.School == spells.SchoolRestoration {
		cost = classPct(cost, fx.Int(classes.HealCost))
	}
	return max(1, cost)
}

// HealCostPct is the percent a character's healing spells cost more
// (negative for less) when its company patches itself: its class's heal
// cost and spell discount, and for patching its patch discount.
func (c *Character) HealCostPct(patching bool) int {
	fx := c.ClassEffects()
	pct := fx.Int(classes.HealCost) - fx.Int(classes.SpellCost)
	if patching {
		pct -= fx.Int(classes.PatchCost)
	}
	return pct
}

// ClassAura is what an ally's class aura gives a character for a combat
// round: Evasion points and a percent less damage.
type ClassAura struct {
	Evasion int
	Resolve int
	Block   int // block chance points (a Knight guarding a ward)
	Attack  int // Attack points (a Warlord's Battle Cry)
	Fallen  int // allies of its company that have fallen (a Ronin's Vengeance)
	// Phase 38c3: an Archon's Mana Shield, the percent less damage spells
	// do to its row.
	SpellResolve int
}

// ClassRT is a character's class state for the battle it is in: nothing in
// it is saved, and a fight's end clears all of it but the Lay on Hands
// uses, which come back with rest.
type ClassRT struct {
	Ward, WardCap int // blows a ward absorbs, and the most it takes from each
	// Phase 50: the battle condition the member began this battle in, from
	// its needs and meal buff: percent on the damage it deals, and percent
	// less damage it takes (negative: more, from thirst).
	FareDamage, FareGuard int
	// WardSigil marks a ward a ward sigil gave (Phase 54 review): a caster's
	// own ward replaces it, and healers do not count it as warded.
	WardSigil    bool
	Bark, Thorns int  // Barkskin's armor and the damage a striker takes
	Rejuv, Per   int  // rounds of Rejuvenation left and its heal each round
	ShieldUsed   bool // Divine Shield has been spent this battle
	OathUsed     int  // Blood Oath blows spent this battle
	Intim        int  // Attack this foe loses against anyone but IntimOwner
	IntimOwner   *ClassRT
	IntimRound   uint64 // the combat round the foe was wounded in
	Cleansed     map[string]bool
	Guards       int         // an Angel's Guard uses spent
	Hands        int         // Lay on Hands uses since the last rest
	Summoned     bool        // this character has called its summon this battle
	Summon       *SummonInfo // set on a summoned creature
	Doll         *DollInfo   // set on a Doll Master's doll (Phase 39d)
	Beast        *BeastInfo  // set on a Beast Tamer's bonded beast (Phase 39e)
	HobbledBy    *ClassRT    // on a foe: the beast whose bite last hobbled it (39i2 Pack hunt)
	Bless        int         // rounds of Bless left

	// Phase 38c1, the Warlord and elite talents.
	Mark      int    // on a foe: the Attack every ally has against it (Marked for Ruin)
	MarkRound uint64 // the combat round it was marked in
	Tackled   []int  // foes this Warlord knocked down that have yet to stand (Relentless)
	CmdUsed   bool   // Warlord's Command has been spent this battle
	Standing  int    // company members standing at the last pass (Warlord's Command)
	WindUsed  bool   // Second Wind has been spent this battle
	// Phase 39a: the Halberdier. Brace is a held blow waiting for a foe's
	// strike; BlowPct, when set, scales the damage of the blow being
	// resolved (a Sweep's 90%, a held blow's 125%) and is cleared at once.
	Brace   bool
	BlowPct int
	// BraceUsed counts the held blows a brace has answered (a Linebreaker's
	// Twin brace holds for two).
	BraceUsed int
	// The Samurai's lineage (Phase 39b).
	IaiSpent     bool   // the first strike of the battle has been made (every strike Iaijutsu covers, for a Sword Saint)
	IaiStrikes   int    // strikes that carried Iaijutsu's edge this battle (a Sword Saint's Twin draw)
	Quiet        int    // rounds in a row no blow has landed on it (Focus)
	Struck       bool   // a blow landed on it since the round began
	QuietStarted bool   // the first round's Focus count has begun
	ZanshinRound uint64 // the combat round Zanshin last gave its turn back
	Bodyguards   int    // Bodyguard steps spent this battle
	BondGuards   int    // bond steps spent this battle: a friend stepped in for a friend (Phase 65)
	BondRefused  bool   // a rival's refusal to guard has been told this battle (Phase 65)
	SidePeak     int    // the most of its side standing this battle (Vengeance)
	Alone        bool   // it is the last of its company standing (a Kenshi's Last stand)
	EliteRT             // Phase 38c2: the rogue and ranger elites
	// The Doll Master's lineage (Phase 39d): Guard String uses spent, the
	// Emergency Splice spent this battle, and the Master's next turn owed to
	// it.
	DollGuards   int
	Spliced      bool
	SpliceTurn   bool
	SplicedAgain bool // a Golem Lord's Rise again has been spent (Phase 39i)
	Cut          int  // on a foe: the Attack its next attack loses to a String Sovereign's Cut strings
	// The Arbalist's lineage (Phase 39h): a bolt just loosed leaves its next
	// turn to the winding (Reload); BoltFired is the first bolt of the battle
	// spent; AimStruck is a blow landing on the holder since its last bolt
	// (Steady Aim); BlowPierce, when set, is the percent of the target's armor
	// the blow being resolved ignores and is cleared at once. On a foe,
	// Shred is the armor its bolts have taken off it this battle.
	Reload     bool
	BoltFired  bool
	AimStruck  bool
	BlowPierce int
	Shred      int
	// Phase 39i2: the Siege Master's Ballista bolts passed through this
	// battle, and the Bastion's Covering shots loosed.
	Through  int
	ColShots int
	// Phase 39i2: a Panacean's Elixir. On the Panacean: how many times it has
	// worked this battle, how many it may, and the percent of an ally's health
	// it leaves. On every ally of its side: the Panacean covering it.
	ElixirSpent int
	ElixirMax   int
	ElixirPct   int
	ElixirBy    *ClassRT
	// The Beast Tamer's lineage (Phase 39e): the Attack Sic gives its beast
	// this round, the Evasion Pack Sense gives the Tamer while the beast
	// stands, and the Rally heals spent this battle.
	Sic       int
	PackSense int
	Rallies   int
	// Benched is a wounded beast's sitting-out told this battle (review fix).
	Benched bool

	// Phase 38c3: the Wizard's elites. Overchannel and the once-a-battle
	// gifts of the Archon, Archmage and Necromancer.
	OverRound   uint64 // the combat round of its last Overchannel
	OverSpent   bool   // Overchannel has been used this battle
	StormUsed   bool   // Archmage's Storm has been spent
	LanceFreed  bool   // the High Sorcerer's Instant Lance has been spent
	AegisUsed   bool   // Archon's Aegis has been spent
	ReflectUsed bool   // Reflection has been spent
	Raised      int    // thralls raised this battle
	BargainUsed bool   // Lich's Bargain has been spent
	// A ward's extras, set when a hex or Arcane Ward grants it. A ward that
	// holds none of them is the plain Phase 38b ward.
	WardMend      int      // heal the holder when the ward breaks
	WardCleanse   bool     // remove one harmful status when the ward breaks
	WardPeace     int      // Evasion the holder has while it holds the ward
	WardLifeBy    *ClassRT // a Wise One's Ward of Life: it has not yet been spent
	WardReflectBy *ClassRT // an Archon's Reflection: it has not yet been spent
	// Saved is what kept this character from falling in the blow just
	// resolved ("ward of life" or "bargain"), for the narration to tell.
	Saved string

	QuickCasts int // damage spells or hexes cast this battle (Quick casting and Quick curses trim every other one)

	// Phase 38c3: the Witch's elites.
	HexLands   int          // hexes it has landed this battle (Twin Hex)
	CircleUsed bool         // Coven Circle has been spent
	DoomUsed   bool         // Crone's Doom has been spent
	LifeUsed   bool         // Ward of Life has been spent
	CurseAtk   int          // on a foe: the Attack allies have against it while it is hexed
	CurseDmg   int          // on a foe: percent more damage every ally's blows deal it while it is hexed
	CurseBy    *Character   // on a foe: the Crone whose hex last landed on it
	HexBuffs   map[int]bool // on a foe: the statuses a hex has laid on it this battle
	HexStreak  int          // on a foe: rounds in a row it has been hexed
	LingerAt   uint64       // on a foe: the hex round after which it is left exposed
	SoulRot    bool         // on a foe: its fall forces a morale check on its group
	PoisonX2   bool         // on a foe: its poison deals double damage
}

// WardEvent is what a ward did in a round's strikes (Phase 38c3), for the
// hooks to narrate and finish: the ward ran out of blows, how much of a
// blow Reflection sends back, and what held the target on its feet.
type WardEvent struct {
	Broke   bool
	Reflect int
	Saved   string
}

// RTState is the character's class battle state, made on first use.
func (c *Character) RTState() *ClassRT {
	if c.RT == nil {
		c.RT = &ClassRT{}
	}
	return c.RT
}

// EndFightRT clears the battle's class state, keeping what rest restores.
func (c *Character) EndFightRT() {
	if c.RT == nil {
		return
	}
	c.RT = &ClassRT{Hands: c.RT.Hands}
	c.Aura = ClassAura{}
}

// IaiReady reports whether the character's Iaijutsu is waiting: it knows it
// and has not yet made its first strike of this battle.
func (c *Character) IaiReady() bool {
	return c.RT != nil && !c.RT.IaiSpent && c.ClassEffects().Has(classes.Iai)
}

// ClassCrit is the critical chance points its class adds to a blow now:
// Crit, and Focus built up over the quiet rounds.
func (c *Character) ClassCrit() int {
	fx := c.ClassEffects()
	if fx == nil {
		return 0
	}
	pts := fx.Int(classes.Crit)
	if c.RT != nil && c.RT.Quiet > 0 {
		pts += min(fx.Int(classes.Focus)*c.RT.Quiet, fx.Int(classes.FocusMax))
	}
	return pts
}

// AbsorbWard takes a ward's share of one blow, and returns what is left.
func (c *Character) AbsorbWard(dmg int) (left, absorbed int) {
	if c.RT == nil || c.RT.Ward <= 0 || dmg <= 0 {
		return dmg, 0
	}
	absorbed = min(dmg, max(c.RT.WardCap, 0))
	c.RT.Ward--
	return dmg - absorbed, absorbed
}

// WardAbsorbed finishes a ward's absorption of absorbed points (Phase 38c3):
// whether it has run out of blows, and the damage an Archon's Reflection
// sends back at the attacker (half the absorbed damage), once a battle.
func (c *Character) WardAbsorbed(absorbed int) (broke bool, reflect int) {
	if c.RT == nil || absorbed <= 0 {
		return false, 0
	}
	if by := c.RT.WardReflectBy; by != nil && !by.ReflectUsed {
		by.ReflectUsed = true
		reflect = max(1, absorbed/2)
	}
	return c.RT.Ward <= 0, reflect
}

// GuardFall caps a blow of dmg that would fell the character (its health
// left being room), when a Ward of Life (the ward it held when the blow
// came) or a Lich's Bargain keeps it at 1 health (Phase 38c3). It returns
// the damage to take and what saved it, "" for nothing.
func (c *Character) GuardFall(dmg, room int, warded bool) (int, string) {
	if dmg <= 0 || dmg < room {
		return dmg, ""
	}
	if c.RT == nil {
		// A Necromancer that has cast nothing this battle has no runtime
		// state yet; its Bargain still holds.
		if !c.ClassEffects().Has(classes.Bargain) {
			return dmg, ""
		}
		c.RTState()
	}
	if by := c.RT.WardLifeBy; warded && by != nil && !by.LifeUsed {
		by.LifeUsed = true
		c.RT.Saved = "ward of life"
		return max(0, room-1), c.RT.Saved
	}
	if !c.RT.BargainUsed && c.ClassEffects().Has(classes.Bargain) {
		c.RT.BargainUsed = true
		c.RT.Saved = "bargain"
		return max(0, room-1), c.RT.Saved
	}
	// Phase 39i2: a Panacean's Elixir keeps a falling ally on its feet.
	if by := c.RT.ElixirBy; by != nil && by.ElixirSpent < by.ElixirMax {
		by.ElixirSpent++
		c.RT.Saved = "elixir"
		return max(0, room-1), c.RT.Saved
	}
	return dmg, ""
}

// ShieldBlow spends a Divine Shield on a blow, once a battle.
func (c *Character) ShieldBlow() bool {
	if !c.ClassEffects().Has(classes.DivineShield) {
		return false
	}
	// 36d review: a relic gives a classless wearer the shield, and it may be
	// struck before it has acted (and so before it has runtime state).
	if rt := c.RTState(); rt.ShieldUsed {
		return false
	}
	c.RT.ShieldUsed = true
	return true
}

// Bless is a cleric's blessing: Attack and Evasion points while it lasts.
const BlessPoints = 5

func (c *Character) blessPoints() int {
	if c.RT != nil && c.RT.Bless > 0 {
		return BlessPoints
	}
	return 0
}

// RestClass renews what rest gives back to a class: Lay on Hands uses.
func (c *Character) RestClass() {
	if c.RT != nil {
		c.RT.Hands = 0
	}
}

// SummonInfo is what a summoned creature (a Hierarch's Angel, a
// Demonologist's Demon) is: its owner, and the gifts the owner's route had
// reached when it was called. It lives only for the battle.
type SummonInfo struct {
	Kind                  string // "angel" or "demon"
	OwnerUser, OwnerMob   int    // who called it
	OwnerKey              string // the owner's company member key
	HPPct                 int    // percent of a warrior's health at its level
	HealthPct             int    // percent of the whole health its template works out to (a thrall)
	Dice, Sides           int    // its natural weapon's dice
	Smite, Rend           int    // percent more damage against the unholy, the holy
	Guards, GuardsUsed    int    // Guard uses a battle and spent
	MercyEvery, MercyNext int    // rounds between Mercy heals, and the next one's round
	MercyFull, MercyTwo   bool
	Wings                 int // Evasion given to allies in the owner's row
	Hellfire              int
	Feast, Mastered       bool
	Cleanse               bool // removes a harmful status from every ally on arrival
	Dread                 int  // the Demon's Dread on arrival: 1 its target's group, 2 every group
	Arrived               bool // the arrival gifts are given
}

// KnowsSpell is HasSpell for casting: a class's rank spell counts only
// while the character's level holds that rank (Phase 38b review).
func (c *Character) KnowsSpell(spellID string) bool {
	if !c.HasSpell(spellID) {
		return false
	}
	class, _ := c.ClassState()
	return !classes.SpellLocked(class, c.Level, spellID)
}
