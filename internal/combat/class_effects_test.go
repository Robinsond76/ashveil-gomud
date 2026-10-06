package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/stormcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 38b: a class's effects reach the combat formulas.

func classed(class string, level int) *characters.Character {
	c := characters.New()
	c.Level, c.Health = level, 100
	c.HealthMax.Value = 100
	c.SetClassState(class, nil)
	return c
}

func TestClassBlowDamageIsUntouchedWithoutAClass(t *testing.T) {
	defenseSpecs(t)
	foe := classed("", 10)
	assert.Equal(t, 10, classBlowDamage(classed("", 30), foe, 10))
}

func TestFinisherAddsAnOpeningStrikeAgainstAWoundedFoe(t *testing.T) {
	defenseSpecs(t)
	assassin := classed("assassin", 12)
	foe := classed("", 12)
	foe.Health = 90
	assert.Equal(t, 10, classBlowDamage(assassin, foe, 10), "a healthy foe is not finished")
	foe.Health = 40
	assert.Greater(t, classBlowDamage(assassin, foe, 10), 10, "at 40% it is")
	foe.Health = 50
	assert.Equal(t, 10, classBlowDamage(assassin, foe, 10), "50% needs Killing eye")
	killer := classed("assassin", 20)
	assert.Greater(t, classBlowDamage(killer, foe, 10), 10)
}

func TestPursuitRaisesDamageAgainstAFoeAtHalfHealth(t *testing.T) {
	defenseSpecs(t)
	hunter := classed("stalker", 12)
	foe := classed("", 12)
	foe.Health = 50
	assert.Equal(t, 12, classBlowDamage(hunter, foe, 10), "+20%")
	foe.Health = 51
	assert.Equal(t, 10, classBlowDamage(hunter, foe, 10))
}

func TestHexedFoesTakeMoreFromAHag(t *testing.T) {
	defenseSpecs(t)
	hag := classed("hag", 12)
	foe := classed("", 12)
	assert.Equal(t, 10, classBlowDamage(hag, foe, 10))
	foe.AddBuff(status.Asleep, true)
	assert.Equal(t, 12, classBlowDamage(hag, foe, 10), "+15%, rounded")
}

// Phase 38c3 review: a Tackle's exposure or a weapon's poison is not a hex;
// the same status counts once a hex has laid it on the foe this battle.
func TestSharedStatusesCountAsAHexOnlyWhenAHexLaidThem(t *testing.T) {
	defenseSpecs(t)
	hag := classed("hag", 12)
	foe := classed("", 12)
	foe.AddBuff(status.Exposed, true)
	assert.False(t, hexed(foe), "a tackled foe is not hexed")
	assert.Equal(t, 10, classBlowDamage(hag, foe, 10))
	foe.RTState().HexBuffs = map[int]bool{status.Exposed: true}
	assert.True(t, hexed(foe), "Frailty laid it")
	assert.Equal(t, 12, classBlowDamage(hag, foe, 10))
}

// Phase 38c3 review: Ashen Curse raises every ally's blows against a hexed
// foe, not only the Crone's own.
func TestAshenCurseRaisesEveryAllysBlows(t *testing.T) {
	defenseSpecs(t)
	ally := classed("", 40)
	foe := classed("", 40)
	foe.RTState().CurseDmg = 15
	assert.Equal(t, 10, classBlowDamage(ally, foe, 10), "only while hexed")
	foe.AddBuff(status.Asleep, true)
	assert.Equal(t, 12, classBlowDamage(ally, foe, 10), "+15%, rounded")
	hag := classed("hag", 40)
	assert.Equal(t, 13, classBlowDamage(hag, foe, 10), "a Hag's own 25% counts instead")
}

func TestBlockAndParryAddTheClassPoints(t *testing.T) {
	defenseSpecs(t)
	stockDefenses(t)
	plain, drilled := classed("", 20), classed("duelist", 20)
	atk := classed("", 20)
	assert.Equal(t, parryChance(plain, atk, 5)+4, parryChance(drilled, atk, 5), "Parrying drill is +4")
	assert.Equal(t, blockChance(plain, atk), blockChance(drilled, atk), "a Duelist's block is its own")
}

func TestDreadSpoilsTheAttackersAim(t *testing.T) {
	defenseSpecs(t)
	stockDefenses(t)
	dread := classed("dread-knight", 40)
	plain := classed("", 40)
	atk := classed("", 40)
	assert.Less(t, attackEdge(atk, dread), attackEdge(atk, plain))
}

func TestAurasAddEvasionAndArmor(t *testing.T) {
	defenseSpecs(t)
	c := classed("", 20)
	base, armor := c.Evasion(), c.GetDefense()
	c.Aura = characters.ClassAura{Evasion: 3, Resolve: 10}
	assert.Equal(t, base+3, c.Evasion())
	assert.Equal(t, armor, c.GetDefense(), "Resolve is a percent off the blow, not armor")
}

