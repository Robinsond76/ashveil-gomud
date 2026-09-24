package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase 23b edges. Each test weapon rolls a fixed 1d1, so a non-critical
// hit's damage differs between a sharpened and a dull weapon by exactly
// the edge's bonus.
const (
	edgeSwordID  = 99231
	edgeDaggerID = 99232
)

func edgeSpecs(t *testing.T) {
	t.Helper()
	loadTestData(t)
	races.LoadDataFiles()
	items.SetTestItemSpec(&items.ItemSpec{ItemId: edgeSwordID, Name: "test sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Damage: items.Damage{DiceRoll: "1d1", Attacks: 1, DiceCount: 1, SideCount: 1}})
	items.SetTestItemSpec(&items.ItemSpec{ItemId: edgeDaggerID, Name: "test dagger", Type: items.Weapon, Subtype: items.Stabbing, Hands: 1, Damage: items.Damage{DiceRoll: "1d1", Attacks: 1, DiceCount: 1, SideCount: 1}})
	room := &rooms.Room{RoomId: 90231, Zone: `Test`, Biome: `city`, Tags: []string{rooms.TagLit}}
	rooms.SetTestRoom(room)
	t.Cleanup(func() {
		items.RemoveTestItemSpec(edgeSwordID)
		items.RemoveTestItemSpec(edgeDaggerID)
		rooms.RemoveTestRoom(room.RoomId)
	})
}

func edgeFighter(roomID int) *characters.Character {
	c := characters.New()
	c.RoomId = roomID
	c.RaceId = 1
	c.Health, c.HealthMax.Value = 1000000, 1000000
	c.SetAggro(0, 1, characters.DefaultAttack)
	return c
}

func sharpenedItem(id, bonus, strikes int) items.Item {
	itm := items.New(id)
	itm.Sharpen(bonus, strikes)
	return itm
}

// TestEdgeAddsBonusAndSpendsOnHit: over many rounds, every round that hit
// spent exactly one strike and every miss spent none; non-critical hits
// deal exactly the bonus more than the same weapon dull.
func TestEdgeAddsBonusAndSpendsOnHit(t *testing.T) {
	edgeSpecs(t)
	target := edgeFighter(90231)

	dullDamage := map[int]bool{}
	for i := 0; i < 300; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = items.New(edgeSwordID)
		res := calculateCombat(*src, *target, User, Mob, 0)
		assert.Empty(t, res.EdgeSpent, "a dull weapon spends no edge")
		if res.Hit && !res.Crit {
			dullDamage[res.DamageToTarget+res.DamageToTargetReduction] = true
		}
	}

	hits := 0
	for i := 0; i < 300; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = sharpenedItem(edgeSwordID, 3, 20)
		res := calculateCombat(*src, *target, User, Mob, 0)
		if !res.Hit {
			assert.Zero(t, res.EdgeSpent[items.Weapon], "a miss or dodge spends nothing")
			continue
		}
		hits++
		assert.Equal(t, 1, res.EdgeSpent[items.Weapon], "one successful strike, one edge strike")
		if !res.Crit {
			raw := res.DamageToTarget + res.DamageToTargetReduction
			assert.True(t, dullDamage[raw-3], "sharpened non-crit damage %d is dull damage + 3", raw)
		}
	}
	require.Positive(t, hits)
}

func TestEdgeStopsWhenStrikesRunOut(t *testing.T) {
	edgeSpecs(t)
	target := edgeFighter(90231)
	src := edgeFighter(90231)
	src.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 1)
	src.Equipment.Weapon.SpendEdge(1)
	for i := 0; i < 50; i++ {
		res := calculateCombat(*src, *target, User, Mob, 0)
		assert.Empty(t, res.EdgeSpent)
	}
}

// TestOffhandEdgeTrackedBySlot: a dull main hand and a sharpened offhand
// charge only the offhand.
func TestOffhandEdgeTrackedBySlot(t *testing.T) {
	edgeSpecs(t)
	target := edgeFighter(90231)
	spentOffhand := 0
	for i := 0; i < 300; i++ {
		src := edgeFighter(90231)
		src.Equipment.Weapon = items.New(edgeSwordID)
		src.Equipment.Offhand = sharpenedItem(edgeDaggerID, 1, 20)
		res := calculateCombat(*src, *target, User, Mob, 0)
		assert.Zero(t, res.EdgeSpent[items.Weapon])
		spentOffhand += res.EdgeSpent[items.Offhand]
	}
	assert.Positive(t, spentOffhand)

	c := edgeFighter(90231)
	c.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 5)
	c.Equipment.Offhand = sharpenedItem(edgeDaggerID, 1, 5)
	spendEdges(c, map[items.ItemType]int{items.Offhand: 2})
	assert.Equal(t, 5, c.Equipment.Weapon.SharpStrikes)
	assert.Equal(t, 3, c.Equipment.Offhand.SharpStrikes)
}

