package expedition

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "low", "", false)
	os.Exit(m.Run())
}

type fakeStore struct {
	saved          Registry
	loadErr        error
	saveErr        error
	failNextSave   bool
	failOnSaveCall int
	loadCalls      int
	saveCalls      int
	log            *orderLog
}

func (f *fakeStore) Load(registry *Registry) error {
	f.loadCalls++
	if f.loadErr != nil {
		return f.loadErr
	}
	*registry = f.saved.Clone()
	return nil
}

func (f *fakeStore) Save(registry Registry) error {
	f.saveCalls++
	if f.log != nil {
		f.log.entries = append(f.log.entries, "save")
	}
	if f.failNextSave {
		f.failNextSave = false
		return assert.AnError
	}
	if f.failOnSaveCall > 0 && f.saveCalls == f.failOnSaveCall {
		return assert.AnError
	}
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = registry.Clone()
	return nil
}

type orderLog struct{ entries []string }

type fakeTimer struct{}

func (fakeTimer) Stop() bool { return true }

type fakeScheduler struct {
	log       *orderLog
	callbacks []func()
	delays    []time.Duration
}

func (f *fakeScheduler) AfterFunc(d time.Duration, cb func()) Timer {
	if f.log != nil {
		f.log.entries = append(f.log.entries, "schedule")
	}
	f.delays = append(f.delays, d)
	f.callbacks = append(f.callbacks, cb)
	return fakeTimer{}
}

func (f *fakeScheduler) fire(index int) {
	if index < len(f.callbacks) {
		f.callbacks[index]()
	}
}

type fakeMover struct {
	user  *users.UserRecord
	moves []int
	err   error
}

func (f *fakeMover) MoveToRoom(userID, roomID int) error {
	if f.err != nil {
		return f.err
	}
	if f.user == nil {
		return assert.AnError
	}
	f.moves = append(f.moves, roomID)
	f.user.Character.RoomId = roomID
	return nil
}

type fakeSurvival struct {
	availableErr error
	applyErr     error
	applied      []survival.Exertion
	needs        []survival.MemberNeeds
}

func (f *fakeSurvival) Available() error { return f.availableErr }

func (f *fakeSurvival) ApplyCompanyExertion(_ int, _ string, cost survival.Exertion) ([]survival.ExertionResult, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	f.applied = append(f.applied, cost)
	return nil, nil
}

func (f *fakeSurvival) CompanyNeeds(_ int) []survival.MemberNeeds { return f.needs }

func baseTime() time.Time {
	return time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)
}

func testProfiles() map[string]expedition.TravelProfile {
	return map[string]expedition.TravelProfile{
		"oak-road": {
			Name:     "oak-road",
			Duration: 10 * time.Second,
			Exertion: survival.Exertion{Hunger: 10, Thirst: 10, Fatigue: 10},
		},
	}
}

func interruptionProfiles() map[string]expedition.TravelProfile {
	profiles := testProfiles()
	p := profiles["oak-road"]
	p.Interruption = &expedition.InterruptionProfile{Kind: expedition.FallenTree, Checkpoint: 5}
	profiles["oak-road"] = p
	return profiles
}

func newTestModule(store Store, scheduler Scheduler, mover Mover, surv Survival, clock func() time.Time, profiles map[string]expedition.TravelProfile) *ExpeditionModule {
	return &ExpeditionModule{
		store:     store,
		scheduler: scheduler,
		mover:     mover,
		survival:  surv,
		clock:     clock,
		profiles:  profiles,
		sessions:  map[int]expedition.TravelSession{},
		timers:    map[int]Timer{},
	}
}

func startRequest() expedition.StartRequest {
	return expedition.StartRequest{
		LeaderUserID:      7,
		OriginRoomID:      100,
		DestinationRoomID: 200,
		ExitName:          "north",
		ProfileName:       "oak-road",
	}
}

