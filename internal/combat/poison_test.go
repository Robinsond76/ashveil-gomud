package combat

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 43b weapon poisons in the strike loop.

const poisonTenID = 99431 // a fixed 10-damage sword, so percent cuts show

func poisonSpecs(t *testing.T) {
	t.Helper()
	defenseSpecs(t)
	items.SetTestItemSpec(&items.ItemSpec{ItemId: poisonTenID, Name: "heavy test sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1,
		Damage: items.Damage{DiceRoll: "10d1", Attacks: 1, DiceCount: 10, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(poisonTenID) })
}

func coated(id int, kind string, contacts int) items.Item {
	itm := items.New(id)
	now := time.Now()
	itm.Coat(kind, now.Add(time.Hour), contacts, now)
	return itm
}

// poisonRounds swings n rounds with a fresh coated blade each time and
// returns how many wounded, how many spent a contact, and how many left a
// poison.
func poisonRounds(t *testing.T, n int, kind string, target *characters.Character, mob ...*mobs.Mob) (wounded, spent, delivered int) {
	t.Helper()
	for i := 0; i < n; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = coated(edgeSwordID, kind, 8)
		res := calculateCombat(*src, *target, User, Mob, 0, 0, mob...)
		if res.DamageToTarget > 0 {
			wounded++
		}
		spent += res.PoisonSpent[items.Weapon]
		for _, id := range res.BuffTarget {
			if status.IsWeaponPoison(id) {
				delivered++
			}
		}
	}
	return
}

func TestCoatedBladeSpendsAContactPerWoundAndDeliversNearFortyPercent(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	wounded, spent, delivered := poisonRounds(t, 800, "bitterleaf", target)
	require.Greater(t, wounded, 300)
	assert.Equal(t, wounded, spent, "one contact per wounding blow; a miss, dodge or absorbed hit spends none")
	rate := float64(delivered) / float64(wounded)
	assert.InDelta(t, 0.40, rate, 0.10, "delivery chance")
}

func TestPoisonAlreadyOnVictimNeverStacksButStillSpendsContact(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	require.NoError(t, target.AddBuff(status.Mirethorn, false))
	wounded, spent, delivered := poisonRounds(t, 300, "bitterleaf", target)
	require.Positive(t, wounded)
	assert.Equal(t, wounded, spent, "the contact is spent anyway")
	assert.Zero(t, delivered, "no second poison, no refresh")
}

func TestDualWieldedPoisonsLeaveOnePerBlowRound(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	bothSpent := false
	for i := 0; i < 400; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = coated(defClawID, "bitterleaf", 8) // two claws always both swing
		src.Equipment.Offhand = coated(defClawID, "leechbane", 8)
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		poisons := 0
		for _, id := range res.BuffTarget {
			if status.IsWeaponPoison(id) {
				poisons++
			}
		}
		assert.LessOrEqual(t, poisons, 1, "one weapon poison per victim, whichever hand")
		if res.PoisonSpent[items.Weapon] > 0 && res.PoisonSpent[items.Offhand] > 0 {
			bothSpent = true
		}
	}
	assert.True(t, bothSpent, "each hand spends its own coating")
}

func TestImmuneAndResistantTargets(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	immune := &mobs.Mob{PoisonSusceptibility: items.PoisonImmune}
	wounded, spent, delivered := poisonRounds(t, 400, "bitterleaf", target, immune)
	require.Positive(t, wounded)
	assert.Equal(t, wounded, spent, "an immune foe still uses the contact")
	assert.Zero(t, delivered)

	resistant := &mobs.Mob{PoisonSusceptibility: items.PoisonResistant}
	wounded, _, delivered = poisonRounds(t, 800, "bitterleaf", target, resistant)
	// The mob argument is only read for a Mob-typed target.
	assert.InDelta(t, 0.20, float64(delivered)/float64(wounded), 0.08, "resistant: half the chance")
}

func TestLapsedOrEmptyCoatingDoesNothing(t *testing.T) {
	poisonSpecs(t)
	target := edgeFighter(90231)
	for i := 0; i < 100; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = coated(edgeSwordID, "bitterleaf", 8)
		src.Equipment.Weapon.CoatExpires = time.Now().Add(-time.Second).Unix()
		res := calculateCombat(*src, *target, User, Mob, 0, 0)
		assert.Empty(t, res.PoisonSpent, "an expired coating never delivers")
		for _, id := range res.BuffTarget {
			assert.False(t, status.IsWeaponPoison(id))
		}
	}
}

func TestCoatingStopsWhenContactsRunOut(t *testing.T) {
	poisonSpecs(t)
	c := edgeFighter(90231)
	c.Equipment.Weapon = coated(edgeSwordID, "bitterleaf", 3)
	spendCoatings(c, map[items.ItemType]int{items.Weapon: 2})
	assert.Equal(t, 1, c.Equipment.Weapon.CoatContacts)
	spendCoatings(c, map[items.ItemType]int{items.Weapon: 1})
	assert.Empty(t, c.Equipment.Weapon.CoatKind, "a spent coating is gone")
}

// TestAttackPlayerVsMobSpendsLeaderCoating drives the real entry point.
func TestAttackPlayerVsMobSpendsLeaderCoating(t *testing.T) {
	poisonSpecs(t)
	user := users.NewUserRecord(4241, 4241)
	user.Character.RoomId = 90231
	user.Character.RaceId = 1
	user.Character.Equipment.Weapon = coated(edgeSwordID, "bitterleaf", 8)
	user.Character.SetAggro(0, 4341, characters.DefaultAttack)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(4241) })

	wounds := 0
	for i := 0; i < 60 && user.Character.Equipment.Weapon.CoatKind != ``; i++ {
		mob := &mobs.Mob{InstanceId: 4341, Character: *edgeFighter(90231)}
		if AttackPlayerVsMob(user, mob).PoisonSpent[items.Weapon] > 0 {
			wounds++
		}
		assert.Equal(t, max(8-wounds, 0), user.Character.Equipment.Weapon.CoatContacts)
	}
	require.Positive(t, wounds)
}

