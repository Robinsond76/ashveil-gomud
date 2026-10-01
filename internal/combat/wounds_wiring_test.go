package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/wounds"
)

// woundFight sets a room, a hostile mob 4343 with a club, and forced crits.
func woundFight(t *testing.T) *mobs.Mob {
	t.Helper()
	edgeSpecs(t)
	forceCrits(t)
	const clubID = 99320
	items.SetTestItemSpec(&items.ItemSpec{ItemId: clubID, Name: "test club", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 1,
		Damage: items.Damage{DiceRoll: "1d4", Attacks: 1, DiceCount: 1, SideCount: 4}})
	t.Cleanup(func() { items.RemoveTestItemSpec(clubID) })
	foe := &mobs.Mob{InstanceId: 4343, Character: *edgeFighter(90231)}
	foe.Character.Equipment.Weapon = items.New(clubID)
	return foe
}

// TestCritWoundsAPlayer: a forced crit through AttackMobVsPlayer leaves a
// lasting fracture on the player and says so.
func TestCritWoundsAPlayer(t *testing.T) {
	foe := woundFight(t)
	user := users.NewUserRecord(90320, 90320)
	*user.Character = *edgeFighter(90231)

	result := AttackMobVsPlayer(foe, user)
	if len(user.Character.Wounds) != 1 {
		t.Fatalf("wounds %+v, want one", user.Character.Wounds)
	}
	w := user.Character.Wounds[0]
	if w.Light || w.Kind != wounds.Fracture || w.Points != (result.DamageToTarget+1)/2 {
		t.Fatalf("wound %+v for %d damage", w, result.DamageToTarget)
	}
	if !strings.Contains(strings.Join(result.MessagesToTarget, "\n"), ", wounded)") {
		t.Fatalf("the hit line should name the wound: %q", result.MessagesToTarget)
	}
}

// TestCritWoundsACompanionNotAnEnemy: a companion (4344) is wounded through
// AttackMobVsMob; an enemy mob struck by a player is not.
func TestCritWoundsACompanionNotAnEnemy(t *testing.T) {
	foe := woundFight(t)
	withChemistry(t, 0)
	companion := &mobs.Mob{InstanceId: 4344, Character: *edgeFighter(90231)}
	AttackMobVsMob(foe, companion)
	if len(companion.Character.Wounds) != 1 || companion.Character.Wounds[0].Light {
		t.Fatalf("companion wounds %+v, want one lasting", companion.Character.Wounds)
	}

	user := users.NewUserRecord(90321, 90321)
	*user.Character = *edgeFighter(90231)
	user.Character.Equipment.Weapon = items.New(edgeSwordID)
	result := AttackPlayerVsMob(user, foe)
	if !result.Crit || len(foe.Character.Wounds) != 0 || len(result.WoundsToTarget) != 0 {
		t.Fatalf("an enemy is never wounded: %+v", foe.Character.Wounds)
	}
	if strings.Contains(strings.Join(result.MessagesToSource, "\n"), "wounded") {
		t.Fatalf("an enemy's hit line names no wound: %q", result.MessagesToSource)
	}
}

// TestCrushingBlowLeavesALightWound: a plain hit of a quarter of max health
// leaves a light bruise; a smaller one leaves nothing.
func TestCrushingBlowLeavesALightWound(t *testing.T) {
	edgeSpecs(t)
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = 100, 100
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	gameplay.Combat.ParryChanceMin, gameplay.Combat.ParryChanceMax = 0, 0
	gameplay.Combat.BlockChanceMin, gameplay.Combat.BlockChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))

	source := edgeFighter(90231)
	source.Equipment.Weapon = items.New(edgeSwordID) // 1 damage a strike
	weak := edgeFighter(90231)
	weak.Health, weak.HealthMax.Value = 4, 4
	result := calculateCombat(*source, *weak, User, User, 0, 0)
	if len(result.WoundsToTarget) != 1 || !result.WoundsToTarget[0].Light || result.WoundsToTarget[0].Kind != wounds.Bruise {
		t.Fatalf("1 of 4 health is crushing: %+v", result.WoundsToTarget)
	}
	strong := edgeFighter(90231)
	strong.Health, strong.HealthMax.Value = 5, 5
	if result := calculateCombat(*source, *strong, User, User, 0, 0); len(result.WoundsToTarget) != 0 {
		t.Fatalf("1 of 5 health is not crushing: %+v", result.WoundsToTarget)
	}
}
