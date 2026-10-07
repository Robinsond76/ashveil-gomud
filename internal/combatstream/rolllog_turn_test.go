package combatstream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 79: `why` numbers rounds within the fight, not by the server's
// round counter, so "Round 3" is the fight's third round and never
// "Round 1314418".
func TestRollLogNumbersRoundsWithinTheFight(t *testing.T) {
	log := NewRollLog()
	src := Ref{UserId: 7, Name: "Aria"}
	foe := Ref{MobInstanceId: 2, Name: "bandit"}
	log.Add(7, Roll{Round: 1314414, FightID: 9, Source: src, Target: foe, Outcome: OutcomeHit, Damage: 4})
	log.Add(7, Roll{Round: 1314414, FightID: 9, Source: foe, Target: src, Outcome: OutcomeMiss})
	log.Add(7, Roll{Round: 1314416, FightID: 9, Source: src, Target: foe, Outcome: OutcomeMiss})
	log.Add(7, Roll{Round: 1314418, FightID: 9, Source: src, Target: foe, Outcome: OutcomeHit, Damage: 2})

	recent := log.Recent(7, 0)
	assert.Equal(t, "Round 3: Aria -> bandit, hit for 2.", recent[0].Describe(0)[0])
	assert.Equal(t, "Round 2: Aria -> bandit, missed.", recent[1].Describe(0)[0])
	assert.Equal(t, "Round 1: bandit -> Aria, missed.", recent[2].Describe(0)[0], "two rolls in one round share its number")
	assert.Equal(t, "Round 1: Aria -> bandit, hit for 4.", recent[3].Describe(0)[0])

	// A new fight starts over at one.
	log.Add(7, Roll{Round: 1314500, FightID: 10, Source: src, Target: foe, Outcome: OutcomeMiss})
	assert.Equal(t, "Round 1: Aria -> bandit, missed.", log.Recent(7, 1)[0].Describe(0)[0])

	// A roll never logged keeps the engine's number, as the tests build them.
	assert.Equal(t, "Round 2: Aria -> bandit, missed.", Roll{Round: 2, Source: src, Target: foe}.Describe(0)[0])
}
