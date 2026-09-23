package expedition

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// multiplierEnv wires fake weather, load, and mount seams onto a module.
type multiplierEnv struct {
	conditions map[string]weather.Condition
	band       *encumbrance.LoadBand
	mountPct   int
}

func (e *multiplierEnv) install(m *ExpeditionModule) {
	m.roomZone = func(roomID int) string {
		switch roomID {
		case 100:
			return "Origin"
		case 200:
			return "Forest"
		}
		return ""
	}
	m.weatherIn = func(zone string) (weather.Condition, bool) {
		c, ok := e.conditions[zone]
		return c, ok
	}
	m.loadBand = func(int) (encumbrance.LoadBand, bool) {
		if e.band == nil {
			return encumbrance.LoadBand{}, false
		}
		return *e.band, true
	}
	m.mountDurationPct = func(int) int {
		if e.mountPct == 0 {
			return 100
		}
		return e.mountPct
	}
}

func rain() weather.Condition {
	return weather.Condition{Name: "rain", TravelDurationPct: 115, ExertionPct: 115, RestRecoveryPct: 90}
}

func captureTravelMessages(t *testing.T) func() string {
	t.Helper()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return func() string {
		events.ProcessEvents()
		return strings.Join(messages, "\n")
	}
}

func TestMultipliersLockedAtStart(t *testing.T) {
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
	env := &multiplierEnv{
		conditions: map[string]weather.Condition{"Origin": rain()},
		band:       &encumbrance.LoadBand{TravelDurationPct: 125, FatiguePct: 115},
		mountPct:   90,
	}
	env.install(module)

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	session := store.saved.Sessions[7]
	// 1.15 × 1.25 × 0.90 = 1.29375 → 129
	assert.Equal(t, 129, session.DurationPct)
	assert.Equal(t, 115, session.ExertionPct)
	assert.Equal(t, 115, session.FatiguePct, "the mount never changes the fatigue multiplier on a route")
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, 12900*time.Millisecond, scheduler.delays[0], "the timer runs on the effective duration")
}

func TestMountChangesRouteDurationOnly(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
	env := &multiplierEnv{mountPct: 90}
	env.install(module)

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	session := store.saved.Sessions[7]
	assert.Equal(t, 90, session.DurationPct)
	assert.Zero(t, session.ExertionPct)
	assert.Zero(t, session.FatiguePct)
}

func TestNeutralDepartureSavesLikeLegacy(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
	(&multiplierEnv{}).install(module)
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	session := store.saved.Sessions[7]
	assert.Zero(t, session.DurationPct)
	assert.Zero(t, session.ExertionPct)
	assert.Zero(t, session.FatiguePct)
}

func TestUntrackedOriginFallsBackToDestinationWeather(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
	(&multiplierEnv{conditions: map[string]weather.Condition{"Forest": rain()}}).install(module)
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.Equal(t, 115, store.saved.Sessions[7].DurationPct)
	assert.Equal(t, 115, store.saved.Sessions[7].ExertionPct)
}

func TestLockedMultipliersPersistReloadAndIgnoreWeatherChanges(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	now := baseTime()
	env := &multiplierEnv{
		conditions: map[string]weather.Condition{"Origin": rain()},
		band:       &encumbrance.LoadBand{TravelDurationPct: 100, FatiguePct: 130},
	}
	parent := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	env.install(parent)
	_, err := parent.StartTravel(startRequest())
	require.NoError(t, err)

	// The storm rolls in mid-journey, and the company sheds its load: the
	// journey keeps the multipliers it set out with.
	env.conditions["Origin"] = weather.Condition{Name: "storm", TravelDurationPct: 140, ExertionPct: 130, RestRecoveryPct: 75}
	env.band = nil

	childScheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	mover := &fakeMover{user: user}
	child := newTestModule(store, childScheduler, mover, surv, func() time.Time { return now }, testProfiles())
	env.install(child)
	now = baseTime().Add(4 * time.Second)
	child.load()
	require.NoError(t, child.loadErr)
	session := child.sessions[7]
	assert.Equal(t, 115, session.DurationPct)
	assert.Equal(t, 115, session.ExertionPct)
	assert.Equal(t, 130, session.FatiguePct)
	require.Len(t, childScheduler.delays, 1)
	assert.Equal(t, 7500*time.Millisecond, childScheduler.delays[0], "11.5s route, 4s gone")

	now = baseTime().Add(12 * time.Second)
	childScheduler.fire(0)
	assert.Equal(t, []int{200}, mover.moves)
	total := sumExertion(surv.applied)
	// base 10/10/10: needs × 1.15, fatigue × 1.15 × 1.30 = 14.95
	assert.Equal(t, survival.Exertion{Hunger: 12, Thirst: 12, Fatigue: 15}, total)
}

