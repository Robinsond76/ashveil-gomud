package camping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeRest struct{}

func (fakeRest) MovementBlocked(int) (bool, string) { return false, "" }
func (fakeRest) LeaderRest(int) (RestActivity, bool) {
	return RestActivity{Resting: true, Remaining: time.Minute}, true
}
func (fakeRest) RestTierOf(int) (Tier, time.Duration, bool) { return TierRested, time.Hour, true }

type plainMovement struct{}

func (plainMovement) MovementBlocked(int) (bool, string) { return false, "" }

func TestRestProviderNone(t *testing.T) {
	SetMovementProvider(nil)
	_, ok := LeaderRest(7)
	assert.False(t, ok)
	tier, _, ok := RestTierOf(7)
	assert.False(t, ok)
	assert.Equal(t, TierNone, tier)
	SetMovementProvider(plainMovement{})
	t.Cleanup(func() { SetMovementProvider(nil) })
	_, ok = LeaderRest(7)
	assert.False(t, ok, "a movement provider without rest views")
	SetMovementProvider(fakeRest{})
	r, ok := LeaderRest(7)
	assert.True(t, ok)
	assert.True(t, r.Resting)
	tier, left, ok := RestTierOf(7)
	assert.True(t, ok)
	assert.Equal(t, TierRested, tier)
	assert.Equal(t, time.Hour, left)
}

type fakeAbandoner struct{ got []int }

func (f *fakeAbandoner) AbandonCamp(leaderUserID int) error {
	f.got = append(f.got, leaderUserID)
	return nil
}

// Phase 27b: AbandonCamp is a no-op without a provider and delegates with
// one.
func TestAbandonCampSeam(t *testing.T) {
	SetCampAbandoner(nil)
	assert.NoError(t, AbandonCamp(7))
	f := &fakeAbandoner{}
	SetCampAbandoner(f)
	t.Cleanup(func() { SetCampAbandoner(nil) })
	assert.NoError(t, AbandonCamp(7))
	assert.Equal(t, []int{7}, f.got)
}
