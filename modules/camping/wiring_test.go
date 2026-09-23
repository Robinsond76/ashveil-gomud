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
func (f formationStub) InstanceFor(int, int) (int, bool)          { return f.instance, true }
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
