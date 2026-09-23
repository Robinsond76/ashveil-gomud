package camping

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Phase 16 Task 4: the camp rest weather multiplier ---

func rainyModule(module *CampingModule) {
	module.weatherIn = func(zone string) (weather.Condition, bool) {
		if zone == "Old Kings Road" {
			return weather.Condition{Name: "rain", TravelDurationPct: 115, ExertionPct: 115, RestRecoveryPct: 90}, true
		}
		return weather.Condition{}, false
	}
}

func campRoomIn(zone string) *rooms.Room {
	room := eligibleRoom()
	room.Zone = zone
	return room
}

func TestCampRestInTrackedZoneStoresAndAppliesScaledRecovery(t *testing.T) {
	now := baseTime()
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(store, scheduler, surv, func() time.Time { return now })
	rainyModule(module)
	user := campUser(t, 7, 100)
	room := campRoomIn("Old Kings Road")

	module.establish(user, room)
	module.lightFire(user, room)
	text := module.startRest(user, room)
	assert.Contains(t, text, "rain makes for a poorer rest")
	require.NotNil(t, store.saved.Camps[7].Rest)
	assert.Equal(t, 18, store.saved.Camps[7].Rest.Recovery, "20 × 90% = 18, locked at rest start")

	// Clearing weather mid-rest doesn't change the locked amount.
	module.weatherIn = func(string) (weather.Condition, bool) { return weather.Condition{}, false }
	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	assert.Equal(t, []int{18}, surv.amounts)
}

func TestCampRestInUntrackedZoneRestoresBaseRecovery(t *testing.T) {
	now := baseTime()
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(&fakeStore{}, scheduler, surv, func() time.Time { return now })
	rainyModule(module)
	user := campUser(t, 7, 100)
	room := campRoomIn("Dunmar")
	module.establish(user, room)
	module.lightFire(user, room)
	assert.NotContains(t, module.startRest(user, room), "poorer")
	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	assert.Equal(t, []int{camping.FatigueRecovery}, surv.amounts)
}

func TestLegacyCampRestCompletesWithBaseRecovery(t *testing.T) {
	legacy := []byte(`camps:
  7:
    leader_user_id: 7
    room_id: 100
    fire_lit: true
    rest:
      started_at_utc: 2026-09-22T12:00:00Z
      state: 0
`)
	var registry Registry
	require.NoError(t, decodeRegistry(legacy, &registry))
	surv := &fakeSurvival{}
	module := newTestModule(&fakeStore{saved: registry}, &fakeScheduler{}, surv, func() time.Time { return baseTime().Add(2 * camping.RestDuration) })
	rainyModule(module)
	module.load()
	require.NoError(t, module.loadErr)
	assert.Equal(t, []int{20}, surv.amounts)
}

// --- Phase 16 Task 5: inns ---

func innRoom() *rooms.Room {
	return &rooms.Room{RoomId: 2003, Title: "The Waymark Inn", Zone: "Dunmar", Tags: []string{"inn", "indoor"}}
}

type buffCall struct {
	name string
	id   int
}

type innEnv struct {
	module    *CampingModule
	store     *fakeStore
	scheduler *fakeScheduler
	surv      *fakeSurvival
	now       *time.Time
	buffs     *[]buffCall
	companion *characters.Character
}

func newInnEnv(t *testing.T) *innEnv {
	t.Helper()
	now := baseTime()
	e := &innEnv{store: &fakeStore{}, scheduler: &fakeScheduler{}, surv: &fakeSurvival{}, now: &now, buffs: &[]buffCall{}}
	e.module = newTestModule(e.store, e.scheduler, e.surv, func() time.Time { return *e.now })
	e.companion = &characters.Character{Name: "Bran"}
	e.module.companySize = func(int) int { return 2 }
	e.module.spawnedCompanions = func(int) []*characters.Character { return []*characters.Character{e.companion} }
	e.module.grantBuff = func(c *characters.Character, id int) error {
		*e.buffs = append(*e.buffs, buffCall{c.Name, id})
		return nil
	}
	return e
}

func TestInnRefusesOutsideAnInn(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 100)
	user.Character.Gold = 100
	assert.Equal(t, "There is no inn here.", e.module.innStatus(user, ineligibleRoom()))
	assert.Equal(t, "There is no inn here.", e.module.innRest(user, ineligibleRoom()))
	assert.Equal(t, 100, user.Character.Gold)
	assert.Zero(t, e.store.saveCalls)
}