func TestParseProfilesRejectsMalformedAndDuplicates(t *testing.T) {
	raw := []any{
		map[string]any{"Name": "oak-road", "Duration": "30s", "Exertion": map[string]any{"Hunger": 4, "Thirst": 4, "Fatigue": 6}, "Interruption": map[string]any{"Kind": "fallen-tree", "Checkpoint": 5}},
		map[string]any{"Name": "oak-road", "Duration": "60s"},
		map[string]any{"Name": "zero", "Duration": 0},
		map[string]any{"Name": "negative", "Duration": "10s", "Exertion": map[string]any{"Hunger": -1}},
		map[string]any{"Name": "seconds-int", "Duration": 45},
		map[string]any{"Name": "unknown-kind", "Duration": "30s", "Interruption": map[string]any{"Kind": "rock-slide", "Checkpoint": 5}},
		map[string]any{"Name": "non-map", "Duration": "30s", "Interruption": "fallen-tree"},
		map[string]any{"Name": "checkpoint-zero", "Duration": "30s", "Interruption": map[string]any{"Kind": "fallen-tree", "Checkpoint": 0}},
		map[string]any{"Name": "checkpoint-ten", "Duration": "30s", "Interruption": map[string]any{"Kind": "fallen-tree", "Checkpoint": 10}},
		map[string]any{"Name": "malformed-interruption", "Duration": "30s", "Interruption": map[string]any{"Kind": "fallen-tree", "Checkpoint": 0}},
	}
	profiles := parseProfiles(raw)

	require.Contains(t, profiles, "oak-road")
	assert.Equal(t, 30*time.Second, profiles["oak-road"].Duration)
	assert.Equal(t, survival.Exertion{Hunger: 4, Thirst: 4, Fatigue: 6}, profiles["oak-road"].Exertion)
	require.NotNil(t, profiles["oak-road"].Interruption)
	assert.Equal(t, expedition.FallenTree, profiles["oak-road"].Interruption.Kind)
	assert.Equal(t, uint8(5), profiles["oak-road"].Interruption.Checkpoint)
	assert.NotContains(t, profiles, "zero")
	assert.NotContains(t, profiles, "negative")
	require.Contains(t, profiles, "seconds-int")
	assert.Equal(t, 45*time.Second, profiles["seconds-int"].Duration)
	assert.Nil(t, profiles["seconds-int"].Interruption)
	assert.NotContains(t, profiles, "unknown-kind")
	assert.NotContains(t, profiles, "non-map")
	assert.NotContains(t, profiles, "checkpoint-zero")
	assert.NotContains(t, profiles, "checkpoint-ten")
	assert.NotContains(t, profiles, "malformed-interruption")
}

func TestStartTravelPersistsBeforeScheduling(t *testing.T) {
	log := &orderLog{}
	store := &fakeStore{log: log}
	scheduler := &fakeScheduler{log: log}
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	handled, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, []string{"save", "schedule"}, log.entries)
	require.Len(t, store.saved.Sessions, 1)
	assert.Equal(t, expedition.Traveling, store.saved.Sessions[7].State)
	assert.Equal(t, baseTime(), store.saved.Sessions[7].StartedAtUTC)
}

func TestStartTravelRefusesDuplicateWithoutReplacing(t *testing.T) {
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	original := store.saved.Sessions[7]

	handled, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, 1, len(scheduler.callbacks), "duplicate start must not schedule a second timer")
	assert.Equal(t, original, store.saved.Sessions[7])
}

func TestStartTravelFailsWithoutProfile(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	req := startRequest()
	req.ProfileName = "missing"
	handled, err := module.StartTravel(req)
	require.Error(t, err)
	assert.True(t, handled)
	assert.Empty(t, store.saved.Sessions)
	assert.Zero(t, store.saveCalls)
}

func TestStartTravelFailsWithoutStore(t *testing.T) {
	store := &fakeStore{saveErr: assert.AnError}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	handled, err := module.StartTravel(startRequest())
	require.Error(t, err)
	assert.True(t, handled)
	assert.Empty(t, module.sessions, "a failed save must leave no session")
	assert.Empty(t, store.saved.Sessions)
}

func TestStartTravelFailsWithoutSurvival(t *testing.T) {
	store := &fakeStore{}
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{availableErr: survival.ErrExertionUnavailable}, baseTime, testProfiles())

	handled, err := module.StartTravel(startRequest())
	require.ErrorIs(t, err, survival.ErrExertionUnavailable)
	assert.True(t, handled)
	assert.Empty(t, module.sessions)
	assert.Zero(t, store.saveCalls)
}