// TestAttackPlayerVsMobSpendsLeaderEdge drives the real entry point: the
// leader's own weapon loses one strike per hit round.
func TestAttackPlayerVsMobSpendsLeaderEdge(t *testing.T) {
	edgeSpecs(t)
	user := users.NewUserRecord(4231, 4231)
	user.Character.RoomId = 90231
	user.Character.RaceId = 1
	user.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 20)
	user.Character.SetAggro(0, 4331, characters.DefaultAttack)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(4231) })

	hits := 0
	for i := 0; i < 40 && user.Character.Equipment.Weapon.Sharpened(); i++ {
		mob := &mobs.Mob{InstanceId: 4331, Character: *edgeFighter(90231)}
		if AttackPlayerVsMob(user, mob).Hit {
			hits++
		}
		assert.Equal(t, max(20-hits, 0), user.Character.Equipment.Weapon.SharpStrikes)
	}
	require.Positive(t, hits)
}

// TestAttackMobVsMobSpendsCompanionEdge: a companion mob's sharpened
// weapon is spent on the live mob, which the company snapshot saves.
func TestAttackMobVsMobSpendsCompanionEdge(t *testing.T) {
	edgeSpecs(t)
	companion := &mobs.Mob{InstanceId: 4332, Character: *edgeFighter(90231)}
	companion.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 20)
	hits := 0
	for i := 0; i < 60; i++ {
		enemy := &mobs.Mob{InstanceId: 4333, Character: *edgeFighter(90231)}
		if AttackMobVsMob(companion, enemy).Hit {
			hits++
		}
	}
	require.Positive(t, hits)
	assert.Equal(t, max(20-hits, 0), companion.Character.Equipment.Weapon.SharpStrikes)
	if hits >= 20 {
		assert.False(t, companion.Character.Equipment.Weapon.Sharpened())
		assert.Zero(t, companion.Character.Equipment.Weapon.SharpBonus)
	}

	// And a mob striking a player spends its own edge, not the player's.
	player := users.NewUserRecord(4232, 4232)
	player.Character.RoomId = 90231
	player.Character.RaceId = 1
	player.Character.Health, player.Character.HealthMax.Value = 1000000, 1000000
	player.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 20)
	users.SetTestUser(player)
	t.Cleanup(func() { users.RemoveTestUser(4232) })
	attacker := &mobs.Mob{InstanceId: 4334, Character: *edgeFighter(90231)}
	attacker.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 20)
	struck := 0
	for i := 0; i < 10; i++ {
		if AttackMobVsPlayer(attacker, player).Hit {
			struck++
		}
	}
	assert.Equal(t, 20-struck, attacker.Character.Equipment.Weapon.SharpStrikes)
	assert.Equal(t, 20, player.Character.Equipment.Weapon.SharpStrikes)
}

// TestAttackPlayerVsMobChargesOffhandEdge: through the real entry point,
// a sharpened offhand dagger is spent and the dull main hand is untouched.
func TestAttackPlayerVsMobChargesOffhandEdge(t *testing.T) {
	edgeSpecs(t)
	user := users.NewUserRecord(4235, 4235)
	user.Character.RoomId = 90231
	user.Character.RaceId = 1
	user.Character.Skills = map[string]int{"dual-wield": 4} // both weapons swing
	user.Character.Equipment.Weapon = items.New(edgeSwordID)
	user.Character.Equipment.Offhand = sharpenedItem(edgeDaggerID, 1, 20)
	user.Character.SetAggro(0, 4335, characters.DefaultAttack)
	users.SetTestUser(user)
	t.Cleanup(func() { users.RemoveTestUser(4235) })

	for i := 0; i < 40; i++ {
		AttackPlayerVsMob(user, &mobs.Mob{InstanceId: 4335, Character: *edgeFighter(90231)})
	}
	assert.Less(t, user.Character.Equipment.Offhand.SharpStrikes, 20, "the offhand's edge was spent")
	assert.False(t, user.Character.Equipment.Weapon.Sharpened())
	assert.Zero(t, user.Character.Equipment.Weapon.SharpStrikes)
}

// TestAttackPlayerVsPlayerSpendsOnlyAttackerEdge: in PvP the attacker's
// edge is spent and the defender's is untouched.
func TestAttackPlayerVsPlayerSpendsOnlyAttackerEdge(t *testing.T) {
	edgeSpecs(t)
	mk := func(id int) *users.UserRecord {
		u := users.NewUserRecord(id, uint64(id))
		u.Character.RoomId = 90231
		u.Character.RaceId = 1
		u.Character.Health, u.Character.HealthMax.Value = 1000000, 1000000
		u.Character.Equipment.Weapon = sharpenedItem(edgeSwordID, 1, 20)
		users.SetTestUser(u)
		t.Cleanup(func() { users.RemoveTestUser(id) })
		return u
	}
	atk, def := mk(4236), mk(4237)
	atk.Character.SetAggro(4237, 0, characters.DefaultAttack)
	hits := 0
	for i := 0; i < 10; i++ {
		if AttackPlayerVsPlayer(atk, def).Hit {
			hits++
		}
	}
	require.Positive(t, hits)
	assert.Equal(t, 20-hits, atk.Character.Equipment.Weapon.SharpStrikes)
	assert.Equal(t, 20, def.Character.Equipment.Weapon.SharpStrikes)
}
