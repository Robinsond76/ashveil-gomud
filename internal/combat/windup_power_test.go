package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/status"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Phase 30d2: a wind-up's blow, through the real AttackMobVsPlayer and
// AttackMobVsMob with a power provider set.

const powerClubID = 99340

// powerFight is a room, an ogre (mob 4350) with a fixed 3-damage club that
// strikes twice a round when ordinary, and every strike landing, undodged,
// with no crits.
func powerFight(t *testing.T, toHit int) *mobs.Mob {
	t.Helper()
	edgeSpecs(t)
	buffs.LoadFlagDataFiles()
	buffs.LoadDataFiles()
	gameplay := configs.GetGamePlayConfig()
	gameplay.Combat.ToHitMin, gameplay.Combat.ToHitMax = configs.ConfigInt(toHit), configs.ConfigInt(toHit)
	gameplay.Combat.CritChanceMin, gameplay.Combat.CritChanceMax = 0, 0
	gameplay.Combat.DodgeChanceMin, gameplay.Combat.DodgeChanceMax = 0, 0
	t.Cleanup(configs.SetTestGamePlayConfig(gameplay))
	items.SetTestItemSpec(&items.ItemSpec{ItemId: powerClubID, Name: "test great club", Type: items.Weapon, Subtype: items.Bludgeoning, Hands: 2,
		Damage: items.Damage{DiceRoll: "2x1d1+2", Attacks: 2, DiceCount: 1, SideCount: 1, BonusDamage: 2}})
	t.Cleanup(func() { items.RemoveTestItemSpec(powerClubID) })
	ogre := &mobs.Mob{InstanceId: 4350, Character: *edgeFighter(90231)}
	ogre.Character.Equipment.Weapon = items.New(powerClubID)
	return ogre
}

// crushing sets the provider: mob 4350's blow is a Crushing Blow.
func crushing(t *testing.T) {
	t.Helper()
	t.Cleanup(SetPowerProvider(func(id int) (Power, bool) {
		if id != 4350 {
			return Power{}, false
		}
		return Power{Name: "Crushing Blow", Multiplier: 2, Status: status.KnockedDown}, true
	}))
}

func TestPowerBlowIsOneDoubledStrikeThatKnocksDown(t *testing.T) {
	ogre := powerFight(t, 100)
	user := users.NewUserRecord(90340, 90340)
	*user.Character = *edgeFighter(90231)

	plain := AttackMobVsPlayer(ogre, user)
	if !plain.Hit || plain.DamageToTarget != 6 {
		t.Fatalf("an ordinary round is two 3-damage strikes, got %d: %+v", plain.DamageToTarget, plain.MessagesToTarget)
	}

	crushing(t)
	r := AttackMobVsPlayer(ogre, user)
	if !r.Hit || r.DamageToTarget != 6 {
		t.Fatalf("a Crushing Blow is one strike of double damage (6), got %d", r.DamageToTarget)
	}
	if len(r.BuffTarget) != 1 || r.BuffTarget[0] != status.KnockedDown {
		t.Fatalf("a blow that got through knocks down: %v", r.BuffTarget)
	}
	for _, lines := range [][]string{r.MessagesToSource, r.MessagesToTarget, r.MessagesToSourceRoom} {
		joined := strings.Join(lines, "\n")
		if strings.Count(joined, "(Crushing Blow, 6 damage, knocked down)") != 1 {
			t.Fatalf("one strike, named with its status: %q", joined)
		}
	}
}

func TestPowerBlowThatMissesKnocksNobodyDown(t *testing.T) {
	ogre := powerFight(t, 0)
	crushing(t)
	user := users.NewUserRecord(90341, 90341)
	*user.Character = *edgeFighter(90231)
	r := AttackMobVsPlayer(ogre, user)
	if r.Hit || r.DamageToTarget != 0 || len(r.BuffTarget) != 0 {
		t.Fatalf("a miss does nothing: %+v", r)
	}
}

func TestPowerBlowOnAMobAndOnlyForItsMob(t *testing.T) {
	ogre := powerFight(t, 100)
	crushing(t)
	withChemistry(t, 0)
	target := &mobs.Mob{InstanceId: 4351, Character: *edgeFighter(90231)}
	if r := AttackMobVsMob(ogre, target); r.DamageToTarget != 6 || len(r.BuffTarget) != 1 {
		t.Fatalf("mob vs mob: a Crushing Blow, got %d %v", r.DamageToTarget, r.BuffTarget)
	}
	other := &mobs.Mob{InstanceId: 4352, Character: ogre.Character}
	r := AttackMobVsMob(other, target)
	if len(r.BuffTarget) != 0 || strings.Contains(strings.Join(r.MessagesToSource, "\n"), "Crushing Blow") {
		t.Fatalf("another mob's blow is ordinary: %+v", r)
	}
}