func TestInnShowsPriceForLeaderAndCompanions(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 42
	text := e.module.innStatus(user, innRoom())
	assert.Contains(t, text, "company of 2")
	assert.Contains(t, text, "10 gold (5 per member)")
	assert.Contains(t, text, "You have 42 gold")
}

func TestInnRestWithoutGoldRefusesWithNoChange(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 9
	assert.Contains(t, e.module.innRest(user, innRoom()), "costs 10 gold, and you have only 9")
	assert.Equal(t, 9, user.Character.Gold)
	assert.Empty(t, e.module.stays)
	assert.Zero(t, e.store.saveCalls)
	assert.Empty(t, e.scheduler.callbacks)
}

func TestInnRestTakesGoldAndPersistsStay(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	assert.Contains(t, e.module.innRest(user, innRoom()), "You pay 10 gold")
	assert.Equal(t, 15, user.Character.Gold)
	stay, ok := e.store.saved.Stays[7]
	require.True(t, ok)
	assert.Equal(t, 2003, stay.RoomID)
	assert.Equal(t, 10, stay.Paid)
	assert.Equal(t, 60, stay.Rest.Recovery)
	assert.Equal(t, 60*time.Second, stay.Duration)
	require.Len(t, e.scheduler.delays, 1)
	assert.Equal(t, 60*time.Second, e.scheduler.delays[0])
}

func TestInnRestFailedSaveRefundsAndLeavesNoStay(t *testing.T) {
	e := newInnEnv(t)
	e.store.saveErr = assert.AnError
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	text := e.module.innRest(user, innRoom())
	assert.Contains(t, text, "save failed")
	assert.Equal(t, 25, user.Character.Gold, "the gold is refunded")
	assert.Empty(t, e.module.stays)
	assert.Empty(t, e.scheduler.callbacks)
}

func TestMovementBlockedDuringStay(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	blocked, _ := e.module.MovementBlocked(7)
	assert.False(t, blocked)
	e.module.innRest(user, innRoom())
	*e.now = baseTime().Add(20 * time.Second)
	blocked, message := e.module.MovementBlocked(7)
	assert.True(t, blocked)
	assert.Contains(t, message, "resting at the inn (40s remaining)")
}

func TestCampRestAndInnStayExcludeEachOther(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 100)
	user.Character.Gold = 25
	camp := eligibleRoom()
	e.module.establish(user, camp)
	e.module.lightFire(user, camp)
	e.module.startRest(user, camp)
	assert.Equal(t, "You are already resting at camp.", e.module.innRest(user, innRoom()))
	assert.Equal(t, 25, user.Character.Gold)

	e2 := newInnEnv(t)
	user2 := campUser(t, 8, 2003)
	user2.Character.Gold = 25
	e2.module.innRest(user2, innRoom())
	e2.module.establish(user2, camp)
	e2.module.lightFire(user2, camp)
	assert.Equal(t, "You are already resting at the inn.", e2.module.startRest(user2, camp))
}

func TestInnRestRefusedWhileTravelling(t *testing.T) {
	e := newInnEnv(t)
	e.module.travelling = func(int) bool { return true }
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	assert.Contains(t, e.module.innRest(user, innRoom()), "while travelling")
	assert.Equal(t, 25, user.Character.Gold)
}

// TestInnTimerCompletesWithoutTouchingBuffs: the timer runs off the game
// loop, so it applies recovery (once) and marks Well Rested pending, but
// never calls the buff seam. The NewRound handler grants it.
func TestInnTimerCompletesWithoutTouchingBuffs(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Name = "Hero"
	user.Character.Gold = 25
	messages := captureMessages(t)
	e.module.innRest(user, innRoom())

	*e.now = baseTime().Add(60 * time.Second)
	e.scheduler.fireLatest()
	assert.Equal(t, []int{60}, e.surv.amounts)
	assert.True(t, e.store.saved.WellRestedPending[7])
	assert.True(t, e.store.saved.InnRecoveryApplied[7])
	assert.Equal(t, camping.Completed, e.store.saved.Stays[7].Rest.State)
	assert.Empty(t, *e.buffs, "the timer never grants buffs")
	assert.False(t, func() bool { b, _ := e.module.MovementBlocked(7); return b }(), "a finished stay no longer blocks movement")

	// Replay: a duplicate timer and a sync apply nothing more.
	e.scheduler.fireLatest()
	e.module.innStatus(user, innRoom())
	assert.Equal(t, []int{60}, e.surv.amounts)

	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Equal(t, []buffCall{{"Hero", 1030}, {"Bran", 1030}}, *e.buffs)
	assert.Empty(t, e.store.saved.Stays, "the stay is removed once recovery and buff are applied")
	assert.False(t, e.store.saved.WellRestedPending[7])
	assert.False(t, e.store.saved.InnRecoveryApplied[7])
	events.ProcessEvents()
	text := strings.Join(*messages, "\n")
	assert.Contains(t, text, "wakes rested")
	assert.Contains(t, text, "well rested")

	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2, "granted once")

	// Stays are repeatable.
	*e.now = baseTime().Add(2 * time.Minute)
	assert.Contains(t, e.module.innRest(user, innRoom()), "You pay 10 gold")
	assert.Equal(t, 5, user.Character.Gold)
}