// Phase 38b review: Aura of Resolve's "10% less damage" is a true 10% off
// a landed blow (on the armor roll it averaged half).
func TestAuraOfResolveTakesItsPercentOffABlow(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	base := strikeAt(armed(edgeSwordID), armed(0))
	require.True(t, base.Hit)
	require.Positive(t, base.DamageToTarget)
	target := armed(0)
	target.Aura = characters.ClassAura{Resolve: 100}
	assert.Zero(t, strikeAt(armed(edgeSwordID), target).DamageToTarget, "all of it")
	target.Aura = characters.ClassAura{Resolve: 50}
	r := strikeAt(armed(edgeSwordID), target)
	assert.Equal(t, base.DamageToTarget-(base.DamageToTarget*50+50)/100, r.DamageToTarget, "half of it")
}

func TestAWardAbsorbsABlowUpToItsSizeAndIsSpent(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	base := strikeAt(armed(edgeSwordID), armed(0))
	require.True(t, base.Hit)

	target := armed(0)
	target.RTState().Ward, target.RTState().WardCap = 1, 1000
	r := strikeAt(armed(edgeSwordID), target)
	assert.True(t, r.Hit)
	assert.Zero(t, r.DamageToTarget, "a big ward takes the whole blow")
	assert.Zero(t, target.RT.Ward, "and is spent")
	assert.Equal(t, base.DamageToTarget, strikeAt(armed(edgeSwordID), target).DamageToTarget, "the next blow lands whole")

	small := armed(0)
	small.RTState().Ward, small.RTState().WardCap = 2, 1
	r = strikeAt(armed(edgeSwordID), small)
	assert.Equal(t, max(0, base.DamageToTarget-1), r.DamageToTarget, "a small ward takes its size")
	assert.Equal(t, 1, small.RT.Ward, "a two-blow ward has one left")
}

func TestDivineShieldIgnoresTheFirstBlowOfABattleOnly(t *testing.T) {
	defenseSpecs(t)
	defenseOdds(t, 0, 0, 0)
	paladin := armed(0)
	paladin.Level = 60
	paladin.SetClassState("paladin", nil)
	paladin.RTState()
	assert.Zero(t, strikeAt(armed(edgeSwordID), paladin).DamageToTarget)
	assert.True(t, paladin.RT.ShieldUsed)
	assert.Positive(t, strikeAt(armed(edgeSwordID), paladin).DamageToTarget, "only once a battle")
	paladin.EndFightRT()
	assert.False(t, paladin.RT.ShieldUsed, "a new battle renews it")
}

// Phase 38c1: a foe the Warlord marked is easier for every ally to hit, and a
// Battle Cry raises its allies' Attack; neither touches the foe's own aim.
func TestWarlordsMarkAndBattleCryRaiseAttack(t *testing.T) {
	defenseSpecs(t)
	ally, foe := classed("", 10), classed("", 10)
	base := attackRating(ally, foe)

	foe.RTState().Mark = 5
	assert.Equal(t, base+5, attackRating(ally, foe), "marked: +5 Attack against it")
	foe.RT.Mark = 10
	assert.Equal(t, base+10, attackRating(ally, foe), "rank 50 mark: +10")
	assert.Equal(t, base, attackRating(foe, ally), "a mark on the foe does not help the foe")
	foe.RT.Mark = 0
	assert.Equal(t, base, attackRating(ally, foe), "lifted")

	ally.Aura.Attack = 3
	assert.Equal(t, base+3, attackRating(ally, foe), "Battle Cry: +3 Attack")
	foe.RTState().Mark = 5
	assert.Equal(t, base+8, attackRating(ally, foe), "they stack")
}

// Phase 39a: a Sweep's or a held blow's share of the damage is the blow's
// own, before armor, and clears with the blow.
func TestBlowPctScalesAClassBlow(t *testing.T) {
	defenseSpecs(t)
	src := classed("sweeper", 12)
	foe := classed("", 12)
	assert.Equal(t, 10, classBlowDamage(src, foe, 10))
	src.RTState().BlowPct = 80
	assert.Equal(t, 8, classBlowDamage(src, foe, 10))
	src.RTState().BlowPct = 125
	assert.Equal(t, 13, classBlowDamage(src, foe, 10), "rounded: 12.5")
	assert.Equal(t, 1, classBlowDamage(src, foe, 1), "a blow that landed still does 1")
	assert.Zero(t, classBlowDamage(src, foe, 0), "a miss stays a miss")
	// Unclassed characters (a companion with no class) scale too.
	plain := classed("", 12)
	plain.RTState().BlowPct = 80
	assert.Equal(t, 8, classBlowDamage(plain, foe, 10))
}

// Phase 39c: a fogbound attacker loses accuracy with a ranged weapon only.
func TestFogPenaltyHitsRangedWeaponsOnly(t *testing.T) {
	c := characters.New()
	assert.Zero(t, fogPenalty(c), "no fog, no penalty")
	c.Buffs.List = []*buffs.Buff{{BuffId: status.Fogbound, TriggersLeft: 3}}
	c.Equipment.Weapon = items.Item{ItemId: 991, Spec: &items.ItemSpec{Subtype: items.Shooting}}
	assert.Equal(t, stormcraft.FogHit, fogPenalty(c))
	c.Equipment.Weapon = items.Item{ItemId: 992, Spec: &items.ItemSpec{Subtype: items.Wearable}}
	assert.Zero(t, fogPenalty(c), "a melee blow ignores the fog")
}
