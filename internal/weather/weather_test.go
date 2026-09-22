package weather

import (
	"errors"
	"testing"
)

func clear() Condition {
	return Condition{Name: "clear", Description: "The sky is clear.", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 100}
}

func TestConditionValidate(t *testing.T) {
	if err := clear().Validate(); err != nil {
		t.Fatalf("expected valid condition, got %v", err)
	}

	cases := []Condition{
		{Name: "", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 100},
		{Name: "storm", TravelDurationPct: PctMin - 1, ExertionPct: 100, RestRecoveryPct: 100},
		{Name: "storm", TravelDurationPct: 100, ExertionPct: PctMax + 1, RestRecoveryPct: 100},
		{Name: "storm", TravelDurationPct: 100, ExertionPct: 100, RestRecoveryPct: 0},
	}
	for i, c := range cases {
		if err := c.Validate(); !errors.Is(err, ErrInvalidCondition) {
			t.Errorf("case %d: expected ErrInvalidCondition, got %v", i, err)
		}
	}
}

func TestEstablished(t *testing.T) {
	w, err := Established("dunmar", clear(), 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Zone != "dunmar" || w.Current != "clear" || w.NextChangeRound != 50 {
		t.Fatalf("unexpected zone weather: %+v", w)
	}

	if _, err := Established("", clear(), 50, 10); !errors.Is(err, ErrInvalidZoneWeather) {
		t.Errorf("expected ErrInvalidZoneWeather for empty zone, got %v", err)
	}
	if _, err := Established("dunmar", Condition{}, 50, 10); err == nil {
		t.Errorf("expected error for invalid condition")
	}
	if _, err := Established("dunmar", clear(), 10, 10); !errors.Is(err, ErrInvalidZoneWeather) {
		t.Errorf("expected ErrInvalidZoneWeather for non-future schedule, got %v", err)
	}
}

func TestDue(t *testing.T) {
	w, err := Established("dunmar", clear(), 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Due(49) {
		t.Errorf("expected not due before scheduled round")
	}
	if !w.Due(50) {
		t.Errorf("expected due at scheduled round")
	}
	if !w.Due(51) {
		t.Errorf("expected due after scheduled round")
	}

	var invalid ZoneWeather
	if invalid.Due(1000) {
		t.Errorf("expected invalid zone weather to never be due")
	}
}

func TestAdvance(t *testing.T) {
	w, err := Established("dunmar", clear(), 50, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storm := Condition{Name: "storm", Description: "A storm rolls in.", TravelDurationPct: 150, ExertionPct: 130, RestRecoveryPct: 80}

	if _, err := w.Advance(40, storm, 90); !errors.Is(err, ErrNotDue) {
		t.Errorf("expected ErrNotDue before schedule, got %v", err)
	}

	advanced, err := w.Advance(50, storm, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced.Zone != "dunmar" || advanced.Current != "storm" || advanced.NextChangeRound != 90 {
		t.Fatalf("unexpected advanced weather: %+v", advanced)
	}

	if _, err := w.Advance(50, storm, 50); !errors.Is(err, ErrInvalidZoneWeather) {
		t.Errorf("expected ErrInvalidZoneWeather for non-future schedule, got %v", err)
	}
	if _, err := w.Advance(50, Condition{}, 90); err == nil {
		t.Errorf("expected error for invalid incoming condition")
	}

	var invalid ZoneWeather
	if _, err := invalid.Advance(50, storm, 90); err == nil {
		t.Errorf("expected error advancing an untracked/invalid zone weather")
	}
}
