package mount

import (
	"errors"
	"testing"
)

func packHorse() MountSpec {
	return MountSpec{Type: "pack-horse", Description: "A sturdy pack horse.", CargoCapacityBonusGrams: 100000, TravelDurationPct: 85}
}

func TestMountSpecValidate(t *testing.T) {
	if err := packHorse().Validate(); err != nil {
		t.Fatalf("expected valid spec, got %v", err)
	}

	cases := []MountSpec{
		{Type: "", CargoCapacityBonusGrams: 100},
		{Type: "mule", CargoCapacityBonusGrams: -1},
		{Type: "mule", CargoCapacityBonusGrams: 0, TravelDurationPct: PctMin - 1},
		{Type: "mule", CargoCapacityBonusGrams: 0, TravelDurationPct: PctMax + 1},
		{Type: "mule", FatiguePct: PctMin - 1},
		{Type: "mule", FatiguePct: PctMax + 1},
		{Type: "mule", Riders: -1},
	}
	for i, c := range cases {
		if err := c.Validate(); !errors.Is(err, ErrInvalidSpec) {
			t.Errorf("case %d: expected ErrInvalidSpec, got %v", i, err)
		}
	}

	// A zero TravelDurationPct means "not configured" and must not fail
	// range validation.
	unset := MountSpec{Type: "mule", CargoCapacityBonusGrams: 0, TravelDurationPct: 0}
	if err := unset.Validate(); err != nil {
		t.Errorf("expected zero TravelDurationPct to be valid (unset), got %v", err)
	}
}

func TestEstablished(t *testing.T) {
	m, err := Established(7, "pack-horse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.LeaderUserID != 7 || m.Type != "pack-horse" {
		t.Fatalf("unexpected mount: %+v", m)
	}

	if _, err := Established(0, "pack-horse"); !errors.Is(err, ErrInvalidMount) {
		t.Errorf("expected ErrInvalidMount for invalid leader, got %v", err)
	}
	if _, err := Established(7, ""); !errors.Is(err, ErrInvalidMount) {
		t.Errorf("expected ErrInvalidMount for empty type, got %v", err)
	}
}

func TestMountSpecFatigueReliefDefaults(t *testing.T) {
	unset := MountSpec{Type: "mule"}
	if err := unset.Validate(); err != nil {
		t.Fatalf("unset FatiguePct/Riders must be valid, got %v", err)
	}
	if got := unset.EffectiveFatiguePct(); got != 100 {
		t.Errorf("FatiguePct 0 means 100, got %d", got)
	}
	if got := unset.EffectiveRiders(); got != DefaultRiders || DefaultRiders != 2 {
		t.Errorf("Riders 0 means 2, got %d", got)
	}
	if got := unset.EffectiveTravelDurationPct(); got != 100 {
		t.Errorf("TravelDurationPct 0 means 100, got %d", got)
	}
	set := MountSpec{Type: "horse", FatiguePct: 75, Riders: 1, TravelDurationPct: 90}
	if set.EffectiveFatiguePct() != 75 || set.EffectiveRiders() != 1 || set.EffectiveTravelDurationPct() != 90 {
		t.Errorf("configured values must pass through: %+v", set)
	}
}

func TestReliefNeutralWithoutProvider(t *testing.T) {
	SetProvider(nil)
	if pct, riders := Relief(7); pct != 100 || riders != 0 {
		t.Errorf("expected (100, 0), got (%d, %d)", pct, riders)
	}
	if got := TravelDurationPct(7); got != 100 {
		t.Errorf("expected 100, got %d", got)
	}
}
