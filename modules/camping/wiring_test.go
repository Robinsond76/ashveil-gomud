package camping

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/camping"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/company"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/survival"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var worldOnce sync.Once

func loadShippedWorld(t *testing.T) {
	t.Helper()
	worldOnce.Do(func() {
		dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		rooms.LoadDataFiles()
		rooms.LoadBiomeDataFiles()
	})
}

type zoneWeather map[string]weather.Condition

func (z zoneWeather) CurrentCondition(zone string) (weather.Condition, bool) {
	c, ok := z[zone]
	return c, ok
}

// TestCampRestCommandInTrackedZoneRecoversScaledAmount: the camp user
// command in the real Fork at the Black Oak (Old Kings Road), with the
// module's native weather seam reading a registered provider.
func TestCampRestCommandInTrackedZoneRecoversScaledAmount(t *testing.T) {
	loadShippedWorld(t)
	fork := rooms.LoadRoom(2002)
	require.NotNil(t, fork)
	weather.SetProvider(zoneWeather{"Old Kings Road": {Name: "storm", TravelDurationPct: 140, ExertionPct: 130, RestRecoveryPct: 75}})
	t.Cleanup(func() { weather.SetProvider(nil) })

	now := baseTime()
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(&fakeStore{}, scheduler, surv, func() time.Time { return now })
	module.weatherIn = nil // native: weather.CurrentCondition
	user := campUser(t, 7, 2002)

	for _, cmd := range []string{"", "fire", "rest"} {
		_, err := module.userCommand(cmd, user, fork, 0)
		require.NoError(t, err)
	}
	require.NotNil(t, module.camps[7].Rest)
	assert.Equal(t, 15, module.camps[7].Rest.Recovery, "20 × storm 75%")
	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	assert.Equal(t, []int{15}, surv.amounts)
}

type rosterStub struct{ refs []survival.MemberRef }

func (r rosterStub) Roster(int) []survival.MemberRef { return r.refs }

type formationStub struct{ instance int }

func (f formationStub) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (f formationStub) InstanceFor(int, int) (int, bool)           { return f.instance, true }
func (f formationStub) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}

// TestInnRestInWaymarkInnThroughToWellRested: the inn user command in the
// real room 2003 with real gold, a real spawned companion found through
// the native roster and formation seams, the real buff grant, and the
// round handler.
func TestInnRestInWaymarkInnThroughToWellRested(t *testing.T) {
	loadShippedWorld(t)
	inn := rooms.LoadRoom(2003)
	require.NotNil(t, inn)

	buffs.SetTestFlag("well-rested")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1030, Name: "Well Rested", TriggerCount: 450, Flags: []string{"well-rested"}})
	t.Cleanup(func() { buffs.RemoveTestBuffSpec(1030) })

	bran := &mobs.Mob{InstanceId: 97001, Character: *characters.New()}
	bran.Character.Name = "Bran"
	bran.Character.RoomId = 2003
	mobs.SetTestInstance(bran)
	t.Cleanup(func() { mobs.RemoveTestInstance(97001) })
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bran"},
	}})
	company.SetFormationProvider(formationStub{instance: 97001})
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
	})

	now := baseTime()
	scheduler := &fakeScheduler{}
	surv := &fakeSurvival{}
	module := newTestModule(&fakeStore{}, scheduler, surv, func() time.Time { return now })
	user := campUser(t, 7, 2003)
	user.Character.Gold = 30
	messages := captureMessages(t)

	_, err := module.innCommand("", user, inn, 0)
	require.NoError(t, err)
	_, err = module.innCommand("rest", user, inn, 0)
	require.NoError(t, err)
	assert.Equal(t, 20, user.Character.Gold, "5 gold × 2 members")

	now = now.Add(60 * time.Second)
	scheduler.fireLatest()
	assert.Equal(t, []int{60}, surv.amounts)
	assert.False(t, user.Character.HasBuffFlag("well-rested"), "not granted off the loop")

	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.HasBuffFlag("well-rested"))
	assert.True(t, bran.Character.HasBuffFlag("well-rested"))
	events.ProcessEvents()
	joined := ""
	for _, m := range *messages {
		joined += m + "\n"
	}
	assert.Contains(t, joined, "The Waymark Inn offers your company of 2 a room")
	assert.Contains(t, joined, "You pay 10 gold")
	assert.Contains(t, joined, "well rested")
}

type formationMap map[int]int

func (f formationMap) FormationFor(int) (company.Formation, bool) { return company.Formation{}, false }
func (f formationMap) InstanceFor(_ int, companionID int) (int, bool) {
	id, ok := f[companionID]
	return id, ok
}
func (f formationMap) LeaderAndKeyForInstance(int) (int, company.MemberKey, bool) {
	return 0, "", false
}

func restedSpecs(t *testing.T) {
	t.Helper()
	buffs.SetTestFlag("rested")
	buffs.SetTestFlag("well-rested")
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1033, Name: "Rested", TriggerCount: 225, Flags: []string{"rested"}})
	buffs.SetTestBuffSpec(&buffs.BuffSpec{BuffId: 1030, Name: "Well Rested", TriggerCount: 450, Flags: []string{"well-rested"}})
	t.Cleanup(func() {
		buffs.RemoveTestBuffSpec(1033)
		buffs.RemoveTestBuffSpec(1030)
	})
}

func testMob(t *testing.T, instanceID int, name string, roomID int) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{InstanceId: instanceID, Character: *characters.New()}
	mob.Character.Name = name
	mob.Character.RoomId = roomID
	mobs.SetTestInstance(mob)
	t.Cleanup(func() { mobs.RemoveTestInstance(instanceID) })
	return mob
}

