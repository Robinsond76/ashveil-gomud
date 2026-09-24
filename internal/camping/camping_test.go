package camping

import (
	"errors"
	"testing"
	"time"
)

var campTime = time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)

func TestCampLifecycle(t *testing.T) {
	camp, err := Established(7, 42)
	if err != nil {
		t.Fatal(err)
	}
	if camp.FireLit || camp.Rest != nil {
		t.Fatalf("new camp = %#v", camp)
	}

	camp, err = camp.LightFire()
	if err != nil || !camp.FireLit {
		t.Fatalf("light fire: %#v, %v", camp, err)
	}
	if _, err := camp.LightFire(); !errors.Is(err, ErrFireAlreadyLit) {
		t.Fatalf("duplicate fire error = %v", err)
	}

	camp, err = camp.StartRest(campTime)
	if err != nil || camp.Rest == nil || camp.Rest.State != Resting {
		t.Fatalf("start rest: %#v, %v", camp, err)
	}
	if _, err := camp.StartRest(campTime); !errors.Is(err, ErrRestAlreadyStarted) {
		t.Fatalf("duplicate rest error = %v", err)
	}
	if camp.RestDue(campTime.Add(RestDuration - time.Nanosecond)) {
		t.Fatal("rest due too early")
	}
	if camp.ProgressAt(campTime.Add(30*time.Second)) != .5 {
		t.Fatal("halfway progress not .5")
	}
	if camp.ProgressAt(campTime.Add(2*RestDuration)) != 1 {
		t.Fatal("progress did not clamp")
	}

	if _, err := camp.Break(); !errors.Is(err, ErrRestInProgress) {
		t.Fatalf("break while resting error = %v", err)
	}
	camp, err = camp.CompleteRest(campTime.Add(RestDuration))
	if err != nil || camp.Rest == nil || camp.Rest.State != Completed {
		t.Fatalf("complete rest: %#v, %v", camp, err)
	}
	if _, err := camp.StartRest(campTime.Add(2 * RestDuration)); !errors.Is(err, ErrRestAlreadyCompleted) {
		t.Fatalf("restart completed error = %v", err)
	}
	camp, err = camp.Break()
	if err != nil || camp != (Camp{}) {
		t.Fatalf("break idle camp: %#v, %v", camp, err)
	}
}

func TestCampRejectsInvalidStateAndIDs(t *testing.T) {
	for _, ids := range [][2]int{{0, 1}, {-1, 1}, {1, 0}, {1, -1}} {
		if _, err := Established(ids[0], ids[1]); !errors.Is(err, ErrInvalidCamp) {
			t.Fatalf("ids %#v error = %v", ids, err)
		}
	}
	camp, _ := Established(1, 2)
	if _, err := camp.StartRest(campTime); !errors.Is(err, ErrFireNotLit) {
		t.Fatalf("unlit rest error = %v", err)
	}
	if _, err := camp.CompleteRest(campTime); !errors.Is(err, ErrNoRest) {
		t.Fatalf("complete without rest error = %v", err)
	}
	if _, err := camp.Break(); err != nil {
		t.Fatal(err)
	}

	bad := Camp{LeaderUserID: 1, RoomID: 2, Rest: &RestSession{StartedAtUTC: campTime, State: RestState(99)}}
	if err := bad.Validate(); !errors.Is(err, ErrInvalidCamp) {
		t.Fatalf("bad state validation = %v", err)
	}
	bad = Camp{LeaderUserID: 1, RoomID: 2, Rest: &RestSession{State: Resting}}
	if err := bad.Validate(); !errors.Is(err, ErrInvalidCamp) {
		t.Fatalf("zero start validation = %v", err)
	}
}

func TestRestTimingIsFixedAtSixtySeconds(t *testing.T) {
	if RestDuration != 60*time.Second {
		t.Fatalf("duration = %v", RestDuration)
	}
	if FatigueRecovery != 20 {
		t.Fatalf("recovery = %d", FatigueRecovery)
	}
	camp, _ := Established(1, 2)
	camp, _ = camp.LightFire()
	camp, _ = camp.StartRest(campTime)
	if camp.ProgressAt(campTime.Add(-time.Hour)) != 0 {
		t.Fatal("pre-start progress did not clamp")
	}
	if !camp.RestDue(campTime.Add(RestDuration)) {
		t.Fatal("rest due at duration")
	}
}

type fakeCampAbandoner struct{ leaders []int }

func (f *fakeCampAbandoner) AbandonForDeath(leaderUserID int) error {
	f.leaders = append(f.leaders, leaderUserID)
	return nil
}

func TestAbandonForDeathNoProvider(t *testing.T) {
	SetAbandonProvider(nil)
	if err := AbandonForDeath(7); err != nil {
		t.Fatalf("no provider: %v", err)
	}
	f := &fakeCampAbandoner{}
	SetAbandonProvider(f)
	t.Cleanup(func() { SetAbandonProvider(nil) })
	if err := AbandonForDeath(7); err != nil || len(f.leaders) != 1 || f.leaders[0] != 7 {
		t.Fatalf("provider not called: %v %v", err, f.leaders)
	}
}
