package rites

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMournsAnyLossButOnlyLongServiceDepartures(t *testing.T) {
	assert.True(t, Mourns(Lost, 0), "a companion lost for good is always mourned")
	assert.False(t, Mourns(Left, LongServiceRounds-1))
	assert.True(t, Mourns(Left, LongServiceRounds))
	assert.False(t, Mourns(Cause("sent away"), 1_000_000), "an unknown cause is no occasion")
}

func TestSteadyAndGrieveStayInsideTheLimits(t *testing.T) {
	assert.Equal(t, 63, Steady(60, true))
	assert.Equal(t, 61, Steady(60, false))
	assert.Equal(t, Ceiling, Steady(Ceiling-1, true), "never above the ceiling")
	assert.Equal(t, Ceiling+7, Steady(Ceiling+7, true), "a loyalty already past it is not pulled back")
	assert.Equal(t, 55, Grieve(60, true))
	assert.Equal(t, 58, Grieve(60, false))
	assert.Equal(t, Floor, Grieve(Floor+1, true), "never below the floor")
	assert.Equal(t, Floor-9, Grieve(Floor-9, true), "a loyalty already under it is not pushed lower")
	// A skipped rite alone can never push someone to leave (loyalty 0).
	loyalty := 100
	for i := 0; i < 50; i++ {
		loyalty = Grieve(loyalty, true)
	}
	assert.Equal(t, Floor, loyalty)
}

func TestHoldingBeatsSkippingForEveryone(t *testing.T) {
	assert.Greater(t, CloseSkip, CloseHold, "a skipped rite hurts more than a held one helps")
	assert.Greater(t, CloseHold, BandHold)
	assert.Greater(t, CloseSkip, BandSkip)
}

func TestWordsForEveryCause(t *testing.T) {
	for _, c := range Causes {
		assert.NotEqual(t, "gone", Phrase(c))
		assert.NotEqual(t, "is gone", Verb(c))
	}
}
