package archetypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayerUtilityLevel(t *testing.T) {
	wiz := wizard()
	assert.Equal(t, 2, PlayerUtilityLevel(wiz, true, "light", 2))
	assert.Equal(t, 4, PlayerUtilityLevel(wiz, true, "light", 9), "capped at 4")
	assert.Zero(t, PlayerUtilityLevel(wiz, true, "traps", 3), "wizards don't do traps")
	assert.Zero(t, PlayerUtilityLevel(wiz, false, "light", 3), "no archetype, no utility")
	assert.Zero(t, PlayerUtilityLevel(wiz, true, "light", 0), "needs the skill")
}

func TestCompanionUtilityLevelFor(t *testing.T) {
	r := rogue()
	assert.Equal(t, 2, CompanionUtilityLevel(r, true, "traps", 12))
	assert.Zero(t, CompanionUtilityLevel(r, true, "light", 12))
	assert.Zero(t, CompanionUtilityLevel(r, false, "traps", 12))
}

func TestBestMember(t *testing.T) {
	members := []UtilityMember{
		{CompanionID: 3, Level: 2},
		{IsLeader: true, Level: 2},
		{CompanionID: 1, Level: 1},
	}
	best, ok := BestMember(members)
	assert.True(t, ok)
	assert.True(t, best.IsLeader, "ties go to the leader")

	members = []UtilityMember{{CompanionID: 3, Level: 2}, {CompanionID: 2, Level: 2}, {IsLeader: true, Level: 1}}
	best, _ = BestMember(members)
	assert.Equal(t, 2, best.CompanionID, "then the lowest companion id")

	members = []UtilityMember{{CompanionID: 3, Level: 4}, {IsLeader: true, Level: 1}}
	best, _ = BestMember(members)
	assert.Equal(t, 3, best.CompanionID, "the highest level wins")

	_, ok = BestMember([]UtilityMember{{IsLeader: true}, {CompanionID: 1}})
	assert.False(t, ok, "nobody qualifies at level 0")
	_, ok = BestMember(nil)
	assert.False(t, ok)
}

func TestCheckMargin(t *testing.T) {
	// score = level*perLevel + perception/4; margin = score + roll - (difficulty*factor + 50)
	assert.Equal(t, 25, CheckScore(1, 20, 20))
	assert.Equal(t, 85, CheckScore(4, 20, 20))
	assert.Equal(t, 0, CheckMargin(25, 50, 5, 5), "exactly on the line succeeds")
	assert.Equal(t, -1, CheckMargin(25, 49, 5, 5))
	assert.Equal(t, 35, CheckMargin(85, 50, 10, 5))
	assert.True(t, CheckMargin(25, 50, 5, 5) >= 0)
}

func TestDisarmExpiry(t *testing.T) {
	until := DisarmedUntil(1000, 900)
	assert.Equal(t, uint64(1900), until)
	assert.False(t, TrapArmedAt(until, 1000))
	assert.False(t, TrapArmedAt(until, 1899))
	assert.True(t, TrapArmedAt(until, 1900), "re-arms at the expiry round")
	assert.True(t, TrapArmedAt(0, 5), "no record means armed")
	assert.Equal(t, uint64(1000), DisarmedUntil(1000, 0), "zero rounds never disarms")
}