// TestCampRestCommandThroughToRested (Phase 23a): the camp user commands
// in the real Fork at the Black Oak, the rest timer, and the round handler
// with the native buff, roster, and formation seams. Bran is present and
// gets Rested; Cara has no mob yet, is owed it, and gets it for the time
// left once her mob spawns.
func TestCampRestCommandThroughToRested(t *testing.T) {
	loadShippedWorld(t)
	restedSpecs(t)
	fork := rooms.LoadRoom(2002)
	require.NotNil(t, fork)

	bran := testMob(t, 97011, "Bran", 2002)
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{
		{Key: survival.LeaderMemberKey, Name: "Hero"},
		{Key: survival.CompanionMemberKey(1), Name: "Bran"},
		{Key: survival.CompanionMemberKey(2), Name: "Cara"},
	}})
	formation := formationMap{1: 97011}
	company.SetFormationProvider(formation)
	t.Cleanup(func() {
		survival.SetRosterProvider(nil)
		company.SetFormationProvider(nil)
	})

	now := baseTime()
	scheduler := &fakeScheduler{}
	store := &fakeStore{}
	module := newTestModule(store, scheduler, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 2002)
	for _, cmd := range []string{"", "fire", "rest"} {
		_, err := module.userCommand(cmd, user, fork, 0)
		require.NoError(t, err)
	}
	now = now.Add(camping.RestDuration)
	scheduler.fireLatest()
	assert.False(t, user.Character.HasBuffFlag("rested"), "not granted off the loop")

	module.onNewRound(events.NewRound{RoundNumber: 1})
	require.True(t, user.Character.HasBuffFlag("rested"))
	assert.Equal(t, 225, user.Character.GetBuffs(1033)[0].TriggersLeft, "15 minutes at the engine's 4-second rounds")
	assert.True(t, bran.Character.HasBuffFlag("rested"))
	require.Contains(t, store.saved.Owed[7], 2)

	now = now.Add(5 * time.Minute)
	cara := testMob(t, 97012, "Cara", 2002)
	formation[2] = 97012
	module.onNewRound(events.NewRound{RoundNumber: 2})
	require.True(t, cara.Character.HasBuffFlag("rested"))
	assert.Equal(t, 150, cara.Character.GetBuffs(1033)[0].TriggersLeft)

	// A relog or copyover respawns Bran without his buffs: he gets the
	// time left back from the durable grant.
	now = now.Add(5 * time.Minute)
	respawned := testMob(t, 97013, "Bran", 2002)
	formation[1] = 97013
	module.onNewRound(events.NewRound{RoundNumber: 3})
	require.True(t, respawned.Character.HasBuffFlag("rested"))
	assert.Equal(t, 75, respawned.Character.GetBuffs(1033)[0].TriggersLeft, "5 minutes left")
	assert.Contains(t, store.saved.Owed[7], 1)
}

// TestInnAfterCampReplacesRestedWithWellRested: real buffs, so a Rested
// leader ends up holding Well Rested alone.
func TestInnAfterCampReplacesRestedWithWellRested(t *testing.T) {
	loadShippedWorld(t)
	restedSpecs(t)
	inn := rooms.LoadRoom(2003)
	require.NotNil(t, inn)
	survival.SetRosterProvider(rosterStub{refs: []survival.MemberRef{{Key: survival.LeaderMemberKey, Name: "Hero"}}})
	t.Cleanup(func() { survival.SetRosterProvider(nil) })

	now := baseTime()
	scheduler := &fakeScheduler{}
	module := newTestModule(&fakeStore{}, scheduler, &fakeSurvival{}, func() time.Time { return now })
	user := campUser(t, 7, 2003)
	user.Character.Gold = 30
	require.NoError(t, user.Character.AddBuff(1033, false))

	_, err := module.innCommand("rest", user, inn, 0)
	require.NoError(t, err)
	now = now.Add(60 * time.Second)
	scheduler.fireLatest()
	module.onNewRound(events.NewRound{RoundNumber: 1})
	assert.True(t, user.Character.HasBuffFlag("well-rested"))
	assert.Zero(t, user.Character.Buffs.TriggersLeft(1033), "Rested removed")
	assert.False(t, user.Character.HasBuffFlag("rested"))
}

// TestPlayerSpawnEventNormalizesSavedTiers: a PlayerSpawn through the real
// event queue reaches the registered module, which leaves one tier.
func TestPlayerSpawnEventNormalizesSavedTiers(t *testing.T) {
	restedSpecs(t)
	user := campUser(t, 7, 1)
	require.NoError(t, user.Character.AddBuff(1030, false))
	require.NoError(t, user.Character.AddBuff(1033, false))
	events.AddToQueue(events.PlayerSpawn{UserId: 7})
	events.ProcessEvents()
	assert.Positive(t, user.Character.Buffs.TriggersLeft(1030))
	assert.Zero(t, user.Character.Buffs.TriggersLeft(1033))
}

// TestExpiredWellRestedDoesNotBlockRested: a Well Rested that ran out but
// isn't pruned yet no longer counts as held.
func TestExpiredWellRestedDoesNotBlockRested(t *testing.T) {
	restedSpecs(t)
	module := newTestModule(&fakeStore{}, &fakeScheduler{}, &fakeSurvival{}, baseTime)
	user := campUser(t, 7, 1)
	require.NoError(t, user.Character.AddBuff(1030, false))
	user.Character.RemoveBuff(1030) // expired, awaiting the prune
	require.True(t, user.Character.HasBuff(1030))
	assert.True(t, module.applyTier(user.Character, camping.TierRested, 225, module.innSettings()))
	assert.True(t, user.Character.HasBuffFlag("rested"))
}
