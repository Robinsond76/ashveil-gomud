package expedition

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/encumbrance"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/expedition"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/weather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wiringAliasesOnce sync.Once

// loadWiringAliases points the data dir at the shipped world (the same
// override TestDunmarOakRoute uses) and loads its direction aliases.
func loadWiringAliases(t *testing.T) {
	t.Helper()
	wiringAliasesOnce.Do(func() {
		dataDir := filepath.Join("..", "..", "_datafiles", "world", "default")
		require.NoError(t, configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": dataDir}))
		keywords.LoadAliases()
	})
}

type trackedWeather struct{ zones map[string]weather.Condition }

func (w trackedWeather) CurrentCondition(zone string) (weather.Condition, bool) {
	c, ok := w.zones[zone]
	return c, ok
}

// heavyLoad is a registered encumbrance provider reporting the leader over
// the 90% band.
type heavyLoad struct{}

func (heavyLoad) CurrentLoad(int) (encumbrance.Load, bool) {
	return encumbrance.Load{PersonalGrams: 190000, CapacityGrams: 200000}, true
}

func (heavyLoad) CurrentBand(int) (encumbrance.LoadBand, bool) {
	return encumbrance.ResolveBand(0.95, []encumbrance.LoadBand{
		{MinRatio: 0.75, TravelDurationPct: 110, FatiguePct: 108},
		{MinRatio: 0.9, TravelDurationPct: 125, FatiguePct: 115},
	}), true
}

// TestGoOnTravelExitWithHeavyLoadInTrackedZoneStartsLongerSession drives
// the real user Go command into the real module's StartTravel, with the
// module's native room, weather, encumbrance, and mount seams.
func TestGoOnTravelExitWithHeavyLoadInTrackedZoneStartsLongerSession(t *testing.T) {
	loadWiringAliases(t)
	for _, r := range []*rooms.Room{
		{RoomId: 96001, Zone: "Wiring Road", Title: "Gate", Biome: "forest", Exits: map[string]exit.RoomExit{"north": {RoomId: 96002, TravelProfile: "oak-road"}}},
		{RoomId: 96002, Zone: "Wiring Road", Title: "Fork", Biome: "forest", Exits: map[string]exit.RoomExit{"south": {RoomId: 96001}}},
	} {
		rooms.SetTestRoom(r)
		id := r.RoomId
		t.Cleanup(func() { rooms.RemoveTestRoom(id) })
	}
	user := travelUser(t, 7, 96001)
	user.Character.ActionPoints = 100

	weather.SetProvider(trackedWeather{zones: map[string]weather.Condition{"Wiring Road": {Name: "rain", TravelDurationPct: 115, ExertionPct: 115, RestRecoveryPct: 90}}})
	encumbrance.SetProvider(heavyLoad{})
	t.Cleanup(func() {
		weather.SetProvider(nil)
		encumbrance.SetProvider(nil)
	})

	store := &fakeStore{}
	scheduler := &fakeScheduler{}
	module := newTestModule(store, scheduler, &fakeMover{user: user}, &fakeSurvival{}, baseTime, testProfiles())
	expedition.SetStartProvider(module)
	t.Cleanup(func() { expedition.SetStartProvider(nil) })

	handled, err := usercommands.Go("north", user, rooms.LoadRoom(96001), 0)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Equal(t, 96001, user.Character.RoomId, "the journey starts in place")

	session, ok := store.saved.Sessions[7]
	require.True(t, ok)
	assert.Equal(t, 144, session.DurationPct, "rain 115% × load 125% = 143.75%")
	assert.Equal(t, 115, session.ExertionPct)
	assert.Equal(t, 115, session.FatiguePct)
	require.Len(t, scheduler.delays, 1)
	assert.Equal(t, 14400*time.Millisecond, scheduler.delays[0], "a 10s route now takes 14.4s")
}