func TestWellRestedWaitsForOfflineLeader(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	e.module.innRest(user, innRoom())
	*e.now = baseTime().Add(time.Minute)
	e.scheduler.fireLatest()

	e.module.lookupUser = func(int) *users.UserRecord { return nil }
	e.module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.Empty(t, *e.buffs)
	assert.True(t, e.module.wellRestedPending[7], "still owed while the leader is offline")

	e.module.lookupUser = nil
	e.module.onNewRound(events.NewRound{RoundNumber: 2})
	assert.Len(t, *e.buffs, 2)
}

func TestStayReloadMidRestReschedules(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	e.module.innRest(user, innRoom())

	childScheduler := &fakeScheduler{}
	childSurv := &fakeSurvival{}
	child := newTestModule(e.store, childScheduler, childSurv, func() time.Time { return baseTime().Add(25 * time.Second) })
	child.load()
	require.NoError(t, child.loadErr)
	require.Contains(t, child.stays, 7)
	require.Len(t, childScheduler.delays, 1)
	assert.Equal(t, 35*time.Second, childScheduler.delays[0])
	assert.Empty(t, childSurv.amounts)
	blocked, _ := child.MovementBlocked(7)
	assert.True(t, blocked)
}

func TestStayReloadOverdueCompletesOnce(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	e.module.innRest(user, innRoom())

	childSurv := &fakeSurvival{}
	childScheduler := &fakeScheduler{}
	child := newTestModule(e.store, childScheduler, childSurv, func() time.Time { return baseTime().Add(time.Hour) })
	child.load()
	assert.Equal(t, []int{60}, childSurv.amounts)
	assert.Empty(t, childScheduler.callbacks, "an overdue stay completes, it isn't rescheduled")
	assert.True(t, child.wellRestedPending[7])

	// A second reload (e.g. copyover) retries nothing: recovery is marked.
	grandchild := newTestModule(e.store, &fakeScheduler{}, childSurv, func() time.Time { return baseTime().Add(2 * time.Hour) })
	grandchild.load()
	assert.Equal(t, []int{60}, childSurv.amounts)
	assert.True(t, grandchild.wellRestedPending[7])
}

func TestRenderCampViewShowsInnStay(t *testing.T) {
	e := newInnEnv(t)
	user := campUser(t, 7, 2003)
	user.Character.Gold = 25
	messages := captureMessages(t)
	e.module.innRest(user, innRoom())
	*e.now = baseTime().Add(30 * time.Second)
	handled, err := e.module.RenderCampView(7)
	require.NoError(t, err)
	assert.True(t, handled)
	events.ProcessEvents()
	assert.Contains(t, strings.Join(*messages, "\n"), "Resting: 50% complete (30s remaining)")
}

func TestParseInnSettings(t *testing.T) {
	s := parseInnSettings(func(name string) any {
		return map[string]any{"InnRoomTag": "tavern", "PricePerMember": 0, "InnRestDuration": "90s", "InnFatigueRecovery": 45, "WellRestedBuffId": 2000}[name]
	})
	assert.Equal(t, innSettings{RoomTag: "tavern", PricePerMember: 0, RestDuration: 90 * time.Second, FatigueRecovery: 45, WellRestedBuffId: 2000}, s)
	d := parseInnSettings(func(string) any { return nil })
	assert.Equal(t, defaultInnSettings(), d)
	bad := parseInnSettings(func(name string) any {
		return map[string]any{"PricePerMember": -3, "InnRestDuration": "soon", "InnFatigueRecovery": 0}[name]
	})
	assert.Equal(t, defaultInnSettings(), bad)
}