func TestSyncAppliesSevenOfTenCheckpoint(t *testing.T) {
	store := &fakeStore{}
	surv := &fakeSurvival{}
	now := baseTime()
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, surv, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	now = baseTime().Add(7 * time.Second)
	require.NoError(t, module.Sync(7))

	require.Len(t, surv.applied, 1)
	assert.Equal(t, survival.Exertion{Hunger: 7, Thirst: 7, Fatigue: 7}, surv.applied[0])
	assert.Equal(t, uint8(7), module.sessions[7].LastExertionCheckpoint)
	assert.Equal(t, uint8(7), store.saved.Sessions[7].LastExertionCheckpoint)
}

func TestStatusRendersProgressAndNeeds(t *testing.T) {
	store := &fakeStore{}
	surv := &fakeSurvival{needs: []survival.MemberNeeds{
		{Key: survival.LeaderMemberKey, Name: "Tester", Needs: survival.Needs{Hunger: 90, Thirst: 80, Fatigue: 70}},
	}}
	now := baseTime()
	module := newTestModule(store, &fakeScheduler{}, &fakeMover{}, surv, func() time.Time { return now }, map[string]expedition.TravelProfile{
		"oak-road": {Name: "oak-road", Duration: 60 * time.Second, Exertion: survival.Exertion{Hunger: 10, Thirst: 10, Fatigue: 10}},
	})

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	now = baseTime().Add(30 * time.Second)
	text := module.status(7)
	assert.Contains(t, text, "oak-road")
	assert.Contains(t, text, "room #100")
	assert.Contains(t, text, "room #200")
	assert.Contains(t, text, "Progress: 50%")
	assert.Contains(t, text, "Tester")
	assert.Contains(t, text, "Hunger 90")
}

func TestStatusWithoutSession(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())
	assert.Equal(t, "You are not travelling.", module.status(7))
}

func TestRenderTravelViewOnlyForActiveSession(t *testing.T) {
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, baseTime, testProfiles())

	handled, err := module.RenderTravelView(7)
	require.NoError(t, err)
	assert.False(t, handled)

	_, err = module.StartTravel(startRequest())
	require.NoError(t, err)

	handled, err = module.RenderTravelView(7)
	require.NoError(t, err)
	assert.True(t, handled)
}

func travelUser(t *testing.T, userId, roomId int) *users.UserRecord {
	t.Helper()
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	user := users.NewUserRecord(userId, 1)
	user.Character.RoomId = roomId
	users.SetTestUser(user)
	return user
}

func savedSession(state expedition.SessionState, startedAt time.Time) Registry {
	return Registry{Sessions: map[int]expedition.TravelSession{
		7: {
			LeaderUserID:      7,
			OriginRoomID:      100,
			DestinationRoomID: 200,
			ExitName:          "north",
			ProfileName:       "oak-road",
			StartedAtUTC:      startedAt,
			State:             state,
		},
	}}
}

func TestLoadReschedulesRemainingSession(t *testing.T) {
	store := &fakeStore{saved: savedSession(expedition.Traveling, baseTime())}
	scheduler := &fakeScheduler{}
	now := baseTime().Add(3 * time.Second)
	module := newTestModule(store, scheduler, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	module.load()
	require.NoError(t, module.loadErr)
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, 7*time.Second, scheduler.delays[0])
	assert.Contains(t, module.sessions, 7)
}

func TestInterruptionSchedulesAtNextBoundaryAndStopsExactlyOnce(t *testing.T) {
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	mover := &fakeMover{}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, surv, func() time.Time { return now }, interruptionProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, 5*time.Second, scheduler.delays[0])

	now = baseTime().Add(5 * time.Second)
	scheduler.fire(0)
	session, ok := module.sessions[7]
	require.True(t, ok)
	assert.Equal(t, expedition.Interrupted, session.State)
	assert.Equal(t, uint8(5), session.LastExertionCheckpoint)
	assert.Equal(t, []survival.Exertion{{Hunger: 5, Thirst: 5, Fatigue: 5}}, surv.applied)
	assert.Empty(t, mover.moves)
	assert.Contains(t, store.saved.Sessions, 7)
	assert.Equal(t, expedition.Interrupted, store.saved.Sessions[7].State)
	assert.Empty(t, module.timers)

	saves := store.saveCalls
	scheduler.fire(0)
	assert.Equal(t, saves, store.saveCalls)
	assert.Len(t, surv.applied, 1)
	assert.Empty(t, mover.moves)
}

