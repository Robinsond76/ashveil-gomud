package expedition

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestEffectiveProfileLegacy(t *testing.T) {
	profile := validFallenTreeProfile()
	session := validSession()
	assert.Equal(t, profile, session.EffectiveProfile(profile), "all pcts 0 must leave the profile unchanged")

	neutral := validSession()
	neutral.DurationPct, neutral.ExertionPct, neutral.FatiguePct = 100, 100, 100
	assert.Equal(t, profile, neutral.EffectiveProfile(profile))
}

func TestEffectiveProfileScales(t *testing.T) {
	profile := validProfile() // 30s, hunger 7, thirst 11, fatigue 13
	session := validSession()
	session.DurationPct = 150
	session.ExertionPct = 115
	session.FatiguePct = 130

	got := session.EffectiveProfile(profile)
	assert.Equal(t, 45*time.Second, got.Duration)
	assert.Equal(t, survival.Exertion{
		Hunger:  8,  // 7 × 1.15 = 8.05
		Thirst:  13, // 11 × 1.15 = 12.65
		Fatigue: 19, // 13 × 1.15 × 1.30 = 19.435
	}, got.Exertion)
	assert.Equal(t, profile.Name, got.Name)
	assert.Equal(t, 30*time.Second, profile.Duration, "the base profile is not mutated")

	fatigueOnly := validSession()
	fatigueOnly.FatiguePct = 200
	got = fatigueOnly.EffectiveProfile(profile)
	assert.Equal(t, survival.Exertion{Hunger: 7, Thirst: 11, Fatigue: 26}, got.Exertion)
	assert.Equal(t, 30*time.Second, got.Duration)

	faster := validSession()
	faster.DurationPct = 90
	assert.Equal(t, 27*time.Second, faster.EffectiveProfile(profile).Duration)
}

func TestValidatePctRange(t *testing.T) {
	for _, pct := range []int{0, PctMin, 100, PctMax} {
		for _, field := range []func(*TravelSession, int){
			func(s *TravelSession, p int) { s.DurationPct = p },
			func(s *TravelSession, p int) { s.ExertionPct = p },
			func(s *TravelSession, p int) { s.FatiguePct = p },
		} {
			s := validSession()
			field(&s, pct)
			assert.NoError(t, s.Validate(), "pct %d", pct)
		}
	}
	for _, pct := range []int{-1, PctMin - 1, PctMax + 1} {
		for _, field := range []func(*TravelSession, int){
			func(s *TravelSession, p int) { s.DurationPct = p },
			func(s *TravelSession, p int) { s.ExertionPct = p },
			func(s *TravelSession, p int) { s.FatiguePct = p },
		} {
			s := validSession()
			field(&s, pct)
			assert.ErrorIs(t, s.Validate(), ErrInvalidSession, "pct %d", pct)
		}
	}
}

func TestClampPct(t *testing.T) {
	assert.Equal(t, 100, ClampPct(0))
	assert.Equal(t, PctMin, ClampPct(3))
	assert.Equal(t, PctMax, ClampPct(9000))
	assert.Equal(t, 137, ClampPct(137))
}

// TestInterruptionCheckpointScales: the interruption fires at the same
// fraction of a longer journey, and the checkpoint index is unchanged.
func TestInterruptionCheckpointScales(t *testing.T) {
	base := validFallenTreeProfile() // 30s, interruption at checkpoint 5
	session := validSession()
	session.DurationPct = 200
	effective := session.EffectiveProfile(base)
	require.NotNil(t, effective.Interruption)
	assert.Equal(t, uint8(5), effective.Interruption.Checkpoint)

	assert.False(t, session.InterruptionDue(baseTime().Add(20*time.Second), effective), "halfway on the base route is only a quarter of the doubled route")
	assert.True(t, session.InterruptionDue(baseTime().Add(30*time.Second), effective))
	assert.NoError(t, session.ValidateForProfile(effective))
}

// TestLegacySessionYAMLLoadsNeutral: a session saved before Phase 16 has no
// multiplier fields and must reload with neutral (zero) values.
func TestLegacySessionYAMLLoadsNeutral(t *testing.T) {
	legacy := `leader_user_id: 7
origin_room_id: 100
destination_room_id: 200
exit_name: north
profile_name: oak-road
started_at_utc: 2026-09-17T12:00:00Z
last_exertion_checkpoint: 3
state: 0
`
	var session TravelSession
	require.NoError(t, yaml.Unmarshal([]byte(legacy), &session))
	require.NoError(t, session.Validate())
	assert.Zero(t, session.DurationPct)
	assert.Equal(t, validProfile(), session.EffectiveProfile(validProfile()))

	out, err := yaml.Marshal(session)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "pct", "neutral multipliers are omitted on save")
}
