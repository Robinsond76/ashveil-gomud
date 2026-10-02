package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/battle"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/wounds"
	"github.com/stretchr/testify/assert"
)

// Phase 33i2: an enemy that lived through a fight recovers from nothing to
// full in 5 game hours (188 rounds), only out of every battle and fight;
// companions and pets keep their own rules.
func TestRegenEnemyVitals(t *testing.T) {
	const room = 990331
	idle := engagementMob(t, 9331, 1, room)
	fighting := engagementMob(t, 9332, 1, room)
	fighting.Character.Aggro = &characters.Aggro{UserId: 1}
	engaged := engagementMob(t, 9333, 1, room)
	companion := engagementMob(t, 9334, 1, room)
	pet := engagementMob(t, 9335, 1, room)
	pet.Character.Charmed = &characters.CharmInfo{UserId: 5, RoundsRemaining: -1}
	wounded := engagementMob(t, 9336, 1, room)
	wounded.Character.Wounds = []wounds.Wound{{Kind: wounds.Cut, Place: "arm", Points: 100}}
	downed := engagementMob(t, 9337, 0, room)
	for _, m := range []*characters.Character{&idle.Character, &fighting.Character, &engaged.Character, &companion.Character, &pet.Character, &wounded.Character, &downed.Character} {
		m.HealthMax.Value, m.ManaMax.Value, m.Mana = 188, 376, 0
	}
	leaderOf := func(id int) (int, company.MemberKey, bool) {
		if id == 9334 {
			return 1, company.CompanionMemberKey(id), true
		}
		return 0, "", false
	}
	inBattle := func(id int) bool { return id == 9333 }

	regenEnemyVitals(leaderOf, inBattle)
	assert.Equal(t, 1+3, idle.Character.Health, "1/188 of max a round, three rounds a pass")
	assert.Equal(t, 6, idle.Character.Mana, "mana too, by its own max")
	for name, m := range map[string]*characters.Character{"fighting": &fighting.Character, "in a battle": &engaged.Character, "a companion": &companion.Character, "a pet": &pet.Character, "downed": &downed.Character} {
		assert.Equal(t, 0, m.Mana, name+" recovers nothing")
	}
	assert.Equal(t, 88, wounded.Character.HealthLimit())

	// From 1 health to full takes 63 passes (189 rounds): 5 game hours.
	passes := 1
	for idle.Character.Health < 188 {
		regenEnemyVitals(leaderOf, inBattle)
		passes++
	}
	assert.Equal(t, 63, passes)
	assert.Equal(t, 188, idle.Character.Health, "capped at its maximum")
	for i := 0; i < 70; i++ {
		regenEnemyVitals(leaderOf, inBattle)
	}
	assert.Equal(t, 88, wounded.Character.Health, "stops at the wound limit")
}

// The real hook: AutoHeal heals an idle enemy on its third-round beat and
// leaves one in a player's battle alone.
func TestAutoHealRecoversEnemiesOutOfBattle(t *testing.T) {
	battle.Reset()
	t.Cleanup(battle.Reset)
	const room = 990332
	idle := engagementMob(t, 9341, 10, room)
	held := engagementMob(t, 9342, 10, room)
	for _, m := range []*characters.Character{&idle.Character, &held.Character} {
		m.HealthMax.Value = 188
	}
	battle.Begin(93410, room, 1, "p", []int{9342})

	AutoHeal(events.NewRound{RoundNumber: 4})
	assert.Equal(t, 10, idle.Character.Health, "off the beat")
	AutoHeal(events.NewRound{RoundNumber: 6})
	assert.Equal(t, 13, idle.Character.Health)
	assert.Equal(t, 10, held.Character.Health, "not while in a battle")
}

func TestEnemyRecovery(t *testing.T) {
	assert.Equal(t, 0, enemyRecovery(0))
	assert.Equal(t, 3, enemyRecovery(1), "at least one a round")
	assert.Equal(t, 3, enemyRecovery(188))
	assert.Equal(t, 6, enemyRecovery(189))
}