func TestInterruptionSaveFailureRestoresTravelingCheckpoint(t *testing.T) {
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	now := baseTime()
	module := newTestModule(store, scheduler, &fakeMover{}, surv, func() time.Time { return now }, interruptionProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(5 * time.Second)
	store.failOnSaveCall = 4 // start, pending exertion, finalized checkpoint, interruption candidate
	scheduler.fire(0)

	session := module.sessions[7]
	assert.Equal(t, expedition.Traveling, session.State)
	assert.Equal(t, uint8(5), session.LastExertionCheckpoint)
	assert.Nil(t, session.Interruption)
	assert.False(t, session.InterruptionTriggered)
	assert.Equal(t, expedition.Traveling, store.saved.Sessions[7].State)
	assert.Empty(t, module.timers)
	assert.Len(t, surv.applied, 1)

	require.NoError(t, module.Sync(7))
	assert.Equal(t, expedition.Interrupted, module.sessions[7].State)
	assert.Equal(t, expedition.Interrupted, store.saved.Sessions[7].State)
	assert.Len(t, surv.applied, 1)
}

func TestLoadCompletesOverdueSession(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{saved: savedSession(expedition.Traveling, baseTime())}
	mover := &fakeMover{user: user}
	surv := &fakeSurvival{}
	now := baseTime().Add(11 * time.Second)
	module := newTestModule(store, &fakeScheduler{}, mover, surv, func() time.Time { return now }, testProfiles())

	module.load()
	require.NoError(t, module.loadErr)
	assert.Equal(t, []int{200}, mover.moves)
	assert.Equal(t, 200, user.Character.RoomId)
	assert.NotContains(t, module.sessions, 7)
	require.Len(t, surv.applied, 1)
	assert.Equal(t, survival.Exertion{Hunger: 10, Thirst: 10, Fatigue: 10}, surv.applied[0])
}

func TestTimerCompletionIsExactlyOnce(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)
	scheduler.fire(0) // stale duplicate callback

	assert.Equal(t, []int{200}, mover.moves, "a duplicate timer must not move twice")
	assert.NotContains(t, module.sessions, 7)
}

func TestCompletionRetainsSessionWhenMoveFails(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user, err: assert.AnError}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)

	require.Contains(t, module.sessions, 7)
	assert.Equal(t, expedition.Completed, module.sessions[7].State)
	assert.Empty(t, mover.moves)
	assert.Equal(t, 100, user.Character.RoomId)

	mover.err = nil
	module.mu.Lock()
	module.recoverLocked()
	module.mu.Unlock()
	assert.Equal(t, []int{200}, mover.moves)
	assert.NotContains(t, module.sessions, 7)
}

func TestRecoveryAtDestinationCleansUpWithoutMoving(t *testing.T) {
	user := travelUser(t, 7, 200)
	store := &fakeStore{saved: savedSession(expedition.Completed, baseTime())}
	mover := &fakeMover{user: user}
	module := newTestModule(store, &fakeScheduler{}, mover, &fakeSurvival{}, baseTime, testProfiles())

	module.load()
	require.NoError(t, module.loadErr)
	assert.Empty(t, mover.moves)
	assert.NotContains(t, module.sessions, 7)
}

func TestRecoveryAtUnexpectedRoomRetainsSession(t *testing.T) {
	user := travelUser(t, 7, 999)
	store := &fakeStore{saved: savedSession(expedition.Completed, baseTime())}
	mover := &fakeMover{user: user}
	module := newTestModule(store, &fakeScheduler{}, mover, &fakeSurvival{}, baseTime, testProfiles())

	module.load()
	require.NoError(t, module.loadErr)
	assert.Empty(t, mover.moves)
	assert.Contains(t, module.sessions, 7)
}

func TestCompletionWithoutUserRetainsSession(t *testing.T) {
	users.ResetActiveUsers()
	t.Cleanup(users.ResetActiveUsers)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)

	assert.Contains(t, module.sessions, 7)
	assert.Equal(t, expedition.Completed, module.sessions[7].State)
	assert.Empty(t, mover.moves)
}