// TestLegacySessionReloadsOnOriginalScheduleAndCost: a session saved before
// Phase 16 (no multiplier fields) runs exactly as it always did, even with
// weather and load that would now change a new journey.
func TestLegacySessionReloadsOnOriginalScheduleAndCost(t *testing.T) {
	user := travelUser(t, 7, 100)
	legacy := []byte(`sessions:
  7:
    leader_user_id: 7
    origin_room_id: 100
    destination_room_id: 200
    exit_name: north
    profile_name: oak-road
    started_at_utc: 2026-09-17T12:00:00Z
    last_exertion_checkpoint: 0
    state: 0
`)
	var registry Registry
	require.NoError(t, decodeRegistry(legacy, &registry))
	store := &fakeStore{saved: registry}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	mover := &fakeMover{user: user}
	now := baseTime().Add(3 * time.Second)
	module := newTestModule(store, scheduler, mover, surv, func() time.Time { return now }, testProfiles())
	(&multiplierEnv{
		conditions: map[string]weather.Condition{"Origin": rain()},
		band:       &encumbrance.LoadBand{TravelDurationPct: 150, FatiguePct: 130},
	}).install(module)

	module.load()
	require.NoError(t, module.loadErr)
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, 7*time.Second, scheduler.delays[0])

	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)
	assert.Equal(t, []int{200}, mover.moves)
	assert.Equal(t, survival.Exertion{Hunger: 10, Thirst: 10, Fatigue: 10}, sumExertion(surv.applied))
}

func sumExertion(costs []survival.Exertion) survival.Exertion {
	total := survival.Exertion{}
	for _, cost := range costs {
		total.Hunger += cost.Hunger
		total.Thirst += cost.Thirst
		total.Fatigue += cost.Fatigue
	}
	return total
}

func TestCollapsedCompanyCannotSetOut(t *testing.T) {
	travelUser(t, 7, 100)
	text := captureTravelMessages(t)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{needs: []survival.MemberNeeds{
		{Key: survival.LeaderMemberKey, Name: "Hero", Needs: survival.Needs{Hunger: 80, Thirst: 80, Fatigue: 40}},
		{Key: survival.CompanionMemberKey(1), Name: "Bran", Needs: survival.Needs{Hunger: 80, Thirst: 80, Fatigue: 0}},
	}}
	module := newTestModule(store, scheduler, &fakeMover{}, surv, baseTime, testProfiles())

	handled, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.True(t, handled, "the refusal is handled: the leader stays put")
	assert.Empty(t, module.sessions)
	assert.Zero(t, store.saveCalls)
	assert.Empty(t, scheduler.callbacks)
	assert.Contains(t, text(), "too exhausted to set out")

	surv.needs[1].Needs.Fatigue = 1 // Exhausted, but not Collapsed
	_, err = module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.Contains(t, module.sessions, 7)
}

func TestDepartureLineNamesFactors(t *testing.T) {
	cases := []struct {
		name string
		env  multiplierEnv
		want string
	}{
		{"weather and load", multiplierEnv{conditions: map[string]weather.Condition{"Origin": rain()}, band: &encumbrance.LoadBand{TravelDurationPct: 125, FatiguePct: 115}}, "Rain and a heavy load slow your pace."},
		{"load alone", multiplierEnv{band: &encumbrance.LoadBand{TravelDurationPct: 110, FatiguePct: 108}}, "A heavy load slows your pace."},
		{"mount alone", multiplierEnv{mountPct: 90}, "Your mount quickens the journey."},
		{"nothing", multiplierEnv{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			travelUser(t, 7, 100)
			text := captureTravelMessages(t)
			module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
			env := tc.env
			env.install(module)
			_, err := module.StartTravel(startRequest())
			require.NoError(t, err)
			out := text()
			if tc.want == "" {
				assert.NotContains(t, out, "pace")
				assert.NotContains(t, out, "quickens")
				return
			}
			assert.Contains(t, out, tc.want)
		})
	}
}

func TestJourneyViewShowsEffectiveDuration(t *testing.T) {
	travelUser(t, 7, 100)
	text := captureTravelMessages(t)
	now := baseTime()
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	(&multiplierEnv{band: &encumbrance.LoadBand{TravelDurationPct: 150, FatiguePct: 100}}).install(module)
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.Contains(t, text(), "Estimated travel: 15s")

	now = baseTime().Add(3 * time.Second)
	blocked, message := module.MovementBlocked(7)
	require.True(t, blocked)
	assert.Contains(t, message, "20%")
	assert.Contains(t, message, "12s remaining")
	_ = expedition.Traveling
}
