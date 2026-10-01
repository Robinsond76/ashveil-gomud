package expedition

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/archetypes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func trailMessages(t *testing.T) *[]string {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	users.SetTestUser(users.NewUserRecord(7, 1))
	events.ProcessEvents()
	messages := []string{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		messages = append(messages, e.(events.Message).Text)
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return &messages
}

// TestTrackerWarnsOfAnAmbushRouteAtDeparture (33f2): a tracker in the
// company reads the road on a route that can be ambushed; none, no line.
func TestTrackerWarnsOfAnAmbushRouteAtDeparture(t *testing.T) {
	for _, tracker := range []bool{true, false} {
		messages := trailMessages(t)
		module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, combatInterruptionProfiles())
		module.mobSpawner = &fakeMobSpawner{}
		module.trailReader = func(leader int, roomIDs ...int) (archetypes.Specialist, bool) {
			assert.Equal(t, 7, leader)
			assert.Equal(t, []int{100}, roomIDs, "the tracker stands at the origin")
			return archetypes.Specialist{Name: "Mira", Level: 1}, tracker
		}
		_, err := module.StartTravel(startRequest())
		require.NoError(t, err)
		events.ProcessEvents()
		text := strings.Join(*messages, "")
		if tracker {
			assert.Contains(t, text, "Mira reads the road ahead")
		} else {
			assert.NotContains(t, text, "road ahead")
		}
	}
	messages := trailMessages(t)
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, interruptionProfiles())
	module.trailReader = func(int, ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{Name: "Mira", Level: 4}, true
	}
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	events.ProcessEvents()
	assert.NotContains(t, strings.Join(*messages, ""), "road ahead", "a route without ambushes gives no warning")
}

// TestTrackerLeadsTheCompanyAroundAnAmbush (33f2): when the ambush fires
// and the tracker succeeds, nothing spawns, the pause is ordinary tracks
// (resumable at once), and the leader is told who led them around it.
func TestTrackerLeadsTheCompanyAroundAnAmbush(t *testing.T) {
	messages := trailMessages(t)
	scheduler := &fakeScheduler{}
	now := baseTime()
	store := &fakeStore{}
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, combatInterruptionProfiles())
	spawner := &fakeMobSpawner{}
	module.mobSpawner = spawner
	module.ambushEvader = func(leader int, roomIDs ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{Name: "Mira", Level: 4}, true
	}
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(5 * time.Second)
	scheduler.fire(0)
	events.ProcessEvents()

	assert.Empty(t, spawner.spawnCalls, "no ambush spawns")
	session := module.sessions[7]
	require.NotNil(t, session.Interruption)
	assert.Equal(t, expedition.Tracks, session.Interruption.Kind)
	assert.Equal(t, expedition.Tracks, store.saved.Sessions[7].Interruption.Kind, "durable")
	text := strings.Join(*messages, "")
	assert.Contains(t, text, "Mira leads the company off the road and around an ambush.")
	assert.Contains(t, text, "Fresh tracks cross")
}

// TestFailedEvasionStillSpawnsTheAmbush (33f2): a tracker who fails leaves
// the ambush as it was.
func TestFailedEvasionStillSpawnsTheAmbush(t *testing.T) {
	trailMessages(t)
	scheduler := &fakeScheduler{}
	now := baseTime()
	module := newTestModule(&fakeStore{}, scheduler, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, combatInterruptionProfiles())
	spawner := &fakeMobSpawner{}
	module.mobSpawner = spawner
	module.ambushEvader = func(int, ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{Name: "Mira", Level: 1}, false
	}
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(5 * time.Second)
	scheduler.fire(0)
	require.Len(t, spawner.spawnCalls, 1)
	assert.Equal(t, expedition.Combat, module.sessions[7].Interruption.Kind)
}

// evadedJourney starts a combat route whose ambush a tracker evades.
func evadedJourney(t *testing.T) (*ExpeditionModule, *fakeStore) {
	t.Helper()
	trailMessages(t)
	scheduler := &fakeScheduler{}
	now := baseTime()
	store := &fakeStore{}
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, combatInterruptionProfiles())
	module.mobSpawner = &fakeMobSpawner{}
	module.ambushEvader = func(int, ...int) (archetypes.Specialist, bool) {
		return archetypes.Specialist{Name: "Mira", Level: 4}, true
	}
	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(5 * time.Second)
	scheduler.fire(0)
	require.Equal(t, expedition.Tracks, module.sessions[7].Interruption.Kind)
	return module, store
}

// TestEvadedAmbushCanBeResumedReturnedAndRecovered (33f2 review finding
// 2): the tracks an evaded ambush leaves are a valid pause on that route.
func TestEvadedAmbushCanBeResumedReturnedAndRecovered(t *testing.T) {
	module, _ := evadedJourney(t)
	assert.Equal(t, "You resume travel along the oak-road route.", module.resume(7))
	assert.Equal(t, expedition.Traveling, module.sessions[7].State)

	module, _ = evadedJourney(t)
	assert.NotContains(t, module.returnToOrigin(7), "invalid")
	_, still := module.sessions[7]
	assert.False(t, still, "the company turned back")

	_, store := evadedJourney(t)
	reloaded := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, combatInterruptionProfiles())
	reloaded.mobSpawner = &fakeMobSpawner{}
	reloaded.load()
	require.Contains(t, reloaded.sessions, 7, "recovery keeps the paused journey")
	assert.Equal(t, "You resume travel along the oak-road route.", reloaded.resume(7))
}

// TestParseInterruptionReadsAmbushAndWeightedRoutes (33f2 review finding
// 4): a route's ambush mob and weighted kinds are read from config.
func TestParseInterruptionReadsAmbushAndWeightedRoutes(t *testing.T) {
	profiles := parseProfiles([]any{
		map[string]any{"Name": "ambush-road", "Duration": "30s", "Interruption": map[string]any{"Kind": "combat", "CombatMobId": 12, "Checkpoint": 4}},
		map[string]any{"Name": "mixed-road", "Duration": "30s", "Interruption": map[string]any{"Checkpoint": 6, "CombatMobId": 12,
			"Kinds": []any{map[string]any{"Kind": "fallen-tree", "Weight": 3}, map[string]any{"Kind": "combat", "Weight": 1}}}},
		map[string]any{"Name": "no-mob", "Duration": "30s", "Interruption": map[string]any{"Kind": "combat", "Checkpoint": 4}},
		map[string]any{"Name": "bad-weight", "Duration": "30s", "Interruption": map[string]any{"Checkpoint": 4, "Kinds": []any{map[string]any{"Kind": "tracks", "Weight": 0}}}},
	})
	require.Contains(t, profiles, "ambush-road")
	assert.Equal(t, 12, profiles["ambush-road"].Interruption.CombatMobID)
	assert.True(t, profiles["ambush-road"].Interruption.ReachesCombat())
	require.Contains(t, profiles, "mixed-road")
	assert.Len(t, profiles["mixed-road"].Interruption.Kinds, 2)
	assert.True(t, profiles["mixed-road"].Interruption.ReachesCombat())
	assert.NotContains(t, profiles, "no-mob", "an ambush needs its mob")
	assert.NotContains(t, profiles, "bad-weight")
}