func TestCheckpointWriteFailureBlocksCompletion(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	store.saveErr = assert.AnError
	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)

	assert.Empty(t, mover.moves)
	assert.Contains(t, module.sessions, 7)
	assert.Equal(t, expedition.Traveling, module.sessions[7].State)
	assert.Equal(t, uint8(0), module.sessions[7].LastExertionCheckpoint)
	require.NotNil(t, module.sessions[7].PendingExertion)
}

func TestSurvivalFailureBlocksCompletion(t *testing.T) {
	user := travelUser(t, 7, 100)
	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	mover := &fakeMover{user: user}
	surv := &fakeSurvival{}
	now := baseTime()
	module := newTestModule(store, scheduler, mover, surv, func() time.Time { return now }, testProfiles())

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)

	surv.applyErr = survival.ErrPersistenceUnavailable
	now = baseTime().Add(10 * time.Second)
	scheduler.fire(0)

	assert.Empty(t, mover.moves)
	assert.Contains(t, module.sessions, 7)
	assert.Equal(t, expedition.Traveling, module.sessions[7].State)
}

func TestCopyoverRestorationReschedulesFromDurableRecord(t *testing.T) {
	store := &fakeStore{}
	now := baseTime()
	parent := newTestModule(store, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())
	_, err := parent.StartTravel(startRequest())
	require.NoError(t, err)

	childScheduler := &fakeScheduler{}
	childNow := baseTime().Add(4 * time.Second)
	child := newTestModule(store, childScheduler, &fakeMover{}, &fakeSurvival{}, func() time.Time { return childNow }, testProfiles())
	child.load()

	require.NoError(t, child.loadErr)
	require.Len(t, childScheduler.delays, 1)
	assert.Equal(t, 6*time.Second, childScheduler.delays[0])
	assert.Equal(t, expedition.Traveling, child.sessions[7].State)
}

func TestDunmarOakRoute(t *testing.T) {
	// Load the shipped default world so the proving route is validated exactly
	// as it ships.
	dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
	require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
	rooms.LoadDataFiles()
	rooms.LoadBiomeDataFiles()
	keywords.LoadAliases()

	origin := rooms.LoadRoom(2001)
	require.NotNil(t, origin, "Dunmar West Gate must exist")
	assert.Equal(t, "Dunmar West Gate", origin.Title)
	destination := rooms.LoadRoom(2002)
	require.NotNil(t, destination, "Fork at the Black Oak must exist")
	assert.Equal(t, "Fork at the Black Oak", destination.Title)

	exitName, destinationId := origin.FindExitByName("north")
	require.NotEmpty(t, exitName, "the west gate must have a north route")
	assert.Equal(t, 2002, destinationId)
	exitInfo, ok := origin.GetExitInfo(exitName)
	require.True(t, ok)
	assert.Equal(t, "oak-road", exitInfo.TravelProfile)

	// The configured profile is valid, positive, and safe for full needs.
	data, err := files.ReadFile("files/data-overlays/config.yaml")
	require.NoError(t, err)
	var cfg struct {
		Profiles []any `yaml:"Profiles"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	profiles := parseProfiles(cfg.Profiles)
	profile, ok := profiles["oak-road"]
	require.True(t, ok, "oak-road must be configured")
	require.Greater(t, profile.Duration, time.Duration(0))
	require.NotNil(t, profile.Interruption)
	assert.Equal(t, expedition.FallenTree, profile.Interruption.Kind)
	assert.Equal(t, uint8(5), profile.Interruption.Checkpoint)

	full := survival.FullNeeds()
	assert.Greater(t, full.Hunger-profile.Exertion.Hunger, 25, "hunger must stay above critical")
	assert.Greater(t, full.Thirst-profile.Exertion.Thirst, 25, "thirst must stay above critical")
	assert.Greater(t, full.Fatigue-profile.Exertion.Fatigue, 25, "fatigue must stay above critical")
}

func TestMovementBlockedWhileTravelling(t *testing.T) {
	now := baseTime()
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeMover{}, &fakeSurvival{}, func() time.Time { return now }, testProfiles())

	blocked, _ := module.MovementBlocked(7)
	assert.False(t, blocked)

	_, err := module.StartTravel(startRequest())
	require.NoError(t, err)
	now = baseTime().Add(4 * time.Second)

	blocked, message := module.MovementBlocked(7)
	assert.True(t, blocked)
	assert.Contains(t, message, "40%")
	assert.Contains(t, message, "travelling")
}