func TestAttackMobVsMobSpendsCompanionCoating(t *testing.T) {
	poisonSpecs(t)
	companion := &mobs.Mob{InstanceId: 4342, Character: *edgeFighter(90231)}
	companion.Character.Equipment.Weapon = coated(edgeSwordID, "leechbane", 8)
	spentAny := false
	for i := 0; i < 60 && companion.Character.Equipment.Weapon.CoatKind != ``; i++ {
		enemy := &mobs.Mob{InstanceId: 4343, Character: *edgeFighter(90231)}
		before := companion.Character.Equipment.Weapon.CoatContacts
		AttackMobVsMob(companion, enemy)
		if companion.Character.Equipment.Weapon.CoatContacts < before || companion.Character.Equipment.Weapon.CoatKind == `` {
			spentAny = true
		}
	}
	assert.True(t, spentAny, "the live companion's own weapon is charged")
}

func TestLeadrootCutsPhysicalDamageByFifteenPercent(t *testing.T) {
	poisonSpecs(t)
	assert.Equal(t, 17, leadrootDamage(leadrooted(t), 20))
	assert.Equal(t, 9, leadrootDamage(leadrooted(t), 10))
	assert.Equal(t, 3, leadrootDamage(leadrooted(t), 3), "rounded down: nothing off a small blow")
	assert.Equal(t, 1, leadrootDamage(leadrooted(t), 1))
	assert.Equal(t, 20, leadrootDamage(characters.New(), 20), "only while it lasts")

	// Through the strike loop: a leadrooted dealer's landed blows average
	// about 15% less than the plain blows.
	target := edgeFighter(90231)
	mean := func(root bool) float64 {
		total, n := 0, 0
		for i := 0; i < 3000 && n < 400; i++ {
			src := edgeFighter(90231)
			src.Equipment.Weapon = items.New(poisonTenID)
			if root {
				require.NoError(t, src.AddBuff(status.Leadroot, false))
			}
			res := calculateCombat(*src, *target, User, Mob, 0, 0)
			if res.Hit && !res.Crit && res.DamageToTarget > 0 {
				total += res.DamageToTarget + res.DamageToTargetReduction
				n++
			}
		}
		require.Positive(t, n)
		return float64(total) / float64(n)
	}
	plain, rooted := mean(false), mean(true)
	assert.InDelta(t, 0.85, rooted/plain, 0.07)
}

func leadrooted(t *testing.T) *characters.Character {
	t.Helper()
	c := characters.New()
	require.NoError(t, c.AddBuff(status.Leadroot, false))
	return c
}

func TestMirethornLowersDodgeTenPoints(t *testing.T) {
	poisonSpecs(t)
	atk := edgeFighter(90231)
	def := edgeFighter(90231)
	base := effectiveDodge(def, atk)
	require.NoError(t, def.AddBuff(status.Mirethorn, false))
	assert.Equal(t, max(0, base-10), effectiveDodge(def, atk))
}

func TestLeechbaneCutsHealingByAQuarterWithAFloorOfOne(t *testing.T) {
	poisonSpecs(t)
	c := edgeFighter(90231)
	c.HealthMax.Value = 100
	c.Health = 10
	assert.Equal(t, 10, c.ApplyHealthChange(10), "no poison: the whole heal")
	require.NoError(t, c.AddBuff(status.Leechbane, false))
	c.HealthMax.Value = 100
	assert.Equal(t, 8, c.ApplyHealthChange(10), "10 heals 8: a quarter (2) off")
	c.Health = 10
	assert.Equal(t, 3, c.ApplyHealthChange(3), "rounded down: 3/4 is nothing")
	c.Health = 10
	assert.Equal(t, 1, c.ApplyHealthChange(1), "a positive heal is never cut to nothing")
	c.Health = 10
	assert.Equal(t, -5, c.ApplyHealthChange(-5), "damage is untouched")
}

func TestBitterleafTicksOnePerCombatRoundForThreeRounds(t *testing.T) {
	poisonSpecs(t)
	c := edgeFighter(90231)
	require.NoError(t, c.AddBuff(status.Bitterleaf, false))
	c.Health = 50 // adding a buff revalidates the character
	require.True(t, status.PoisonLive(c))
	var damage, rounds int
	for i := 0; i < 5; i++ {
		for _, ch := range status.Tick(c) {
			damage += ch.Damage
			rounds++
		}
	}
	assert.Equal(t, 3, damage, "1 damage a round for 3 rounds")
	assert.Equal(t, 3, rounds)
	assert.Equal(t, 47, c.Health)
}

func TestCleansingTakesAWeaponPoisonOff(t *testing.T) {
	poisonSpecs(t)
	c := edgeFighter(90231)
	require.NoError(t, c.AddBuff(status.Leadroot, false))
	assert.Equal(t, "leadroot", status.CleanseOne(c))
	assert.False(t, status.PoisonLive(c))
	require.NoError(t, c.AddBuff(status.Bitterleaf, false))
	c.CancelBuffsWithFlag("poison")
	assert.False(t, status.PoisonLive(c), "curepoison clears them by their poison flag")
}
