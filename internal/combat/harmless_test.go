package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// TestHarmlessRaceNeverDamages: found live (Phase 44): the tutorial's straw
// soldiers (race 19, dice 0d0, "cannot hurt anyone") hit for the flat damage
// floor and killed a companion. A body with no weapon and no natural damage
// deals nothing, whatever the flat damage bonus is.
func TestHarmlessRaceNeverDamages(t *testing.T) {
	loadTestData(t)
	races.LoadDataFiles()
	room := &rooms.Room{RoomId: 90231, Zone: `Test`, Biome: `city`, Tags: []string{rooms.TagLit}}
	rooms.SetTestRoom(room)
	t.Cleanup(func() { rooms.RemoveTestRoom(room.RoomId) })

	cfg := configs.GetGamePlayConfig()
	cfg.Combat.ToHitMin, cfg.Combat.ToHitMax = 100, 100
	cfg.Combat.DodgeChanceMin, cfg.Combat.DodgeChanceMax = 0, 0
	cfg.Combat.ParryChanceMin, cfg.Combat.ParryChanceMax = 0, 0
	cfg.Combat.BlockChanceMin, cfg.Combat.BlockChanceMax = 0, 0
	assert.Greater(t, int(cfg.Combat.DamageBonusMin), 0, "the shipped flat damage floor is what armed the straw")
	t.Cleanup(configs.SetTestGamePlayConfig(cfg))

	assert.True(t, harmlessStrike(0, 19), "the dummy race is harmless unarmed")
	assert.False(t, harmlessStrike(edgeSwordID, 19), "a weapon in hand is not")
	assert.False(t, harmlessStrike(0, 1), "a human's fists are not")

	items.SetTestItemSpec(&items.ItemSpec{ItemId: edgeSwordID, Name: "test sword", Type: items.Weapon, Subtype: items.Slashing, Hands: 1, Damage: items.Damage{DiceRoll: "1d1", Attacks: 1, DiceCount: 1, SideCount: 1}})
	t.Cleanup(func() { items.RemoveTestItemSpec(edgeSwordID) })

	for i := 0; i < 200; i++ {
		straw, ally := edgeFighter(90231), edgeFighter(90231)
		straw.RaceId = 19
		res := calculateCombat(*straw, *ally, Mob, User, 0, 0)
		assert.False(t, res.Hit)
		assert.Zero(t, res.DamageToTarget)
	}
	straw, ally := edgeFighter(90231), edgeFighter(90231)
	straw.RaceId = 19
	assert.Zero(t, expectedDPS(*straw, *ally), "an assessment counts it as no threat")

	fists, ally := edgeFighter(90231), edgeFighter(90231)
	assert.Greater(t, expectedDPS(*fists, *ally), 0.0)
}
