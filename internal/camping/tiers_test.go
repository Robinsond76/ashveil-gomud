package camping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDecideNeverDowngrades(t *testing.T) {
	grant, remove := Decide(TierWellRested, TierRested)
	assert.False(t, grant, "a camp rest never downgrades Well Rested")
	assert.Empty(t, remove)
}

func TestDecideRemovesLowerTier(t *testing.T) {
	grant, remove := Decide(TierRested, TierWellRested)
	assert.True(t, grant)
	assert.Equal(t, []Tier{TierRested}, remove)

	grant, remove = Decide(TierNone, TierRested)
	assert.True(t, grant)
	assert.Empty(t, remove)

	grant, remove = Decide(TierRested, TierRested)
	assert.True(t, grant, "the same tier refreshes")
	assert.Empty(t, remove)

	grant, remove = Decide(TierWellRested, TierWellRested)
	assert.True(t, grant)
	assert.Empty(t, remove)

	grant, _ = Decide(TierRested, TierNone)
	assert.False(t, grant, "no tier grants nothing")
}

func TestMergeOwedKeepsHigherTier(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	well := OwedGrant{BuffID: 1030, Tier: TierWellRested, ExpiresAtUTC: now.Add(30 * time.Minute)}
	rested := OwedGrant{BuffID: 1033, Tier: TierRested, ExpiresAtUTC: now.Add(15 * time.Minute)}

	assert.Equal(t, well, MergeOwed(well, true, rested, now), "an owed Well Rested isn't downgraded")
	assert.Equal(t, well, MergeOwed(rested, true, well, now), "a higher tier replaces")
	later := rested
	later.ExpiresAtUTC = now.Add(20 * time.Minute)
	assert.Equal(t, later, MergeOwed(rested, true, later, now), "the same tier replaces")
	assert.Equal(t, rested, MergeOwed(OwedGrant{}, false, rested, now), "nothing owed yet")
}

func TestMergeOwedReplacesExpired(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	expired := OwedGrant{BuffID: 1030, Tier: TierWellRested, ExpiresAtUTC: now}
	rested := OwedGrant{BuffID: 1033, Tier: TierRested, ExpiresAtUTC: now.Add(15 * time.Minute)}
	assert.Equal(t, rested, MergeOwed(expired, true, rested, now))
	assert.True(t, expired.Expired(now), "expiry is inclusive")
	assert.False(t, rested.Expired(now))
}

func TestRemainingRounds(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, 225, RemainingRounds(now.Add(15*time.Minute), now, 4))
	assert.Equal(t, 1, RemainingRounds(now.Add(time.Second), now, 4), "rounds up, at least one")
	assert.Equal(t, 0, RemainingRounds(now, now, 4), "expired")
	assert.Equal(t, 0, RemainingRounds(now.Add(-time.Minute), now, 4))
	assert.Equal(t, 3, RemainingRounds(now.Add(10*time.Second), now, 0), "a bad round length falls back to 4s")
	assert.Equal(t, 225, RoundsFor(15*time.Minute, 4))
	assert.Equal(t, 1, RoundsFor(0, 4), "at least one round")
}

func TestTierNames(t *testing.T) {
	assert.Equal(t, "Rested", TierRested.String())
	assert.Equal(t, "Well Rested", TierWellRested.String())
	assert.True(t, TierRested.Valid())
	assert.False(t, TierNone.Valid())
	assert.False(t, Tier(9).Valid())
}
